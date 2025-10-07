# CloudWorkstation Marketplace Commands Test Guide

## Overview

The CloudWorkstation marketplace system has been fully implemented with complete CLI integration and daemon backend support. All marketplace commands now use real daemon API calls instead of mock responses.

## Available Marketplace Commands

### 1. List Templates
```bash
# List all marketplace templates
./bin/cws marketplace list

# List with filters
./bin/cws marketplace list --category machine-learning --min-rating 4.0 --verified true
./bin/cws marketplace list --architecture arm64 --tags "pytorch,gpu" --limit 10
```

### 2. Search Templates
```bash
# Search by query
./bin/cws marketplace search "genomics"

# Search with filters
./bin/cws marketplace search "machine learning" --category ai --min-rating 4.5
```

### 3. Get Template Info
```bash
# Get detailed information about a specific template
./bin/cws marketplace info genomics-pipeline-v3
./bin/cws marketplace info machine-learning-gpu
```

### 4. Install Template (NEW)
```bash
# Install a marketplace template locally
./bin/cws marketplace install genomics-pipeline-v3

# Install with custom local name
./bin/cws marketplace install machine-learning-gpu --as my-ml-env

# Install and download AMI for faster launches
./bin/cws marketplace install genomics-pipeline-v3 --as genomics-env --download-ami
```

### 5. Browse Featured Templates
```bash
# Show featured templates
./bin/cws marketplace featured
```

### 6. Browse Trending Templates
```bash
# Show trending templates (default: week)
./bin/cws marketplace trending

# Show trending for specific timeframe
./bin/cws marketplace trending --timeframe month
```

### 7. Browse Categories
```bash
# List available categories
./bin/cws marketplace categories
```

### 8. Publish Template
```bash
# Publish from running instance
./bin/cws marketplace publish my-instance --name "My Custom Template" --category machine-learning --description "Optimized ML environment"

# Publish with additional metadata
./bin/cws marketplace publish my-instance \
  --name "Advanced Genomics Pipeline" \
  --category bioinformatics \
  --description "Complete genomics workflow" \
  --tags "genomics,gatk,bioconductor" \
  --license MIT \
  --generate-ami \
  --regions us-east-1,us-west-2
```

### 9. Review Template
```bash
# Add a review
./bin/cws marketplace review genomics-pipeline-v3 \
  --rating 5 \
  --title "Excellent for research" \
  --comment "Saved weeks of setup time" \
  --use-case "genomics-analysis"
```

### 10. Fork Template
```bash
# Fork a template for customization
./bin/cws marketplace fork machine-learning-gpu \
  --name "My Custom ML Environment" \
  --description "Customized for our research needs"
```

### 11. My Publications
```bash
# Show templates you've published
./bin/cws marketplace my-publications
```

## Testing the Implementation

### 1. Start the Daemon
```bash
# Start daemon with marketplace support
./bin/cwsd
```

### 2. Test Basic Commands
```bash
# Test listing (should show sample data)
./bin/cws marketplace list

# Test categories
./bin/cws marketplace categories

# Test featured templates
./bin/cws marketplace featured
```

### 3. Test Template Installation
```bash
# Install a sample template
./bin/cws marketplace install genomics-pipeline-v3 --as test-genomics

# Check if it was tracked properly
./bin/cws marketplace info genomics-pipeline-v3
```

## Key Features Implemented

### ✅ Complete CLI Integration
- All commands use real daemon API calls (no more mock responses)
- Proper argument parsing with `--flag value` syntax
- Comprehensive error handling and user-friendly messages
- Rich output with emojis and formatting

### ✅ Full Daemon Backend
- RESTful API endpoints for all marketplace operations
- Marketplace registry with sample data
- Analytics and tracking support
- Template installation endpoint
- Review and rating system

### ✅ Template Installation System
- Install marketplace templates locally
- Optional custom naming
- AMI download integration
- Usage tracking for analytics
- Clear usage instructions after installation

### ✅ Advanced Features
- Search with multiple filters (category, rating, architecture, etc.)
- Template publishing from running instances
- Review and rating system
- Template forking for customization
- Analytics and usage tracking
- Featured and trending template discovery

## Architecture Overview

```
CLI Commands (internal/cli/marketplace.go)
    ↓
API Client (pkg/api/client/http_client.go)
    ↓
Daemon API (pkg/daemon/marketplace_handlers.go)
    ↓
Marketplace Registry (pkg/marketplace/registry.go)
    ↓
Sample Data & Analytics
```

## Next Steps

The marketplace system is now fully functional with:
- Complete CLI command suite
- Full daemon backend integration
- Sample data for testing
- Analytics and tracking support

Ready for production deployment and real marketplace data integration!