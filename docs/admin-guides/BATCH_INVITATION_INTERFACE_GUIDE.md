# Batch Invitation Interface Guide

Prism v0.4.3 provides multiple interfaces for managing batch invitations, each designed for different usage scenarios. This guide explains how to use the batch invitation system through the GUI, TUI, and CLI interfaces.

## Overview

The batch invitation system is accessible through three interfaces:

1. **Graphical User Interface (GUI)** - For desktop users who prefer point-and-click operations
2. **Terminal User Interface (TUI)** - For terminal users who want visual feedback
3. **Command Line Interface (CLI)** - For scripting, automation, and remote administration

All interfaces use the same core functionality, ensuring consistent behavior regardless of which interface you choose.

## GUI Interface

The graphical interface provides a user-friendly way to manage batch invitations through the Prism desktop application.

### Accessing the Batch Invitation Interface

1. Launch the Prism desktop application
2. Navigate to the "Profiles" section in the sidebar
3. Select the "Batch Invitations" tab

### Create Batch Invitations

1. Select the "Create Invitations" tab
2. Click "Browse" to select a CSV file containing invitation details
3. Configure options:
   - S3 Config Path (optional): S3 path for shared configuration
   - Parent Token (optional): Token of parent invitation
   - CSV has header: Check if your CSV includes a header row
   - Concurrency: Number of parallel invitation creations
4. Click "Browse" to select an output file for results
5. Click "Create Invitations" to process the batch

For new users, click "Generate Template" to create a sample CSV format.

### Export Invitations

1. Select the "Export Invitations" tab
2. Click "Browse" to select an output file location
3. Click "Export Invitations" to export all current invitations to CSV
4. The current invitations table shows what will be exported

### Accept Invitations

1. Select the "Accept Invitations" tab
2. Click "Browse" to select a CSV file containing encoded invitations
3. Enter an optional name prefix for the created profiles
4. Check "CSV has header" if your file includes a header row
5. Click "Accept Invitations" to process the batch

### Results View

After each operation, the results panel shows:
- Operation type
- Total invitations processed
- Number of successful operations
- Number of failed operations
- Output file location (if applicable)
- Error messages (if any occurred)

You can click "Open CSV" to view the results file or "Open Folder" to browse the output directory.

## TUI Interface

The terminal user interface provides visual management of invitations in terminal environments.

### Accessing the Invitation Dashboard

1. Launch the Prism TUI: `prism tui`
2. Navigate to "Profiles" using Tab key or keyboard shortcuts
3. Select "Invitation Management" from the menu
4. Press Enter to access the invitation dashboard

### Dashboard Layout

The invitation dashboard shows:
- Summary statistics at the top (total invitations, types, expiring soon)
- Table of invitations with sorting options
- Detailed information panel for selected invitation
- Device binding information when available
- Action shortcuts at the bottom

### Keyboard Controls

- **↑/↓ or j/k**: Navigate through invitations
- **Enter**: View details for selected invitation
- **e**: Export all invitations to CSV
- **r**: Revoke selected invitation
- **f5**: Refresh the invitation list
- **?**: Show help screen
- **Esc**: Go back/close details view

### Batch Operations

1. Press **e** to export all invitations
2. In the export dialog:
   - Enter output file path using the file browser
   - Select export options (include encoded data, etc.)
   - Press Enter to confirm

### Viewing Device Information

1. Select an invitation in the dashboard
2. Press Enter to view details
3. Device binding information is shown in the details view
4. For invitations with multiple devices, the list shows all registered devices

## CLI Interface

The command line interface provides powerful batch operations that can be integrated into scripts and automation workflows.

### Create Batch Invitations

```bash
prism profiles invitations batch-create \
  --csv-file invitations.csv \
  --s3-config s3://bucket/path \
  --parent-token "inv-abcdefg" \
  --concurrency 10 \
  --output-file results.csv \
  --has-header
```

**Options:**
- `--csv-file`: Path to CSV file with invitation details (required)
- `--s3-config`: S3 path for shared configuration (optional)
- `--parent-token`: Token of parent invitation (optional)
- `--concurrency`: Number of parallel invitation creations (default: 5)
- `--has-header`: Whether CSV has a header row (default: true)
- `--output-file`: Path to export results (optional)
- `--include-encoded`: Include encoded data in output (default: false)

### Export All Invitations

```bash
prism profiles invitations batch-export \
  --output-file invitations.csv \
  --include-encoded
```

**Options:**
- `--output-file`: Path for exported CSV file (default: "invitations.csv")
- `--include-encoded`: Include encoded invitations in output (default: true)

### Accept Batch Invitations

