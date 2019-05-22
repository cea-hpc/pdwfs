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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/cea-hpc/pdwfs/config"
	"github.com/cea-hpc/pdwfs/redigo/redis"
)

// error checking helpers

func try(err error) {
	if err != nil {
		panic(err)
	}
}

var check = try

// testing utilities

// assert, ok and equals are from https://github.com/benbjohnson/testing

// Assert fails the test if the condition is false.
func Assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// Ok fails the test if an err is not nil.
func Ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// Equals fails the test if exp is not equal to act.
func Equals(tb testing.TB, exp, act interface{}, msg string) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: %s\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, msg, exp, act)
		tb.FailNow()
	}
}

// RedisTestServer used to running tests
type RedisTestServer struct {
	cmd  *exec.Cmd
	port int
}

// NewRedisTestServer returns a RedisTestServer instance with an available port (not yet started)
func NewRedisTestServer() *RedisTestServer {
	// check redis-server binary is in PATH
	_, err := exec.LookPath("redis-server")
	check(err)
	port := 6379
	for {
		// find a free port
		if _, err := redis.Dial("tcp", fmt.Sprintf(":%d", port)); err == nil {
			port++
		}
		break
	}
	return &RedisTestServer{
		cmd:  exec.Command("redis-server", "--save", "\"\"", "--port", strconv.Itoa(port)),
		port: port,
	}
}

// Start the Redis server, make sure its up and running
func (r *RedisTestServer) Start() {
	try(r.cmd.Start())
	time.Sleep(50 * time.Millisecond)
	for {
		if conn, err := redis.Dial("tcp", fmt.Sprintf(":%d", r.port)); err == nil {
			conn.Close()
			return
		}
	}
}

// Stop the Redis server process
// FIXME: this call is used in defer statements in all tests to make sure the server is stopped
// after each test, however this is not always happening and the server stays up after some test failures
// which is annoying for the subsequent tests as it leads to uncontrolled behaviour...probably better to revert
// to miniredis
func (r *RedisTestServer) Stop() {
	if err := r.cmd.Process.Signal(os.Interrupt); err != nil {
		panic(err)
	}
	r.cmd.Wait()
}

//InitRedisTestServer returns a new Redis test server with its configuration for clients
func InitRedisTestServer() (*RedisTestServer, *config.Redis) {
	server := NewRedisTestServer()
	server.Start()
	conf := config.NewRedisConf()
	conf.Addrs = []string{fmt.Sprintf(":%d", server.port)}
	return server, conf
}

//GetMountPathConf returns a default configuration (for testing)
func GetMountPathConf() *config.Mount {
	cwd, err := filepath.Abs(".")
	if err != nil {
		panic(err)
	}
	return &config.Mount{
		Path:       cwd,
		StripeSize: config.DefaultStripeSize,
	}
}
