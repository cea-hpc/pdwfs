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

int test_fputc_fgetc() {
 
    FILE* f = fopen(TESTFILE, "w");
    CHECK_NULL(f, "fopen")
 
    int c = fputc('a', f);
    assert(c == 'a');
    c = fputc('\0', f);
    assert(c == '\0');

    fclose(f);

    f = fopen(TESTFILE, "r");
    CHECK_NULL(f, "fopen")

    c = fgetc(f);
    assert(c == 'a');
    c = fgetc(f);
    assert(c == '\0');

    fclose(f);
    unlink(TESTFILE);

    return 0;
}
