# Prism Templates - Simple Guide

![Template Illustration](https://via.placeholder.com/800x200?text=Prism+Templates)

## What are Templates?

Templates are like recipes for creating cloud computers. Each template tells Prism exactly what software to install and how to set everything up.

Think of templates like cookie cutters - they help make sure your cloud computer has exactly the right shape and ingredients for your research!

## Template Basics

Every template has:

1. **A name** - like "python-research" or "r-research"
2. **A description** - explains what the template is good for
3. **Build steps** - instructions for setting up the computer
4. **Tests** - ways to check everything is working correctly

## Available Templates

Prism comes with several ready-made templates:

- **python-research**: Python with data science tools
- **r-research**: R with statistical packages
- **neuroimaging**: Tools for brain research
- **bioinformatics**: Tools for genetics research
- **gis-research**: Tools for maps and geography
- **desktop-research**: Full desktop environment

## Using Templates

Using a template is super easy:

```bash
prism launch python-research my-project
```

This tells Prism to:
1. Find the python-research template
2. Create a new cloud computer using that template
3. Name your new computer "my-project"

## Finding Template Information

To see all available templates:
```bash
prism templates
```

To learn more about a specific template:
```bash
prism template info python-research
```

## What's Inside a Template?

Templates are written in YAML format (a simple text format). Here's what a simple template looks like:

```yaml
name: python-research
description: Python environment with data science tools
base: ubuntu-22.04-server-lts
architecture: x86_64

build_steps:
  - name: Update system
    script: |
      apt-get update
      apt-get upgrade -y
      
  - name: Install Python packages
    script: |
      pip install numpy pandas scikit-learn jupyter

validation:
  - name: Check Python
    script: python --version
```

This template:
1. Starts with Ubuntu 22.04
2. Updates the system
3. Installs Python packages
4. Checks that Python is working

## Advanced: Creating Your Own Templates

If you want to create your own template:

1. Create a new YAML file in the `/templates` directory
2. Follow the format shown above
3. Test your template with `prism ami validate my-template.yaml`
4. Build it with `prism ami build my-template.yaml`

## Template Structure Explained

A template needs these main parts:

| Part | What it does |
|------|--------------|
| `name` | A unique name for your template |
| `description` | Explains what the template is for |
| `base` | The starting operating system |
| `architecture` | CPU type (x86_64 or arm64) |
| `build_steps` | Instructions for setting up the computer |
| `validation` | Tests to make sure everything works |

## Build Steps

Each build step has:
- A name (what the step does)
- A script (commands to run)
- An optional timeout (maximum time for the step)

## Validation Tests

Validation tests check that everything is working:
- Each test has a name and a script
- The script should return success (exit code 0) if everything is OK
- If any test fails, the template won't work

## Research User Integration (Phase 5A+)

🎉 **NEW**: Templates can now automatically create research users with persistent identities!

### Research-Enabled Templates

Some templates support automatic research user creation. These templates let you create persistent users that work across different instances:

```bash
# Launch with automatic research user creation
prism launch python-ml-research my-project --research-user alice
# ✅ Creates instance + research user 'alice' + SSH keys + EFS home directory
```

### Research User Template Configuration

Research-enabled templates include a `research_user` section:

```yaml
name: "Python ML Research (Research User Enabled)"
research_user:
  auto_create: true                    # Create research user automatically
  require_efs: true                    # Set up persistent home directory
  efs_mount_point: "/efs"              # Where to mount EFS storage
  install_ssh_keys: true              # Generate SSH keys automatically
  default_shell: "/bin/bash"           # Default shell for research users
  default_groups: ["research", "docker"]  # Groups for research users
```

### Benefits of Research User Templates

- **Persistent Identity**: Same username and files across all instances
- **Automatic Setup**: SSH keys and home directories created automatically
- **Cross-Template Compatible**: Same research user works with any template
- **Team Collaboration**: Multiple research users can share files

## Tips for Good Templates

1. **Keep it simple**: Only include what you really need
2. **Test thoroughly**: Make sure everything works
3. **Add comments**: Explain what complex commands do
4. **Set timeouts**: Some steps might take a long time
5. **Clean up**: Remove temporary files to save space
6. **Consider research users**: Add research user support for persistent identity

## Help with Templates

If you need help with templates:
- Check the full template documentation in TEMPLATE_FORMAT_ADVANCED.md
- Look at existing templates for examples
- Ask someone with more experience for help

Remember: Templates help make sure your research environment works the same way every time!