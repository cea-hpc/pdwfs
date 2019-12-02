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
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/uio.h>
#include <sys/stat.h>
#include <sys/statfs.h>
#include <sys/statvfs.h>   
#include <errno.h>
#include <fcntl.h>

#include <glib.h>
#include <glib/gprintf.h>

#include "libc.h"
#include "libpdwfs_go.h"
#include "utils.h"

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

#define NOT_IMPLEMENTED(symb) {\
    pdwfs_fprintf(stderr, RED, "ERROR", "%s not implemented by pdwfs\n", symb);\
    exit(EXIT_FAILURE);\
}

#define IS_STD_FD(fd) (fd == STDIN_FILENO || fd == STDOUT_FILENO || fd == STDERR_FILENO)

#define PATH_NOT_MANAGED(path) (!pdwfs_initialized || !contains_path(mount_register, path))
#define FD_NOT_MANAGED(fd) (!pdwfs_initialized || IS_STD_FD(fd) || !contains_fd(fd_register, fd))
#define STREAM_NOT_MANAGED(stream) FD_NOT_MANAGED(fileno(stream))


//-----------------------------------------------------------------------------------------
// mount_register
//
// used to register pdwfs mount points and check if a filename belongs to one of the mount point
// if not, the call is passed to the system libc call

// callback when a key is removed from the hash table
void mount_key_removed(gpointer key) {
    g_free(key);
}

GHashTable *new_mount_register() {
    return g_hash_table_new_full(g_str_hash, g_str_equal, (GDestroyNotify)mount_key_removed, NULL);
}

void free_mount_register(GHashTable *self) {
    g_hash_table_destroy(self);
}

// register pdwfs mount points into the hash table
void register_mounts(GHashTable *self, gchar **mounts) {
    int i = 0;
    gchar* mount = mounts[0];
	while (mount != NULL) {
        if (g_strcmp0(mount, "") !=0) {
            g_hash_table_insert(self, g_strdup(mount), "");
        }
        i++;
        mount = mounts[i];
    }
}

// lookup function used in g_hash_table_find
gboolean finder(gpointer mount, gpointer unused, gpointer abspath) {
    return g_str_has_prefix(abspath, mount);
}

// returns 1 if the path in argument belongs to one of the mount points registered
int contains_path(GHashTable *self, const char *path) {
    
    char *apath = abspath(path);
    if (!apath) {
        return 0;
    }
    gpointer item_ptr = g_hash_table_find(self, (GHRFunc)finder, apath);
    free(apath);
    return (item_ptr) ? 1 : 0;    
}

// end of mount_register
//-----------------------------------------------------------------------------------------


//-----------------------------------------------------------------------------------------
// fd_register
//
// when a newly created file is managed by pdwfs, the fd_register creates a "twin" local temporary file
// to provide a valid system file descriptor or valid FILE stream object

// callback when a key is removed
void fd_key_removed(gpointer fd_ptr) {
    close(GPOINTER_TO_INT(fd_ptr));
}

GHashTable *new_fd_register() {
    return g_hash_table_new_full(g_direct_hash, g_direct_equal, (GDestroyNotify)fd_key_removed, NULL);
}

void free_fd_register(GHashTable *self) {
    g_hash_table_destroy(self);
}

// returns a new FILE stream object and registers its file descriptor
FILE* get_new_stream(GHashTable *self) {
    FILE *fp = tmpfile();
    g_hash_table_insert(self, GINT_TO_POINTER(fileno(fp)), GINT_TO_POINTER(0));
    return fp;
}

// returns a new file descriptor and registers it
int get_new_fd(GHashTable *self) {
    return fileno(get_new_stream(self));
}

void remove_fd(GHashTable *self, int fd) {
    g_hash_table_remove(self, GINT_TO_POINTER(fd));
}

int contains_fd(GHashTable *self, int fd) {
    return g_hash_table_contains(self, GINT_TO_POINTER(fd));    
}

// end of fd_register
//-----------------------------------------------------------------------------------------


static int pdwfs_initialized = 0;
// there are cases where pdwfs is not yet initialized and a another library constructor
// (called before pdwfs.so constructor) does some IO (e.g libselinux, libnuma)
// in such case we do not know yet if the file/fd is managed by pdwfs, 
// so we defer the call to the real system calls (there's hardly any chance that these IOs 
// calls are the one we intend to intercept anyway)

static GHashTable *fd_register = NULL;
static GHashTable *mount_register = NULL;

