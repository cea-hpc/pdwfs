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
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/cea-hpc/pdwfs/config"
	"github.com/gomodule/redigo/redis"
)

//Inode object
type Inode struct {
	mountConf   *config.Mount
	dataStore   *DataStore
	metaStore   IRedisClient
	path        string
	metaBaseKey string
	isDir       *bool
	mode        *os.FileMode
	mtx         *sync.RWMutex
}

//NewInode ...
func NewInode(mountConf *config.Mount, dataStore *DataStore, metaStore IRedisClient, path string) *Inode {
	return &Inode{
		mountConf:   mountConf,
		dataStore:   dataStore,
		metaStore:   metaStore,
		path:        path,
		mtx:         &sync.RWMutex{},
		metaBaseKey: "{" + path + "}", // curly braces define the key to use for hashing such that all metadata keys goes on the same instance
	}
}

func (i *Inode) exists() (bool, error) {
	if i.mode != nil {
		return true, nil
	}
	return i.metaStore.Exists(i.metaBaseKey + ":mode")
}

func (i *Inode) initMeta(isDir bool, mode os.FileMode) {
	// FIXME: use Redis pipeline
	if isDir {
		Try(i.metaStore.SetNX(i.metaBaseKey+":isDir", []byte("1")))
	} else {
		Try(i.metaStore.SetNX(i.metaBaseKey+":isDir", []byte("0")))
	}

	Try(i.metaStore.SetNX(i.metaBaseKey+":mode", []byte(strconv.FormatInt(int64(mode), 10))))
}

func (i *Inode) delMeta() {
	// FIXME: use Redis pipeline with multiple keys
	Try(i.metaStore.Unlink(i.metaBaseKey + ":children"))
	Try(i.metaStore.Unlink(i.metaBaseKey + ":isDir"))
	Try(i.metaStore.Unlink(i.metaBaseKey + ":mode"))
}

//IsDir ...
func (i *Inode) IsDir() bool {
	if i.isDir == nil {
		key := i.metaBaseKey + ":isDir"
		res, err := i.metaStore.Get(key)
		if err != nil && err == redis.ErrNil {
			panic(fmt.Errorf("key '%s' not found", key))
		}
		Check(err)
		isDir := string(res[0]) == "1"
		i.isDir = &isDir
	}
	return (*i.isDir)
}

//Mode returns the inode access mode
func (i *Inode) Mode() os.FileMode {
	if i.mode == nil {
		key := i.metaBaseKey + ":mode"
		val, err := i.metaStore.Get(key)
		if err != nil && err == redis.ErrNil {
			panic(fmt.Errorf("key '%s' not found", key))
		}
		Check(err)
		res, err := strconv.ParseInt(string(val), 10, 64)
		Check(err)
		m := os.FileMode(res)
		i.mode = &m
	}
	return (*i.mode)
}

//Path returns the Path of the file
func (i *Inode) Path() string {
	return i.path
}

//Name returns the inode base name (for os.FileInfo interface)
func (i *Inode) Name() string {
	return i.path
}

//Sys no op (to fulfill os.FileMode interface)
func (i *Inode) Sys() interface{} {
	return nil
}

// ModTime IS NOT IMPLEMENTED (here to fulfill the os.FileInfo interface)
func (i *Inode) ModTime() time.Time {
	return time.Now()
}

//Size returns the size of the file
func (i *Inode) Size() int64 {
	if i.IsDir() {
		return 0
	}
	return i.dataStore.GetSize(i.path)
}

func (i *Inode) childrenPath() []string {
	res, err := i.metaStore.SMembers(i.metaBaseKey + ":children")
	Check(err)
	return res
}

func (i *Inode) setChild(child *Inode) error {
	return i.metaStore.SAdd(i.metaBaseKey+":children", child.Path())
}

func (i *Inode) removeChild(child *Inode) error {
	return i.metaStore.SRem(i.metaBaseKey+":children", child.Path())
}

func (i *Inode) getChildren() ([]*Inode, error) {
	if !i.IsDir() {
		return nil, ErrNotDirectory
	}
	paths, err := i.metaStore.SMembers(i.metaBaseKey + ":children")
	Check(err)
	children := make([]*Inode, 0, len(paths))
	for _, path := range paths {
		children = append(children, NewInode(i.mountConf, i.dataStore, i.metaStore, path))
	}
	return children, nil
}

func (i *Inode) getFile(flag int) (File, error) {
	if i.IsDir() {
		return nil, ErrIsDirectory
	}

	if hasFlag(os.O_TRUNC, flag) {
		i.dataStore.Remove(i.path)
	}

	var f File = NewMemFile(i.dataStore, i.path, i.mtx)

	if hasFlag(os.O_APPEND, flag) {
		f.Seek(0, os.SEEK_END)
	} else {
		f.Seek(0, os.SEEK_SET)
	}
	if hasFlag(os.O_RDWR, flag) {
		return f, nil
	} else if hasFlag(os.O_WRONLY, flag) {
		f = &woFile{f}
	} else {
		f = &roFile{f}
	}

	return f, nil
}

func (i *Inode) remove() {
	if !i.IsDir() {
		i.dataStore.Remove(i.path)
	} else {
		if children, _ := i.getChildren(); children != nil {
			for _, child := range children {
				child.remove()
			}
		}
	}
	i.delMeta()
}
