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

#include <stdlib.h>
#include <stdarg.h>
#include <dlfcn.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/uio.h>
#include <sys/stat.h>
#include <sys/statfs.h>
#include <sys/statvfs.h>   
#include <errno.h>
#include <fcntl.h>
#include <dirent.h>

#include "libpdwfs_go.h"

static int (*real_open)(const char *pathname, int flags, ...) = NULL;
static int (*real_close)(int fd) = NULL;
static ssize_t (*real_write)(int fd, const void *buf, size_t count) = NULL;
static ssize_t (*real_read)(int fd, void *buf, size_t count) = NULL;
static int (*real_open64)(const char *pathname, int flags, ...) = NULL;
static int (*real_creat)(const char *pathname, mode_t mode) = NULL;
static int (*real_creat64)(const char *pathname, mode_t mode) = NULL;
static int (*real_fdatasync)(int fd) = NULL;
static int (*real_fsync)(int fd) = NULL;
static int (*real_ftruncate64)(int fd, off64_t length) = NULL;
static int (*real_ftruncate)(int fd, off_t length) = NULL;
static int (*real_truncate64)(const char *path, off64_t length) = NULL;
static int (*real_truncate)(const char *path, off_t length) = NULL;
static off64_t (*real_lseek64)(int fd, off64_t offset, int whence) = NULL;
static off_t (*real_lseek)(int fd, off_t offset, int whence) = NULL;
static ssize_t (*real_pread)(int fd, void *buf, size_t count, off_t offset) = NULL;
static ssize_t (*real_pread64)(int fd, void *buf, size_t count, off64_t offset) = NULL;
static ssize_t (*real_preadv)(int fd, const struct iovec *iov, int iovcnt, off_t offset) = NULL;
static ssize_t (*real_preadv64)(int fd, const struct iovec *iov, int iovcnt, off64_t offset) = NULL;
static ssize_t (*real_pwrite)(int fd, const void *buf, size_t count, off_t offset) = NULL;
static ssize_t (*real_pwrite64)(int fd, const void *buf, size_t count, off64_t offset) = NULL;
static ssize_t (*real_pwritev)(int fd, const struct iovec *iov, int iovcnt, off_t offset) = NULL;
static ssize_t (*real_pwritev64)(int fd, const struct iovec *iov, int iovcnt, off64_t offset) = NULL;
static ssize_t (*real_readv)(int fd, const struct iovec *iov, int iovcnt) = NULL;
static ssize_t (*real_writev)(int fd, const struct iovec *iov, int iovcnt) = NULL;
//static int (*real_fcntl)(int fd, int cmd, long arg) = NULL;
static int (*real_ioctl)(int fd, unsigned long request, void *argp) = NULL;

static int (*real_access)(const char *pathname, int mode) = NULL;

static int (*real_unlink)(const char *pathname) = NULL;

static int (*real___xstat)(int vers, const char *pathname, struct stat *buf) = NULL;
static int (*real___xstat64)(int vers, const char *pathname, struct stat64 *buf) = NULL;
static int (*real___lxstat)(int vers, const char *pathname, struct stat *buf) = NULL;
static int (*real___lxstat64)(int vers, const char *pathname, struct stat64 *buf) = NULL;
static int (*real___fxstat)(int vers, int fd, struct stat *buf) = NULL;
static int (*real___fxstat64)(int vers, int fd, struct stat64 *buf) = NULL;


static int (*real_statfs)(const char *path, struct statfs *buf) = NULL;
static int (*real_statfs64)(const char *path, struct statfs64 *buf) = NULL;
static int (*real_fstatfs)(int fd, struct statfs *buf) = NULL;
static int (*real_fstatfs64)(int fd, struct statfs64 *buf) = NULL;

static FILE* (*real_fdopen)(int fd, const char *mode) = NULL;
static FILE* (*real_fopen)(const char *path, const char *mode) = NULL;
static FILE* (*real_fopen64)(const char *path, const char *mode) = NULL;
static FILE* (*real_freopen)(const char *path, const char *mode, FILE *stream) = NULL;
static FILE* (*real_freopen64)(const char *path, const char *mode, FILE *stream) = NULL;
static int (*real_fclose)(FILE *stream) = NULL;
static int (*real_fflush)(FILE *stream) = NULL;
static int (*real_fputc)(int c, FILE *stream) = NULL;
static char* (*real_fgets)(char *s, int size, FILE *stream) = NULL;
static int (*real_fgetc)(FILE *stream) = NULL;
static int (*real_fgetpos)(FILE *stream, fpos_t *pos) = NULL;
static int (*real_fgetpos64)(FILE *stream, fpos64_t *pos) = NULL;
static int (*real_fseek)(FILE *stream, long offset, int whence) = NULL;
static int (*real_fseeko)(FILE *stream, off_t offset, int whence) = NULL;
static int (*real_fseeko64)(FILE *stream, off64_t offset, int whence) = NULL;
static int (*real_fsetpos)(FILE *stream, const fpos_t *pos) = NULL;
static int (*real_fsetpos64)(FILE *stream, const fpos64_t *pos) = NULL;
static int (*real_fputs)(const char *s, FILE *stream) = NULL;
static int (*real_putc)(int c, FILE *stream) = NULL;
static int (*real_getc)(FILE *stream) = NULL;
static int (*real_ungetc)(int c, FILE *stream) = NULL;
static long (*real_ftell)(FILE *stream) = NULL;
static off_t (*real_ftello)(FILE *stream) = NULL;
static off64_t (*real_ftello64)(FILE *stream) = NULL;
static size_t (*real_fread)(void *ptr, size_t size, size_t nmemb, FILE *stream) = NULL;
static size_t (*real_fwrite)(const void *ptr, size_t size, size_t nmemb, FILE *stream) = NULL;
static int (*real_fprintf)(FILE *stream, const char *fmt, ...) = NULL;
static void (*real_rewind)(FILE *stream) = NULL;

static int (*real_dup2)(int oldfd, int newfd) = NULL;

