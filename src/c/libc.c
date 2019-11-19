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

#include <stdarg.h>
#include <dlfcn.h>
#include <unistd.h>
#include <sys/uio.h>
#include <sys/stat.h>
#include <sys/statfs.h>
#include <sys/statvfs.h>   
#include "libc.h"

static int (*ptr_open)(const char *pathname, int flags, ...) = NULL;
static int (*ptr_close)(int fd) = NULL;
static ssize_t (*ptr_write)(int fd, const void *buf, size_t count) = NULL;
static ssize_t (*ptr_read)(int fd, void *buf, size_t count) = NULL;
static int (*ptr_open64)(const char *pathname, int flags, ...) = NULL;
static int (*ptr_creat)(const char *pathname, mode_t mode) = NULL;
static int (*ptr_creat64)(const char *pathname, mode_t mode) = NULL;
static int (*ptr_fdatasync)(int fd) = NULL;
static int (*ptr_fsync)(int fd) = NULL;
static int (*ptr_ftruncate64)(int fd, off64_t length) = NULL;
static int (*ptr_ftruncate)(int fd, off_t length) = NULL;
static int (*ptr_truncate64)(const char *path, off64_t length) = NULL;
static int (*ptr_truncate)(const char *path, off_t length) = NULL;
static off64_t (*ptr_lseek64)(int fd, off64_t offset, int whence) = NULL;
static off_t (*ptr_lseek)(int fd, off_t offset, int whence) = NULL;
static ssize_t (*ptr_pread)(int fd, void *buf, size_t count, off_t offset) = NULL;
static ssize_t (*ptr_pread64)(int fd, void *buf, size_t count, off64_t offset) = NULL;
static ssize_t (*ptr_preadv)(int fd, const struct iovec *iov, int iovcnt, off_t offset) = NULL;
static ssize_t (*ptr_preadv64)(int fd, const struct iovec *iov, int iovcnt, off64_t offset) = NULL;
static ssize_t (*ptr_pwrite)(int fd, const void *buf, size_t count, off_t offset) = NULL;
static ssize_t (*ptr_pwrite64)(int fd, const void *buf, size_t count, off64_t offset) = NULL;
static ssize_t (*ptr_pwritev)(int fd, const struct iovec *iov, int iovcnt, off_t offset) = NULL;
static ssize_t (*ptr_pwritev64)(int fd, const struct iovec *iov, int iovcnt, off64_t offset) = NULL;
static ssize_t (*ptr_readv)(int fd, const struct iovec *iov, int iovcnt) = NULL;
static ssize_t (*ptr_writev)(int fd, const struct iovec *iov, int iovcnt) = NULL;
static int (*ptr_ioctl)(int fd, unsigned long request, void *argp) = NULL;
static int (*ptr_access)(const char *pathname, int mode) = NULL;
static int (*ptr_unlink)(const char *pathname) = NULL;
static int (*ptr___xstat)(int vers, const char *pathname, struct stat *buf) = NULL;
static int (*ptr___xstat64)(int vers, const char *pathname, struct stat64 *buf) = NULL;
static int (*ptr___lxstat)(int vers, const char *pathname, struct stat *buf) = NULL;
static int (*ptr___lxstat64)(int vers, const char *pathname, struct stat64 *buf) = NULL;
static int (*ptr___fxstat)(int vers, int fd, struct stat *buf) = NULL;
static int (*ptr___fxstat64)(int vers, int fd, struct stat64 *buf) = NULL;
static int (*ptr_statfs)(const char *path, struct statfs *buf) = NULL;
static int (*ptr_statfs64)(const char *path, struct statfs64 *buf) = NULL;
static int (*ptr_fstatfs)(int fd, struct statfs *buf) = NULL;
static int (*ptr_fstatfs64)(int fd, struct statfs64 *buf) = NULL;
static FILE* (*ptr_fdopen)(int fd, const char *mode) = NULL;
static FILE* (*ptr_fopen)(const char *path, const char *mode) = NULL;
static FILE* (*ptr_fopen64)(const char *path, const char *mode) = NULL;
static FILE* (*ptr_freopen)(const char *path, const char *mode, FILE *stream) = NULL;
static FILE* (*ptr_freopen64)(const char *path, const char *mode, FILE *stream) = NULL;
static int (*ptr_fclose)(FILE *stream) = NULL;
static int (*ptr_fflush)(FILE *stream) = NULL;
static int (*ptr_fputc)(int c, FILE *stream) = NULL;
static char* (*ptr_fgets)(char *s, int size, FILE *stream) = NULL;
static int (*ptr_fgetc)(FILE *stream) = NULL;
static int (*ptr_fgetpos)(FILE *stream, fpos_t *pos) = NULL;
static int (*ptr_fgetpos64)(FILE *stream, fpos64_t *pos) = NULL;
static int (*ptr_fseek)(FILE *stream, long offset, int whence) = NULL;
static int (*ptr_fseeko)(FILE *stream, off_t offset, int whence) = NULL;
static int (*ptr_fseeko64)(FILE *stream, off64_t offset, int whence) = NULL;
static int (*ptr_fsetpos)(FILE *stream, const fpos_t *pos) = NULL;
static int (*ptr_fsetpos64)(FILE *stream, const fpos64_t *pos) = NULL;
static int (*ptr_fputs)(const char *s, FILE *stream) = NULL;
static int (*ptr_putc)(int c, FILE *stream) = NULL;
static int (*ptr_getc)(FILE *stream) = NULL;
static int (*ptr_ungetc)(int c, FILE *stream) = NULL;
static long (*ptr_ftell)(FILE *stream) = NULL;
static off_t (*ptr_ftello)(FILE *stream) = NULL;
static off64_t (*ptr_ftello64)(FILE *stream) = NULL;
static size_t (*ptr_fread)(void *ptr, size_t size, size_t nmemb, FILE *stream) = NULL;
static size_t (*ptr_fwrite)(const void *ptr, size_t size, size_t nmemb, FILE *stream) = NULL;
static void (*ptr_rewind)(FILE *stream) = NULL;
static int (*ptr_dup2)(int oldfd, int newfd) = NULL;
static int (*ptr_unlinkat)(int dirfd, const char *pathname, int flags) = NULL;
static int (*ptr_openat)(int dirfd, const char *pathname, int flags, ...) = NULL; 
static int (*ptr_faccessat)(int dirfd, const char *pathname, int mode, int flags) = NULL;
static int (*ptr___fxstatat)(int vers, int dirfd, const char *pathname, struct stat *buf, int flags) = NULL;
static int (*ptr___fxstatat64)(int vers, int dirfd, const char *pathname, struct stat64 *buf, int flags) = NULL;
static int (*ptr_mkdir)(const char *pathname, mode_t mode) = NULL;
static int (*ptr_mkdirat)(int dirfd, const char *pathname, mode_t mode) = NULL; 
static int (*ptr_rmdir)(const char *pathname) = NULL;
static int (*ptr_rename)(const char *oldpath, const char *newpath) = NULL;
static int (*ptr_renameat)(int olddirfd, const char *oldpath, int newdirfd, const char *newpath) = NULL;
static int (*ptr_renameat2)(int olddirfd, const char *oldpath, int newdirfd, const char *newpath, unsigned int flags) = NULL;
static int (*ptr_posix_fadvise)(int fd, off_t offset, off_t len, int advice) = NULL;
static int (*ptr_posix_fadvise64)(int fd, off64_t offset, off64_t len, int advice) = NULL;
static int (*ptr_statvfs)(const char *pathname, struct statvfs *buf) = NULL;
static int (*ptr_statvfs64)(const char *pathname, struct statvfs64 *buf) = NULL;
static int (*ptr_fstatvfs)(int fd, struct statvfs *buf) = NULL;
static int (*ptr_fstatvfs64)(int fd, struct statvfs64 *buf) = NULL;
static ssize_t (*ptr___getdelim)(char **buf, size_t *bufsiz, int delimiter, FILE *fp) = NULL;
static ssize_t (*ptr_getline)(char **lineptr, size_t *n, FILE *stream) = NULL; 
static DIR* (*ptr_opendir)(const char* path) = NULL;
static int (*ptr_feof)(FILE *stream) = NULL;
static int (*ptr_ferror)(FILE *stream) = NULL;
static ssize_t (*ptr_getxattr)(const char *path, const char *name, void *value,  size_t size) = NULL;


