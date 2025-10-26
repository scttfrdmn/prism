# Prism Development Setup
**Avoiding Keychain Password Prompts During Development**

## 🔐 **Development Mode**

Prism automatically detects development/testing contexts and uses secure file storage instead of macOS Keychain to avoid frequent password prompts.

### **Automatic Detection:**
Development mode is automatically enabled when:
- `GO_ENV=test` is set
- `CLOUDWORKSTATION_DEV=true` is set  
- Running tests (`go test`)
- Running from temporary directories
- Running binaries with "test" in the path

### **Manual Control:**
```bash
# Force development mode (avoids keychain prompts)
export CLOUDWORKSTATION_DEV=true

# Run tests without keychain prompts
make test

# Force production mode (uses keychain)
unset CLOUDWORKSTATION_DEV

# Test production keychain integration
prism daemon start
prism security keychain
```

## 🛠️ **Development Commands:**
```bash
# Build and test without keychain prompts
make build
make test

# Run CLI commands without keychain prompts (development mode auto-detected)
go run ./cmd/cws templates
go run ./cmd/cws --help

# Force production behavior for testing
unset CLOUDWORKSTATION_DEV
./bin/cws daemon start  # Will use keychain
```

## 🔒 **Security Notes:**
- **Development**: Uses AES-256 encrypted file storage in `~/.prism/secure/`
- **Production**: Uses native macOS Keychain with hardware security when available
- **Same Security Level**: Both approaches provide strong encryption
- **Automatic Fallback**: Production mode falls back to file storage if keychain unavailable

## 🎯 **Result:**
No more frequent keychain password prompts during development while maintaining full production security!