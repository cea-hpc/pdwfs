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
	"testing"
)

func TestInodeMeta(t *testing.T) {
	client, _ := GetRedisClient()
	defer client.FlushAll()

	mountConf := GetMountPathConf()

	i := NewInode(mountConf, client, "id")

	res, err := i.hasMeta()
	Ok(t, err)
	Equals(t, false, res, "no metadata expected")

	md := &inodeMeta{
		Name: "Luke",
	}

	err = i.initMeta(md)
	Ok(t, err)

	res, err = i.hasMeta()
	Ok(t, err)
	Equals(t, true, res, "metadata expected")

	md2 := &inodeMeta{
		Name: "Yoda",
	}

	err = i.initMeta(md2) // should be a no op
	Ok(t, err)

	val := i.getMeta()
	Ok(t, err)
	Equals(t, md, val, "Wrong metadata")

	err = i.setMeta(md2)
	Ok(t, err)

	val = i.getMeta()
	Ok(t, err)
	Equals(t, md2, val, "Wrong metadata")

	err = i.delMeta()
	Ok(t, err)

	val = i.getMeta()
	var nilMeta *inodeMeta
	Equals(t, nilMeta, val, "Wrong metadata")
}