```bash
prism profiles invitations batch-accept \
  --csv-file encoded_invitations.csv \
  --name-prefix "Team" \
  --has-header
```

**Options:**
- `--csv-file`: Path to CSV file with encoded invitations (required)
- `--name-prefix`: Prefix for created profile names (optional)
- `--has-header`: Whether CSV has a header row (default: true)

### Device Management

```bash
# Batch device operations
prism profiles invitations devices batch-operation \
  --csv-file devices.csv \
  --operation revoke \
  --output-file results.csv

# Export device information
prism profiles invitations devices export-info \
  --output-file device_info.csv

# Revoke all devices
prism profiles invitations devices batch-revoke-all \
  --confirm \
  --output-file revocation_results.csv
```

## CSV Format Reference

### Input CSV for Creating Invitations

```csv
Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Test User 1,read_only,30,no,no,yes,1
Test User 2,read_write,60,no,no,yes,2
Test Admin,admin,90,yes,no,yes,3
```

**Required columns:**
- `Name`: Recipient name
- `Type`: One of `read_only`, `read_write`, or `admin`

**Optional columns:**
- `ValidDays`: Days until expiration (default: 30)
- `CanInvite`: Whether recipient can invite others (default: false, true for admin)
- `Transferable`: Whether invitation can be transferred (default: false)
- `DeviceBound`: Whether invitation is bound to device (default: true)
- `MaxDevices`: Maximum number of authorized devices (default: 1)

### Output CSV Format

```csv
Name,Type,Token,Valid Days,Can Invite,Transferable,Device Bound,Max Devices,Status,Encoded Data,Error
Test User 1,read_only,inv-abcdefg,30,no,no,yes,1,Success,BASE64ENCODEDDATA,
Test User 2,read_write,inv-hijklmn,60,no,no,yes,2,Success,BASE64ENCODEDDATA,
Test User 3,read_write,,,,,,,Failed,,Invalid type
```

## Interface Comparison

| Feature | GUI | TUI | CLI |
|---------|-----|-----|-----|
| Create batch invitations | ✅ | ❌ | ✅ |
| Export all invitations | ✅ | ✅ | ✅ |
| Accept batch invitations | ✅ | ❌ | ✅ |
| Device management | ✅ | ✅ | ✅ |
| Invitation dashboard | ✅ | ✅ | ❌ |
| File dialogs | ✅ | ✅ | ❌ |
| Preview CSV content | ✅ | ❌ | ❌ |
| Script automation | ❌ | ❌ | ✅ |
| Remote operation | ❌ | ✅ | ✅ |
| No dependencies | ❌ | ✅ | ✅ |

## Best Practices

### For GUI Users

1. **Preview First**: Always check the CSV preview before creating invitations
2. **Template Usage**: Use the "Generate Template" button for first-time users
3. **Naming Convention**: Establish a consistent naming convention for invitations
4. **Output Management**: Create a dedicated folder for invitation CSV exports

### For TUI Users

1. **Regular Audits**: Use the dashboard to review invitations regularly
2. **Device Verification**: Check device binding status for security-critical invitations
3. **Export Backups**: Regularly export all invitations as backup
4. **Screen Size**: Ensure terminal size is at least 100x30 for optimal display

### For CLI Users

1. **Scripting**: Integrate batch operations into onboarding/offboarding scripts
2. **Error Handling**: Capture and analyze output files for error detection
3. **Validation**: Use `--dry-run` (when available) before large batch operations
4. **Automation**: Schedule regular exports for compliance and audit purposes

## Advanced Usage

### Combining Interfaces

You can use multiple interfaces together for different tasks:
- GUI for initial setup and template generation
- TUI for day-to-day management and monitoring
- CLI for scheduled backups and batch operations

### Integration with Other Systems

The batch invitation system can be integrated with:
- Identity management systems via CSV export/import
- User onboarding workflows through CLI automation
- Audit and compliance processes through scheduled exports
- Collaboration tools via output file sharing

## Troubleshooting

### Common GUI Issues

- **File not found**: Ensure file paths don't contain special characters
- **Preview fails**: Check CSV format and encoding (UTF-8 recommended)
- **Operation hangs**: Try reducing concurrency for large batches

### Common TUI Issues

- **Display issues**: Resize terminal to at least 100x30
- **Slow refresh**: Try disabling detail view for large invitation lists
- **Navigation problems**: Check keyboard layout compatibility

### Common CLI Issues

- **Permission denied**: Check file permissions for input/output files
- **Invalid CSV**: Verify CSV format and try with `--has-header=false`
- **Rate limiting**: Reduce concurrency for large batch operations

For all other issues, check logs at `~/.prism/logs/batch_operations.log`