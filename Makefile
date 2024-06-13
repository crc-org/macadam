.PHONY: all build clean cross test

GIT_VERSION ?= $(shell git describe --always --dirty)
VERSION_LDFLAGS=-X github.com/crc-org/macadam/pkg/cmdline.gitVersion=$(GIT_VERSION)

DEFAULT_GOOS=$(shell go env GOOS)
DEFAULT_GOARCH=$(shell go env GOARCH)

all: build

build: bin/macadam-$(DEFAULT_GOOS)-$(DEFAULT_GOARCH)

TOOLS_DIR := tools
include tools/tools.mk

cross: bin/macadam-darwin-amd64 bin/macadam-darwin-arm64 bin/macadam-linux-amd64 bin/macadam-linux-arm64 bin/macadam-windows-amd64

test: build
	@go test -v ./pkg/...
	@go test -v -timeout 20m ./test

clean:
	rm -rf bin

bin/macadam-darwin-amd64: GOOS=darwin
bin/macadam-darwin-amd64: GOARCH=amd64
bin/macadam-darwin-amd64: force-build
	go build -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH) ./cmd/macadam

bin/macadam-darwin-arm64: GOOS=darwin
bin/macadam-darwin-arm64: GOARCH=arm64
bin/macadam-darwin-arm64: force-build
	go build -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH) ./cmd/macadam

bin/macadam-linux-amd64: GOOS=linux
bin/macadam-linux-amd64: GOARCH=amd64
bin/macadam-linux-amd64: force-build
	go build -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH) ./cmd/macadam

bin/macadam-linux-arm64: GOOS=linux
bin/macadam-linux-arm64: GOARCH=arm64
bin/macadam-linux-arm64: force-build
	go build -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH) ./cmd/macadam

bin/macadam-windows-amd64: GOOS=windows
bin/macadam-windows-amd64: GOARCH=amd64
bin/macadam-windows-amd64: force-build
	go build -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH) ./cmd/macadam

# the go compiler is doing a good job at not rebuilding unchanged files
# this phony target ensures bin/macadam-* are always considered out of date
# and rebuilt. If the code was unchanged, go won't rebuild anything so that's
# fast. Forcing the rebuild ensure we rebuild when needed, ie when the source code
# changed, without adding explicit dependencies to the go files/go.mod
.PHONY: force-build
force-build:

.PHONY: lint
lint: $(TOOLS_BINDIR)/golangci-lint
	"$(TOOLS_BINDIR)"/golangci-lint run
