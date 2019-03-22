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
	"errors"
	"fmt"
	"unsafe"

	"github.com/cea-hpc/pdwfs/config"
	"github.com/go-redis/redis"
)

const maxRedisString = 512 * 1024 * 1024 // 512MB

var (
	// ErrEndOfBuffer is thrown when read past the buffer size
	ErrEndOfBuffer = errors.New("End of Buffer")
	// ErrMaxRedisString is thrown when a string larger than 512MB is being written
	ErrMaxRedisString = errors.New("Maximum Redis String of 512MB reached")
)

// RedisBuffer is a Buffer in memory working on a slice of bytes.
type RedisBuffer struct {
	//FIXME: conf is currently only use for the BlockSize, consider passing the BlockSize directly
	// instead of MountPathConfig which is irrelevant for the most part of RedisBuffer
	conf    *config.Mount
	redis   IRedisClient
	key     string
	bufSize int
}

// NewRedisBuffer creates a new data volume based on a Buffer
func NewRedisBuffer(conf *config.Mount, client IRedisClient, key string) *RedisBuffer {
	if conf.BlockSize == 0 {
		panic(fmt.Errorf("BlockSize in configuration is not set"))
	}
	return &RedisBuffer{
		conf:    conf,
		redis:   client,
		key:     key,
		bufSize: conf.BlockSize,
	}
}

// Size returns the length of the Buffer
func (b *RedisBuffer) Size() int64 {
	n, err := b.redis.StrLen(b.key).Result()
	if err != nil {
		panic(err)
	}
	return n
}

func byte2StringNoCopy(b []byte) string {
	// is this really not copying b ??
	return *(*string)(unsafe.Pointer(&b))
}

func string2byteNoCopy(s string) []byte {
	// is this really not copying s ??
	return *(*[]byte)(unsafe.Pointer(&s))
}

func (b *RedisBuffer) writeString(off int64, data string) (int, error) {
	newLen, err := b.redis.SetRange(b.key, off, data).Result()
	if err != nil {
		return 0, err
	}
	dataLen := len(data)
	if (off + int64(dataLen)) > newLen {
		// FIXME: to be tested
		return int(newLen - off), ErrMaxRedisString
	}
	return dataLen, nil

}

//WriteAt writes data to the Buffer starting at byte offset off.
func (b *RedisBuffer) WriteAt(data []byte, off int64) (int, error) {
	return b.writeString(off, byte2StringNoCopy(data))
}

//WriteVecAt writes a vector of byte slices to the Buffer starting at byte offset off.
func (b *RedisBuffer) WriteVecAt(datav [][]byte, off int64) (int, error) {
	var n int
	for _, data := range datav {
		wrote, err := b.writeString(off, byte2StringNoCopy(data))
		if err != nil {
			return n, err
		}
		off += int64(wrote)
		n += wrote
	}
	return n, nil
}

/*
//WriteVecAt writes a vector of byte slices to the Buffer starting at byte offset off.
func (b *RedisBuffer) WriteVecAt(datav [][]byte, off int64) (int, error) {
	var n int
	pipeRet := make([]*redis.IntCmd, len(datav))
	pipe := b.redis.Pipeline()
	for i, data := range datav {
		pipeRet[i] = pipe.SetRange(b.key, off+int64(n), byte2StringNoCopy(data))
		n += len(data)
	}
	_, err := pipe.Exec()
	if err != nil {
		return -1, err
	}

	newLen := pipeRet[len(datav)-1].Val() // total length of string after pipe execution
	if (off + int64(n)) > newLen {
		// all data has not been written, Redis string limit may be been reached
		// FIXME: to be tested
		return int(newLen - off), ErrMaxRedisString
	}
	return n, nil
}
*/

//Clear resets the Buffer to default capacity and zero length
func (b *RedisBuffer) Clear() error {
	return b.redis.Unlink(b.key).Err()
}

//ReadAt reads in dst from the Buffer starting at byte offset off
func (b *RedisBuffer) ReadAt(dst []byte, off int64) (int, error) {
	if off >= b.Size() {
		return 0, ErrEndOfBuffer
	}
	val, err := b.redis.GetRange(b.key, off, off+int64(len(dst))-1).Result()
	n := copy(dst, val) // can we avoid this copy ?
	if err != nil {
		return n, err
	}
	if n < len(dst) {
		return n, ErrEndOfBuffer
	}
	return n, nil
}

//ReadVecAt reads a vector of byte slices from the Buffer starting at byte offset off
func (b *RedisBuffer) ReadVecAt(dstv [][]byte, off int64) (int, error) {
	if off >= b.Size() {
		return 0, ErrEndOfBuffer
	}
	var n, ldstv int
	for _, dst := range dstv {
		val, err := b.redis.GetRange(b.key, off, off+int64(len(dst))-1).Result()
		n += copy(dst, val) // can we avoid this copy ?
		if err != nil {
			return n, err
		}
		off += int64(n)
		ldstv += len(dst)
	}
	if n < ldstv {
		return n, ErrEndOfBuffer
	}
	return n, nil
}

var trimStringScript = redis.NewScript(`
		local str = redis.call("GETRANGE", KEYS[1], 0, ARGV[1])
		return redis.call("SET", KEYS[1], str)
	`)

// Resize resizes the Buffer to a given size.
// It returns an error if the given size is negative.
// If the Buffer is larger than the specified size, the extra data is lost.
// If the Buffer is smaller, it is extended and the extended part (hole)
// reads as zero bytes.
func (b *RedisBuffer) Resize(size int64) error {
	if size < 0 {
		return errors.New("Resize: size must be non-negative")
	}
	bufSize := b.Size()
	if size == bufSize {
		return nil
	} else if size == 0 {
		return b.redis.Set(b.key, "", 0).Err()
	} else if size < bufSize {
		return trimStringScript.Run(b.redis, []string{b.key}, size-1).Err()
	}
	/* else size > bufSize */
	return b.redis.SetRange(b.key, size-1, "\x00").Err()
}
