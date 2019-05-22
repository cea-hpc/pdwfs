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
	"bytes"
	"testing"

	"github.com/cea-hpc/pdwfs/util"
)

func TestLayout(t *testing.T) {

	stripeSize := int64(1024)

	// all in one stripe, starting at 0
	data := make([]byte, 500)
	s := stripeLayout(stripeSize, 0, data)
	util.Equals(t, len(s), 1, "Nb of stripe error")

	if s[0].id != 0 || s[0].off != 0 || len(s[0].data) != 500 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[0].id, s[0].off, len(s[0].data))
	}

	// all in one stripe, starting at 500
	data = make([]byte, 500)
	s = stripeLayout(stripeSize, 500, data)
	util.Equals(t, len(s), 1, "Nb of stripe error")

	if s[0].id != 0 || s[0].off != 500 || len(s[0].data) != 500 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[0].id, s[0].off, len(s[0].data))
	}

	// taking exactly one stripe
	data = make([]byte, 1024)
	s = stripeLayout(stripeSize, 0, data)
	util.Equals(t, len(s), 1, "Nb of stripe error")

	if s[0].id != 0 || s[0].off != 0 || len(s[0].data) != 1024 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[0].id, s[0].off, len(s[0].data))
	}

	// taking one stripe + 1 byte
	data = make([]byte, 1025)
	s = stripeLayout(stripeSize, 0, data)
	util.Equals(t, len(s), 2, "Nb of stripe error")

	if s[0].id != 0 || s[0].off != 0 || len(s[0].data) != 1024 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[0].id, s[0].off, len(s[0].data))
	}
	if s[1].id != 1 || s[1].off != 0 || len(s[1].data) != 1 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[1].id, s[1].off, len(s[1].data))
	}

	// taking exactly two stripe
	data = make([]byte, 2048)
	s = stripeLayout(stripeSize, 0, data)
	util.Equals(t, len(s), 2, "Nb of stripe error")

	if s[0].id != 0 || s[0].off != 0 || len(s[0].data) != 1024 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[0].id, s[0].off, len(s[0].data))
	}
	if s[1].id != 1 || s[1].off != 0 || len(s[1].data) != 1024 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[1].id, s[1].off, len(s[1].data))
	}

	// spanning two stripes
	data = make([]byte, 1000)
	s = stripeLayout(stripeSize, 500, data)
	util.Equals(t, len(s), 2, "Nb of stripe error")

	if s[0].id != 0 || s[0].off != 500 || len(s[0].data) != 524 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[0].id, s[0].off, len(s[0].data))
	}
	if s[1].id != 1 || s[1].off != 0 || len(s[1].data) != 476 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[1].id, s[1].off, len(s[1].data))
	}

	// spanning three stripes, starting on second one, one byte on fourth stripe
	data = make([]byte, 2049)
	s = stripeLayout(stripeSize, 1024, data)
	util.Equals(t, len(s), 3, "Nb of stripe error")

	if s[0].id != 1 || s[0].off != 0 || len(s[0].data) != 1024 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[0].id, s[0].off, len(s[0].data))
	}
	if s[1].id != 2 || s[1].off != 0 || len(s[1].data) != 1024 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[1].id, s[1].off, len(s[1].data))
	}
	if s[2].id != 3 || s[2].off != 0 || len(s[2].data) != 1 {
		t.Errorf("error in stripe data: id %d, off %d, len %d", s[2].id, s[2].off, len(s[2].data))
	}
}

func writeData(t *testing.T, stripeSize int64, data []byte, off int64) {
	redis, conf := util.InitRedisTestServer()
	defer redis.Stop()

	store := NewDataStore(NewRedisRing(conf), stripeSize)
	defer store.Close()

	store.WriteAt("myfile", off, data)
	readData := make([]byte, len(data), len(data))
	n := store.ReadAt("myfile", off, readData)
	util.Equals(t, int64(len(data)), n, "number of bytes read does not match input")
	util.Equals(t, data, readData, "read data does not match written data")
}

