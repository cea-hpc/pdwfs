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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestCreate(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)
	// NewRedisFS file with absolute path
	{
		f, err := fs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		Ok(t, err)
		Equals(t, "/testfile", f.Name(), "Wrong name")
	}

	// NewRedisFS same file again
	{
		_, err := fs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE, 0666)
		Ok(t, err)

	}

	// NewRedisFS same file again, but truncate it
	{
		_, err := fs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		Ok(t, err)
	}

	// NewRedisFS same file again with O_CREATE|O_EXCL, which is an error
	{
		_, err := fs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err == nil {
			t.Fatalf("Expected error creating file: %s", err)
		}
	}

	// NewRedisFS file with unkown parent
	{
		_, err := fs.OpenFile("/testfile/testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err == nil {
			t.Errorf("Expected error creating file")
		}
	}
}

func TestCreateRelative(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	cwd, err := os.Getwd()
	Ok(t, err)
	mountConf.Path = cwd

	fs := NewRedisFS(redisConf, mountConf)
	// NewRedisFS file with relative path (workingDir == root)
	{
		f, err := fs.OpenFile("relFile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		Ok(t, err)
		Equals(t, filepath.Join(cwd, "relFile"), f.Name(), "Wrong name")
	}
}

func TestMkdirAbs(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)

	// NewRedisFS dir with absolute path
	{
		err := fs.Mkdir("/usr", 0)
		Ok(t, err)
	}

	// NewRedisFS dir twice
	{
		err := fs.Mkdir("/usr", 0)
		if err == nil {
			t.Fatalf("Expecting error creating directory: %s", "/home")
		}
	}
}

func TestMkdirRel(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	cwd, err := os.Getwd()
	Ok(t, err)
	mountConf.Path = cwd

	fs := NewRedisFS(redisConf, mountConf)

	// NewRedisFS dir with relative path
	{
		err := fs.Mkdir("home", 0)
		Ok(t, err)
	}
}

func TestMkdirTree(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)

	err := fs.Mkdir("/home", 0)
	Ok(t, err)

	err = fs.Mkdir("/home/blang", 0)
	Ok(t, err)

	err = fs.Mkdir("/home/blang/goprojects", 0)
	Ok(t, err)

	err = fs.Mkdir("/home/johndoe/goprojects", 0)
	if err == nil {
		t.Errorf("Expected error creating directory with non-existing parent")
	}

	//TODO: Subdir of file
}

func TestReadDir(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)

	dirs := []string{"/home", "/home/linus", "/home/rob", "/home/pike", "/home/blang"}
	expectNames := []string{"/home/README.txt", "/home/blang", "/home/linus", "/home/pike", "/home/rob"}
	for _, dir := range dirs {
		err := fs.Mkdir(dir, 0777)
		Ok(t, err)
	}
	f, err := fs.OpenFile("/home/README.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	Ok(t, err)

	f.Close()

	fis, err := fs.ReadDir("/home")
	Ok(t, err)

	Equals(t, len(fis), len(expectNames), "Wrong size")

	for i, n := range expectNames {
		Equals(t, n, fis[i].Name(), "Wrong name")
	}

	// Readdir empty directory
	if fis, err := fs.ReadDir("/home/blang"); err != nil {
		t.Errorf("Error readdir(empty directory): %s", err)
	} else if l := len(fis); l > 0 {
		t.Errorf("Found entries in non-existing directory: %d", l)
	}

	// Readdir file
	if _, err := fs.ReadDir("/home/README.txt"); err == nil {
		t.Errorf("Expected error readdir(file)")
	}

	// Readdir subdir of file
	if _, err := fs.ReadDir("/home/README.txt/info"); err == nil {
		t.Errorf("Expected error readdir(subdir of file)")
	}

	// Readdir non existing directory
	if _, err := fs.ReadDir("/usr"); err == nil {
		t.Errorf("Expected error readdir(nofound)")
	}

}

