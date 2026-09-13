#!/usr/bin/env bash
set -euo pipefail

version="0.0.0+pre-001"
0.0.0+pre-001
BASE_URL="https://github.com/iilei/drooster-balance/releases/download/v${version}"
REPO="example-repo-local"
ARTIFACTORY="http://localhost:8081/artifactory"
CRED="admin:password"

FILES=(
  "checksums.txt"
  "checksums.txt.sig"
  "drooster-balance_${version}_darwin_universal2.tar.gz"
  "drooster-balance_${version}_darwin_universal2.tar.gz.sha256"
  "drooster-balance_${version}_darwin_universal2.tar.gz.sig"
  "drooster-balance_${version}_linux_arm64.tar.gz"
  "drooster-balance_${version}_linux_arm64.tar.gz.sha256"
  "drooster-balance_${version}_linux_arm64.tar.gz.sig"
  "drooster-balance_${version}_linux_armv7.tar.gz"
  "drooster-balance_${version}_linux_armv7.tar.gz.sha256"
  "drooster-balance_${version}_linux_armv7.tar.gz.sig"
  "drooster-balance_${version}_linux_x86_64.tar.gz"
  "drooster-balance_${version}_linux_x86_64.tar.gz.sha256"
  "drooster-balance_${version}_linux_x86_64.tar.gz.sig"
  "drooster-balance_${version}_windows_amd64.zip"
  "drooster-balance_${version}_windows_amd64.zip.sha256"
  "drooster-balance_${version}_windows_amd64.zip.sig"
  "drooster-balance_${version}_windows_arm64.zip"
  "drooster-balance_${version}_windows_arm64.zip.sha256"
  "drooster-balance_${version}_windows_arm64.zip.sig"
)

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
echo $TMPDIR

for f in "${FILES[@]}"; do
  echo "==> $f"
  echo "curl -sfL -o" "$TMPDIR/$f" "$BASE_URL/$f"
  # Download
  curl -sfL -o "$TMPDIR/$f" "$BASE_URL/$f"
  # Push to Artifactory, preserving relative path under a namespace
  curl -sf -u "$CRED" -T "$TMPDIR/$f" \
    "$ARTIFACTORY/$REPO/drooster-balance/${version}/$f"
done

echo "All done."
