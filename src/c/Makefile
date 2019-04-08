CFLAGS = -std=c99 -g -O2 -Wall -Werror $(shell pkg-config --cflags glib-2.0)
LDFLAGS = $(shell pkg-config --libs glib-2.0)

BUILDDIR=../../build
SCRIPTSDIR=../../scripts

C_OBJ = $(BUILDDIR)/pdwfs.o $(BUILDDIR)/libc.o $(BUILDDIR)/utils.o

all: dirs $(BUILDDIR)/lib/pdwfs.so

dirs:
	mkdir -p $(BUILDDIR)/tests

$(BUILDDIR)/%.o : %.c
	gcc $(CFLAGS) -fPIC -I$(BUILDDIR)/include -c $< -o $@  

$(BUILDDIR)/lib/pdwfs.so: $(BUILDDIR)/lib/libpdwfs_go.so $(C_OBJ)
	gcc $(LDFLAGS) -fPIC -shared -Wl,-rpath=\$$ORIGIN/../lib -o $@ $(C_OBJ) -ldl -lpdwfs_go -L$(BUILDDIR)/lib

clean:
	rm -f $(BUILDDIR)/pdwfs.o
	rm -f $(BUILDDIR)/lib/pdwfs.so
	make -C tests clean

test: all
	make -C tests
	@echo "*****************************"
	@echo "* Execute testsuite on disk *"
	@echo "*****************************"
	- $(BUILDDIR)/tests/testsuite
	@echo "***********************************"
	@echo "* Execute testsuite through pdwfs *"
	@echo "***********************************"
	redis-server --daemonize yes --save ""
	- $(BUILDDIR)/bin/pdwfs -p . -- $(BUILDDIR)/tests/testsuite
	redis-cli shutdown   
