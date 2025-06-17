# Cloud Workstation Platform

A command-line tool for launching pre-configured research environments in the cloud in seconds, not hours.

## Quick Start

```bash
# Launch an R research environment
./cws launch r-research my-instance

# List running instances  
./cws list

# Connect to your instance
./cws connect my-instance

# Stop instance to save costs
./cws stop my-instance

# Clean up when done
./cws delete my-instance
```

## Prerequisites

- Go 1.19+
- AWS CLI configured with credentials
- AWS account with EC2 permissions

## Installation

```bash
# Clone and build
git clone <your-repo-url>
cd cloudworkstation
go build -o cws main.go

# Or install directly
go install github.com/yourusername/cloudworkstation@latest
```

## Available Templates

- **r-research**: R + RStudio Server + tidyverse packages
- **python-research**: Python + Jupyter + data science stack  
- **basic-ubuntu**: Plain Ubuntu 22.04 for general use

## Configuration

The tool stores state in `~/.cloudworkstation/state.json` and uses your default AWS profile.

To use a different AWS profile:
```bash
export AWS_PROFILE=research
./cws launch r-research my-instance
```

## Cost Management

All instances are launched with detailed cost tracking. Use `./cws list` to see estimated daily costs.

Remember to stop instances when not in use to minimize costs!

## Development

```bash
# Run tests
go test ./...

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o cws-linux main.go
GOOS=windows GOARCH=amd64 go build -o cws-windows.exe main.go
GOOS=darwin GOARCH=amd64 go build -o cws-macos main.go
```

## License

MIT License - see LICENSE file for details.
