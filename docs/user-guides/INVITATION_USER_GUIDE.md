# Prism Invitation System User Guide

**Version**: v0.35.3+
**Target Audience**: Researchers, Students, Lab Administrators, Course Instructors

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Accepting Invitations](#accepting-invitations)
4. [Sending Invitations](#sending-invitations)
5. [Bulk Invitations](#bulk-invitations)
6. [Shared Tokens](#shared-tokens)
7. [Managing Invitations](#managing-invitations)
8. [Troubleshooting](#troubleshooting)

---

## Overview

The Prism invitation system enables research teams, university classes, and lab environments to onboard collaborators with **zero manual configuration**. From invitation acceptance to SSH key provisioning, the entire workflow is automated.

### Key Features

- 🎫 **Individual Invitations**: Send invitations to specific users via email
- 📧 **Bulk Invitations**: Invite entire classes or teams (50+ people)
- 🔗 **Shared Tokens**: Reusable tokens for workshops and large groups
- 👤 **Automatic Provisioning**: SSH keys, UID/GID, and home directories configured automatically
- 🔐 **Role-Based Access**: Assign roles (owner, admin, member, viewer) with appropriate permissions
- 🎨 **Multi-Interface**: Use CLI or GUI - whichever you prefer

---

## Quick Start

### For Invitation Recipients

```bash
# 1. Receive invitation token (via email or shared link)

# 2. Accept invitation (CLI)
prism invitation accept eyJhbGciOiJIUzI1NiIs...

# 3. Access granted automatically! Connect to workspaces:
prism workspace list  # See available workspaces
prism workspace connect my-workspace  # Connect via SSH
```

### For Project Owners

```bash
# Send individual invitation
prism invitation send researcher@university.edu --project ml-research --role member

# Send bulk invitations for a class
prism invitation bulk cs101-project students.csv --role member

# Create shared token for a workshop
prism invitation shared create "AI Workshop 2026" --limit 100 --expires 7d
```

---

## Accepting Invitations

### Method 1: CLI (Command Line)

**Step 1**: Receive your invitation token via email or Slack

**Step 2**: Accept the invitation
```bash
prism invitation accept YOUR_TOKEN_HERE
```

**Example**:
```bash
$ prism invitation accept eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

✅ Invitation Accepted
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Project: ML Research Lab
Role:    Member
Status:  Active

🔑 Research User Provisioned
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Username:  alice
UID/GID:   10001/10001
SSH Key:   ~/.prism/ssh_keys/id_ed25519
Home Dir:  /efs/home/alice

You now have access to all project workspaces!

Try: prism workspace list
```

**What Happens Automatically**:
1. ✅ You're added as a project member with the assigned role
2. ✅ Your research user account is created with a unique UID/GID
3. ✅ SSH keys are generated and stored in `~/.prism/ssh_keys/`
4. ✅ Your EFS home directory is configured at `/efs/home/your-username`
5. ✅ You can immediately connect to all project workspaces

---

### Method 2: GUI (Graphical Interface)

**Step 1**: Open Prism GUI
```bash
prism-gui
```

**Step 2**: Navigate to Invitations
- Click "Invitations" in the left sidebar
- Badge shows number of pending invitations

**Step 3**: Add and Accept Invitation
1. Paste your token in the "Invitation Token" field
2. Click "Add Invitation"
3. Review project details (name, role, message)
4. Click "Accept" button
5. Confirm in the dialog

**Visual Walkthrough**:
```
┌─────────────────────────────────────────────┐
│ Invitations (2 pending)                     │
├─────────────────────────────────────────────┤
│                                             │
│ Add Invitation                              │
│ ┌─────────────────────────────────────────┐ │
│ │ Paste token here...                     │ │
│ └─────────────────────────────────────────┘ │
│ [Add Invitation]                            │
│                                             │
│ Your Invitations                            │
│ ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓ │
│ ┃ ML Research Lab        Member  [Accept]┃ │
│ ┃ Invited 2 days ago     5 days remaining┃ │
│ ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛ │
└─────────────────────────────────────────────┘
```

---

## Sending Invitations

### Prerequisites

- You must be a project **owner** or **admin**
- The project must exist before sending invitations

### Individual Invitations

**CLI**:
```bash
# Basic invitation
prism invitation send user@university.edu --project my-project

# With custom role and message
prism invitation send researcher@lab.edu \
  --project genomics-lab \
  --role admin \
  --message "Welcome to the genomics research lab!"

# With custom expiration (default: 7 days)
prism invitation send student@university.edu \
  --project cs101 \
  --role member \
  --expires 14d
```

**GUI**:
1. Navigate to Projects → Select Project
2. Click "Invite Member" button
3. Fill in form:
   - Email address
   - Role (owner/admin/member/viewer)
   - Optional message
   - Expiration (default: 7 days)
4. Click "Send Invitation"

**Role Descriptions**:
- **Owner**: Full project control (budgets, members, deletion)
- **Admin**: Manage members and resources (cannot delete project)
- **Member**: Launch workspaces, manage own resources
- **Viewer**: Read-only access (view workspaces and data)

---

## Bulk Invitations

Perfect for university classes, workshops, and large teams.

### CSV Format

Create a CSV file with email addresses:

**students.csv**:
```csv
email
alice@university.edu
bob@university.edu
charlie@university.edu
```

**Or with roles**:
```csv
email,role
alice@university.edu,admin
bob@university.edu,member
charlie@university.edu,member
```

### Sending Bulk Invitations

**Step 1**: Check AWS Quota (recommended for large classes)
```bash
# Ensure you have sufficient AWS capacity
prism invitation quota-check --instance-type t3.medium --count 50

✅ Quota Check Passed
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Required vCPUs:    100 (50 × t3.medium)
Current Usage:     12 vCPUs
Quota Limit:       128 vCPUs
Available:         116 vCPUs

You have sufficient capacity for 50 workspaces.
```

**Step 2**: Send Bulk Invitations
```bash
# Using CSV file
prism invitation bulk my-project students.csv --role member

# Using email list (comma-separated)
prism invitation bulk my-project \
  "alice@edu,bob@edu,charlie@edu" \
  --role member \
  --message "Welcome to CS 101 Fall 2025!"

# Results
✅ Bulk Invitations Sent
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Sent:     48
Failed:    2 (invalid email format)
Skipped:   0 (already members)
Total:    50

Tokens have been generated and are ready for email delivery.
```

---

## Shared Tokens

Shared tokens are perfect for:
- 📚 University classes (students redeem the same token)
- 🎤 Conference workshops (QR code on slides)
- 👥 Large teams (single link shared via Slack/email)

### Creating Shared Tokens

**CLI**:
```bash
# Create basic shared token
prism invitation shared create "CS 101 Fall 2025" \
  --project cs101 \
  --limit 50 \
  --expires 7d

# With custom role and message
prism invitation shared create "ML Workshop ICML 2026" \
  --project ml-workshop \
  --role viewer \
  --limit 100 \
  --expires 2d \
  --message "Welcome to the ML workshop!"

# Response
✅ Shared Token Created
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Name:       CS 101 Fall 2025
Token:      https://prism.dev/invite/shared/abc123...
Redemption Limit: 50
Expires:    Nov 16, 2025 (7 days)
Role:       member

Share this URL with your students:
https://prism.dev/invite/shared/abc123def456...
```

**GUI**:
1. Navigate to Invitations → "Shared Tokens" tab
2. Click "Create Shared Token"
3. Fill in form:
   - Token name
   - Redemption limit (how many people can use it)
   - Expiration (e.g., 7d, 14d, 30d)
   - Role
   - Optional message
4. Click "Create"
5. Copy URL or display QR code

### Sharing Shared Tokens

**Method 1: URL** (Email/Slack)
```
Share this link: https://prism.dev/invite/shared/abc123...

Students can click the link and run:
prism invitation redeem abc123def456...
```

**Method 2: QR Code** (Presentations/Posters)
1. In GUI: Click "Show QR Code" button
2. Display QR code on slides/posters
3. Attendees scan with phone → Opens redemption URL
4. Run: `prism invitation redeem <token>`

**Method 3: Command** (Documentation)
```bash
# Include in course materials
prism invitation redeem abc123def456...
```

### Managing Shared Tokens

**View Active Tokens**:
```bash
prism invitation shared list --project my-project

Active Shared Tokens
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CS 101 Fall 2025
  Redeemed:  23/50
  Expires:   3 days remaining
  Role:      member

ML Workshop
  Redeemed:  87/100
  Expires:   12 hours remaining
  Role:      viewer
```

**Extend Expiration**:
```bash
prism invitation shared extend TOKEN_ID --days 7
```

**Revoke Token**:
```bash
prism invitation shared revoke TOKEN_ID
```

---

## Managing Invitations

### Viewing Your Invitations (Recipients)

**CLI**:
```bash
# List all your invitations
prism invitation list

My Invitations
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ML Research Lab       Member    Pending    5 days remaining
CS 101 Project        Viewer    Accepted   Joined 2 days ago
Genomics Pipeline     Admin     Declined   Declined 1 week ago
```

**GUI**:
- Navigate to Invitations
- View table with all invitations
- Filter by status: Pending / Accepted / Declined / Expired

### Viewing Sent Invitations (Senders)

**CLI**:
```bash
# List invitations you sent for a project
prism invitation sent --project my-project

Sent Invitations for "My Project"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
alice@edu     Admin     Accepted   2 days ago
bob@edu       Member    Pending    5 days remaining
charlie@edu   Member    Expired    Expired yesterday
```

**GUI**:
- Navigate to Projects → Select Project → "Members" tab
- View "Pending Invitations" section

### Declining Invitations

**CLI**:
```bash
# Decline with reason
prism invitation decline TOKEN --reason "Not available this semester"
```

**GUI**:
1. Navigate to Invitations
2. Select invitation
3. Click "Decline" button
4. Enter optional reason
5. Confirm

### Revoking Sent Invitations

**CLI**:
```bash
# Revoke an invitation you sent
prism invitation revoke INVITATION_ID
```

**GUI**:
1. Navigate to Projects → Select Project → Members
2. Find invitation in "Pending Invitations"
3. Click "Revoke" button

---

## Troubleshooting

### Invitation Acceptance Issues

**Problem**: "Invitation not found" error
**Solution**:
- Check token is copied correctly (no spaces/line breaks)
- Token may have expired (contact sender)
- Token may have been revoked (contact sender)

**Problem**: "Invitation already accepted"
**Solution**:
- You've already accepted this invitation
- Check `prism workspace list` to see your project workspaces

**Problem**: "Permission denied after accepting"
**Solution**:
- SSH keys may not be configured correctly
- Run: `prism user provision --fix-keys`
- Verify: `ls ~/.prism/ssh_keys/`

---

### SSH Connection Issues

**Problem**: "Permission denied (publickey)"
**Solution**:
```bash
# Check SSH key exists
ls ~/.prism/ssh_keys/id_ed25519

# Re-provision if missing
prism user provision

# Verify key is added to instances
prism user ssh-key status
```

**Problem**: "UID/GID mismatch - permission denied"
**Solution**:
- Your UID/GID should be consistent across all instances
- Run: `prism user status` to check UID/GID
- Contact admin if UID/GID is inconsistent

---

### Quota Issues (Bulk Invitations)

**Problem**: "Insufficient EC2 vCPU quota"
**Solution**:
```bash
# Check current quota
prism invitation quota-check --instance-type t3.medium --count 50

# If insufficient, options:
# 1. Request AWS quota increase (Service Quotas console)
# 2. Use smaller instance type
# 3. Send invitations in batches
# 4. Use existing instances (don't launch new)
```

---

### Shared Token Issues

**Problem**: "Redemption limit reached"
**Solution**:
- Token has reached maximum redemption limit
- Contact sender to create new token or extend limit

**Problem**: "Token expired"
**Solution**:
- Token expiration date has passed
- Contact sender to create new token or extend expiration

---

## Best Practices

### For Recipients

1. **Accept Promptly**: Invitations expire (typically 7 days)
2. **Save SSH Keys**: Backup `~/.prism/ssh_keys/` directory
3. **Check Permissions**: Run `prism workspace list` to verify access after accepting
4. **Communicate**: Contact sender if you encounter issues

### For Senders

1. **Use Appropriate Roles**: Don't grant more permissions than needed
2. **Check Quota First**: Run quota-check before bulk invitations
3. **Set Reasonable Expiration**: Default 7 days is usually sufficient
4. **Monitor Acceptance**: Check `prism invitation sent` to track acceptance rate
5. **Revoke Unused**: Revoke invitations that won't be accepted

### For Course Instructors

1. **Shared Tokens for Classes**: Easier than individual invitations
2. **QR Codes for In-Person**: Display QR code on first day of class
3. **Generous Limits**: Set limit slightly higher than enrollment (account for drops/adds)
4. **Extend if Needed**: Can extend expiration if students join late
5. **Viewer Role for Auditors**: Use viewer role for non-credit students

---

## Examples by Use Case

### University Course (CS 101 - 50 students)

**Setup**:
```bash
# 1. Create project
prism project create cs101-fall2025 --budget 500

# 2. Check quota
prism invitation quota-check --instance-type t3.medium --count 50

# 3. Create shared token
prism invitation shared create "CS 101 Fall 2025" \
  --project cs101-fall2025 \
  --limit 60 \
  --expires 30d \
  --role member

# 4. Share token via course website and QR code on syllabus
```

---

### Research Lab (10 members)

**Setup**:
```bash
# 1. Create project
prism project create genomics-lab --budget 2000

# 2. Invite PI and co-PI as owners
prism invitation send pi@university.edu --project genomics-lab --role owner
prism invitation send copi@university.edu --project genomics-lab --role owner

# 3. Invite postdocs as admins
prism invitation send postdoc1@university.edu --project genomics-lab --role admin
prism invitation send postdoc2@university.edu --project genomics-lab --role admin

# 4. Invite grad students and research assistants as members
prism invitation bulk genomics-lab students.csv --role member
```

---

### Conference Workshop (100 attendees)

**Setup**:
```bash
# 1. Create project
prism project create icml-ml-workshop --budget 100

# 2. Create shared token (short expiration - workshop duration only)
prism invitation shared create "ICML 2026 ML Workshop" \
  --project icml-ml-workshop \
  --limit 120 \
  --expires 2d \
  --role viewer

# 3. Generate QR code (via GUI)
# 4. Display QR code on slides
# 5. Attendees scan and redeem during workshop
```

---

## Security Considerations

### Token Security

- **Tokens are sensitive**: Treat them like passwords
- **Don't share publicly**: Only share with intended recipients
- **Expiration**: Set appropriate expiration (shorter for sensitive projects)
- **Revocation**: Revoke tokens if accidentally shared publicly

### SSH Key Security

- **Private keys stay private**: Never share `~/.prism/ssh_keys/id_ed25519`
- **Permissions**: Keys should be `0600` (read/write for owner only)
- **Backup**: Keep secure backup of SSH keys
- **Rotation**: Keys can be regenerated with `prism user ssh-key rotate`

### Role-Based Access

- **Principle of least privilege**: Grant minimum necessary role
- **Owner role sparingly**: Only project leads should be owners
- **Viewer for external**: Use viewer role for external collaborators
- **Regular audits**: Review project members periodically

---

## Additional Resources

- [Getting Started Guide](QUICK_START.md)
- [Research Users Guide](USER_GUIDE_RESEARCH_USERS.md)
- Project Management Guide
- API Documentation

---

## Support

**Issues**: https://github.com/scttfrdmn/prism/issues
**Documentation**: https://github.com/scttfrdmn/prism/tree/main/docs
**Discussions**: https://github.com/scttfrdmn/prism/discussions
