# Prism AWS IAM Permissions

This document defines the AWS IAM permissions Prism needs, why each is required,
and how to set them up. It reflects the permissions the current codebase actually
calls.

There are **two distinct policies**:

1. **User / daemon policy** — attached to the IAM user (or role) whose credentials
   you run Prism with. This is the one you attach to yourself. See
   [`prism-iam-policy.json`](../prism-iam-policy.json).
2. **Instance role** (`Prism-Instance-Profile-Role`) — attached to each *workspace*
   so it can use SSM and stop itself when idle. Prism **auto-creates** this on first
   launch; you only need to pre-provision it manually if your user lacks IAM
   permissions. See [`prism-instance-role-policy.json`](../prism-instance-role-policy.json).

An optional add-on policy covers compliance reporting and the template marketplace:
[`prism-iam-policy-optional.json`](../prism-iam-policy-optional.json).

---

## How Prism authenticates

Run Prism with AWS credentials from any standard source:

1. **Browser login (recommended)** — `aws login` (AWS CLI v2.32+), which uses IAM
   Identity Center / federated identity with no long-term keys.
2. **Long-term access keys** — `aws configure`, stored in `~/.aws/credentials`.
3. **An attached IAM role** — when running Prism on an EC2/ECS host.

See the [AWS Setup Guide](../user-guides/AWS_SETUP_GUIDE.md) for the full walkthrough.
Whatever the source, the identity needs the user/daemon policy below.

---

## Quick start

```bash
# Create the user/daemon policy from the checked-in JSON
aws iam create-policy \
  --policy-name PrismAccess \
  --policy-document file://prism-iam-policy.json

# Attach it to your IAM user
aws iam attach-user-policy \
  --user-name YOUR_USERNAME \
  --policy-arn arn:aws:iam::YOUR_ACCOUNT_ID:policy/PrismAccess
```

Prefer least privilege? Start with the **Core tier** below (EC2 + STS) and add
service blocks as you enable features. Prism degrades gracefully: if a permission
is missing, that feature is unavailable — it does not crash the tool.

---

## User / daemon policy — by service

The complete policy is [`prism-iam-policy.json`](../prism-iam-policy.json). Each
block below explains what it enables.

### EC2 — instance, storage, networking, keys (required)

Core lifecycle: `RunInstances`, `TerminateInstances`, `Start/StopInstances`,
`DescribeInstances`, `DescribeInstanceStatus`, `DescribeInstanceTypes`,
`DescribeInstanceTypeOfferings`, `DescribeImages`.

Storage: `CreateVolume`, `DeleteVolume`, `AttachVolume`, `DetachVolume`,
`ModifyVolume` (resize), `DescribeVolumes`.

Instance config & diagnostics: `ModifyInstanceAttribute`, `GetConsoleOutput`,
`CreateTags`, `DescribeTags`.

Networking: `DescribeVpcs`, `DescribeSubnets`, `DescribeAvailabilityZones`,
`DescribeRouteTables`, `DescribeSecurityGroups`, `CreateSecurityGroup`,
`Authorize/RevokeSecurityGroupIngress`.

Keys: `DescribeKeyPairs`, `ImportKeyPair`, `DeleteKeyPair`.

### EC2 spot pricing (recommended)

`DescribeSpotPriceHistory` — real-time spot rate lookups for cost estimates and
`--spot` launches.

### EC2 snapshots & AMIs (for snapshot / custom-AMI features)

`CreateImage`, `RegisterImage`, `DeregisterImage`, `ModifyImageAttribute`,
`CreateSnapshot`, `DeleteSnapshot`, `DescribeSnapshots` — used by
`prism snapshot`, `prism ami`, and template-to-AMI workflows.

### EC2 capacity reservations (for `prism capacity-block`)

`CreateCapacityReservation`, `DescribeCapacityReservations`,
`CancelCapacityReservation` — reserved GPU/ML capacity blocks.

### EFS — shared filesystems (recommended)

`CreateFileSystem`, `DeleteFileSystem`, `DescribeFileSystems`,
`Create/Delete/DescribeMountTargets`, `Create/DescribeTags` — `prism volume`
shared storage across workspaces.

### FSx for Lustre — high-performance storage (optional)

`CreateFileSystem`, `DeleteFileSystem`, `UpdateFileSystem`, `DescribeFileSystems`,
`TagResource`, `ListTagsForResource` — high-throughput scratch storage for HPC.

### S3 — file transfer, backups, EFS staging (recommended)

`CreateBucket`, `ListBucket`, `GetBucketLocation`, `GetObject`, `PutObject`,
`DeleteObject`, scoped to Prism-owned buckets:
`prism-temp-*` (SSM file push/pull), `prism-backups-*` (`prism backup`),
`prism-efs-*` (EFS staging), and `prism-transfer*` (S3-backed transfers).

### IAM — instance profile (recommended; scoped)

