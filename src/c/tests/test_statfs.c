/*
* Copyright 2019 CEA
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* 	http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/

#include <fcntl.h>
#include <unistd.h>
#include <assert.h>
#include <sys/statfs.h>
#include "tests.h"

int test_statfs() {
    
    int fd = open(TESTFILE, O_CREAT|O_RDWR, 0777);
    CHECK_ERROR(fd, "open")

    // test getting stats on mounted file system and check fs type
    struct statfs fsstats;
    int err = statfs(TESTFILE, &fsstats);
    CHECK_ERROR(err, "statfs")
    
    if (getenv("PDWFS")) {
        // if running on pdwfs, we check it "fakes" an ext2 filesystem
        assert(fsstats.f_type == 0xEF53);
    }
    close(fd);
    unlink(TESTFILE);

    return 0;
}
