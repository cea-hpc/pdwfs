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
	"strings"
	"testing"
)

func TestBlocksLayout(t *testing.T) {
	client, _ := GetRedisClient()
	defer client.FlushAll()

	conf := GetMountPathConf()
	conf.BlockSize = 1024

	b := NewRedisBlockedBuffer(conf, client, "Key")

	// all in one block, starting at 0
	rb := b.newBlocksLayout(0, 500)
	Equals(t, len(rb), 1, "Nb of block error")

	if rb[0].id != 0 || rb[0].off != 0 || rb[0].len != 500 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[0].id, rb[0].off, rb[0].len)
	}

	// all in one block, starting at 500
	rb = b.newBlocksLayout(500, 500)
	Equals(t, len(rb), 1, "Nb of block error")

	if rb[0].id != 0 || rb[0].off != 500 || rb[0].len != 500 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[0].id, rb[0].off, rb[0].len)
	}

	// taking exactly one block
	rb = b.newBlocksLayout(0, 1024)
	Equals(t, len(rb), 1, "Nb of block error")

	if rb[0].id != 0 || rb[0].off != 0 || rb[0].len != 1024 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[0].id, rb[0].off, rb[0].len)
	}

	// taking one block + 1 byte
	rb = b.newBlocksLayout(0, 1025)
	Equals(t, len(rb), 2, "Nb of block error")

	if rb[0].id != 0 || rb[0].off != 0 || rb[0].len != 1024 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[0].id, rb[0].off, rb[0].len)
	}
	if rb[1].id != 1 || rb[1].off != 0 || rb[1].len != 1 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[1].id, rb[1].off, rb[1].len)
	}

	// taking exactly two block
	rb = b.newBlocksLayout(0, 2048)
	Equals(t, len(rb), 2, "Nb of block error")

	if rb[0].id != 0 || rb[0].off != 0 || rb[0].len != 1024 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[0].id, rb[0].off, rb[0].len)
	}
	if rb[1].id != 1 || rb[1].off != 0 || rb[1].len != 1024 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[1].id, rb[1].off, rb[1].len)
	}

	// spanning two blocks
	rb = b.newBlocksLayout(500, 1000)
	Equals(t, len(rb), 2, "Nb of block error")

	if rb[0].id != 0 || rb[0].off != 500 || rb[0].len != 524 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[0].id, rb[0].off, rb[0].len)
	}
	if rb[1].id != 1 || rb[1].off != 0 || rb[1].len != 476 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[1].id, rb[1].off, rb[1].len)
	}

	// spanning three blocks, starting on second one, one byte on fourth block
	rb = b.newBlocksLayout(1024, 2049)
	Equals(t, len(rb), 3, "Nb of block error")

	if rb[0].id != 1 || rb[0].off != 0 || rb[0].len != 1024 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[0].id, rb[0].off, rb[0].len)
	}
	if rb[1].id != 2 || rb[1].off != 0 || rb[1].len != 1024 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[1].id, rb[1].off, rb[1].len)
	}
	if rb[2].id != 3 || rb[2].off != 0 || rb[2].len != 1 {
		t.Errorf("error in block data: id %d, off %d, len %d", rb[2].id, rb[2].off, rb[2].len)
	}
}

func writeBlockedBuffer(t *testing.T, blockSize int, datav [][]byte, offset int64) (*RedisBlockedBuffer, error) {
	client, _ := GetRedisClient()
	defer client.FlushAll()

	conf := GetMountPathConf()
	conf.BlockSize = blockSize

	b := NewRedisBlockedBuffer(conf, client, "Key")

	var ntot int
	for _, data := range datav {
		ntot += len(data)
	}

	if n, err := b.WriteVecAt(datav, offset); err != nil {
		return b, fmt.Errorf("Unexpected error: %b", err)
	} else if n != ntot {
		return b, fmt.Errorf("Invalid write count: %d, expecetd %d", n, ntot)
	}

	readData := make([][]byte, len(datav))
	for i, data := range datav {
		readData[i] = make([]byte, len(data))
	}
	if n, err := b.ReadVecAt(readData, offset); err != nil && err != ErrEndOfBuffer || n == 0 {
		return b, fmt.Errorf("Error in read: %d, %b", n, err)
	}

	for i, read := range readData {
		if string(read) != string(datav[i]) {
			return b, fmt.Errorf("Read data does not match written data: %b vs %b", read, datav[i])
		}
	}

	return b, nil
}

