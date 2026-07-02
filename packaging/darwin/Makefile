SHELL := bash

MACADAM_VERSION = $(shell git describe --always --dirty)
GVPROXY_VERSION ?= 0.8.9
VFKIT_VERSION ?= 0.6.3
GVPROXY_RELEASE_URL ?= https://github.com/containers/gvisor-tap-vsock/releases/download/v$(GVPROXY_VERSION)/gvproxy-darwin
VFKIT_RELEASE_URL ?= https://github.com/crc-org/vfkit/releases/download/v$(VFKIT_VERSION)/vfkit-unsigned
PACKAGE_DIR ?= out/packaging
TMP_BIN ?= tmp-bin
PACKAGE_ROOT ?= root
PKG_NAME := macadam-installer-macos-universal.pkg

.PHONY: pkginstaller packagedir
default: pkginstaller

$(TMP_BIN)/gvproxy:
	mkdir -p $(TMP_BIN)
	cd $(TMP_BIN) && curl -sLo gvproxy $(GVPROXY_RELEASE_URL)

$(TMP_BIN)/vfkit:
	mkdir -p $(TMP_BIN)
	cd $(TMP_BIN) && curl -sLo vfkit $(VFKIT_RELEASE_URL)

$(TMP_BIN)/macadam:
	mkdir -p $(TMP_BIN)
	$(MAKE) -C ../.. bin/macadam-darwin-amd64
	$(MAKE) -C ../.. bin/macadam-darwin-arm64
	cp ../../bin/macadam-darwin-amd64 $(TMP_BIN)/macadam-darwin-amd64
	cp ../../bin/macadam-darwin-arm64 $(TMP_BIN)/macadam-darwin-arm64

packagedir: package_root Distribution welcome.html
	mkdir -p $(PACKAGE_DIR)
	cp -r Resources $(PACKAGE_DIR)/
	cp welcome.html $(PACKAGE_DIR)/Resources/
	cp Distribution $(PACKAGE_DIR)/
	cp -r scripts $(PACKAGE_DIR)/
	cp -r $(PACKAGE_ROOT) $(PACKAGE_DIR)/
	cp package.sh $(PACKAGE_DIR)/
	cd $(PACKAGE_DIR) && pkgbuild --analyze --root ./root component.plist
# Modify echo with an executable like podman
	echo '$(MACADAM_VERSION)' > $(PACKAGE_DIR)/VERSION
	cp vfkit.entitlements $(PACKAGE_DIR)/

package_root: clean-pkgroot $(TMP_BIN)/gvproxy $(TMP_BIN)/vfkit $(TMP_BIN)/macadam
	mkdir -p $(PACKAGE_ROOT)/macadam/bin
	cp $(TMP_BIN)/gvproxy $(PACKAGE_ROOT)/macadam/bin/
	cp $(TMP_BIN)/vfkit $(PACKAGE_ROOT)/macadam/bin/
	cp $(TMP_BIN)/macadam-darwin-amd64 $(PACKAGE_ROOT)/macadam/bin/
	cp $(TMP_BIN)/macadam-darwin-arm64 $(PACKAGE_ROOT)/macadam/bin/
	chmod a+x $(PACKAGE_ROOT)/macadam/bin/*

%: %.in
	@sed -e 's/__VERSION__/'$(MACADAM_VERSION)'/g' $< >$@

pkginstaller: packagedir
	cd $(PACKAGE_DIR) && ./package.sh ..

_notarize: pkginstaller
	xcrun notarytool submit --apple-id $(NOTARIZE_USERNAME) --password $(NOTARIZE_PASSWORD) --team-id=$(NOTARIZE_TEAM) -f json --wait out/$(PKG_NAME)

notarize: _notarize
	xcrun stapler staple out/$(PKG_NAME)

.PHONY: clean clean-pkgroot
clean:
	rm -rf $(TMP_BIN) $(PACKAGE_ROOT) $(PACKAGE_DIR) out Distribution welcome.html ../../test/version/version

clean-pkgroot:
	rm -rf $(PACKAGE_ROOT) $(PACKAGE_DIR) Distribution welcome.html
