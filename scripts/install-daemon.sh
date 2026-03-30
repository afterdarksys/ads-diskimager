#!/bin/bash
set -e

# Diskimager Daemon Installation Script
# This script installs diskimager as a systemd service

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="diskimager"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/diskimager"
DATA_DIR="/var/lib/diskimager"
LOG_DIR="/var/log/diskimager"
RUN_DIR="/var/run"
SYSTEMD_DIR="/etc/systemd/system"
USER="diskimager"
GROUP="diskimager"

# Print functions
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Detect the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Find the binary
find_binary() {
    if [ -f "$PROJECT_ROOT/$BINARY_NAME" ]; then
        BINARY_PATH="$PROJECT_ROOT/$BINARY_NAME"
    elif [ -f "$PROJECT_ROOT/bin/$BINARY_NAME" ]; then
        BINARY_PATH="$PROJECT_ROOT/bin/$BINARY_NAME"
    elif command -v "$BINARY_NAME" &> /dev/null; then
        BINARY_PATH=$(command -v "$BINARY_NAME")
    else
        print_error "Cannot find $BINARY_NAME binary"
        print_info "Please build the binary first: go build -o $BINARY_NAME"
        exit 1
    fi
}

# Create user and group
create_user() {
    print_info "Creating system user and group..."

    if id "$USER" &>/dev/null; then
        print_warning "User $USER already exists"
    else
        useradd -r -s /bin/false -d "$DATA_DIR" -c "Diskimager Service Account" "$USER"
        print_success "Created user $USER"
    fi

    if getent group "$GROUP" &>/dev/null; then
        print_warning "Group $GROUP already exists"
    else
        groupadd -r "$GROUP"
        print_success "Created group $GROUP"
    fi
}

# Create directories
create_directories() {
    print_info "Creating directories..."

    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$LOG_DIR"

    # Set ownership
    chown -R "$USER:$GROUP" "$DATA_DIR"
    chown -R "$USER:$GROUP" "$LOG_DIR"

    # Set permissions
    chmod 755 "$CONFIG_DIR"
    chmod 750 "$DATA_DIR"
    chmod 750 "$LOG_DIR"

    print_success "Created directories"
}

# Copy binary
install_binary() {
    print_info "Installing binary..."

    cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
    chmod 755 "$INSTALL_DIR/$BINARY_NAME"

    print_success "Installed $BINARY_NAME to $INSTALL_DIR"
}

# Copy systemd service files
install_systemd_services() {
    print_info "Installing systemd service files..."

    if [ -f "$PROJECT_ROOT/systemd/diskimager-serve.service" ]; then
        cp "$PROJECT_ROOT/systemd/diskimager-serve.service" "$SYSTEMD_DIR/"
        print_success "Installed diskimager-serve.service"
    else
        print_warning "diskimager-serve.service not found"
    fi

    if [ -f "$PROJECT_ROOT/systemd/diskimager-web.service" ]; then
        cp "$PROJECT_ROOT/systemd/diskimager-web.service" "$SYSTEMD_DIR/"
        print_success "Installed diskimager-web.service"
    else
        print_warning "diskimager-web.service not found"
    fi
}

# Copy example configuration files
install_configs() {
    print_info "Installing example configuration files..."

    if [ -f "$PROJECT_ROOT/config/serve.example.json" ] && [ ! -f "$CONFIG_DIR/serve.json" ]; then
        cp "$PROJECT_ROOT/config/serve.example.json" "$CONFIG_DIR/serve.json"
        chmod 640 "$CONFIG_DIR/serve.json"
        chown root:$GROUP "$CONFIG_DIR/serve.json"
        print_success "Installed serve.json (please edit with your settings)"
    fi

    if [ -f "$PROJECT_ROOT/config/web.example.json" ] && [ ! -f "$CONFIG_DIR/web.json" ]; then
        cp "$PROJECT_ROOT/config/web.example.json" "$CONFIG_DIR/web.json"
        chmod 640 "$CONFIG_DIR/web.json"
        chown root:$GROUP "$CONFIG_DIR/web.json"
        print_success "Installed web.json (please edit with your settings)"
    fi
}

# Reload systemd
reload_systemd() {
    print_info "Reloading systemd daemon..."
    systemctl daemon-reload
    print_success "Systemd daemon reloaded"
}

# Enable services (optional)
enable_services() {
    echo ""
    read -p "Do you want to enable diskimager-serve service on boot? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        systemctl enable diskimager-serve.service
        print_success "Enabled diskimager-serve.service"
    fi

    read -p "Do you want to enable diskimager-web service on boot? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        systemctl enable diskimager-web.service
        print_success "Enabled diskimager-web.service"
    fi
}

# Start services (optional)
start_services() {
    echo ""
    read -p "Do you want to start diskimager-serve service now? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        systemctl start diskimager-serve.service
        print_success "Started diskimager-serve.service"
    fi

    read -p "Do you want to start diskimager-web service now? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        systemctl start diskimager-web.service
        print_success "Started diskimager-web.service"
    fi
}

# Print usage instructions
print_usage() {
    echo ""
    echo "============================================"
    print_success "Installation completed successfully!"
    echo "============================================"
    echo ""
    echo "Configuration files:"
    echo "  - Collection Server: $CONFIG_DIR/serve.json"
    echo "  - Web UI Server:     $CONFIG_DIR/web.json"
    echo ""
    echo "Data directory:   $DATA_DIR"
    echo "Log directory:    $LOG_DIR"
    echo ""
    echo "Service management commands:"
    echo "  - View status:    systemctl status diskimager-serve"
    echo "                   systemctl status diskimager-web"
    echo ""
    echo "  - Start services: systemctl start diskimager-serve"
    echo "                   systemctl start diskimager-web"
    echo ""
    echo "  - Stop services:  systemctl stop diskimager-serve"
    echo "                   systemctl stop diskimager-web"
    echo ""
    echo "  - Restart:        systemctl restart diskimager-serve"
    echo "                   systemctl restart diskimager-web"
    echo ""
    echo "  - Enable on boot: systemctl enable diskimager-serve"
    echo "                   systemctl enable diskimager-web"
    echo ""
    echo "  - View logs:      journalctl -u diskimager-serve -f"
    echo "                   journalctl -u diskimager-web -f"
    echo ""
    echo "Built-in daemon management:"
    echo "  - $BINARY_NAME daemon status [serve|web]"
    echo "  - $BINARY_NAME daemon start [serve|web]"
    echo "  - $BINARY_NAME daemon stop [serve|web]"
    echo "  - $BINARY_NAME daemon restart [serve|web]"
    echo "  - $BINARY_NAME daemon logs [serve|web]"
    echo ""
    print_warning "IMPORTANT: Edit configuration files before starting services!"
    echo "  - Update TLS certificate paths in $CONFIG_DIR/serve.json"
    echo "  - Update storage paths and bind addresses as needed"
    echo ""
}

# Main installation flow
main() {
    echo "============================================"
    echo "  Diskimager Daemon Installation"
    echo "============================================"
    echo ""

    check_root
    find_binary

    print_info "Binary found at: $BINARY_PATH"
    print_info "Installation directory: $INSTALL_DIR"
    print_info "Configuration directory: $CONFIG_DIR"
    print_info "Data directory: $DATA_DIR"
    print_info "Log directory: $LOG_DIR"
    echo ""

    read -p "Continue with installation? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Installation cancelled"
        exit 0
    fi

    create_user
    create_directories
    install_binary
    install_systemd_services
    install_configs
    reload_systemd
    enable_services
    start_services
    print_usage
}

# Run main
main