func TestWriteBlockedBuffer(t *testing.T) {

	// Single value data vector

	// Data fits within a single block, start at 0 offset
	blockSize := 1024
	data := strings.Repeat("0123456789", 100) // 1000 bytes
	offset := 0
	b, err := writeBlockedBuffer(t, blockSize, [][]byte{[]byte(data)}, int64(offset))
	Ok(t, err)
	Equals(t, len(b.blocks), 1, "Nb of block error")

	// Data fits within a single block, start at non-zero offset
	blockSize = 1024
	data = strings.Repeat("0123456789", 50) // 500 bytes
	offset = 500
	b, err = writeBlockedBuffer(t, blockSize, [][]byte{[]byte(data)}, int64(offset))
	Ok(t, err)
	Equals(t, len(b.blocks), 1, "Nb of block error")

	// Data fits exactly within a single block
	blockSize = 1000
	data = strings.Repeat("0123456789", 100) // 1000 bytes
	offset = 0
	b, err = writeBlockedBuffer(t, blockSize, [][]byte{[]byte(data)}, int64(offset))
	if err != nil && err != ErrEndOfBuffer {
		t.Errorf("WriteBlockedBuffer error: %b", err)
	}
	Equals(t, len(b.blocks), 1, "Nb of block error")

	// Data fits within a block + 1 byte in next block
	blockSize = 999
	data = strings.Repeat("0123456789", 100) // 1000 bytes
	offset = 0
	b, err = writeBlockedBuffer(t, blockSize, [][]byte{[]byte(data)}, int64(offset))
	if err != nil && err != ErrEndOfBuffer {
		t.Errorf("WriteBlockedBuffer error: %b", err)
	}
	Equals(t, len(b.blocks), 2, "Nb of block error")

	// Data fits in two blocks
	blockSize = 1000
	data = strings.Repeat("0123456789", 100) // 1000 bytes
	offset = 500
	b, err = writeBlockedBuffer(t, blockSize, [][]byte{[]byte(data)}, int64(offset))
	Ok(t, err)
	Equals(t, len(b.blocks), 2, "Nb of block error")

	// Data fits in three blocks starting on second
	blockSize = 1000
	data = strings.Repeat("0123456789", 200) // 2000 bytes
	offset = 1500
	b, err = writeBlockedBuffer(t, blockSize, [][]byte{[]byte(data)}, int64(offset))
	Ok(t, err)
	Equals(t, len(b.blocks), 3, "Nb of block error")

	// Multiple value data vector

	// Data vector fits within a single block
	blockSize = 1000
	data = strings.Repeat("0123456789", 10) // 100 bytes
	offset = 500
	b, err = writeBlockedBuffer(t, blockSize, [][]byte{[]byte(data), []byte(data)}, int64(offset))
	Ok(t, err)
	Equals(t, len(b.blocks), 1, "Nb of block error")

	// Data vector fits exactly two blocks
	blockSize = 1000
	data = strings.Repeat("0123456789", 100) // 1000 bytes
	offset = 0
	b, err = writeBlockedBuffer(t, blockSize, [][]byte{[]byte(data), []byte(data)}, int64(offset))
	Ok(t, err)
	Equals(t, len(b.blocks), 2, "Nb of block error")

}

func TestEndOfBlockedBuffer(t *testing.T) {
	client, _ := GetRedisClient()
	defer client.FlushAll()

	conf := GetMountPathConf()
	conf.BlockSize = 20 // 20 bytes block size

	b := NewRedisBlockedBuffer(conf, client, "Key")

	// test reading empty BlockedBuffer
	rdata1 := make([]byte, 30) // read over two blocks
	read, err := b.ReadAt(rdata1, int64(0))
	if err != nil && err != ErrEndOfBuffer {
		t.Errorf("Error in read, %d, %s", read, err)
	}
	if read != 0 {
		t.Errorf("Different number of bytes read (%d) and written (%d)", read, 0)
	}

	data := strings.Repeat("0123456789", 3) // 30 bytes to write
	wrote, err := b.WriteAt([]byte(data), int64(0))
	Ok(t, err)

	rdata := make([]byte, len(data)+10) // read more than what was written
	read, err = b.ReadAt(rdata, int64(0))
	if err != nil && err != ErrEndOfBuffer {
		t.Errorf("Error in read, %d, %b", read, err)
	}
	if read != wrote {
		t.Errorf("Different number of bytes read (%d) and written (%d)", read, wrote)
	}
}

