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

void test_pwritev() {

    int fd = open(TESTFILE, O_CREAT|O_RDWR, 0777);
    CHECK_ERROR(fd, "open")

    int n = write(fd, "Hello ", 6);
    CHECK_ERROR(n, "write")

    char *str0 = "Golang ";
    char *str1 = "World !\n";
    struct iovec iov[2];

    iov[0].iov_base = str0;
    iov[0].iov_len = strlen(str0);
    iov[1].iov_base = str1;
    iov[1].iov_len = strlen(str1);

    n = pwritev(fd, iov, 2, 6);
    CHECK_ERROR(n, "pwritev")

    close(fd);

    fd = open(TESTFILE, O_RDONLY, 0777);
    CHECK_ERROR(fd, "open")

    char buf[21];
    n = read(fd, &buf, 21);
    CHECK_ERROR(n, "read")

    assert(strncmp(buf, "Hello Golang World !\n", 21) == 0);

    close(fd);
    unlink(TESTFILE);
}

int main() {
    test_pwritev();
}