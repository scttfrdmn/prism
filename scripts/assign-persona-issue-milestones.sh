#!/bin/bash
# Assign milestones to persona gap issues

set -e

echo "Assigning v0.5.8 milestone to all 56 persona issues..."
echo ""

# Process all issues #138-193
success_count=0
fail_count=0

for issue in {138..193}; do
  if gh issue edit $issue --milestone "v0.5.8" 2>/dev/null; then
    ((success_count++))
    echo "  ✓ #$issue"
  else
    ((fail_count++))
    echo "  ✗ #$issue"
  fi
done

echo ""
echo "✅ Milestone assignment complete"
echo ""
echo "Results:"
echo "  Success: $success_count issues"
echo "  Failed: $fail_count issues"
echo ""
echo "Summary by milestone:"
echo "  v0.5.8: All 56 issues (#138-193)"
echo ""
echo "Summary by persona:"
echo "  Solo Researcher: 10 issues (#138-147)"
echo "  Lab Environment: 12 issues (#148-159)"
echo "  University Class: 16 issues (#160-175)"
echo "  Conference Workshop: 9 issues (#176-184)"
echo "  Cross-Institutional: 9 issues (#185-193)"
echo ""
echo "View milestone: https://github.com/scttfrdmn/prism/milestone/23"
