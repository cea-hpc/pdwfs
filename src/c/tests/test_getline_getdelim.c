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
#include <unistd.h>
#include <assert.h>
#include "tests.h"

int test_getline_getdelim() {
 
    int n = 0;

    FILE* f = fopen(TESTFILE, "w");
    CHECK_NULL(f, "fopen")
 
    n = fprintf(f, "Hello World !\n");
    CHECK_ERROR(n, "fprintf")
    n = fprintf(f, "Hello Go !\n");
    CHECK_ERROR(n, "fprintf")
    
    fclose(f);

    f = fopen(TESTFILE, "r");
    CHECK_NULL(f, "fopen")

    char * line = NULL;
    size_t len = 0;
    ssize_t read = 0;

    read = getline(&line, &len, f);
    CHECK_ERROR(read, "getline")
        
    assert(strncmp(line, "Hello World !\n", 14) == 0);

    read = getdelim(&line, &len, '\n', f);
    CHECK_ERROR(read, "getdelim")
    
    assert(strncmp(line, "Hello Go !\n", 11) == 0);

    free(line);
    fclose(f);
    unlink(TESTFILE);

    return 0;
}
