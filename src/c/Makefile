CFLAGS = -std=c99 -g -O2 -Wall -Werror $(shell pkg-config --cflags glib-2.0)
LDFLAGS = $(shell pkg-config --libs glib-2.0)

BUILDDIR=../../build
SCRIPTSDIR=../../scripts

C_TESTS = $(patsubst tests/%.c, $(BUILDDIR)/tests/%, $(wildcard tests/*.c))

all: dirs $(BUILDDIR)/lib/pdwfs.so $(C_TESTS)

dirs:
	mkdir -p $(BUILDDIR)/tests

$(BUILDDIR)/%.o : %.c
	gcc $(CFLAGS) -fPIC -I$(BUILDDIR)/include -c $< -o $@  

$(BUILDDIR)/lib/pdwfs.so: $(BUILDDIR)/lib/libpdwfs_go.so $(BUILDDIR)/pdwfs.o $(BUILDDIR)/libc.o $(BUILDDIR)/utils.o
	gcc $(LDFLAGS) -fPIC -shared -Wl,-rpath=\$$ORIGIN/../lib -o $@ $(BUILDDIR)/pdwfs.o $(BUILDDIR)/libc.o $(BUILDDIR)/utils.o -ldl -lpdwfs_go -L$(BUILDDIR)/lib

$(BUILDDIR)/tests/%: $(BUILDDIR)/tests/%.o
	gcc $(LDFLAGS) -o $@ $^

$(BUILDDIR)/tests/%.o : tests/%.c
	gcc $(CFLAGS) -c $< -o $@  

clean:
	rm -f $(BUILDDIR)/pdwfs.o
	rm -f $(BUILDDIR)/lib/pdwfs.so
	rm -rf $(BUILDDIR)/tests

test-runs := $(addsuffix .test-disk,$(C_TESTS)) $(addsuffix .test-pdwfs,$(C_TESTS))

test: all $(test-runs)

%.test-disk:  # execute tests on disk
	$(subst .test-disk,,$@)

%.test-pdwfs: $(BUILDDIR)/lib/pdwfs.so  # execute tests through pdwfs
	$(BUILDDIR)/bin/pdwfs -p . -- $(subst .test-pdwfs,,$@)
	@redis-cli flushall > /dev/null
