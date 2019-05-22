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

// from libc-testsuite

#include <stdio.h>

#define RUN_TEST(a) { \
extern int test_ ##a (void); \
int e = test_ ##a (); \
if (e) printf("%s test failed, %d error(s)\n", #a, e); \
else   printf("%s test passed\n", #a); \
err += e; \
}

int main()
{
	int err=0;

	RUN_TEST(access);
	RUN_TEST(feof);
	RUN_TEST(fgets);
	RUN_TEST(fopen_fclose);
	RUN_TEST(fprintf);
	RUN_TEST(fputc_fgetc);
	RUN_TEST(ftruncate);
	RUN_TEST(fwrite_fread);
	RUN_TEST(lseek);
	RUN_TEST(mkdir_rmdir);
	RUN_TEST(open_close);
	RUN_TEST(pread);
	RUN_TEST(preadv);
	RUN_TEST(pwrite);
	RUN_TEST(pwritev);
	RUN_TEST(readv);
	RUN_TEST(stat);
    RUN_TEST(stat_size);
	RUN_TEST(statfs);
	RUN_TEST(unlink);
	RUN_TEST(write_read);
	RUN_TEST(writev);

	printf("\ntotal errors: %d\n", err);
	return !!err;
}
