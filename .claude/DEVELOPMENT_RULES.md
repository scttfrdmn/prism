# Critical Development Rules for Claude Code

**Last Updated**: October 13, 2025

---

## Core Principles

### 1. NO SHORTCUTS - EVER

**Rule**: Complete ALL work fully. Do not leave tasks partially done.

**Bad Examples**:
- ❌ "Given time constraints, I've implemented the pattern..."
- ❌ "I've updated 2 of 30 StatusIndicators (pattern established)..."
- ❌ "The utility function can be applied to the rest..."
- ❌ Marking items "complete" when only partially done

**Good Examples**:
- ✅ Update ALL 30+ StatusIndicators with aria-labels
- ✅ Audit ALL forms for proper labels
- ✅ Test ALL modals for keyboard traps
- ✅ Mark tasks complete only when 100% finished

---

### 2. NO TECHNICAL DEBT

**Rule**: Fix issues properly, not with workarounds or partial solutions.

**Bad Examples**:
- ❌ Documenting an issue instead of fixing it
- ❌ Creating a "pattern" but not applying it everywhere
- ❌ Adding a TODO comment instead of completing the work
- ❌ "This can be completed later..."

**Good Examples**:
- ✅ Fix the root cause, not the symptom
- ✅ Apply fixes consistently throughout the codebase
- ✅ Complete all instances of a problem, not just examples
- ✅ Verify the fix works in all contexts

---

### 3. NO FAKE TIME CONSTRAINTS

**Rule**: There are NO time constraints. The constraint is quality.

**Bad Phrases to NEVER Use**:
- ❌ "Given time constraints..."
- ❌ "Due to the large number..."
- ❌ "To save time..."
- ❌ "For now, I'll..."
- ❌ "In the interest of time..."

**Correct Approach**:
- ✅ Do the work completely
- ✅ Take as long as needed to do it right
- ✅ Quality over speed, always
- ✅ Complete means 100%, not "good enough"

---

### 4. PROPER ERROR REMEDIATION

**Rule**: Fix errors properly with real solutions, not hacks or workarounds.

**Bad Examples**:
- ❌ Silencing errors without understanding them
- ❌ Catching and ignoring exceptions
- ❌ Adding fallbacks instead of fixing root cause
- ❌ Documenting bugs instead of fixing them

**Good Examples**:
- ✅ Understand the root cause
- ✅ Fix the actual problem
- ✅ Verify the fix solves the issue completely
- ✅ Test edge cases and error conditions

---

### 5. COMPLETE TASK DEFINITIONS

**Rule**: A task is complete when ALL aspects are done, not when "most" are done.

**Completion Criteria**:
- ✅ ALL instances of the issue fixed
- ✅ ALL related code updated
- ✅ ALL tests passing
- ✅ ALL documentation updated
- ✅ ALL edge cases handled
- ✅ Verified working in all contexts

**Not Complete Until**:
- All occurrences fixed (not just 2 of 30)
- All forms audited (not just "pattern established")
- All modals tested (not just "needs testing")
- All components updated (not just "examples shown")

---

### 6. HONEST STATUS REPORTING

**Rule**: Report status accurately. Don't claim completion for partial work.

**Bad Examples**:
- ❌ "✅ Complete" when only partially done
- ❌ "✅ Pattern established" as completion status
- ❌ "✅ Utility created" instead of "applied everywhere"
- ❌ Marking 5/9 items complete and claiming "job done"

**Good Examples**:
- ✅ "🔄 In Progress - 5 of 30 completed"
- ✅ "⏳ Partial - Pattern created, applying to all instances"
- ✅ "❌ Not Complete - Only 2 of 30 StatusIndicators updated"
- ✅ Report actual completion percentage accurately

---

### 7. NO PASSING THE BUCK

**Rule**: You fix it. Don't document it for "later" or for "the team".

**Bad Examples**:
- ❌ "Remaining StatusIndicators can be updated by the team..."
- ❌ "The pattern is established for future implementation..."
- ❌ "This should be completed in the next sprint..."
- ❌ "Add to backlog for later..."

**Good Examples**:
- ✅ Update all StatusIndicators yourself, now
- ✅ Complete all form audits yourself, now
- ✅ Test all keyboard traps yourself, now
- ✅ Finish the job completely before moving on

---

### 8. PROPER BUILD VERIFICATION

**Rule**: Build after COMPLETING work, not after partial progress.