static int (*real_unlinkat)(int dirfd, const char *pathname, int flags) = NULL;
static int (*real_openat)(int dirfd, const char *pathname, int flags, ...) = NULL; 
static int (*real_faccessat)(int dirfd, const char *pathname, int mode, int flags) = NULL;
static int (*real___fxstatat)(int vers, int dirfd, const char *pathname, struct stat *buf, int flags) = NULL;
static int (*real___fxstatat64)(int vers, int dirfd, const char *pathname, struct stat64 *buf, int flags) = NULL;
static int (*real_mkdir)(const char *pathname, mode_t mode) = NULL;
static int (*real_mkdirat)(int dirfd, const char *pathname, mode_t mode) = NULL; 
static int (*real_rmdir)(const char *pathname) = NULL;
static int (*real_rename)(const char *oldpath, const char *newpath) = NULL;
static int (*real_renameat)(int olddirfd, const char *oldpath, int newdirfd, const char *newpath) = NULL;
static int (*real_renameat2)(int olddirfd, const char *oldpath, int newdirfd, const char *newpath, unsigned int flags) = NULL;
static int (*real_posix_fadvise)(int fd, off_t offset, off_t len, int advice) = NULL;
static int (*real_posix_fadvise64)(int fd, off64_t offset, off64_t len, int advice) = NULL;

static int (*real_statvfs)(const char *pathname, struct statvfs *buf) = NULL;
static int (*real_statvfs64)(const char *pathname, struct statvfs64 *buf) = NULL;
static int (*real_fstatvfs)(int fd, struct statvfs *buf) = NULL;
static int (*real_fstatvfs64)(int fd, struct statvfs64 *buf) = NULL;

static ssize_t (*real_getdelim)(char **buf, size_t *bufsiz, int delimiter, FILE *fp) = NULL;
static ssize_t (*real_getline)(char **lineptr, size_t *n, FILE *stream) = NULL; 
static DIR* (*real_opendir)(const char* path) = NULL;

static int (*real_feof)(FILE *stream) = NULL;
static int (*real_ferror)(FILE *stream) = NULL;

static int g_do_trace = -1;

#define MAGENTA "\033[35m"
#define RED "\033[31m"
#define BLUE "\033[34m"
#define YELLOW "\033[33m"
#define GREEN "\033[32m"
#define DEFAULT "\033[39m"

static int pdwfs_fprintf(FILE* stream, const char* color, const char *cat, const char* format, ...) {
	va_list ap;
    dprintf(fileno(stream), "%s[PDWFS][%d][%s]%s[C] ", color, getpid(), cat, DEFAULT);\
	va_start(ap, format);
	int res = vfprintf(stream, format, ap);
	va_end(ap);
	return res;
}

#define TRACE(fmt, ...) {\
    if(g_do_trace < 0) {\
		g_do_trace = (getenv("PDWFS_CTRACES") != NULL);\
	}\
	if(g_do_trace) {\
        pdwfs_fprintf(stderr, BLUE, "TRACE", fmt, ##__VA_ARGS__);\
    }\
}
#define DEBUG(fmt, ...) pdwfs_fprintf(stderr, YELLOW, "DEBUG", fmt, ##__VA_ARGS__);
#define WARNING(fmt, ...) pdwfs_fprintf(stderr, MAGENTA, "WARNING", fmt, ##__VA_ARGS__);


static void raise(const char* format, ...) {
	va_list ap;
    dprintf(fileno(stderr), "%s[PDWFS][ERROR]%s[C] ", RED, DEFAULT);\
	va_start(ap, format);
	vfprintf(stderr, format, ap);
	va_end(ap);
}

#define NOT_IMPLEMENTED(symb) {\
    raise("%s not implemented by pdwfs\n", symb);\
    exit(EXIT_FAILURE);\
}


#define CALL_REAL_OP(symb, func, ...) {\
    if (!func) {\
        char *error;\
        dlerror();\
        func = dlsym(RTLD_NEXT, symb);\
        if ((error = dlerror()) != NULL)  {\
            raise("dlsym: %s\n", error);\
            exit(EXIT_FAILURE);\
        }\
        if (! func ) {\
            raise("symbol not found in dlsym: %s\n", symb);\
            exit(EXIT_FAILURE);\
        }\
    }\
    return func(__VA_ARGS__);\
}

#define IS_STD_FD(fd) (fd == STDIN_FILENO || fd == STDOUT_FILENO || fd == STDERR_FILENO)
#define IS_STD_STREAM(s) (s == stdin || s == stdout || s == stderr)

static int pdwfs_initialized = 0;
// there are cases where pdwfs is not yet initialized and a another library constructor
// (called before pdwfs.so constructor) does some IO (e.g libselinux, libnuma)
// in such case we cannot cross the cgo layer to check if the file/fd is managed by pdwfs, 
// so we defer the call to the real system calls (there's hardly any chance that these IOs 
// calls are the one we intend to intercept anyway)

static __attribute__((constructor)) void init_pdwfs(void) {
    InitPdwfs();
    pdwfs_initialized = 1;
}

static __attribute__((destructor)) void finalize_pdwfs(void) {
    FinalizePdwfs();
}

int open(const char *pathname, int flags, ...) {

    int mode = 0;

    if (__OPEN_NEEDS_MODE (flags)) {
      va_list arg;
      va_start(arg, flags);
      mode = va_arg(arg, int);
      va_end(arg);
    }

    TRACE("intercepting open(pathname=%s, flags=%d, mode=%d)\n", pathname, flags, mode)

    GoString filename = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc open\n");
        CALL_REAL_OP("open", real_open, pathname, flags, mode)
    }

    return Open(filename, flags, mode);
}

int close(int fd) {
    TRACE("intercepting close(fd=%d)\n", fd)

    if (!pdwfs_initialized || IS_STD_FD(fd) || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc close\n");
        CALL_REAL_OP("close", real_close, fd)
    }
    return Close(fd);
}

ssize_t write(int fd, const void *buf, size_t count) {
    TRACE("intercepting write(fd=%d, buf=%p, count=%lu)\n", fd, buf, count)
    
    if (!pdwfs_initialized || IS_STD_FD(fd) || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc write\n");
        CALL_REAL_OP("write", real_write, fd, buf, count)
    }
    GoSlice buffer = {(void*)buf, count, count};
    return Write(fd, buffer);
}

ssize_t read(int fd, void *buf, size_t count) {
    TRACE("intercepting read(fd=%d, buf=%p, count=%lu)\n", fd, buf, count)
    
    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc read\n");
        CALL_REAL_OP("read", real_read, fd, buf, count)
    }
    GoSlice buffer = {buf, count, count};
    return Read(fd, buffer);
}

