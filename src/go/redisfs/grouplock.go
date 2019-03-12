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
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

// GroupLockOptions describe the options for the lock
type GroupLockOptions struct {
	// The number of time the acquisition of a lock will be retried.
	// Default: 0 = do not retry
	RetryCount int

	// RetryDelay is the amount of time to wait between retries.
	// Default: 100ms
	RetryDelay time.Duration

	// Any entity belonging to the same Group can acquire the lock
	// Default: random token = only one can acquire it (normal behaviour)
	Group string

	// The lock can be acquired only if the PreviousGroup released it
	// Default: "" = no previous condition on lock acquisition (normal behaviour)
	PreviousGroup string

	// Whether to keep the lock redis key when all entities of a Group have released it
	// Default: false = lock key is deleted when released (normal behaviour)
	KeepKeyOnRelease bool

	// Maximum duration to lock a key for, redis key is deleted on timeout
	// This is mutually exclusive with KeepKeyOnRelease=true since we expect the key to live
	// even after release for the next Group to acquire
	// Default: 0 = no timeout
	LockTimeout time.Duration

	// Specify an max holders count, GroupLock will be released only if all
	// expected holders have acquired and released the lock
	// Default: 0 = disable feature (GroupLock is released when no one holds it)
	MaxExpectedHolders int
}

func (o *GroupLockOptions) normalize() *GroupLockOptions {
	if o.RetryCount < 0 {
		o.RetryCount = 0
	}
	if o.RetryDelay < 1 {
		o.RetryDelay = 100 * time.Millisecond
	}
	if o.Group == "" {
		var err error
		o.Group, err = randomToken()
		if err != nil {
			panic(err)
		}
	}
	if o.LockTimeout > 0 && o.KeepKeyOnRelease {
		panic(fmt.Errorf("LockTimeout and KeepKeyOnRelease are mutually exclusive"))
	}
	return o
}

var luaAcquire = redis.NewScript(
	`local filelock = redis.call("get", KEYS[1])
	local requesterGrp = ARGV[1]
	local previousGrp = ARGV[2]
	local ttl = ARGV[3]
	local maxHolders = tonumber(ARGV[4])
	if filelock then
		filelock = cjson.decode(filelock)
		local holderGrp = filelock[1]
		local n = filelock[2]
		if requesterGrp == holderGrp or (previousGrp == holderGrp and n == 0) then
			-- acquire the lock !
			if maxHolders == 0 then
				n = n + 1
			end
			redis.call("set", KEYS[1], cjson.encode({requesterGrp, n}))
			if tonumber(ttl) ~= 0 then
				redis.call("expire", KEYS[1], ttl)
			end
			return n
		else
			return -1
		end
	else
		if previousGrp == "" then
			-- acquire the lock !
			local n
			if maxHolders > 0 then
				n = maxHolders
			else
				n = 1
			end
			redis.call("set", KEYS[1], cjson.encode({requesterGrp, n}))
			if tonumber(ttl) ~= 0 then
				redis.call("expire", KEYS[1], ttl)
			end
			return n
		else
			return -1
		end
	end`)

var luaRefresh = redis.NewScript(
	`local filelock = redis.call("get", KEYS[1])
	local requesterGrp = ARGV[1]
	local ttl = ARGV[2]
	if filelock then
		filelock = cjson.decode(filelock)
		local holderGrp = filelock[1]
		local n = filelock[2]
		if requesterGrp ~= holderGrp then
			return -1
		end
		if tonumber(ttl) ~= 0 then
			redis.call("expire", KEYS[1], ttl)
		else
			redis.call("persist", KEYS[1])
		end
		return n
	else
		return -1
	end`)

var luaRelease = redis.NewScript(
	`local filelock = redis.call("get", KEYS[1])
	local requesterGrp = ARGV[1]
	local delete = tonumber(ARGV[2])
	if filelock then
		filelock = cjson.decode(filelock)
		local holderGrp = filelock[1]
		local n = filelock[2]
		if requesterGrp ~= holderGrp or n == 0 then
			return -1
		end
		n = n - 1
		if n == 0 and delete == 1 then
			redis.call("del", KEYS[1])
		else
			redis.call("set", KEYS[1], cjson.encode({requesterGrp, n}))
		end
		return n
	else
		return -1
	end`)

var emptyCtx = context.Background()

// ErrLockNotObtained may be returned by Obtain() and Run()
// if a lock could not be obtained.
var (
	ErrLockUnlockFailed     = errors.New("lock unlock failed")
	ErrLockNotObtained      = errors.New("lock not obtained")
	ErrLockDurationExceeded = errors.New("lock duration exceeded")
)

// RedisClient is a minimal client interface.
type RedisClient interface {
	SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(scripts ...string) *redis.BoolSliceCmd
	ScriptLoad(script string) *redis.StringCmd
}

