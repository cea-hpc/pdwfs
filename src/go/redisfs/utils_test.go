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
	"testing"
)

func TestSplitPath(t *testing.T) {
	const PathSeperator = "/"
	if p := SplitPath("/", PathSeperator); !reflect.DeepEqual(p, []string{""}) {
		t.Errorf("Invalid path: %q", p)
	}
	if p := SplitPath("./test", PathSeperator); !reflect.DeepEqual(p, []string{".", "test"}) {
		t.Errorf("Invalid path: %q", p)
	}
	if p := SplitPath(".", PathSeperator); !reflect.DeepEqual(p, []string{"."}) {
		t.Errorf("Invalid path: %q", p)
	}
	if p := SplitPath("test", PathSeperator); !reflect.DeepEqual(p, []string{".", "test"}) {
		t.Errorf("Invalid path: %q", p)
	}
	if p := SplitPath("/usr/src/linux/", PathSeperator); !reflect.DeepEqual(p, []string{"", "usr", "src", "linux"}) {
		t.Errorf("Invalid path: %q", p)
	}
	if p := SplitPath("usr/src/linux/", PathSeperator); !reflect.DeepEqual(p, []string{".", "usr", "src", "linux"}) {
		t.Errorf("Invalid path: %q", p)
	}
}
