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
	"math/bits"
	"sync"

	"github.com/cea-hpc/pdwfs/config"
	"github.com/go-redis/redis"
)

var (
	errBlockMustExists = errors.New("Getting a block for reading that does not exist")
)

// RedisBlockedBuffer is a composite Buffer that writes/reads data in a set fixed-sized Buffers (or blocks).
type RedisBlockedBuffer struct {
	conf        *config.Mount
	client      IRedisClient
	blocks      map[int]Buffer
	blockSize   int
	key         string
	keyNbBlocks string
	mtx         *sync.RWMutex
}

// NewRedisBlockedBuffer creates a new data volume based on a Buffer
func NewRedisBlockedBuffer(conf *config.Mount, client IRedisClient, key string) *RedisBlockedBuffer {
	if conf.BlockSize == 0 {
		panic(fmt.Errorf("BlockSize in configuration is not set"))
	}
	return &RedisBlockedBuffer{
		conf:        conf,
		client:      client,
		blocks:      make(map[int]Buffer),
		blockSize:   conf.BlockSize,
		key:         key,
		keyNbBlocks: key + ":NbBlocks",
		mtx:         &sync.RWMutex{},
	}
}

func (b *RedisBlockedBuffer) metaAddBlock(id int) {
	try(b.client.SetBit(b.keyNbBlocks, int64(id), 1).Err())
}

func (b *RedisBlockedBuffer) metaRemoveBlock(id int) {
	try(b.client.SetBit(b.keyNbBlocks, int64(id), 0).Err())
}

//NbBlocks returns the total number of blocks taken
func (b *RedisBlockedBuffer) NbBlocks() int {
	blocksBitmap, err := b.client.Get(b.keyNbBlocks).Bytes()
	if err == redis.Nil {
		return 0
	}
	check(err)
	// get last non-null byte in bitmap
	i := len(blocksBitmap) - 1
	for {
		if blocksBitmap[i] != 0 || i == 0 {
			break
		}
		i--
	}
	nbBlocks := 8*i + (8 - bits.TrailingZeros8(uint8(blocksBitmap[i])))
	return nbBlocks
}

func (b *RedisBlockedBuffer) getBlockedBufferByID(id int) Buffer {
	if _, ok := b.blocks[id]; !ok {
		b.metaAddBlock(id)
		b.blocks[id] = NewRedisBuffer(b.conf, b.client, fmt.Sprintf("%s:%d", b.key, id))
	}
	return b.blocks[id]
}

func (b *RedisBlockedBuffer) removeBlockedBufferByID(id int) error {
	err := b.getBlockedBufferByID(id).Clear()
	if err != nil {
		return err
	}
	b.metaRemoveBlock(id)
	delete(b.blocks, id)
	return nil
}

// Size returns the total length of data in the RedisBlockedBuffer
func (b *RedisBlockedBuffer) Size() int64 {
	n := b.NbBlocks()
	if n == 0 {
		return 0
	}
	return int64(b.blockSize*(n-1)) + b.getBlockedBufferByID(n-1).Size()
}

type blockInfo struct {
	id   int
	off  int64    // offset relative to beginning of Buffer
	data [][]byte // slice of data slices
	len  int      // length of data in Buffer
	buf  Buffer
}

func (b *RedisBlockedBuffer) newBlocksLayout(off, size int64) []blockInfo {

	startID := int(off / int64(b.blockSize))
	endID := int((off + size - 1) / int64(b.blockSize)) // last block inclusive
	nBlocks := endID - startID + 1

	rb := make([]blockInfo, nBlocks)

	// first block
	rb[0].id = startID
	rb[0].off = off % int64(b.blockSize)
	if nBlocks == 1 {
		rb[0].len = int(size)
	} else {
		rb[0].len = int(int64(b.blockSize) - rb[0].off)
	}
	rb[0].buf = b.getBlockedBufferByID(rb[0].id)

	if nBlocks > 1 {
		//last block, inclusive
		rb[nBlocks-1].id = endID
		rb[nBlocks-1].off = 0
		rb[nBlocks-1].len = int((off+size-1)%int64(b.blockSize) + 1)
		rb[nBlocks-1].buf = b.getBlockedBufferByID(rb[nBlocks-1].id)

		// other blocks (nBlocks > 2)
		for i := 1; i < nBlocks-1; i++ {
			rb[i].id = startID + i
			rb[i].off = 0
			rb[i].len = b.blockSize
			rb[i].buf = b.getBlockedBufferByID(rb[i].id)
		}
	}
	return rb
}

func (b *RedisBlockedBuffer) relevantBlocks(datav [][]byte, off, size int64) []blockInfo {

	blockInfos := b.newBlocksLayout(off, size)

	nBlocks := len(blockInfos)
	nData := len(datav)
	iBlock := 0
	block := &blockInfos[iBlock]
	offsetInBlock := int((*block).off)
	iData := 0
	data := datav[iData]
	offsetInData := 0
	for {
		remainBlockSize := b.blockSize - offsetInBlock
		remainDataSize := len(data) - offsetInData

		if remainDataSize <= remainBlockSize {
			(*block).data = append((*block).data, data[offsetInData:offsetInData+remainDataSize])
			offsetInBlock += remainDataSize
			// move to next data
			iData++
			if iData >= nData {
				break
			}
			data = datav[iData]
			offsetInData = 0
			continue
		} else {
			(*block).data = append((*block).data, data[offsetInData:offsetInData+remainBlockSize])
			offsetInData += remainBlockSize
			// move to next block
			iBlock++
			if iBlock >= nBlocks {
				break
			}
			block = &blockInfos[iBlock]
			offsetInBlock = 0
			continue
		}
	}
	return blockInfos
}

