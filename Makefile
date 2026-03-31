.PHONY: all build check clean cross test

GIT_VERSION ?= $(shell git describe --always --dirty)
VERSION_LDFLAGS=-X github.com/crc-org/macadam/pkg/cmdline.gitVersion=$(GIT_VERSION)
MACADAM_LDFLAGS = \
    $(VERSION_LDFLAGS) \
    -X go.podman.io/common/pkg/config.additionalHelperBinariesDir=$(HELPER_BINARIES_DIR)
# opengpg and btrfs support are used by github.com/containers/image and
# github.com/containers/storage when container images are fetched.
# These require external C libraries and their headers, it's simpler to disable
# them for now. Hopefully podman-machine does not use these features.
BUILDTAGS=remote containers_image_openpgp exclude_graphdriver_btrfs btrfs_noversion

DEFAULT_GOOS=$(shell go env GOOS)
DEFAULT_GOARCH=$(shell go env GOARCH)

all: build

build: bin/macadam-$(DEFAULT_GOOS)-$(DEFAULT_GOARCH)

TOOLS_DIR := tools
include tools/tools.mk

cross-non-darwin: bin/macadam-linux-amd64 bin/macadam-linux-arm64 bin/macadam-windows-amd64 bin/macadam-windows-arm64

cross: cross-non-darwin bin/macadam-darwin-amd64 bin/macadam-darwin-arm64

check: cross-lint vendorcheck test

test:
	@go test -tags "$(BUILDTAGS)" -v ./pkg/...

e2e:
ifndef IMAGE
IMAGE = --image=""
endif
ifndef BINARY
BINARY = --macadam-binary=""
endif
ifndef GINKGO_TAGS
GINKGO_LABELS = "$(DEFAULT_GOOS)"
else
GINKGO_LABELS = "$(DEFAULT_GOOS) && $(GINKGO_TAGS)"
endif

e2e:
	@go test --timeout=90m -tags "$(BUILDTAGS)"  -v ./test/e2e/... $(IMAGE) $(BINARY) --ginkgo.label-filter=$(GINKGO_LABELS)

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
bin/macadam-linux-amd64: HELPER_BINARIES_DIR=/usr/libexec
bin/macadam-linux-amd64: force-build
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -tags "$(BUILDTAGS)" -ldflags "$(MACADAM_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH) ./cmd/macadam

bin/macadam-linux-arm64: GOOS=linux
bin/macadam-linux-arm64: GOARCH=arm64
bin/macadam-linux-arm64: HELPER_BINARIES_DIR=/usr/libexec
bin/macadam-linux-arm64: force-build
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -tags "$(BUILDTAGS)" -ldflags "$(MACADAM_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH) ./cmd/macadam

bin/macadam-windows-amd64: GOOS=windows
bin/macadam-windows-amd64: GOARCH=amd64
bin/macadam-windows-amd64: force-build
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -tags "$(BUILDTAGS)" -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH).exe ./cmd/macadam

bin/macadam-windows-arm64: GOOS=windows
bin/macadam-windows-arm64: GOARCH=arm64
bin/macadam-windows-arm64: force-build
	GOARCH=$(GOARCH) GOOS=$(GOOS) go build -tags "$(BUILDTAGS)" -ldflags "$(VERSION_LDFLAGS)" -o bin/macadam-$(GOOS)-$(GOARCH).exe ./cmd/macadam

.PHONY: lint
lint: $(TOOLS_BINDIR)/golangci-lint
	@"$(TOOLS_BINDIR)"/golangci-lint run

.PHONY: cross-lint
cross-lint:
	GOOS=linux $(MAKE) lint
	GOOS=darwin $(MAKE) lint
	GOOS=windows $(MAKE) lint

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

e2e_builder = GOOS=$(1) GOARCH=$(2) go test -v ./test/e2e/ -tags "$(BUILDTAGS)" -ldflags="$(3)" -c -o bin/$(1)-$(2)/$(4)

build_e2e:
	$(call e2e_builder,$(DEFAULT_GOOS),$(DEFAULT_GOARCH),$(MACADAM_LDFLAGS),e2e.test)

build_e2e_all:
	$(call e2e_builder,linux,amd64,$(MACADAM_LDFLAGS),e2e.test)
	$(call e2e_builder,linux,arm64,$(MACADAM_LDFLAGS),e2e.test)
	$(call e2e_builder,darwin,amd64,$(MACADAM_LDFLAGS),e2e.test)
	$(call e2e_builder,darwin,arm64,$(MACADAM_LDFLAGS),e2e.test)
	$(call e2e_builder,windows,amd64,$(MACADAM_LDFLAGS),e2e.test.exe)

containerized_e2e:
ifndef IMAGE_NAME
IMAGE_NAME = macadam-e2e
endif
ifndef OS
OS = $(DEFAULT_GOOS)
endif
ifndef ARCH
ARCH = $(DEFAULT_GOARCH)
endif

containerized_e2e: clean
	podman build -t ${IMAGE_NAME}:${OS}-${ARCH} -f test/image/Containerfile --build-arg=OS=${OS} --build-arg=ARCH=${ARCH} .