// GroupLock allows (repeated) distributed locking.
type GroupLock struct {
	client   RedisClient
	key      string
	opts     GroupLockOptions
	isLocked bool

	//FIXME: holders aims at storing the number of remaining lock holders of the same group
	// it should instead be returned by to Lock/Obtain or Unlock, and not as a GroupLock attribute
	holders int
	mutex   sync.Mutex
}

// Run runs a callback handler with a Redis lock. It may return ErrLockNotObtained
// if a lock was not successfully acquired.
func Run(client RedisClient, key string, opts *GroupLockOptions, handler func()) error {
	locker, err := Obtain(client, key, opts)
	if err != nil {
		return err
	}

	sem := make(chan struct{})
	go func() {
		handler()
		close(sem)
	}()

	//FIXME: hack, if no timeout is specified, the case time.After(0) is reached immediately
	if locker.opts.LockTimeout == 0 {
		locker.opts.LockTimeout = math.MaxInt64
	}

	select {
	case <-sem:
		return locker.Unlock()
	case <-time.After(locker.opts.LockTimeout):
		return ErrLockDurationExceeded
	}
}

// Obtain is a shortcut for New().Lock(). It may return ErrLockNotObtained
// if a lock was not successfully acquired.
func Obtain(client RedisClient, key string, opts *GroupLockOptions) (*GroupLock, error) {
	locker := NewGroupLock(client, key, opts)
	if ok, err := locker.Lock(); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrLockNotObtained
	}
	return locker, nil
}

// NewGroupLock creates a new distributed locker on a given key.
func NewGroupLock(client RedisClient, key string, opts *GroupLockOptions) *GroupLock {
	var o GroupLockOptions
	if opts != nil {
		o = *opts
	}
	o.normalize()

	return &GroupLock{client: client, key: key, opts: o, isLocked: false}
}

// IsLocked returns true if a lock is still being held.
func (l *GroupLock) IsLocked() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.isLocked
}

// Lock applies the lock, don't forget to defer the Unlock() function to release the lock after usage.
func (l *GroupLock) Lock() (bool, error) {
	return l.LockWithContext(emptyCtx)
}

// LockWithContext is like Lock but allows to pass an additional context which allows cancelling
// lock attempts prematurely.
func (l *GroupLock) LockWithContext(ctx context.Context) (bool, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.isLocked {
		return l.refresh(ctx)
	}
	return l.create(ctx)
}

// Unlock releases the lock
func (l *GroupLock) Unlock() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.release()
}

// Helpers

func (l *GroupLock) create(ctx context.Context) (bool, error) {
	l.reset()

	// Calculate the timestamp we are willing to wait for
	attempts := l.opts.RetryCount + 1
	var retryDelay *time.Timer

	for {

		// Try to obtain a lock
		ok, err := l.obtain()
		if err != nil {
			return false, err
		} else if ok {
			l.isLocked = true
			return true, nil
		}

		if attempts--; attempts <= 0 {
			return false, nil
		}

		if retryDelay == nil {
			retryDelay = time.NewTimer(l.opts.RetryDelay)
			defer retryDelay.Stop()
		} else {
			retryDelay.Reset(l.opts.RetryDelay)
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-retryDelay.C:
		}
	}
}

func (l *GroupLock) refresh(ctx context.Context) (bool, error) {
	ttl := strconv.FormatInt(int64(l.opts.LockTimeout/time.Second), 10)
	res, err := luaRefresh.Run(l.client, []string{l.key}, l.opts.Group, ttl).Result()
	if err == redis.Nil {
		err = nil
	}
	if err != nil {
		return false, err
	}
	if i, ok := res.(int64); ok && i == -1 {
		return l.create(ctx)
	}
	return true, err
}

func (l *GroupLock) obtain() (bool, error) {
	ttl := strconv.FormatInt(int64(l.opts.LockTimeout/time.Second), 10)
	res, err := luaAcquire.Run(l.client, []string{l.key}, l.opts.Group, l.opts.PreviousGroup, ttl, l.opts.MaxExpectedHolders).Result()
	if err == redis.Nil {
		err = nil
	}
	n, ok := res.(int64)
	if !ok || n == -1 {
		return false, err
	}
	l.holders = int(n)
	return true, err
}

func (l *GroupLock) release() error {
	defer l.reset()

	delete := 1
	if l.opts.KeepKeyOnRelease {
		delete = 0
	}
	res, err := luaRelease.Run(l.client, []string{l.key}, l.opts.Group, delete).Result()
	if err == redis.Nil {
		err = nil
	}
	n, ok := res.(int64)
	if !ok || n == -1 {
		return ErrLockUnlockFailed
	}
	l.holders = int(n)
	return err
}

func (l *GroupLock) reset() {
	l.isLocked = false
}
