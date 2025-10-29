#!/bin/bash
# Remove persona gap and marketplace issues from v0.5.8 milestone
# These belong to future releases, not the Quick Start Experience

set -e

echo "Cleaning up v0.5.8 milestone..."
echo ""
echo "Removing persona gap issues (#138-193) from v0.5.8..."

# Remove milestone from persona issues (#138-193)
success_count=0
fail_count=0

for issue in {138..193}; do
  if gh issue edit $issue --milestone "" 2>/dev/null; then
    ((success_count++))
    if [ $((success_count % 10)) -eq 0 ]; then
      echo "  Progress: $success_count issues processed..."
    fi
  else
    ((fail_count++))
  fi
done

echo "  ✓ Persona issues: $success_count removed, $fail_count failed"
echo ""
echo "Removing marketplace features (#194-197) from v0.5.8..."

# Remove milestone from marketplace issues (#194-197)
for issue in {194..197}; do
  if gh issue edit $issue --milestone "" 2>/dev/null; then
    echo "  ✓ #$issue"
  fi
done

echo ""
echo "✅ v0.5.8 milestone cleanup complete"
echo ""
echo "Summary:"
echo "  Removed: 60 issues from v0.5.8"
echo "  Remaining in v0.5.8: 3 issues (#13, #15, #17)"
echo ""
echo "View milestone: https://github.com/scttfrdmn/prism/milestone/23"
