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
	"encoding/json"
	"os"
	filepath "path"
	"strings"
	"sync"
	"time"

	"github.com/cea-hpc/pdwfs/config"
	"github.com/go-redis/redis"
)

//InodeRegister ...
type InodeRegister struct {
	mountConf *config.Mount
	client    IRedisClient
	fiMap     map[string]*Inode
	root      *Inode
	cwd       *Inode
}

//Inode object
type Inode struct {
	mountConf *config.Mount
	client    IRedisClient
	id        string
	redisBuf  *RedisBlockedBuffer
	mtx       *sync.RWMutex
}

type inodeMeta struct {
	Name     string
	IsDir    bool
	Mode     os.FileMode
	ParentID string
	Size     int64
	ModTime  time.Time
	Children map[string]string
}

//NewInodeRegister constructor
func NewInodeRegister(redisConf *config.Redis, mountConf *config.Mount) *InodeRegister {
	client := NewRedisClient(redisConf)
	root := createInode(mountConf, client, mountConf.Path, true, 0600, nil, true)
	return &InodeRegister{
		mountConf: mountConf,
		client:    client,
		fiMap:     map[string]*Inode{mountConf.Path: root},
		root:      root,
		cwd:       root,
	}
}

//Finalize ...
func (ir *InodeRegister) Finalize() error {
	return ir.client.Close()
}

func createInode(mountConf *config.Mount, client IRedisClient, name string, dir bool, mode os.FileMode, parent *Inode, root bool) *Inode {

	var parentID, id string
	var err error
	if root {
		id = name
		parentID = ""
	} else {
		id, err = randomToken() //FIXME: add an id collision check ?
		if err != nil {
			panic(err)
		}
		parentID = parent.ID()
	}

	inode := NewInode(mountConf, client, id)

	inode.initMeta(&inodeMeta{
		Name:     name,
		IsDir:    dir,
		Mode:     mode,
		ParentID: parentID,
		ModTime:  time.Now(),
	})

	return inode
}

//CreateInode ...
func (ir *InodeRegister) CreateInode(name string, dir bool, mode os.FileMode, parent *Inode) *Inode {
	i := createInode(ir.mountConf, ir.client, name, dir, mode, parent, false)
	parent.setChild(i)
	ir.fiMap[i.ID()] = i
	return i
}

func (ir *InodeRegister) getInode(id string) (*Inode, bool) {
	if i, ok := ir.fiMap[id]; ok {
		return i, true
	}
	i := NewInode(ir.mountConf, ir.client, id)

	if ok, _ := i.hasMeta(); !ok {
		return nil, false
	}
	ir.fiMap[id] = i
	return i, true
}

func (ir *InodeRegister) getParent(inode *Inode) *Inode {
	parentID := inode.ParentID()
	if parentID == "" {
		return nil
	}
	parent, _ := ir.getInode(parentID)
	return parent
}

func (ir *InodeRegister) getChildByName(parent *Inode, base string) (*Inode, bool) {
	childID, ok := parent.getChildID(base)
	if !ok {
		return nil, false
	}
	if child, ok := ir.getInode(childID); ok {
		return child, true
	}
	return nil, false
}

