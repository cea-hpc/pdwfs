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
	"testing"

	"github.com/cea-hpc/pdwfs/util"
)

func TestUnlinkMultiKeys(t *testing.T) {
	server, conf := util.InitRedisTestServer()
	defer server.Stop()

	client := NewRedisClient(conf.Addrs[0])
	defer client.Close()

	err := client.Set("foo", []byte("bar"))
	util.Ok(t, err)

	err = client.Set("foo2", []byte("bar2"))
	util.Ok(t, err)

	ok, err := client.Exists("foo")
	util.Ok(t, err)
	util.Assert(t, ok, "key should exist")

	ok, err = client.Exists("foo2")
	util.Ok(t, err)
	util.Assert(t, ok, "key should exist")

	err = client.Unlink("foo", "foo2")
	util.Ok(t, err)

	ok, err = client.Exists("foo")
	util.Ok(t, err)
	util.Assert(t, !ok, "key should not exist")

	ok, err = client.Exists("foo2")
	util.Ok(t, err)
	util.Assert(t, !ok, "key should not exist")
}
