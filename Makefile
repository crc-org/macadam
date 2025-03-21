.PHONY: all build check clean cross test

GIT_VERSION ?= $(shell git describe --always --dirty)
VERSION_LDFLAGS=-X github.com/crc-org/macadam/pkg/cmdline.gitVersion=$(GIT_VERSION)
# opengpg and btrfs support are used by github.com/containers/image and
# github.com/containers/storage when container images are fetched.
# These require external C libraries and their headers, it's simpler to disable
# them for now. Hopefully podman-machine does not use these features.
BUILDTAGS=containers_image_openpgp exclude_graphdriver_btrfs btrfs_noversion
WINBUILDTAGS=remote ${BUILDTAGS}  

DEFAULT_GOOS=$(shell go env GOOS)
DEFAULT_GOARCH=$(shell go env GOARCH)

all: build

build: bin/macadam-$(DEFAULT_GOOS)-$(DEFAULT_GOARCH)

TOOLS_DIR := tools
include tools/tools.mk

cross: bin/macadam-darwin-amd64 bin/macadam-darwin-arm64 bin/macadam-linux-amd64 bin/macadam-linux-arm64 bin/macadam-windows-amd64

check: lint vendorcheck test

test:
	@go test -tags "$(BUILDTAGS)" -v ./pkg/...

clean:
	@rm -rf bin

bin/macadam-darwin-amd64: GOOS=darwin
bin/macadam-darwin-amd64: GOARCH=amd64
bin/macadam-darwin-amd64: force-build
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -tags "$(BUILDTAGS)" -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH) ./cmd/macadam

bin/macadam-darwin-arm64: GOOS=darwin
bin/macadam-darwin-arm64: GOARCH=arm64
bin/macadam-darwin-arm64: force-build
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -tags "$(BUILDTAGS)" -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH) ./cmd/macadam

bin/macadam-linux-amd64: GOOS=linux
bin/macadam-linux-amd64: GOARCH=amd64
bin/macadam-linux-amd64: force-build
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -tags "$(BUILDTAGS)" -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH) ./cmd/macadam

bin/macadam-linux-arm64: GOOS=linux
bin/macadam-linux-arm64: GOARCH=arm64
bin/macadam-linux-arm64: force-build
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -tags "$(BUILDTAGS)" -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH) ./cmd/macadam

bin/macadam-windows-amd64: GOOS=windows
bin/macadam-windows-amd64: GOARCH=amd64
bin/macadam-windows-amd64: force-build
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -tags "$(WINBUILDTAGS)" -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH).exe ./cmd/macadam

.PHONY: lint
lint: $(TOOLS_BINDIR)/golangci-lint
	@"$(TOOLS_BINDIR)"/golangci-lint run

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

.PHONY: vendorcheck
vendorcheck:
	./build-scripts/verify-vendor.sh

# the go compiler is doing a good job at not rebuilding unchanged files
# this phony target ensures bin/macadam-* are always considered out of date
# and rebuilt. If the code was unchanged, go won't rebuild anything so that's
# fast. Forcing the rebuild ensure we rebuild when needed, ie when the source code
# changed, without adding explicit dependencies to the go files/go.mod
.PHONY: force-build
force-build:

