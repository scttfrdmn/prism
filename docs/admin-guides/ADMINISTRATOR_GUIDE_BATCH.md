# Administrator Guide: Batch Invitation Management

This guide provides Prism administrators with best practices and strategies for managing batch invitations at scale.

## Introduction

The batch invitation system in Prism v0.4.3 introduces powerful tools for managing large numbers of invitations efficiently. This guide will help administrators understand how to leverage these tools for various organizational needs.

## Security Considerations

### Permission Inheritance

Batch-created invitations inherit security constraints from the parent token. When using a parent token:

1. Child invitations cannot exceed parent permissions
2. If parent is device-bound, all children will be device-bound
3. If parent cannot be transferred, children cannot be transferable
4. Child max device counts cannot exceed parent max device count

### Security Recommendations

1. **Device Binding**: Enable device binding for all production invitations
2. **Hierarchical Structure**: Create organizational hierarchy of invitations
   ```
   Organization Admin (device-bound, can invite)
   └── Department Admins (device-bound, can invite)
       └── Team Members (device-bound, cannot invite)
   ```
3. **Permission Tiers**: Limit higher-tier permissions based on need
   - `admin` - Reserved for system administrators
   - `read_write` - For active contributors
   - `read_only` - For viewers, auditors, and reviewers

### Monitoring and Compliance

1. **Regular Audits**: Schedule regular exports of all invitations
   ```bash
   prism profiles invitations batch-export \
     --output-file $(date +%Y-%m-%d)_invitation_audit.csv
   ```

2. **Device Inventory**: Maintain device registry information
   ```bash
   prism profiles invitations devices export-info \
     --output-file $(date +%Y-%m-%d)_device_audit.csv
   ```

3. **Invitation Lifecycle**: Document invitation expiration and renewal policies

## Batch Management Strategies

### For Educational Institutions

Educational settings often require creating many similar invitations at the start of a term:

1. **Class Template**:
   ```csv
   Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
   Student-{{ID}},read_write,120,no,no,yes,2
   ```

2. **Generation Script**:
   ```bash
   #!/bin/bash
   echo "Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices" > class_invitations.csv
   while IFS=, read -r id name email; do
     echo "Student-$id,read_write,120,no,no,yes,2" >> class_invitations.csv
   done < student_roster.csv
   ```

3. **Batch Creation**:
   ```bash
   prism profiles invitations batch-create \
     --csv-file class_invitations.csv \
     --s3-config s3://university/cs101/config \
     --output-file invitations_result.csv \
     --include-encoded
   ```

4. **Distribution Strategy**: Use mail merge to email each student their invitation token

### For Enterprise Teams

Enterprises typically need more granular control and structured invitation management:

1. **Department Structure**:
   ```csv
   Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
   Engineering-Admin,admin,365,yes,no,yes,3
   Marketing-Admin,admin,365,yes,no,yes,3
   Finance-Admin,admin,365,yes,no,yes,3
   HR-Admin,admin,365,yes,no,yes,3
   ```

2. **Delegation Model**: Department admins create team invitations
   ```bash
   # Export department admin invitations with encoded data
   prism profiles invitations batch-export \
     --output-file department_admins.csv \
     --include-encoded
   
   # Process department_admins.csv to extract tokens
   # Then for each department:
   prism profiles invitations batch-create \
     --csv-file engineering_team.csv \
     --parent-token "TOKEN_FROM_DEPARTMENT_ADMIN" \
     --output-file engineering_invitations.csv
   ```

3. **Tracking Strategy**: Use consistent naming conventions and metadata

### For Open Source Projects

Open source projects often need to manage contributor access across many repositories:

1. **Contributor Tiers**:
   ```csv
   Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
   Core-Contributor-{{name}},read_write,365,yes,no,yes,3
   Regular-Contributor-{{name}},read_write,180,no,no,yes,2
   First-Time-Contributor-{{name}},read_only,60,no,no,yes,1
   ```

