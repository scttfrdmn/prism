# Research User Management Guide

Research users are persistent identities (username, UID/GID, SSH keys, home
directory) that survive across workspaces, so a person keeps the same login and
files whichever instance they connect to. This guide covers managing them with
the `prism user` command family.

For the end-user perspective, see the
[Research Users user guide](../user-guides/USER_GUIDE_RESEARCH_USERS.md).

## Command reference

```bash
prism user create <username> [flags]   # Create a research user
prism user list                        # List research users
prism user info <username>             # Show details for one user
prism user update <username> [flags]   # Update a user's settings
prism user delete <username>           # Delete a research user
prism user keys <command>              # Manage a user's SSH keys
```

## Creating users

```bash
# Minimal — generates an Ed25519 SSH key by default
prism user create alice

# With profile details
prism user create alice \
  --full-name "Alice Researcher" \
  --email alice@university.edu \
  --shell /bin/bash

# Skip automatic key generation (add one later with `prism user keys add`)
prism user create alice --generate-ssh-key=false

# Use RSA instead of Ed25519
prism user create alice --key-type rsa
```

Key flags: `--full-name`, `--email`, `--shell` (default `/bin/bash`),
`--generate-ssh-key` (default true), `--key-type` (`ed25519` | `rsa`, default
`ed25519`).

## Updating users

```bash
# Update profile fields
prism user update alice --full-name "Alice R." --email alice@lab.edu

# Manage secondary group membership
prism user update alice --add-groups gpu-users,data-team

# Toggle Docker access
prism user update alice --docker
prism user update alice --no-docker
```

## Inspecting users

```bash
prism user list                    # All research users
prism user info alice              # Details for one user
prism user info alice --output json
```

## SSH key management

SSH keys are managed under `prism user keys`:

```bash
prism user keys list alice                     # List a user's keys
prism user keys generate alice ed25519         # Generate a new key pair
prism user keys add alice ~/.ssh/id_rsa.pub    # Add an existing public key
prism user keys add alice ~/.ssh/id_rsa.pub --comment "Laptop key"
prism user keys remove alice <key-id>          # Remove a key
```

## Deleting users

```bash
prism user delete alice
```

## How research users reach workspaces

When you launch a workspace with `--research-user <name>` (or provision one into
a running workspace), Prism creates the Linux account with the stored UID/GID and
installs the user's SSH keys and home directory. The same identity is reused on
every workspace, so files and access are consistent across instances. Shared
storage (EFS) mounted into a workspace preserves per-user ownership.

## Best practices

- **Prefer Ed25519 keys** — the default for `create` and `keys generate`.
- **Use groups** (`--add-groups`) to grant capabilities like GPU or shared-data
  access rather than per-user tweaks.
- **Remove departed users** with `prism user delete` to reclaim UIDs and revoke
  access.