func TestRemove(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)

	err := fs.Mkdir("/tmp", 0777)
	Ok(t, err)

	f, err := fs.OpenFile("/tmp/README.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	Ok(t, err)

	_, err = f.Write([]byte("test"))
	Ok(t, err)

	f.Close()

	// remove non existing file
	if err := fs.Remove("/nonexisting.txt"); err == nil {
		t.Errorf("Expected remove to fail")
	}

	// remove non existing file from an non existing directory
	if err := fs.Remove("/nonexisting/nonexisting.txt"); err == nil {
		t.Errorf("Expected remove to fail")
	}

	// remove created file
	err = fs.Remove("/tmp/README.txt")
	Ok(t, err)

	if _, err = fs.OpenFile("/tmp/README.txt", os.O_RDWR, 0666); err == nil {
		t.Errorf("Could open removed file!")
	}

	// Recreate file and check its size
	if f, err = fs.OpenFile("/tmp/README.txt", os.O_CREATE|os.O_RDWR, 0666); err != nil {
		t.Errorf("Could not create removed file!")
	}
	fi, err := fs.Stat(f.Name())
	Ok(t, err)

	if fi.Size() != 0 {
		t.Errorf("Error in Size, got %d (expecting 0)", fi.Size())
	}

	err = fs.Remove("/tmp")
	Ok(t, err)

	if fis, err := fs.ReadDir("/"); err != nil {
		t.Errorf("Readdir error: %s", err)
	} else if len(fis) != 0 {
		t.Errorf("Found files: %s", fis)
	}
}

func TestReadWrite(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)

	f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
	Ok(t, err)

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Write abc
	if n, err := f.Write([]byte(abc)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(abc) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, os.SEEK_SET); err != nil || n != 0 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	// Seek to end of file
	if n, err := f.Seek(0, os.SEEK_END); err != nil || n != 32 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	// Write dots at end of file
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, os.SEEK_SET); err != nil || n != 0 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	p := make([]byte, len(dots)+len(abc)+len(dots))
	if n, err := f.Read(p); err != nil || n != len(dots)+len(abc)+len(dots) {
		t.Errorf("Read error: %d %s", n, err)
	} else if s := string(p); s != dots+abc+dots {
		t.Errorf("Invalid read: %s", s)
	}
}

func TestOpenRO(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)

	f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDONLY, 0666)
	Ok(t, err)

	// Write first dots
	if _, err := f.Write([]byte(dots)); err == nil {
		t.Fatalf("Expected write error")
	}
	f.Close()
}

func TestOpenWO(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)

	f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_WRONLY, 0666)
	Ok(t, err)

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, os.SEEK_SET); err != nil || n != 0 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	// Try reading
	p := make([]byte, len(dots))
	if n, err := f.Read(p); err == nil || n > 0 {
		t.Errorf("Expected invalid read: %d %s", n, err)
	}

	f.Close()
}

func TestOpenAppend(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)

	f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
	Ok(t, err)

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Errorf("Invalid write count: %d", n)
	}
	f.Close()

	// Reopen file in append mode
	f, err = fs.OpenFile("/readme.txt", os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}

	// append dots
	if n, err := f.Write([]byte(abc)); err != nil {
		t.Errorf("Unexpected error: %s", err)
	} else if n != len(abc) {
		t.Errorf("Invalid write count: %d", n)
	}

	// Seek to beginning of file
	if n, err := f.Seek(0, os.SEEK_SET); err != nil || n != 0 {
		t.Errorf("Seek error: %d %s", n, err)
	}

	p := make([]byte, len(dots)+len(abc))
	if n, err := f.Read(p); err != nil || n != len(dots)+len(abc) {
		t.Errorf("Read error: %d %s", n, err)
	} else if s := string(p); s != dots+abc {
		t.Errorf("Invalid read: %s", s)
	}
	f.Close()
}

