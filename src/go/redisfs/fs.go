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
	"fmt"
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
	inodes    *InodeRegister
}

// NewRedisFS a new RedisFS filesystem which entirely resides in memory
func NewRedisFS(redisConf *config.Redis, mountConf *config.Mount) *RedisFS {
	return &RedisFS{
		mountConf: mountConf,
		inodes:    NewInodeRegister(redisConf, mountConf),
	}
}

// Finalize performs close up actions on the virtual file system
func (fs *RedisFS) Finalize() error {
	return nil
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

// Mkdir creates a new directory with given permissions
func (fs *RedisFS) Mkdir(name string, perm os.FileMode) error {
	if err := fs.ValidatePath(name); err != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: err}
	}
	name = filepath.Clean(name)
	base := filepath.Base(name)
	parent, fi, err := fs.inodes.FileInfo(name)
	if err != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: err}
	}
	if fi != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: fmt.Errorf("Directory %q already exists", name)}
	}
	fs.inodes.CreateInode(base, true, perm, parent)
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
	path = filepath.Clean(path)
	_, fi, err := fs.inodes.FileInfo(path)
	if err != nil {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: err}
	}
	if fi == nil || !fi.IsDir() {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: ErrNotDirectory}
	}

	fis, err := fs.inodes.getChildren(fi)
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
	name = filepath.Clean(name)
	base := filepath.Base(name)
	fiParent, fiNode, err := fs.inodes.FileInfo(name)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: err}
	}

	if fiNode == nil {
		if !hasFlag(os.O_CREATE, flag) {
			return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
		}
		fiNode = fs.inodes.CreateInode(base, false, perm, fiParent)
	} else { // file exists
		if hasFlag(os.O_CREATE|os.O_EXCL, flag) {
			return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrExist}
		}
		if fiNode.IsDir() {
			return nil, &os.PathError{Op: "open", Path: name, Err: ErrIsDirectory}
		}
	}

	if !hasFlag(os.O_RDONLY, flag) {
		fiNode.touch()
	}
	return fs.inodes.getFile(fiNode, flag)
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
	name = filepath.Clean(name)
	fiParent, fiNode, err := fs.inodes.FileInfo(name)
	if err != nil {
		return &os.PathError{Op: "remove", Path: name, Err: err}
	}
	if fiNode == nil {
		return &os.PathError{Op: "remove", Path: name, Err: os.ErrNotExist}
	}
	fiParent.removeChild(fiNode)
	fs.inodes.delete(fiNode)
	return nil
}

// Rename renames (moves) a file.
// Handles to the oldpath persist but might return oldpath if Name() is called.
func (fs *RedisFS) Rename(oldpath, newpath string) error {
	if err := fs.ValidatePath(oldpath); err != nil {
		return &os.PathError{Op: "rename", Path: oldpath, Err: err}
	}
	oldpath = filepath.Clean(oldpath)
	fiOldParent, fiOld, err := fs.inodes.FileInfo(oldpath)
	if err != nil {
		return &os.PathError{Op: "rename", Path: oldpath, Err: err}
	}
	if fiOld == nil {
		return &os.PathError{Op: "rename", Path: oldpath, Err: os.ErrNotExist}
	}

	if err := fs.ValidatePath(newpath); err != nil {
		return &os.PathError{Op: "rename", Path: newpath, Err: err}
	}
	newpath = filepath.Clean(newpath)
	fiNewParent, fiNew, err := fs.inodes.FileInfo(newpath)
	if err != nil {
		return &os.PathError{Op: "rename", Path: newpath, Err: err}
	}

	if fiNew != nil {
		return &os.PathError{Op: "rename", Path: newpath, Err: os.ErrExist}
	}

	newBase := filepath.Base(newpath)

	// Relink
	fiOldParent.removeChild(fiOld)
	fiOld.relink(fiNewParent, newBase)
	fiNewParent.setChild(fiOld)
	return nil
}

// Stat returns the Inode structure describing the named file.
// If there is an error, it will be of type *PathError.
func (fs *RedisFS) Stat(name string) (os.FileInfo, error) {
	if err := fs.ValidatePath(name); err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}
	name = filepath.Clean(name)
	_, fi, err := fs.inodes.FileInfo(name)
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
