# Web Service Tunneling Testing Guide

## Overview

This document provides testing procedures for the automatic web service tunneling feature implemented in CloudWorkstation.

## Features to Test

### 1. Automatic Tunnel Creation on Connect

**Test**: `cws connect` should automatically create tunnels for all web services

```bash
# Launch instance with web services
cws launch python-ml test-jupyter --size S

# Connect - should show tunnel creation
cws connect test-jupyter

# Expected output:
# 🌐 Setting up tunnels for web services...
# ✅ Tunnels created:
#    • Jupyter Lab: http://localhost:8888?token=abc123
#    • (other services...)
# 🔗 Connecting to test-jupyter...
```

**Validation**:
- Tunnels created before SSH connection
- Auth tokens displayed for Jupyter
- URLs are clickable/bookmarkable
- SSH connection still works if tunnel fails

### 2. Web Service List Command

**Test**: `cws web list` shows all available services with tunnel status

```bash
cws web list test-jupyter

# Expected output:
# Web services for test-jupyter:
#
# ✅ Jupyter Lab (port 8888)
#    URL: http://localhost:8888?token=abc123
#
# ❌ RStudio Server (port 8787)
#    Not tunneled - use 'cws web open test-jupyter rstudio-server' to access
```

**Validation**:
- Shows all services configured in template
- Indicates tunnel status (✅ tunneled / ❌ not tunneled)
- Displays URLs for tunneled services
- Shows auth tokens when available

### 3. Web Service Open Command

**Test**: `cws web open` creates tunnel and opens browser

```bash
cws web open test-jupyter jupyter

# Expected output:
# 🌐 Creating tunnel for jupyter...
# ✅ Tunnel created: http://localhost:8888?token=abc123
# 🌐 Opening in browser...
# ✅ Browser opened
```

**Validation**:
- Tunnel created if not exists
- Browser opens automatically
- URL includes auth token
- Works across platforms (macOS, Linux, Windows)

### 4. Web Service Close Command

**Test**: `cws web close` closes tunnels

```bash
# Close specific service
cws web close test-jupyter jupyter

# Close all services
cws web close test-jupyter

# Expected output:
# 🔒 Closing tunnel for test-jupyter/jupyter...
# ✅ Tunnel closed
```

**Validation**:
- Tunnels actually close (ports released)
- Can close individual or all tunnels
- Graceful error if tunnel doesn't exist

### 5. Jupyter Token Extraction

**Test**: Token automatically extracted from Jupyter

```bash
# Launch Jupyter instance
cws launch python-ml test-jupyter --size S

# Connect or open web service
cws connect test-jupyter
# or
cws web open test-jupyter jupyter
```

**Validation**:
- Token appears in URL
- Token is valid (can access Jupyter)
- Works with Jupyter Lab and Jupyter Notebook
- Graceful degradation if token extraction fails

### 6. Multiple Services

**Test**: Multiple services can have tunnels simultaneously

```bash
# Launch R instance (has RStudio + Shiny)
cws launch r-research test-r --size M

# Create tunnels for all services
cws connect test-r

# List all tunnels
cws web list test-r
```

**Validation**:
- Multiple tunnels coexist
- Each service on correct port
- No port conflicts
- All services accessible

### 7. Service Metadata on Launch

**Test**: New instances have service metadata

```bash
# Launch instance
cws launch python-ml test-services --size S

# Check instance has services
cws show test-services | grep -i service
```

**Validation**:
- Services array populated
- Port numbers correct
- Service names match template
- Descriptions included

### 8. GUI Integration

**Test**: GUI can open web services

```bash
# Start GUI
cws-gui

# In GUI:
# 1. Select instance with web services
# 2. Click "Open Web Service" or similar
# 3. Select service (Jupyter, RStudio, etc.)
```

**Validation**:
- Service list displayed
- Tunnel created on selection
- Web content displayed in GUI (if iframe implemented)
- Handles auth tokens correctly

## Test Instances

### Minimal Test Instance
```bash
cws launch python-ml test-web-minimal --size S --spot
# Fast launch, low cost
# Services: Jupyter Lab (port 8888)
```

### Full-Featured Test Instance
```bash
cws launch r-research test-web-full --size M
# Complete testing
# Services: RStudio Server (8787), Shiny Server (3838)
```

## Known Limitations

1. **Token Extraction**: Only works for Jupyter currently
   - RStudio uses authentication but no token extraction yet
   - Shiny Server typically has no authentication

2. **SSH Key Resolution**: Uses hardcoded profile path
   - TODO: Get actual profile name from instance metadata

3. **Port Allocation**: Uses same port numbers locally
   - Works well for single instance
   - May conflict if multiple instances have same service

4. **Browser Opening**: Platform-specific
   - macOS: Uses `open`
   - Linux: Uses `xdg-open`
   - Windows: Uses `cmd /c start`

## Success Criteria

- ✅ Tunnels created automatically on connect
- ✅ Web service commands work (list, open, close)
- ✅ Jupyter tokens extracted and included in URLs
- ✅ Browser opens automatically
- ✅ Multiple services can coexist
- ✅ GUI can list and open services
- ✅ Zero manual SSH commands needed
- ✅ Graceful error handling (warnings, not failures)

## Cleanup

After testing, remove test instances:

```bash
cws delete test-web-services --yes
cws delete test-jupyter --yes
cws delete test-r --yes
# etc.
```
