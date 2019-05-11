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
//
// DataStore component that uses multiple Redis instances in a "ring"
// to store the content of files stripped accross the Redis instances

package redisfs

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gomodule/redigo/redis"
)

// helpers functions

// builds the key string use to address stripes in Redis
func key(name string, id int64) string {
	return fmt.Sprintf("%s:%d", name, id)
}

func divmod(n, d int64) (q, r int64) {
	q = n / d
	r = n % d
	return
}

// DataStore uses multiple Redis instances (ring) to store flat sequences of bytes stripped accross instances
type DataStore struct {
	ring       *RedisRing
	stripeSize int64
}

// NewDataStore returns a DataStore struct instance
func NewDataStore(ring *RedisRing, stripeSize int64) *DataStore {
	return &DataStore{
		ring:       ring,
		stripeSize: stripeSize,
	}
}

// Close the Redis clients in the ring
func (s DataStore) Close() error {
	return s.ring.Close()
}

type stripeInfo struct {
	id   int64
	off  int64  // offset relative to beginning of stripe
	data []byte // data in stripe starting at offset off
}

// returns a slice of stripeInfo struct describing the stripping of a sequence of bytes 'data'
// written or read from offset 'off'
func stripeLayout(stripeSize, off int64, data []byte) []stripeInfo {
	stripes := make([]stripeInfo, 0, 100)
	var id, offset, pos, stripeLen int64
loop:
	for {
		if pos >= int64(len(data)) {
			break loop
		}
		if len(stripes) >= cap(stripes) {
			// grow stripes
			newStripes := make([]stripeInfo, 2*cap(stripes))
			copy(newStripes, stripes)
			stripes = newStripes
		}
		id, offset = divmod(off+pos, stripeSize)
		if stripeLen = int64(len(data[pos:])); stripeLen > stripeSize-offset {
			stripeLen = stripeSize - offset
		}
		stripes = append(stripes, stripeInfo{id, offset, data[pos : pos+stripeLen]})
		pos += stripeLen
	}
	return stripes
}

// writes a single stripe in the store
// Note: each Redis instance in the store contains a set of all the stripes stored by that instance for a specific file
// this is used when searching the last stripe of a file to compute its size, see next methods
func (s DataStore) writeStripe(name string, stripe stripeInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	stripeKey := key(name, stripe.id)
	client := s.ring.GetClient(stripeKey)
	conn := client.pool.Get()
	defer conn.Close()

	// uses Redis pipeline feature MULTI/EXEC
	conn.Send("MULTI")
	conn.Send("SADD", name+":stripes", stripe.id)
	if stripe.off == 0 && int64(len(stripe.data)) == s.stripeSize {
		// SET is faster than SETRANGE
		conn.Send("SET", stripeKey, stripe.data)
	} else {
		conn.Send("SETRANGE", stripeKey, stripe.off, stripe.data)
	}
	_, err := conn.Do("EXEC")
	Check(err)
}

// erases the stripe from its instance
func (s DataStore) removeStripe(name string, id int64, wg *sync.WaitGroup) {
	defer wg.Done()
	stripeKey := key(name, id)
	client := s.ring.GetClient(stripeKey)
	conn := client.pool.Get()
	defer conn.Close()

	conn.Send("MULTI")
	conn.Send("SREM", name+":stripes", id)
	conn.Send("UNLINK", stripeKey)
	_, err := conn.Do("EXEC")
	Check(err)
}

// reads stripe data from its Redis instance, copy the data into the destination buffer
// and returns the number of bytes read
func (s DataStore) readStripe(name string, stripe stripeInfo, wg *sync.WaitGroup, read *int64) {
	defer wg.Done()
	stripeKey := key(name, stripe.id)
	client := s.ring.GetClient(stripeKey)
	conn := client.pool.Get()
	defer conn.Close()

	var res []byte
	var err error
	size := int64(len(stripe.data))
	if stripe.off == 0 && size == s.stripeSize {
		res, err = redis.Bytes(conn.Do("GET", stripeKey))
	} else {
		res, err = redis.Bytes(conn.Do("GETRANGE", stripeKey, stripe.off, stripe.off+size-1))
	}
	if err != nil && err != redis.ErrNil {
		panic(err)
	}
	// copy res into destination data buffer and atomically increment the number of bytes read
	atomic.AddInt64(read, int64(copy(stripe.data, res)))
}

var trimStripeScript = redis.NewScript(1, `
		local str = redis.call("GETRANGE", KEYS[1], 0, ARGV[1])
		return redis.call("SET", KEYS[1], str)
	`)

func (s DataStore) trimStripe(name string, id int64, size int64, wg *sync.WaitGroup) {
	defer wg.Done()
	stripeKey := key(name, id)
	client := s.ring.GetClient(stripeKey)
	conn := client.pool.Get()
	defer conn.Close()

	if size == 0 {
		_, err := conn.Do("SET", stripeKey, []byte(""))
		Check(err)
	} else {
		Try(err(trimStripeScript.Do(conn, stripeKey, size-1)))
	}
}

