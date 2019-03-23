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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//Mount point configuration
type Mount struct {
	Path          string
	BlockSize     int
	WriteParallel bool
	ReadParallel  bool
}

//Redis connection configuration
type Redis struct {
	RedisAddrs        []string
	RedisCluster      bool
	RedisClusterAddrs []string
}

//Pdwfs configuration
type Pdwfs struct {
	Mounts    map[string]*Mount
	RedisConf *Redis
}

func validateMountPath(path string) string {
	path, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	if _, err = os.Stat(path); os.IsExist(err) {
		entries, err := ioutil.ReadDir(path)
		if err != nil {
			panic(err)
		}
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
	}

	conf := Pdwfs{
		Mounts:    map[string]*Mount{},
		RedisConf: &defaultRedis,
	}

	if addrs := os.Getenv("PDWFS_REDIS"); addrs != "" {
		conf.RedisConf.RedisAddrs = strings.Split(addrs, ",")
	}

	if path := os.Getenv("PDWFS_MOUNTPATH"); path != "" {
		mount := Mount{
			Path:          path,
			BlockSize:     1 * 1024 * 1024, // 1MB
			WriteParallel: true,
			ReadParallel:  true,
		}
		conf.Mounts[path] = &mount
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
		if err != nil {
			panic(err)
		}
		defer jsonFile.Close()
		content, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal([]byte(content), &conf)
		if err != nil {
			panic(err)
		}
	}

	// Options normalization

	if val := os.Getenv("PDWFS_LOGS"); val == "" {
		log.SetOutput(ioutil.Discard)
	}
	log.SetFlags(log.Lshortfile)
	log.SetPrefix("[PDWFS] ")

	normalized := map[string]*Mount{}

	for path, conf := range conf.Mounts {
		conf.Path = validateMountPath(path)
		// NOTE: we may add a different Redis database index per mount point for isolation
		// (but not suitable with Redis Cluster)
		normalized[conf.Path] = conf
	}
	conf.Mounts = normalized

	return &conf
}

// Dump writes the configuration in a JSON file
func (c *Pdwfs) Dump() error {
	content, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("pdwfs.json", content, 0644)
	if err != nil {
		return err
	}
	return nil
}