func TestResizeBlockedBuffer(t *testing.T) {
	client, _ := GetRedisClient()
	defer client.FlushAll()

	conf := GetMountPathConf()
	conf.BlockSize = 100 // 100 bytes block size

	b := NewRedisBlockedBuffer(conf, client, "Key")

	Equals(t, len(b.blocks), 0, "Nb of block error")

	err := b.Resize(100)
	Ok(t, err)
	if len(b.blocks) != 1 || b.blocks[0].Size() != 100 {
		t.Errorf("Error in blocks, n blocks %d, len: %d", len(b.blocks), b.blocks[0].Size())
	}

	err = b.Resize(100) // no op
	Ok(t, err)
	if len(b.blocks) != 1 || b.blocks[0].Size() != 100 {
		t.Errorf("Error in blocks, n blocks %d, len: %d", len(b.blocks), b.blocks[0].Size())
	}

	err = b.Resize(250)
	Ok(t, err)
	if len(b.blocks) != 3 || b.blocks[0].Size() != 100 || b.blocks[1].Size() != 100 || b.blocks[2].Size() != 50 {
		t.Errorf("Error in blocks, n blocks %d, len: %d, %d, %d", len(b.blocks), b.blocks[0].Size(), b.blocks[1].Size(), b.blocks[2].Size())
	}

	err = b.Resize(200)
	Ok(t, err)
	if len(b.blocks) != 2 || b.blocks[0].Size() != 100 || b.blocks[1].Size() != 100 {
		t.Errorf("Error in blocks, n blocks %d, len: %d, %d", len(b.blocks), b.blocks[0].Size(), b.blocks[1].Size())
	}

	err = b.Resize(150)
	Ok(t, err)
	if len(b.blocks) != 2 || b.blocks[0].Size() != 100 || b.blocks[1].Size() != 50 {
		t.Errorf("Error in blocks, n blocks %d, len: %d, %d", len(b.blocks), b.blocks[0].Size(), b.blocks[1].Size())
	}

	err = b.Resize(0)
	Ok(t, err)
	if len(b.blocks) != 1 || b.blocks[0].Size() != 0 {
		t.Errorf("Error in blocks, n blocks %d, len: %d", len(b.blocks), b.blocks[0].Size())
	}

}

func TestTruncate(t *testing.T) {
	client, _ := GetRedisClient()
	defer client.FlushAll()

	conf := GetMountPathConf()
	conf.BlockSize = 20 // 20 bytes block size

	b := NewRedisBlockedBuffer(conf, client, "Key")

	data := strings.Repeat("0123456789", 3) // 30 bytes to write
	wrote, err := b.WriteAt([]byte(data), int64(0))
	Ok(t, err)
	Equals(t, wrote, 30, "Error in WriteAt")

	newLen := 15
	err = b.Resize(int64(newLen))
	Ok(t, err)

	rdata := make([]byte, len(data)+10)
	read, err := b.ReadAt(rdata, 0)
	if err != nil && err != ErrEndOfBuffer {
		t.Errorf("Error in read, %d, %b", read, err)
	}
	if read != newLen {
		t.Errorf("Different number of bytes read (%d) and truncated (%d)", read, newLen)
	}

}

func TestMetaBlock(t *testing.T) {
	client, _ := GetRedisClient()
	defer client.FlushAll()

	conf := GetMountPathConf()
	conf.BlockSize = 20 // 20 bytes block size

	b := NewRedisBlockedBuffer(conf, client, "Key")

	n := b.NbBlocks()
	Equals(t, 0, n, "Wrong number of blocks")

	b.metaAddBlock(4)
	n = b.NbBlocks()
	Equals(t, 5, n, "Wrong number of blocks")

	b.metaAddBlock(9)
	n = b.NbBlocks()
	Equals(t, 10, n, "Wrong number of blocks")

	b.metaAddBlock(16)
	n = b.NbBlocks()
	Equals(t, 17, n, "Wrong number of blocks")

	b.metaRemoveBlock(16)
	n = b.NbBlocks()
	Equals(t, 10, n, "Wrong number of blocks")

}
