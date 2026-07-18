# Installation

Install Prism on macOS, Windows, or Linux. Each install provides the `prism`
CLI and the `prismd` daemon; the desktop GUI ships as a `.dmg`/`.msi` and in
Homebrew.

After installing, verify:

```bash
prism version
```

Then continue to [AWS Setup](AWS_SETUP_GUIDE.md) and the
[Quick Start](QUICK_START.md).

## macOS

=== "Homebrew (recommended)"

    ```bash
    brew install scttfrdmn/tap/prism
    prism version
    ```

    Includes the `prism` CLI, `prismd` daemon, and (where available) the
    `prism-gui` desktop app. Update with `brew upgrade`.

=== "Desktop app (.dmg)"

    1. Download `Prism-v*.dmg` from the
       [releases page](https://github.com/scttfrdmn/prism/releases/latest)
       (universal — Intel and Apple Silicon).
    2. Open the `.dmg` and drag **Prism.app** to **Applications**.
    3. First launch: macOS Gatekeeper may block the unsigned app — right-click
       **Prism.app** → **Open** → **Open** (once).

## Windows

=== "Scoop (recommended)"

    ```powershell
    scoop bucket add scttfrdmn https://github.com/scttfrdmn/scoop-bucket
    scoop install prism
    prism version
    ```

=== "MSI installer"

    1. Download `Prism-v*.msi` from the
       [releases page](https://github.com/scttfrdmn/prism/releases/latest).
    2. Run the installer (adds `prism`/`prismd` to your `PATH`).

    Silent install:
    ```powershell
    msiexec /i Prism-v*.msi /quiet
    ```

## Linux

=== "Release archive"

    Download the latest archive for your platform from the
    [releases page](https://github.com/scttfrdmn/prism/releases/latest), extract,
    and move the binaries onto your `PATH`:

    ```bash
    tar xz -f prism_<version>_linux_amd64.tar.gz
    sudo mv prism prismd /usr/local/bin/
    prism version
    ```

=== "Build from source"

    ```bash
    git clone https://github.com/scttfrdmn/prism.git
    cd prism
    make build
    sudo make install
    ```

    Requires Go 1.26+. See the repository for GUI build prerequisites.

## Next steps

- **[AWS Setup](AWS_SETUP_GUIDE.md)** — authenticate (`aws login`) and the IAM
  permissions Prism needs
- **[Quick Start](QUICK_START.md)** — launch your first workspace
- **[Troubleshooting](TROUBLESHOOTING.md)** — common install and daemon issues
