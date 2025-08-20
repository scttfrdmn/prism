# Disabled GitHub Actions

These GitHub Actions workflows have been disabled to reduce email notifications for solo development.

## What was disabled:
- **CI/CD workflows**: build, test, and release automation
- **Package management**: Homebrew, Chocolatey, Conda auto-updates  
- **Security scanning**: dependency and AWS vulnerability scans
- **Documentation**: docs building and deployment
- **Integration testing**: AWS integration and GUI testing
- **Dependabot**: automated dependency updates

## Pre-commit hooks handle:
- Code formatting and linting
- Basic tests
- Security checks
- Documentation validation

## To re-enable workflows:
```bash
mv .github/workflows-disabled/* .github/workflows/
mv .github/dependabot.yml.disabled .github/dependabot.yml
```

## To re-enable specific workflows only:
```bash
# Example: Re-enable just release workflow
mv .github/workflows-disabled/release.yml .github/workflows/
```

**Note**: This is a reasonable approach for solo development where pre-commit hooks provide adequate quality control without the notification overhead.