# Batch Device Management Guide

CloudWorkstation v0.4.3 introduces a comprehensive batch device management system for efficiently managing device registrations across multiple invitations. This guide explains how to use these features to maintain security and control access to your CloudWorkstation resources.

## Overview

The batch device management system provides administrators with tools to:

- Perform operations on multiple devices simultaneously
- Export device information across all invitations
- Validate device authenticity and authorization
- Revoke devices in bulk when needed
- Track device operations with detailed reporting

This system is particularly valuable for enterprise environments where managing many devices across multiple teams is required.

## Basic Concepts

### Device Binding

CloudWorkstation invitations can be configured with device binding, which restricts invitation usage to specific devices. When device binding is enabled:

- Each device that uses an invitation is registered with a unique device ID
- The invitation can only be used on registered devices
- Device limits can be set (e.g., limit to 1, 2, or more devices)
- Devices can be individually revoked without invalidating the invitation

### Registry System

The system maintains a centralized registry that tracks:
- Which devices are authorized to use which invitations
- When each device was registered
- Usage patterns and last seen times
- Device metadata for audit purposes

## Command Line Interface

### 1. Batch Device Operations

Perform operations on multiple devices using a CSV file:

```bash
cws profiles invitations devices batch-operation \
  --csv-file devices.csv \
  --operation revoke \
  --output-file results.csv
```

**Supported operations:**
- `revoke`: Revoke device access to the invitation
- `validate`: Verify device is authorized
- `info`: Gather information about the device

**CSV format:**
```
Device ID,Token,Name,Operation
d1234567890abcdef,inv-abcdefg,User Device,revoke
d2345678901bcdefg,inv-bcdefgh,Other Device,validate
```

**Options:**
- `--csv-file`: Path to CSV file with device operations (required)
- `--operation`: Override operation for all devices in CSV
- `--output-file`: Path to export results
- `--concurrency`: Number of concurrent operations (default: 5)
- `--has-header`: Whether CSV has header row (default: true)

### 2. Export Device Information

Export information about all registered devices:

```bash
cws profiles invitations devices export-info \
  --output-file device_inventory.csv
```

This command:
1. Retrieves all active invitations
2. For each invitation, fetches all registered devices
3. Compiles comprehensive device information
4. Exports to CSV format for analysis and record-keeping

**Options:**
- `--output-file`: Path for the exported CSV file
- `--concurrency`: Number of concurrent operations (default: 5)

### 3. Revoke All Devices

Revoke all devices across all invitations:

```bash
cws profiles invitations devices batch-revoke-all \
  --confirm \
  --output-file revocation_results.csv
```

This command:
1. Retrieves all active invitations
2. Finds all registered devices across all invitations
3. Revokes all devices in a single batch operation
4. Reports success/failure for each device

**Options:**
- `--confirm`: Required to confirm this powerful operation
- `--output-file`: Path to export results
- `--concurrency`: Number of concurrent operations (default: 5)

## CSV Format Reference

### Input CSV for Batch Operations

```csv
Device ID,Token,Name,Operation
d1234567890abcdef,inv-abcdefg,User Device,revoke
d2345678901bcdefg,inv-bcdefgh,Other Device,validate
d3456789012cdefgh,inv-cdefghi,Admin Device,info
```

**Required columns:**
- `Device ID`: The unique identifier of the device
- `Token`: The invitation token

**Optional columns:**
- `Name`: Descriptive name for reporting (often the invitation name)
- `Operation`: One of: `revoke`, `validate`, `info` (default: `revoke`)

### Output CSV Format

```csv
Device ID,Token,Invitation Name,Operation,Status,Registered At,Last Seen,Details,Error
d1234567890abcdef,inv-abcdefg,User Device,revoke,Success,2023-07-15T10:30:00Z,,device_type: mobile,
d2345678901bcdefg,inv-bcdefgh,Other Device,validate,Success,2023-07-16T14:22:15Z,2023-07-17T09:45:30Z,,
d3456789012cdefgh,inv-cdefghi,Admin Device,info,Failed,,,,Device not found
```

## Best Practices

### Security Management

1. **Regular Audits**: Use `export-info` to periodically audit all registered devices
2. **Shared Device Policies**: Use device binding with appropriate device limits:
   - Personal use: Limit to 1 device
   - Shared workstations: Higher limits with regular validation
3. **Revocation Workflow**: Create a standard procedure for device revocation when:
   - Employees leave the organization
   - Devices are lost or stolen
   - Suspicious activity is detected

### Performance Considerations

1. **Batch Size**: For very large operations (1000+ devices), consider splitting into multiple batches
2. **Concurrency Tuning**:
   - Increase for faster processing on reliable connections (e.g., `--concurrency 10`)
   - Decrease if experiencing timeouts or rate limiting (e.g., `--concurrency 3`)

### Data Management

1. **CSV Generation**: Use spreadsheet software or scripts to generate properly formatted CSV files
2. **Result Archiving**: Save operation results for audit history
3. **Inventory Management**: Maintain a central device inventory using export-info

## Examples

### Example 1: Revoking Compromised Devices

```bash
# Create CSV with compromised devices
cat > compromised.csv << EOF
Device ID,Token,Name,Operation
d1234567890abcdef,inv-abcdefg,John's Laptop,revoke
d9876543210fedcba,inv-abcdefg,John's Phone,revoke
EOF

# Revoke the devices
cws profiles invitations devices batch-operation \
  --csv-file compromised.csv \
  --output-file revocation_results.csv
```

### Example 2: Auditing All Devices

```bash
# Export all device information
cws profiles invitations devices export-info \
  --output-file device_audit.csv

# This can be further processed with tools like Excel, Python, etc.
```

### Example 3: Validating Specific Devices

```bash
# Create CSV with devices to validate
cat > validate_devices.csv << EOF
Device ID,Token,Name,Operation
d1111111111abcdef,inv-xxxxxxx,Research Lab 1,validate
d2222222222bcdefg,inv-yyyyyyy,Research Lab 2,validate
EOF

# Validate the devices
cws profiles invitations devices batch-operation \
  --csv-file validate_devices.csv \
  --output-file validation_results.csv
```

### Example 4: Emergency Access Revocation

In case of a security incident:

```bash
# Immediately revoke all devices
cws profiles invitations devices batch-revoke-all \
  --confirm \
  --output-file emergency_revocation.csv
```

## Troubleshooting

1. **Device Not Found**: 
   - Verify the device ID is correct
   - Check if the device was already revoked
   - Ensure the invitation token is still valid

2. **Registry Connection Issues**:
   - Verify AWS credentials and permissions
   - Check network connectivity to registry service
   - Retry with lower concurrency

3. **Partial Success**:
   - When some operations succeed and others fail, review the output file
   - Isolate failed operations and retry if appropriate

4. **CSV Format Problems**:
   - Ensure columns match the required format
   - Check for special characters or encoding issues
   - Use a text editor rather than spreadsheet software if encoding issues persist

## Technical Notes

- Device IDs are generated using device-specific hardware information
- The registry uses an eventually consistent storage model
- Operations are performed with atomic consistency where possible
- All operations are logged for audit purposes