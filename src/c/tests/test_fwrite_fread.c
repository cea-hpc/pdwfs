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

void test_fwrite_fread() {
 
    FILE* f = fopen(TESTFILE, "w");
    CHECK_NULL(f, "fopen")
 
    size_t n = fwrite("Hello World !\n", 1, 14, f);
    CHECK_ERROR(n, "fwrite")

    int ret = fflush(f);
    CHECK_ERROR(ret, "fflush")

    fclose(f);

    f = fopen(TESTFILE, "r");
    CHECK_NULL(f, "fopen")

    char buf[14];
    n = fread(&buf, 1, 14, f);
    CHECK_ERROR(n, "fread")
    assert(n != 0); // check EOF is not reached

    n = fread(&buf, 1, 14, f);
    CHECK_ERROR(n, "fread")
    assert(n == 0); // check EOF is reached

    assert(strncmp(buf, "Hello World !\n", 14) == 0);

    fclose(f);
    unlink(TESTFILE);
}

int main() {
    test_fwrite_fread();
}