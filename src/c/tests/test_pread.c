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

void test_pread() {
    
    int fd = open(TESTFILE, O_CREAT|O_RDWR, 0777);
    CHECK_ERROR(fd, "open")
      
    int n = write(fd, "Hello World !\n", 14);
    CHECK_ERROR(n, "write")

    close(fd);

    fd = open(TESTFILE, O_RDONLY, 0777);
    CHECK_ERROR(fd, "open")

    char buf[8];
    n = pread(fd, buf, 8, 6);
    CHECK_ERROR(n, "pread")

    assert(strncmp(buf, "World !\n", 8) == 0);

    // checks that pread has not changed the file offset
    int l = lseek(fd, 0, SEEK_CUR);
    assert(l == 0);

    close(fd);
    unlink(TESTFILE);
}

int main() {
    test_pread();
}