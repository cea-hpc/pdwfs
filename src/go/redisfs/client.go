// Copyright 2019 CEA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redisfs

import (
	"fmt"
	"strconv"
	"sync"
	"time"
	"unsafe"
	"sync/atomic"

	"github.com/cea-hpc/pdwfs/config"
	"github.com/cea-hpc/pdwfs/util"
	"github.com/go-redis/redis"
)

// Try ...
func Try(err error) {
	if err != nil {
		panic(err)
	}
}

// Check is an alias for Try
var Check = Try

// IRedisClient interface to allow multiple client implementations (client, ring, cluster)
type IRedisClient interface {
	SetRange(string, int64, string) *redis.IntCmd
	GetRange(string, int64, int64) *redis.StringCmd
	Exists(keys ...string) *redis.IntCmd
	Set(string, interface{}, time.Duration) *redis.StatusCmd
	Get(string) *redis.StringCmd
	Del(...string) *redis.IntCmd
	Unlink(...string) *redis.IntCmd
	SAdd(key string, members ...interface{}) *redis.IntCmd
	SRem(key string, members ...interface{}) *redis.IntCmd
	SMembers(key string) *redis.StringSliceCmd
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(hashes ...string) *redis.BoolSliceCmd
	ScriptLoad(script string) *redis.StringCmd
	Pipeline() redis.Pipeliner
	Ping() *redis.StatusCmd
	Close() error
}

// NewRedisClient ...
func NewRedisClient(conf *config.Redis) IRedisClient {

	addrs := make(map[string]string)
	for i, addr := range conf.Addrs {
		addrs[fmt.Sprintf("shard%d", i)] = addr
	}

	opt := &redis.RingOptions{
		Addrs: addrs,
		// disable timeouts and heartbeating(sort of)
		PoolTimeout:        1 * time.Hour,
		ReadTimeout:        1 * time.Hour,
		IdleTimeout:        1 * time.Hour,
		HeartbeatFrequency: 1 * time.Hour,
	}
	return redis.NewRing(opt)
}

// RedisRing ...
type RedisRing struct {
	clients map[string]*redis.Client
	hash *util.ConsistentHash
}

// NewRedisRing ...
func NewRedisRing(conf *config.Redis) *RedisRing {

	shards := make([]string, len(conf.Addrs))
	clients := make(map[string]*redis.Client)
	for i, addr := range conf.Addrs {
		shards[i] = fmt.Sprintf("shard%d", i)
		clients[shards[i]] = redis.NewClient(&redis.Options{Addr: addr})
	}
	hash := util.NewConsistentHash(100, nil)
	hash.Add(shards...)

	return &RedisRing{
		clients: clients,
		hash: hash,	
	}
}

// GetClient ...
func (r *RedisRing) GetClient(key string) *redis.Client {
	return r.clients[r.hash.Get(key)]
} 

// Close ...
func (r *RedisRing) Close() error {
	var err error
	for _, client := range r.clients {
		err = client.Close()
	}
	return err
}

func byte2StringNoCopy(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}


// FileContentClient ...
type FileContentClient struct {
	redis *RedisRing
	conf        *config.Redis
	stripeSize  int64
}

// NewFileContentClient returns a redis client
func NewFileContentClient(conf *config.Redis, stripeSize int64) *FileContentClient {
	return &FileContentClient{
		redis: NewRedisRing(conf),
		conf:        conf,
		stripeSize:  stripeSize,
	}
}

type stripeInfo struct {
	id  int64
	off int64 // offset relative to beginning of stripe
	len int64 // length of data in stripe
}

func stripeLayout(stripeSize, off, size int64) []stripeInfo {

	startID := off / stripeSize
	endID := (off + size - 1) / stripeSize // last block inclusive
	nStripes := endID - startID + 1

	s := make([]stripeInfo, nStripes)

	// first stripe
	s[0].id = startID
	s[0].off = off % stripeSize
	if nStripes == 1 {
		s[0].len = size
	} else {
		s[0].len = stripeSize - s[0].off
	}

	if nStripes > 1 {
		//last stripe, inclusive
		s[nStripes-1].id = endID
		s[nStripes-1].off = 0
		s[nStripes-1].len = (off+size-1)%stripeSize + 1

		// other stripes (nStripes > 2)
		for i := int64(1); i < nStripes-1; i++ {
			s[i].id = startID + i
			s[i].off = 0
			s[i].len = stripeSize
		}
	}
	return s
}

var setSize = redis.NewScript(`
		local tentativeSize = tonumber(ARGV[1])
		if redis.call("EXISTS", KEYS[1]) == 1 then
			local curSize = tonumber(redis.call("GET", KEYS[1]))
			if curSize > tentativeSize then
				return curSize
			end
		end
		redis.call("SET", KEYS[1], tentativeSize)
		return tentativeSize
	`)

