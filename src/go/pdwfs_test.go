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

package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/cea-hpc/pdwfs/config"
	"github.com/cea-hpc/pdwfs/util"
)

func writeFile(pdwfs *PdwFS, filename string, data []byte, perm os.FileMode) (int, error) {
	mount, err := pdwfs.getMount(filename)
	if err != nil {
		return -1, err
	}
	f, err := mount.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return -1, err
	}
	return f.Write(data)
}

func readFile(pdwfs *PdwFS, filename string) ([]byte, error) {
	mount, err := pdwfs.getMount(filename)
	if err != nil {
		return nil, err
	}
	f, err := mount.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

func TestMultiMount(t *testing.T) {

	redis, redisConf := util.InitRedisTestServer()
	defer redis.Stop()

	conf := config.New()
	conf.Redis = redisConf

	// create two fake mount paths
	conf.Mounts["/rebels/luke"] = &config.Mount{
		Path:       "/rebels/luke",
		StripeSize: 2 * 1024, // 2KB
	}
	conf.Mounts["/empire/vader"] = &config.Mount{
		Path:       "/empire/vader",
		StripeSize: 1024, // 1KB
	}
	pdwfs := NewPdwFS(conf)
	defer pdwfs.finalize()

	_, err := writeFile(pdwfs, "/rebels/luke/quotes", []byte("Vader's on that ship.\n"), os.FileMode(0777))
	util.Ok(t, err)

	_, err = writeFile(pdwfs, "/empire/vader/quotes", []byte("The Force is strong with this one.\n"), os.FileMode(0777))
	util.Ok(t, err)

	data, err := readFile(pdwfs, "/rebels/luke/quotes")
	util.Equals(t, "Vader's on that ship.\n", string(data), "Bad quote !")

	data, err = readFile(pdwfs, "/empire/vader/quotes")
	util.Equals(t, "The Force is strong with this one.\n", string(data), "Bad quote !")

}