static __attribute__((constructor)) void init_pdwfs(void) {
    char buf[1024];
    GoSlice mounts_buf = {&buf, 0, 1024};
    InitPdwfs(mounts_buf);

    mount_register = new_mount_register();
    // parse the list of mount point paths obtained from the Go layer
    gchar **mounts = g_strsplit((char*)mounts_buf.data, "@", -1);
    register_mounts(mount_register, mounts);
    g_strfreev(mounts);
    fd_register = new_fd_register();
    pdwfs_initialized = 1;
}

static __attribute__((destructor)) void finalize_pdwfs(void) {
    FinalizePdwfs();
    free_fd_register(fd_register);
    free_mount_register(mount_register);
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

    if PATH_NOT_MANAGED(pathname) {
        return libc_open(pathname, flags, mode);
    }

    GoString filename = {strdup(pathname), strlen(pathname)};

    int fd = get_new_fd(fd_register);

    int ret = Open(filename, flags, mode, fd);
    if (ret < 0) {
        errno = GetErrno();
        remove_fd(fd_register, fd);
    }
    return ret;
}

int open64(const char *pathname, int flags, ...) __attribute__((alias("open"))); 

int close(int fd) {
    TRACE("intercepting close(fd=%d)\n", fd)

    if FD_NOT_MANAGED(fd) {
        return libc_close(fd);
    }
    int ret = Close(fd);
    remove_fd(fd_register, fd);
    return ret;
}

ssize_t write(int fd, const void *buf, size_t count) {
    TRACE("intercepting write(fd=%d, buf=%p, count=%lu)\n", fd, buf, count)
    
    if FD_NOT_MANAGED(fd) {
        return libc_write(fd, buf, count);
    }
    GoSlice buffer = {(void*)buf, count, count};
    return Write(fd, buffer);
}

ssize_t read(int fd, void *buf, size_t count) {
    TRACE("intercepting read(fd=%d, buf=%p, count=%lu)\n", fd, buf, count)
    
    if FD_NOT_MANAGED(fd) {
        return libc_read(fd, buf, count);
    }
    GoSlice buffer = {buf, count, count};
    return Read(fd, buffer);
}

int creat(const char *pathname, mode_t mode) {
    TRACE("intercepting creat(pathname=%s, mode=%d)\n", pathname, mode)
    
    if PATH_NOT_MANAGED(pathname) {
        return libc_creat(pathname, mode);
    }
    NOT_IMPLEMENTED("creat")
}

int creat64(const char *pathname, mode_t mode) __attribute__((alias("creat"))); 

int fdatasync(int fd) {
    TRACE("intercepting fdatasync(fd=%d)\n", fd)
    
    if FD_NOT_MANAGED(fd) {
        return libc_fdatasync(fd);
    }
    NOT_IMPLEMENTED("fdatasync")
}

int fsync(int fd) {
    TRACE("intercepting fsync(fd=%d)\n", fd)
    
    if FD_NOT_MANAGED(fd) {
        return libc_fsync(fd);
    }
    NOT_IMPLEMENTED("fsync")
}

int ftruncate64(int fd, off64_t length) {
    TRACE("intercepting ftrunctate64(fd=%d, length=%ld)\n", fd, length)

    if FD_NOT_MANAGED(fd) {
        return libc_ftruncate64(fd, length);
    }
    return Ftruncate(fd, length);
}

int ftruncate(int fd, off_t length) {
    TRACE("callled ftruncate(fd=%d; length=%ld)\n", fd, length)

    if FD_NOT_MANAGED(fd) {
        return libc_ftruncate(fd, length);
    }
    return Ftruncate(fd, length);
}

int truncate64(const char *path, off64_t length) {
    TRACE("intercepting truncate64(path=%s, length=%ld)\n", path, length)

    if PATH_NOT_MANAGED(path) {
        return libc_truncate64(path, length);
    }
    NOT_IMPLEMENTED("truncate64")
}

int truncate(const char *path, off_t length) {
    TRACE("intercepting truncate(path=%s, length=%ld)\n", path, length)

    if PATH_NOT_MANAGED(path) {
        return libc_truncate(path, length);
    }
    NOT_IMPLEMENTED("truncate")
}

off64_t lseek64(int fd, off64_t offset, int whence) {
    TRACE("intercepting lseek64(fd=%d, offset=%ld; whence=%d)\n", fd, offset, whence)

    if FD_NOT_MANAGED(fd) {
        return libc_lseek64(fd, offset, whence);
    }
    return Lseek(fd, offset, whence);
}

