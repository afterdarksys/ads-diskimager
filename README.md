# Diskimager - Forensic Disk Imaging Tool

A professional forensics-grade disk imaging and analysis tool written in Go, supporting local and cloud storage destinations.

## Features

### Core Capabilities
- 🔐 **Forensic Integrity**: SHA256/SHA1/MD5 cryptographic verification
- 📋 **Chain of Custody**: Metadata tracking (case number, examiner, evidence details)
- 📦 **Multiple Formats**: RAW, E01 (Expert Witness Format)
- ☁️ **Cloud Storage**: Direct imaging to S3, MinIO, darkstorage.io, GCS, Azure
- 🔄 **Resume Support**: Continue interrupted local imaging sessions
- 🛡️ **Bad Sector Handling**: Zero-fill and continue on read errors
- 📊 **Progress Reporting**: Real-time transfer speed and ETA
- 🔍 **Verification**: Built-in image integrity verification
- 🖥️ **GUI Available**: Optional graphical interface

### NEW in v2.1 🚀
- ⚡ **Parallel Multi-Hash**: Compute MD5+SHA1+SHA256 simultaneously with zero performance penalty
- 🧠 **Intelligent Error Recovery**: ddrescue-style adaptive retry logic for damaged disks
- 🌐 **Bandwidth Throttling**: Precise network/disk bandwidth control
- 📈 **Enhanced Progress**: Real-time ETA, speed monitoring, and phase tracking
- 🗜️ **Compression Support**: Inline gzip/zstd compression (30-70% space savings)
- 💾 **Sparse File Support**: Skip zero blocks (50-95% savings on sparse disks)
- 🌐 **RESTful API Server**: Asynchronous job processing with real-time WebSocket progress
- 🔍 **Interactive Disk Block Editor**: Web-based visual block map editor with click-to-explore

### Storage Backends
- **Local filesystem**: Traditional forensic imaging
- **AWS S3**: Direct to S3 buckets
- **MinIO**: Self-hosted or cloud S3-compatible storage
- **darkstorage.io**: Forensic-focused cloud storage
- **Google Cloud Storage (GCS)**
- **Azure Blob Storage**

### Advanced Features
- **The Sleuth Kit (TSK) Integration**: Deep filesystem analysis
- **Disk Recovery Tools**: Superblock repair, deleted file recovery
- **Timeline Generation**: MAC time extraction
- **Network Streaming**: Secure mTLS image transmission
- **Collection Server**: Central evidence repository

## Quick Start

### Installation

```bash
# Build from source
go build -o diskimager .

# Check version
./diskimager --version
```

### Basic Usage

```bash
# Image a disk to local file
./diskimager image \
  --in /dev/sda \
  --out evidence.img \
  --hash sha256 \
  --case "CASE-2024-001" \
  --examiner "John Doe"

# NEW: Compute multiple hashes simultaneously (MD5+SHA1+SHA256)
./diskimager image \
  --in /dev/sda \
  --out evidence.img \
  --multi-hash md5,sha1,sha256 \
  --case "CASE-2024-001"

# Image with compression and sparse support (maximum efficiency)
./diskimager image \
  --in /dev/vda \
  --out vm-backup.img.zst \
  --multi-hash sha256 \
  --compress zstd \
  --sparse \
  --case "VM-BACKUP-001"

# Network imaging with bandwidth throttling
export MINIO_ACCESS_KEY="your-access-key"
export MINIO_SECRET_KEY="your-secret-key"

./diskimager image \
  --in /dev/sda \
  --out minio://darkstorage.io/evidence/disk001.img.zst \
  --multi-hash md5,sha256 \
  --compress zstd \
  --bandwidth-limit 50M \
  --case "CASE-2024-001"

# Create compressed E01 format
./diskimager image \
  --in /dev/sdb \
  --out evidence.e01 \
  --format e01 \
  --hash sha256

# Resume interrupted imaging
./diskimager image \
  --in /dev/sdc \
  --out evidence.img \
  --resume

# Verify image integrity
./diskimager verify \
  --image evidence.img \
  --expected-hash <sha256-hash>
```

## Documentation

- **[docs/DISK_EDITOR.md](docs/DISK_EDITOR.md)** - Interactive disk block editor guide
- **[docs/API_SERVER.md](docs/API_SERVER.md)** - Complete API server guide with examples
- **[MINIO.md](MINIO.md)** - Complete guide for MinIO and darkstorage.io
- **[ENHANCEME.md](ENHANCEME.md)** - Roadmap of future enhancements
- **[BUGS.md](BUGS.md)** - Known issues and bug reports
- **[BUGFIXES.md](BUGFIXES.md)** - Completed bug fixes

