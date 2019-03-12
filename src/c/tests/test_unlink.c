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
#include "tests.h"

void test_unlink() {

    int fd = open(TESTFILE, O_CREAT|O_RDWR, 0777);
    CHECK_ERROR(fd, "open")

    int n = write(fd, "Hello World !\n", 14);
    CHECK_ERROR(n, "write")

    close(fd);

    int ret = unlink(TESTFILE);
    CHECK_ERROR(ret, "unlink")

    fd = open(TESTFILE, O_RDONLY, 0777);
    assert(fd == -1);

    fd = open(TESTFILE, O_CREAT|O_RDWR, 0777);
    CHECK_ERROR(fd, "open")

    // check there's nothing to read (ensure that content of file was deleted in backend storage)
    char buf[14];
    n = read(fd, &buf, 14);
    CHECK_ERROR(n, "read")
    assert(n == 0); // check there's nothing to read

    close(fd);
    unlink(TESTFILE);
}

int main() {
    test_unlink();
 }