# Diskimager Daemon - Quick Usage Guide

## Quick Start

### 1. Build the Binary
```bash
go build -o diskimager
```

### 2. Install as Daemon (Linux only)
```bash
sudo ./scripts/install-daemon.sh
```

### 3. Start Services
```bash
# Start collection server
sudo systemctl start diskimager-serve

# Start web UI server
sudo systemctl start diskimager-web

# Or use built-in command
sudo diskimager daemon start serve
sudo diskimager daemon start web
```

## Usage Examples

### Running in Foreground (Development)

**Collection Server**:
```bash
# Run with basic settings
./diskimager serve --config config/serve.example.json

# Run with custom PID file
./diskimager serve --config config.json --pid-file /tmp/diskimager-serve.pid

# Run with syslog logging
./diskimager serve --config config.json --syslog
```

**Web UI Server**:
```bash
# Run on default port
./diskimager web

# Run on custom port
./diskimager web --port 9090

# Run with syslog logging
./diskimager web --port 8080 --syslog
```

### Running as Daemon (Production)

**Start with systemd**:
```bash
sudo systemctl start diskimager-serve
sudo systemctl start diskimager-web
```

**Enable on boot**:
```bash
sudo systemctl enable diskimager-serve
sudo systemctl enable diskimager-web
```

**Check status**:
```bash
systemctl status diskimager-serve
systemctl status diskimager-web
```

### Managing Services

**Using built-in daemon command**:
```bash
# Show status
diskimager daemon status
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
diskimager daemon logs web -f              # Follow logs
diskimager daemon logs serve -n 100        # Last 100 lines
```

**Using systemctl**:
```bash
# Start/Stop/Restart
sudo systemctl start diskimager-serve
sudo systemctl stop diskimager-serve
sudo systemctl restart diskimager-serve

# Enable/Disable on boot
sudo systemctl enable diskimager-serve
sudo systemctl disable diskimager-serve

# View logs
journalctl -u diskimager-serve
journalctl -u diskimager-serve -f          # Follow
journalctl -u diskimager-serve -n 50       # Last 50 lines
journalctl -u diskimager-serve --since today
```

### Signal Handling

The daemons gracefully handle:

- **SIGTERM** - Graceful shutdown (systemd stop)
- **SIGINT** - Graceful shutdown (Ctrl+C)
- **SIGHUP** - Reload/restart signal

```bash
# Graceful shutdown (waits for uploads to complete)
sudo systemctl stop diskimager-serve

# Immediate restart
sudo systemctl restart diskimager-serve

# Reload configuration (if implemented)
sudo systemctl reload diskimager-serve
```

### Configuration

**Collection Server** (`/etc/diskimager/serve.json`):
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

**Web UI Server** (command-line flags):
```bash
diskimager web --port 8080 --daemon
```

### Monitoring

**Check service health**:
```bash
# Is the service running?
systemctl is-active diskimager-serve

# Is it enabled on boot?
systemctl is-enabled diskimager-serve

# Detailed status
systemctl status diskimager-serve
```

**View resource usage**:
```bash
# CPU and memory usage
systemd-cgtop

# Specific service metrics
systemctl show diskimager-serve --property=MemoryCurrent
systemctl show diskimager-serve --property=CPUUsageNSec
```

**Monitor logs in real-time**:
```bash
# Follow collection server logs
journalctl -u diskimager-serve -f

# Follow web server logs
journalctl -u diskimager-web -f

# Filter by priority
journalctl -u diskimager-serve -p err        # Errors only
journalctl -u diskimager-serve -p warning    # Warnings and above
```

### Troubleshooting

**Service won't start**:
```bash
# Check logs for errors
journalctl -u diskimager-serve -n 50

# Check configuration syntax
jq . /etc/diskimager/serve.json

# Verify TLS certificates
openssl x509 -in /etc/diskimager/certs/server.pem -text -noout

# Check port availability
sudo lsof -i :8443
```

**Service keeps restarting**:
```bash
# View crash logs
journalctl -u diskimager-serve --since "1 hour ago" -p err

# Check for core dumps
coredumpctl list
coredumpctl info diskimager

# Check resource limits
systemctl show diskimager-serve --property=LimitNOFILE
```

**Graceful shutdown timeout**:
```bash
# Check current timeout
systemctl show diskimager-serve --property=TimeoutStopSec

# Increase timeout for long uploads
sudo systemctl edit diskimager-serve
# Add:
# [Service]
# TimeoutStopSec=60s
```

### Uninstallation

```bash
# Stop and remove services
sudo ./scripts/uninstall-daemon.sh

# Manual cleanup if needed
sudo systemctl stop diskimager-serve diskimager-web
sudo systemctl disable diskimager-serve diskimager-web
sudo rm /etc/systemd/system/diskimager-*.service
sudo systemctl daemon-reload
sudo rm /usr/local/bin/diskimager
```

## Security Best Practices

1. **Always run as dedicated user**: Never run as root
2. **Use strong TLS certificates**: 4096-bit RSA or ECDSA
3. **Restrict network access**: Use firewall rules
4. **Monitor logs**: Set up log aggregation and alerting
5. **Keep updated**: Regularly update binary and dependencies
6. **Backup configuration**: Include in regular backup routine
7. **Test recovery**: Regularly test service restart procedures

## Advanced Features

### Custom systemd unit

Create custom service for specific use case:
```bash
sudo cp /etc/systemd/system/diskimager-serve.service \
       /etc/systemd/system/diskimager-forensics.service

sudo systemctl edit --full diskimager-forensics.service
# Modify configuration paths and settings

sudo systemctl daemon-reload
sudo systemctl start diskimager-forensics
```

### Integration with monitoring

**Prometheus metrics** (via node_exporter):
```bash
# Install node_exporter
sudo apt install prometheus-node-exporter

# Metrics available at
curl http://localhost:9100/metrics
```

**Centralized logging** (via rsyslog):
```bash
# Forward to remote syslog server
# Edit /etc/rsyslog.d/diskimager.conf:
:programname, isequal, "diskimager-serve" @@syslog.example.com:514
```

## Support

- Full documentation: `DAEMON.md`
- GitHub issues: https://github.com/afterdarksys/diskimager/issues
- Configuration examples: `config/` directory
- Service files: `systemd/` directory
- Installation scripts: `scripts/` directory
