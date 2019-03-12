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

#include <unistd.h>
#include <assert.h>
#include "tests.h"

void test_fopen() {
    
    FILE *f = NULL;
    
    f = fopen(TESTFILE, "r");
    assert(f == NULL);

    f = fopen(TESTFILE, "w");
    CHECK_NULL(f, "fopen")

    int ret = fclose(f);
    CHECK_ERROR(ret, "fclose")

    ret = unlink(TESTFILE);
    CHECK_ERROR(ret, "unlink")
}

int main() {
    test_fopen();
}