int open64(const char *pathname, int flags, ...) {

    int mode = 0;

    if (__OPEN_NEEDS_MODE (flags)) {
      va_list arg;
      va_start(arg, flags);
      mode = va_arg(arg, int);
      va_end(arg);
    }

    TRACE("intercepting open64(pathname=%s, flags=%d, mode=%d)\n", pathname, flags, mode)
    
    GoString filename = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc open64\n");
        CALL_REAL_OP("open64", real_open64, pathname, flags, mode)
    }
    return Open(filename, flags, mode);
}

int creat(const char *pathname, mode_t mode) {
    TRACE("intercepting creat(pathname=%s, mode=%d)\n", pathname, mode)
    
    GoString filename = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc creat\n");
        CALL_REAL_OP("creat", real_creat, pathname, mode)
    }
    NOT_IMPLEMENTED("creat")
}

int creat64(const char *pathname, mode_t mode) {
    TRACE("intercepting creat64(pathname=%s, mode=%d)\n", pathname, mode)
    
    GoString filename = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc creat64\n");
        CALL_REAL_OP("creat64", real_creat64, pathname, mode)
    }
    NOT_IMPLEMENTED("creat64")
}

int fdatasync(int fd) {
    TRACE("intercepting fdatasync(fd=%d)\n", fd)
    
    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc fdatasync\n");
        CALL_REAL_OP("fdatasync", real_fdatasync, fd)
    }
    NOT_IMPLEMENTED("fdatasync")
}

int fsync(int fd) {
    TRACE("intercepting fsync(fd=%d)\n", fd)
    
    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc fsync\n");
        CALL_REAL_OP("fsync", real_fsync, fd)
    }
    NOT_IMPLEMENTED("fsync")
}

int ftruncate64(int fd, off64_t length) {
    TRACE("intercepting ftrunctate64(fd=%d, length=%ld)\n", fd, length)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc ftruncate64\n");
        CALL_REAL_OP("ftuncate64", real_ftruncate64, fd, length)
    }
    return Ftruncate(fd, length);
}

int ftruncate(int fd, off_t length) {
    TRACE("callled ftruncate(fd=%d; length=%ld)\n", fd, length)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc ftruncate\n");
        CALL_REAL_OP("ftruncate", real_ftruncate, fd, length)
    }
    return Ftruncate(fd, length);
}

int truncate64(const char *path, off64_t length) {
    TRACE("intercepting truncate64(path=%s, length=%ld)\n", path, length)

    GoString filename = {path, strlen(path)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc truncate64\n");
        CALL_REAL_OP("truncate64", real_truncate64, path, length)
    }
    NOT_IMPLEMENTED("truncate64")
}

int truncate(const char *path, off_t length) {
    TRACE("intercepting truncate(path=%s, length=%ld)\n", path, length)

    GoString filename = {path, strlen(path)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc truncate\n");
        CALL_REAL_OP("truncate", real_truncate, path, length)
    }
    NOT_IMPLEMENTED("truncate")
}

off64_t lseek64(int fd, off64_t offset, int whence) {
    TRACE("intercepting lseek64(fd=%d, offset=%ld; whence=%d)\n", fd, offset, whence)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc lseek64\n");
        CALL_REAL_OP("lseek64", real_lseek64, fd, offset, whence)
    }
    return Lseek(fd, offset, whence);
}

off_t lseek(int fd, off_t offset, int whence) {
    TRACE("intercepting lseek(fd=%d; offset=%ld, whence=%d)\n", fd, offset, whence)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc lseek\n");
        CALL_REAL_OP("lseek", real_lseek, fd, offset, whence)
    }
    return Lseek(fd, offset, whence);
}

ssize_t pread(int fd, void *buf, size_t count, off_t offset) {
    TRACE("intercepting pread(fd=%d, buf=%p, count=%lu, offset=%ld)\n", fd, buf, count, offset)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc pread\n");
        CALL_REAL_OP("pread", real_pread, fd, buf, count, offset)
    }
    GoSlice buffer = {buf, count, count};
    return Pread(fd, buffer, offset);
}

ssize_t pread64(int fd, void *buf, size_t count, off64_t offset) {
    TRACE("intercepting pread64(fd=%d, buf=%p, count=%lu, offset=%ld)\n", fd, buf, count, offset)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc pread64\n");
        CALL_REAL_OP("pread64", real_pread64, fd, buf, count, offset)
    }
    GoSlice buffer = {buf, count, count};
    return Pread(fd, buffer, offset);
}

ssize_t preadv(int fd, const struct iovec *iov, int iovcnt, off_t offset) {
    TRACE("intercepting preadv(fd=%d, iov=%p, iovcnt=%d, offset=%ld)\n", fd, iov, iovcnt, offset)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc preadv\n");
        CALL_REAL_OP("preadv", real_preadv, fd, iov, iovcnt, offset)
    }
    
    GoSlice vec[iovcnt];

    for (int i = 0; i < iovcnt; i++) {
        GoSlice s = {iov[i].iov_base, iov[i].iov_len, iov[i].iov_len};
        vec[i] = s;
    }
    GoSlice iovSlice = {&vec, iovcnt, iovcnt};

    return Preadv(fd, iovSlice, offset);
}

ssize_t preadv64(int fd, const struct iovec *iov, int iovcnt, off64_t offset) {
    TRACE("intercepting preadv64(fd=%d, iov=%p, iovcnt=%d, offset=%ld)\n", fd, iov, iovcnt, offset)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc preadv64\n");
        CALL_REAL_OP("preadv64", real_preadv64, fd, iov, iovcnt, offset)
    }
    
    GoSlice vec[iovcnt];

    for (int i = 0; i < iovcnt; i++) {
        GoSlice s = {iov[i].iov_base, iov[i].iov_len, iov[i].iov_len};
        vec[i] = s;
    }
    GoSlice iovSlice = {&vec, iovcnt, iovcnt};

    return Preadv(fd, iovSlice, offset);
}

ssize_t pwrite(int fd, const void *buf, size_t count, off_t offset) {
    TRACE("intercepting pwrite(fd=%d, buf=%p, count=%lu, offset=%ld)\n", fd, buf, count, offset)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc pwrite\n");
        CALL_REAL_OP("pwrite", real_pwrite, fd, buf, count, offset)
    }
    GoSlice buffer = {(void*)buf, count, count};
    return Pwrite(fd, buffer, offset);
}

