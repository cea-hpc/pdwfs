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

/*
#cgo CFLAGS: -D_LARGEFILE64_SOURCE
#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/statfs.h>
#include <sys/statvfs.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/cea-hpc/pdwfs/config"
	"github.com/cea-hpc/pdwfs/redisfs"
)

// ---------- Unexported ---------------

var (
	errFdPoolTempCreate = errors.New("Failed to create a tempfile to get a valid fd")
	errInvalidFd        = errors.New("Invalid file descriptor")
)

func logError(data ...interface{}) {
	log.Println("\033[31mERROR", fmt.Sprint(data...), "\033[39m")
}

type fileTuple struct {
	tempFileName string
	cFile        *C.FILE
	redisFile    *redisfs.File
}

//PdwFS is a virtual filesystem object built on top of github.com/cea-hpc/pdwfs/redisfs
type PdwFS struct {
	mounts    map[string]*redisfs.RedisFS
	conf      *config.Pdwfs
	prefix    string
	fdFileMap map[int]*fileTuple
	lock      sync.RWMutex
}

//NewPdwFS returns a pdwfs virtual filesystem
func NewPdwFS(conf *config.Pdwfs) *PdwFS {
	defer func() {
		if err := recover(); err != nil {
			logError(err)
			os.Exit(1)
		}
	}()
	if len(conf.Mounts) == 0 {
		panic("No mount path specified...")
	}
	mounts := map[string]*redisfs.RedisFS{}
	for path, mountConf := range conf.Mounts {
		mounts[path] = redisfs.NewRedisFS(conf.RedisConf, mountConf)
	}
	return &PdwFS{
		mounts:    mounts,
		conf:      conf,
		fdFileMap: make(map[int]*fileTuple),
		lock:      sync.RWMutex{},
	}
}

func (fs *PdwFS) getMount(filename string) (*redisfs.RedisFS, error) {
	if filename == "" {
		//short-circuit filepath.Abs as Abs behaviour is to return working directory on empty string
		// this is not the behaviour we want
		return nil, nil
	}
	p, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	for path, mount := range fs.mounts {
		if strings.HasPrefix(p, path) {
			return mount, nil
		}
	}
	return nil, nil
}

func (fs *PdwFS) registerFile(redisFile *redisfs.File) (*C.FILE, error) {
	// create a temporary file to get a valid system file descritor
	// NOTE: TempFile uses os.OpenFile which uses openat syscall which is currently not intercepted
	// so if a user set a mount point in the same temp folder used by TempFile, we're not
	// entering in a recursive loop (intercepting a file in temp, creating a twin temp file, etc)
	tempFile, err := ioutil.TempFile("", "pdwfs")
	if err != nil {
		panic(err)
	}
	tempFileName := tempFile.Name()
	tempFile.Close()
	path := C.CString(tempFileName)
	mode := C.CString("r")
	cFile, err := C.fopen(path, mode)
	C.free(unsafe.Pointer(path))
	C.free(unsafe.Pointer(mode))
	if err != nil {
		panic(err)
	}
	fd := int(C.fileno(cFile))
	fs.fdFileMap[fd] = &fileTuple{tempFileName, cFile, redisFile}
	return cFile, nil
}

func (fs *PdwFS) closeFd(fd int) error {
	//FIXME: not thread-safe if multiple thread handle the same fd...
	if _, ok := fs.fdFileMap[fd]; !ok {
		return errInvalidFd
	}
	delete(fs.fdFileMap, fd)
	// deleting fd from the managed fd map first ensures the subsequent call to close(fd),
	// (which will be intercepted by pdwfs), will be passed to the real libc close call
	C.close(C.int(fd))
	return nil
}

func (fs *PdwFS) getFileFromFd(fd int) (*redisfs.File, error) {
	if fileTuple, ok := fs.fdFileMap[fd]; ok {
		return fileTuple.redisFile, nil
	}
	return nil, errInvalidFd
}

func (fs *PdwFS) isFdManaged(fd int) bool {
	if _, ok := fs.fdFileMap[fd]; ok {
		return true
	}
	return false
}

func (fs *PdwFS) finalize() {
	for _, mount := range fs.mounts {
		err := mount.Finalize()
		if err != nil {
			panic(err)
		}
	}
	// clean up all temp files created
	for _, fileTuple := range fs.fdFileMap {
		err := os.Remove(fileTuple.tempFileName)
		if err != nil {
			panic(err)
		}
	}
}

// ----------------Exported to C ----------------

var pdwfs *PdwFS

// InitPdwfs is called once when pdwfs.so library is loaded (gcc constructor attribute)
//export InitPdwfs
func InitPdwfs() {
	conf := config.New()
	if dump := os.Getenv("PDWFS_DUMPCONF"); dump != "" {
		if err := conf.Dump(); err != nil {
			logError(err)
			os.Exit(1)
		}
	}
	pdwfs = NewPdwFS(conf)
}

// FinalizePdwfs is called once when pdwfs.so library is unloaded (gcc destructor attribute)
//export FinalizePdwfs
func FinalizePdwfs() {
	pdwfs.finalize()
}

//IsFileManaged returns 1 if the directory in argument is managed by pdwfs from config
//export IsFileManaged
func IsFileManaged(filename string) int {
	mount, err := pdwfs.getMount(filename)
	if err != nil {
		panic(err)
	}
	if mount != nil {
		return 1
	}
	return 0
}

//IsFdManaged checks whether a file descriptor is managed
//export IsFdManaged
func IsFdManaged(fd int) int {
	if pdwfs.isFdManaged(fd) {
		return 1
	}
	return 0
}

//Open implements open libc call
//export Open
func Open(filename string, flags int, mode int) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	mount, err := pdwfs.getMount(filename)
	if err != nil {
		logError(err)
		return -1
	}
	file, err := mount.OpenFile(filename, flags, os.FileMode(mode))
	if err != nil {
		logError(err)
		return -1
	}
	cFile, err := pdwfs.registerFile(&file)
	if err != nil {
		logError(err)
		return -1
	}
	return int(C.fileno(cFile))
}

//Fopen implements fopen libc call
//export Fopen
func Fopen(filename string, mode string) *C.FILE {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	mount, err := pdwfs.getMount(filename)
	if err != nil {
		logError(err)
		return (*C.FILE)(C.NULL)
	}
	var flags int
	switch mode {
	case "r":
		flags = os.O_RDONLY
	case "w":
		flags = os.O_RDWR | os.O_CREATE
	default:
		panic(fmt.Sprintf("fopen mode '%s' unknown or not implemented yet", mode))
	}
	file, err := mount.OpenFile(filename, flags, os.FileMode(0600))
	if err != nil {
		logError(err)
		return (*C.FILE)(C.NULL)
	}
	cFile, err := pdwfs.registerFile(&file)
	if err != nil {
		logError(err)
		return (*C.FILE)(C.NULL)
	}
	return cFile
}

//Close implements close libc call
//export Close
func Close(fd int) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	err = (*file).Close()
	if err != nil {
		logError(err)
		return -1
	}
	err = pdwfs.closeFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	return 0
}

//Write implements write libc call
//export Write
func Write(fd int, buf []byte) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	n, err := (*file).Write(buf)
	if err != nil {
		logError(err)
		return -1
	}
	return n
}

//Pwrite implements pwrite libc call
//export Pwrite
func Pwrite(fd int, buf []byte, off int64) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	n, err := (*file).WriteAt(buf, off)
	if err != nil {
		logError(err)
		return -1
	}
	return n
}

//Writev implements writev libc call
//export Writev
func Writev(fd int, iov [][]byte) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	n, err := (*file).WriteVec(iov)
	if err != nil {
		logError(err)
		return -1
	}
	return n
}

//Pwritev implements pwritev libc call
//export Pwritev
func Pwritev(fd int, iov [][]byte, off int64) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	n, err := (*file).WriteVecAt(iov, off)
	if err != nil {
		logError(err)
		return -1
	}
	return n
}

//Read implements read libc call
//export Read
func Read(fd int, buf []byte) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	n, err := (*file).Read(buf)
	if err != nil && err != io.EOF {
		logError(err)
		return -1
	}
	return n
}

//Pread implements pread libc call
//export Pread
func Pread(fd int, buf []byte, off int64) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	n, err := (*file).ReadAt(buf, off)
	if err != nil && err != io.EOF {
		logError(err)
		return -1
	}
	return n
}

//Readv implements readv libc call
//export Readv
func Readv(fd int, iov [][]byte) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	n, err := (*file).ReadVec(iov)
	if err != nil && err != io.EOF {
		logError(err)
		return -1
	}
	return n
}

//Preadv implements preadv libc call
//export Preadv
func Preadv(fd int, iov [][]byte, off int64) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	n, err := (*file).ReadVecAt(iov, off)
	if err != nil && err != io.EOF {
		logError(err)
		return -1
	}
	return n
}

//Lseek implements lseek libc call
//export Lseek
func Lseek(fd int, offset int64, whence int) int64 {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	n, err := (*file).Seek(offset, whence)
	if err != nil {
		logError(err)
		return -1
	}
	return n
}

//Unlink implements unlink libc call
//export Unlink
func Unlink(filename string) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	mount, err := pdwfs.getMount(filename)
	if err != nil {
		logError(err)
		return -1
	}
	err = mount.Remove(filename)
	if err != nil {
		logError(err)
		return -1
	}
	return 0
}

//Mkdir implements mkdir libc call
//export Mkdir
func Mkdir(dirname string, mode int) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	mount, err := pdwfs.getMount(dirname)
	if err != nil {
		logError(err)
		return -1
	}
	err = mount.Mkdir(dirname, os.FileMode(mode))
	if err != nil {
		logError(err)
		return -1
	}
	return 0
}

//Rmdir implements rmdir libc call
//export Rmdir
func Rmdir(dirname string) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	mount, err := pdwfs.getMount(dirname)
	if err != nil {
		logError(err)
		return -1
	}
	err = mount.RmDir(dirname)
	if err != nil {
		logError(err)
		return -1
	}
	return 0
}

//Access implements access libc call
//export Access
func Access(filename string, mode int) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	mount, err := pdwfs.getMount(filename)
	if err != nil {
		logError(err)
		return -1
	}
	_, err = mount.Stat(filename)
	if err != nil {
		//FIXME: don't log any error if mode is F_OK and error returned is ErrNotExist, this is normal behaviour
		logError(err)
		return -1
	}
	//FIXME: check versus the mode (R_OK, W_OK, X_OK)
	return 0
}

// Ftruncate implements ftruncate libc call
//export Ftruncate
func Ftruncate(fd int, length int64) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	err = (*file).Truncate(length)
	if err != nil {
		logError(err)
		return -1
	}
	return 0
}

func stat(filename string, stats *C.struct_stat) int {
	mount, err := pdwfs.getMount(filename)
	if err != nil {
		logError(err)
		return -1
	}
	inode, err := mount.Stat(filename)
	if err != nil {
		logError(err)
		return -1
	}
	// Only implements value required by test applications
	if inode.IsDir() {
		stats.st_mode = C.__S_IFDIR
	} else {
		stats.st_mode = C.__S_IFREG
	}
	stats.st_size = C.long(inode.Size()) // total file size in bytes
	return 0
}

//Stat implements part of __xstat libc call
//export Stat
func Stat(filename string, stats *C.struct_stat) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	return stat(filename, stats)
}

func stat64(filename string, stats *C.struct_stat64) int {
	mount, err := pdwfs.getMount(filename)
	if err != nil {
		logError(err)
		return -1
	}
	inode, err := mount.Stat(filename)
	if err != nil {
		logError(err)
		return -1
	}
	// Only implements value required by test applications
	if inode.IsDir() {
		stats.st_mode = C.__S_IFDIR
	} else {
		stats.st_mode = C.__S_IFREG
	}
	stats.st_size = C.long(inode.Size()) // total file size in bytes
	return 0
}

//Stat64 implements part of __stat64 libc call
//export Stat64
func Stat64(filename string, stats *C.struct_stat64) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	return stat64(filename, stats)
}

//Fstat implements part of __fxstat libc call, cf. Stat
//export Fstat
func Fstat(fd int, stats *C.struct_stat) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	return stat((*file).Name(), stats)
}

//Fstat64 implements part of __fxstat64 libc call, cf. Stat
//export Fstat64
func Fstat64(fd int, stats *C.struct_stat64) int {
	pdwfs.lock.Lock()
	defer pdwfs.lock.Unlock()
	file, err := pdwfs.getFileFromFd(fd)
	if err != nil {
		logError(err)
		return -1
	}
	return stat64((*file).Name(), stats)
}

//Lstat implements part of __lxstat libc call (symlink are not supported so it's an alias to Stat)
//export Lstat
func Lstat(filename string, stats *C.struct_stat) int {
	return Stat(filename, stats)
}

//Lstat64 implements part of __lxstat64 libc call (symlink are not supported so it's an alias to Stat)
//export Lstat64
func Lstat64(filename string, stats *C.struct_stat64) int {
	return Stat64(filename, stats)
}

func statfs() syscall.Statfs_t {
	return syscall.Statfs_t{
		Type:   0x0BD00BD0, // we fake a Lustre file system, see lustre_user.h (ext2 is 0xEF53, see man statfs)
		Bsize:  1,          // block size
		Blocks: 1,          // number of blocks
		Bfree:  1,          // total free blocks
		Bavail: 1,          // free blocks available to user (unpriviledged)
		Files:  1,          // total file nodes in fs
		Ffree:  1,          // free file nodes in fs
	}
}

//Statfs implements part of statfs libc call
//export Statfs
func Statfs(filename string, fsstats *C.struct_statfs) int {
	//FIXME: this information should be returned by the redisfs instance managing 'filename'
	s := statfs()
	fsstats.f_type = C.long(s.Type)      // fs type
	fsstats.f_bsize = C.long(s.Bsize)    // block size
	fsstats.f_blocks = C.ulong(s.Blocks) // number of blocks
	fsstats.f_bfree = C.ulong(s.Bfree)   // total free blocks
	fsstats.f_bavail = C.ulong(s.Bavail) // free blocks available to user (unpriviledged)
	fsstats.f_files = C.ulong(s.Files)   // total file nodes in fs
	fsstats.f_ffree = C.ulong(s.Ffree)   // free file nodes in fs
	return 0
}

//Statfs64 implements part of statfs64 libc call
//export Statfs64
func Statfs64(filename string, fsstats *C.struct_statfs64) int {
	//FIXME: this information should be returned by the redisfs instance managing 'filename'
	s := statfs()
	fsstats.f_type = C.long(s.Type)      // fs type
	fsstats.f_bsize = C.long(s.Bsize)    // block size
	fsstats.f_blocks = C.ulong(s.Blocks) // number of blocks
	fsstats.f_bfree = C.ulong(s.Bfree)   // total free blocks
	fsstats.f_bavail = C.ulong(s.Bavail) // free blocks available to user (unpriviledged)
	fsstats.f_files = C.ulong(s.Files)   // total file nodes in fs
	fsstats.f_ffree = C.ulong(s.Ffree)   // free file nodes in fs
	return 0
}

//Statvfs implements part of statvfs libc call
//export Statvfs
func Statvfs(filename string, vfsstats *C.struct_statvfs) int {
	//FIXME: this information should be returned by the redisfs instance managing 'filename'
	s := statfs()
	vfsstats.f_bsize = C.ulong(s.Bsize) // block size
	//NOTE: statvfs is used by openmpi to get the fs page size (bsize) in mpool_hugepage_component.c
	return 0
}

//Statvfs64 implements part of statvfs libc call
//export Statvfs64
func Statvfs64(filename string, vfsstats *C.struct_statvfs64) int {
	//FIXME: this information should be returned by the redisfs instance managing 'filename'
	s := statfs()
	vfsstats.f_bsize = C.ulong(s.Bsize) // block size
	//NOTE: statvfs is used by openmpi to get the fs page size (bsize) in mpool_hugepage_component.c
	return 0
}

//Fadvise ...
//export Fadvise
func Fadvise(fd int, offset, len int64, advice int) int {
	//FIXME: currently no-op, could be leveraged in the future for caching/prefetching
	return 0
}

//Fflush ...
//export Fflush
func Fflush(f *C.FILE) int {
	//currently no-op
	return 0
}

func main() {}
