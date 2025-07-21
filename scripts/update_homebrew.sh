#!/bin/bash

# Script to update the Homebrew formula with correct SHA256 checksums for releases
# Usage: ./scripts/update_homebrew.sh <version> <path_to_release_dir>

set -e

# Validate arguments
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <version> <path_to_release_dir>"
    echo "Example: $0 0.4.2 ./dist/v0.4.2"
    exit 1
fi

VERSION=$1
RELEASE_DIR=$2

# Ensure version starts with 'v' if not already
VERSION_NUM=${VERSION#v}
VERSION_TAG="v$VERSION_NUM"

# Check if release directory exists
if [ ! -d "$RELEASE_DIR" ]; then
    echo "Error: Release directory $RELEASE_DIR does not exist"
    exit 1
fi

# Define archive files
DARWIN_AMD64="cloudworkstation-darwin-amd64.tar.gz"
DARWIN_ARM64="cloudworkstation-darwin-arm64.tar.gz"
LINUX_AMD64="cloudworkstation-linux-amd64.tar.gz"
LINUX_ARM64="cloudworkstation-linux-arm64.tar.gz"

# Check if archive files exist in release directory
for archive in "$DARWIN_AMD64" "$DARWIN_ARM64" "$LINUX_AMD64" "$LINUX_ARM64"; do
    if [ ! -f "$RELEASE_DIR/$archive" ]; then
        echo "Error: Archive file $archive not found in $RELEASE_DIR"
        exit 1
    fi
done

# Calculate SHA256 checksums
SHA_DARWIN_AMD64=$(shasum -a 256 "$RELEASE_DIR/$DARWIN_AMD64" | awk '{print $1}')
SHA_DARWIN_ARM64=$(shasum -a 256 "$RELEASE_DIR/$DARWIN_ARM64" | awk '{print $1}')
SHA_LINUX_AMD64=$(shasum -a 256 "$RELEASE_DIR/$LINUX_AMD64" | awk '{print $1}')
SHA_LINUX_ARM64=$(shasum -a 256 "$RELEASE_DIR/$LINUX_ARM64" | awk '{print $1}')

# Update the formula
FORMULA_PATH="scripts/homebrew/cloudworkstation.rb"

# Backup original formula
cp "$FORMULA_PATH" "$FORMULA_PATH.bak"

# Update version and SHA256 checksums
sed -i.tmp "s|version \".*\"|version \"$VERSION_NUM\"|g" "$FORMULA_PATH"
sed -i.tmp "s|/v[0-9.]*\(/cloudworkstation-darwin-arm64.tar.gz\"|/$VERSION_TAG\1\"|g" "$FORMULA_PATH"
sed -i.tmp "s|/v[0-9.]*\(/cloudworkstation-darwin-amd64.tar.gz\"|/$VERSION_TAG\1\"|g" "$FORMULA_PATH"
sed -i.tmp "s|/v[0-9.]*\(/cloudworkstation-linux-arm64.tar.gz\"|/$VERSION_TAG\1\"|g" "$FORMULA_PATH"
sed -i.tmp "s|/v[0-9.]*\(/cloudworkstation-linux-amd64.tar.gz\"|/$VERSION_TAG\1\"|g" "$FORMULA_PATH"

sed -i.tmp "s|sha256 \".*\" # Will be updated during release process.*darwin-arm64|sha256 \"$SHA_DARWIN_ARM64\" # Updated for $VERSION_TAG darwin-arm64|g" "$FORMULA_PATH"
sed -i.tmp "s|sha256 \".*\" # Will be updated during release process.*darwin-amd64|sha256 \"$SHA_DARWIN_AMD64\" # Updated for $VERSION_TAG darwin-amd64|g" "$FORMULA_PATH"
sed -i.tmp "s|sha256 \".*\" # Will be updated during release process.*linux-arm64|sha256 \"$SHA_LINUX_ARM64\" # Updated for $VERSION_TAG linux-arm64|g" "$FORMULA_PATH"
sed -i.tmp "s|sha256 \".*\" # Will be updated during release process.*linux-amd64|sha256 \"$SHA_LINUX_AMD64\" # Updated for $VERSION_TAG linux-amd64|g" "$FORMULA_PATH"

# Clean up temporary files
rm -f "$FORMULA_PATH.tmp"

echo "Updated $FORMULA_PATH with checksums for $VERSION_TAG"
echo ""
echo "Darwin ARM64: $SHA_DARWIN_ARM64"
echo "Darwin AMD64: $SHA_DARWIN_AMD64"
echo "Linux ARM64:  $SHA_LINUX_ARM64"
echo "Linux AMD64:  $SHA_LINUX_AMD64"
echo ""
echo "To validate the formula, run: brew install --build-from-source $FORMULA_PATH"
echo "To submit the formula to the tap, copy it to your homebrew-cloudworkstation repository"