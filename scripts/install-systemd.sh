#!/bin/bash
set -e

INSTALL_DIR="/opt/axiom"
BIN_DIR="/usr/local/bin"
CONFIG_DIR="/etc/axiom"
DATA_DIR="/var/lib/axiom"

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root"
   exit 1
fi

echo "Installing Axiom IDP..."

# Create directories
mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR"

# Create axiom user
if ! id -u axiom > /dev/null 2>&1; then
    useradd -r -s /bin/false -d "$DATA_DIR" axiom
fi

# Download/copy binary
if [ -z "$1" ]; then
    echo "Usage: $0 <path-to-axiom-binary>"
    exit 1
fi

cp "$1" "$BIN_DIR/axiom-server"
chmod +x "$BIN_DIR/axiom-server"

# Create systemd service
cat > /etc/systemd/system/axiom.service << 'EOF'
[Unit]
Description=Axiom IDP Server
After=network.target

[Service]
Type=simple
User=axiom
WorkingDirectory=/var/lib/axiom
ExecStart=/usr/local/bin/axiom-server
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Create default config if not exists
if [ ! -f "$CONFIG_DIR/.env" ]; then
    cat > "$CONFIG_DIR/.env" << 'EOF'
AXIOM_PORT=8080
AXIOM_HOST=0.0.0.0
AXIOM_ENV=production
AXIOM_LOG_LEVEL=info
AXIOM_DB_DRIVER=sqlite3
AXIOM_DB_URL=file:/var/lib/axiom/axiom.db
AXIOM_SESSION_SECRET=change-me-in-production
EOF
    chmod 600 "$CONFIG_DIR/.env"
fi

# Set permissions
chown -R axiom:axiom "$DATA_DIR" "$CONFIG_DIR"
chmod 750 "$DATA_DIR" "$CONFIG_DIR"

# Reload systemd
systemctl daemon-reload

echo "Installation complete!"
echo ""
echo "To start the service:"
echo "  sudo systemctl start axiom"
echo ""
echo "To enable at boot:"
echo "  sudo systemctl enable axiom"
echo ""
echo "To view logs:"
echo "  sudo journalctl -u axiom -f"
echo ""
echo "Configuration file: $CONFIG_DIR/.env"
