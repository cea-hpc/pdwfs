BUILDDIR=build

#install dir
PREFIX ?= /usr/local

all: dirs pdwfslibc tools

dirs:
	mkdir -p $(BUILDDIR)

pdwfslibc: pdwfsgo
	make -C src/c

pdwfsgo:
	make -C src/go

.PHONY: tools
tools: tools/pdwfs 
	mkdir -p $(BUILDDIR)/bin
	install tools/pdwfs $(BUILDDIR)/bin/
	install tools/pdwfs-slurm $(BUILDDIR)/bin/
	install tools/redis.srun $(BUILDDIR)/bin/
	#cd tools/pv && ./configure && make  # if pv needs rebuild
	install tools/pv/pv $(BUILDDIR)/bin/


test: tools pdwfslibc
	make -C src/go test
	make -C src/c test

clean:
	rm -rf $(BUILDDIR)
	rm -rf dist

install: pdwfslibc
	install -d $(PREFIX)/lib $(PREFIX)/bin
	install $(BUILDDIR)/lib/libpdwfs_go.so $(PREFIX)/lib
	install $(BUILDDIR)/lib/pdwfs.so $(PREFIX)/lib
	install tools/pdwfs $(PREFIX)/bin
	install tools/pdwfs-slurm $(PREFIX)/bin
	install tools/redis.srun $(PREFIX)/bin
	chmod +x $(PREFIX)/bin/*

tag:
	git tag -a $(TAG) -m "Version $(TAG)"

dist:
	tools/create_tarballs.sh

bench:
	make -C examples/ior_benchmark bench
