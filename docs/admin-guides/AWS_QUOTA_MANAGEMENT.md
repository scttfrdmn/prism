# AWS Quota Management

AWS imposes **service quotas** (formerly "limits") on how many resources an
account can run — most commonly a cap on total vCPUs per instance family per
region. Hitting one produces an opaque launch failure like *"The requested
configuration is currently not supported"* or `VcpuLimitExceeded`. Prism can
check the relevant quotas up front and generate an increase request so you find
out before a launch fails, not after.

## Common quota types

| Quota | Common default | What it limits |
|-------|----------------|----------------|
| Running On-Demand Standard (A, C, D, H, I, M, R, T, Z) | 32 vCPUs | Total vCPUs across standard families |
| Running On-Demand G and VT | 8 vCPUs | GPU instances (G, VT) |
| Running On-Demand P | 8 vCPUs | GPU instances (P family) |
| Running On-Demand X | 8 vCPUs | High-memory instances |
| EBS gp3 storage | 50 TiB | Total gp3 volume storage per region |

**Example**: with 24 vCPUs already running, launching a `p3.8xlarge` (32 vCPUs)
needs 56 total — over the default 32 vCPU standard quota, so the launch fails.

## Commands

Quota tooling lives under `prism admin quota`:

```bash
# List the AWS quotas relevant to Prism launches (current region)
prism admin quota list

# Show quota headroom for a specific instance type
prism admin quota show --instance-type t3.medium
prism admin quota show --instance-type g4dn.xlarge

# Generate guidance for requesting a quota increase
prism admin quota request --instance-type p3.8xlarge
```

`prism admin quota show` reports the quota that governs the given instance type
and how much of it is already in use, so you can tell whether a launch will fit.

## Requesting an increase

`prism admin quota request --instance-type <type>` produces the specific AWS
Service Quotas request to file (the quota name, region, and a suggested new
value). Submit it via the
[AWS Service Quotas console](https://console.aws.amazon.com/servicequotas/);
increases for standard families are often automatic, while large GPU/FPGA
requests may need AWS review.

## Required permission

Quota checks need `servicequotas:GetServiceQuota` and
`servicequotas:ListServiceQuotas` on the credentials Prism runs with — included
in the recommended [IAM policy](AWS_IAM_PERMISSIONS.md).