ssize_t pwrite64(int fd, const void *buf, size_t count, off64_t offset) {
    TRACE("intercepting pwrite64(fd=%d, buf=%p, count=%lu, offset=%ld)\n", fd, buf, count, offset)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc pwrite64\n");
        CALL_REAL_OP("pwrite64", real_pwrite64, fd, buf, count, offset)
    }
    GoSlice buffer = {(void*)buf, count, count};
    return Pwrite(fd, buffer, offset);
}

ssize_t pwritev(int fd, const struct iovec *iov, int iovcnt, off_t offset) {
    TRACE("intercepting pwritev(fd=%d, iov=%p, iovcnt=%d, offset=%ld)\n", fd, iov, iovcnt, offset)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc pwritev\n");
        CALL_REAL_OP("pwritev", real_pwritev, fd, iov, iovcnt, offset)
    }
    
    GoSlice vec[iovcnt];

    for (int i = 0; i < iovcnt; i++) {
        GoSlice s = {iov[i].iov_base, iov[i].iov_len, iov[i].iov_len};
        vec[i] = s;
    }
    GoSlice iovSlice = {&vec, iovcnt, iovcnt};

    return Pwritev(fd, iovSlice, offset);
}

ssize_t pwritev64(int fd, const struct iovec *iov, int iovcnt, off64_t offset) {
    TRACE("intercepting pwritev64(fd=%d, iov=%p, iovcnt=%d, offset=%ld)\n", fd, iov, iovcnt, offset)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc pwritev64\n");
        CALL_REAL_OP("pwritev64", real_pwritev64, fd, iov, iovcnt, offset)
    }
    
    GoSlice vec[iovcnt];

    for (int i = 0; i < iovcnt; i++) {
        GoSlice s = {iov[i].iov_base, iov[i].iov_len, iov[i].iov_len};
        vec[i] = s;
    }
    GoSlice iovSlice = {&vec, iovcnt, iovcnt};

    return Pwritev(fd, iovSlice, offset);
}

ssize_t readv(int fd, const struct iovec *iov, int iovcnt) {
    TRACE("intercepting readv(fd=%d, iov=%p, iovcnt=%d)\n", fd, iov, iovcnt)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc readv\n");
        CALL_REAL_OP("readv", real_readv, fd, iov, iovcnt)
    }

    GoSlice vec[iovcnt];

    for (int i = 0; i < iovcnt; i++) {
        GoSlice s = {iov[i].iov_base, iov[i].iov_len, iov[i].iov_len};
        vec[i] = s;
    }
    GoSlice iovSlice = {&vec, iovcnt, iovcnt};

    return Readv(fd, iovSlice);
}

ssize_t writev(int fd, const struct iovec *iov, int iovcnt) {
    TRACE("intercepting writev(fd=%d, iov=%p, iovcnt=%d)\n", fd, iov, iovcnt)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc writev\n");
        CALL_REAL_OP("writev", real_writev, fd, iov, iovcnt)
    }

    GoSlice vec[iovcnt];

    for (int i = 0; i < iovcnt; i++) {
        GoSlice s = {iov[i].iov_base, iov[i].iov_len, iov[i].iov_len};
        vec[i] = s;
    }
    GoSlice iovSlice = {&vec, iovcnt, iovcnt};

    return Writev(fd, iovSlice);
}


// we don't intercept fcntl because its signature has variadic args with a too complex 
// behaviour to be reproduced and passed down to the real fcntl
/***
int fcntl(int fd, int cmd, long arg) {
    TRACE("intercepting fcntl(fd=%d, cmd=%d, arg=%ld)\n", fd, cmd, arg)
    
    if (IsFdManaged(fd)) {
        WARNING("fcntl called on a managed fd, fcntl is not intercepted")
    }
    TRACE("calling libc fcntl\n");
    CALL_REAL_OP("fcntl", real_fcntl, fd, cmd, arg)
***/


int ioctl(int fd, unsigned long request, void *argp) {
    TRACE("intercepting ioctl(fd=%d, request=%lu, argp=%p)\n", fd, request, argp)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc ioctl\n");
        CALL_REAL_OP("ioctl", real_ioctl, fd, request, argp)
    }
    NOT_IMPLEMENTED("ioctl")
}


int access(const char *pathname, int mode) {
    TRACE("intercepting access(pathname=%s, mode=%d)\n", pathname, mode)

    GoString filename = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc access\n");
        CALL_REAL_OP("access", real_access, pathname, mode)
    }
    return Access(filename, mode);
}


int unlink(const char *pathname) {
    TRACE("intercepting unlink(pathname=%s)\n", pathname)

    GoString filename = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc unlink\n");
        CALL_REAL_OP("unlink", real_unlink, pathname)
    }
    return Unlink(filename);
}


int __xstat(int vers, const char *pathname, struct stat *buf) {
    TRACE("intercepting __xstat(vers=%d, pathname=%s, buf=%p)\n", vers, pathname, buf)

    GoString filename = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc __xstat\n");
        CALL_REAL_OP("__xstat", real___xstat, vers, pathname, buf)
    }
    return Stat(filename, buf);
}

int __xstat64(int vers, const char *pathname, struct stat64 *buf) {
    TRACE("intercepting __xstat64(vers=%d, pathname=%s, buf=%p)\n", vers, pathname, buf)

    GoString filename = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc __xstat64\n");
        CALL_REAL_OP("__xstat64", real___xstat64, vers, pathname, buf)
    }
    return Stat64(filename, buf);
}

int __lxstat(int vers, const char *pathname, struct stat *buf) {
    TRACE("intercepting __lxstat(vers=%d, pathname=%s, buf=%p)\n", vers, pathname, buf)

    GoString filename = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc __lxstat\n");
        CALL_REAL_OP("__lxstat", real___lxstat, vers, pathname, buf)
    }
    return Lstat(filename, buf);
}

int __lxstat64(int vers, const char *pathname, struct stat64 *buf) {
    TRACE("intercepting __lxstat64(vers=%d, pathname=%s, buf=%p)\n", vers, pathname, buf)

    GoString filename = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc __lxstat64\n");
        CALL_REAL_OP("__lxstat64", real___lxstat64, vers, pathname, buf)
    }
    return Lstat64(filename, buf);
}

