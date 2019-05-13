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
// Redis client and clients ring implementations

package redisfs

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cea-hpc/pdwfs/config"
	"github.com/cea-hpc/pdwfs/redigo/redis"
	"github.com/cea-hpc/pdwfs/util"
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

// RedisClient is a client to a single Redis instance, safe to use by multiple goroutines
type RedisClient struct {
	pool *redis.Pool
}

// NewRedisClient creates a new RedisClient instance
func NewRedisClient(addr string) *RedisClient {
	return &RedisClient{
		pool: &redis.Pool{
			MaxIdle:     5,
			MaxActive:   50,   // max active connection at the same time
			Wait:        true, // throttles goroutines to MaxActive goroutines
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", addr)
			},
		},
	}
}

// Close the connection pool
func (c *RedisClient) Close() error {
	return c.pool.Close()
}

// the following methods implements some type-safe and concurrent-safe Redis commands

// SetRange command
func (c *RedisClient) SetRange(key string, offset int64, data []byte) error {
	conn := c.pool.Get()
	defer conn.Close()
	return err(conn.Do("SETRANGE", key, offset, data))
}

// GetRange command
func (c *RedisClient) GetRange(key string, start, end int64) ([]byte, error) {
	conn := c.pool.Get()
	defer conn.Close()
	b, err := redis.Bytes(conn.Do("GETRANGE", key, start, end))
	if err == redis.ErrNil {
		return b, ErrRedisKeyNotFound
	}
	return b, err
}

// Exists command
func (c *RedisClient) Exists(key string) (bool, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Bool(conn.Do("EXISTS", key))
}

// Set command
func (c *RedisClient) Set(key string, data []byte) error {
	conn := c.pool.Get()
	defer conn.Close()
	return err(conn.Do("SET", key, data))
}

// SetNX command
func (c *RedisClient) SetNX(key string, data []byte) error {
	conn := c.pool.Get()
	defer conn.Close()
	return err(conn.Do("SETNX", key, data))
}

// Get command
func (c *RedisClient) Get(key string) ([]byte, error) {
	conn := c.pool.Get()
	defer conn.Close()
	b, err := redis.Bytes(conn.Do("GET", key))
	if err == redis.ErrNil {
		return b, ErrRedisKeyNotFound
	}
	return b, err
}

// Unlink command
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

// SAdd command
func (c *RedisClient) SAdd(key string, member string) error {
	conn := c.pool.Get()
	defer conn.Close()
	return err(conn.Do("SADD", key, member))
}

// SRem command
func (c *RedisClient) SRem(key string, member string) error {
	conn := c.pool.Get()
	defer conn.Close()
	return err(conn.Do("SREM", key, member))
}

// SMembers command
func (c *RedisClient) SMembers(key string) ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", key))
}

// Pipe wraps the Redis pipeline feature of redigo
type Pipe struct {
	conn redis.Conn
}

// Do registers a new command in the pipeline
func (p Pipe) Do(cmd string, args ...interface{}) {
	Try(p.conn.Send(cmd, args...))
}

// Flush flushes all pipeline commands to Redis
func (p Pipe) Flush() {
	defer p.conn.Close()
	_, err := p.conn.Do("EXEC")
	Check(err)
}

// Pipeline returns a Pipe instance
func (c *RedisClient) Pipeline() *Pipe {
	conn := c.pool.Get()
	conn.Send("MULTI")
	return &Pipe{conn}
}

// RedisRing manages multiple Redis instances and use consistent hashing to distribute the load
type RedisRing struct {
	clients map[string]*RedisClient
	hash    *util.ConsistentHash
}

// NewRedisRing returns a new RedisRing instance
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

// GetClient returns a client from the ring based on a key
// if the key has curly braces in it (e.g "{mydirectory}/file"), only the string within the braces is used
// in the hasing process to get a client
func (r *RedisRing) GetClient(key string) *RedisClient {
	k := key
	if s := strings.IndexByte(key, '{'); s > -1 {
		if e := strings.IndexByte(key[s+1:], '}'); e > 0 {
			k = key[s+1 : s+e+1]
		}
	}
	return r.clients[r.hash.Get(k)]
}

// Close all clients in the ring
func (r *RedisRing) Close() error {
	var err error
	for _, client := range r.clients {
		err = client.Close()
	}
	return err
}
