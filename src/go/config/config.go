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

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const maxRedisString = 512 * 1024 * 1024 // 512MB

func try(err error) {
	if err != nil {
		panic(err)
	}
}

var check = try

//Mount point configuration
type Mount struct {
	Path      string
	BlockSize int
}

//Redis connection configuration
type Redis struct {
	RedisAddrs        []string
	RedisCluster      bool
	RedisClusterAddrs []string
	UseUnlink         bool
}

//Pdwfs configuration
type Pdwfs struct {
	Mounts    map[string]*Mount
	RedisConf *Redis
}

func validateMountPath(path string) string {
	path, err := filepath.Abs(path)
	check(err)
	if _, err = os.Stat(path); os.IsExist(err) {
		entries, err := ioutil.ReadDir(path)
		check(err)
		if len(entries) != 0 {
			log.Printf("WARNING mountPath '%s' is not empty, files will not be available for reading through pdwfs", path)
		}
	}
	return path
}

//New returns a new config object
func New() *Pdwfs {

	defaultRedis := Redis{
		RedisAddrs:        []string{":6379"},
		RedisCluster:      false,
		RedisClusterAddrs: []string{":7001", ":7002", ":7003", ":7004", ":7005", ":7006"},
		UseUnlink:         true,
	}

	conf := Pdwfs{
		Mounts:    map[string]*Mount{},
		RedisConf: &defaultRedis,
	}

	if addrs := os.Getenv("PDWFS_REDIS"); addrs != "" {
		s := strings.Split(addrs, ",")
		var a []string
		for _, i := range s {
			if i != "" {
				a = append(a, i)
			}
		}
		conf.RedisConf.RedisAddrs = a
	}

	if path := os.Getenv("PDWFS_MOUNTPATH"); path != "" {
		conf.Mounts[path] = &Mount{
			Path:      path,
			BlockSize: 10 * 1024 * 1024, // 10MB
		}
	}

	if blockSize := os.Getenv("PDWFS_BLOCKSIZE"); blockSize != "" {
		for _, mount := range conf.Mounts {
			size, err := strconv.Atoi(blockSize)
			if err != nil {
				log.Fatalln("Can't convert BlockSize in PDWFS_BLOCKSIZE to int")
			}
			mount.BlockSize = size * 1024 * 1024
		}
	}

	if confFile := os.Getenv("PDWFS_CONF"); confFile != "" {
		jsonFile, err := os.Open(confFile)
		check(err)
		defer jsonFile.Close()
		content, _ := ioutil.ReadAll(jsonFile)
		try(json.Unmarshal([]byte(content), &conf))
	}

	// Options verifications and normalization

	if val := os.Getenv("PDWFS_LOGS"); val == "" {
		log.SetOutput(ioutil.Discard)
	}
	log.SetFlags(log.Lshortfile)
	log.SetPrefix("[PDWFS] ")

	normalized := map[string]*Mount{}

	for path, conf := range conf.Mounts {
		conf.Path = validateMountPath(path)
		if conf.BlockSize > maxRedisString {
			err := fmt.Sprintf("Mount point '%s' block size (%dMB) is above what Redis can sustain, set block size <= 512MB", path, conf.BlockSize/(1024*1024))
			panic(err)
		}
		normalized[conf.Path] = conf
	}
	conf.Mounts = normalized

	return &conf
}

// Dump writes the configuration in a JSON file
func (c *Pdwfs) Dump() {
	content, err := json.MarshalIndent(c, "", "    ")
	check(err)
	try(ioutil.WriteFile("pdwfs.json", content, 0644))
}