2. **Progressive Access**: Upgrade contributors through tiers as they demonstrate commitment
3. **Integration Strategy**: Connect with GitHub/GitLab for automatic invitation creation

## Automation Strategies

### Continuous Integration

Integrate batch invitation management into CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
name: User Onboarding

on:
  push:
    paths:
      - 'users/**'

jobs:
  process-invitations:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Install Prism
        run: |
          # Install Prism CLI
          
      - name: Process New Users
        run: |
          prism profiles invitations batch-create \
            --csv-file ./users/pending.csv \
            --output-file ./users/processed.csv
            
      - name: Archive Results
        uses: actions/upload-artifact@v2
        with:
          name: invitation-results
          path: ./users/processed.csv
```

### User Lifecycle Management

Create scripts for comprehensive user lifecycle management:

```bash
#!/bin/bash
# onboard.sh - Process new user onboarding

# Generate invitations
prism profiles invitations batch-create \
  --csv-file new_users.csv \
  --output-file new_invitations.csv \
  --include-encoded

# Send welcome emails with invitations
while IFS=, read -r name email token encoded rest; do
  if [[ "$name" != "Name" ]]; then # Skip header
    echo "Sending invitation to $name ($email)"
    echo -e "Subject: Welcome to Prism\n\nUse this invitation code to get started: $encoded" | \
      sendmail "$email"
  fi
done < new_invitations.csv

# Archive processed invitations
mkdir -p archive/$(date +%Y-%m-%d)
cp new_invitations.csv archive/$(date +%Y-%m-%d)/
```

### Scheduled Operations

Set up scheduled operations for maintenance tasks:

```bash
# Example crontab entries

# Daily invitation audit (midnight)
0 0 * * * prism profiles invitations batch-export --output-file /var/log/prism/invitations_$(date +\%Y\%m\%d).csv

# Weekly device audit (Sunday 1 AM)
0 1 * * 0 prism profiles invitations devices export-info --output-file /var/log/prism/devices_$(date +\%Y\%m\%d).csv

# Monthly expired invitation cleanup (1st of month, 2 AM)
0 2 1 * * /usr/local/bin/cleanup_expired_invitations.sh
```

## Scaling Considerations

### Performance Optimization

For managing large numbers of invitations:

1. **Concurrency Tuning**: Adjust based on system capabilities
   ```bash
   # For high-performance systems
   prism profiles invitations batch-create --concurrency 20 ...
   
   # For limited resources
   prism profiles invitations batch-create --concurrency 5 ...
   ```

2. **Batch Sizing**: Split very large batches (1000+) into multiple operations
3. **Resource Allocation**: Ensure sufficient memory when processing large CSV files

### Geographic Distribution

For multi-region organizations:

1. **Region-Specific Templates**: Create templates optimized for each region
2. **Delegation Strategy**: Assign regional administrators with appropriate parent tokens
3. **Naming Convention**: Include region codes in invitation names for clarity

## Disaster Recovery

### Backup Strategy

1. **Regular Exports**: Schedule regular exports of all invitations
   ```bash
   # Daily backup script
   #!/bin/bash
   BACKUP_DIR="/backup/prism/invitations"
   DATE=$(date +%Y-%m-%d)
   mkdir -p "$BACKUP_DIR/$DATE"
   
   # Export invitations with all data
   prism profiles invitations batch-export \
     --output-file "$BACKUP_DIR/$DATE/invitations.csv" \
     --include-encoded
   
   # Export device information
   prism profiles invitations devices export-info \
     --output-file "$BACKUP_DIR/$DATE/devices.csv"
   
   # Compress for archival
   tar -czf "$BACKUP_DIR/$DATE.tar.gz" -C "$BACKUP_DIR" "$DATE"
   rm -rf "$BACKUP_DIR/$DATE"
   ```

2. **Version Control**: Store invitation templates in version control
3. **Recovery Testing**: Regularly validate recovery procedures

### Emergency Procedures

1. **Revocation**: In case of security breach
   ```bash
   # Emergency revocation of all devices
   prism profiles invitations devices batch-revoke-all --confirm
   ```

2. **Invitation Regeneration**: Process to recreate critical invitations
3. **Communication Plan**: Templates for notifying users of security events

## Advanced Configuration

### Custom CSV Processors

Create custom processors for specialized workflows:

```python
#!/usr/bin/env python3
# process_hr_data.py - Convert HR records to Prism invitation format

