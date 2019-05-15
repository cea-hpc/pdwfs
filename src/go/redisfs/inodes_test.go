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
	"os"
	"testing"

	"github.com/cea-hpc/pdwfs/util"
)

func TestInodeMeta(t *testing.T) {
	redis, confRedis := util.InitRedisTestServer()
	defer redis.Stop()

	confMount := util.GetMountPathConf()

	ring := NewRedisRing(confRedis)
	defer ring.Close()
	store := NewDataStore(ring, int64(confMount.StripeSize))
	defer store.Close()

	i := NewInode(store, ring, "id")

	res := i.exists()
	util.Equals(t, false, res, "no metadata expected")

	i.initMeta(true, 0600)

	res = i.exists()
	util.Equals(t, true, res, "metadata expected")

	i.initMeta(false, 0777) // should be a no op

	d := i.IsDir()
	util.Equals(t, d, true, "should be a dir")

	m := i.Mode()
	util.Equals(t, m, os.FileMode(0600), "should be 0600 mode")

	i.delMeta()
}
