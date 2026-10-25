#!/usr/bin/env bash
set -euo pipefail

# Build script for Delvo multi-platform release artifacts
# Produces versioned tar.gz and zip archives with SHA-256 checksums.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

VERSION="${1:-0.1.0}"
VERSION="${VERSION#v}" # Strip leading 'v' if provided

DIST_DIR="${REPO_ROOT}/dist"
TMP_DIR="${DIST_DIR}/.tmp_build"

echo "=================================================="
echo " Building Delvo release artifacts for v${VERSION}"
echo "=================================================="

rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}" "${TMP_DIR}"

cd "${REPO_ROOT}"

# Matrix: OS ARCH ARCHIVE_FORMAT
TARGETS=(
  "android arm64 tar.gz"
  "linux amd64 tar.gz"
  "linux arm64 tar.gz"
  "darwin amd64 tar.gz"
  "darwin arm64 tar.gz"
  "windows amd64 zip"
)

LDFLAGS="-s -w -X main.Version=${VERSION}"

for target in "${TARGETS[@]}"; do
  read -r os arch format <<< "${target}"
  archive_name="delvo-${os}-${arch}.${format}"
  stage_dir="${TMP_DIR}/${os}-${arch}"
  mkdir -p "${stage_dir}"

  bin_name="dvo"
  if [ "${os}" = "windows" ]; then
    bin_name="dvo.exe"
  fi

  echo "==> Building ${os}/${arch} -> ${archive_name}..."
  CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" go build \
    -ldflags "${LDFLAGS}" \
    -trimpath \
    -o "${stage_dir}/${bin_name}" \
    "${REPO_ROOT}/cmd/dvo"

  # Copy documentation and license into bundle
  cp "${REPO_ROOT}/README.md" "${stage_dir}/README.md"
  cp "${REPO_ROOT}/LICENSE" "${stage_dir}/LICENSE"

  # Package into archive
  if [ "${format}" = "tar.gz" ]; then
    tar -czf "${DIST_DIR}/${archive_name}" -C "${stage_dir}" "${bin_name}" "README.md" "LICENSE"
  elif [ "${format}" = "zip" ]; then
    if command -v zip >/dev/null 2>&1; then
      (cd "${stage_dir}" && zip -q -9 "${DIST_DIR}/${archive_name}" "${bin_name}" "README.md" "LICENSE")
    elif command -v python3 >/dev/null 2>&1; then
      python3 -c "
import zipfile, os
stage = '${stage_dir}'
archive = '${DIST_DIR}/${archive_name}'
with zipfile.ZipFile(archive, 'w', zipfile.ZIP_DEFLATED) as zf:
    for f in ['${bin_name}', 'README.md', 'LICENSE']:
        zf.write(os.path.join(stage, f), arcname=f)
"
    else
      echo "Error: Neither zip nor python3 is available to create zip archive." >&2
      exit 1
    fi
  fi

  echo "    Done: ${DIST_DIR}/${archive_name}"
done

# Cleanup temporary staging folder
rm -rf "${TMP_DIR}"

# Generate SHA-256 Checksums
echo "==> Generating SHA-256 checksums..."
cd "${DIST_DIR}"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum delvo-* > checksums.txt
elif command -v shasum >/dev/null 2>&1; then
  shasum -a 256 delvo-* > checksums.txt
else
  python3 -c "
import hashlib, glob
with open('checksums.txt', 'w') as out:
    for f in sorted(glob.glob('delvo-*')):
        with open(f, 'rb') as fp:
            h = hashlib.sha256(fp.read()).hexdigest()
        out.write(f'{h}  {f}\n')
"
fi

echo "=================================================="
echo " Release artifacts generated in ${DIST_DIR}:"
ls -lh "${DIST_DIR}"
echo ""
echo " Checksums:"
cat "${DIST_DIR}/checksums.txt"
echo "=================================================="
