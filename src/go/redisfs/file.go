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
	"sync"
)

// MemFile represents a file backed by a Store which is secured from concurrent access.
type MemFile struct {
	buf    Buffer
	path   string
	offset int64
	mtx    *sync.RWMutex
}

// NewMemFile creates a file which byte slice is safe from concurrent access,
// the file itself is not thread-safe.
func NewMemFile(buf Buffer, path string, mtx *sync.RWMutex) *MemFile {
	return &MemFile{
		buf:  buf,
		path: path,
		mtx:  mtx,
	}
}

// Name of the file
func (f MemFile) Name() string {
	return f.path
}

// Size of file
func (f MemFile) Size() int64 {
	return f.buf.Size()
}

// Sync has no effect
func (f MemFile) Sync() error {
	return nil
}

// Truncate changes the size of the file
func (f MemFile) Truncate(size int64) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.buf.Resize(size)
}

// Close the file (no op)
func (f MemFile) Close() error {
	return nil
}

func (f MemFile) readAt(dst []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("readVecAt: negative offset")
	}

	read, err := f.buf.ReadAt(dst, off)
	if err != nil {
		if err == ErrEndOfBuffer {
			return read, io.EOF
		}
		return read, err
	}
	if read < len(dst) {
		return read, io.EOF
	}
	return read, nil
}

// Read reads len(dst) byte starting at the current offset.
func (f *MemFile) Read(dst []byte) (int, error) {
	f.mtx.RLock() // should this be a Lock instead of RLock to safeguard offset update?
	defer f.mtx.RUnlock()
	read, err := f.readAt(dst, f.offset)
	f.offset += int64(read)
	return read, err
}

// ReadAt reads len(dst) bytes starting at offset off.
func (f MemFile) ReadAt(dst []byte, off int64) (int, error) {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.readAt(dst, off)
}

func (f MemFile) readVecAt(dstv [][]byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("readVecAt: negative offset")
	}

	var size int
	for _, dst := range dstv {
		size += len(dst)
	}
	read, err := f.buf.ReadVecAt(dstv, off)
	if err != nil {
		if err == ErrEndOfBuffer {
			return read, io.EOF
		}
		return read, err
	}
	if read < size {
		return read, io.EOF
	}
	return read, nil
}

// ReadVec reads a vector of byte slices starting at the current offset.
func (f *MemFile) ReadVec(dstv [][]byte) (int, error) {
	f.mtx.RLock() // should this be a Lock instead of RLock to safeguard offset update?
	defer f.mtx.RUnlock()
	read, err := f.readVecAt(dstv, f.offset)
	f.offset += int64(read)
	return read, err
}

// ReadVecAt reads a vector of byte slices starting at offset off.
func (f MemFile) ReadVecAt(dstv [][]byte, off int64) (int, error) {
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.readVecAt(dstv, off)
}

func (f MemFile) writeAt(data []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("writeAt: negative offset")
	}
	return f.buf.WriteAt(data, off)
}

// Write writes len(data) byte starting at the current offset
func (f *MemFile) Write(data []byte) (int, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	wrote, err := f.writeAt(data, f.offset)
	f.offset += int64(wrote)
	return wrote, err
}

// WriteAt writes len(data) byte starting at the offset off
func (f MemFile) WriteAt(data []byte, off int64) (int, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.writeAt(data, off)
}

func (f MemFile) writeVecAt(datav [][]byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("writeVecAt: negative offset")
	}
	return f.buf.WriteVecAt(datav, off)
}

// WriteVec writes a vector of byte slices starting at the current offset
func (f *MemFile) WriteVec(datav [][]byte) (int, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	wrote, err := f.writeVecAt(datav, f.offset)
	f.offset += int64(wrote)
	return wrote, err
}

// WriteVecAt writes a vector of byte slices at offset off
func (f MemFile) WriteVecAt(datav [][]byte, off int64) (int, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.writeVecAt(datav, off)
}

// Seek sets the offset for the next Read or Write to offset off,
// interpreted according to whence:
// 	0 (os.SEEK_SET) means relative to the origin of the file
// 	1 (os.SEEK_CUR) means relative to the current offset
// 	2 (os.SEEK_END) means relative to the end of the file
// It returns the new offset and an error, if any.
func (f *MemFile) Seek(off int64, whence int) (int64, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	var abs int64
	switch whence {
	case os.SEEK_SET: // Relative to the origin of the file
		abs = off
	case os.SEEK_CUR: // Relative to the current offset
		abs = int64(f.offset) + off
	case os.SEEK_END: // Relative to the end
		abs = f.Size() + off
	default:
		return 0, errors.New("Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("Seek: negative position")
	}
	if abs > f.Size() {
		return 0, errors.New("Seek: too far")
	}
	f.offset = abs
	return abs, nil
}