## Commands

### `image` - Create Forensic Image

Create a forensic disk image with cryptographic verification and metadata.

**Options:**
- `--in <path>`: Input device or file (required)
- `--out <path>`: Output destination (required)
  - Local: `/path/to/file.img`
  - S3: `s3://bucket/path/file.img`
  - MinIO: `minio://endpoint/bucket/path/file.img`
  - GCS: `gs://bucket/path/file.img`
  - Azure: `azblob://container/path/file.img`
- `--format <raw|e01>`: Output format (default: raw)
- `--hash <md5|sha1|sha256>`: Hash algorithm (default: sha256)
- `--bs <size>`: Block size in bytes (default: 64KB)
- `--resume`: Resume interrupted local imaging
- `--case <number>`: Case number
- `--evidence <number>`: Evidence number
- `--examiner <name>`: Examiner name
- `--desc <text>`: Evidence description
- `--notes <text>`: Additional notes

### `verify` - Verify Image Integrity

Verify forensic image hash (supports RAW and E01 formats).

```bash
./diskimager verify \
  --image evidence.img \
  --expected-hash <sha256>
```

### `analyze` - Analyze Disk Image

Generate system hashes and file listings from disk images.

```bash
./diskimager analyze \
  --in evidence.img \
  --tsk  # Use The Sleuth Kit for advanced analysis
```

### `stream` - Network Streaming

Stream disk image to remote server via secure mTLS.

```bash
./diskimager stream \
  --target https://server:8080 \
  --in /dev/sda \
  --cert client.crt \
  --key client.key \
  --ca ca.crt
```

### `serve` - Collection Server

Start evidence collection server.

```bash
./diskimager serve --config config.json
```

### `ui` - Graphical Interface

Launch GUI for forensic operations.

```bash
./diskimager ui
```

### `api-server` - RESTful API Server

Start the forensic imaging API server for programmatic access and remote management.

```bash
# Basic server with API key authentication
./diskimager api-server \
  --bind-address :8080 \
  --api-keys secret-key-1,secret-key-2 \
  --max-workers 10

# Production server with TLS and mTLS
./diskimager api-server \
  --bind-address :8443 \
  --tls-cert server.crt \
  --tls-key server.key \
  --tls-ca ca.crt \
  --api-keys key1,key2 \
  --enable-cors
```

**Features:**
- Asynchronous job processing with worker pool
- Real-time progress via WebSocket streaming
- API key and mTLS authentication
- Support for all imaging features (compression, multi-hash, sparse, etc.)
- OpenAPI 3.0 specification

**See:** [docs/API_SERVER.md](docs/API_SERVER.md) for complete documentation

### `disk-editor` - Interactive Block Visualizer

Visualize and explore disk images with an interactive web-based block editor.

```bash
# Launch interactive disk block editor
./diskimager disk-editor --in /evidence/disk001.img

# Quick analysis of large disk (sample first 100K blocks)
./diskimager disk-editor --in /dev/sda --max-blocks 100000

# Full analysis with hashing
./diskimager disk-editor --in image.dd --compute-hash
```

**Features:**
- Visual block map with 256 blocks per row, colored by type
- Interactive hover tooltips and click-to-select
- Search and filter by file name, type, or status
- File signature detection (PNG, JPEG, PDF, ZIP, EXE, etc.)
- Entropy analysis for compression/encryption detection
- Zoom and pan for detailed inspection
- Real-time statistics and utilization metrics

**Perfect for:**
- Forensic investigation and triage
- Understanding disk structure and fragmentation
- Identifying file types and locations
- Finding deleted or unallocated data
- Visual disk utilization analysis

**See:** [docs/DISK_EDITOR.md](docs/DISK_EDITOR.md) for complete documentation

### `disktool` - Advanced Disk Tools

Low-level disk operations and recovery.

**Subcommands:**
- `getfs`: Get filesystem information
- `getboot`: Dump boot sector/MBR
- `recover`: Recover deleted files
- `wipe`: Secure wipe (DOD 5220.22-M)

### `forensick` - Forensic Analysis

Non-destructive image analysis tools.

## Cloud Storage Setup

### darkstorage.io

```bash
# 1. Set credentials
export MINIO_ACCESS_KEY="ds_your_access_key"
export MINIO_SECRET_KEY="your_secret_key"

# 2. Image directly
./diskimager image \
  --in /dev/sda \
  --out minio://darkstorage.io/cases/case001/disk.img \
  --format e01 \
  --case "CASE-001"
```