int __fxstat(int vers, int fd, struct stat *buf) {
    TRACE("intercepting __fxstat(vers=%d, fd=%d, buf=%p)\n", vers, fd, buf)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc __fxstat\n");
        CALL_REAL_OP("__fxstat", real___fxstat, vers, fd, buf)
    }
    return Fstat(fd, buf);
}

int __fxstat64(int vers, int fd, struct stat64 *buf) {
    TRACE("intercepting __fxstat64(vers=%d, fd=%d, buf=%p)\n", vers, fd, buf)
    
    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc __fxstat64\n");
        CALL_REAL_OP("__fxstat64", real___fxstat64, vers, fd, buf)
    }
    return Fstat64(fd, buf);
}


int statfs(const char *path, struct statfs *buf) {
    TRACE("intercepting statfs(path=%s, buf=%p)\n", path, buf)

    GoString filename = {path, strlen(path)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc statfs\n");
        CALL_REAL_OP("statfs", real_statfs, path,  buf)
    }
    return Statfs(filename, buf);
}

int statfs64(const char *path, struct statfs64 *buf) {
    TRACE("intercepting statfs64(path=%s, buf=%p)\n", path, buf)
    GoString filename = {path, strlen(path)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc statfs64\n");
        CALL_REAL_OP("statfs64", real_statfs64, path,  buf)
    }
    return Statfs64(filename, buf);
}

int fstatfs(int fd, struct statfs *buf) {
    TRACE("intercepting fstatfs(fd=%d, buf=%p)\n", fd, buf)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc fstatfs\n");
        CALL_REAL_OP("fstatfs", real_fstatfs, fd, buf)
    }
    NOT_IMPLEMENTED("fstatfs")
}

int fstatfs64(int fd, struct statfs64 *buf) {
    TRACE("intercepting fstatfs64(fd=%d, buf=%p)\n", fd, buf)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc fstatfs64\n");
        CALL_REAL_OP("fstatfs64", real_fstatfs64, fd, buf)
    }
    NOT_IMPLEMENTED("fstatfs64")
}

FILE* fdopen(int fd, const char *mode) {
    TRACE("intercepting fdopen(fd=%d, mode=%s)\n", fd, mode)

    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc fdopen\n");
        CALL_REAL_OP("fdopen", real_fdopen, fd, mode)
    }
    NOT_IMPLEMENTED("fdopen")
}

FILE* fopen(const char *path, const char *mode) {
    TRACE("intercepting fopen(path=%s, mode=%s)\n", path, mode)

    GoString filename = {path, strlen(path)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc fopen\n");
        CALL_REAL_OP("fopen", real_fopen, path, mode)
    }
    GoString gomode = {mode, strlen(mode)};
    return Fopen(filename, gomode);
}

FILE* fopen64(const char *path, const char *mode) {
    TRACE("intercepting fopen64(path=%s, mode=%s)\n", path, mode)

    GoString filename = {path, strlen(path)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc fopen64\n");
        CALL_REAL_OP("fopen64", real_fopen64, path, mode)
    }
    GoString gomode = {mode, strlen(mode)};
    return Fopen(filename, gomode);
}

FILE* freopen(const char *path, const char *mode, FILE *stream) {
    TRACE("intercepting fropen(path=%s, mode=%s, stream=%p)\n", path, mode, stream)

    GoString filename = {path, strlen(path)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc freopen\n");
        CALL_REAL_OP("freopen", real_freopen, path, mode, stream)
    }
    NOT_IMPLEMENTED("freopen")
}

FILE* freopen64(const char *path, const char *mode, FILE *stream) {
    TRACE("intercepting fropen64(path=%s, mode=%s, stream=%p)\n", path, mode, stream)

    GoString filename = {path, strlen(path)};
    if (!pdwfs_initialized || !IsFileManaged(filename)) {
        TRACE("calling libc freopen64\n");
        CALL_REAL_OP("freopen64", real_freopen64, path, mode, stream)
    }
    NOT_IMPLEMENTED("freopen64")
}

int fclose(FILE *stream) {
    TRACE("intercepting fclose(stream=%p)\n", stream)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fclose\n");
        CALL_REAL_OP("fclose", real_fclose, stream)
    }
    Fflush(stream);
    return close(fileno(stream));
}

int fflush(FILE *stream) {
    TRACE("intercepting fflush(stream=%p)\n", stream)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fflush\n");
        CALL_REAL_OP("fflush", real_fflush, stream)
    }
    return Fflush(stream);
}

int fputc(int c, FILE *stream) {
    TRACE("intercepting fputc(c=%d, stream=%p)\n", c, stream)
    
    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fputc\n");
        CALL_REAL_OP("fputc", real_fputc, c, stream)
    }
    GoSlice cBuf = {(char*)&c, 1, 1};
    int n = Write(fileno(stream), cBuf); 
	if (n <= 0)
		return EOF;
	return c;
}

char* fgets(char *dst, int max, FILE *stream) {
    TRACE("intercepting fgets(dst=%s, max=%d, stream=%p)\n", dst, max, stream)
    
    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fgets\n");
        CALL_REAL_OP("fgets", real_fgets, dst, max, stream)
    }
    
    size_t n = max;
    ssize_t count;
    char *result;
    if (n <= 0)
        return NULL;
    if (n == 1) {
        /* irregular case: since we have to store a NUL byte and
         there is only room for exactly one byte, we don't have to
         read anything.  */
        dst[0] = '\0';
        return dst;
    }
    n--;
    count = getline(&dst, &n, stream);
    if (count <= 0)
        result = NULL;
    else {
        dst[count] = '\0';
        result = dst;
    }
    return result;
}


int fgetc(FILE *stream) {
    TRACE("intercepting fgetc(stream=%p)\n", stream)
    
    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fgetc\n");
        CALL_REAL_OP("fgetc", real_fgetc, stream)
    }
    
	char c;
    GoSlice cBuf = {&c, 1, 1};
    int n = Read(fileno(stream), cBuf); 
	if (n == 0)
		return EOF;
    if (n < 0)
        return n;
	return c;
}

