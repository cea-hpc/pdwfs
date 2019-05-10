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
	"strings"
	"time"

	"github.com/cea-hpc/pdwfs/config"
	"github.com/cea-hpc/pdwfs/util"
	"github.com/gomodule/redigo/redis"
)

var (
	// ErrRedisKeyNotFound is returned if a queried key in Redis is not found
	ErrRedisKeyNotFound = errors.New("Redis key not found")
)

// Try ...
func Try(err error) {
	if err != nil {
		panic(err)
	}
}

// Check is an alias for Try
var Check = Try

func err(a interface{}, err error) error {
	return err
}

// IRedisClient ...
type IRedisClient interface {
	Close() error
	SetRange(key string, offset int64, data []byte) error
	GetRange(key string, start, end int64) ([]byte, error)
	Exists(key string) (bool, error)
	Set(key string, data []byte) error
	SetNX(key string, data []byte) error
	Get(key string) ([]byte, error)
	Unlink(key string) error
	SAdd(key string, member string) error
	SRem(key string, member string) error
	SMembers(key string) ([]string, error)
}

// RedisClient is a client to a single Redis instance, safe to use by multiple goroutines
type RedisClient struct {
	pool *redis.Pool
}

// NewRedisClient creates a new RedisClient instance
func NewRedisClient(addr string) *RedisClient {
	return &RedisClient{
		pool: &redis.Pool{
			MaxIdle:     3,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", addr)
			},
		},
	}
}

// Close ...
func (c *RedisClient) Close() error {
	return c.pool.Close()
}

// SetRange ...
func (c *RedisClient) SetRange(key string, offset int64, data []byte) error {
	conn := c.pool.Get()
	defer conn.Close()
	return err(conn.Do("SETRANGE", key, offset, data))
}

// GetRange ...
func (c *RedisClient) GetRange(key string, start, end int64) ([]byte, error) {
	conn := c.pool.Get()
	defer conn.Close()
	b, err := redis.Bytes(conn.Do("GETRANGE", key, start, end))
	if err == redis.ErrNil {
		return b, ErrRedisKeyNotFound
	}
	return b, err
}

// Exists ...
func (c *RedisClient) Exists(key string) (bool, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Bool(conn.Do("EXISTS", key))
}

// Set ...
func (c *RedisClient) Set(key string, data []byte) error {
	conn := c.pool.Get()
	defer conn.Close()
	return err(conn.Do("SET", key, data))
}

// SetNX ...
func (c *RedisClient) SetNX(key string, data []byte) error {
	conn := c.pool.Get()
	defer conn.Close()
	return err(conn.Do("SETNX", key, data))
}

// Get ...
func (c *RedisClient) Get(key string) ([]byte, error) {
	conn := c.pool.Get()
	defer conn.Close()
	b, err := redis.Bytes(conn.Do("GET", key))
	if err == redis.ErrNil {
		return b, ErrRedisKeyNotFound
	}
	return b, err
}

// Unlink ...
func (c *RedisClient) Unlink(keys ...string) error {
	// convert slice of string in slice of interface{} ref: https://golang.org/doc/faq#convert_slice_of_interface
	k := make([]interface{}, len(keys))
	for i, v := range keys {
		k[i] = v
	}
	conn := c.pool.Get()
	defer conn.Close()
	return err(conn.Do("UNLINK", k...))
}

// SAdd ...
func (c *RedisClient) SAdd(key string, member string) error {
	conn := c.pool.Get()
	defer conn.Close()
	return err(conn.Do("SADD", key, member))
}

// SRem ...
func (c *RedisClient) SRem(key string, member string) error {
	conn := c.pool.Get()
	defer conn.Close()
	return err(conn.Do("SREM", key, member))
}

// SMembers ...
func (c *RedisClient) SMembers(key string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", key))
}

// RedisRing ...
type RedisRing struct {
	clients map[string]*RedisClient
	hash    *util.ConsistentHash
}

// NewRedisRing ...
func NewRedisRing(conf *config.Redis) *RedisRing {

	ids := make([]string, len(conf.Addrs))
	clients := make(map[string]*RedisClient)
	for i, addr := range conf.Addrs {
		ids[i] = fmt.Sprintf("%d", i)
		clients[ids[i]] = NewRedisClient(addr)
	}
	hash := util.NewConsistentHash(100, nil)
	hash.Add(ids...)

	return &RedisRing{
		clients: clients,
		hash:    hash,
	}
}

// GetClient ...
func (r *RedisRing) GetClient(key string) *RedisClient {
	k := key
	if s := strings.IndexByte(key, '{'); s > -1 {
		if e := strings.IndexByte(key[s+1:], '}'); e > 0 {
			k = key[s+1 : s+e+1]
		}
	}
	return r.clients[r.hash.Get(k)]
}

// Close ...
func (r *RedisRing) Close() error {
	var err error
	for _, client := range r.clients {
		err = client.Close()
	}
	return err
}

// SetRange ...
func (r *RedisRing) SetRange(key string, offset int64, data []byte) error {
	return r.GetClient(key).SetRange(key, offset, data)
}

// GetRange ...
func (r *RedisRing) GetRange(key string, start, end int64) ([]byte, error) {
	return r.GetClient(key).GetRange(key, start, end)
}

// Exists ...
func (r *RedisRing) Exists(key string) (bool, error) {
	return r.GetClient(key).Exists(key)
}

// Set ...
func (r *RedisRing) Set(key string, data []byte) error {
	return r.GetClient(key).Set(key, data)
}

// SetNX ...
func (r *RedisRing) SetNX(key string, data []byte) error {
	return r.GetClient(key).SetNX(key, data)
}

// Get ...
func (r *RedisRing) Get(key string) ([]byte, error) {
	return r.GetClient(key).Get(key)
}

// Unlink ...
func (r *RedisRing) Unlink(key string) error {
	return r.GetClient(key).Unlink(key)
}

// SAdd ...
func (r *RedisRing) SAdd(key string, member string) error {
	return r.GetClient(key).SAdd(key, member)
}

// SRem ...
func (r *RedisRing) SRem(key string, member string) error {
	return r.GetClient(key).SRem(key, member)
}

// SMembers ...
func (r *RedisRing) SMembers(key string) ([]string, error) {
	return r.GetClient(key).SMembers(key)
}
