# Batch Invitation System Guide

Prism v0.4.3 introduces a robust batch invitation system for efficiently managing multiple user invitations at once. This guide explains how to use this feature to streamline the process of sharing access to your Prism resources.

## Overview

The batch invitation system allows administrators to:

- Create multiple invitations at once from a CSV file
- Export invitation data to CSV for distribution
- Accept multiple invitations from a CSV file
- Track invitation results and failures

This is especially useful in educational and team environments where many users need access to Prism resources.

## Creating Batch Invitations

### CSV Format

Create a CSV file with the following columns:

```
Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Test User 1,read_only,30,no,no,yes,1
Test User 2,read_write,60,no,no,yes,2
Test Admin,admin,90,yes,no,yes,3
```

**Required fields:**
- **Name**: The recipient's name
- **Type**: One of `read_only`, `read_write`, or `admin`

**Optional fields:**
- **ValidDays**: Days until expiration (default: 30)
- **CanInvite**: Whether recipient can invite others (default: false, true for admin type)
- **Transferable**: Whether invitation can be transferred (default: false)
- **DeviceBound**: Whether invitation is bound to device (default: true)
- **MaxDevices**: Maximum number of authorized devices (default: 1)

### Command Line

Create batch invitations:

```bash
prism profiles invitations batch-create \
  --csv-file invitations.csv \
  --s3-config s3://my-bucket/config \
  --has-header \
  --output-file results.csv
```

**Options:**
- `--csv-file`: Path to the CSV file containing invitation details
- `--s3-config`: Optional S3 path to shared configuration
- `--parent-token`: Optional parent invitation token
- `--concurrency`: Number of concurrent invitation creations (default: 5)
- `--has-header`: Whether the CSV has a header row (default: true)
- `--output-file`: Path to export results
- `--include-encoded`: Include encoded invitations in output (default: false)

## Exporting Invitations

Export all active invitations to a CSV file:

```bash
prism profiles invitations batch-export \
  --output-file invitations.csv \
  --include-encoded
```

**Options:**
- `--output-file`: Path for the exported CSV file
- `--include-encoded`: Include encoded invitations in output (default: true)

## Accepting Batch Invitations

Accept multiple invitations from a CSV file:

```bash
prism profiles invitations batch-accept \
  --csv-file invitations.csv \
  --name-prefix "Team" \
  --has-header
```

**Options:**
- `--csv-file`: Path to CSV file containing encoded invitations
- `--name-prefix`: Optional prefix for created profile names
- `--has-header`: Whether the CSV has a header row (default: true)

## Security Features

The batch invitation system inherits all the security features of the secure profile system:

1. **Device Binding**: Invitations can be bound to specific devices
2. **Hierarchical Permissions**: Sub-invitations cannot exceed parent permissions
3. **Multi-Level Controls**: Fine-grained permission management
4. **Cross-Platform Security**: Works across different operating systems
5. **Registry Integration**: Centralized tracking of authorized devices

## Best Practices

1. **Use Secure Settings**: Keep default security settings (device binding enabled)
2. **Monitor Results**: Always check the results for any failures
3. **Limit Concurrency**: For large batches, control concurrency to avoid API rate limits
4. **Backup Results**: Always save the output file containing tokens and results
5. **Use Descriptive Names**: Include enough information in names to identify recipients

## Examples

### Create Invitations for a Class

```bash
# Create CSV with student information
cat > students.csv << EOF
Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Student 1,read_only,90,no,no,yes,2
Student 2,read_only,90,no,no,yes,2
Student 3,read_only,90,no,no,yes,2
EOF

# Create invitations and export results
prism profiles invitations batch-create \
  --csv-file students.csv \
  --output-file class_invitations.csv \
  --include-encoded
```

### Export for Distribution

```bash
# Export all current invitations
prism profiles invitations batch-export \
  --output-file distribution.csv \
  --include-encoded
```

### Accept Multiple Invitations

```bash
# Accept invitations with team prefix
prism profiles invitations batch-accept \
  --csv-file team_invitations.csv \
  --name-prefix "Team" \
  --has-header
```

## Troubleshooting

1. **CSV Format Issues**: Ensure CSV format is correct and contains required columns
2. **Permission Errors**: Check that you have sufficient permissions to create invitations
3. **Rate Limiting**: Reduce concurrency if experiencing API rate limits
4. **Missing Encoded Data**: Ensure the CSV for batch-accept includes the encoded invitation column

## Technical Notes

- CSV parsing supports flexible boolean formats: "yes"/"no", "true"/"false", "1"/"0"
- Case-insensitive invitation type matching for user convenience
- Worker pool pattern ensures efficient concurrent processing
- Thread-safe result collection with mutex protection
- Comprehensive validation ensures all invitations are well-formed