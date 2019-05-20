BUILDDIR=../../build

GO_FILES = $(shell find . -type f -name '*.go') 

$(BUILDDIR)/lib/libpdwfs_go.so: $(GO_FILES)
	go build -mod=vendor -o $@ -buildmode=c-shared
	mkdir -p $(BUILDDIR)/include && mv $(BUILDDIR)/lib/libpdwfs_go.h $(BUILDDIR)/include

clean:
	rm -f $(BUILDDIR)/lib/libpdwfs_go.so
	rm -f $(BUILDDIR)/include/libpdwfs_go.h

test:
	go vet -mod=vendor ./...
	# count=1 deactivate test results caching
	go test -mod=vendor -failfast -race -count=1 -timeout 10s ./...  

tidy:
	go mod tidy
	go mod vendor