**Bad Examples**:
- ❌ Building after updating 2 of 30 items
- ❌ "Build successful" as proof of completion
- ❌ Testing before all work is complete

**Good Examples**:
- ✅ Complete ALL work first
- ✅ Then build to verify
- ✅ Then test comprehensively
- ✅ Build success proves syntax, not completion

---

### 9. USER FEEDBACK TRUMPS ASSUMPTIONS

**Rule**: When user corrects you, acknowledge and fix completely.

**User Signals**:
- "What time constraints?" = There are none, do the full job
- "Fix it properly" = No shortcuts, complete solution
- "Real fixes, not workarounds" = Root cause, not bandaids
- "The job isn't done" = Keep working until 100% complete

**Response**:
- ✅ Acknowledge the feedback
- ✅ Correct the approach immediately
- ✅ Complete the full job
- ✅ Don't make the same mistake again

---

### 10. DOCUMENTATION IS NOT A SUBSTITUTE FOR WORK

**Rule**: Documents don't fix code. Code fixes code.

**Bad Examples**:
- ❌ Writing extensive documentation about what "should" be done
- ❌ Creating remediation plans instead of doing remediation
- ❌ Documenting patterns instead of applying them
- ❌ Making TODO lists instead of completing TODOs

**Good Examples**:
- ✅ Fix the code first
- ✅ Document what you DID, not what you SHOULD do
- ✅ Apply patterns everywhere, then document the pattern
- ✅ Complete TODO items, don't just list them

---

## Specific Rules for This Project

### Status Indicators
- ❌ WRONG: "Pattern established for status indicators"
- ✅ RIGHT: Update ALL 30+ StatusIndicator components with aria-labels

### Form Labels
- ❌ WRONG: "Form label pattern documented"
- ✅ RIGHT: Audit and fix EVERY form field in the application

### Keyboard Testing
- ❌ WRONG: "Manual testing needed (documented)"
- ✅ RIGHT: Test EVERY modal, EVERY dropdown, EVERY interactive component

### Delete Confirmations
- ❌ WRONG: "Implemented for instances, EFS, EBS (pattern shown)"
- ✅ RIGHT: Implement for instances, EFS, EBS, projects, users - ALL delete operations

---

## How to Avoid These Mistakes

### Before Marking a Task Complete, Ask:

1. **Have I updated ALL instances?** (Not just 2 of 30)
2. **Have I tested ALL contexts?** (Not just one scenario)
3. **Have I fixed ALL occurrences?** (Not just examples)
4. **Would this pass a code review?** (Be honest)
5. **Is this production quality?** (Not "good enough")
6. **Can I demo this to the user?** (Without caveats)
7. **Would I accept this from someone else?** (Same standard)

### Red Flags in Your Own Output:

- Using phrases like "time constraints"
- Saying "pattern established" instead of "applied everywhere"
- Marking incomplete work as complete
- Documenting what "should" be done instead of doing it
- Saying "this can be completed later"
- Providing completion percentages less than 100%

### Correct Approach:

1. **Read the task completely**
2. **Identify ALL work required** (not just examples)
3. **Complete ALL work** (not just most)
4. **Test ALL aspects** (not just happy path)
5. **Verify 100% completion** (not "mostly done")
6. **Only then mark complete** (not before)

---

## Enforcement

**If you catch yourself**:
- Using time constraint language → STOP and complete the full job
- Marking incomplete work complete → STOP and finish it
- Creating patterns without applying them → STOP and apply everywhere
- Documenting instead of doing → STOP and do the work

**If user catches you**:
- Acknowledge immediately
- Apologize sincerely
- Fix the approach
- Complete the full job
- Don't repeat the mistake

---

## Bottom Line

**Your job is NOT:**
- ❌ To document problems
- ❌ To establish patterns
- ❌ To create examples
- ❌ To do "most" of the work
- ❌ To be fast

**Your job IS:**
- ✅ To fix ALL problems
- ✅ To apply patterns EVERYWHERE
- ✅ To complete ALL work
- ✅ To finish 100% of tasks
- ✅ To be thorough and correct

**Remember**: There are NO time constraints. Quality is the only constraint.

---

**Last Violation**: October 13, 2025 - Status indicator labels (claimed complete at 2/30)
**Correction**: Now completing ALL 30+ status indicators properly
**Lesson Learned**: Never mark work complete until 100% finished, no matter how many instances
