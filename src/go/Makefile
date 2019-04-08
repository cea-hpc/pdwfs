BUILDDIR=../../build
SCRIPTSDIR=../../scripts

GO_FILES = $(wildcard config/*.go) $(wildcard redisfs/*.go) 

$(BUILDDIR)/lib/libpdwfs_go.so: $(GO_FILES)
	go build -mod=vendor -o $@ -buildmode=c-shared
	mkdir -p $(BUILDDIR)/include && mv $(BUILDDIR)/lib/libpdwfs_go.h $(BUILDDIR)/include

clean:
	rm -f $(BUILDDIR)/lib/libpdwfs_go.so
	rm -f $(BUILDDIR)/include/libpdwfs_go.h

test:
	go vet -mod=vendor ./...
	# count=1 deactivate test results caching
	go test -mod=vendor -failfast -race -count=1 ./...  