off_t lseek(int fd, off_t offset, int whence) {
    TRACE("intercepting lseek(fd=%d; offset=%ld, whence=%d)\n", fd, offset, whence)

    if FD_NOT_MANAGED(fd) {
        return libc_lseek(fd, offset, whence);
    }
    return Lseek(fd, offset, whence);
}

ssize_t pread(int fd, void *buf, size_t count, off_t offset) {
    TRACE("intercepting pread(fd=%d, buf=%p, count=%lu, offset=%ld)\n", fd, buf, count, offset)

    if FD_NOT_MANAGED(fd) {
        return libc_pread(fd, buf, count, offset);
    }
    GoSlice buffer = {buf, count, count};
    return Pread(fd, buffer, offset);
}

ssize_t pread64(int fd, void *buf, size_t count, off64_t offset) {
    TRACE("intercepting pread64(fd=%d, buf=%p, count=%lu, offset=%ld)\n", fd, buf, count, offset)

    if FD_NOT_MANAGED(fd) {
        return libc_pread64(fd, buf, count, offset);
    }
    GoSlice buffer = {buf, count, count};
    return Pread(fd, buffer, offset);
}

ssize_t preadv(int fd, const struct iovec *iov, int iovcnt, off_t offset) {
    TRACE("intercepting preadv(fd=%d, iov=%p, iovcnt=%d, offset=%ld)\n", fd, iov, iovcnt, offset)

    if FD_NOT_MANAGED(fd) {
        return libc_preadv(fd, iov, iovcnt, offset);
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

    if FD_NOT_MANAGED(fd) {
        return libc_preadv64(fd, iov, iovcnt, offset);
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

    if FD_NOT_MANAGED(fd) {
        return libc_pwrite(fd, buf, count, offset);
    }
    GoSlice buffer = {(void*)buf, count, count};
    return Pwrite(fd, buffer, offset);
}

ssize_t pwrite64(int fd, const void *buf, size_t count, off64_t offset) {
    TRACE("intercepting pwrite64(fd=%d, buf=%p, count=%lu, offset=%ld)\n", fd, buf, count, offset)

    if FD_NOT_MANAGED(fd) {
        return libc_pwrite64(fd, buf, count, offset);
    }
    GoSlice buffer = {(void*)buf, count, count};
    return Pwrite(fd, buffer, offset);
}

ssize_t pwritev(int fd, const struct iovec *iov, int iovcnt, off_t offset) {
    TRACE("intercepting pwritev(fd=%d, iov=%p, iovcnt=%d, offset=%ld)\n", fd, iov, iovcnt, offset)

    if FD_NOT_MANAGED(fd) {
        return libc_pwritev(fd, iov, iovcnt, offset);
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

    if FD_NOT_MANAGED(fd) {
        return libc_pwritev64(fd, iov, iovcnt, offset);
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

    if FD_NOT_MANAGED(fd) {
        return libc_readv(fd, iov, iovcnt);
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

    if FD_NOT_MANAGED(fd) {
        return libc_writev(fd, iov, iovcnt);
    }

    GoSlice vec[iovcnt];

    for (int i = 0; i < iovcnt; i++) {
        GoSlice s = {iov[i].iov_base, iov[i].iov_len, iov[i].iov_len};
        vec[i] = s;
    }
    GoSlice iovSlice = {&vec, iovcnt, iovcnt};

    return Writev(fd, iovSlice);
}

int ioctl(int fd, unsigned long request, void *argp) {
    TRACE("intercepting ioctl(fd=%d, request=%lu, argp=%p)\n", fd, request, argp)

    if FD_NOT_MANAGED(fd) {
        return libc_ioctl(fd, request, argp);
    }
    NOT_IMPLEMENTED("ioctl")
}

int access(const char *pathname, int mode) {
    TRACE("intercepting access(pathname=%s, mode=%d)\n", pathname, mode)

    if PATH_NOT_MANAGED(pathname) {
        return libc_access(pathname, mode);
    }
    GoString filename = {strdup(pathname), strlen(pathname)};
    return Access(filename, mode);
}

int unlink(const char *pathname) {
    TRACE("intercepting unlink(pathname=%s)\n", pathname)

    if PATH_NOT_MANAGED(pathname) {
        return libc_unlink(pathname);
    }
    GoString filename = {strdup(pathname), strlen(pathname)};
    int ret = Unlink(filename);
    if (ret < 0) {
        errno = GetErrno();
    }
    return ret;
}

int __xstat(int vers, const char *pathname, struct stat *buf) {
    TRACE("intercepting __xstat(vers=%d, pathname=%s, buf=%p)\n", vers, pathname, buf)

    if PATH_NOT_MANAGED(pathname) {
        return libc__xstat(vers, pathname, buf);
    }
    GoString filename = {strdup(pathname), strlen(pathname)};
    return Stat(filename, buf);
}

int __xstat64(int vers, const char *pathname, struct stat64 *buf) {
    TRACE("intercepting __xstat64(vers=%d, pathname=%s, buf=%p)\n", vers, pathname, buf)

    if PATH_NOT_MANAGED(pathname) {
        return libc__xstat64(vers, pathname, buf);
    }
    GoString filename = {strdup(pathname), strlen(pathname)};
    return Stat64(filename, buf);
}

int __lxstat(int vers, const char *pathname, struct stat *buf) {
    TRACE("intercepting __lxstat(vers=%d, pathname=%s, buf=%p)\n", vers, pathname, buf)

    if PATH_NOT_MANAGED(pathname) {
        return libc__lxstat(vers, pathname, buf);
    }
    GoString filename = {strdup(pathname), strlen(pathname)};
    return Lstat(filename, buf);
}

int __lxstat64(int vers, const char *pathname, struct stat64 *buf) {
    TRACE("intercepting __lxstat64(vers=%d, pathname=%s, buf=%p)\n", vers, pathname, buf)

    if PATH_NOT_MANAGED(pathname) {
        return libc__lxstat64(vers, pathname, buf);
    }
    GoString filename = {strdup(pathname), strlen(pathname)};
    return Lstat64(filename, buf);
}

int __fxstat(int vers, int fd, struct stat *buf) {
    TRACE("intercepting __fxstat(vers=%d, fd=%d, buf=%p)\n", vers, fd, buf)

    if FD_NOT_MANAGED(fd) {
        return libc__fxstat(vers, fd, buf);
    }
    return Fstat(fd, buf);
}

int __fxstat64(int vers, int fd, struct stat64 *buf) {
    TRACE("intercepting __fxstat64(vers=%d, fd=%d, buf=%p)\n", vers, fd, buf)
    
    if FD_NOT_MANAGED(fd) {
        return libc__fxstat64(vers, fd, buf);
    }
    return Fstat64(fd, buf);
}

int statfs(const char *path, struct statfs *buf) {
    TRACE("intercepting statfs(path=%s, buf=%p)\n", path, buf)

    if PATH_NOT_MANAGED(path) {
        return libc_statfs(path,  buf);
    }
    GoString filename = {strdup(path), strlen(path)};
    return Statfs(filename, buf);
}

int statfs64(const char *path, struct statfs64 *buf) {
    TRACE("intercepting statfs64(path=%s, buf=%p)\n", path, buf)

    if PATH_NOT_MANAGED(path) {
        return libc_statfs64(path,  buf);
    }
    GoString filename = {strdup(path), strlen(path)};
    return Statfs64(filename, buf);
}

int fstatfs(int fd, struct statfs *buf) {
    TRACE("intercepting fstatfs(fd=%d, buf=%p)\n", fd, buf)

    if FD_NOT_MANAGED(fd) {
        return libc_fstatfs(fd, buf);
    }
    NOT_IMPLEMENTED("fstatfs")
}

int fstatfs64(int fd, struct statfs64 *buf) {
    TRACE("intercepting fstatfs64(fd=%d, buf=%p)\n", fd, buf)

    if FD_NOT_MANAGED(fd) {
        return libc_fstatfs64(fd, buf);
    }
    NOT_IMPLEMENTED("fstatfs64")
}

FILE* fdopen(int fd, const char *mode) {
    TRACE("intercepting fdopen(fd=%d, mode=%s)\n", fd, mode)

    if FD_NOT_MANAGED(fd) {
        return libc_fdopen(fd, mode);
    }
    NOT_IMPLEMENTED("fdopen")
}

FILE* fopen(const char *path, const char *mode) {
    TRACE("intercepting fopen(path=%s, mode=%s)\n", path, mode)

    if PATH_NOT_MANAGED(path) {
        return libc_fopen(path, mode);
    }

    FILE *stream = get_new_stream(fd_register);
    GoString gopath = {strdup(path), strlen(path)};
    GoString gomode = {strdup(mode), strlen(mode)};
    int ret = Fopen(gopath, gomode, fileno(stream));
    if (ret < 0) {
        remove_fd(fd_register, fileno(stream));
        return (FILE*)(NULL);
    }
    return stream;
}

FILE* fopen64(const char *path, const char *mode) __attribute__((alias("fopen")));

FILE* freopen(const char *path, const char *mode, FILE *stream) {
    TRACE("intercepting fropen(path=%s, mode=%s, stream=%p)\n", path, mode, stream)

    if PATH_NOT_MANAGED(path) {
        return libc_freopen(path, mode, stream);
    }
    NOT_IMPLEMENTED("freopen")
}

FILE* freopen64(const char *path, const char *mode, FILE *stream)  __attribute__((alias("fopen")));

int fclose(FILE *stream) {
    TRACE("intercepting fclose(stream=%p)\n", stream)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fclose(stream);
    }
    Fflush(stream);
    return close(fileno(stream));
}

int fflush(FILE *stream) {
    TRACE("intercepting fflush(stream=%p)\n", stream)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fflush(stream);
    }
    return Fflush(stream);
}

int fputc(int c, FILE *stream) {
    TRACE("intercepting fputc(c=%d, stream=%p)\n", c, stream)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fputc(c, stream);
    }
    GoSlice cBuf = {(char*)&c, 1, 1};
    int n = Write(fileno(stream), cBuf); 

    if (n <= 0){
        stream->_flags |= (_IO_ERR_SEEN|_IO_EOF_SEEN);
        return EOF;
    }

	return c;
}

char* fgets(char *dst, int max, FILE *stream) {
    TRACE("intercepting fgets(dst=%s, max=%d, stream=%p)\n", dst, max, stream)
    
    if STREAM_NOT_MANAGED(stream) {
        return libc_fgets(dst, max, stream);
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
    
    if STREAM_NOT_MANAGED(stream) {
        return libc_fgetc(stream);
    }
    
	char c;
    GoSlice cBuf = {&c, 1, 1};
    int n = Read(fileno(stream), cBuf); 
	if (n == 0){
        stream->_flags |= (_IO_ERR_SEEN|_IO_EOF_SEEN);
		return EOF;
    }
    if (n < 0){
        stream->_flags |= _IO_ERR_SEEN;
        return n;
    }
	return c;
}

int fgetpos(FILE *stream, fpos_t *pos) {
    TRACE("intercepting fgetpos(stream=%p, pos=%p)\n", stream, pos)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fgetpos(stream, pos);
    }
    NOT_IMPLEMENTED("fgetpos")
}

int fgetpos64(FILE *stream, fpos64_t *pos) {
    TRACE("intercepting fgetpos64(stream=%p, pos=%p)\n", stream, pos)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fgetpos64(stream, pos);
    }
    NOT_IMPLEMENTED("fgetpos64")
}

int fseek(FILE *stream, long offset, int whence) {
    TRACE("intercepting fseek(stream=%p, offset=%ld, whence=%d)\n", stream, offset, whence)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fseek(stream, offset, whence);
    }
    NOT_IMPLEMENTED("fseek")
}

