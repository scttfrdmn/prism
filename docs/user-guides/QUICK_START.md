# Getting Started

Get a research workstation running in under 5 minutes.

## 1. Install Prism

=== "macOS (Homebrew)"
    ```bash
    brew install scttfrdmn/tap/prism
    ```

=== "Windows (Scoop)"
    ```powershell
    scoop bucket add scttfrdmn https://github.com/scttfrdmn/scoop-bucket
    scoop install prism
    ```

=== "Linux"
    Download the latest release from the [releases page](https://github.com/scttfrdmn/prism/releases/latest), extract, and add to your PATH.

Verify:
```bash
prism version
```

---

## 2. Authenticate with AWS

Prism uses your AWS credentials to launch instances in your account.

**Step 1 — Log in** (requires AWS CLI v2.32+):

```bash
aws login
```

This opens a browser window. Sign in with your IAM user or federated identity. Credentials are cached for up to 12 hours and refresh automatically.

> **No browser?** Use `aws login --remote` for cross-device authentication, or `aws configure` to set up long-term access keys (see [AWS Setup Guide](AWS_SETUP_GUIDE.md)).

Verify it works:
```bash
aws sts get-caller-identity
```

**Step 2 — Add a Prism profile**:

```bash
prism profiles add
```

This interactive wizard links a Prism profile to your AWS credentials and default region. You only need to do this once.

---

## 3. First-time wizard (optional)

If this is your first time running Prism, `prism init` validates your AWS credentials, walks you through picking a template and size, and launches your first workspace — all in one guided flow:

```bash
prism init
```

---

## 4. Launch a workspace

```bash
# See what's available
prism templates

# Launch a Python + Jupyter environment (~2 minutes)
prism workspace launch python-ml my-project

# Check it's ready
prism workspace list
```

---

## 5. Connect

```bash
prism workspace connect my-project
```

Prints the SSH command and any web service URLs (Jupyter, RStudio, etc.).

---

## 6. Stop when done

```bash
prism workspace stop my-project
```

Stopped workspaces have no compute cost. Resume later with `prism workspace start my-project`, or delete permanently with `prism workspace delete my-project`.

Hibernation preserves RAM state and reduces cost further:
```bash
prism workspace hibernate my-project
prism workspace resume my-project
```

---

## Common templates

| Template | What's included |
|----------|-----------------|
| `python-ml` | Python, PyTorch, TensorFlow, Jupyter |
| `r-research` | R, RStudio Server, Bioconductor |
| `genomics` | GATK, BWA, samtools, STAR |
| `bioinformatics` | Conda, Snakemake, BioPython |
| `deep-learning` | CUDA, cuDNN, GPU-ready PyTorch |
| `data-science` | Pandas, scikit-learn, DuckDB |
| `hpc-base` | MPI, OpenMP, GCC, CMake |

Run `prism templates info <name>` for details on any template.

---

## Two interfaces

### CLI
The `prism` command is the primary interface — scriptable and pipeline-friendly:
```bash
prism workspace launch python-ml my-project --size L --spot --ttl 8h
```

### GUI (desktop app)
A visual desktop app for managing workspaces, storage, and settings:
```bash
prism gui
```
Or launch from your Applications folder (macOS) / Start menu (Windows).

---

## Common workflows

### Data science
```bash
prism workspace launch python-ml data-analysis --size L
prism workspace connect data-analysis
# Jupyter available at http://localhost:8888 via SSH tunnel
```

### R / statistics
```bash
prism workspace launch r-research stats-project
prism workspace connect stats-project
# RStudio Server at http://localhost:8787 via SSH tunnel
```

### Shared storage (EFS)
```bash
prism volume create shared-datasets
prism volume attach shared-datasets my-project
```

### Time-limited workspace
```bash
prism workspace launch python-ml my-project --ttl 8h --dns my-ws
# Auto-stops after 8 hours; accessible at my-ws.abc123.prismcloud.host
```

---

## Troubleshooting

**"Daemon not running"**
```bash
prism admin daemon status
prism admin daemon stop     # Then run any prism command to auto-restart
```

**"AWS credentials not found"**
```bash
aws sts get-caller-identity   # Verify credentials are valid
aws login                     # Re-authenticate if expired
```

**Instance launch fails**
```bash
prism workspace launch python-ml my-project --region us-east-1  # Try different region
```

For more, see the [Troubleshooting Guide](TROUBLESHOOTING.md).

---

## Next steps

- **[CLI Reference](CLI_REFERENCE.md)** — all commands and flags
- **[GUI Guide](GUI_USER_GUIDE.md)** — desktop app walkthrough
- **[Workspace Lifecycle](WORKSPACE_LIFECYCLE.md)** — TTL, DNS, idle detection
- **[Templates](TEMPLATE_FORMAT.md)** — template format and customization
- **[AWS Setup Guide](AWS_SETUP_GUIDE.md)** — IAM permissions, long-term keys, multiple profiles