type chanReturnData struct {
	n   int
	err error
}

func (b *RedisBlockedBuffer) writeBlocks(blockInfos []blockInfo) (int, error) {

	retChan := make(chan chanReturnData)
	for _, blockInfo := range blockInfos {
		go func(buf Buffer, datav [][]byte, off int64, retChan chan<- chanReturnData) {
			wrote, err := buf.WriteVecAt(datav, off)
			retChan <- chanReturnData{wrote, err}
		}(blockInfo.buf, blockInfo.data, blockInfo.off, retChan)
	}

	var n int
	var err error
	for range blockInfos {
		retData := <-retChan
		if retData.err != nil {
			err = retData.err
		}
		n += retData.n
	}
	return n, err
}

//WriteAt writes a byte slices starting at byte offset off.
func (b *RedisBlockedBuffer) WriteAt(data []byte, off int64) (int, error) {
	blockInfos := b.relevantBlocks([][]byte{data}, off, int64(len(data)))
	return b.writeBlocks(blockInfos)
}

//WriteVecAt writes a vector of byte slices starting at byte offset off.
func (b *RedisBlockedBuffer) WriteVecAt(datav [][]byte, off int64) (int, error) {
	var size int
	for _, data := range datav {
		size += len(data)
	}
	blockInfos := b.relevantBlocks(datav, off, int64(size))
	return b.writeBlocks(blockInfos)
}

func (b *RedisBlockedBuffer) readBlocks(blockInfos []blockInfo) (int, error) {

	retChan := make(chan chanReturnData)
	for _, blockInfo := range blockInfos {
		go func(buf Buffer, datav [][]byte, off int64, retChan chan<- chanReturnData) {
			read, err := buf.ReadVecAt(datav, off)
			retChan <- chanReturnData{read, err}
		}(blockInfo.buf, blockInfo.data, blockInfo.off, retChan)
	}

	var n int
	var err error
	for range blockInfos {
		retData := <-retChan
		if retData.err != nil {
			err = retData.err
		}
		n += retData.n
	}
	return n, err
}

//ReadAt reads a vector of byte slices starting at byte offset off
func (b *RedisBlockedBuffer) ReadAt(data []byte, off int64) (int, error) {
	blockInfos := b.relevantBlocks([][]byte{data}, off, int64(len(data)))
	return b.readBlocks(blockInfos)
}

//ReadVecAt reads a vector of byte slices from the Buffer starting at byte offset off
func (b *RedisBlockedBuffer) ReadVecAt(datav [][]byte, off int64) (int, error) {
	var size int
	for _, data := range datav {
		size += len(data)
	}
	blockInfos := b.relevantBlocks(datav, off, int64(size))
	return b.readBlocks(blockInfos)
}

// Resize resizes the Buffer to a given size.
// It returns an error if the given size is negative.
// If the Buffer is larger than the specified size, the extra data is lost.
// If the Buffer is smaller, it is extended and the extended part (hole)
// reads as zero bytes.
func (b *RedisBlockedBuffer) Resize(size int64) error {
	if size < 0 {
		return errors.New("Resize: size must be non-negative")
	}
	if storeSize := b.Size(); size == storeSize {
		return nil
	} else if size < storeSize {
		err := b.shrink(size)
		if err != nil {
			return err
		}
	} else {
		err := b.grow(size)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *RedisBlockedBuffer) shrink(size int64) error {
	newBlocksInfo := b.newBlocksLayout(0, size)

	newLastBlockInfo := newBlocksInfo[len(newBlocksInfo)-1]
	// remove all existing blocks after this new last block
	for id, max := newLastBlockInfo.id+1, b.NbBlocks(); id < max; id++ {
		err := b.removeBlockedBufferByID(id)
		if err != nil {
			return err
		}
	}
	// resize the last block to correct size
	err := newLastBlockInfo.buf.Resize(int64(newLastBlockInfo.len))
	if err != nil {
		return err
	}
	return nil
}

func (b *RedisBlockedBuffer) grow(size int64) error {

	var lastBlockID int // ID of existing last block
	if l := b.NbBlocks(); l == 0 {
		// empty RedisBlockedBuffer, no block was ever created
		lastBlockID = 0
	} else {
		lastBlockID = l - 1
	}

	newBlocksInfo := b.newBlocksLayout(0, size)

	oldLastBlockInfo := newBlocksInfo[lastBlockID]
	// get all blocks including and after the existing last one and size them correctly
	for id := oldLastBlockInfo.id; id < len(newBlocksInfo); id++ {
		blockInfo := newBlocksInfo[id]
		blockInfo.buf.Resize(int64(blockInfo.len))
	}
	return nil
}

// Clear the Buffers
func (b *RedisBlockedBuffer) Clear() error {
	var err error
	for id, max := 0, b.NbBlocks(); id < max; id++ {
		err = b.removeBlockedBufferByID(id)
	}
	// FIXME: wrap or stack all errors in one error ?
	b.client.Del(b.keyNbBlocks)
	return err
}
