#!/bin/bash
set -e

# Diskimager Daemon Uninstallation Script
# This script removes diskimager systemd services

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

# Stop services
stop_services() {
    print_info "Stopping services..."

    if systemctl is-active --quiet diskimager-serve.service; then
        systemctl stop diskimager-serve.service
        print_success "Stopped diskimager-serve.service"
    else
        print_info "diskimager-serve.service is not running"
    fi

    if systemctl is-active --quiet diskimager-web.service; then
        systemctl stop diskimager-web.service
        print_success "Stopped diskimager-web.service"
    else
        print_info "diskimager-web.service is not running"
    fi
}

# Disable services
disable_services() {
    print_info "Disabling services..."

    if systemctl is-enabled --quiet diskimager-serve.service 2>/dev/null; then
        systemctl disable diskimager-serve.service
        print_success "Disabled diskimager-serve.service"
    fi

    if systemctl is-enabled --quiet diskimager-web.service 2>/dev/null; then
        systemctl disable diskimager-web.service
        print_success "Disabled diskimager-web.service"
    fi
}

# Remove systemd service files
remove_systemd_services() {
    print_info "Removing systemd service files..."

    if [ -f "$SYSTEMD_DIR/diskimager-serve.service" ]; then
        rm -f "$SYSTEMD_DIR/diskimager-serve.service"
        print_success "Removed diskimager-serve.service"
    fi

    if [ -f "$SYSTEMD_DIR/diskimager-web.service" ]; then
        rm -f "$SYSTEMD_DIR/diskimager-web.service"
        print_success "Removed diskimager-web.service"
    fi

    # Remove alias if exists
    if [ -L "$SYSTEMD_DIR/diskimager-collector.service" ]; then
        rm -f "$SYSTEMD_DIR/diskimager-collector.service"
    fi

    if [ -L "$SYSTEMD_DIR/diskimager-ui.service" ]; then
        rm -f "$SYSTEMD_DIR/diskimager-ui.service"
    fi
}

# Remove binary
remove_binary() {
    print_info "Removing binary..."

    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        rm -f "$INSTALL_DIR/$BINARY_NAME"
        print_success "Removed $BINARY_NAME from $INSTALL_DIR"
    else
        print_info "Binary not found in $INSTALL_DIR"
    fi
}

# Remove PID files
remove_pid_files() {
    print_info "Removing PID files..."

    rm -f /var/run/diskimager-serve.pid
    rm -f /var/run/diskimager-web.pid

    print_success "Removed PID files"
}

# Remove configuration and data
remove_data() {
    echo ""
    print_warning "The following directories contain configuration and data:"
    echo "  - Configuration: $CONFIG_DIR"
    echo "  - Data: $DATA_DIR"
    echo "  - Logs: $LOG_DIR"
    echo ""
    read -p "Do you want to remove configuration files? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [ -d "$CONFIG_DIR" ]; then
            rm -rf "$CONFIG_DIR"
            print_success "Removed configuration directory"
        fi
    fi

    read -p "Do you want to remove data directory? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [ -d "$DATA_DIR" ]; then
            rm -rf "$DATA_DIR"
            print_success "Removed data directory"
        fi
    fi

    read -p "Do you want to remove log directory? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [ -d "$LOG_DIR" ]; then
            rm -rf "$LOG_DIR"
            print_success "Removed log directory"
        fi
    fi
}

# Remove user and group
remove_user() {
    echo ""
    read -p "Do you want to remove the $USER user and group? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if id "$USER" &>/dev/null; then
            userdel "$USER" 2>/dev/null || print_warning "Failed to remove user $USER"
            print_success "Removed user $USER"
        fi

        if getent group "$GROUP" &>/dev/null; then
            groupdel "$GROUP" 2>/dev/null || print_warning "Failed to remove group $GROUP"
            print_success "Removed group $GROUP"
        fi
    fi
}

# Reload systemd
reload_systemd() {
    print_info "Reloading systemd daemon..."
    systemctl daemon-reload
    systemctl reset-failed 2>/dev/null || true
    print_success "Systemd daemon reloaded"
}

# Main uninstallation flow
main() {
    echo "============================================"
    echo "  Diskimager Daemon Uninstallation"
    echo "============================================"
    echo ""

    check_root

    print_warning "This will remove diskimager daemon services from your system"
    echo ""
    read -p "Continue with uninstallation? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Uninstallation cancelled"
        exit 0
    fi

    stop_services
    disable_services
    remove_systemd_services
    remove_binary
    remove_pid_files
    reload_systemd
    remove_data
    remove_user

    echo ""
    echo "============================================"
    print_success "Uninstallation completed!"
    echo "============================================"
    echo ""
}

# Run main
main
