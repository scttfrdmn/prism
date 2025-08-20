#!/bin/bash
# CloudWorkstation Template Creator
# Creates a new AMI template with the basic structure

set -e

# Check if template name provided
if [ -z "$1" ]; then
  echo "Usage: $0 <template-name>"
  echo "Example: $0 my-custom-template"
  exit 1
fi

TEMPLATE_NAME="$1"
TEMPLATE_FILE="templates/${TEMPLATE_NAME}.yml"

# Check if file already exists
if [ -f "$TEMPLATE_FILE" ]; then
  echo "Error: Template $TEMPLATE_FILE already exists"
  exit 1
fi

# Create template file
cat > "$TEMPLATE_FILE" << EOF
name: "New Template: $TEMPLATE_NAME"
base: "ubuntu-22.04-server-lts"
description: "Custom template for $TEMPLATE_NAME"

build_steps:
  - name: "System updates"
    script: |
      apt-get update -y && apt-get upgrade -y
      apt-get install -y build-essential curl wget software-properties-common git

  - name: "Install required packages"
    script: |
      # TODO: Add your package installation commands here
      apt-get install -y python3 python3-pip
      
      # Example: Install additional packages
      # apt-get install -y package1 package2 package3

  - name: "Configure services"
    script: |
      # TODO: Add service configuration here
      echo "Services configured" > /var/log/services-configured.log

  - name: "Setup user environment"
    script: |
      # Create default researcher user
      useradd -m -s /bin/bash researcher
      # Password authentication disabled - use SSH key authentication only
      usermod -aG sudo researcher
      
      # Create projects directory
      mkdir -p /home/researcher/projects
      chown -R researcher:researcher /home/researcher/projects

  - name: "Cleanup"
    script: |
      apt-get autoremove -y && apt-get autoclean
      rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

validation:
  - name: "System updated"
    command: "apt-get update -y"
    success: true
  
  - name: "Python installed"
    command: "python3 --version"
    success: true
  
  - name: "User setup"
    command: "id researcher"
    success: true

tags:
  Name: "$TEMPLATE_NAME"
  Type: "custom"
  Software: "Ubuntu,Python"
  Purpose: "development"

min_disk_size: 10
EOF

chmod +x "$TEMPLATE_FILE"

echo "Template created: $TEMPLATE_FILE"
echo "Edit the file to customize your template, then validate with:"
echo "  cws ami validate $TEMPLATE_NAME"
echo ""
echo "After validation, build the AMI with:"
echo "  cws ami build $TEMPLATE_NAME"