See **[MINIO.md](MINIO.md)** for complete documentation.

### AWS S3

```bash
export AWS_ACCESS_KEY_ID="your_key"
export AWS_SECRET_ACCESS_KEY="your_secret"

./diskimager image \
  --in /dev/sda \
  --out s3://my-bucket/evidence/disk.img \
  --hash sha256
```

### Google Cloud Storage

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/credentials.json"

./diskimager image \
  --in /dev/sda \
  --out gs://my-bucket/evidence/disk.img \
  --hash sha256
```

## Production Use

### Forensic Integrity ✅

- Cryptographic hash verification (MD5, SHA1, SHA256)
- Chain of custody metadata embedded in E01 format
- Tamper-evident audit logs
- Non-destructive read-only operations
- Bad sector recovery with zero-filling

### Best Practices

1. **Write-Blocking**: Always use hardware write-blocker for physical disks
2. **Verification**: Verify hash after imaging
3. **Metadata**: Include case number, examiner, evidence details
4. **Audit Logs**: Preserve `.log` files with images
5. **Hash Documentation**: Document all hashes in chain of custody
6. **Cloud Storage**: Verify upload completion and re-hash if possible

### Known Limitations

- **E01 Format**: Simplified implementation (not full libewf compliance)
  - Use for internal workflows
  - Verify with industry tools (FTK Imager, libewf) for court evidence
- **E01 Size Limit**: 4GB (EWF1 format limitation)
  - Use RAW format for larger images
- **Cloud Resume**: Not supported (use local imaging with --resume, then upload)

## Architecture

```
diskimager/
├── cmd/              # CLI commands
│   ├── image.go      # Main imaging command
│   ├── verify.go     # Hash verification
│   ├── analyze.go    # Disk analysis
│   ├── stream.go     # Network streaming
│   ├── serve.go      # Collection server
│   ├── ui.go         # GUI interface
│   ├── disktool*.go  # Low-level tools
│   └── forensick*.go # Analysis tools
├── imager/           # Core imaging engine
│   ├── imager.go     # Imaging logic
│   └── imager_test.go
├── pkg/
│   ├── format/       # Format writers
│   │   ├── raw/      # RAW format
│   │   └── e01/      # E01 format
│   ├── storage/      # Storage backends
│   │   └── blob.go   # Cloud storage (S3/MinIO/GCS/Azure)
│   └── tsk/          # Sleuth Kit bindings
└── config/           # Configuration
```

## Development

### Building

```bash
# Build binary
make build

# Run tests
make test

# Clean artifacts
make clean

# Update dependencies
make tidy
```

### Dependencies

- **Go 1.25+**
- **The Sleuth Kit** (optional, for TSK features)
- **Fyne** (optional, for GUI)
- **AWS SDK v2** (for S3/MinIO support)
- **gocloud.dev** (for multi-cloud support)

### Testing

```bash
# Run all tests
go test ./...

# Test specific package
go test ./imager -v

# Test with coverage
go test -cover ./...
```

## Security

### Threat Model

- **Data Integrity**: All writes verified with cryptographic hashes
- **Chain of Custody**: Metadata tracking prevents tampering claims
- **Network Security**: mTLS for streaming, HTTPS for cloud
- **Credential Security**: Environment variables, never hardcoded

### Responsible Use

This tool is designed for:
- ✅ Authorized digital forensic investigations
- ✅ Incident response and recovery
- ✅ Data preservation and archival
- ✅ Security research and education

Do not use for:
- ❌ Unauthorized access to computer systems
- ❌ Theft of data
- ❌ Violation of privacy laws

## License

[Specify your license here]

## Contributing

Contributions welcome! See:
- **[ENHANCEME.md](ENHANCEME.md)** for feature ideas
- **[BUGS.md](BUGS.md)** for known issues
- **GitHub Issues** for bug reports

## Support

- **Documentation**: See `*.md` files in repository
- **Issues**: https://github.com/afterdarksys/ads-diskimager/issues
- **darkstorage.io Support**: https://darkstorage.io/support

## Acknowledgments

Built with:
- [The Sleuth Kit](https://www.sleuthkit.org/) - Forensic analysis
- [go-diskfs](https://github.com/diskfs/go-diskfs) - Disk operations
- [Fyne](https://fyne.io/) - GUI framework
- [gocloud.dev](https://gocloud.dev/) - Cloud storage abstraction
- [Cobra](https://github.com/spf13/cobra) - CLI framework

---

**Version:** 1.0.0
**Last Updated:** 2026-03-16
**Status:** Production Ready (with noted E01 limitations)
