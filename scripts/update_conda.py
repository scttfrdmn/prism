#!/usr/bin/env python3
"""
Script to update the Conda package meta.yaml with correct SHA256 checksums for releases.
Usage: python scripts/update_conda.py <version> <path_to_release_dir>
"""

import argparse
import hashlib
import os
import re
import sys

def calculate_sha256(file_path):
    """Calculate SHA256 checksum for a file."""
    sha256_hash = hashlib.sha256()
    with open(file_path, "rb") as f:
        for byte_block in iter(lambda: f.read(4096), b""):
            sha256_hash.update(byte_block)
    return sha256_hash.hexdigest()

def update_meta_yaml(version, release_dir):
    """Update meta.yaml with new version and SHA256 checksums."""
    # Ensure version format is consistent
    if version.startswith('v'):
        version_num = version[1:]
    else:
        version_num = version
        version = f"v{version}"

    # Define archive files and their placeholders
    files = {
        "cloudworkstation-linux-amd64.tar.gz": "REPLACE_WITH_ACTUAL_CHECKSUM_LINUX_AMD64",
        "cloudworkstation-linux-arm64.tar.gz": "REPLACE_WITH_ACTUAL_CHECKSUM_LINUX_ARM64",
        "cloudworkstation-darwin-amd64.tar.gz": "REPLACE_WITH_ACTUAL_CHECKSUM_DARWIN_AMD64",
        "cloudworkstation-darwin-arm64.tar.gz": "REPLACE_WITH_ACTUAL_CHECKSUM_DARWIN_ARM64",
        "cloudworkstation-windows-amd64.zip": "REPLACE_WITH_ACTUAL_CHECKSUM_WINDOWS"
    }

    # Calculate checksums for all files
    checksums = {}
    for file_name, placeholder in files.items():
        file_path = os.path.join(release_dir, file_name)
        if os.path.exists(file_path):
            checksum = calculate_sha256(file_path)
            checksums[placeholder] = checksum
            print(f"Calculated checksum for {file_name}: {checksum}")
        else:
            print(f"Warning: {file_path} not found, skipping checksum calculation")

    # Update meta.yaml
    meta_yaml_path = "scripts/conda/meta.yaml"
    with open(meta_yaml_path, "r") as file:
        content = file.read()

    # Update version
    content = re.sub(r'{% set version = "[^"]+" %}', f'{{% set version = "{version_num}" %}}', content)

    # Update checksums
    for placeholder, checksum in checksums.items():
        content = content.replace(placeholder, checksum)

    # Write updated content back to file
    with open(meta_yaml_path, "w") as file:
        file.write(content)

    print(f"\nUpdated {meta_yaml_path} with version {version_num} and new checksums.")

def main():
    parser = argparse.ArgumentParser(description="Update Conda package meta.yaml with checksums")
    parser.add_argument("version", help="Version number (with or without 'v' prefix)")
    parser.add_argument("release_dir", help="Path to release directory containing archives")
    
    args = parser.parse_args()
    
    if not os.path.isdir(args.release_dir):
        print(f"Error: Release directory {args.release_dir} does not exist")
        return 1
    
    update_meta_yaml(args.version, args.release_dir)
    return 0

if __name__ == "__main__":
    sys.exit(main())