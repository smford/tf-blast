#!/usr/bin/env bash
set -euo pipefail

# build-release.sh: Cross-compiles tf-blast for all major OS/Arch platforms,
# creates distribution archives (.tar.gz and .zip), and generates SHA256 checksums.

VERSION="${1:-1.0.0}"
VERSION_NO_V="${VERSION#v}"
DIST_DIR="dist"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

LDFLAGS="-s -w -X main.Version=${VERSION}"

TARGETS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

echo "==> Building tf-blast release artifacts for ${VERSION}..."

for TARGET in "${TARGETS[@]}"; do
  GOOS="${TARGET%/*}"
  GOARCH="${TARGET#*/}"
  EXT=""
  if [ "$GOOS" = "windows" ]; then
    EXT=".exe"
  fi

  BIN_NAME="tf-blast${EXT}"
  TMP_BUILD_DIR="${DIST_DIR}/tmp_${GOOS}_${GOARCH}"
  mkdir -p "$TMP_BUILD_DIR"

  echo "  --> Compiling ${GOOS}/${GOARCH}..."
  GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags="$LDFLAGS" -o "${TMP_BUILD_DIR}/${BIN_NAME}" ./cmd/tf-blast

  # Copy LICENSE and README into archive folder
  cp LICENSE "$TMP_BUILD_DIR/" 2>/dev/null || true
  cp README.md "$TMP_BUILD_DIR/" 2>/dev/null || true

  ARCHIVE_NAME="tf-blast_${VERSION_NO_V}_${GOOS}_${GOARCH}"

  if [ "$GOOS" = "windows" ]; then
    (cd "$TMP_BUILD_DIR" && zip -q -r "../${ARCHIVE_NAME}.zip" ./*)
    cp "${DIST_DIR}/${ARCHIVE_NAME}.zip" "${DIST_DIR}/tf-blast_${GOOS}_${GOARCH}.zip"
    cp "${DIST_DIR}/${ARCHIVE_NAME}.zip" "${DIST_DIR}/tf-blast-${GOOS}-${GOARCH}.zip"
  else
    tar -czf "${DIST_DIR}/${ARCHIVE_NAME}.tar.gz" -C "$TMP_BUILD_DIR" .
    cp "${DIST_DIR}/${ARCHIVE_NAME}.tar.gz" "${DIST_DIR}/tf-blast_${GOOS}_${GOARCH}.tar.gz"
    cp "${DIST_DIR}/${ARCHIVE_NAME}.tar.gz" "${DIST_DIR}/tf-blast-${GOOS}-${GOARCH}.tar.gz"
  fi

  rm -rf "$TMP_BUILD_DIR"
done

echo "==> Generating SHA256 checksums..."
cd "$DIST_DIR"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum tf-blast_* > "tf-blast_${VERSION_NO_V}_checksums.txt"
else
  shasum -a 256 tf-blast_* > "tf-blast_${VERSION_NO_V}_checksums.txt"
fi
cd ..

echo "==> Release build complete! Artifacts in ${DIST_DIR}/:"
ls -lh "$DIST_DIR"
