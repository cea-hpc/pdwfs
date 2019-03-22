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

	"github.com/cea-hpc/pdwfs/config"
)

//InodeRegister ...
type InodeRegister struct {
	mountConf *config.Mount
	client    IRedisClient
	fiMap     map[string]*Inode
	root      *Inode
}

//Inode object
type Inode struct {
	mountConf   *config.Mount
	client      IRedisClient
	id          string
	redisBuf    *RedisBlockedBuffer
	mtx         *sync.RWMutex
	metaBaseKey string
	isDir       *bool
	mode        *os.FileMode
}

//NewInodeRegister constructor
func NewInodeRegister(redisConf *config.Redis, mountConf *config.Mount) *InodeRegister {
	client := NewRedisClient(redisConf)

	// create root inode
	root := NewInode(mountConf, client, mountConf.Path)
	root.initMeta(true, 0600)

	return &InodeRegister{
		mountConf: mountConf,
		client:    client,
		fiMap:     map[string]*Inode{mountConf.Path: root},
		root:      root,
	}
}

//Finalize ...
func (ir *InodeRegister) Finalize() error {
	return ir.client.Close()
}

//CreateInode ...
func (ir *InodeRegister) CreateInode(path string, dir bool, mode os.FileMode, parent *Inode) *Inode {

	inode := NewInode(ir.mountConf, ir.client, path)
	inode.initMeta(dir, mode)

	parent.setChild(inode)
	ir.fiMap[inode.ID()] = inode
	return inode
}

func (ir *InodeRegister) getInode(id string) (*Inode, bool) {
	if i, ok := ir.fiMap[id]; ok {
		return i, true
	}
	i := NewInode(ir.mountConf, ir.client, id)

	if ok, _ := i.exists(); !ok {
		return nil, false
	}
	ir.fiMap[id] = i
	return i, true
}

func (ir *InodeRegister) getChildren(inode *Inode) ([]*Inode, error) {
	IDs := inode.childrenID()
	children := make([]*Inode, 0, len(IDs))
	for _, id := range IDs {
		child, ok := ir.getInode(id)
		if !ok {
			panic("Inode not found")
		}
		children = append(children, child)
	}
	return children, nil
}

func (ir *InodeRegister) getFile(inode *Inode, flag int) (File, error) {
	if inode.IsDir() {
		return nil, ErrIsDirectory
	}
	inodeBuf := inode.getBuffer()

	if hasFlag(os.O_TRUNC, flag) {
		inodeBuf.Clear()
	}

	var f File = NewMemFile(inodeBuf, inode.Name(), inode.mtx)

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

func (ir *InodeRegister) delete(inode *Inode) {
	if buf := inode.getBuffer(); buf != nil {
		buf.Clear()
	}
	children, _ := ir.getChildren(inode)
	for _, child := range children {
		ir.delete(child)
	}
	delete(ir.fiMap, inode.Name())
	err := inode.delMeta()
	if err != nil {
		panic(err)
	}
}

//NewInode ...
func NewInode(mountConf *config.Mount, client IRedisClient, id string) *Inode {
	return &Inode{
		mountConf:   mountConf,
		client:      client,
		id:          id,
		mtx:         &sync.RWMutex{},
		metaBaseKey: "{" + id + "}", // hastag to ensure all metadata keys goes on the same instance
	}
}
func (i *Inode) getBuffer() *RedisBlockedBuffer {
	if i.redisBuf == nil && !i.IsDir() {
		i.redisBuf = NewRedisBlockedBuffer(i.mountConf, i.client, i.ID())
	}
	return i.redisBuf
}
func (i *Inode) exists() (bool, error) {
	if i.mode != nil {
		return true, nil
	}
	ret, err := i.client.Exists(i.metaBaseKey + ":mode").Result()
	return ret != 0, err
}

func (i *Inode) initMeta(isDir bool, mode os.FileMode) error {
	pipeline := i.client.Pipeline()
	pipeline.SetNX(i.metaBaseKey+":isDir", isDir, 0)
	pipeline.SetNX(i.metaBaseKey+":mode", uint32(mode), 0)
	_, err := pipeline.Exec()
	return err
}

func (i *Inode) delMeta() error {
	pipeline := i.client.Pipeline()
	pipeline.Del(i.metaBaseKey + ":children")
	pipeline.Del(i.metaBaseKey + ":isDir")
	pipeline.Del(i.metaBaseKey + ":mode")
	_, err := pipeline.Exec()
	return err
}

//IsDir ...
func (i *Inode) IsDir() bool {
	if i.isDir == nil {
		res := i.client.Get(i.metaBaseKey+":isDir").Val() == "1"
		i.isDir = &res
	}
	return (*i.isDir)
}

//Mode returns the inode access mode
func (i *Inode) Mode() os.FileMode {
	if i.mode == nil {
		val := i.client.Get(i.metaBaseKey + ":mode").Val()
		res, err := strconv.Atoi(val)
		if err != nil {
			panic(err)
		}
		m := os.FileMode(res)
		i.mode = &m
	}
	return (*i.mode)
}

//ID returns the ID of the file
func (i *Inode) ID() string {
	return i.id
}

//Name returns the inode base name
func (i *Inode) Name() string {
	return i.ID()
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
	return i.getBuffer().Size()
}

func (i *Inode) childrenID() []string {
	return i.client.SMembers(i.metaBaseKey + ":children").Val()
}

func (i *Inode) setChild(child *Inode) error {
	return i.client.SAdd(i.metaBaseKey+":children", child.ID()).Err()
}

func (i *Inode) removeChild(child *Inode) error {
	return i.client.SRem(i.metaBaseKey+":children", child.ID()).Err()
}
