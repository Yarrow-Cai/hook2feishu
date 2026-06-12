BINARY=hook2feishu
LDFLAGS=-ldflags="-s -w"

.PHONY: build release clean test vet

build:
	go build $(LDFLAGS) -o $(BINARY) .

release: clean
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY).exe .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)_darwin_amd64 .
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)_darwin_arm64 .
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)_linux_amd64 .
	@echo "Release binaries in dist/"

clean:
	rm -rf dist/ $(BINARY) $(BINARY).exe

test:
	go test ./...

vet:
	go vet ./...
