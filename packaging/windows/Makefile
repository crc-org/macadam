# Derive an MSI-compatible major.minor.build version from git describe.
# Strips the leading "v", splits on "-" to isolate the tag (e.g. "0.3.2"
# from "0.3.2-5-gabcdef"), validates it with PowerShell's [version] type,
# and clamps each component to MSI limits (major/minor 0-255, build 0-65535).
# Falls back to "0.0.0" when no valid semver tag is reachable.
MACADAM_VERSION := $(shell powershell -NoProfile -Command "\
  $$tag = (git describe --tags --always --abbrev=0) -replace '^v',''; \
  if ($$tag -as [version]) { \
    $$v = [version]$$tag; \
    '{0}.{1}.{2}' -f [Math]::Min($$v.Major, 255), \
                     [Math]::Min($$v.Minor, 255), \
                     [Math]::Min([Math]::Max($$v.Build, 0), 65535) \
  } else { '0.0.0' }")
GVPROXY_VERSION ?= 0.8.9
GVPROXY_RELEASE_URL ?= https://github.com/containers/gvisor-tap-vsock/releases/download/v$(GVPROXY_VERSION)/gvproxy-windows.exe
PACKAGE_DIR ?= msi
BUILD_DIR ?= out
OBJECT_DIR ?= obj

$(PACKAGE_DIR)/gvproxy.exe:
	@powershell -NoProfile -Command "if (-not (Test-Path '$(PACKAGE_DIR)')) { New-Item -ItemType Directory -Path '$(PACKAGE_DIR)' | Out-Null }"
	@powershell -NoProfile -Command "Invoke-WebRequest -Uri '$(GVPROXY_RELEASE_URL)' -OutFile '$(PACKAGE_DIR)/gvproxy.exe' -UseBasicParsing"

$(PACKAGE_DIR)/macadam.exe:
	@powershell -NoProfile -Command "if (-not (Test-Path '$(PACKAGE_DIR)')) { New-Item -ItemType Directory -Path '$(PACKAGE_DIR)' | Out-Null }"
	@powershell -NoProfile -Command "Set-Location '../..'; if (-not (Test-Path 'bin')) { New-Item -ItemType Directory -Path 'bin' | Out-Null }; $$env:GOOS='windows'; $$env:GOARCH='amd64'; go build -tags 'remote containers_image_openpgp exclude_graphdriver_btrfs btrfs_noversion' -ldflags '-X github.com/crc-org/macadam/pkg/cmdline.gitVersion=$(MACADAM_VERSION)' -o bin/macadam-windows-amd64.exe ./cmd/macadam"
	@powershell -NoProfile -Command "Copy-Item '../../bin/macadam-windows-amd64.exe' '$(PACKAGE_DIR)/macadam.exe' -Force"

.PHONY: msidir clean
msidir:
	$(MAKE) clean
	$(MAKE) $(PACKAGE_DIR)/gvproxy.exe $(PACKAGE_DIR)/macadam.exe

$(BUILD_DIR)/macadam-windows-amd64.msi: msidir
	@powershell -NoProfile -Command "if (-not (Test-Path '$(BUILD_DIR)')) { New-Item -ItemType Directory -Path '$(BUILD_DIR)' | Out-Null }"
	dotnet build macadam-installer.wixproj --property:DefineConstants="Version=$(MACADAM_VERSION)" --output $(BUILD_DIR)

clean:
	@powershell -NoProfile -Command "if (Test-Path '$(PACKAGE_DIR)') { Remove-Item -Recurse -Force '$(PACKAGE_DIR)' }"
	@powershell -NoProfile -Command "if (Test-Path '$(BUILD_DIR)') { Remove-Item -Recurse -Force '$(BUILD_DIR)' }"
	@powershell -NoProfile -Command "if (Test-Path '$(OBJECT_DIR)') { Remove-Item -Recurse -Force '$(OBJECT_DIR)' }"
