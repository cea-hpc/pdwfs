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

int test_ferror() {
 
    // Testing ferror on failing fgetc call
    FILE *f = fopen(TESTFILE, "w");
    CHECK_NULL(f, "fopen")
 
    int c = fputc('A', f);
    assert(c == 'A');

    int ret = ferror(f);
    assert(ret == 0);

    c = fgetc(f);
    ret = ferror(f);
    assert(ret != 0);

    clearerr(f);
    ret = ferror(f);
    assert(ret == 0);

    fclose(f);

    // Testing ferror on failing fread call
    f = fopen(TESTFILE, "w");
    CHECK_NULL(f, "fopen")

    size_t n = fwrite("Hello World !\n", 1, 14, f);
    CHECK_ERROR(n, "fwrite")

    ret = fflush(f);
    CHECK_ERROR(ret, "fflush")

    char buf[14];
    n = fread(&buf, 1, 14, f);

    ret = ferror(f);
    assert(ret != 0);

    fclose(f);

    // Testing ferror on failing fputc and fwrite calls
    f = fopen(TESTFILE, "r");
    CHECK_NULL(f, "fopen")

    fputc('A', f);
    
    ret = ferror(f);
    assert(ret != 0);

    clearerr(f);
    n = fwrite("Hello World !\n", 1, 14, f);
    CHECK_ERROR(n, "fwrite")
    
    ret = ferror(f);
    assert(ret != 0);

    unlink(TESTFILE);

    return 0;
}
