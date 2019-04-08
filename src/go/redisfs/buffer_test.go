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
	"reflect"
	"strings"
	"testing"
)

func TestWriteBuffer(t *testing.T) {
	server, client, _ := InitTestRedis()
	defer server.Close()

	conf := GetMountPathConf()

	key := "key"

	b := NewRedisBuffer(conf, client, key)

	// Write first dots
	if n, err := b.WriteVecAt([][]byte{[]byte(dots)}, 0); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}
	if s, err := client.GetRange(key, 0, int64(len(dots))-1).Result(); err != nil || s != dots {
		t.Errorf("Error in GetRange: %s (val: %q)", err, s)
	}

	// Write second time: abc - Buffer must grow
	if n, err := b.WriteVecAt([][]byte{[]byte(abc)}, int64(len(dots))); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(abc) {
		t.Errorf("Invalid write count: %d", n)
	}
	if s, err := client.GetRange(key, 0, int64(len(dots+abc))-1).Result(); err != nil || s != dots+abc {
		t.Errorf("Error in GetRange: %s (val: %q)", err, s)
	}

	if s := client.StrLen(key).Val(); s != int64(len(dots)+len(abc)) {
		t.Errorf("Origin Buffer did not grow: len=%d", s)
	}

	// Test on case when no Buffer grow is needed
	// Write dots on start of the Buffer
	if n, err := b.WriteVecAt([][]byte{[]byte(dots)}, 0); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}
	if s, err := client.GetRange(key, 0, int64(len(dots))-1).Result(); err != nil || s != dots {
		t.Errorf("Error in GetRange: %s (val: %q)", err, s)
	}

	if s := client.StrLen(key).Val(); s != int64(len(dots)+len(abc)) {
		t.Errorf("Origin Buffer should not grow: len=%d", s)
	}

	// Can not read, ptr at the end
	p := make([]byte, len(dots))
	end := len(dots) + len(abc)
	if n, err := b.ReadVecAt([][]byte{p}, int64(end)); err == nil || n > 0 {
		t.Errorf("Expected read error: %d %s", n, err)
	}

	// Read dots
	if n, err := b.ReadVecAt([][]byte{p}, 0); err != nil || n != len(dots) || string(p) != dots {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}

	// Read abc
	if n, err := b.ReadVecAt([][]byte{p}, int64(len(dots))); err != nil || n != len(abc) || string(p) != abc {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}

	// Write so that Buffer must expand more than 2x
	if n, err := b.WriteVecAt([][]byte{[]byte(large)}, int64(end)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(large) {
		t.Errorf("Invalid write count: %d", n)
	}
	if s, err := client.GetRange(key, 0, int64(len(dots+abc+large))-1).Result(); err != nil || s != dots+abc+large {
		t.Errorf("Error in GetRange: %s (val: %q)", err, s)
	}

	if s := client.StrLen(key).Val(); s != int64(len(dots)+len(abc)+len(large)) {
		t.Errorf("Origin Buffer did not grow: len=%d", s)
	}

}

// TestBufferGrowWriteAndSeek tests if Write and Seek inside the
// Buffers boundaries result in invalid growth
func TestBufferGrowWriteAndSeek(t *testing.T) {
	server, client, _ := InitTestRedis()
	defer server.Close()

	conf := GetMountPathConf()

	key := "key"

	b := NewRedisBuffer(conf, client, key)

	writeByte := func(bt byte, off int64) {
		n, err := b.WriteVecAt([][]byte{[]byte{bt}}, off)
		if err != nil {
			t.Fatalf("Error on write: %s", err)
		} else if n != 1 {
			t.Fatalf("Unexpected num of bytes written: %d", n)
		}
	}

	// Buffer: [][XXX]
	writeByte(0x01, 0)
	// Buffer: [1][XX]
	writeByte(0x02, 1)
	// Buffer: [1,2][X]
	writeByte(0x03, 0)
	// Buffer: [3,2][X]
	writeByte(0x01, 2) // write to end
	// Buffer: [3,2,1][]

	// Check content of buf
	data, err := client.Get(key).Result()
	Ok(t, err)

	buf := []byte(data)
	if !reflect.DeepEqual([]byte{0x03, 0x02, 0x01}, buf) {
		t.Fatalf("Invalid Buffer: %s, len=%d, cap=%d", buf, len(buf), cap(buf))
	}
}

func TestEndOfBuffer(t *testing.T) {
	server, client, _ := InitTestRedis()
	defer server.Close()

	conf := GetMountPathConf()
	conf.BlockSize = 20 // 20 bytes capacity Buffer

	b := NewRedisBuffer(conf, client, "Key")

	data := strings.Repeat("0123456789", 3) // 30 bytes to write
	wrote, err := b.WriteAt([]byte(data), int64(0))
	Ok(t, err)

	rdata := make([]byte, len(data)+10) // read more than what was written
	read, err := b.ReadAt(rdata, int64(0))
	if err != nil && err != ErrEndOfBuffer {
		t.Errorf("Error in read, %d, %s", read, err)
	}
	if read != wrote {
		t.Errorf("Different number of bytes read (%d) and written (%d)", read, wrote)
	}
}
