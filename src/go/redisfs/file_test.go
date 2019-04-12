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
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/cea-hpc/pdwfs/config"
	"github.com/cea-hpc/pdwfs/util"
)

const (
	dots = "1....2....3....4"
	abc  = "abcdefghijklmnop"
)

var (
	large = strings.Repeat("0123456789", 200)     // 2000 bytes
	huge  = strings.Repeat("0123456789", 2000000) // 20 MB
)

func setupMemFile(t *testing.T) (*MemFile, *miniredis.Miniredis, *FileContentClient) {
	server, conf := util.InitMiniRedis()
	dataClient := NewFileContentClient(conf, config.DefaultStripeSize)
	f := NewMemFile(dataClient, "/path/to/file", &sync.RWMutex{})
	return f, server, dataClient
}

func TestFileInterface(t *testing.T) {
	f, server, client := setupMemFile(t)
	defer server.Close()
	defer client.Close()

	_ = File(f)
}

func TestWrite(t *testing.T) {
	f, server, client := setupMemFile(t)
	defer server.Close()
	defer client.Close()

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Write second time: abc
	if n, err := f.Write([]byte(abc)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(abc) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Reset seek cursor
	if n, err := f.Seek(0, os.SEEK_SET); err != nil || n != 0 {
		t.Errorf("Invalid seek result: %d %s", n, err)
	}

	// Write dots from beginning
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Restore seek cursor
	if n, err := f.Seek(0, os.SEEK_END); err != nil {
		t.Errorf("Invalid seek result: %d %s", n, err)
	}

	// Can not read, ptr at the end
	p := make([]byte, len(dots))
	if n, err := f.Read(p); err == nil || n > 0 {
		t.Errorf("Expected read error: %d %s", n, err)
	}

	if n, err := f.Seek(0, os.SEEK_SET); err != nil || n != 0 {
		t.Errorf("Invalid seek result: %d %s", n, err)
	}

	// Read dots
	if n, err := f.Read(p); err != nil || n != len(dots) || string(p) != dots {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}

	// Read abc
	if n, err := f.Read(p); err != nil || n != len(abc) || string(p) != abc {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}

	// Seek abc backwards
	if n, err := f.Seek(int64(-len(abc)), os.SEEK_END); err != nil || n != int64(len(dots)) {
		t.Errorf("Invalid seek result: %d %s", n, err)
	}

	// Seek 8 forwards
	if n, err := f.Seek(int64(len(abc)/2), os.SEEK_CUR); err != nil || n != int64(16)+int64(len(abc)/2) {
		t.Errorf("Invalid seek result: %d %s", n, err)
	}

	// Seek to end
	if n, err := f.Seek(0, os.SEEK_END); err != nil || n != int64(f.Size()) {
		t.Errorf("Invalid seek result: %d %s", n, err)
	}

	// Write large
	if n, err := f.Write([]byte(large)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(large) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Write huge
	if n, err := f.Write([]byte(huge)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(huge) {
		t.Errorf("Invalid write count: %d", n)
	}
}

func TestSeek(t *testing.T) {
	f, server, client := setupMemFile(t)
	defer server.Close()
	defer client.Close()

	// write dots
	if n, err := f.Write([]byte(dots)); err != nil || n != len(dots) {
		t.Errorf("Unexpected write error: %d %s", n, err)
	}

	// invalid whence
	if _, err := f.Seek(0, 4); err == nil {
		t.Errorf("Expected invalid whence error")
	}
	// seek to -1
	if _, err := f.Seek(-1, os.SEEK_SET); err == nil {
		t.Errorf("Expected invalid position error")
	}

	// seek to end
	if _, err := f.Seek(0, os.SEEK_END); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	// seek past the end
	if _, err := f.Seek(1, os.SEEK_END); err != nil {
		t.Errorf("Can't seek past end")
	}
}

func TestRead(t *testing.T) {
	f, server, client := setupMemFile(t)
	defer server.Close()
	defer client.Close()

	// write dots
	if n, err := f.Write([]byte(dots)); err != nil || n != len(dots) {
		t.Errorf("Unexpected write error: %d %s", n, err)
	}
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		t.Errorf("Unexepected seek error %s", err)
	}

	// p := make([]byte, len(dots))
	var p []byte

	// Read to empty buffer, err==nil, n == 0
	if n, err := f.Read(p); err != nil || n > 0 {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}
}

func TestReadAt(t *testing.T) {
	f, server, client := setupMemFile(t)
	defer server.Close()
	defer client.Close()

	// write dots
	if n, err := f.Write([]byte(dots)); err != nil || n != len(dots) {
		t.Errorf("Unexpected write error: %d %s", n, err)
	}

	p := make([]byte, len(dots))

	// Read to empty buffer, err==nil, n == 0
	if n, err := f.ReadAt(p[:0], 0); err != nil || n > 0 {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}

	// Read the entire buffer, err==nil, n == len(dots)
	if n, err := f.ReadAt(p, 0); err != nil || n != len(dots) || string(p[:n]) != dots {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}

	// Read the buffer while crossing the end, err==io.EOF, n == len(dots)-1
	if n, err := f.ReadAt(p, 1); err != io.EOF || n != len(dots)-1 || string(p[:n]) != dots[1:] {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}

	// Read at the buffer's end, err==io.EOF, n == 0
	if n, err := f.ReadAt(p, int64(len(dots))); err != io.EOF || n > 0 {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}

	// Read past the buffer's end, err==io.EOF, n == 0
	if n, err := f.ReadAt(p, int64(len(dots)+1)); err != io.EOF || n > 0 {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}
}

func TestSize(t *testing.T) {
	f, server, client := setupMemFile(t)
	defer server.Close()
	defer client.Close()

	// write dots
	if n, err := f.Write([]byte(dots)); err != nil || n != len(dots) {
		t.Errorf("Unexpected write error: %d %s", n, err)
	}

	if s := f.Size(); s != int64(len(dots)) {
		t.Fatalf("Unexpected file size: %d (expected %d)", s, int64(len(dots)))
	}
}
