#!/usr/bin/env bash
#
# Publishes dokrypt npm packages after GoReleaser has built the binaries.
#
# Usage:
#   ./scripts/npm-publish.sh <version> [--dry-run]
#
# Example:
#   ./scripts/npm-publish.sh 1.0.0
#   ./scripts/npm-publish.sh 1.0.0 --dry-run

set -euo pipefail

VERSION="${1:?Usage: npm-publish.sh <version> [--dry-run]}"
DRY_RUN=""
if [[ "${2:-}" == "--dry-run" ]]; then
  DRY_RUN="--dry-run"
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
NPM="$ROOT/npm"

# GoReleaser output directory → npm package directory mapping
declare -A PLATFORM_MAP=(
  ["dokrypt_linux_amd64_v1"]="@dokrypt/linux-x64"
  ["dokrypt_linux_arm64_v8.0"]="@dokrypt/linux-arm64"
  ["dokrypt_darwin_amd64_v1"]="@dokrypt/darwin-x64"
  ["dokrypt_darwin_arm64_v8.0"]="@dokrypt/darwin-arm64"
  ["dokrypt_windows_amd64_v1"]="@dokrypt/win32-x64"
)

# Binary name per platform
declare -A BINARY_NAME=(
  ["@dokrypt/linux-x64"]="dokrypt"
  ["@dokrypt/linux-arm64"]="dokrypt"
  ["@dokrypt/darwin-x64"]="dokrypt"
  ["@dokrypt/darwin-arm64"]="dokrypt"
  ["@dokrypt/win32-x64"]="dokrypt.exe"
)

echo "Publishing dokrypt v${VERSION} to npm"
echo ""

# --- Step 1: Copy binaries into npm package directories ---
echo "Copying binaries from dist/..."
for goreleaser_dir in "${!PLATFORM_MAP[@]}"; do
  pkg="${PLATFORM_MAP[$goreleaser_dir]}"
  bin="${BINARY_NAME[$pkg]}"
  src="$DIST/$goreleaser_dir/$bin"
  dest="$NPM/$pkg/bin/$bin"

  if [[ ! -f "$src" ]]; then
    echo "  SKIP $pkg (binary not found: $src)"
    continue
  fi

  cp "$src" "$dest"
  chmod +x "$dest"
  echo "  OK   $pkg <- $goreleaser_dir/$bin"
done

# --- Step 2: Update version in all package.json files ---
echo ""
echo "Setting version to ${VERSION}..."

# Update platform packages
for pkg in "${PLATFORM_MAP[@]}"; do
  pkg_json="$NPM/$pkg/package.json"
  sed -i "s/\"version\": \"[^\"]*\"/\"version\": \"${VERSION}\"/" "$pkg_json"
  echo "  OK   $pkg/package.json"
done

# Update main package (version + optionalDependencies versions)
main_json="$NPM/dokrypt/package.json"
sed -i "s/\"version\": \"[^\"]*\"/\"version\": \"${VERSION}\"/" "$main_json"
for pkg in "${PLATFORM_MAP[@]}"; do
  # Update the version of each optionalDependency
  pkg_name=$(echo "$pkg" | sed 's/\//\\\//g')
  sed -i "s/\"${pkg_name}\": \"[^\"]*\"/\"${pkg_name}\": \"${VERSION}\"/" "$main_json"
done
echo "  OK   dokrypt/package.json"

# --- Step 3: Publish platform packages first ---
echo ""
echo "Publishing platform packages..."
for pkg in "${PLATFORM_MAP[@]}"; do
  pkg_dir="$NPM/$pkg"
  bin="${BINARY_NAME[$pkg]}"

  # Skip if binary wasn't copied
  if [[ ! -f "$pkg_dir/bin/$bin" ]]; then
    echo "  SKIP $pkg (no binary)"
    continue
  fi

  echo "  Publishing $pkg..."
  (cd "$pkg_dir" && npm publish --access public $DRY_RUN)
done

# --- Step 4: Publish main package ---
echo ""
echo "Publishing dokrypt (main package)..."
(cd "$NPM/dokrypt" && npm publish --access public $DRY_RUN)

echo ""
echo "Done! Published dokrypt v${VERSION}"
echo ""
echo "Users can now install with:"
echo "  npm install -g dokrypt"
