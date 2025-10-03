#!/bin/bash

echo "CloudWorkstation Coverage Analysis"
echo "=================================="
echo ""

for pkg in $(find . -name "*.go" -not -path "./vendor/*" -not -path "./node_modules/*" -not -name "*test*" | sed 's|/[^/]*\.go$||' | sort -u); do
  source_files=$(find "$pkg" -maxdepth 1 -name "*.go" -not -name "*test*" 2>/dev/null | wc -l)
  test_files=$(find "$pkg" -maxdepth 1 -name "*test.go" 2>/dev/null | wc -l)

  if [ "$test_files" -eq 0 ]; then
    status="❌ NO TESTS"
  elif [ "$test_files" -lt 2 ]; then
    status="⚠️  LIMITED"
  else
    status="✅ COVERED"
  fi

  printf "%-35s Source: %-3d Test: %-3d %s\n" "$pkg" "$source_files" "$test_files" "$status"
done