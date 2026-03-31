# Diskimager Daemon Mode

Complete guide to running Diskimager as a systemd daemon for production deployments.

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Service Management](#service-management)
5. [Logging and Monitoring](#logging-and-monitoring)
6. [Security Considerations](#security-considerations)
7. [Troubleshooting](#troubleshooting)
8. [Uninstallation](#uninstallation)

## Overview

Diskimager provides two server modes that can run as systemd daemons:

1. **Collection Server (`diskimager-serve`)** - Network collection server with mTLS authentication
2. **Web UI Server (`diskimager-web`)** - REST API and WebSocket server with web interface

### Features

- **Systemd Integration**: Full Type=notify support with sd_notify
- **Graceful Shutdown**: Properly handles SIGTERM, SIGINT, and SIGHUP
- **In-Progress Job Handling**: Waits for active uploads/jobs before shutdown
- **Automatic Restart**: Configurable restart policies on failure
- **Journald Logging**: Structured logging to system journal
- **Security Hardening**: Sandboxed execution with minimal privileges
- **PID File Management**: Prevents duplicate instances
- **Health Monitoring**: Systemd watchdog support

## Installation

### Prerequisites

- Linux system with systemd (systemd 219+)
- Root or sudo access
- Go 1.19+ (for building from source)

### Quick Install

```bash
# Build the binary
go build -o diskimager

# Install as daemon (requires root)
sudo ./scripts/install-daemon.sh
```

The installation script will:
- Create `diskimager` system user and group
- Install binary to `/usr/local/bin/diskimager`
- Copy systemd service files to `/etc/systemd/system/`
- Create configuration directory at `/etc/diskimager/`
- Create data directory at `/var/lib/diskimager/`
- Create log directory at `/var/log/diskimager/`
- Set proper permissions
- Reload systemd daemon

### Manual Installation

```bash
# Create user and group
sudo useradd -r -s /bin/false -d /var/lib/diskimager diskimager

# Create directories
sudo mkdir -p /etc/diskimager /var/lib/diskimager /var/log/diskimager
sudo chown -R diskimager:diskimager /var/lib/diskimager /var/log/diskimager

# Install binary
sudo cp diskimager /usr/local/bin/
sudo chmod 755 /usr/local/bin/diskimager

# Install systemd units
sudo cp systemd/diskimager-serve.service /etc/systemd/system/
sudo cp systemd/diskimager-web.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload
```

## Configuration

### Collection Server Configuration

Edit `/etc/diskimager/serve.json`:

```json
{
  "server": {
    "bind_address": "0.0.0.0:8443",
    "storage_path": "/var/lib/diskimager/uploads",
    "auth_mode": "mtls",
    "tls_ca": "/etc/diskimager/certs/ca.pem",
    "tls_cert": "/etc/diskimager/certs/server.pem",
    "tls_key": "/etc/diskimager/certs/server-key.pem"
  }
}
```

**Configuration Options**:

- `bind_address` - Address to listen on (default: `0.0.0.0:8443`)
- `storage_path` - Directory for uploaded disk images
- `auth_mode` - Authentication mode: `mtls` (mutual TLS)
- `tls_ca` - Path to CA certificate for client verification
- `tls_cert` - Path to server certificate
- `tls_key` - Path to server private key

### Web UI Server Configuration

Edit `/etc/diskimager/web.json`:

```json
{
  "web": {
    "bind_address": "0.0.0.0:8080",
    "storage_path": "/var/lib/diskimager",
    "max_concurrent_jobs": 4,
    "enable_cors": true,
    "allowed_origins": ["*"]
  }
}
```

**Configuration Options**:

- `bind_address` - Address to listen on (default: `0.0.0.0:8080`)
- `storage_path` - Working directory for imaging jobs
- `max_concurrent_jobs` - Maximum concurrent imaging operations
- `enable_cors` - Enable CORS for web API
- `allowed_origins` - CORS allowed origins

### TLS Certificate Setup

For the collection server, you need to generate TLS certificates:

```bash
# Create certificate directory
sudo mkdir -p /etc/diskimager/certs

# Generate CA certificate
openssl req -x509 -newkey rsa:4096 -days 365 -nodes \
  -keyout /etc/diskimager/certs/ca-key.pem \
  -out /etc/diskimager/certs/ca.pem \
  -subj "/CN=Diskimager CA"

# Generate server certificate
openssl req -newkey rsa:4096 -nodes \
  -keyout /etc/diskimager/certs/server-key.pem \
  -out /etc/diskimager/certs/server-req.pem \
  -subj "/CN=diskimager-server"

openssl x509 -req -in /etc/diskimager/certs/server-req.pem \
  -CA /etc/diskimager/certs/ca.pem \
  -CAkey /etc/diskimager/certs/ca-key.pem \
  -CAcreateserial \
  -out /etc/diskimager/certs/server.pem \
  -days 365

# Set permissions
sudo chmod 600 /etc/diskimager/certs/*-key.pem
sudo chmod 644 /etc/diskimager/certs/*.pem
sudo chown -R root:diskimager /etc/diskimager/certs
```

## Service Management

### Using Built-in Daemon Command

```bash
# Show status of both services
diskimager daemon status

# Show status of specific service
diskimager daemon status serve
diskimager daemon status web

# Start services
sudo diskimager daemon start serve
sudo diskimager daemon start web

# Stop services
sudo diskimager daemon stop serve
sudo diskimager daemon stop web

# Restart services
sudo diskimager daemon restart serve
sudo diskimager daemon restart web

# View logs
diskimager daemon logs serve
diskimager daemon logs web -f  # Follow logs
diskimager daemon logs web -n 100  # Last 100 lines
```

### Using systemctl Directly

```bash
# Enable services to start on boot
sudo systemctl enable diskimager-serve
sudo systemctl enable diskimager-web

# Start services
sudo systemctl start diskimager-serve
sudo systemctl start diskimager-web

# Stop services
sudo systemctl stop diskimager-serve
sudo systemctl stop diskimager-web

# Restart services
sudo systemctl restart diskimager-serve
sudo systemctl restart diskimager-web

# Check status
sudo systemctl status diskimager-serve
sudo systemctl status diskimager-web

# View logs
journalctl -u diskimager-serve -f
journalctl -u diskimager-web -f
```

### Service Dependencies

Both services require network to be online:
- `After=network-online.target`
- `Wants=network-online.target`

The services will wait for network connectivity before starting.

## Logging and Monitoring

### Viewing Logs

**Using journalctl**:

```bash
# View recent logs
journalctl -u diskimager-serve -n 50

# Follow logs in real-time
journalctl -u diskimager-serve -f

# View logs since a specific time
journalctl -u diskimager-serve --since "1 hour ago"
journalctl -u diskimager-serve --since "2024-01-01"

# View logs with specific priority
journalctl -u diskimager-serve -p err  # Errors only
journalctl -u diskimager-serve -p warning  # Warnings and above

# Export logs
journalctl -u diskimager-serve --since today > diskimager-serve.log
```

**Log Levels**:

The daemons support the following log levels:
- DEBUG - Detailed diagnostic information
- INFO - General informational messages (default)
- WARN - Warning messages
- ERROR - Error messages
- FATAL - Fatal errors causing shutdown

Set log level via environment variable:
```bash
sudo systemctl edit diskimager-serve
# Add:
# [Service]
# Environment="DISKIMAGER_LOG_LEVEL=debug"
```

### Monitoring Service Health

**Check if service is running**:

```bash
systemctl is-active diskimager-serve
systemctl is-enabled diskimager-serve
```

**View service status**:

```bash
systemctl status diskimager-serve
```

**Monitor resource usage**:

```bash
systemd-cgtop
systemctl show diskimager-serve --property=MemoryCurrent
systemctl show diskimager-serve --property=CPUUsageNSec
```

### Systemd Watchdog

Both services support systemd watchdog (configured for 30 seconds):

```bash
# Check watchdog status
systemctl show diskimager-serve --property=WatchdogTimestamp
systemctl show diskimager-serve --property=WatchdogTimestampMonotonic
```

If the service doesn't send keepalive within 30 seconds, systemd will restart it.

## Security Considerations

### Service Hardening

The systemd service files include extensive security hardening:

**User Isolation**:
- Services run as dedicated `diskimager` user
- No shell access (`/bin/false`)
- Home directory isolated

**Filesystem Protection**:
- `PrivateTmp=true` - Private /tmp directory
- `ProtectSystem=strict` - Read-only system directories
- `ProtectHome=true` - Inaccessible home directories
- `ReadWritePaths` - Limited write access to data/log directories

**System Call Filtering**:
- `SystemCallFilter=@system-service` - Restricted to safe syscalls
- `SystemCallErrorNumber=EPERM` - Deny dangerous calls

**Capability Restrictions**:
- `NoNewPrivileges=true` - Cannot gain new privileges
- `ProtectKernelTunables=true` - No kernel parameter changes
- `ProtectKernelModules=true` - Cannot load kernel modules
- `ProtectControlGroups=true` - Read-only cgroups
- `RestrictAddressFamilies` - Limited network protocols
- `RestrictNamespaces=true` - No namespace creation
- `RestrictRealtime=true` - No realtime scheduling
- `RestrictSUIDSGID=true` - No SUID/SGID files
- `LockPersonality=true` - Prevent personality changes
- `MemoryDenyWriteExecute=true` - No RWX memory pages

### Network Security

**Collection Server**:
- Requires mTLS (mutual TLS) authentication
- All clients must present valid certificates
- TLS 1.3 enforced (minimum version)
- Certificate-based client identification

**Web UI Server**:
- Bind to localhost (`127.0.0.1`) for local-only access
- Use reverse proxy (nginx/caddy) for external access
- Enable HTTPS at reverse proxy layer
- Implement authentication at reverse proxy

### File Permissions

```bash
# Configuration files (sensitive)
sudo chmod 640 /etc/diskimager/*.json
sudo chown root:diskimager /etc/diskimager/*.json

# TLS private keys (highly sensitive)
sudo chmod 600 /etc/diskimager/certs/*-key.pem
sudo chown root:diskimager /etc/diskimager/certs/*-key.pem

# Data directory
sudo chmod 750 /var/lib/diskimager
sudo chown diskimager:diskimager /var/lib/diskimager

# Log directory
sudo chmod 750 /var/log/diskimager
sudo chown diskimager:diskimager /var/log/diskimager
```

### Firewall Configuration

```bash
# Allow collection server (mTLS)
sudo firewall-cmd --permanent --add-port=8443/tcp
sudo firewall-cmd --reload

# Web UI (if exposing externally - use with caution)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

## Troubleshooting

### Service Won't Start

**Check logs**:
```bash
journalctl -u diskimager-serve -n 50
systemctl status diskimager-serve
```

**Common issues**:

1. **Configuration file error**:
   ```bash
   # Validate JSON syntax
   jq . /etc/diskimager/serve.json
   ```

2. **TLS certificate issues**:
   ```bash
   # Check certificate validity
   openssl x509 -in /etc/diskimager/certs/server.pem -text -noout

   # Verify certificate matches key
   openssl x509 -noout -modulus -in /etc/diskimager/certs/server.pem | openssl md5
   openssl rsa -noout -modulus -in /etc/diskimager/certs/server-key.pem | openssl md5
   ```

3. **Port already in use**:
   ```bash
   sudo lsof -i :8443
   sudo netstat -tlnp | grep 8443
   ```

4. **Permission denied**:
   ```bash
   # Check directory permissions
   ls -la /var/lib/diskimager
   ls -la /etc/diskimager

   # Fix permissions
   sudo chown -R diskimager:diskimager /var/lib/diskimager
   ```

### Service Crashes or Restarts

**View crash logs**:
```bash
journalctl -u diskimager-serve --since "1 hour ago" -p err
coredumpctl list
coredumpctl info diskimager
```

**Check resource limits**:
```bash
systemctl show diskimager-serve --property=LimitNOFILE
systemctl show diskimager-serve --property=LimitNPROC
```

**Increase limits if needed**:
```bash
sudo systemctl edit diskimager-serve
# Add:
# [Service]
# LimitNOFILE=65536
# LimitNPROC=1024
```

### Graceful Shutdown Issues

**Check shutdown timeout**:
```bash
systemctl show diskimager-serve --property=TimeoutStopSec
```

**Adjust if uploads take longer**:
```bash
sudo systemctl edit diskimager-serve
# Add:
# [Service]
# TimeoutStopSec=60s
```

### High Memory Usage

**Monitor memory**:
```bash
systemctl status diskimager-serve
journalctl -u diskimager-serve | grep -i memory
```

**Set memory limits**:
```bash
sudo systemctl edit diskimager-serve
# Add:
# [Service]
# MemoryMax=2G
# MemoryHigh=1.5G
```

### Debug Mode

Enable debug logging:
```bash
sudo systemctl edit diskimager-serve
# Add:
# [Service]
# Environment="DISKIMAGER_LOG_LEVEL=debug"

sudo systemctl restart diskimager-serve
journalctl -u diskimager-serve -f
```

## Uninstallation

### Quick Uninstall

```bash
sudo ./scripts/uninstall-daemon.sh
```

The uninstall script will:
- Stop running services
- Disable services
- Remove systemd unit files
- Remove binary from `/usr/local/bin`
- Optionally remove configuration, data, and logs (with confirmation)
- Optionally remove user and group (with confirmation)
- Reload systemd daemon

### Manual Uninstallation

```bash
# Stop and disable services
sudo systemctl stop diskimager-serve diskimager-web
sudo systemctl disable diskimager-serve diskimager-web

# Remove systemd units
sudo rm /etc/systemd/system/diskimager-serve.service
sudo rm /etc/systemd/system/diskimager-web.service
sudo systemctl daemon-reload

# Remove binary
sudo rm /usr/local/bin/diskimager

# Remove configuration (optional)
sudo rm -rf /etc/diskimager

# Remove data (optional - contains disk images!)
sudo rm -rf /var/lib/diskimager

# Remove logs (optional)
sudo rm -rf /var/log/diskimager

# Remove user and group (optional)
sudo userdel diskimager
sudo groupdel diskimager
```

## Advanced Configuration

### Running Multiple Instances

You can run multiple instances with different configurations:

```bash
# Copy and modify service file
sudo cp /etc/systemd/system/diskimager-serve.service \
       /etc/systemd/system/diskimager-serve-alt.service

# Edit the new service file
sudo systemctl edit --full diskimager-serve-alt.service
# Change:
# - ExecStart config path
# - PID file path
# - Working directory

sudo systemctl daemon-reload
sudo systemctl start diskimager-serve-alt
```

### Integration with Reverse Proxy

**Nginx configuration** for web UI:

```nginx
server {
    listen 80;
    server_name diskimager.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name diskimager.example.com;

    ssl_certificate /etc/letsencrypt/live/diskimager.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/diskimager.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Monitoring with Prometheus

Export metrics to Prometheus:

```bash
# Add prometheus node_exporter for system metrics
sudo apt install prometheus-node-exporter

# Scrape systemd metrics
# Add to prometheus.yml:
# - job_name: 'diskimager'
#   static_configs:
#   - targets: ['localhost:9100']
```

## Best Practices

1. **Always run services as dedicated user** - Never run as root
2. **Use strong TLS certificates** - 4096-bit RSA or ECDSA P-384
3. **Rotate certificates regularly** - Set up automated renewal
4. **Monitor logs** - Set up centralized logging (ELK, Loki)
5. **Set up alerting** - Monitor service health and disk space
6. **Backup configuration** - Include in regular backup routine
7. **Test recovery** - Regularly test service restart and recovery
8. **Keep updated** - Update binary and dependencies regularly
9. **Resource limits** - Set appropriate memory and CPU limits
10. **Network isolation** - Use firewalls and network segmentation

## Support

For issues and questions:
- GitHub Issues: https://github.com/afterdarksys/diskimager/issues
- Documentation: https://github.com/afterdarksys/diskimager

## License

See LICENSE file for details.
