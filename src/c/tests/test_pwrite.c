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
#include "tests.h"

void test_pwrite() {

    int fd = open(TESTFILE, O_CREAT|O_RDWR, 0777);
    CHECK_ERROR(fd, "open")

    int n = write(fd, "Hello World !\n", 14);
    CHECK_ERROR(n, "write")

    n = pwrite(fd, "Golang !\n", 9, 6);
    CHECK_ERROR(n, "pwrite")

    // checks that pwrite has not changed the file offset
    int l = lseek(fd, 0, SEEK_CUR);
    assert(l == 14);

    close(fd);

    fd = open(TESTFILE, O_RDONLY, 0777);
    CHECK_ERROR(fd, "open")

    char buf[15];
    n = read(fd, &buf, 15);
    CHECK_ERROR(n, "read")

    assert(strncmp(buf, "Hello Golang !\n", 15) == 0);

    close(fd);
    unlink(TESTFILE);
}

int main() {
    test_pwrite();
}