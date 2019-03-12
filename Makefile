BUILDDIR=build

#install dir
PREFIX ?= $(HOME)/opt/pdwfs

all: dirs pdwfslibc scripts

dirs:
	mkdir -p $(BUILDDIR)

pdwfslibc: pdwfsgo
	make -C src/c

pdwfsgo:
	make -C src/go

.PHONY: scripts
scripts: scripts/pdwfs 
	mkdir -p $(BUILDDIR)/bin
	install scripts/pdwfs $(BUILDDIR)/bin/	 

test: scripts pdwfslibc
	make -C src/go test
	make -C src/c test

clean:
	rm -rf $(BUILDDIR)
	rm -rf dist

install: pdwfslibc
	install -d $(PREFIX)/lib $(PREFIX)/bin
	install $(BUILDDIR)/lib/libpdwfs_go.so $(PREFIX)/lib
	install $(BUILDDIR)/lib/pdwfs.so $(PREFIX)/lib
	install scripts/pdwfs $(PREFIX)/bin
	chmod +x $(PREFIX)/bin/*

tag:
	git tag -a $(TAG) -m "Version $(TAG)"

dist:
	scripts/create_tarballs.sh
