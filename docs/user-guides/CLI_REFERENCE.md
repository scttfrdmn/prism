# CLI Reference

Complete reference for the `prism` command-line interface.

---

## Workspace commands

Workspaces are cloud instances. All workspace operations follow the pattern `prism workspace <action> <name>`.

```bash
# Launch a new workspace
prism workspace launch <template> <name>
prism workspace launch python-ml my-project
prism workspace launch python-ml my-project --size L
prism workspace launch python-ml my-project --spot
prism workspace launch python-ml my-project --region us-east-1
prism workspace launch python-ml my-project --project brain-study
prism workspace launch python-ml my-project --instance-type m7i.2xlarge

# Time-to-live: auto-stop after a duration (e.g., 8h, 24h, 30m)
prism workspace launch python-ml my-project --ttl 8h
prism workspace launch r-research paper-analysis --ttl 24h

# DNS name: custom hostname (defaults to sanitized workspace name)
# Workspace registers as {name}.{account}.prismcloud.host
prism workspace launch python-ml my-project --dns ml-workspace
# → Accessible at ml-workspace.abc123.prismcloud.host

# Idle timeout override: override default idle detection threshold
prism workspace launch python-ml my-project --idle-timeout 2h

# List workspaces
prism workspace list                         # Running and stopped workspaces
prism workspace cost                         # Show cost breakdown and savings

# Connect to a workspace
prism workspace connect <name>               # Print SSH command + web service URLs

# Run a command remotely
prism workspace exec <name> -- <command>
prism workspace exec my-project -- python train.py

# File operations (via S3 relay)
prism workspace files list <name>            # List files on the instance
prism workspace files push <name> ./data.csv /home/ubuntu/data.csv
prism workspace files pull <name> /home/ubuntu/results.csv ./results.csv

# Open a web service (Jupyter, RStudio, etc.)
prism workspace web <name>                   # List available web URLs
prism workspace web <name> --open jupyter    # Open Jupyter in browser

# Stop / start / delete
prism workspace stop <name>                  # Stop (preserves storage, no compute cost)
prism workspace start <name>                 # Restart a stopped workspace
prism workspace delete <name>                # Permanently terminate

# Hibernation (preserves RAM state, reduces cost)
prism workspace hibernate <name>             # Hibernate
prism workspace resume <name>                # Resume from hibernation

# Instance status
prism workspace status <name>                # Detailed status including cloud-init progress
```

**Size options**: `XS`, `S`, `M`, `L`, `XL` — Prism selects an appropriate instance type for each.

**Global flags** that apply to all workspace commands:

| Flag | Description |
|------|-------------|
| `--profile <name>` | Use a specific Prism profile |
| `--region <region>` | Override the AWS region |
| `--dry-run` | Preview what would happen without making changes |
| `--ttl <duration>` | Auto-stop after duration (e.g., `8h`, `24h`, `30m`). spored enforces on-instance. |
| `--dns <name>` | Custom DNS record name. Registers as `{name}.{account}.prismcloud.host`. |
| `--idle-timeout <duration>` | Override idle detection threshold (default: profile policy) |

---

## Templates

```bash
prism templates                              # List all available templates
prism templates info <name>                  # Detailed info: packages, ports, inheritance
prism templates info python-ml
```

**Built-in templates**:

| Template | Description |
|----------|-------------|
| `python-ml` | Python, PyTorch, TensorFlow, Jupyter |
| `r-research` | R, RStudio Server, Bioconductor |
| `genomics` | GATK, BWA, samtools, STAR |
| `bioinformatics` | Conda, Snakemake, BioPython |
| `deep-learning` | CUDA, cuDNN, GPU-ready PyTorch |
| `data-science` | Pandas, scikit-learn, DuckDB |
| `hpc-base` | MPI, OpenMP, GCC, CMake |

---

## Profiles

Profiles map a name to an AWS credential profile and region. The daemon uses the active profile for all AWS operations.

```bash
prism profiles setup                          # Interactive wizard (recommended)
prism profiles add personal <name> --aws-profile <p> --region <r>   # Add explicitly
prism profiles list                           # Show all profiles (active profile marked)
prism profiles switch <name>                  # Make a profile active
prism profiles delete <name>                  # Remove a profile
```

---

## Storage (volumes)

Prism supports EFS (shared, multi-attach) and EBS (single-instance block) volumes.

```bash
# EFS volumes (shared network storage)
prism volume create <name>                   # Create an EFS volume
prism volume list                            # List all volumes
prism volume attach <volume> <workspace>     # Attach to a running workspace
prism volume detach <volume> <workspace>     # Detach
prism volume delete <name>                   # Delete permanently

# EBS volumes are created at launch time with --storage-size
prism workspace launch python-ml my-project --storage-size 100
```

---

## Budget & cost

```bash
prism budget list                            # Show all budgets and status
prism budget create <project> <amount>       # Create a $N budget for a project
prism budget status <project>                # Detailed spend and alerts

# Via project subcommand
prism project budget set <project> <amount>  # Set/update budget
prism project budget status <project>        # Budget status
prism project budget history <project>       # Spend history
prism project budget prevent-launches <project>   # Block new launches at budget limit
prism project budget allow-launches <project>     # Re-enable launches
```

---

## SSH keys

```bash
prism keys list                              # List SSH keys registered with Prism
prism keys generate                          # Generate a new key pair
prism keys add <path>                        # Register an existing public key
```

---

## Projects

Projects group workspaces and enforce budget limits. Optional — workspaces can run without a project.

```bash
prism project create <name>                  # Create a project
prism project create <name> --budget 1000    # Create with a $1000 budget
prism project list                           # List all projects
prism project info <name>                    # Project details, members, spend
prism project instances <name>               # Workspaces in this project
prism project members <name>                 # List members
prism project members <name> add <email> <role>     # Add member (owner/admin/viewer)
prism project members <name> remove <email>  # Remove member
prism project delete <name>                  # Delete (must have no running workspaces)
```

---

## Invitations

```bash
prism invitation send <project> <email>      # Invite a collaborator
prism invitation list <project>              # List invitations for a project
prism invitation mine                        # Invitations sent to you
prism invitation accept <id>                 # Accept an invitation
prism invitation decline <id>                # Decline
prism invitation revoke <id>                 # Cancel a pending invite (owner only)
```

---

## Admin / daemon

The daemon (`prismd`) runs on `localhost:8947` and auto-starts on first use.

```bash
prism admin daemon status                    # Health check and version
prism admin daemon start                     # Start explicitly (usually not needed)
prism admin daemon stop                      # Stop the daemon

prism admin policy list                      # List access control policies
prism admin quota show --instance-type t3.large   # Check AWS quota for a type
prism admin aws-health                       # AWS service health status
prism admin rightsizing analyze              # Instance sizing recommendations
```

---

## Security & compliance

```bash
prism security status                        # Security monitoring / audit status
prism security health                        # Run a security health check
prism security dashboard                     # Threat analysis and recommendations
prism security keychain                      # Keychain provider diagnostics

prism security compliance frameworks         # List supported frameworks
prism security compliance validate nist-800-171   # Validate against a framework
prism security compliance report hipaa       # Generate a compliance report
```

---

## Other commands

```bash
prism gui                                    # Launch the desktop GUI
prism init                                   # Guided first launch (validates AWS creds, launches a workspace)
prism version                                # Show current version
prism templates                              # Alias: same as prism templates list
```

---

## Getting help

Every command supports `--help`:

```bash
prism --help
prism workspace --help
prism workspace launch --help
```

Report bugs and request features: <https://github.com/scttfrdmn/prism/issues>
