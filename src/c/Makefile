CFLAGS = -std=c99 -g -O2 -Wall -Werror
LDFLAGS = 

BUILDDIR=../../build
SCRIPTSDIR=../../scripts

C_TESTS = $(patsubst tests/%.c, $(BUILDDIR)/tests/%, $(wildcard tests/*.c))

all: dirs $(BUILDDIR)/lib/pdwfs.so $(C_TESTS)

dirs:
	mkdir -p $(BUILDDIR)/tests

$(BUILDDIR)/pdwfs.o : pdwfs.c
	gcc $(CFLAGS) -fPIC -I$(BUILDDIR)/include -c $< -o $@  

$(BUILDDIR)/lib/pdwfs.so: $(BUILDDIR)/lib/libpdwfs_go.so $(BUILDDIR)/pdwfs.o 
	gcc $(LDFLAGS) -fPIC -shared -Wl,-rpath=\$$ORIGIN/../lib -o $@ $(BUILDDIR)/pdwfs.o -ldl -lpdwfs_go -L$(BUILDDIR)/lib

$(BUILDDIR)/tests/%: $(BUILDDIR)/tests/%.o
	gcc $(LDFLAGS) -o $@ $^

$(BUILDDIR)/tests/%.o : tests/%.c
	gcc $(CFLAGS) -c $< -o $@  

clean:
	rm -f $(BUILDDIR)/pdwfs.o
	rm -f $(BUILDDIR)/lib/pdwfs.so
	rm -rf $(BUILDDIR)/tests

test-runs := $(addsuffix .test-run,$(C_TESTS))

test: all $(test-runs)

%.test-run: $(BUILDDIR)/lib/pdwfs.so
	$(BUILDDIR)/bin/pdwfs -p . -- $(subst .test-run,,$@)
	@redis-cli flushall > /dev/null