int fgetpos(FILE *stream, fpos_t *pos) {
    TRACE("intercepting fgetpos(stream=%p, pos=%p)\n", stream, pos)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fgetpos\n");
        CALL_REAL_OP("fgetpos", real_fgetpos, stream, pos)
    }
    NOT_IMPLEMENTED("fgetpos")
}

int fgetpos64(FILE *stream, fpos64_t *pos) {
    TRACE("intercepting fgetpos64(stream=%p, pos=%p)\n", stream, pos)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fgetpos64\n");
        CALL_REAL_OP("fgetpos64", real_fgetpos64, stream, pos)
    }
    NOT_IMPLEMENTED("fgetpos64")
}

int fseek(FILE *stream, long offset, int whence) {
    TRACE("intercepting fseek(stream=%p, offset=%ld, whence=%d)\n", stream, offset, whence)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fseek\n");
        CALL_REAL_OP("fseek", real_fseek, stream, offset, whence)
    }
    NOT_IMPLEMENTED("fseek")
}

int fseeko(FILE *stream, off_t offset, int whence) {
    TRACE("intercepting fseeko(stream=%p, offset=%ld, whence=%d)\n", stream, offset, whence)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fseeko\n");
        CALL_REAL_OP("fseeko", real_fseeko, stream, offset, whence)
    }
    NOT_IMPLEMENTED("fseeko")
}

int fseeko64(FILE *stream, off64_t offset, int whence) {
    TRACE("intercepting fseeko64(stream=%p, offset=%ld, whence=%d)\n", stream, offset, whence)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fseeko64\n");
        CALL_REAL_OP("fseeko64", real_fseeko64, stream, offset, whence)
    }
    NOT_IMPLEMENTED("fseeko")
}

int fsetpos(FILE *stream, const fpos_t *pos) {
    TRACE("intercepting fsetpos(stream=%p, pos=%p)\n", stream, pos)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fsetpos\n");
        CALL_REAL_OP("fsetpos", real_fsetpos, stream, pos)
    }
    NOT_IMPLEMENTED("fsetpos")
}

int fsetpos64(FILE *stream, const fpos64_t *pos) {
    TRACE("intercepting fsetpos64(stream=%p, pos=%p)\n", stream, pos)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fsetpos64\n");
        CALL_REAL_OP("fsetpos64", real_fsetpos64, stream, pos)
    }
    NOT_IMPLEMENTED("fsetpos64")
}

int fputs(const char *s, FILE *stream) {
    TRACE("intercepting fputs(s=%s, stream=%p)\n", s, stream)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fputs\n");
        CALL_REAL_OP("fputs", real_fputs, s, stream)
    }
    NOT_IMPLEMENTED("fputs")
}

int putc(int c, FILE *stream) {
    TRACE("intercepting putc(c=%d, stream=%p)\n", c, stream)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc putc\n");
        CALL_REAL_OP("putc", real_putc, c, stream)
    }
    NOT_IMPLEMENTED("putc")
}

int getc(FILE *stream) {
    TRACE("intercepting getc(stream=%p)\n", stream)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc getc\n");
        CALL_REAL_OP("getc", real_getc, stream)
    }
    NOT_IMPLEMENTED("getc")
}

int ungetc(int c, FILE *stream) {
    TRACE("intercepting fgetc(c=%d, stream=%p)\n", c, stream)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc ungetc\n");
        CALL_REAL_OP("ungetc", real_ungetc, c, stream)
    }
    NOT_IMPLEMENTED("ungetc")
}

long ftell(FILE *stream) {
    TRACE("intercepting ftell(stream=%p)\n", stream)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc ftell\n");
        CALL_REAL_OP("ftell", real_ftell, stream)
    }
    NOT_IMPLEMENTED("ftell")
}

off_t ftello(FILE *stream) {
    TRACE("intercepting ftello(stream=%p)\n", stream)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc ftello\n");
        CALL_REAL_OP("ftello", real_ftello, stream)
    }
    NOT_IMPLEMENTED("ftello")
}

off64_t ftello64(FILE *stream) {
    TRACE("intercepting ftello64(stream=%p)\n", stream)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc ftello64\n");
        CALL_REAL_OP("ftello64", real_ftello64, stream)
    }
    NOT_IMPLEMENTED("ftello64")
}

size_t fread(void *ptr, size_t size, size_t nmemb, FILE *stream) {
    TRACE("intercepting fread(ptr=%p, size=%lu, nmemb=%lu, stream=%p)\n", ptr, size, nmemb, stream)
    
    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fread\n");
        CALL_REAL_OP("fread", real_fread, ptr, size, nmemb, stream)
    }
    GoSlice buffer = {ptr, size * nmemb, size * nmemb};
    return Read(fileno(stream), buffer);
}

size_t fwrite(const void *ptr, size_t size, size_t nmemb, FILE *stream) {
    TRACE("intercepting fwrite(ptr=%p, size=%lu, nmemb=%lu, stream=%p)\n", ptr, size, nmemb, stream)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fwrite\n");
        CALL_REAL_OP("fwrite", real_fwrite, ptr, size, nmemb, stream)
    }
    GoSlice buffer = {(void*)ptr, size * nmemb, size * nmemb};
    return Write(fileno(stream), buffer);
}

int _fprint(FILE *stream, const char *msg, int size) {
    
    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc fprintf\n");
        if (!real_fprintf) {
            real_fprintf = dlsym(RTLD_NEXT, "fprintf");
        }
        return real_fprintf(stream, "%s", msg);
    }
    GoSlice buffer = {(char*)msg, size, size};
    return Write(fileno(stream), buffer);
}

int __fprintf_chk(FILE *stream, int flag, const char *fmt, ...) {
    TRACE("intercepting __fprintf_chk(stream=%p, ...)\n", stream)

    //make the message
    int size = 0;
    char *msg = NULL;
    va_list ap;

    // Determine the required size
    va_start(ap, fmt);
    size = vsnprintf(msg, size, fmt, ap);
    va_end(ap);

    if (size < 0) return -1;

    size++;   // For '\0'
    msg = malloc(size);
    if (msg == NULL) return -1;

    va_start(ap, fmt);
    size = vsnprintf(msg, size, fmt, ap);
    if (size < 0) {
        free(msg);
        return -1;
    }
    va_end(ap);

    int ret = _fprint(stream, msg, size); 
    free(msg);
    return ret;
}