static int g_do_trace = -1;

#define RED "\033[31m"
#define BLUE "\033[34m"
#define DEFAULT "\033[39m"

static int pdwfs_fprintf(FILE* stream, const char* color, const char *cat, const char* format, ...) {
	va_list ap;
    dprintf(fileno(stream), "%s[PDWFS][%d][%s]%s[C] ", color, getpid(), cat, DEFAULT);\
	va_start(ap, format);
	int res = vfprintf(stream, format, ap);
	va_end(ap);
	return res;
}

#define CALL_NEXT(symb, ...) {\
    if(g_do_trace < 0) {\
		g_do_trace = (getenv("PDWFS_CTRACES") != NULL);\
	}\
	if(g_do_trace) {\
        pdwfs_fprintf(stderr, BLUE, "TRACE", "calling libc %s\n", #symb);\
    }\
    if (! ptr_ ## symb) {\
        char *error;\
        dlerror();\
        ptr_ ## symb = dlsym(RTLD_NEXT, #symb);\
        if ((error = dlerror()) != NULL)  {\
            pdwfs_fprintf(stderr, RED, "ERROR", "dlsym: %s\n", error);\
            exit(EXIT_FAILURE);\
        }\
        if (! ptr_ ## symb ) {\
            pdwfs_fprintf(stderr, RED, "ERROR", "symbol not found in dlsym: %s\n", #symb);\
            exit(EXIT_FAILURE);\
        }\
    }\
    return ptr_ ## symb(__VA_ARGS__);\
}

int libc_open(const char *pathname, int flags, int mode) {
    CALL_NEXT(open, pathname, flags, mode)
}

int libc_close(int fd) {
    CALL_NEXT(close, fd)
}

ssize_t libc_write(int fd, const void *buf, size_t count) {
    CALL_NEXT(write, fd, buf, count)
}

ssize_t libc_read(int fd, void *buf, size_t count) {
    CALL_NEXT(read, fd, buf, count)
}

int libc_open64(const char *pathname, int flags, int mode) {
    CALL_NEXT(open64, pathname, flags, mode)
}

int libc_creat(const char *pathname, mode_t mode) {
    CALL_NEXT(creat, pathname, mode)
}

int libc_creat64(const char *pathname, mode_t mode) {
    CALL_NEXT(creat64, pathname, mode)
}

int libc_fdatasync(int fd) {
    CALL_NEXT(fdatasync, fd)
}

int libc_fsync(int fd) {
    CALL_NEXT(fsync, fd)
}

int libc_ftruncate64(int fd, off64_t length) {
    CALL_NEXT(ftruncate64, fd, length)
}

int libc_ftruncate(int fd, off_t length) {
    CALL_NEXT(ftruncate, fd, length)
}

int libc_truncate64(const char *path, off64_t length) {
    CALL_NEXT(truncate64, path, length)
}

int libc_truncate(const char *path, off_t length) {
    CALL_NEXT(truncate, path, length)
}

off64_t libc_lseek64(int fd, off64_t offset, int whence) {
    CALL_NEXT(lseek64, fd, offset, whence)
}

off_t libc_lseek(int fd, off_t offset, int whence) {
    CALL_NEXT(lseek, fd, offset, whence)
}

ssize_t libc_pread(int fd, void *buf, size_t count, off_t offset) {
    CALL_NEXT(pread, fd, buf, count, offset)
}

ssize_t libc_pread64(int fd, void *buf, size_t count, off64_t offset) {
    CALL_NEXT(pread64, fd, buf, count, offset)
}

ssize_t libc_preadv(int fd, const struct iovec *iov, int iovcnt, off_t offset) {
    CALL_NEXT(preadv, fd, iov, iovcnt, offset)
}

ssize_t libc_preadv64(int fd, const struct iovec *iov, int iovcnt, off64_t offset) {
    CALL_NEXT(preadv64, fd, iov, iovcnt, offset)
}

ssize_t libc_pwrite(int fd, const void *buf, size_t count, off_t offset) {
    CALL_NEXT(pwrite, fd, buf, count, offset)
}

ssize_t libc_pwrite64(int fd, const void *buf, size_t count, off64_t offset) {
    CALL_NEXT(pwrite64, fd, buf, count, offset)
}

ssize_t libc_pwritev(int fd, const struct iovec *iov, int iovcnt, off_t offset) {
    CALL_NEXT(pwritev, fd, iov, iovcnt, offset)
}

ssize_t libc_pwritev64(int fd, const struct iovec *iov, int iovcnt, off64_t offset) {
    CALL_NEXT(pwritev64, fd, iov, iovcnt, offset)
}

ssize_t libc_readv(int fd, const struct iovec *iov, int iovcnt) {
    CALL_NEXT(readv, fd, iov, iovcnt)
}

ssize_t libc_writev(int fd, const struct iovec *iov, int iovcnt) {
    CALL_NEXT(writev, fd, iov, iovcnt)
}

int libc_ioctl(int fd, unsigned long request, void *argp) {
    CALL_NEXT(ioctl, fd, request, argp)
}

int libc_access(const char *pathname, int mode) {
    CALL_NEXT(access, pathname, mode)
}

int libc_unlink(const char *pathname) {
    CALL_NEXT(unlink, pathname)
}

int libc__xstat(int vers, const char *pathname, struct stat *buf) {
    CALL_NEXT(__xstat, vers, pathname, buf)
}

int libc__xstat64(int vers, const char *pathname, struct stat64 *buf) {
    CALL_NEXT(__xstat64, vers, pathname, buf)
}

int libc__lxstat(int vers, const char *pathname, struct stat *buf) {
    CALL_NEXT(__lxstat, vers, pathname, buf)
}

int libc__lxstat64(int vers, const char *pathname, struct stat64 *buf) {
    CALL_NEXT(__lxstat64, vers, pathname, buf)
}

int libc__fxstat(int vers, int fd, struct stat *buf) {
    CALL_NEXT(__fxstat, vers, fd, buf)
}

int libc__fxstat64(int vers, int fd, struct stat64 *buf) {
    CALL_NEXT(__fxstat64, vers, fd, buf)
}

int libc_statfs(const char *path, struct statfs *buf) {
    CALL_NEXT(statfs, path,  buf)
}

int libc_statfs64(const char *path, struct statfs64 *buf) {
    CALL_NEXT(statfs64, path,  buf)
}

int libc_fstatfs(int fd, struct statfs *buf) {
    CALL_NEXT(fstatfs, fd, buf)
}

int libc_fstatfs64(int fd, struct statfs64 *buf) {
    CALL_NEXT(fstatfs64, fd, buf)
}

FILE* libc_fdopen(int fd, const char *mode) {
    CALL_NEXT(fdopen, fd, mode)
}

FILE* libc_fopen(const char *path, const char *mode) {
    CALL_NEXT(fopen, path, mode)
}

FILE* libc_fopen64(const char *path, const char *mode) {
    CALL_NEXT(fopen64, path, mode)
}

FILE* libc_freopen(const char *path, const char *mode, FILE *stream) {
    CALL_NEXT(freopen, path, mode, stream)
}

FILE* libc_freopen64(const char *path, const char *mode, FILE *stream) {
    CALL_NEXT(freopen64, path, mode, stream)
}

int libc_fclose(FILE *stream) {
    CALL_NEXT(fclose, stream)
}

int libc_fflush(FILE *stream) {
    CALL_NEXT(fflush, stream)
}

int libc_fputc(int c, FILE *stream) {
    CALL_NEXT(fputc, c, stream)
}

char* libc_fgets(char *dst, int max, FILE *stream) {
    CALL_NEXT(fgets, dst, max, stream)
}

int libc_fgetc(FILE *stream) {
    CALL_NEXT(fgetc, stream)
}

int libc_fgetpos(FILE *stream, fpos_t *pos) {
    CALL_NEXT(fgetpos, stream, pos)
}

int libc_fgetpos64(FILE *stream, fpos64_t *pos) {
    CALL_NEXT(fgetpos64, stream, pos)
}

int libc_fseek(FILE *stream, long offset, int whence) {
    CALL_NEXT(fseek, stream, offset, whence)
}

int libc_fseeko(FILE *stream, off_t offset, int whence) {
    CALL_NEXT(fseeko, stream, offset, whence)
}

int libc_fseeko64(FILE *stream, off64_t offset, int whence) {
    CALL_NEXT(fseeko64, stream, offset, whence)
}

int libc_fsetpos(FILE *stream, const fpos_t *pos) {
    CALL_NEXT(fsetpos, stream, pos)
}

int libc_fsetpos64(FILE *stream, const fpos64_t *pos) {
    CALL_NEXT(fsetpos64, stream, pos)
}

int libc_fputs(const char *s, FILE *stream) {
    CALL_NEXT(fputs, s, stream)
}

int libc_putc(int c, FILE *stream) {
    CALL_NEXT(putc, c, stream)
}

int libc_getc(FILE *stream) {
    CALL_NEXT(getc, stream)
}

int libc_ungetc(int c, FILE *stream) {
    CALL_NEXT(ungetc, c, stream)
}

long libc_ftell(FILE *stream) {
    CALL_NEXT(ftell, stream)
}

off_t libc_ftello(FILE *stream) {
    CALL_NEXT(ftello, stream)
}

off64_t libc_ftello64(FILE *stream) {
    CALL_NEXT(ftello64, stream)
}

size_t libc_fread(void *ptr, size_t size, size_t nmemb, FILE *stream) {
    CALL_NEXT(fread, ptr, size, nmemb, stream)
}

size_t libc_fwrite(const void *ptr, size_t size, size_t nmemb, FILE *stream) {
    CALL_NEXT(fwrite, ptr, size, nmemb, stream)
}

void libc_rewind(FILE *stream) {
    CALL_NEXT(rewind, stream)
}

int libc_dup2(int oldfd, int newfd) {
    CALL_NEXT(dup2, oldfd, newfd)
}

int libc_unlinkat(int dirfd, const char *pathname, int flags) {
    CALL_NEXT(unlinkat, dirfd, pathname, flags)
}

int libc_faccessat(int dirfd, const char *pathname, int mode, int flags) {
    CALL_NEXT(faccessat, dirfd, pathname, mode, flags)
}

// __fxstatat is the glibc function corresponding to fstatat syscall
int libc__fxstatat(int vers, int dirfd, const char *pathname, struct stat *buf, int flags) {
    CALL_NEXT(__fxstatat, vers, dirfd, pathname, buf, flags)
}

// __fxstatat64 is the LARGEFILE64 version of glibc function corresponding to fstatat syscall
int libc__fxstatat64(int vers, int dirfd, const char *pathname, struct stat64 *buf, int flags) {
    CALL_NEXT(__fxstatat64, vers, dirfd, pathname, buf, flags)
}

int libc_openat(int dirfd, const char *pathname, int flags, int mode) {
    CALL_NEXT(openat, dirfd, pathname, flags, mode)
}

int libc_mkdir(const char *pathname, mode_t mode) {
    CALL_NEXT(mkdir, pathname, mode)
}

int libc_mkdirat(int dirfd, const char *pathname, mode_t mode) {
    CALL_NEXT(mkdirat, dirfd, pathname, mode)
}

int libc_rmdir(const char *pathname) {
    CALL_NEXT(rmdir, pathname)
}

int libc_rename(const char *oldpath, const char *newpath) {
    CALL_NEXT(rename, oldpath, newpath)
}

int libc_renameat(int olddirfd, const char *oldpath, int newdirfd, const char *newpath) {
    CALL_NEXT(renameat, olddirfd, oldpath, newdirfd, newpath)
}

int libc_renameat2(int olddirfd, const char *oldpath, int newdirfd, const char *newpath, unsigned int flags) {
    CALL_NEXT(renameat2, olddirfd, oldpath, newdirfd, newpath, flags)
}

int libc_posix_fadvise(int fd, off_t offset, off_t len, int advice) {
    CALL_NEXT(posix_fadvise, fd, offset, len, advice)
}

int libc_posix_fadvise64(int fd, off64_t offset, off64_t len, int advice) {
    CALL_NEXT(posix_fadvise64, fd, offset, len, advice)
}

int libc_statvfs(const char *pathname, struct statvfs *buf) {
    CALL_NEXT(statvfs, pathname, buf)
}

int libc_statvfs64(const char *pathname, struct statvfs64 *buf) {
    CALL_NEXT(statvfs64, pathname, buf)
}

int libc_fstatvfs(int fd, struct statvfs *buf) {
    CALL_NEXT(fstatvfs, fd, buf)
}

int libc_fstatvfs64(int fd, struct statvfs64 *buf) {
    CALL_NEXT(fstatvfs64, fd, buf)
}

ssize_t libc___getdelim(char **buf, size_t *bufsiz, int delimiter, FILE *fp) {
    CALL_NEXT(__getdelim, buf, bufsiz, delimiter, fp)
}

ssize_t libc_getline(char **buf, size_t *bufsiz, FILE *stream) {
    CALL_NEXT(getline, buf, bufsiz, stream)
}

DIR* libc_opendir(const char* path) {
    CALL_NEXT(opendir, path)
}

int libc_feof(FILE *stream) {
    CALL_NEXT(feof, stream)
}

int libc_ferror(FILE *stream) {
    CALL_NEXT(ferror, stream)
}

ssize_t libc_getxattr(const char *path, const char *name, void *value,  size_t size) {
    CALL_NEXT(getxattr, path, name, value, size)
}