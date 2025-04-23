#!/bin/bash
set -euxo pipefail

BASEDIR=$(dirname "$0")

if [ -z "$1" ]; then
  echo "Usage: $0 <output>" >&2
  exit 1
fi
OUTPUT=$1

CODESIGN_IDENTITY=${CODESIGN_IDENTITY:--}
PRODUCTSIGN_IDENTITY=${PRODUCTSIGN_IDENTITY:-mock}
NO_CODESIGN=${NO_CODESIGN:-0}

binDir="${BASEDIR}/root/macadam/bin"

version=$(cat "${BASEDIR}/VERSION")

function build_fat(){
    echo "Creating universal binary"
    lipo -create -output "${binDir}/macadam" "${binDir}/macadam-darwin-arm64" "${binDir}/macadam-darwin-amd64"
    rm "${binDir}/macadam-darwin-arm64" "${binDir}/macadam-darwin-amd64"
}

function sign() {
  local opts=""
  entitlements="${BASEDIR}/$(basename "$1").entitlements"
  if [ -f "${entitlements}" ]; then
      opts="--entitlements ${entitlements}"
  fi
  codesign --sign "${CODESIGN_IDENTITY}" --options runtime --timestamp --force ${opts} "$1"
}

build_fat

sign "${binDir}/macadam"
sign "${binDir}/gvproxy"
sign "${binDir}/vfkit"

pkgbuild --identifier com.redhat.macadam --version ${version} \
  --scripts "${BASEDIR}/scripts" \
  --root "${BASEDIR}/root" \
  --install-location /opt \
  --component-plist "${BASEDIR}/component.plist" \
  "${OUTPUT}/macadam.pkg"

productbuild --distribution "${BASEDIR}/Distribution" \
  --resources "${BASEDIR}/Resources" \
  --package-path "${OUTPUT}" \
  "${OUTPUT}/macadam-unsigned.pkg"
rm -f "${OUTPUT}/macadam.pkg"

if [ ! "${NO_CODESIGN}" -eq "1" ]; then
  productsign --timestamp --sign "${PRODUCTSIGN_IDENTITY}" "${OUTPUT}/macadam-unsigned.pkg" "${OUTPUT}/macadam-installer-macos-universal.pkg"
else
  mv "${OUTPUT}/macadam-unsigned.pkg" "${OUTPUT}/macadam-installer-macos-universal.pkg"
fi
