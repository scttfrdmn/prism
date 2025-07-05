# CloudWorkstation Demo Setup Guide

This guide explains how to set up a proper environment for demonstrating CloudWorkstation to potential users.

## Prerequisites

1. **AWS Account Setup**
   - Ensure you have AWS credentials with EC2, SSM, and EFS permissions
   - Configure the AWS CLI with `aws configure`
   - Test AWS access with `aws sts get-caller-identity`

2. **CloudWorkstation Installation**
   - Install the latest CloudWorkstation release
   - Verify installation with `cws version`
   - Run `cws daemon start` to ensure the daemon is running

3. **Network Requirements**
   - Stable internet connection (preferably wired)
   - Access to AWS APIs (no firewall blocking)
   - Low-latency connection for smooth demo experience

## Template Preparation

Create and validate the following templates before the demo:

1. **Base Templates**
   ```bash
   # Create basic-ubuntu template
   cws ami template create basic-ubuntu --base ubuntu-22.04 --version 1.0.0
   
   # Create desktop-research template
   cws ami template create desktop-research --base basic-ubuntu --version 1.0.0
   ```

2. **Application Templates with Dependencies**
   ```bash
   # Create r-research template (depends on basic-ubuntu)
   cws ami template create r-research --base basic-ubuntu --version 1.0.0
   cws ami template dependency add r-research basic-ubuntu --version 1.0.0 --operator ">="
   
   # Create python-ml template (depends on basic-ubuntu)
   cws ami template create python-ml --base basic-ubuntu --version 2.0.0
   cws ami template dependency add python-ml basic-ubuntu --version 1.0.0 --operator ">="
   
   # Create data-science template (depends on both r-research and python-ml)
   cws ami template create data-science --base desktop-research --version 1.0.0
   cws ami template dependency add data-science r-research --version 1.0.0 --operator ">="
   cws ami template dependency add data-science python-ml --version 2.0.0 --operator ">="
   ```

3. **Version the Templates**
   ```bash
   # Create multiple versions of python-ml template
   cws ami template version increment python-ml minor
   # Make some changes to the template
   cws ami template version increment python-ml patch
   ```

4. **Validate Templates and Dependencies**
   ```bash
   # Check all dependencies are correctly set up
   cws ami template dependency check data-science
   
   # Validate the dependency graph
   cws ami template dependency graph data-science
   ```

## Pre-Launch Instances

Create instances ahead of time to avoid waiting during the demo:

```bash
# Launch r-research instance
cws launch r-research demo-r --region us-east-1

# Launch data-science instance
cws launch data-science demo-ds --region us-east-1 --size L
```

Save the connection information for these instances to use as fallbacks.

## Terminal Setup

1. **Font and Colors**
   - Use a large font size (at least 18pt)
   - Use a high-contrast color scheme
   - Consider a clean terminal theme like Solarized

2. **Command Preparation**
   - Create a text file with all demo commands for easy copy/paste
   - Test each command before the demo

3. **Screen Recording**
   - Set up screen recording software (OBS, QuickTime, etc.)
   - Record a backup demo ahead of time

## Backup Materials

Prepare the following backup materials in case of technical issues:

1. **Screenshots**
   - Capture screenshots of each step in the demo
   - Include successful command outputs
   - Show instance connection screens

2. **Pre-recorded Demo**
   - Record a full run-through of the demo
   - Ensure the recording shows all key features

3. **Slides**
   - Create backup slides explaining each step
   - Include architecture diagrams
   - Highlight key benefits

## Testing Checklist

Before the demo, verify:

- [  ] AWS credentials are working
- [  ] CloudWorkstation daemon is running
- [  ] All templates are created and versioned
- [  ] Dependencies are correctly configured
- [  ] Pre-launched instances are running
- [  ] Terminal is configured with proper font/colors
- [  ] Backup materials are ready and accessible
- [  ] Internet connection is stable
- [  ] Command reference file is prepared

## Common Issues and Solutions

1. **Slow AWS API Responses**
   - Use the `--region` flag to specify a closer region
   - Have pre-launched instances ready

2. **Template Not Found**
   - Ensure templates were created and published
   - Check for typos in template names

3. **Dependency Resolution Failures**
   - Verify templates exist in the expected versions
   - Check that all dependencies are properly configured

4. **Instance Launch Failures**
   - Check AWS service status
   - Verify subnet and security group configurations
   - Have fallback regions ready

5. **Network Issues**
   - Prepare to switch to cellular hotspot
   - Have backup screenshots/recording ready

## Post-Demo

After the demo, be prepared to:

1. **Answer Technical Questions**
   - Have reference documentation ready
   - Know how to explain the architecture

2. **Provide Next Steps**
   - Have clear instructions for users who want to try CloudWorkstation
   - Prepare a follow-up email template

3. **Clean Up Resources**
   - Terminate demo instances
   - Document any issues that occurred during the demo for fixes