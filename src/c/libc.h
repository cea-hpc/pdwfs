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

#ifndef LIBC_H
#define LIBC_H

#include <stdio.h>
#include <stdlib.h>
#include <dirent.h>

int libc_open(const char *pathname, int flags, int mode);
int libc_close(int fd);
ssize_t libc_write(int fd, const void *buf, size_t count);
ssize_t libc_read(int fd, void *buf, size_t count);
int libc_open64(const char *pathname, int flags, int mode);
int libc_creat(const char *pathname, mode_t mode);
int libc_creat64(const char *pathname, mode_t mode);
int libc_fdatasync(int fd);
int libc_fsync(int fd);
int libc_ftruncate64(int fd, off64_t length);
int libc_ftruncate(int fd, off_t length);
int libc_truncate64(const char *path, off64_t length);
int libc_truncate(const char *path, off_t length);
off64_t libc_lseek64(int fd, off64_t offset, int whence);
off_t libc_lseek(int fd, off_t offset, int whence);
ssize_t libc_pread(int fd, void *buf, size_t count, off_t offset);
ssize_t libc_pread64(int fd, void *buf, size_t count, off64_t offset);
ssize_t libc_preadv(int fd, const struct iovec *iov, int iovcnt, off_t offset);
ssize_t libc_preadv64(int fd, const struct iovec *iov, int iovcnt, off64_t offset);
ssize_t libc_pwrite(int fd, const void *buf, size_t count, off_t offset);
ssize_t libc_pwrite64(int fd, const void *buf, size_t count, off64_t offset);
ssize_t libc_pwritev(int fd, const struct iovec *iov, int iovcnt, off_t offset);
ssize_t libc_pwritev64(int fd, const struct iovec *iov, int iovcnt, off64_t offset);
ssize_t libc_readv(int fd, const struct iovec *iov, int iovcnt);
ssize_t libc_writev(int fd, const struct iovec *iov, int iovcnt);
int libc_ioctl(int fd, unsigned long request, void *argp);
int libc_access(const char *pathname, int mode);
int libc_unlink(const char *pathname);
int libc__xstat(int vers, const char *pathname, struct stat *buf);
int libc__xstat64(int vers, const char *pathname, struct stat64 *buf);
int libc__lxstat(int vers, const char *pathname, struct stat *buf);
int libc__lxstat64(int vers, const char *pathname, struct stat64 *buf);
int libc__fxstat(int vers, int fd, struct stat *buf);
int libc__fxstat64(int vers, int fd, struct stat64 *buf);
int libc_statfs(const char *path, struct statfs *buf);
int libc_statfs64(const char *path, struct statfs64 *buf);
int libc_fstatfs(int fd, struct statfs *buf);
int libc_fstatfs64(int fd, struct statfs64 *buf);
FILE* libc_fdopen(int fd, const char *mode);
FILE* libc_fopen(const char *path, const char *mode);
FILE* libc_fopen64(const char *path, const char *mode);
FILE* libc_freopen(const char *path, const char *mode, FILE *stream);
FILE* libc_freopen64(const char *path, const char *mode, FILE *stream);
int libc_fclose(FILE *stream);
int libc_fflush(FILE *stream);
int libc_fputc(int c, FILE *stream);
char* libc_fgets(char *dst, int max, FILE *stream);
int libc_fgetc(FILE *stream);
int libc_fgetpos(FILE *stream, fpos_t *pos);
int libc_fgetpos64(FILE *stream, fpos64_t *pos);
int libc_fseek(FILE *stream, long offset, int whence);
int libc_fseeko(FILE *stream, off_t offset, int whence);
int libc_fseeko64(FILE *stream, off64_t offset, int whence);
int libc_fsetpos(FILE *stream, const fpos_t *pos);
int libc_fsetpos64(FILE *stream, const fpos64_t *pos);
int libc_fputs(const char *s, FILE *stream);
int libc_putc(int c, FILE *stream);
int libc_getc(FILE *stream);
int libc_ungetc(int c, FILE *stream);
long libc_ftell(FILE *stream);
off_t libc_ftello(FILE *stream);
off64_t libc_ftello64(FILE *stream);
size_t libc_fread(void *ptr, size_t size, size_t nmemb, FILE *stream);
size_t libc_fwrite(const void *ptr, size_t size, size_t nmemb, FILE *stream);
void libc_rewind(FILE *stream);
int libc_dup2(int oldfd, int newfd);
int libc_unlinkat(int dirfd, const char *pathname, int flags);
int libc_faccessat(int dirfd, const char *pathname, int mode, int flags);
int libc__fxstatat(int vers, int dirfd, const char *pathname, struct stat *buf, int flags);
int libc__fxstatat64(int vers, int dirfd, const char *pathname, struct stat64 *buf, int flags);
int libc_openat(int dirfd, const char *pathname, int flags, int mode);
int libc_mkdir(const char *pathname, mode_t mode);
int libc_mkdirat(int dirfd, const char *pathname, mode_t mode);
int libc_rmdir(const char *pathname);
int libc_rename(const char *oldpath, const char *newpath);
int libc_renameat(int olddirfd, const char *oldpath, int newdirfd, const char *newpath);
int libc_renameat2(int olddirfd, const char *oldpath, int newdirfd, const char *newpath, unsigned int flags);
int libc_posix_fadvise(int fd, off_t offset, off_t len, int advice);
int libc_posix_fadvise64(int fd, off64_t offset, off64_t len, int advice);
int libc_statvfs(const char *pathname, struct statvfs *buf);
int libc_statvfs64(const char *pathname, struct statvfs64 *buf);
int libc_fstatvfs(int fd, struct statvfs *buf);
int libc_fstatvfs64(int fd, struct statvfs64 *buf);
ssize_t libc___getdelim(char **buf, size_t *bufsiz, int delimiter, FILE *fp);
ssize_t libc_getline(char **buf, size_t *bufsiz, FILE *stream);
DIR* libc_opendir(const char* path);
int libc_feof(FILE *stream);
int libc_ferror(FILE *stream);
ssize_t libc_getxattr(const char *path, const char *name, void *value,  size_t size);


#endif