func TestWriteData(t *testing.T) {

	var stripeSize int64

	// Data fits within a single stripe, start at 0 offset
	stripeSize = 1024
	data := bytes.Repeat([]byte("0123456789"), 100) // 1000 bytes
	offset := 0
	writeData(t, stripeSize, data, int64(offset))

	// Data fits within a single stripe, start at non-zero offset
	stripeSize = 1024
	data = bytes.Repeat([]byte("0123456789"), 50) // 500 bytes
	offset = 500
	writeData(t, stripeSize, data, int64(offset))

	// Data fits exactly within a single stripe
	stripeSize = 1000
	data = bytes.Repeat([]byte("0123456789"), 100) // 1000 bytes
	offset = 0
	writeData(t, stripeSize, data, int64(offset))

	// Data fits within a stripe + 1 byte in next stripe
	stripeSize = 999
	data = bytes.Repeat([]byte("0123456789"), 100) // 1000 bytes
	offset = 0
	writeData(t, stripeSize, data, int64(offset))

	// Data fits in two stripes
	stripeSize = 1000
	data = bytes.Repeat([]byte("0123456789"), 100) // 1000 bytes
	offset = 500
	writeData(t, stripeSize, data, int64(offset))

	// Data fits in three stripes starting on second
	stripeSize = 1000
	data = bytes.Repeat([]byte("0123456789"), 200) // 2000 bytes
	offset = 1500
	writeData(t, stripeSize, data, int64(offset))
}

func TestReadEmpty(t *testing.T) {
	redis, conf := util.InitRedisTestServer()
	defer redis.Stop()

	store := NewDataStore(NewRedisRing(conf), 100)
	defer store.Close()

	readData := make([]byte, 1000, 1000)
	n := store.ReadAt("myfile", 0, readData)
	util.Equals(t, int64(0), n, "number of byte read should be 0")
}

func TestGetSize(t *testing.T) {
	redis, conf := util.InitRedisTestServer()
	defer redis.Stop()

	store := NewDataStore(NewRedisRing(conf), 100)
	defer store.Close()

	data := bytes.Repeat([]byte("0123456789"), 500) // 5000 bytes
	store.WriteAt("myfile", 0, data)

	readData := make([]byte, len(data), len(data))
	store.ReadAt("myfile", 0, readData)
	util.Equals(t, data, readData, "read data is different from written data")

	s := store.GetSize("myfile")
	util.Equals(t, int64(len(data)), s, "size is incorrect")
}

func TestResize(t *testing.T) {
	redis, conf := util.InitRedisTestServer()
	defer redis.Stop()

	store := NewDataStore(NewRedisRing(conf), 100)
	defer store.Close()

	store.Resize("myfile", 100)
	util.Equals(t, int64(100), store.GetSize("myfile"), "resize error")

	store.Resize("myfile", 100) // no op
	util.Equals(t, int64(100), store.GetSize("myfile"), "resize error")

	store.Resize("myfile", 250)
	util.Equals(t, int64(250), store.GetSize("myfile"), "resize error")

	store.Resize("myfile", 150)
	util.Equals(t, int64(150), store.GetSize("myfile"), "resize error")

	store.Resize("myfile", 0)
	util.Equals(t, int64(0), store.GetSize("myfile"), "resize error")
}

func TestTruncate(t *testing.T) {
	redis, conf := util.InitRedisTestServer()
	defer redis.Stop()

	store := NewDataStore(NewRedisRing(conf), 20) // 20 bytes stripes
	defer store.Close()

	data := bytes.Repeat([]byte("0123456789"), 3) // 30 bytes to write
	store.WriteAt("myfile", 0, data)

	store.Resize("myfile", 15)

	readSize := int64(len(data) + 10)
	readData := make([]byte, readSize, readSize)
	n := store.ReadAt("myfile", 0, readData)
	util.Equals(t, int64(15), n, "read error")
	util.Equals(t, data[:15], readData[:n], "data read does not match data written")
}