int fseeko(FILE *stream, off_t offset, int whence) {
    TRACE("intercepting fseeko(stream=%p, offset=%ld, whence=%d)\n", stream, offset, whence)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fseeko(stream, offset, whence);
    }
    NOT_IMPLEMENTED("fseeko")
}

int fseeko64(FILE *stream, off64_t offset, int whence) {
    TRACE("intercepting fseeko64(stream=%p, offset=%ld, whence=%d)\n", stream, offset, whence)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fseeko64(stream, offset, whence);
    }
    NOT_IMPLEMENTED("fseeko")
}

int fsetpos(FILE *stream, const fpos_t *pos) {
    TRACE("intercepting fsetpos(stream=%p, pos=%p)\n", stream, pos)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fsetpos(stream, pos);
    }
    NOT_IMPLEMENTED("fsetpos")
}

int fsetpos64(FILE *stream, const fpos64_t *pos) {
    TRACE("intercepting fsetpos64(stream=%p, pos=%p)\n", stream, pos)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fsetpos64(stream, pos);
    }
    NOT_IMPLEMENTED("fsetpos64")
}

int fputs(const char *s, FILE *stream) {
    TRACE("intercepting fputs(s=%s, stream=%p)\n", s, stream)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fputs(s, stream);
    }
    size_t len = strlen(s);
	return (fwrite(s, 1, len, stream)==len) - 1;
}

