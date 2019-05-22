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
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cea-hpc/pdwfs/config"
)

var (
	// ErrFileNotManaged is returned if the file doesn't belong to a filesystem tree intercepted by pdwfs.
	ErrFileNotManaged = errors.New("File is not managed by pdwfs")
	// ErrReadOnly is returned if the file is read-only and write operations are disabled.
	ErrReadOnly = errors.New("File is read-only")
	// ErrWriteOnly is returned if the file is write-only and read operations are disabled.
	ErrWriteOnly = errors.New("File is write-only")
	// ErrIsDirectory is returned if the file under operation is not a regular file but a directory.
	ErrIsDirectory = errors.New("Is directory")
	// ErrNotDirectory is returned if a file is not a directory
	ErrNotDirectory = errors.New("Is not a directory")
	// ErrDirNotEmpty is returned if a directory is not empty (rmdir)
	ErrDirNotEmpty = errors.New("Directory is not empty")
	// ErrParentDirNotExist is returned if the parent directory does not exist
	ErrParentDirNotExist = errors.New("Parent directory does not exist")
)

// File represents a File with common operations.
// It differs from os.File so e.g. Stat() needs to be called from the Filesystem instead.
//   osfile.Stat() -> filesystem.Stat(file.Name())
type File interface {
	Name() string
	Sync() error
	// Truncate shrinks or extends the size of the File to the specified size.
	Truncate(int64) error
	io.Reader
	io.ReaderAt
	io.Writer
	io.WriterAt
	io.Seeker
	io.Closer
	WriteVec([][]byte) (int, error)
	ReadVec([][]byte) (int, error)
	WriteVecAt([][]byte, int64) (int, error)
	ReadVecAt([][]byte, int64) (int, error)
}

// PathSeparator used to separate path segments
const PathSeparator = "/"

// RedisFS is a in-memory filesystem
type RedisFS struct {
	mountConf *config.Mount
	dataStore *DataStore
	redisRing *RedisRing
	inodes    map[string]*Inode
	root      *Inode
}

// NewRedisFS a new RedisFS filesystem which entirely resides in memory
func NewRedisFS(redisConf *config.Redis, mountConf *config.Mount) *RedisFS {
	redisRing := NewRedisRing(redisConf)
	dataStore := NewDataStore(redisRing, int64(mountConf.StripeSize))

	// create root inode
	//FIXME: mount path (root) should only be created it it exists on the FS at startup
	root := NewInode(dataStore, redisRing, mountConf.Path)
	root.initMeta(true, 0600)

	return &RedisFS{
		mountConf: mountConf,
		redisRing: redisRing,
		dataStore: dataStore,
		inodes:    map[string]*Inode{root.Path(): root},
		root:      root,
	}
}

// Finalize performs close up actions on the virtual file system
func (fs *RedisFS) Finalize() {
	fs.redisRing.Close()
	fs.dataStore.Close()
}

// ValidatePath ensures path belongs to a filesystem tree catched by pdwfs
func (fs *RedisFS) ValidatePath(path string) error {
	p, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(p, fs.mountConf.Path) {
		return ErrFileNotManaged
	}
	return nil
}

func (fs *RedisFS) createInode(path string, dir bool, mode os.FileMode, parent *Inode) *Inode {
	i := NewInode(fs.dataStore, fs.redisRing, path)
	i.initMeta(dir, mode)
	parent.setChild(i)
	fs.inodes[i.Path()] = i
	return i
}

func (fs *RedisFS) getInode(path string) (*Inode, bool) {
	if i, ok := fs.inodes[path]; ok {
		return i, true
	}
	i := NewInode(fs.dataStore, fs.redisRing, path)
	if ok := i.exists(); !ok {
		return nil, false
	}
	fs.inodes[i.Path()] = i
	return i, true
}

func (fs *RedisFS) removeInode(i *Inode) {
	i.remove()
	delete(fs.inodes, i.Path())
}

func (fs *RedisFS) fileInfo(abspath string) (parent, node *Inode, err error) {
	if abspath == fs.root.Path() {
		return nil, fs.root, nil
	}
	parentPath := filepath.Dir(abspath)
	fiParent, _ := fs.getInode(parentPath)
	if fiParent == nil || !fiParent.IsDir() {
		return nil, nil, ErrParentDirNotExist
	}
	fiNode, _ := fs.getInode(abspath)
	return fiParent, fiNode, nil
}

// Mkdir creates a new directory with given permissions
func (fs *RedisFS) Mkdir(name string, perm os.FileMode) error {
	if err := fs.ValidatePath(name); err != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: err}
	}
	path, err := filepath.Abs(name)
	Check(err)
	fiParent, fiNode, err := fs.fileInfo(path)
	if err != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: err}
	}
	if fiNode == fs.root {
		//FIXME: hack to cover the case the app creates the mount path directory,
		// because we create the root dir at initialization of RedisFS, it should fail with ErrExist.
		// Proper way to do this is to create the root dir in RedisFS only if it already exists on FS at startup
		return nil
	}
	if fiNode != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: os.ErrExist}
	}
	fs.createInode(path, true, perm, fiParent)
	return nil
}

