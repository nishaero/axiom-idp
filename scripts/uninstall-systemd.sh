#!/bin/bash
set -e

echo "Uninstalling Axiom IDP..."

# Stop service
sudo systemctl stop axiom 2>/dev/null || true
sudo systemctl disable axiom 2>/dev/null || true

# Remove service file
sudo rm -f /etc/systemd/system/axiom.service

# Remove binary
sudo rm -f /usr/local/bin/axiom-server

# Reload systemd
sudo systemctl daemon-reload

echo "Axiom IDP uninstalled"
echo ""
echo "To remove configuration and data:"
echo "  sudo rm -rf /etc/axiom"
echo "  sudo rm -rf /var/lib/axiom"
echo ""
echo "To remove axiom user:"
echo "  sudo userdel axiom"