import csv
import sys
import datetime

# Calculate expiration based on contract end date
def calculate_valid_days(contract_end):
    if not contract_end:
        return 90  # Default
    
    end_date = datetime.datetime.strptime(contract_end, '%Y-%m-%d')
    today = datetime.datetime.today()
    days = (end_date - today).days
    
    return max(30, days)  # Minimum 30 days

# Process HR data
with open(sys.argv[1], 'r') as infile, open(sys.argv[2], 'w') as outfile:
    reader = csv.DictReader(infile)
    writer = csv.writer(outfile)
    
    # Write header
    writer.writerow(['Name', 'Type', 'ValidDays', 'CanInvite', 'Transferable', 'DeviceBound', 'MaxDevices'])
    
    for row in reader:
        # Determine invitation type based on role
        inv_type = 'read_only'
        if row['Role'].lower() in ('manager', 'director', 'vp'):
            inv_type = 'admin'
        elif row['Role'].lower() in ('developer', 'engineer', 'designer'):
            inv_type = 'read_write'
            
        # Calculate valid days
        valid_days = calculate_valid_days(row['ContractEnd'])
        
        # Determine if they can invite others
        can_invite = 'yes' if row['Role'].lower() in ('manager', 'director', 'vp') else 'no'
        
        # Write invitation row
        writer.writerow([
            f"{row['FirstName']}-{row['LastName']}",
            inv_type,
            valid_days,
            can_invite,
            'no',
            'yes',
            2
        ])

print(f"Processed {sys.argv[1]} to {sys.argv[2]}")
```

### Notification Integration

Integrate with notification systems to inform users:

```python
#!/usr/bin/env python3
# send_invitations.py - Send invitations via email, Slack, etc.

import csv
import sys
import smtplib
import requests
from email.message import EmailMessage

# Configuration
SLACK_WEBHOOK = "https://hooks.slack.com/services/..."
EMAIL_FROM = "prism@example.com"
EMAIL_SERVER = "smtp.example.com"

# Read invitation results
with open(sys.argv[1], 'r') as infile:
    reader = csv.DictReader(infile)
    for row in reader:
        if row['Status'] != 'Success':
            continue
            
        name = row['Name']
        encoded = row['Encoded Data']
        
        # Send email notification
        msg = EmailMessage()
        msg.set_content(f"""
        Hello {name},
        
        You have been invited to Prism.
        
        Your invitation code is:
        {encoded}
        
        To accept this invitation, run:
        prism profiles accept-invitation --encoded '{encoded}' --name '{name}'
        
        This invitation will expire in {row['Valid Days']} days.
        """)
        
        msg['Subject'] = 'Prism Invitation'
        msg['From'] = EMAIL_FROM
        msg['To'] = f"{name.lower().replace(' ', '.')}@example.com"
        
        with smtplib.SMTP(EMAIL_SERVER) as s:
            s.send_message(msg)
            
        # Send Slack notification
        requests.post(SLACK_WEBHOOK, json={
            "text": f"Prism invitation sent to {name}."
        })
        
        print(f"Notifications sent for {name}")
```

## Conclusion

The batch invitation system provides powerful tools for managing Prism access at scale. By following the strategies and best practices in this guide, administrators can efficiently manage invitations across organizations of any size while maintaining security and control.

For more detailed information about specific interfaces, refer to the [Batch Invitation Interface Guide](./BATCH_INVITATION_INTERFACE_GUIDE.md).