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
	"sync/atomic"

	"github.com/gomodule/redigo/redis"
)

// DataStore ...
type DataStore struct {
	ring       *RedisRing
	stripeSize int64
}

// NewDataStore returns a redis client
func NewDataStore(ring *RedisRing, stripeSize int64) *DataStore {
	return &DataStore{
		ring:       ring,
		stripeSize: stripeSize,
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

// WriteAt ...
func (s DataStore) WriteAt(name string, off int64, data []byte) {
	dataLen := int64(len(data))
	stripes := stripeLayout(s.stripeSize, off, dataLen)
	var k int64
	wg := sync.WaitGroup{}
	for _, stripe := range stripes {
		if len(data) == 0 {
			continue
		}
		wg.Add(1)
		go func(key string, off int64, data []byte) {
			defer wg.Done()
			if off == 0 && int64(len(data)) == s.stripeSize {
				s.ring.Set(key, data)
			} else {
				s.ring.SetRange(key, off, data)
			}
		}(fmt.Sprintf("%s:%d", name, stripe.id), stripe.off, data[k:k+stripe.len])
		k += stripe.len
	}
	wg.Wait()
	Try(s.setSizeIfMax(name, off+dataLen))
}

// ReadAt ...
func (s DataStore) ReadAt(name string, off int64, dst []byte) int64 {
	var k, n int64
	stripes := stripeLayout(s.stripeSize, off, int64(len(dst)))
	wg := sync.WaitGroup{}
	for _, stripe := range stripes {
		wg.Add(1)
		go func(key string, off, size int64, dst []byte) {
			defer wg.Done()
			var res []byte
			var err error
			if off == 0 && size == s.stripeSize {
				res, err = s.ring.Get(key)
			} else {
				res, err = s.ring.GetRange(key, off, off+size-1)
			}
			if err != nil && err != ErrRedisKeyNotFound {
				panic(err)
			}
			read := copy(dst, res)
			atomic.AddInt64(&n, int64(read))
		}(fmt.Sprintf("%s:%d", name, stripe.id), stripe.off, stripe.len, dst[k:k+stripe.len])
		k += stripe.len
	}
	wg.Wait()
	return n
}

func (s DataStore) removeStripe(key string) {
	Try(s.ring.Unlink(key))
}

func (s DataStore) setSize(name string, size int64) {
	Try(s.ring.Set(name+":size", []byte(strconv.FormatInt(size, 10))))
}

var setSizeIfMaxScript = redis.NewScript(1, `
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

func (s DataStore) setSizeIfMax(name string, size int64) error {
	key := name + ":size"
	client := s.ring.GetClient(key)
	conn := client.pool.Get()
	defer conn.Close()
	return err(setSizeIfMaxScript.Do(conn, key, size))
}

// GetSize ...
func (s DataStore) GetSize(name string) int64 {
	res, err := s.ring.Get(name + ":size")
	if err != nil && err == ErrRedisKeyNotFound {
		return 0
	}
	Check(err)
	size, err := strconv.ParseInt(string(res), 10, 64)
	Check(err)
	return size
}

// Remove ...
func (s DataStore) Remove(name string) {
	stripes := stripeLayout(s.stripeSize, 0, s.GetSize(name))
	for _, stripe := range stripes {
		s.removeStripe(fmt.Sprintf("%s:%d", name, stripe.id))
	}
	Try(s.ring.Unlink(name + ":size"))
}

// Resize ...
func (s DataStore) Resize(name string, size int64) {
	if size < 0 {
		panic(fmt.Errorf("size must be non-negative"))
	}
	curSize := s.GetSize(name)
	switch {
	case size == curSize:
	case size < curSize:
		s.shrink(name, size)
	default:
		s.grow(name, size)
	}
}

var trimStrScript = redis.NewScript(1, `
		local str = redis.call("GETRANGE", KEYS[1], 0, ARGV[1])
		return redis.call("SET", KEYS[1], str)
	`)

func (s DataStore) trimStr(key string, size int64) {
	client := s.ring.GetClient(key)
	conn := client.pool.Get()
	defer conn.Close()
	Try(err(trimStrScript.Do(conn, key, size)))
}

func (s DataStore) shrink(name string, size int64) {
	curStripes := stripeLayout(s.stripeSize, 0, s.GetSize(name))
	curLastStripe := curStripes[len(curStripes)-1]
	newStripes := stripeLayout(s.stripeSize, 0, size)
	newLastStripe := newStripes[len(newStripes)-1]
	// remove all existing stripes after this new last stripe
	for id := newLastStripe.id + 1; id <= curLastStripe.id; id++ {
		s.removeStripe(fmt.Sprintf("%s:%d", name, id))
	}
	// resize the last stripe
	stripeKey := fmt.Sprintf("%s:%d", name, newLastStripe.id)
	s.trimStr(stripeKey, newLastStripe.len-1)
	s.setSize(name, size)
}

func (s DataStore) grow(name string, size int64) {
	curStripes := stripeLayout(s.stripeSize, 0, s.GetSize(name))
	curLastStripe := curStripes[len(curStripes)-1]
	newStripes := stripeLayout(s.stripeSize, 0, size)
	newLastStripe := newStripes[len(newStripes)-1]
	// fill new stripes with null bytes
	for id := curLastStripe.id + 1; id <= newLastStripe.id; id++ {
		key := fmt.Sprintf("%s:%d", name, id)
		Try(s.ring.SetRange(key, s.stripeSize-1, []byte("\x00")))
	}
	// fill curLastStripe with null byte if needed
	if (curLastStripe.off + curLastStripe.len) < s.stripeSize {
		key := fmt.Sprintf("%s:%d", name, curLastStripe.id)
		Try(s.ring.SetRange(key, s.stripeSize-1, []byte("\x00")))
	}
	s.setSize(name, size)
}

// Close ...
func (s DataStore) Close() error {
	return s.ring.Close()
}
