# Batch Invitation Examples

This document provides practical examples of using Prism's batch invitation system for common scenarios. These examples demonstrate how to efficiently manage multiple invitations in various settings.

## Example 1: Creating Invitations for a Research Team

### Scenario
You're setting up Prism access for a research team with different roles:
- Principal investigators need admin access
- Research associates need read/write access 
- Research assistants need read-only access

### Step 1: Create CSV File

```csv
Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Dr. Jane Smith,admin,365,yes,no,yes,3
Dr. Robert Chen,admin,365,yes,no,yes,3
Sarah Johnson,read_write,180,no,no,yes,2
Michael Wong,read_write,180,no,no,yes,2
Alex Taylor,read_only,90,no,no,yes,1
Chris Davis,read_only,90,no,no,yes,1
Pat Rivera,read_only,90,no,no,yes,1
```

Save this as `research_team.csv`.

### Step 2: Create Batch Invitations

```bash
prism profiles invitations batch-create \
  --csv-file research_team.csv \
  --s3-config s3://university-lab/shared-config \
  --output-file research_team_invitations.csv \
  --include-encoded
```

### Step 3: Share Invitations

The output file `research_team_invitations.csv` now contains encoded invitations that can be shared with team members.

For security reasons, consider sharing invitations individually rather than distributing the entire CSV file.

## Example 2: Setting Up a Classroom Environment

### Scenario
You're preparing Prism for a class of 30 students for a semester-long course.

### Step 1: Create Student Invitation Template

```csv
Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Student-1,read_write,120,no,no,yes,1
Student-2,read_write,120,no,no,yes,1
Student-3,read_write,120,no,no,yes,1
...
Student-30,read_write,120,no,no,yes,1
```

You can generate this file programmatically:

```bash
# Generate template CSV
echo "Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices" > class_invitations.csv
for i in {1..30}; do
  echo "Student-$i,read_write,120,no,no,yes,1" >> class_invitations.csv
done
```

### Step 2: Create Batch Invitations with S3 Configuration

```bash
prism profiles invitations batch-create \
  --csv-file class_invitations.csv \
  --s3-config s3://university/cs101/config \
  --concurrency 10 \
  --output-file class_results.csv \
  --include-encoded
```

### Step 3: Set Up a Web Portal

Create a simple web page where students can retrieve their individual invitation code using their student ID.

## Example 3: Enterprise Team with Hierarchical Permissions

### Scenario
You're setting up access for a corporate team with hierarchical permission structure.

### Step 1: Create Parent Invitation for Department Head

```bash
prism profiles invitations create "Department Head" --type admin --valid-days 365
```

This will output an invitation token. Note this token as `PARENT_TOKEN`.

### Step 2: Create Manager Invitations

Create CSV for team managers:
```csv
Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Team Manager 1,read_write,180,yes,no,yes,2
Team Manager 2,read_write,180,yes,no,yes,2
Team Manager 3,read_write,180,yes,no,yes,2
```

```bash
prism profiles invitations batch-create \
  --csv-file managers.csv \
  --parent-token PARENT_TOKEN \
  --output-file managers_invitations.csv \
  --include-encoded
```

### Step 3: Allow Managers to Create Team Member Invitations

Distribute the invitations to managers, who can then use their tokens to create invitations for their team members.

## Example 4: Time-Limited Workshop Access

### Scenario
You're running a 2-day workshop and need to provide temporary access to participants.

### Step 1: Create Short-Term Invitations

```csv
Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
Workshop-1,read_only,3,no,no,yes,1
Workshop-2,read_only,3,no,no,yes,1
Workshop-3,read_only,3,no,no,yes,1
...
Workshop-25,read_only,3,no,no,yes,1
```

### Step 2: Create and Distribute Invitations

```bash
prism profiles invitations batch-create \
  --csv-file workshop.csv \
  --s3-config s3://workshops/data-science-intro \
  --output-file workshop_invitations.csv \
  --include-encoded
```

## Example 5: Different Permission Levels

### Scenario
You need to create invitations with various permission levels for different team members.

### CSV with Mixed Permissions

```csv
Name,Type,ValidDays,CanInvite,Transferable,DeviceBound,MaxDevices
CTO,admin,365,yes,yes,no,5
Lead Developer,admin,365,yes,no,yes,3
Developer 1,read_write,180,no,no,yes,2
Developer 2,read_write,180,no,no,yes,2
Intern 1,read_only,90,no,no,yes,1
Intern 2,read_only,90,no,no,yes,1
Client Reviewer,read_only,30,no,no,yes,1
```

Note the different settings:
- CTO has transferable permissions and can use on 5 devices
- Lead Developer can invite others but permission is device-bound
- Developers have standard read/write access
- Interns and client have read-only access with different expiration periods

## Example 6: Accepting Multiple Invitations

### Scenario
You've received a CSV file with multiple invitations for different projects.

### Step 1: Save the Received CSV file as `received_invitations.csv`

### Step 2: Accept All Invitations

```bash
prism profiles invitations batch-accept \
  --csv-file received_invitations.csv \
  --name-prefix "Project-" \
  --has-header
```

This will create profiles named "Project-[Name]" for each invitation in the CSV.

## Example 7: Generating Custom Reports

### Scenario
You need to track all active invitations and generate a report.

### Step 1: Export All Current Invitations

```bash
prism profiles invitations batch-export \
  --output-file all_invitations.csv
```

### Step 2: Process the CSV

Use your preferred spreadsheet software or data processing tools to analyze the exported data:
- Filter by expiration date to find soon-to-expire invitations
- Group by permission type to audit access levels
- Count devices per user to monitor resource usage

## Best Practices

1. **Use Meaningful Names**: Include identifiers in names that help you track who each invitation belongs to
2. **Limit Validity Periods**: Set appropriate expiration dates based on project timelines
3. **Enable Device Binding**: Keep device binding enabled for better security
4. **Manage Parent Tokens**: Be careful with parent tokens that allow invitation creation
5. **Track Invitation Data**: Keep records of all invitations created for audit purposes
6. **Regular Exports**: Periodically export invitation data to maintain visibility
7. **Secure Distribution**: Use secure methods to distribute invitation tokens to recipients

## Troubleshooting

### Common Issues and Solutions

1. **CSV Format Problems**
   - Make sure the CSV file uses the correct column names
   - Check for special characters or formatting issues
   - Use a text editor instead of spreadsheet software if encoding issues occur

2. **Failed Invitations**
   - Check if the parent token has sufficient permissions
   - Verify that the invitation type is one of: read_only, read_write, admin
   - Ensure valid days is a positive integer

3. **Connection Issues**
   - Check AWS credentials and permissions
   - Verify that S3 configuration path is accessible
   - Ensure the daemon is running and accessible