// main DataStore public API

// WriteAt writes the content of 'data' keyed by 'name' at offset 'off' into the DataStore
// the content is stripped and each stripe is written concurrently in its own goroutine
// Note: goroutines are throttled by the limited connection pools of each Redis instance
func (s DataStore) WriteAt(name string, off int64, data []byte) {
	wg := sync.WaitGroup{}
	for _, stripe := range stripeLayout(s.stripeSize, off, data) {
		wg.Add(1)
		go s.writeStripe(name, stripe, &wg)
	}
	wg.Wait()
}

// ReadAt reads data into 'dst' byte slice and returns the number of read bytes
func (s DataStore) ReadAt(name string, off int64, dst []byte) int64 {
	var read int64
	wg := sync.WaitGroup{}
	for _, stripe := range stripeLayout(s.stripeSize, off, dst) {
		wg.Add(1)
		go s.readStripe(name, stripe, &wg, &read)
	}
	wg.Wait()
	return read
}

// Remove all stripes keyed by 'name'
func (s DataStore) Remove(name string) {
	wg := sync.WaitGroup{}
	lastStripe := s.searchLastStripe(name)
	for i := int64(0); i <= lastStripe; i++ {
		wg.Add(1)
		go s.removeStripe(name, i, &wg)
	}
	wg.Wait()
}

// gather from all Redis instances the list of stripes keyed by 'name' and returns the highest stripe ID
func (s DataStore) searchLastStripe(name string) int64 {
	retChan := make(chan int64, len(s.ring.clients))
	wg := sync.WaitGroup{}
	for _, client := range s.ring.clients {
		wg.Add(1)
		go func(c *RedisClient, wg *sync.WaitGroup, ch chan int64) {
			defer wg.Done()
			conn := c.pool.Get()
			defer conn.Close()
			ids, err := redis.Int64s(conn.Do("SMEMBERS", name+":stripes"))
			Check(err)
			max := int64(-1)
			for _, id := range ids {
				if id > max {
					max = id
				}
			}
			ch <- max
		}(client, &wg, retChan)
	}
	wg.Wait()
	var n int64
	max := int64(-1)
	for i := 0; i < len(s.ring.clients); i++ {
		n = <-retChan
		if n > max {
			max = n
		}
	}
	return max
}

// GetSize returns the total size in bytes of data stored keyed by 'name' (all stripes).
func (s DataStore) GetSize(name string) int64 {
	ilast := s.searchLastStripe(name)
	if ilast < 0 {
		return 0
	}
	lastStripe, err := s.ring.Get(key(name, ilast))
	Check(err)
	return ilast*s.stripeSize + int64(len(lastStripe))
}

// helper to obtain the last stripe ID and length based on the total size and stripe size
func lastStripeInfo(size, stripeSize int64) (stripeID, stripeLen int64) {
	stripeID, stripeLen = divmod(size, stripeSize)
	if stripeLen == 0 {
		stripeID--
		stripeLen = stripeSize
	}
	return
}

// Resize (grow or shrink) the data content keyed by 'name'
func (s DataStore) Resize(name string, newSize int64) {
	if newSize < 0 {
		panic(fmt.Errorf("size must be non-negative"))
	}
	curSize := s.GetSize(name)
	curLastStripeID, curLastStripeLen := lastStripeInfo(curSize, s.stripeSize)
	newLastStripeID, newLastStripeLen := lastStripeInfo(newSize, s.stripeSize)
	switch {
	case newSize < curSize: // shrink
		// remove all existing stripes after this new last stripe
		wg := sync.WaitGroup{}
		for id := newLastStripeID + 1; id <= curLastStripeID; id++ {
			wg.Add(1)
			go s.removeStripe(name, id, &wg)
		}
		// resize the last stripe
		wg.Add(1)
		go s.trimStripe(name, newLastStripeID, newLastStripeLen, &wg)
		wg.Wait()

	case newSize > curSize: // grow
		// write new stripes but the last
		wg := sync.WaitGroup{}
		for id := curLastStripeID + 1; id < newLastStripeID; id++ {
			wg.Add(1)
			go s.writeStripe(name, stripeInfo{id, s.stripeSize - 1, []byte("\x00")}, &wg)
		}
		// write last stripe
		wg.Add(1)
		go s.writeStripe(name, stripeInfo{newLastStripeID, newLastStripeLen - 1, []byte("\x00")}, &wg)
		// fill current last stripe with null bytes if needed
		if curLastStripeLen < s.stripeSize {
			wg.Add(1)
			go s.writeStripe(name, stripeInfo{curLastStripeID, s.stripeSize - curLastStripeLen - 1, []byte("\x00")}, &wg)
		}
		wg.Wait()
	}
}