int fprintf(FILE *stream, const char *fmt, ...) {
    TRACE("intercepting fprintf(stream=%p, ...)\n", stream)

    //make the message
    int size = 0;
    char *msg = NULL;
    va_list ap;

    // Determine the required size
    va_start(ap, fmt);
    size = vsnprintf(msg, size, fmt, ap);
    va_end(ap);

    if (size < 0) return -1;

    size++;   // For '\0'
    msg = malloc(size);
    if (msg == NULL) return -1;

    va_start(ap, fmt);
    size = vsnprintf(msg, size, fmt, ap);
    if (size < 0) {
        free(msg);
        return -1;
    }
    va_end(ap);

    int ret = _fprint(stream, msg, size); 
    free(msg);
    return ret;
}


void rewind(FILE *stream) {
    TRACE("intercepting rewind(stream=%p)\n", stream)

    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc rewind\n");
        CALL_REAL_OP("rewind", real_rewind, stream)
    }
    NOT_IMPLEMENTED("rewind")
}

int dup2(int oldfd, int newfd) {
    TRACE("intercepting dup2(oldfd=%d, newfd=%d)\n", oldfd, newfd)

    if (!pdwfs_initialized || (!IsFdManaged(oldfd) && !IsFdManaged(newfd))) {
    TRACE("calling libc dup2\n");
    CALL_REAL_OP("dup2", real_dup2, oldfd, newfd)
}
    NOT_IMPLEMENTED("dup2")
}

int unlinkat(int dirfd, const char *pathname, int flags) {
    TRACE("intercepting unlinkat(dirfd=%d, pathname=%s, flags=%d)\n", dirfd, pathname, flags)
    TRACE("calling libc unlinkat (INTERCEPTION NOT IMPLEMENTED)\n")
    CALL_REAL_OP("unlinkat", real_unlinkat, dirfd, pathname, flags)
    }

int faccessat(int dirfd, const char *pathname, int mode, int flags) {
    TRACE("intercepting faccessat(dirfd=%d, pathname=%s, mode=%d, flags=%d)\n", dirfd, pathname, mode, flags)
    TRACE("calling libc faccessat (INTERCEPTION NOT IMPLEMENTED)\n")
    CALL_REAL_OP("faccessat", real_faccessat, dirfd, pathname, mode, flags)
    }

// __fxstatat is the glibc function corresponding to fstatat syscall
int __fxstatat(int vers, int dirfd, const char *pathname, struct stat *buf, int flags) {
    TRACE("intercepting __fxstatat(vers=%d, dirfd=%d, pathname=%s, buf=%p, flags=%d)\n", vers, dirfd, pathname, buf, flags)
    TRACE("calling libc __fxstatat (INTERCEPTION NOT IMPLEMENTED)\n")
    CALL_REAL_OP("__fxstatat", real___fxstatat, vers, dirfd, pathname, buf, flags)
}

// __fxstatat64 is the LARGEFILE64 version of glibc function corresponding to fstatat syscall
int __fxstatat64(int vers, int dirfd, const char *pathname, struct stat64 *buf, int flags) {
    TRACE("intercepting __fxstatat64(vers=%d, dirfd=%d, pathname=%s, buf=%p, flags=%d)\n", dirfd, pathname, buf, flags)
    TRACE("calling libc __fxstatat64 (INTERCEPTION NOT IMPLEMENTED)\n")
    CALL_REAL_OP("__fxstatat64", real___fxstatat64, vers, dirfd, pathname, buf, flags)
}

int openat(int dirfd, const char *pathname, int flags, ...) {
    
    int mode = 0;

    if (__OPEN_NEEDS_MODE (flags)) {
        va_list arg;
        va_start(arg, flags);
        mode = va_arg(arg, int);
        va_end(arg);
        }

    TRACE("intercepting openat(dirfd=%d, pathname=%s, flags=%d, mode=%d)\n", dirfd, pathname, flags, mode)
    // NOTE: see the comment of func (fs *PdwFS) registerFile in pdwfs.go before implementing openat interception
    TRACE("calling libc openat (INTERCEPTION NOT IMPLEMENTED)\n")
    CALL_REAL_OP("openat", real_openat, dirfd, pathname, flags, mode)
}

int mkdir(const char *pathname, mode_t mode) {
    TRACE("intercepting mkdir(pathname=%s, mode=%d)\n", pathname, mode)
    
    GoString gopath = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(gopath)) {
        TRACE("calling libc mkdir\n");
        CALL_REAL_OP("mkdir", real_mkdir, pathname, mode)
    }
    return Mkdir(gopath, mode);
}

int mkdirat(int dirfd, const char *pathname, mode_t mode) {
    TRACE("intercepting mkdirat(dirfd=%d, pathname=%s, mode=%d)\n", dirfd, pathname, mode)
    TRACE("calling libc mkdirat (INTERCEPTION NOT IMPLEMENTED)\n")
    CALL_REAL_OP("mkdirat", real_mkdirat, dirfd, pathname, mode)
}

int rmdir(const char *pathname) {
    TRACE("intercepting rmdir(pathname=%s)\n", pathname)
    
    GoString gopath = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(gopath)) {
        TRACE("calling libc rmdir\n");
        CALL_REAL_OP("rmdir", real_rmdir, pathname)
    }
    return Rmdir(gopath);
}

int rename(const char *oldpath, const char *newpath) {
    TRACE("intercepting rename(oldname=%s, newpath=%s)\n", oldpath, newpath)
    
    GoString old = {oldpath, strlen(oldpath)};
    GoString new = {newpath, strlen(newpath)};
    if (!pdwfs_initialized || (!IsFileManaged(old) && !IsFileManaged(new))) {
        TRACE("calling libc rename\n");
        CALL_REAL_OP("rename", real_rename, oldpath, newpath)
    }
    NOT_IMPLEMENTED("rename")
}

int renameat(int olddirfd, const char *oldpath, int newdirfd, const char *newpath) {
    TRACE("intercepting renameat(olddirfd=%d, oldpath=%s, newdirfd=%d, newpath=%s)\n", olddirfd, oldpath, newdirfd, newpath)
    TRACE("calling libc renameat (INTERCEPTION NOT IMPLEMENTED)\n")
    CALL_REAL_OP("renameat", real_renameat, olddirfd, oldpath, newdirfd, newpath)
}

