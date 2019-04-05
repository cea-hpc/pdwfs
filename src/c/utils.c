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

#include "utils.h"

char* abspath (const char *name) {
    // implementation of realpath with resolved=NULL, without links, and no need for file to exist on fs (from glibc)

    char *rpath, *dest = NULL;
    const char *start, *end, *rpath_limit;

    if (name == NULL) {
        errno = EINVAL;
        return NULL;
    }

    if (name[0] == '\0') {
        errno = ENOENT;
        return NULL;
    }
    
    rpath = malloc(PATH_MAX);
    if (rpath == NULL)
        return NULL;
    rpath_limit = rpath + PATH_MAX;
    
    if (name[0] != '/') {
        if (!getcwd (rpath, PATH_MAX)) {
	        free(rpath);
	        return NULL;
	    }
        dest = memchr(rpath, '\0', (size_t)-1);
    }
    else {
        rpath[0] = '/';
        dest = rpath + 1;
    }
    for (start = end = name; *start; start = end) {

        /* Skip sequence of multiple path-separators.  */
        while (*start == '/')
            ++start;

        /* Find end of path component.  */
        for (end = start; *end && *end != '/'; ++end)
            /* Nothing.  */;

        if (end - start == 0)
            break;
        else if (end - start == 1 && start[0] == '.')
            /* nothing */;
        else if (end - start == 2 && start[0] == '.' && start[1] == '.') {
            /* Back up to previous component, ignore if at root already.  */
            if (dest > rpath + 1)
                while ((--dest)[-1] != '/');
        } else {
            size_t new_size;

            if (dest[-1] != '/')
                *dest++ = '/';

            if (dest + (end - start) >= rpath_limit) {
                ptrdiff_t dest_offset = dest - rpath;
                char *new_rpath;

                new_size = rpath_limit - rpath;
                if (end - start + 1 > PATH_MAX)
                    new_size += end - start + 1;
                else
                    new_size += PATH_MAX;
                new_rpath = (char *) realloc (rpath, new_size);
                if (new_rpath == NULL) {
                    free(rpath);
                    return NULL;
                }
                rpath = new_rpath;
                rpath_limit = rpath + new_size;

                dest = rpath + dest_offset;
            }

            dest = mempcpy(dest, start, end - start);
            *dest = '\0';
        }
    }
    if (dest > rpath + 1 && dest[-1] == '/')
        --dest;
    *dest = '\0';
    return rpath;
}
