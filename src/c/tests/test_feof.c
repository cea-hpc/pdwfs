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

int test_feof() {
 
    FILE* f = fopen(TESTFILE, "w");
    CHECK_NULL(f, "fopen")
 
    int n = fprintf(f, "Hello %s !\n", "World");
    CHECK_ERROR(n, "fprintf")

    fclose(f);

    f = fopen(TESTFILE, "r");
    CHECK_NULL(f, "fopen")

    int ret = feof(f);
    assert(ret == 0); // EOF is not reached

    char buf[1024];
    while (fgets(buf, 1024, f) != NULL) {}
    assert(strlen(buf) == 14);
    assert(strncmp(buf, "Hello World !\n", 14) == 0);

    ret = feof(f);
    assert(ret != 0); // EOF is reached

    fclose(f);
    unlink(TESTFILE);

    return 0;
}