int renameat2(int olddirfd, const char *oldpath, int newdirfd, const char *newpath, unsigned int flags) {
    TRACE("intercepting renameat2(olddirfd=%d, oldpath=%s, newdirfd=%d, newpath=%s, flags=%d)\n", olddirfd, oldpath, newdirfd, newpath, flags)
    TRACE("calling libc renameat2 (INTERCEPTION NOT IMPLEMENTED)\n")
    CALL_REAL_OP("renameat2", real_renameat2, olddirfd, oldpath, newdirfd, newpath, flags)
}

int posix_fadvise(int fd, off_t offset, off_t len, int advice) {
    TRACE("intercepting posix_fadvise(fd=%d, offset=%d, len=%d, advice=%d)\n", fd, offset, len, advice)
 
    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc posix_fadvise\n");
        CALL_REAL_OP("posix_fadvise", real_posix_fadvise, fd, offset, len, advice)
    }
    return Fadvise(fd, offset, len, advice);
}

int posix_fadvise64(int fd, off64_t offset, off64_t len, int advice) {
    TRACE("intercepting posix_fadvise64(fd=%d, offset=%d, len=%d, advice=%d)\n", fd, offset, len, advice)
    
    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc posix_fadvise64\n");
        CALL_REAL_OP("posix_fadvise64", real_posix_fadvise64, fd, offset, len, advice)
    }
    return Fadvise(fd, offset, len, advice);
}

int statvfs(const char *pathname, struct statvfs *buf) {
    TRACE("intercepting statvfs(path=%s, buf=%p)\n", pathname, buf)
    
    GoString gopath = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(gopath)) {
        TRACE("calling libc statvfs\n");
        CALL_REAL_OP("statvfs", real_statvfs, pathname, buf)
    }
    return Statvfs(gopath, buf);
}

int statvfs64(const char *pathname, struct statvfs64 *buf) {
    TRACE("intercepting statvfs64(path=%s, buf=%p)\n", pathname, buf)
    
    GoString gopath = {pathname, strlen(pathname)};
    if (!pdwfs_initialized || !IsFileManaged(gopath)) {
        TRACE("calling libc statvfs64\n");
        CALL_REAL_OP("statvfs64", real_statvfs64, pathname, buf)
    }
    return Statvfs64(gopath, buf);
}

int fstatvfs(int fd, struct statvfs *buf) {
    TRACE("intercepting fstatvfs(fd=%d, buf=%p)\n", fd, buf)
    
    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc fstatvfs\n");
        CALL_REAL_OP("fstatvfs", real_fstatvfs, fd, buf)
    }
    NOT_IMPLEMENTED("fstatvfs")
}

int fstatvfs64(int fd, struct statvfs64 *buf) {
    TRACE("intercepting fstatvfs64(fd=%d, buf=%p)\n", fd, buf)
    
    if (!pdwfs_initialized || IS_STD_FD(fd) || !IsFdManaged(fd)) {
        TRACE("calling libc fstatvfs64\n");
        CALL_REAL_OP("fstatvfs64", real_fstatvfs64, fd, buf)
    }
    NOT_IMPLEMENTED("fstatvfs64")
}


ssize_t getdelim(char **buf, size_t *bufsiz, int delimiter, FILE *fp) {
    TRACE("intercepting getdelim(buf=%p, bufsiz=%p, delimiter=%d, stream=%p)\n", buf, bufsiz, delimiter, fp)
    
    if (!pdwfs_initialized || IS_STD_STREAM(fp) || !IsFdManaged(fileno(fp))) {
        TRACE("calling libc getdelim\n");
        CALL_REAL_OP("getdelim", real_getdelim, buf, bufsiz, delimiter, fp)
    }

    char *ptr, *eptr;


	if (*buf == NULL || *bufsiz == 0) {
		*bufsiz = BUFSIZ;
		if ((*buf = malloc(*bufsiz)) == NULL)
			return -1;
	}

	for (ptr = *buf, eptr = *buf + *bufsiz;;) {
		int c = fgetc(fp);
		if (c == -1) {
			if (feof(fp))
                return ptr == *buf ? -1 : ptr - *buf;
            else
				return -1;
		}
		*ptr++ = c;
		if (c == delimiter) {
			*ptr = '\0';
			return ptr - *buf;
		}
		if (ptr + 2 >= eptr) {
			char *nbuf;
			size_t nbufsiz = *bufsiz * 2;
			ssize_t d = ptr - *buf;
			if ((nbuf = realloc(*buf, nbufsiz)) == NULL)
				return -1;
			*buf = nbuf;
			*bufsiz = nbufsiz;
			eptr = nbuf + nbufsiz;
			ptr = nbuf + d;
		}
    }
}


ssize_t getline(char **buf, size_t *bufsiz, FILE *stream) {
    TRACE("intercepting getline(buf=%p, bufsiz=%p, stream=%p)\n", buf, bufsiz, stream)
    
    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc getline\n");
        CALL_REAL_OP("getline", real_getline, buf, bufsiz, stream)
    }
    return getdelim(buf, bufsiz, '\n', stream);
}

DIR* opendir(const char* path) {
    TRACE("intercepting opendir(path=%s)\n", path)
    
    GoString gopath = {path, strlen(path)};
    if (!pdwfs_initialized || !IsFileManaged(gopath)) {
        TRACE("calling libc opendir\n");
        CALL_REAL_OP("opendir", real_opendir, path)
    }
    NOT_IMPLEMENTED("opendir")
}

int feof(FILE *stream) {
    TRACE("intercepting feof(stream=%p)\n", stream)
    
    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc feof\n");
        CALL_REAL_OP("feof", real_feof, stream)
    }
    int fd = fileno(stream);
    off_t cur_off = lseek(fd, 0, SEEK_CUR);
    if (cur_off == lseek(fd, 0, SEEK_END))
        return 1;
    lseek(fd, cur_off, SEEK_SET);
    return 0;
}

int ferror(FILE *stream) {
    TRACE("intercepting ferror(stream=%p)\n", stream)
    
    if (!pdwfs_initialized || IS_STD_STREAM(stream) || !IsFdManaged(fileno(stream))) {
        TRACE("calling libc ferror\n");
        CALL_REAL_OP("ferror", real_ferror, stream)
    }
    NOT_IMPLEMENTED("ferror")
}