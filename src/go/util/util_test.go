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

package util

import (
	"testing"

	"github.com/gomodule/redigo/redis"
)

func TestRedisTestServer(t *testing.T) {
	server := NewRedisTestServer()
	server.Start()
	defer server.Stop()
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		panic(err)
	}
	_, err = conn.Do("SET", "foo", "bar")
	if err != nil {
		panic(err)
	}
	s, err := redis.String(conn.Do("GET", "foo"))
	if err != nil {
		panic(err)
	}
	Equals(t, "bar", s, "reply should be bar")
}