// WriteAt ...
func (c FileContentClient) WriteAt(name string, off int64, data []byte) {
	dataLen := int64(len(data))
	stripes := stripeLayout(c.stripeSize, off, dataLen)
	var k int64
	wg := sync.WaitGroup{}
	for _, stripe := range stripes {
		if len(data) == 0 {
			continue
		}
		wg.Add(1)
		go func(key string, off int64, data string) {
			defer wg.Done()
			if off == 0 && int64(len(data)) == c.stripeSize {
				c.redis.GetClient(key).Set(key, data, 0)
			} else {
				c.redis.GetClient(key).SetRange(key, off, data)
			}
		}(fmt.Sprintf("%s:%d", name, stripe.id), stripe.off, byte2StringNoCopy(data[k:k+stripe.len]))
		k += stripe.len	
	}
	wg.Wait()
	key := name + ":size"
	Try(setSize.Run(c.redis.GetClient(key), []string{key}, off+dataLen).Err())
}

// ReadAt ...
func (c FileContentClient) ReadAt(name string, off int64, dst []byte) int64 {
	var k, n int64
	stripes := stripeLayout(c.stripeSize, off, int64(len(dst)))
	wg := sync.WaitGroup{}
	for _, stripe := range stripes {
		wg.Add(1)
		go func(key string, off, size int64, dst []byte) {
			defer wg.Done()
			var res string
			var err error
			if off == 0 && size == c.stripeSize {
				res, err = c.redis.GetClient(key).Get(key).Result()
			} else {
				res, err = c.redis.GetClient(key).GetRange(key, off, off+size-1).Result()
			}
			if err != nil && err != redis.Nil {
				panic(err)
			}
			read := copy(dst, res)
			atomic.AddInt64(&n, int64(read))
		}(fmt.Sprintf("%s:%d", name, stripe.id), stripe.off, stripe.len, dst[k:k+stripe.len])
		k += stripe.len
	}
	wg.Wait()
	k = 0
	return n
}

func (c FileContentClient) removeStripe(key string) {
	if c.conf.UseUnlink {
		Try(c.redis.GetClient(key).Unlink(key).Err())
	} else {
		Try(c.redis.GetClient(key).Del(key).Err())
	}
}

// GetSize ...
func (c FileContentClient) GetSize(name string) int64 {
	key := name + ":size"
	s, err := c.redis.GetClient(key).Get(key).Result()
	if err != nil && err == redis.Nil {
		return 0
	}
	Check(err)
	size, err := strconv.ParseInt(s, 10, 64)
	Check(err)
	return size

}

// Remove ...
func (c FileContentClient) Remove(name string) {
	stripes := stripeLayout(c.stripeSize, 0, c.GetSize(name))
	for _, stripe := range stripes {
		c.removeStripe(fmt.Sprintf("%s:%d", name, stripe.id))
	}
	key := name + ":size"
	Try(c.redis.GetClient(key).Del(key).Err())
}

// Resize ...
func (c FileContentClient) Resize(name string, size int64) {
	if size < 0 {
		panic(fmt.Errorf("size must be non-negative"))
	}
	curSize := c.GetSize(name)
	switch {
	case size == curSize:
	case size < curSize:
		c.shrink(name, size)
	default:
		c.grow(name, size)
	}
}

var trimString = redis.NewScript(`
		local str = redis.call("GETRANGE", KEYS[1], 0, ARGV[1])
		return redis.call("SET", KEYS[1], str)
	`)

func (c FileContentClient) shrink(name string, size int64) {
	curStripes := stripeLayout(c.stripeSize, 0, c.GetSize(name))
	curLastStripe := curStripes[len(curStripes)-1]
	newStripes := stripeLayout(c.stripeSize, 0, size)
	newLastStripe := newStripes[len(newStripes)-1]
	// remove all existing stripes after this new last stripe
	for id := newLastStripe.id + 1; id <= curLastStripe.id; id++ {
		c.removeStripe(fmt.Sprintf("%s:%d", name, id))
	}
	// resize the last stripe
	stripeKey := fmt.Sprintf("%s:%d", name, newLastStripe.id)
	Try(trimString.Run(c.redis.GetClient(stripeKey), []string{stripeKey}, newLastStripe.len-1).Err())
	key := name + ":size"
	Try(c.redis.GetClient(key).Set(key, size, 0).Err())
}

func (c FileContentClient) grow(name string, size int64) {
	curStripes := stripeLayout(c.stripeSize, 0, c.GetSize(name))
	curLastStripe := curStripes[len(curStripes)-1]
	newStripes := stripeLayout(c.stripeSize, 0, size)
	newLastStripe := newStripes[len(newStripes)-1]
	// fill new stripes with null bytes
	for id := curLastStripe.id + 1; id <= newLastStripe.id; id++ {
		key := fmt.Sprintf("%s:%d", name, id)
		Try(c.redis.GetClient(key).SetRange(key, c.stripeSize-1, "\x00").Err())
	}
	// fill curLastStripe with null byte if needed
	if (curLastStripe.off + curLastStripe.len) < c.stripeSize {
		key := fmt.Sprintf("%s:%d", name, curLastStripe.id)
		Try(c.redis.GetClient(key).SetRange(key, c.stripeSize-1, "\x00").Err())
	}
	key := name + ":size"
	Try(c.redis.GetClient(key).Set(key, size, 0).Err())
}

// Close ...
func (c FileContentClient) Close() error {
	return c.redis.Close()
}