// byName implements sort.Interface
type byName []os.FileInfo

// Len returns the length of the slice
func (f byName) Len() int { return len(f) }

// Less sorts slice by Name
func (f byName) Less(i, j int) bool { return f[i].Name() < f[j].Name() }

// Swap two elements by index
func (f byName) Swap(i, j int) { f[i], f[j] = f[j], f[i] }

// ReadDir reads the directory named by path and returns a list of sorted directory entries.
func (fs *RedisFS) ReadDir(path string) ([]os.FileInfo, error) {
	if err := fs.ValidatePath(path); err != nil {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: err}
	}
	path, err := filepath.Abs(path)
	Check(err)
	_, fi, err := fs.fileInfo(path)
	if err != nil {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: err}
	}
	if fi == nil || !fi.IsDir() {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: ErrNotDirectory}
	}

	fis, err := fi.getChildren()
	if err != nil {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: err}
	}
	f := make([]os.FileInfo, len(fis))
	for i := 0; i < len(fis); i++ {
		f[i] = fis[i]
	}
	sort.Sort(byName(f))
	return f, nil
}

//RmDir remove a directory if it has no entry
func (fs *RedisFS) RmDir(path string) error {
	if err := fs.ValidatePath(path); err != nil {
		return &os.PathError{Op: "rmdir", Path: path, Err: err}
	}
	path = filepath.Clean(path)
	entries, err := fs.ReadDir(path)
	if err != nil {
		return &os.PathError{Op: "rmdir", Path: path, Err: err}
	}
	if len(entries) != 0 {
		return &os.PathError{Op: "rmdir", Path: path, Err: ErrDirNotEmpty}
	}
	return fs.Remove(path)
}

func hasFlag(flag int, flags int) bool {
	return flags&flag == flag
}

// OpenFile opens a file handle with a specified flag (os.O_RDONLY etc.) and perm (e.g. 0666).
// If success the returned File can be used for I/O. Otherwise an error is returned, which
// is a *os.PathError and can be extracted for further information.
func (fs *RedisFS) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	if err := fs.ValidatePath(name); err != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: err}
	}
	path, err := filepath.Abs(name)
	Check(err)
	fiParent, fiNode, err := fs.fileInfo(path)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: err}
	}

	if fiNode == nil {
		if !hasFlag(os.O_CREATE, flag) {
			return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
		}
		fiNode = fs.createInode(path, false, perm, fiParent)
	} else { // file exists
		if hasFlag(os.O_CREATE|os.O_EXCL, flag) {
			return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrExist}
		}
		if fiNode.IsDir() {
			return nil, &os.PathError{Op: "open", Path: name, Err: ErrIsDirectory}
		}
	}
	return fiNode.getFile(flag)
}

// roFile wraps the given file and disables Write(..) operation.
type roFile struct {
	File
}

// Write is disabled and returns ErrorReadOnly
func (f *roFile) Write(p []byte) (n int, err error) {
	return 0, ErrReadOnly
}

// woFile wraps the given file and disables Read(..) operation.
type woFile struct {
	File
}

// Read is disabled and returns ErrorWroteOnly
func (f *woFile) Read(p []byte) (n int, err error) {
	return 0, ErrWriteOnly
}

// Remove removes the named file or directory.
// If there is an error, it will be of type *PathError.
func (fs *RedisFS) Remove(name string) error {
	if err := fs.ValidatePath(name); err != nil {
		return &os.PathError{Op: "remove", Path: name, Err: err}
	}
	path, err := filepath.Abs(name)
	Check(err)
	fiParent, fiNode, err := fs.fileInfo(path)
	if err != nil {
		return &os.PathError{Op: "remove", Path: name, Err: err}
	}
	if fiNode == nil {
		return &os.PathError{Op: "remove", Path: name, Err: os.ErrNotExist}
	}
	fiParent.removeChild(fiNode)
	fs.removeInode(fiNode)
	return nil
}

// Stat returns the Inode structure describing the named file.
// If there is an error, it will be of type *PathError.
func (fs *RedisFS) Stat(name string) (os.FileInfo, error) {
	if err := fs.ValidatePath(name); err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}
	path, err := filepath.Abs(name)
	Check(err)
	_, fi, err := fs.fileInfo(path)
	if err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}
	if fi == nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
	}
	return fi, nil
}

// Lstat returns a Inode describing the named file.
// RedisFS does not support symbolic links.
// Alias for fs.Stat(name)
func (fs *RedisFS) Lstat(name string) (os.FileInfo, error) {
	return fs.Stat(name)
}
