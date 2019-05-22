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

#include <unistd.h>
#include <assert.h>
#include <sys/stat.h>
#include "tests.h"

int test_mkdir_rmdir() {

    int ret = mkdir(TESTDIR, 0777);
    assert(ret == 0);

    struct stat dirstats;
    int err = stat(TESTDIR, &dirstats);
    CHECK_ERROR(err, "stat")

    if (!S_ISDIR(dirstats.st_mode)) {
        fprintf(stderr, "Error: created file is not a directory from stat\n");
        exit(EXIT_FAILURE);
    }

    ret = rmdir(TESTDIR);
    assert(ret == 0);

    err = stat(TESTDIR, &dirstats);
    assert(err != 0);

    return 0;
}
