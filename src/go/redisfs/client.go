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
	"time"

	"github.com/go-redis/redis"
	"github.com/cea-hpc/pdwfs/config"
)

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

// NewRedisClient returns a redis client matching the IRedisClient interface
func NewRedisClient(conf *config.Redis) IRedisClient {
	var client IRedisClient
	/*
		// single Redis client
		client := redis.NewClient(&redis.Options{
			Addr:     conf.RedisAddrs[0],
			PoolSize: 50,
			Password: "", // no password set
			DB:       0,  // use default DB
		})
	*/
	if conf.RedisCluster {
		// Redis cluster client
		//FIXME: cluter is not working...
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    conf.RedisClusterAddrs,
			ReadOnly: true,
		})
	} else {
		// Ring client
		addrs := make(map[string]string)
		for i, addr := range conf.RedisAddrs {
			addrs[fmt.Sprintf("shard%d", i)] = addr
		}
		client = redis.NewRing(&redis.RingOptions{Addrs: addrs})
	}
	return client
}

// Buffer is a linear addressable abstract structure to store data
type Buffer interface {
	WriteAt([]byte, int64) (int, error)
	ReadAt([]byte, int64) (int, error)
	WriteVecAt([][]byte, int64) (int, error)
	ReadVecAt([][]byte, int64) (int, error)
	Clear() error
	Size() int64
	Resize(int64) error
}
