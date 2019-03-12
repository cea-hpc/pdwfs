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

void test_fgets() {
 
    FILE* f = fopen(TESTFILE, "w");
    CHECK_NULL(f, "fopen")
 
    int n = fprintf(f, "Hello %s !\n", "World");
    CHECK_ERROR(n, "fprintf")

    fclose(f);

    f = fopen(TESTFILE, "r");
    CHECK_NULL(f, "fopen")

    char buf[1024];
    char* s = fgets(buf, 1024, f);
    assert(s == buf);
    assert(strlen(buf) == 14);
    assert(strncmp(buf, "Hello World !\n", 14) == 0);

    s = fgets(buf, 1024, f);
    assert(s == NULL); // check EOF is reached

    assert(strncmp(buf, "Hello World !\n", 14) == 0);

    fclose(f);
    unlink(TESTFILE);
}

int main() {
    test_fgets();
}