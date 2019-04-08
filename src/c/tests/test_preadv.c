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

#define _GNU_SOURCE

#include <fcntl.h>
#include <unistd.h>
#include <assert.h>
#include <sys/uio.h>
#include "tests.h"

int test_preadv() {
    
    int fd = open(TESTFILE, O_CREAT|O_RDWR, 0777);
    CHECK_ERROR(fd, "open")
      
    int n = write(fd, "Hello Golang World !\n", 21);
    CHECK_ERROR(n, "write")

    close(fd);

    fd = open(TESTFILE, O_RDONLY, 0777);
    CHECK_ERROR(fd, "open")

    char start[7], end[8];
    struct iovec iov[2];
    iov[0].iov_base = start;
    iov[0].iov_len = sizeof(start);
    iov[1].iov_base = end;
    iov[1].iov_len = sizeof(end);

    n = preadv(fd, iov, 2, 6);
    CHECK_ERROR(n, "preadv")

    assert(strncmp(start, "Golang ", 7) == 0);
    assert(strncmp(end, "World !\n", 8) == 0);

    close(fd);
    unlink(TESTFILE);

    return 0;
}
