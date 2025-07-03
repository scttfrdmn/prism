# CloudWorkstation Project Development Rules

## Core Development Philosophy

### 🚫 **RULE #1: NO WORKAROUNDS, NO SHORTCUTS, NO CHEATING**

**This is the foundational rule of CloudWorkstation development.**

When encountering problems, issues, or obstacles during development:

#### ❌ **FORBIDDEN APPROACHES:**
- **No Workarounds**: Don't circumvent problems with temporary fixes
- **No Fallbacks**: Don't implement "Plan B" solutions when the correct approach fails
- **No Cheating**: Don't skip steps, avoid proper implementation, or use shortcuts
- **No "Good Enough"**: Don't accept suboptimal solutions to move forward
- **No Silent Failures**: Don't ignore errors or warnings
- **No Technical Debt**: Don't defer proper solutions for "later"

#### ✅ **REQUIRED APPROACHES:**
- **Fix the Root Cause**: Identify and resolve the actual problem
- **Work Through Issues**: Persist until the correct solution is implemented
- **Understand the Problem**: Don't proceed without understanding why something failed
- **Implement Properly**: Use the right tools, patterns, and approaches
- **Test Thoroughly**: Ensure fixes actually work and don't introduce regressions
- **Document Solutions**: Record how problems were solved for future reference

### 📋 **Examples of Rule Application**

#### Docker Compose Version Warning
**❌ Wrong**: Ignore the warning message about obsolete `version` attribute
**✅ Right**: Remove the obsolete `version: '3.8'` line from docker-compose.test.yml

#### Test Coverage Below Target
**❌ Wrong**: Lower the coverage requirement from 85% to accommodate current state
**✅ Right**: Write comprehensive tests to achieve the 85% coverage target

#### AWS API Integration Failing
**❌ Wrong**: Mock all AWS calls and skip integration testing
**✅ Right**: Debug LocalStack setup, fix configuration, ensure real integration tests work

#### Build Errors
**❌ Wrong**: Comment out failing code or disable failing tests
**✅ Right**: Fix the underlying compilation or logic errors

#### Dependency Issues
**❌ Wrong**: Use older versions to avoid compatibility problems
**✅ Right**: Update code to work with current dependencies or fix compatibility issues

#### Linting Errors
**❌ Wrong**: Disable linting rules for problematic code
**✅ Right**: Refactor code to comply with linting standards

### 🎯 **Benefits of This Approach**

1. **Long-term Quality**: Solutions that last and don't create future problems
2. **Deep Understanding**: Developers understand the system thoroughly
3. **Reliable Foundation**: Each component can be trusted to work correctly
4. **Maintainability**: Code remains clean and understandable
5. **Professional Standards**: Meets enterprise-grade development practices

### 🚨 **When This Rule Is Most Important**

- **Under Time Pressure**: When deadlines loom, the temptation to shortcut increases
- **Complex Problems**: When the solution isn't immediately obvious
- **External Dependencies**: When third-party tools or services cause issues
- **Testing Failures**: When tests are difficult to make pass
- **Performance Issues**: When optimization requires significant effort
- **Integration Challenges**: When systems don't work together easily
- **Code Quality Gates**: When linting or quality checks raise issues

### 💡 **Implementation Guidelines**

#### Problem Solving Process
1. **Identify**: Clearly define what is actually broken or failing
2. **Research**: Understand the expected behavior and why it's not working
3. **Plan**: Design the correct solution approach
4. **Implement**: Execute the proper fix without shortcuts
5. **Verify**: Test that the solution actually works
6. **Document**: Record the problem and solution for future reference

#### Code Review Standards
- All code must solve problems correctly, not work around them
- Temporary fixes must include TODO comments with concrete resolution plans
- Comments explaining "why we had to do it this way" are red flags
- Any compromise must be explicitly documented and justified
- All code must pass linting before submission
- Code style must follow project conventions consistently

#### Testing Requirements
- Tests must verify actual functionality, not mock everything away
- Integration tests must use real external dependencies when possible
- Coverage targets are requirements, not suggestions:
  - Minimum 85% test coverage for the overall project
  - Minimum 80% test coverage for each individual file
- Flaky tests must be fixed, not ignored or disabled

### 📚 **Documentation Requirements**

All development decisions should be documented with:
- **Problem Statement**: What was broken or needed
- **Solution Approach**: Why this approach was chosen
- **Implementation Details**: How it was actually solved
- **Verification**: How we know it works correctly
- **Future Considerations**: Any implications for future development

### ⚖️ **Enforcement**

This rule applies to:
- **All Code**: Production, test, configuration, documentation
- **All Developers**: Solo development, team contributions, external PRs
- **All Phases**: Initial development, bug fixes, refactoring, maintenance
- **All Components**: Backend, frontend, infrastructure, tooling

### 🎖️ **Success Metrics**

A successful implementation following this rule will demonstrate:
- Zero known workarounds in the codebase
- All tests pass reliably
- All coverage targets met (85%+ overall, 80%+ per file)
- All files pass linting without errors or exemptions
- All warnings and errors addressed
- All external dependencies work correctly
- All documentation is accurate and complete

---

**Remember**: Short-term pain of doing things right is infinitely better than long-term pain of maintaining shortcuts and workarounds.

**CloudWorkstation aims to be a professional-grade tool that researchers can depend on. This level of reliability requires uncompromising development standards.**