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
#include <sys/stat.h>
#include "tests.h"

int test_stat() {

    int fd = open(TESTFILE, O_CREAT|O_RDWR, 0777);
    CHECK_ERROR(fd, "open")

    // test getting stats on file and checking file is a REGular file (used in OpenMPI MPI-IO layer)
    struct stat filestats;
    int err = fstat(fd, &filestats);
    CHECK_ERROR(err, "fstat")

    if (!S_ISREG(filestats.st_mode)) {
        fprintf(stderr, "Error: created file is not a regular file from fstat\n");
        exit(EXIT_FAILURE);
    }

    err = stat(TESTFILE, &filestats);
    CHECK_ERROR(err, "stat")

    if (!S_ISREG(filestats.st_mode)) {
        fprintf(stderr, "Error: created file is not a regular file from stat\n");
        exit(EXIT_FAILURE);
    }
    assert(filestats.st_size == 0);

    close(fd);
    unlink(TESTFILE);

    return 0;
}

int test_stat_size() {
    
    int fd = open(TESTFILE, O_CREAT|O_RDWR, 0777);
    CHECK_ERROR(fd, "open")

    int n = write(fd, "Hello World !\n", 14);
    CHECK_ERROR(n, "write")

    struct stat filestats;
    int err = stat(TESTFILE, &filestats);
    CHECK_ERROR(err, "stat")

    assert(filestats.st_size == 14);

    close(fd);
    unlink(TESTFILE);

    return 0;
}
