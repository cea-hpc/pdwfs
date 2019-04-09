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
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/cea-hpc/pdwfs/config"
)

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

//InitTestRedis returns a new miniredis server and client
func InitTestRedis() (*miniredis.Miniredis, IRedisClient, *config.Redis) {
	s, err := miniredis.Run()
	check(err)
	redisConf := &config.Redis{
		RedisAddrs: []string{s.Addr()},
		UseUnlink:  false, // miniredis does not support Unlink, defaults to Del
	}
	return s, NewRedisClient(redisConf), redisConf
}

//GetMountPathConf returns a default configuration (for testing)
func GetMountPathConf() *config.Mount {
	cwd, err := filepath.Abs(".")
	check(err)
	return &config.Mount{
		Path:      cwd,
		BlockSize: 1024 * 1024, // 1Mo
	}
}

// SplitPath splits the given path in segments:
// 	"/" 				-> []string{""}
//	"./file" 			-> []string{".", "file"}
//	"file" 				-> []string{".", "file"}
//	"/usr/src/linux/" 	-> []string{"", "usr", "src", "linux"}
// The returned slice of path segments consists of one more more segments.
func SplitPath(path string, sep string) []string {
	path = strings.TrimSpace(path)
	path = strings.TrimSuffix(path, sep)
	if path == "" { // was "/"
		return []string{""}
	}
	if path == "." {
		return []string{"."}
	}

	if len(path) > 0 && !strings.HasPrefix(path, sep) && !strings.HasPrefix(path, "."+sep) {
		path = "./" + path
	}
	parts := strings.Split(path, sep)

	return parts
}

func try(err error) {
	if err != nil {
		panic(err)
	}
}

var check = try