int putc(int c, FILE *stream) {
    TRACE("intercepting putc(c=%d, stream=%p)\n", c, stream)

    if STREAM_NOT_MANAGED(stream) {
        return libc_putc(c, stream);
    }
    NOT_IMPLEMENTED("putc")
}

int getc(FILE *stream) {
    TRACE("intercepting getc(stream=%p)\n", stream)

    if STREAM_NOT_MANAGED(stream) {
        return libc_getc(stream);
    }
    NOT_IMPLEMENTED("getc")
}

int ungetc(int c, FILE *stream) {
    TRACE("intercepting fgetc(c=%d, stream=%p)\n", c, stream)

    if STREAM_NOT_MANAGED(stream) {
        return libc_ungetc(c, stream);
    }
    NOT_IMPLEMENTED("ungetc")
}

long ftell(FILE *stream) {
    TRACE("intercepting ftell(stream=%p)\n", stream)

    if STREAM_NOT_MANAGED(stream) {
        return libc_ftell(stream);
    }
    NOT_IMPLEMENTED("ftell")
}

off_t ftello(FILE *stream) {
    TRACE("intercepting ftello(stream=%p)\n", stream)

    if STREAM_NOT_MANAGED(stream) {
        return libc_ftello(stream);
    }
    NOT_IMPLEMENTED("ftello")
}

