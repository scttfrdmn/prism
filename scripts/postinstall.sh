#!/bin/bash
# Post-installation script for CloudWorkstation Linux packages

set -e

echo "CloudWorkstation has been installed!"
echo ""
echo "Quick start:"
echo "  cws templates                           # List available templates"
echo "  cws launch <template-name> <instance>   # Launch a workstation"
echo ""
echo "The daemon (cwsd) starts automatically when needed - no setup required!"
echo ""
echo "AWS credentials are required. Configure with:"
echo "  aws configure"
echo ""
echo "Documentation: https://github.com/scttfrdmn/cloudworkstation"
