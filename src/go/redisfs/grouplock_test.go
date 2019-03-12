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
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/cea-hpc/pdwfs/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const testRedisKey = "__redis_lock_unit_test__"

var _ = Describe("GroupLock", func() {

	var newLock = func() *GroupLock {
		return NewGroupLock(redisClient, testRedisKey, &GroupLockOptions{
			RetryCount:  4,
			RetryDelay:  25 * time.Millisecond,
			LockTimeout: time.Second,
		})
	}

	var newProducerLock = func() *GroupLock {
		return NewGroupLock(redisClient, testRedisKey, &GroupLockOptions{
			RetryCount:       0,
			LockTimeout:      0 * time.Second,
			Group:            "Producer",
			PreviousGroup:    "",
			KeepKeyOnRelease: true,
		})
	}

	var newConsumerLock = func() *GroupLock {
		return NewGroupLock(redisClient, testRedisKey, &GroupLockOptions{
			RetryCount:       0,
			LockTimeout:      0 * time.Second,
			Group:            "Consumer",
			PreviousGroup:    "Producer",
			KeepKeyOnRelease: false,
		})
	}

	var getTTL = func() (time.Duration, error) {
		return redisClient.PTTL(testRedisKey).Result()
	}

	AfterEach(func() {
		Expect(redisClient.Del(testRedisKey).Err()).NotTo(HaveOccurred())
	})

	It("should normalize options", func() {
		locker := NewGroupLock(redisClient, testRedisKey, &GroupLockOptions{
			RetryCount: -1,
			RetryDelay: -1,
		})
		Expect(locker.opts).To(Equal(GroupLockOptions{
			LockTimeout:      0 * time.Second,
			RetryCount:       0,
			RetryDelay:       100 * time.Millisecond,
			Group:            locker.opts.Group, // test voided since it's a random token
			PreviousGroup:    "",
			KeepKeyOnRelease: false,
		}))
	})

	It("should be unlocked when created", func() {
		locker := newLock()
		Expect(locker.IsLocked()).To(BeFalse())
	})

	It("should fail obtain with error", func() {
		locker := newLock()
		locker.Lock()
		defer locker.Unlock()

		_, err := Obtain(redisClient, testRedisKey, nil)
		Expect(err).To(Equal(ErrLockNotObtained))
	})

	It("should obtain through short-cut", func() {
		locker := newLock()
		Expect(Obtain(redisClient, testRedisKey, nil)).To(BeAssignableToTypeOf(locker))
	})

	It("should obtain fresh locks", func() {
		locker := newLock()
		Expect(locker.Lock()).To(BeTrue())
		Expect(locker.IsLocked()).To(BeTrue())

		val := "[\"" + locker.opts.Group + "\",1]"
		Expect(redisClient.Get(testRedisKey).Result()).To(Equal(string(val)))
		Expect(getTTL()).To(BeNumerically("~", time.Second, 10*time.Millisecond))
	})

	It("should retry if enabled", func() {
		locker := newLock()
		Expect(redisClient.Set(testRedisKey, "[\"SomeGroup\",1]", 0).Err()).NotTo(HaveOccurred())
		Expect(redisClient.PExpire(testRedisKey, 30*time.Millisecond).Err()).NotTo(HaveOccurred())

		Expect(locker.Lock()).To(BeTrue())
		Expect(locker.IsLocked()).To(BeTrue())

		Expect(redisClient.Get(testRedisKey).Result()).To(Equal("[\"" + locker.opts.Group + "\",1]"))
		Expect(getTTL()).To(BeNumerically("~", time.Second, 10*time.Millisecond))
	})

	It("should not retry if not enabled", func() {
		locker := newLock()
		Expect(redisClient.Set(testRedisKey, "[\"SomeGroup\",1]", 0).Err()).NotTo(HaveOccurred())
		Expect(redisClient.PExpire(testRedisKey, 150*time.Millisecond).Err()).NotTo(HaveOccurred())
		locker.opts.RetryCount = 0

		Expect(locker.Lock()).To(BeFalse())
		Expect(locker.IsLocked()).To(BeFalse())
		Expect(getTTL()).To(BeNumerically("~", 150*time.Millisecond, 10*time.Millisecond))
	})

	It("should give up when retry count reached", func() {
		locker := newLock()
		Expect(redisClient.Set(testRedisKey, "[\"SomeGroup\",1]", 0).Err()).NotTo(HaveOccurred())
		Expect(redisClient.PExpire(testRedisKey, 150*time.Millisecond).Err()).NotTo(HaveOccurred())

		Expect(locker.Lock()).To(BeFalse())
		Expect(locker.IsLocked()).To(BeFalse())

		Expect(redisClient.Get(testRedisKey).Result()).To(Equal("[\"SomeGroup\",1]"))
		Expect(getTTL()).To(BeNumerically("~", 45*time.Millisecond, 20*time.Millisecond))
	})

	It("should release own locks", func() {
		locker := newLock()
		Expect(locker.Lock()).To(BeTrue())
		Expect(locker.IsLocked()).To(BeTrue())

		Expect(locker.Unlock()).NotTo(HaveOccurred())
		Expect(locker.IsLocked()).To(BeFalse())
		Expect(redisClient.Get(testRedisKey).Err()).To(Equal(redis.Nil))
	})

	It("should failure on release expired lock", func() {
		locker := newLock()
		Expect(locker.Lock()).To(BeTrue())
		Expect(locker.IsLocked()).To(BeTrue())

		time.Sleep(locker.opts.LockTimeout * 2)

		err := locker.Unlock()
		Expect(err).To(Equal(ErrLockUnlockFailed))
	})

	It("should not release someone else's locks", func() {
		locker := newLock()
		Expect(redisClient.Set(testRedisKey, "[\"SomeGroup\",1]", 0).Err()).NotTo(HaveOccurred())
		Expect(locker.IsLocked()).To(BeFalse())

		err := locker.Unlock()
		Expect(err).To(Equal(ErrLockUnlockFailed))
		Expect(locker.IsLocked()).To(BeFalse())
		Expect(redisClient.Get(testRedisKey).Val()).To(Equal("[\"SomeGroup\",1]"))
	})

	It("should refresh locks", func() {
		locker := newLock()
		Expect(locker.Lock()).To(BeTrue())
		Expect(locker.IsLocked()).To(BeTrue())

		time.Sleep(50 * time.Millisecond)
		Expect(getTTL()).To(BeNumerically("~", 950*time.Millisecond, 10*time.Millisecond))

		Expect(locker.Lock()).To(BeTrue())
		Expect(locker.IsLocked()).To(BeTrue())
		Expect(getTTL()).To(BeNumerically("~", time.Second, 10*time.Millisecond))
	})

	It("should re-create expired locks on refresh", func() {
		locker := newLock()
		Expect(locker.Lock()).To(BeTrue())
		Expect(locker.IsLocked()).To(BeTrue())

		Expect(redisClient.Del(testRedisKey).Err()).NotTo(HaveOccurred())

		Expect(locker.Lock()).To(BeTrue())
		Expect(locker.IsLocked()).To(BeTrue())
		Expect(getTTL()).To(BeNumerically("~", time.Second, 10*time.Millisecond))
	})

	It("should not re-capture expired locks acquiredby someone else", func() {
		locker := newLock()
		Expect(locker.Lock()).To(BeTrue())
		Expect(locker.IsLocked()).To(BeTrue())
		Expect(redisClient.Set(testRedisKey, "[\"SomeGroup\",1]", 0).Err()).NotTo(HaveOccurred())

		Expect(locker.Lock()).To(BeFalse())
		Expect(locker.IsLocked()).To(BeFalse())
	})

	It("should prevent multiple locks (fuzzing)", func() {
		res := int32(0)
		wg := new(sync.WaitGroup)
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()

				locker := newLock()
				wait := rand.Int63n(int64(50 * time.Millisecond))
				time.Sleep(time.Duration(wait))

				ok, err := locker.Lock()
				if err != nil {
					atomic.AddInt32(&res, 100)
					return
				} else if !ok {
					return
				}
				atomic.AddInt32(&res, 1)
			}()
		}
		wg.Wait()
		Expect(res).To(Equal(int32(1)))
	})

	It("should error when lock time exceeded while running handler", func() {
		err := Run(redisClient, testRedisKey, &GroupLockOptions{LockTimeout: time.Millisecond}, func() {
			time.Sleep(time.Millisecond * 5)
		})

		Expect(err).To(Equal(ErrLockDurationExceeded))
	})

	It("should retry and wait for locks if requested", func() {
		var (
			wg  sync.WaitGroup
			res int32
		)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer GinkgoRecover()

			opts := &GroupLockOptions{
				LockTimeout: 5 * time.Second,
				RetryCount:  10,
				RetryDelay:  10 * time.Millisecond,
			}

			err := Run(redisClient, testRedisKey, opts, func() {
				atomic.AddInt32(&res, 1)
			})
			Expect(err).NotTo(HaveOccurred())
		}()

		err := Run(redisClient, testRedisKey, nil, func() {
			atomic.AddInt32(&res, 1)
			time.Sleep(20 * time.Millisecond)
		})
		wg.Wait()

		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(int32(2)))
	})

	It("should give up retrying after timeout", func() {
		var (
			wg  sync.WaitGroup
			res int32
		)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer GinkgoRecover()

			opts := &GroupLockOptions{
				LockTimeout: 5 * time.Second,
				RetryCount:  1,
				RetryDelay:  10 * time.Millisecond,
			}

			err := Run(redisClient, testRedisKey, opts, func() {
				atomic.AddInt32(&res, 1)
			})
			Expect(err).To(Equal(ErrLockNotObtained))
		}()

		err := Run(redisClient, testRedisKey, nil, func() {
			atomic.AddInt32(&res, 1)
			time.Sleep(100 * time.Millisecond)
		})
		wg.Wait()

		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal(int32(1)))
	})

	// TESTING GROUP LOCKING FEATURES ...

	It("should re-acquire from the same Group", func() {
		pLock1 := newProducerLock()
		Expect(pLock1.Lock()).To(BeTrue())
		Expect(pLock1.IsLocked()).To(BeTrue())

		pLock2 := newProducerLock()
		Expect(pLock2.Lock()).To(BeTrue())
		Expect(pLock2.IsLocked()).To(BeTrue())
	})

	It("should not acquire from a different Group", func() {
		pLock := newProducerLock()
		Expect(pLock.Lock()).To(BeTrue())
		Expect(pLock.IsLocked()).To(BeTrue())

		cLock := newConsumerLock()
		Expect(cLock.Lock()).To(BeFalse())
		Expect(cLock.IsLocked()).To(BeFalse())
	})

	It("should not acquire if previous Group never acquired", func() {
		cLock := newConsumerLock()
		Expect(cLock.Lock()).To(BeFalse())
		Expect(cLock.IsLocked()).To(BeFalse())
	})

	It("should acquire when previous Group released", func() {
		pLock1 := newProducerLock()
		Expect(pLock1.Lock()).To(BeTrue())
		Expect(pLock1.IsLocked()).To(BeTrue())

		pLock2 := newProducerLock()
		Expect(pLock2.Lock()).To(BeTrue())
		Expect(pLock2.IsLocked()).To(BeTrue())

		Expect(pLock2.Unlock()).NotTo(HaveOccurred())
		Expect(pLock2.IsLocked()).To(BeFalse())

		cLock := newConsumerLock()
		Expect(cLock.Lock()).To(BeFalse())

		Expect(pLock1.Unlock()).NotTo(HaveOccurred())
		Expect(pLock1.IsLocked()).To(BeFalse())

		Expect(cLock.Lock()).To(BeTrue())
		Expect(cLock.IsLocked()).To(BeTrue())
	})

	It("should store the number of remaining holders", func() {
		pLock1 := newProducerLock()
		Expect(pLock1.Lock()).To(BeTrue())
		Expect(pLock1.holders).To(Equal(1))

		pLock2 := newProducerLock()
		Expect(pLock2.Lock()).To(BeTrue())
		Expect(pLock2.holders).To(Equal(2))

		Expect(pLock2.Unlock()).NotTo(HaveOccurred())
		Expect(pLock2.holders).To(Equal(1))

		cLock := newConsumerLock()
		Expect(cLock.Lock()).To(BeFalse())
		Expect(cLock.holders).To(Equal(0))

		Expect(pLock1.Unlock()).NotTo(HaveOccurred())
		Expect(pLock1.holders).To(Equal(0))

		Expect(cLock.Lock()).To(BeTrue())
		Expect(cLock.holders).To(Equal(1))
	})

})

// --------------------------------------------------------------------

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	AfterEach(func() {
		Expect(redisClient.Del(testRedisKey).Err()).NotTo(HaveOccurred())
	})
	RunSpecs(t, "redis-lock")
}

var redisClient IRedisClient

var _ = BeforeSuite(func() {
	conf := &config.Redis{RedisAddrs: []string{"127.0.0.1:6379"}}
	redisClient = NewRedisClient(conf)
	Expect(redisClient.Ping().Err()).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(redisClient.Close()).To(Succeed())
})