off64_t ftello64(FILE *stream) {
    TRACE("intercepting ftello64(stream=%p)\n", stream)

    if STREAM_NOT_MANAGED(stream) {
        return libc_ftello64(stream);
    }
    NOT_IMPLEMENTED("ftello64")
}

size_t fread(void *ptr, size_t size, size_t nmemb, FILE *stream) {
    TRACE("intercepting fread(ptr=%p, size=%lu, nmemb=%lu, stream=%p)\n", ptr, size, nmemb, stream)
    
    if STREAM_NOT_MANAGED(stream) {
        return libc_fread(ptr, size, nmemb, stream);
    }
    GoSlice buffer = {ptr, size * nmemb, size * nmemb};
    int ret = Read(fileno(stream), buffer);

    if (ret != nmemb) {
        if (ret == -1)
            stream->_flags |= _IO_ERR_SEEN;
        int fd = fileno(stream);
        off_t cur_off = lseek(fd, 0, SEEK_CUR);
        if (cur_off == lseek(fd, 0, SEEK_END)) 
            stream->_flags |= _IO_EOF_SEEN;
        lseek(fd, cur_off, SEEK_SET);
    }

    return ret;
}

size_t fwrite(const void *ptr, size_t size, size_t nmemb, FILE *stream) {
    TRACE("intercepting fwrite(ptr=%p, size=%lu, nmemb=%lu, stream=%p)\n", ptr, size, nmemb, stream)

    if STREAM_NOT_MANAGED(stream) {
        return libc_fwrite(ptr, size, nmemb, stream);
    }
    GoSlice buffer = {(void*)ptr, size * nmemb, size * nmemb};
    int ret = Write(fileno(stream), buffer);
    if (ret != nmemb) {
        errno = GetErrno();
        stream->_flags |= _IO_ERR_SEEN;
    }
    return ret;
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

    int ret = fputs(msg, stream); 
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

    int ret = fputs(msg, stream); 
    free(msg);
    return ret;
}

void rewind(FILE *stream) {
    TRACE("intercepting rewind(stream=%p)\n", stream)

    if STREAM_NOT_MANAGED(stream) {
        libc_rewind(stream);
        return;
    }
    NOT_IMPLEMENTED("rewind")
}

int dup2(int oldfd, int newfd) {
    TRACE("intercepting dup2(oldfd=%d, newfd=%d)\n", oldfd, newfd)

    if (FD_NOT_MANAGED(oldfd) && FD_NOT_MANAGED(newfd)) {
        return libc_dup2(oldfd, newfd);
    }
    NOT_IMPLEMENTED("dup2")
}

int unlinkat(int dirfd, const char *pathname, int flags) {
    TRACE("intercepting unlinkat(dirfd=%d, pathname=%s, flags=%d) (PASS THOUGH)\n", dirfd, pathname, flags)
    return libc_unlinkat(dirfd, pathname, flags);
    }