//FileInfo ...
func (ir *InodeRegister) FileInfo(path string) (parent *Inode, node *Inode, err error) {

	// remove root path from path
	path = strings.Replace(path, ir.root.Name(), "", 1)

	segments := SplitPath(path, PathSeparator)

	// Shortcut for working directory and root
	if len(segments) == 1 {
		if segments[0] == "" {
			return nil, ir.root, nil
		} else if segments[0] == "." {
			return ir.getParent(ir.cwd), ir.cwd, nil
		}
	}

	// Determine root to traverse
	parent = ir.root
	if segments[0] == "." {
		parent = ir.cwd
	}
	segments = segments[1:]

	// Further directories
	if len(segments) > 1 {
		for _, seg := range segments[:len(segments)-1] {

			if !parent.hasChildren() {
				return nil, nil, os.ErrNotExist
			}
			if entry, ok := ir.getChildByName(parent, seg); ok && entry.IsDir() {
				parent = entry
			} else {
				return nil, nil, os.ErrNotExist
			}
		}
	}

	lastSeg := segments[len(segments)-1]
	if parent.hasChildren() {
		if node, ok := ir.getChildByName(parent, lastSeg); ok {
			return parent, node, nil
		}
	}
	return parent, nil, nil
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

//AbsPath returns the inode absolute path
func (ir *InodeRegister) AbsPath(inode *Inode) string {
	parent := ir.getParent(inode)
	base := inode.Name()
	if parent == nil {
		// root inode
		return base
	}
	return filepath.Join(ir.AbsPath(parent), base)
}

func (ir *InodeRegister) getFile(inode *Inode, flag int) (File, error) {
	if inode.IsDir() {
		return nil, ErrIsDirectory
	}
	inodeBuf := inode.getBuffer()

	if hasFlag(os.O_TRUNC, flag) {
		inodeBuf.Clear()
	}

	var f File = NewMemFile(inodeBuf, ir.AbsPath(inode), inode.mtx)

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
	inode.delMeta()
}

//NewInode ...
func NewInode(mountConf *config.Mount, client IRedisClient, id string) *Inode {
	return &Inode{
		mountConf: mountConf,
		client:    client,
		id:        id,
		mtx:       &sync.RWMutex{},
	}
}

func (i *Inode) hasMeta() (bool, error) {
	ret, err := i.client.Exists(i.id).Result()
	return ret != 0, err
}

func (i *Inode) initMeta(md *inodeMeta) error {
	jsoned, err := json.Marshal(md)
	if err != nil {
		return err
	}
	return i.client.SetNX(i.id, string(jsoned), 0).Err()
}

//FIXME: a lock is required when modifying metadata since all metadata are retrieved,
// modified and then resend, it may be best to use a hash for metadata and only update
// in an atomic way the necessary hash key
// Besides, this is a very basic lock implementation...
func (i *Inode) lockMeta() {
	for {
		if ok := i.client.SetNX(i.id+":metalock", "locked", 0).Val(); ok {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (i *Inode) unlockMeta() {
	i.client.Del(i.id + ":metalock")
}

func (i *Inode) setMeta(md *inodeMeta) error {
	jsoned, err := json.Marshal(md)
	if err != nil {
		return err
	}
	return i.client.Set(i.id, string(jsoned), 0).Err()
}

func (i *Inode) getMeta() *inodeMeta {
	//TODO: implement Metadata caching with hash checking
	var md inodeMeta
	jsoned, err := i.client.Get(i.id).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(jsoned), &md)
	if err != nil {
		panic(err)
	}
	return &md
}

func (i *Inode) getBuffer() *RedisBlockedBuffer {
	if i.redisBuf == nil && !i.IsDir() {
		i.redisBuf = NewRedisBlockedBuffer(i.mountConf, i.client, i.ID())
	}
	return i.redisBuf
}

func (i *Inode) delMeta() error {
	return i.client.Del(i.id).Err()
}

//ID returns the ID of the file
func (i *Inode) ID() string {
	return i.id
}

//Sys no op (to fulfill os.FileMode interface)
func (i *Inode) Sys() interface{} {
	return nil
}

//IsDir ...
func (i *Inode) IsDir() bool {
	return i.getMeta().IsDir
}

//Size returns the size of the file
func (i *Inode) Size() int64 {
	if i.IsDir() {
		return 0
	}
	return i.getBuffer().Size()
}

// ModTime returns the modification time.
// Modification time is updated on:
// 	- Creation
// 	- Rename
// 	- Open (except with O_RDONLY)
func (i *Inode) ModTime() time.Time {
	return i.getMeta().ModTime
}

//Mode returns the inode access mode
func (i *Inode) Mode() os.FileMode {
	return i.getMeta().Mode
}

//Name returns the inode base name
func (i *Inode) Name() string {
	return i.getMeta().Name
}

//ParentID returns the inode parent ID
func (i *Inode) ParentID() string {
	return i.getMeta().ParentID
}

func (i *Inode) hasChildren() bool {
	return len(i.getMeta().Children) != 0
}

func (i *Inode) getChildID(base string) (string, bool) {
	if childID, ok := i.getMeta().Children[base]; ok {
		return childID, ok
	}
	return "", false
}

func (i *Inode) childrenID() []string {
	md := i.getMeta()
	children := make([]string, 0, len(md.Children))
	for _, child := range md.Children {
		children = append(children, child)
	}
	return children
}

func (i *Inode) setChild(child *Inode) error {
	i.lockMeta()
	defer i.unlockMeta()
	md := i.getMeta()
	if md.Children == nil {
		md.Children = make(map[string]string)
	}
	md.Children[child.Name()] = child.ID()
	return i.setMeta(md)
}

func (i *Inode) removeChild(child *Inode) error {
	i.lockMeta()
	defer i.unlockMeta()
	md := i.getMeta()
	delete(md.Children, child.Name())
	return i.setMeta(md)
}

func (i *Inode) relink(newParent *Inode, newBase string) error {
	i.lockMeta()
	defer i.unlockMeta()
	md := i.getMeta()
	md.ParentID = newParent.ID()
	md.ModTime = time.Now()
	md.Name = newBase
	return i.setMeta(md)
}

// update modification time
func (i *Inode) touch() {
	i.lockMeta()
	defer i.unlockMeta()
	md := i.getMeta()
	md.ModTime = time.Now()
	i.setMeta(md)
}
