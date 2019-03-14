BUILDDIR=../../build
SCRIPTSDIR=../../scripts

$(BUILDDIR)/lib/libpdwfs_go.so:
	go build -mod=vendor -o $@ -buildmode=c-shared
	mkdir -p $(BUILDDIR)/include && mv $(BUILDDIR)/lib/libpdwfs_go.h $(BUILDDIR)/include

clean:
	rm -f $(BUILDDIR)/lib/libpdwfs_go.so
	rm -f $(BUILDDIR)/include/libpdwfs_go.h

test:
	go vet -mod=vendor ./...
	go test -mod=vendor ./...
