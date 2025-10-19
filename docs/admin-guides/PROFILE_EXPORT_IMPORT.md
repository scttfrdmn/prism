# CloudWorkstation Profile Export/Import Guide

CloudWorkstation v0.4.2 introduces profile export and import functionality, allowing users to:

1. Back up their CloudWorkstation profiles
2. Share profile configurations between machines
3. Transfer profiles to team members

This document explains how to use these features effectively.

## Profile Export

You can export your profiles to a file using the command line interface:

```bash
cws profiles export my-profiles.zip
```

This creates a ZIP file containing your profile configurations.

### Export Options

Several options are available for customizing your exports:

| Option | Description |
|--------|-------------|
| `--include-credentials` | Include AWS credentials (use with caution) |
| `--include-invitations` | Include invitation profiles (default: true) |
| `--profiles profile1,profile2` | Export only specific profiles |
| `--format zip|json` | Export format (default: zip) |
| `--password password` | Password protect the export (zip format only) |

### Example Uses

**Export all profiles without credentials (safest option):**
```bash
cws profiles export my-profiles.zip
```

**Export specific profiles:**
```bash
cws profiles export personal-profiles.zip --profiles personal,work
```

**Export with credentials (only for personal backups):**
```bash
cws profiles export full-backup.zip --include-credentials
```

**Export in JSON format:**
```bash
cws profiles export profiles.json --format json
```

## Profile Import

You can import profiles from a previously exported file:

```bash
cws profiles import my-profiles.zip
```

### Import Options

Several options control how imports are handled:

| Option | Description |
|--------|-------------|
| `--mode skip\|overwrite\|rename` | How to handle conflicts (default: rename) |
| `--profiles profile1,profile2` | Import only specific profiles |
| `--import-credentials` | Import credentials if available |
| `--password password` | Password for encrypted imports |

### Handling Profile Conflicts

When importing profiles, conflicts can occur if profiles with the same ID already exist. Three resolution modes are available:

1. **rename** (default): Rename imported profiles to avoid conflicts
2. **skip**: Skip importing profiles that would conflict
3. **overwrite**: Replace existing profiles with imported ones

### Example Uses

**Import all profiles, renaming any conflicts:**
```bash
cws profiles import my-profiles.zip
```

**Import only specific profiles:**
```bash
cws profiles import team-profiles.zip --profiles team-project,shared
```

**Import and skip any profiles that already exist:**
```bash
cws profiles import my-profiles.zip --mode skip
```

**Import with credentials:**
```bash
cws profiles import my-profiles.zip --import-credentials
```

## Security Considerations

### Credential Handling

By default, credentials are **not** included in exports for security reasons. This prevents accidental sharing of AWS access keys.

**For personal backups only**, you can include credentials with the `--include-credentials` flag. However, this should be used with caution:

- Always store export files with credentials securely
- Consider using password protection (`--password`)
- Never share exports containing credentials with others

### Password Protection

For sensitive exports, particularly those including credentials, you can add password protection:

```bash
cws profiles export secure-backup.zip --include-credentials --password "my-secure-password"
```

When importing, provide the same password:

```bash
cws profiles import secure-backup.zip --password "my-secure-password"
```

## Sharing with Teams

The export/import functionality is particularly useful for teams who need to share common CloudWorkstation configurations.

**Best practice for sharing with teams:**

1. Create profiles without credentials
2. Export without credentials
3. Share the export file with team members
4. Team members import the profiles
5. Each team member configures their own credentials

```bash
# Team lead:
cws profiles export team-profiles.zip --profiles team-project,shared

# Team members:
cws profiles import team-profiles.zip
```

## Working with Invitation Profiles

When exporting invitation profiles:

- The invitation token is exported
- The invitation's expiration status is preserved
- The recipient's ability to use the invitation depends on whether the invitation is still valid

To exclude invitation profiles from export:

```bash
cws profiles export personal-only.zip --include-invitations=false
```

## File Formats

### ZIP Format

The default export format is ZIP, which includes:

- `profiles.json` - Profile configurations
- `credentials/` directory (if credentials included)
- Metadata files

### JSON Format

For simpler integration with other tools, you can export in plain JSON format:

```bash
cws profiles export profiles.json --format json
```

Note that JSON exports cannot include credentials.

## Troubleshooting

### Common Issues

1. **Import fails with "invalid profiles format"**: The import file may be corrupted or created with an incompatible version.

2. **Credentials not imported**: Credentials are only imported if:
   - They were included in the export (`--include-credentials`)
   - The import was run with `--import-credentials`

3. **Profiles missing after import**: Check if:
   - The profiles were filtered out during export/import
   - There were naming conflicts and the chosen mode (`--mode`) skipped the profiles

### Getting Help

For additional assistance with profile export/import:

```bash
cws profiles export --help
cws profiles import --help
```

## Version Compatibility

Profile export/import is available in CloudWorkstation v0.4.2 and later. Exports created with newer versions may not be compatible with older versions of CloudWorkstation.