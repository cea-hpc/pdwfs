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
	"strings"
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

func TestGetInto(t *testing.T) {
	server, conf := util.InitRedisTestServer()
	defer server.Stop()

	client := NewRedisClient(conf.Addrs[0])
	defer client.Close()

	data := []byte("0123456789")

	err := client.Set("foo", data)
	util.Ok(t, err)

	// Destination buffer has same size as data
	b := make([]byte, 10)
	read, err := client.GetInto("foo", b)
	util.Ok(t, err)
	util.Equals(t, len(data), read, "wrong number of bytes read")
	util.Equals(t, data, b, "read data does not match written data")

	// Destination buffer has larger size
	b = make([]byte, 20)
	read, err = client.GetInto("foo", b)
	util.Ok(t, err)
	util.Equals(t, len(data), read, "wrong number of bytes read")
	util.Equals(t, data, b[:len(data)], "read data does not match written data")

	// Destination buffer with smaller size returns an error
	b = make([]byte, 5)
	read, err = client.GetInto("foo", b)
	util.Assert(t, err != nil, "should raise an error")
	util.Assert(t, strings.Contains(err.Error(), "destination buffer is too small"), "a different error is expected")
}

func TestGetRangeInto(t *testing.T) {
	server, conf := util.InitRedisTestServer()
	defer server.Stop()

	client := NewRedisClient(conf.Addrs[0])
	defer client.Close()

	data := []byte("0123456789")

	err := client.Set("foo", data)
	util.Ok(t, err)

	b := make([]byte, 5)
	read, err := client.GetRangeInto("foo", 4, 8, b)
	util.Ok(t, err)
	util.Equals(t, len(b), read, "wrong number of bytes read")
	util.Equals(t, data[4:9], b, "read data does not match written data")
}
