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
	"strings"
	"sync"
	"time"
	"unsafe"

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
	StrLen(string) *redis.IntCmd
	SetRange(string, int64, string) *redis.IntCmd
	GetRange(string, int64, int64) *redis.StringCmd
	Exists(keys ...string) *redis.IntCmd
	Set(string, interface{}, time.Duration) *redis.StatusCmd
	Get(string) *redis.StringCmd
	Del(...string) *redis.IntCmd
	Unlink(...string) *redis.IntCmd
	SetNX(string, interface{}, time.Duration) *redis.BoolCmd
	SetBit(key string, offset int64, value int) *redis.IntCmd
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
	PTTL(key string) *redis.DurationCmd
	PExpire(key string, expiration time.Duration) *redis.BoolCmd
	FlushAll() *redis.StatusCmd
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

func byte2StringNoCopy(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Asynchronous writer pool

// Job ...
type Job struct {
	async   bool
	key     string
	offset  int64
	value   []byte
	ackChan chan bool
}

func worker(conf *config.Redis, controllerIn chan *Job, controllerOut chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	client := NewRedisClient(conf)
	defer client.Close()
	var data string
	for job := range controllerIn {
		if job.async {
			stateKey := job.key + ":state"
			data = string(job.value) // copy
			Try(client.Set(stateKey, "INCONSISTENT", 0).Err())
			job.ackChan <- true
			if job.offset < 0 {
				// FIXME: negative offset means something for Redis
				p := client.Pipeline()
				p.Set(job.key, data, 0)
				p.Del(stateKey)
				_, err := p.Exec()
				Check(err)
			} else {
				p := client.Pipeline()
				p.SetRange(job.key, job.offset, data)
				p.Del(stateKey)
				_, err := p.Exec()
				Check(err)
			}
			controllerOut <- len(job.value)
		} else {
			// synchronous case
			data = byte2StringNoCopy(job.value) // no copy
			if job.offset < 0 {
				// FIXME: negative offset means something for Redis
				Try(client.Set(job.key, data, 0).Err())
			} else {
				Try(client.SetRange(job.key, job.offset, data).Err())
			}
			job.ackChan <- true
		}
	}
}

// WritePool ...
type WritePool struct {
	maxSize     int64
	workerHash  *util.ConsistentHash
	jobRequest  chan *Job
	workerChans map[string]chan *Job
	workerResp  chan int
	end         chan bool
	wg          sync.WaitGroup
}

// NewWritePool ...
func NewWritePool(conf *config.Redis) *WritePool {
	p := &WritePool{
		maxSize:     conf.WritePoolBufferSize,
		workerHash:  util.NewConsistentHash(100, nil),
		jobRequest:  make(chan *Job),
		workerChans: make(map[string]chan *Job),
		workerResp:  make(chan int, 1000),
		end:         make(chan bool),
		wg:          sync.WaitGroup{},
	}

	// start workers
	workers := conf.WritePoolWorkers
	p.wg.Add(workers)
	ids := make([]string, workers)
	for i := 0; i < workers; i++ {
		ids[i] = strconv.Itoa(i)
	}
	p.workerHash.Add(ids...) // building hash in one shot is much faster than in a loop
	for _, id := range ids {
		c := make(chan *Job, 1000)
		p.workerChans[id] = c
		go worker(conf, c, p.workerResp, &p.wg)
	}

	// start the controller
	p.wg.Add(1)
	go func() {
		var asyncBufSize int64
		var asyncJobPending int
		defer p.wg.Done()
		for {
			select {
			case <-p.end:
				for _, c := range p.workerChans {
					close(c)
				}
				return
			case job := <-p.jobRequest:
				jobSize := int64(len(job.value))
				if asyncBufSize+jobSize > p.maxSize {
					// job doesn't fit in asynchronous buffer
					job.async = false
				}
				if job.async {
					asyncBufSize += jobSize
					asyncJobPending++
				}
				worker := p.workerHash.Get(job.key)
				p.workerChans[worker] <- job
			case wrote := <-p.workerResp:
				// receive only from async jobs
				asyncBufSize -= int64(wrote)
				asyncJobPending--
			}
		}
	}()

	return p
}

// SetRange ...
func (p *WritePool) SetRange(key string, off int64, value []byte) {
	ack := make(chan bool) // could come from a pool instead
	p.jobRequest <- &Job{true, key, off, value, ack}
	<-ack
}

// Close ...
func (p *WritePool) Close() {
	p.end <- true
	p.wg.Wait()
}

// FileContentClient ...
type FileContentClient struct {
	redisClient IRedisClient
	writePool   *WritePool
	conf        *config.Redis
	stripeSize  int64
}

// NewFileContentClient returns a redis client
func NewFileContentClient(conf *config.Redis, stripeSize int64) *FileContentClient {
	return &FileContentClient{
		redisClient: NewRedisClient(conf),
		writePool:   NewWritePool(conf),
		conf:        conf,
		stripeSize:  stripeSize,
	}
}

func (c FileContentClient) writeStripe(key string, value []byte) {
	if c.conf.UseWritePool {
		c.writePool.SetRange(key, -1, value)
	} else {
		Try(c.redisClient.Set(key, byte2StringNoCopy(value), 0).Err())
	}
	
}

func (c FileContentClient) writeStripeAt(key string, off int64, value []byte) {
	if c.conf.UseWritePool {
		c.writePool.SetRange(key, off, value)
	} else {
		Try(c.redisClient.SetRange(key, off, byte2StringNoCopy(value)).Err())
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
	wg.Add(len(stripes))
	for _, stripe := range stripes {
		go func(key string, off int64, data []byte, wg *sync.WaitGroup) {
			defer wg.Done()
			if off == 0 && int64(len(data)) == c.stripeSize {
				c.writeStripe(key, data)
			} else {
				c.writeStripeAt(key, off, data)
			}
		}(fmt.Sprintf("%s:%d", name, stripe.id), stripe.off, data[k:k+stripe.len], &wg)
		k += stripe.len
	}
	wg.Wait()
	Try(setSize.Run(c.redisClient, []string{name + ":size"}, off+dataLen).Err())
}

var noArgCmd = redis.NewScript(`
		local keyState = redis.call("GET", KEYS[1])
		if keyState == "INCONSISTENT" then
			return redis.error_reply(keyState)
		end
		return redis.call(ARGV[1], KEYS[2])
	`)

var twoArgsCmd = redis.NewScript(`
	local keyState = redis.call("GET", KEYS[1])
	if keyState == "INCONSISTENT" then
		return redis.error_reply(keyState)
	end
	return redis.call(ARGV[1], KEYS[2], ARGV[2], ARGV[3])
`)

const sleep = 1 * time.Millisecond

func (c FileContentClient) readStripe(key string) (string, bool) {
	for {
		val, err := noArgCmd.Run(c.redisClient, []string{key + ":state", key}, "GET").String()
		if err == nil {
			return val, true
		}
		if err == redis.Nil {
			return "", false
		}
		if err.Error() == "INCONSISTENT" {
			time.Sleep(sleep)
			continue
		}
		panic(err)
	}
}

func (c FileContentClient) readStripeRange(key string, start, end int64) (string, bool) {
	for {
		val, err := twoArgsCmd.Run(c.redisClient, []string{key + ":state", key}, "GETRANGE", start, end).String()
		if err == nil {
			return val, true
		}
		if err == redis.Nil {
			return "", false
		}
		if err.Error() == "INCONSISTENT" {
			time.Sleep(sleep)
			continue
		}
		panic(err)
	}
}

type readAtReturn struct {
	id   int
	data string
	ok   bool
}

// ReadAt ...
func (c FileContentClient) ReadAt(name string, off, size int64) (string, bool) {
	var res strings.Builder
	ok := false

	stripes := stripeLayout(c.stripeSize, off, size)
	wg := sync.WaitGroup{}
	wg.Add(len(stripes))
	retChan := make(chan *readAtReturn, len(stripes))
	for i, stripe := range stripes {
		go func(key string, off, size int64, ret chan *readAtReturn, i int, wg *sync.WaitGroup) {
			defer wg.Done()
			var s string
			var ok bool
			if off == 0 && size == c.stripeSize {
				s, ok = c.readStripe(key)
			} else {
				s, ok = c.readStripeRange(key, off, off+size-1)
			}
			ret <- &readAtReturn{i, s, ok}
		}(fmt.Sprintf("%s:%d", name, stripe.id), stripe.off, stripe.len, retChan, i, &wg)
	}
	wg.Wait()
	var i int
	for {
		// retrieve result and put them in order
		if i == len(stripes) {
			break
		}
		r := <-retChan
		if r.id != i {
			retChan <- r
			continue
		}
		res.WriteString(r.data)
		if r.ok {
			ok = r.ok
		}
		i++
	}
	return res.String(), ok
}

func (c FileContentClient) removeStripe(key string) bool {
	cmd := "DEL"
	if c.conf.UseUnlink {
		cmd = "UNLINK"
	}
	for {
		err := noArgCmd.Run(c.redisClient, []string{key + ":state", key}, cmd).Err()
		if err == nil {
			return true
		}
		if err == redis.Nil {
			return false
		}
		if err.Error() == "INCONSISTENT" {
			time.Sleep(sleep)
			continue
		}
		panic(err)
	}
}

// GetSize ...
func (c FileContentClient) GetSize(name string) int64 {
	s, err := c.redisClient.Get(name + ":size").Result()
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
	Try(c.redisClient.Del(name + ":size").Err())
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
	Try(trimString.Run(c.redisClient, []string{stripeKey}, newLastStripe.len-1).Err())
	Try(c.redisClient.Set(name+":size", size, 0).Err())
}

func (c FileContentClient) grow(name string, size int64) {
	curStripes := stripeLayout(c.stripeSize, 0, c.GetSize(name))
	curLastStripe := curStripes[len(curStripes)-1]
	newStripes := stripeLayout(c.stripeSize, 0, size)
	newLastStripe := newStripes[len(newStripes)-1]
	// fill new stripes with null bytes
	for id := curLastStripe.id + 1; id <= newLastStripe.id; id++ {
		c.writeStripeAt(fmt.Sprintf("%s:%d", name, id), c.stripeSize-1, []byte("\x00"))
	}
	// fill curLastStripe with null byte if needed
	if (curLastStripe.off + curLastStripe.len) < c.stripeSize {
		c.writeStripeAt(fmt.Sprintf("%s:%d", name, curLastStripe.id), c.stripeSize-1, []byte("\x00"))
	}
	Try(c.redisClient.Set(name+":size", size, 0).Err())
}

// Close ...
func (c FileContentClient) Close() error {
	c.writePool.Close()
	return c.redisClient.Close()
}