`GetRole`, `GetInstanceProfile`, `CreateRole`, `TagRole`, `AttachRolePolicy`,
`PutRolePolicy`, `CreateInstanceProfile`, `TagInstanceProfile`,
`AddRoleToInstanceProfile`, `PassRole` — **scoped to the Prism role/profile ARNs
only**. Lets Prism auto-create `Prism-Instance-Profile-Role` (see below). Without
it, workspaces launch but SSM and autonomous idle detection are unavailable.

### SSM — remote command execution (recommended)

`SendCommand`, `GetCommandInvocation`, `ListCommands`, `ListCommandInvocations`,
`DescribeInstanceInformation` — provisioning, EFS mounting, research-user setup,
and file operations without SSH keys.

### Pricing & Cost Explorer — cost features (recommended)

`pricing:GetProducts` — region-aware on-demand/spot rate lookups (via truffle).
`ce:GetCostAndUsage`, `ce:GetCostForecast` — billed-cost reconciliation, budget
tracking, discount/credit discovery, and storage cost analytics.

### CloudWatch — rightsizing & metrics (optional)

`GetMetricStatistics`, `GetMetricData`, `ListMetrics` — `prism` rightsizing
recommendations and volume/instance metrics.

### Service Quotas — launch preflight (recommended)

`GetServiceQuota`, `ListServiceQuotas` — checks your vCPU quota before a launch so
you get a clear message instead of an opaque `RunInstances` failure.

### STS — identity (required)

`GetCallerIdentity` — validates credentials and resolves the account ID used for
resource naming and DNS subdomains.

---

## Optional add-on policy

[`prism-iam-policy-optional.json`](../prism-iam-policy-optional.json) — attach only
if you use these features:

- **Compliance reporting** (`prism` compliance views): `organizations:ListPolicies`,
  `organizations:DescribeOrganization`, `artifact:ListReports`, `artifact:GetReport`.
- **Template marketplace registry** (`prism marketplace`): `dynamodb:*` on
  `prism-*` tables.

---

## Instance role — `Prism-Instance-Profile-Role`

This is a **separate role attached to each workspace**, not to you. Prism
auto-creates it on first launch (when your user policy includes the scoped `iam:*`
actions above). It is:

- **Trust policy**: `ec2.amazonaws.com` may `sts:AssumeRole`.
- **AWS-managed policy**: `AmazonSSMManagedInstanceCore` (SSM agent connectivity).
- **Inline policy** ([`prism-instance-role-policy.json`](../prism-instance-role-policy.json)):
  `ec2:CreateTags`, `DescribeTags`, `DescribeInstances`, `StopInstances` — lets the
  on-instance agent (spored) tag itself and stop the instance when idle.

**Pre-provisioning**: if end users can't be granted `iam:CreateRole`, an
administrator can create this role + instance profile once (named exactly
`Prism-Instance-Profile-Role` / `Prism-Instance-Profile`) and Prism will use it.

**Known limitation**: the auto-created inline policy does **not** grant the S3-read
or Route53 permissions that spored uses for its binary download and DNS
self-registration (`*.prismcloud.host`). Those rely on separately-provisioned
access; DNS registration and SSM file-pull to fresh instances may be unavailable
under the minimal auto-created role until this is closed. Track/adjust if you rely
on custom DNS.

---

## Permission tiers

**Core (minimum to launch)** — EC2 (instance/network/keys) + STS. You can launch,
stop, and terminate SSH-accessible workspaces. No shared storage, no SSM, no cost
analytics.

**Recommended (full single-user experience)** — Core + EFS + IAM (scoped) + SSM +
Pricing/Cost Explorer + Service Quotas + S3 + spot pricing. This is
`prism-iam-policy.json` and what most researchers should attach.

**Optional** — FSx, CloudWatch, snapshots/AMIs, capacity reservations (in the main
policy but only exercised by those features) plus the compliance/marketplace add-on.

---

## Security best practices

1. **Never use root credentials.** Create a dedicated IAM user or use IAM Identity
   Center (`aws login`).
2. **Scope by tag** where possible — Prism tags all resources `ManagedBy=Prism` /
   `Prism=true`, so you can add a condition limiting Prism to its own resources.
3. **Prefer short-lived credentials** (`aws login` / SSO) over long-term keys.
4. **Enable CloudTrail** to audit the API calls Prism makes.

---

## Troubleshooting

**`You are not authorized to perform this operation`** — attach the missing service
block from `prism-iam-policy.json`. The error names the action; find its block above.

**`User is not authorized to perform: iam:CreateRole`** — your user lacks the scoped
IAM permissions. Either add them, or have an admin pre-provision
`Prism-Instance-Profile-Role` (see above). Workspaces still launch (SSH-only);
SSM and idle detection are disabled.

**`AccessDenied` on cost or pricing calls** — add the Pricing / Cost Explorer block;
cost estimates fall back to a static table without it.

Verify your identity and permissions:

```bash
aws sts get-caller-identity
prism workspace launch "Basic Ubuntu (APT)" test --dry-run   # shows what would be created
```

---

## Getting help

- **IAM Policy Simulator**: https://policysim.aws.amazon.com/
- **CloudTrail**: review which API calls are denied
- **Issues**: https://github.com/scttfrdmn/prism/issues