func TestTruncateToLength(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	var params = []struct {
		size int64
		err  bool
	}{
		{-1, true},
		{0, false},
		{int64(len(dots) - 1), false},
		{int64(len(dots)), false},
		{int64(len(dots) + 1), false},
	}
	for _, param := range params {
		fs := NewRedisFS(redisConf, mountConf)
		f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
		Ok(t, err)
		if n, err := f.Write([]byte(dots)); err != nil {
			t.Errorf("Unexpected error: %s", err)
		} else if n != len(dots) {
			t.Errorf("Invalid write count: %d", n)
		}
		f.Close()

		newSize := param.size
		err = f.Truncate(newSize)
		if param.err {
			if err == nil {
				t.Errorf("Error expected truncating file to length %d", newSize)
			}
			return
		} else if err != nil {
			t.Errorf("Error truncating file: %s", err)
		}

		b, err := readFile(fs, "/readme.txt")
		Ok(t, err)

		if int64(len(b)) != newSize {
			t.Errorf("File should be empty after truncation: %d", len(b))
		}
		if fi, err := fs.Stat("/readme.txt"); err != nil {
			t.Errorf("Error stat file: %s", err)
		} else if fi.Size() != newSize {
			t.Errorf("Filesize should be %d after truncation", newSize)
		}
	}
}

func TestTruncateToZero(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)

	const content = "read me"

	if _, err := writeFile(fs, "/readme.txt", os.O_CREATE|os.O_RDWR, 0666, []byte(content)); err != nil {
		t.Errorf("Unexpected error writing file: %s", err)
	}

	f, err := fs.OpenFile("/readme.txt", os.O_RDWR|os.O_TRUNC, 0666)
	Ok(t, err)

	f.Close()

	b, err := readFile(fs, "/readme.txt")
	Ok(t, err)

	if len(b) != 0 {
		t.Errorf("File should be empty after truncation")
	}
	if fi, err := fs.Stat("/readme.txt"); err != nil {
		t.Errorf("Error stat file: %s", err)
	} else if fi.Size() != 0 {
		t.Errorf("Filesize should be 0 after truncation")
	}
}

func TestStat(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)

	f, err := fs.OpenFile("/readme.txt", os.O_CREATE|os.O_RDWR, 0666)
	Ok(t, err)

	fi, err := fs.Stat(f.Name())
	Ok(t, err)

	if s := fi.Size(); s != int64(0) {
		t.Errorf("Invalid size: %d", s)
	}

	// Write first dots
	if n, err := f.Write([]byte(dots)); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	} else if n != len(dots) {
		t.Fatalf("Invalid write count: %d", n)
	}
	f.Close()

	if err := fs.Mkdir("/tmp", 0777); err != nil {
		t.Fatalf("Mkdir error: %s", err)
	}

	fi, err = fs.Stat(f.Name())
	Ok(t, err)

	// File name is abs name
	if name := f.Name(); name != "/readme.txt" {
		t.Errorf("Invalid file name: %s", name)
	}

	if s := fi.Size(); s != int64(len(dots)) {
		t.Errorf("Invalid size: %d", s)
	}
	if fi.IsDir() {
		t.Errorf("Invalid IsDir")
	}
	if m := fi.Mode(); m != 0666 {
		t.Errorf("Invalid mode: %d", m)
	}
}

func writeFile(fs *RedisFS, name string, flags int, mode os.FileMode, b []byte) (int, error) {
	f, err := fs.OpenFile(name, flags, mode)
	if err != nil {
		return 0, err
	}
	return f.Write(b)
}

func readFile(fs *RedisFS, name string) ([]byte, error) {
	f, err := fs.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

func TestVolumesConcurrentAccess(t *testing.T) {
	client, redisConf := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()
	mountConf.Path = "/"

	fs := NewRedisFS(redisConf, mountConf)

	f1, err := fs.OpenFile("/testfile", os.O_RDWR|os.O_CREATE, 0666)
	Ok(t, err)

	f2, err := fs.OpenFile("/testfile", os.O_RDWR, 0666)
	Ok(t, err)

	// f1 write dots
	if n, err := f1.Write([]byte(dots)); err != nil || n != len(dots) {
		t.Errorf("Unexpected write error: %d %s", n, err)
	}

	p := make([]byte, len(dots))

	// f2 read dots
	if n, err := f2.Read(p); err != nil || n != len(dots) || string(p) != dots {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}

	// f2 write dots
	if n, err := f2.Write([]byte(abc)); err != nil || n != len(abc) {
		t.Errorf("Unexpected write error: %d %s", n, err)
	}

	// f1 read dots
	if n, err := f1.Read(p); err != nil || n != len(abc) || string(p) != abc {
		t.Errorf("Unexpected read error: %d %s, res: %s", n, err, string(p))
	}

}