int faccessat(int dirfd, const char *pathname, int mode, int flags) {
    TRACE("intercepting faccessat(dirfd=%d, pathname=%s, mode=%d, flags=%d) (PASS THOUGH)\n", dirfd, pathname, mode, flags)
    return libc_faccessat(dirfd, pathname, mode, flags);
    }

// __fxstatat is the glibc function corresponding to fstatat syscall
int __fxstatat(int vers, int dirfd, const char *pathname, struct stat *buf, int flags) {
    TRACE("intercepting __fxstatat(vers=%d, dirfd=%d, pathname=%s, buf=%p, flags=%d) (PASS THOUGH)\n", vers, dirfd, pathname, buf, flags)
    return libc__fxstatat(vers, dirfd, pathname, buf, flags);
}

// __fxstatat64 is the LARGEFILE64 version of glibc function corresponding to fstatat syscall
int __fxstatat64(int vers, int dirfd, const char *pathname, struct stat64 *buf, int flags) {
    TRACE("intercepting __fxstatat64(vers=%d, dirfd=%d, pathname=%s, buf=%p, flags=%d) (PASS THOUGH)\n", dirfd, pathname, buf, flags)
    return libc__fxstatat64(vers, dirfd, pathname, buf, flags);
}

int openat(int dirfd, const char *pathname, int flags, ...) {
    
    int mode = 0;

    if (__OPEN_NEEDS_MODE (flags)) {
        va_list arg;
        va_start(arg, flags);
        mode = va_arg(arg, int);
        va_end(arg);
        }

    TRACE("intercepting openat(dirfd=%d, pathname=%s, flags=%d, mode=%d) (PASS THOUGH)\n", dirfd, pathname, flags, mode)
    return libc_openat(dirfd, pathname, flags, mode);
}

int mkdir(const char *pathname, mode_t mode) {
    TRACE("intercepting mkdir(pathname=%s, mode=%d)\n", pathname, mode)
    
    if PATH_NOT_MANAGED(pathname) {
        return libc_mkdir(pathname, mode);
    }
    GoString gopath = {strdup(pathname), strlen(pathname)};
    return Mkdir(gopath, mode);
}

int mkdirat(int dirfd, const char *pathname, mode_t mode) {
    TRACE("intercepting mkdirat(dirfd=%d, pathname=%s, mode=%d) (PASS THROUGH)\n", dirfd, pathname, mode)
    return libc_mkdirat(dirfd, pathname, mode);
}

int rmdir(const char *pathname) {
    TRACE("intercepting rmdir(pathname=%s)\n", pathname)
    
    if PATH_NOT_MANAGED(pathname) {
        return libc_rmdir(pathname);
    }
    GoString gopath = {strdup(pathname), strlen(pathname)};
    return Rmdir(gopath);
}

int rename(const char *oldpath, const char *newpath) {
    TRACE("intercepting rename(oldname=%s, newpath=%s)\n", oldpath, newpath)
    
    if (PATH_NOT_MANAGED(oldpath) && PATH_NOT_MANAGED(newpath)) {
        return libc_rename(oldpath, newpath);
    }
    NOT_IMPLEMENTED("rename")
}

int renameat(int olddirfd, const char *oldpath, int newdirfd, const char *newpath) {
    TRACE("intercepting renameat(olddirfd=%d, oldpath=%s, newdirfd=%d, newpath=%s) (PASS THROUGH)\n", olddirfd, oldpath, newdirfd, newpath)
    return libc_renameat(olddirfd, oldpath, newdirfd, newpath);
}

int renameat2(int olddirfd, const char *oldpath, int newdirfd, const char *newpath, unsigned int flags) {
    TRACE("intercepting renameat2(olddirfd=%d, oldpath=%s, newdirfd=%d, newpath=%s, flags=%d) (PASS THROUGH)\n", olddirfd, oldpath, newdirfd, newpath, flags)
    return libc_renameat2(olddirfd, oldpath, newdirfd, newpath, flags);
}

int posix_fadvise(int fd, off_t offset, off_t len, int advice) {
    TRACE("intercepting posix_fadvise(fd=%d, offset=%d, len=%d, advice=%d)\n", fd, offset, len, advice)
 
    if FD_NOT_MANAGED(fd) {
        return libc_posix_fadvise(fd, offset, len, advice);
    }
    return Fadvise(fd, offset, len, advice);
}

int posix_fadvise64(int fd, off64_t offset, off64_t len, int advice) {
    TRACE("intercepting posix_fadvise64(fd=%d, offset=%d, len=%d, advice=%d)\n", fd, offset, len, advice)
    
    if FD_NOT_MANAGED(fd) {
        return libc_posix_fadvise64(fd, offset, len, advice);
    }
    return Fadvise(fd, offset, len, advice);
}

