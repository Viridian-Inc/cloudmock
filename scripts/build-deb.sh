#!/bin/bash
set -euo pipefail

# Usage: ./scripts/build-deb.sh <version> <arch> <binary-path>
# Example: ./scripts/build-deb.sh 1.0.3 amd64 dist/cloudmock-linux-amd64

VERSION="${1:?Usage: build-deb.sh <version> <arch> <binary-path>}"
ARCH="${2:?}"
BINARY="${3:?}"

PKG="cloudmock_${VERSION}_${ARCH}"
mkdir -p "${PKG}/DEBIAN"
mkdir -p "${PKG}/usr/local/bin"

cp "${BINARY}" "${PKG}/usr/local/bin/cloudmock"
chmod 755 "${PKG}/usr/local/bin/cloudmock"

cat > "${PKG}/DEBIAN/control" <<EOF
Package: cloudmock
Version: ${VERSION}
Section: devel
Priority: optional
Architecture: ${ARCH}
Maintainer: Viridian Inc <eng@viri.app>
Homepage: https://cloudmock.io
Description: Local AWS emulation — 98 services, one binary, built-in devtools
 CloudMock emulates 98 AWS services locally. Point your AWS SDK at
 localhost:4566 and develop without a cloud account.
EOF

dpkg-deb --build "${PKG}"
echo "Built ${PKG}.deb"
