VERSION=$(shell git describe 2>/dev/null || echo "develop")
BUILD=$(shell git rev-parse --short HEAD 2>/dev/null || echo "develop")
PLATFORMS := windows linux darwin
GOOS = $(word 1, $@)
BINARY := turing-screen
LDFLAGS=-ldflags "-s -w -X=github.com/alexwbaule/turing-screen/internal/application/logger.Version=$(VERSION) -X=github.com/alexwbaule/turing-screen/internal/application/logger.Build=$(BUILD)"


build:
	mkdir -p bin/
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/$(BINARY) -v cmd/$(BINARY)/main.go

.PHONY: $(PLATFORMS)

$(PLATFORMS): build
	mkdir -p bin/
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-$(GOOS)-amd64 cmd/$(BINARY)/main.go
	chmod 755 bin/$(BINARY)-$(GOOS)-amd64

.PHONY: release
release: windows linux darwin