int statvfs(const char *pathname, struct statvfs *buf) {
    TRACE("intercepting statvfs(path=%s, buf=%p)\n", pathname, buf)
    
    if PATH_NOT_MANAGED(pathname) {
        return libc_statvfs(pathname, buf);
    }
    GoString gopath = {strdup(pathname), strlen(pathname)};
    return Statvfs(gopath, buf);
}

int statvfs64(const char *pathname, struct statvfs64 *buf) {
    TRACE("intercepting statvfs64(path=%s, buf=%p)\n", pathname, buf)
    
    if PATH_NOT_MANAGED(pathname) {
        return libc_statvfs64(pathname, buf);
    }
    GoString gopath = {strdup(pathname), strlen(pathname)};
    return Statvfs64(gopath, buf);
}

int fstatvfs(int fd, struct statvfs *buf) {
    TRACE("intercepting fstatvfs(fd=%d, buf=%p)\n", fd, buf)
    
    if FD_NOT_MANAGED(fd) {
        return libc_fstatvfs(fd, buf);
    }
    NOT_IMPLEMENTED("fstatvfs")
}

int fstatvfs64(int fd, struct statvfs64 *buf) {
    TRACE("intercepting fstatvfs64(fd=%d, buf=%p)\n", fd, buf)
    
    if FD_NOT_MANAGED(fd) {
        return libc_fstatvfs64(fd, buf);
    }
    NOT_IMPLEMENTED("fstatvfs64")
}

ssize_t getdelim(char **buf, size_t *bufsiz, int delimiter, FILE *stream) {
    TRACE("intercepting getdelim(buf=%p, bufsiz=%p, delimiter=%d, stream=%p)\n", buf, bufsiz, delimiter, stream)
    
    if STREAM_NOT_MANAGED(stream) {
        return libc_getdelim(buf, bufsiz, delimiter, stream);
    }

    char *ptr, *eptr;


	if (*buf == NULL || *bufsiz == 0) {
		*bufsiz = BUFSIZ;
		if ((*buf = malloc(*bufsiz)) == NULL)
			return -1;
	}

	for (ptr = *buf, eptr = *buf + *bufsiz;;) {
		int c = fgetc(stream);
		if (c == -1) {
			if (feof(stream))
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
    
    if STREAM_NOT_MANAGED(stream) {
        return libc_getline(buf, bufsiz, stream);
    }
    return getdelim(buf, bufsiz, '\n', stream);
}

DIR* opendir(const char* path) {
    TRACE("intercepting opendir(path=%s)\n", path)
    
    if PATH_NOT_MANAGED(path) {
        return libc_opendir(path);
    }
    NOT_IMPLEMENTED("opendir")
}

int feof(FILE *stream) {
    TRACE("intercepting feof(stream=%p)\n", stream)
    
    if STREAM_NOT_MANAGED(stream) {
        return libc_feof(stream);
    }
    return (((stream)->_flags & _IO_EOF_SEEN) != 0);
    // int fd = fileno(stream);
    // off_t cur_off = lseek(fd, 0, SEEK_CUR);
    // if (cur_off == lseek(fd, 0, SEEK_END))
    //     return 1;
    // lseek(fd, cur_off, SEEK_SET);
    // return 0;
}

int ferror(FILE *stream) {
    TRACE("intercepting ferror(stream=%p)\n", stream)
    
    if STREAM_NOT_MANAGED(stream) {
        return libc_ferror(stream);
    }
    
    return ((stream->_flags & _IO_ERR_SEEN) != 0);
}

void clearerr(FILE *stream) {

    if STREAM_NOT_MANAGED(stream) {
        return libc_clearerr(stream);
    }

    stream->_flags &= ~(_IO_ERR_SEEN|_IO_EOF_SEEN);
}

ssize_t getxattr(const char *path, const char *name, void *value,  size_t size) {
    TRACE("intercepting getxattr(path=%s, name=%s, value=%p, size=%d)\n", path, name, value, size)
    
    if PATH_NOT_MANAGED(path) {
        return libc_getxattr(path, name, value, size);
    }
    // hack for CEA computing facility relying on some extended attributes
    // TODO: intercept setxattr to record attributes instead ?
    errno = EOPNOTSUPP;
    return -1;
}