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
	"strconv"
	"sync"
	"time"
)

//Inode object
type Inode struct {
	dataStore *DataStore
	redisRing *RedisRing
	path      string
	keyPrefix string
	mtx       *sync.RWMutex
}

//NewInode ...
func NewInode(dataStore *DataStore, ring *RedisRing, path string) *Inode {
	return &Inode{
		dataStore: dataStore,
		redisRing: ring,
		path:      path,
		mtx:       &sync.RWMutex{},
		keyPrefix: "{" + path + "}", // key prefix is in curly braces to ensure all metadata keys goes on the same instance (see RedisRing)
	}
}

func (i *Inode) exists() bool {
	client := i.redisRing.GetClient(i.keyPrefix)
	ret, err := client.Exists(i.keyPrefix + ":mode")
	Check(err)
	return ret
}

func (i *Inode) initMeta(isDir bool, mode os.FileMode) {
	pipeline := i.redisRing.GetClient(i.keyPrefix).Pipeline()
	if isDir {
		pipeline.Do("SADD", i.keyPrefix+":children", "")
	}
	pipeline.Do("SETNX", i.keyPrefix+":mode", []byte(strconv.FormatInt(int64(mode), 10)))
	pipeline.Flush()
}

func (i *Inode) delMeta() {
	pipeline := i.redisRing.GetClient(i.keyPrefix).Pipeline()
	pipeline.Do("UNLINK", i.keyPrefix+":children")
	pipeline.Do("UNLINK", i.keyPrefix+":mode")
	pipeline.Flush()
}

//IsDir ...
func (i *Inode) IsDir() bool {
	client := i.redisRing.GetClient(i.keyPrefix)
	res, err := client.Exists(i.keyPrefix + ":children")
	Check(err)
	return res
}

//Mode returns the inode access mode
func (i *Inode) Mode() os.FileMode {
	client := i.redisRing.GetClient(i.keyPrefix)
	val, err := client.Get(i.keyPrefix + ":mode")
	Check(err)
	res, err := strconv.ParseInt(string(val), 10, 64)
	Check(err)
	return os.FileMode(res)
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

func (i *Inode) setChild(child *Inode) {
	client := i.redisRing.GetClient(i.keyPrefix)
	Try(client.SAdd(i.keyPrefix+":children", child.Path()))
}

func (i *Inode) removeChild(child *Inode) {
	client := i.redisRing.GetClient(i.keyPrefix)
	Try(client.SRem(i.keyPrefix+":children", child.Path()))
}

func (i *Inode) getChildren() ([]*Inode, error) {
	if !i.IsDir() {
		return nil, ErrNotDirectory
	}
	client := i.redisRing.GetClient(i.keyPrefix)
	paths, err := client.SMembers(i.keyPrefix + ":children")
	Check(err)
	children := make([]*Inode, 0, len(paths)-1)
	for _, path := range paths {
		if path != "" {
			children = append(children, NewInode(i.dataStore, i.redisRing, path))
		}
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
