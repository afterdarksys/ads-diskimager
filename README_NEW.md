# Diskimager - Forensics-Grade Disk Imaging Suite

A complete professional forensics-grade disk imaging and analysis tool written in Go, supporting acquisition, restoration, conversion, and comprehensive chain of custody documentation.

## ✨ New Features (2026 Update)

### 🔄 **Restore Capability**
- Bit-for-bit restoration to physical disks
- Interactive safety confirmations
- Mount detection prevents accidental overwrites
- Size verification ensures destination compatibility
- System disk protection
- Post-restore hash verification

### 🔐 **Enhanced Forensic Integrity**
- **SMART Data Collection**: Capture disk health metrics
- **Write-Blocker Validation**: Verify source protection
- **Disk Geometry Preservation**: CHS addressing preserved
- **Adler32 Checksums**: Enhanced E01 format compliance
- **Secure Audit Logs**: 0600 permissions for sensitive metadata

### 💿 **Virtual Disk Support**
- Convert forensic images to VMDK (VMware)
- Convert forensic images to VHD (Microsoft)
- Direct analysis in virtual machines
- Preserves disk geometry for compatibility

## Core Capabilities

- 🔐 **Forensic Integrity**: SHA256/SHA1/MD5 cryptographic verification
- 📋 **Chain of Custody**: Complete metadata tracking with SMART data
- 📦 **Multiple Formats**: RAW, E01 (with Adler32), VMDK, VHD
- ☁️ **Cloud Storage**: Direct imaging to S3, MinIO, GCS, Azure
- 🔄 **Resume Support**: Continue interrupted local imaging sessions
- 🛡️ **Bad Sector Handling**: Zero-fill and continue on read errors
- 📊 **Progress Reporting**: Real-time transfer speed and ETA
- 🔍 **Verification**: Built-in image integrity verification (RAW + E01)
- 🖥️ **GUI Available**: Optional graphical interface

## Storage Backends

- **Local filesystem**: Traditional forensic imaging
- **AWS S3**: Direct to S3 buckets
- **MinIO**: Self-hosted or cloud S3-compatible storage
- **darkstorage.io**: Forensic-focused cloud storage
- **Google Cloud Storage (GCS)**
- **Azure Blob Storage**

## Advanced Features

- **SMART Data Collection**: Disk health, temperature, power-on hours
- **Write-Blocker Validation**: Verify hardware write protection
- **Disk Geometry Preservation**: CHS addressing for legacy systems
- **The Sleuth Kit (TSK) Integration**: Deep filesystem analysis
- **Disk Recovery Tools**: Superblock repair, deleted file recovery
- **File Carving**: Extract JPEG, PDF, PNG from raw images
- **Timeline Generation**: MAC time extraction
- **Network Streaming**: Secure mTLS image transmission
- **Collection Server**: Central evidence repository
- **Secure Wipe**: DoD 5220.22-M multi-pass overwrite

## Quick Start

### Installation

```bash
# Build from source
go build -o diskimager .

# Or use make
make build
```

### Basic Forensic Acquisition

```bash
# Complete forensic acquisition with SMART data
sudo ./diskimager image \
  --in /dev/sda \
  --out evidence.img \
  --hash sha256 \
  --case "CASE-2026-001" \
  --examiner "Jane Smith" \
  --evidence "HDD-001" \
  --smart \
  --geometry \
  --verify-write-block

# Output includes:
# - evidence.img (forensic image)
# - evidence.img.log (secure audit log with SMART data)
```

### Restore to Disk

```bash
# Restore with verification
sudo ./diskimager restore \
  --image evidence.img \
  --out /dev/sdb \
  --verify

# Interactive confirmation required
# Automatic safety checks:
# ✓ Size verification
# ✓ Mount detection
# ✓ System disk protection
# ✓ Post-restore hash verification
```

### Convert to Virtual Disk

```bash
# Convert to VMware VMDK
./diskimager convert \
  --in evidence.img \
  --out working_copy.vmdk \
  --format vmdk

# Convert to Microsoft VHD
./diskimager convert \
  --in evidence.img \
  --out working_copy.vhd \
  --format vhd

# Mount in VM for analysis
```

## Command Reference

### `image` - Create Forensic Image

Create a forensic disk image with full metadata and verification.

**Basic Options:**
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

**Chain of Custody:**
- `--case <number>`: Case number
- `--evidence <number>`: Evidence number
- `--examiner <name>`: Examiner name
- `--desc <text>`: Evidence description
- `--notes <text>`: Additional notes

**Forensic Enhancements:**
- `--smart`: Collect SMART data (health, temperature, hours)
- `--geometry`: Collect disk geometry (CHS)
- `--verify-write-block`: Verify source is write-protected

**Examples:**

```bash
# Maximum forensic documentation
sudo ./diskimager image \
  --in /dev/sda \
  --out evidence.e01 \
  --format e01 \
  --hash sha256 \
  --case "CASE-2026-001" \
  --evidence "LAPTOP-HDD-001" \
  --examiner "Jane Smith" \
  --desc "Suspect's primary hard drive" \
  --notes "Collected with hardware write-blocker" \
  --smart \
  --geometry \
  --verify-write-block

# Cloud storage with metadata
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"

./diskimager image \
  --in /dev/sdb \
  --out s3://evidence-bucket/case001/disk.img \
  --hash sha256 \
  --case "CASE-2026-001" \
  --smart
```

### `restore` - Restore Forensic Image

Restore a forensic image to a physical disk or file.

**Options:**
- `--image <path>`: Input image file (required)
- `--out <path>`: Output device or file (required)
- `--verify`: Verify written data with hash (default: true)
- `--hash <algo>`: Hash algorithm for verification (default: sha256)
- `--force`: Skip safety checks (DANGEROUS)

**Safety Features:**
- Interactive confirmation required
- Mount detection
- Size verification
- System disk protection (/dev/sda, /dev/disk0, etc.)
- Post-restore hash verification

**Examples:**

```bash
# Restore RAW image
sudo ./diskimager restore \
  --image evidence.img \
  --out /dev/sdb \
  --verify

# Restore E01 image (auto-decompressed)
sudo ./diskimager restore \
  --image evidence.e01 \
  --out /dev/sdc \
  --verify

# Restore to file (for testing)
./diskimager restore \
  --image evidence.img \
  --out restored_copy.img \
  --verify
```

### `convert` - Convert to Virtual Disk

Convert forensic images to virtual machine disk formats.

**Options:**
- `--in <path>`: Input image file (required)
- `--out <path>`: Output virtual disk path (required)
- `--format <vmdk|vhd>`: Output format (default: vmdk)

**Examples:**

```bash
# Convert to VMware VMDK
./diskimager convert \
  --in evidence.img \
  --out analysis_vm.vmdk \
  --format vmdk

# Convert to Microsoft VHD
./diskimager convert \
  --in evidence.e01 \
  --out analysis_vm.vhd \
  --format vhd
```

### `verify` - Verify Image Integrity

Verify forensic image hash (supports RAW and E01 formats).

**Examples:**

```bash
# Verify RAW image
./diskimager verify \
  --image evidence.img \
  --expected-hash <sha256-hash>

# Verify E01 image (decompresses and hashes raw data)
./diskimager verify \
  --image evidence.e01 \
  --expected-hash <sha256-hash>
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
  --ca ca.crt \
  --case "CASE-2026-001"
```

### `serve` - Collection Server

Start evidence collection server.

```bash
./diskimager serve --config config.json
```

### `disktool` - Advanced Disk Tools

Low-level disk operations and recovery.

**Subcommands:**
- `getfs`: Get filesystem information
- `getboot`: Dump boot sector/MBR
- `recover`: Recover deleted files (file carving)
- `wipe`: Secure wipe (DoD 5220.22-M) **with confirmation**

**Examples:**

```bash
# File carving from disk
sudo ./diskimager disktool recover \
  -d /dev/sda \
  --out-dir recovered/

# Secure wipe (requires confirmation)
sudo ./diskimager disktool wipe \
  -d /dev/sdb \
  --passes 7
```

## Forensic Workflow

### Complete Acquisition Workflow

```bash
# 1. Verify write-blocker
sudo ./diskimager image \
  --in /dev/sda \
  --out /dev/null \
  --verify-write-block
# Should show: ✓ Device is write-protected

# 2. Full acquisition with metadata
sudo ./diskimager image \
  --in /dev/sda \
  --out evidence_case001.e01 \
  --format e01 \
  --hash sha256 \
  --case "CASE-2026-001" \
  --evidence "SUSPECT-LAPTOP-HDD" \
  --examiner "Jane Smith, CFE" \
  --desc "Dell Latitude E7450 primary drive" \
  --notes "Seized under warrant SW-2026-001234" \
  --smart \
  --geometry

# 3. Verify acquisition
./diskimager verify \
  --image evidence_case001.e01 \
  --expected-hash <hash-from-step-2>

# 4. Create working copy for analysis
./diskimager convert \
  --in evidence_case001.e01 \
  --out analysis_vm.vmdk \
  --format vmdk

# 5. Document chain of custody
cat evidence_case001.e01.log
# Contains:
# - Source device info
# - SMART data (health, serial, hours)
# - Disk geometry
# - Hash values
# - Timestamps
# - Bad sectors (if any)
```

### Restoration Workflow

```bash
# 1. Verify destination size
sudo ./diskimager restore \
  --image evidence.img \
  --out /dev/sdb
# Will check: ✓ Destination (500GB) >= Image (320GB)

# 2. Interactive confirmation
# Type exact device path: /dev/sdb
# Countdown: 3 seconds

# 3. Restore with verification
# Progress: Restoring evidence.img -> /dev/sdb
# ✓ Wrote 320000000000 bytes
# Restored data hash: [sha256]

# 4. Verify restoration
sudo dd if=/dev/sdb bs=1M | sha256sum
# Compare with original hash
```

## Cloud Storage Setup

### AWS S3

```bash
export AWS_ACCESS_KEY_ID="your_key"
export AWS_SECRET_ACCESS_KEY="your_secret"

./diskimager image \
  --in /dev/sda \
  --out s3://evidence-bucket/case001/disk.img \
  --hash sha256
```

### MinIO / darkstorage.io

```bash
export MINIO_ACCESS_KEY="ds_your_access_key"
export MINIO_SECRET_KEY="your_secret_key"

./diskimager image \
  --in /dev/sda \
  --out minio://darkstorage.io/cases/case001/disk.img \
  --format e01 \
  --case "CASE-001"
```

See **[MINIO.md](MINIO.md)** for complete documentation.

## Production Best Practices

### Forensic Integrity ✅

- Cryptographic hash verification (MD5, SHA1, SHA256)
- Chain of custody metadata embedded in E01 format
- SMART data capture for disk health documentation
- Disk geometry preservation
- Tamper-evident audit logs (secure 0600 permissions)
- Non-destructive read-only operations
- Bad sector recovery with zero-filling

### Recommended Procedures

1. **Write-Blocking**: Always use hardware write-blocker for physical disks
   ```bash
   --verify-write-block  # Validates protection
   ```

2. **Metadata Collection**: Document everything
   ```bash
   --smart --geometry --case --evidence --examiner
   ```

3. **Verification**: Always verify acquisition
   ```bash
   ./diskimager verify --image evidence.img --expected-hash <hash>
   ```

4. **Chain of Custody**: Preserve audit logs with images
   - `evidence.img.log` contains complete forensic metadata
   - Secure permissions (0600) protect sensitive information

5. **Hash Documentation**: Document all hashes in chain of custody forms

6. **Cloud Storage**: Verify upload completion and integrity

7. **Working Copies**: Never analyze original evidence
   ```bash
   # Create VM copy for analysis
   ./diskimager convert --in evidence.e01 --out working.vmdk
   ```

### Known Limitations

- **E01 Format**: Enhanced with Adler32 checksums, but still simplified
  - Use for internal workflows
  - Verify with industry tools (FTK Imager, libewf) for court evidence
  - Missing: Complex section headers, detailed volume information
- **E01 Size Limit**: 4GB (EWF1 format limitation)
  - Use RAW format for larger images
  - Or split functionality (future enhancement)
- **Cloud Resume**: Not supported (use local imaging with --resume, then upload)
- **Write-Blocker Detection**: Platform-specific, may require admin privileges

## Security Features

### Data Protection
- **Audit Logs**: 0600 permissions (owner read/write only)
- **Credentials**: Environment variables only, never hardcoded
- **mTLS**: Mutual TLS for network streaming
- **Wipe Confirmation**: Interactive "WIPE" confirmation for destructive operations

### Safety Checks
- **Mount Detection**: Prevents writing to mounted filesystems
- **System Disk Protection**: Blocks accidental /dev/sda overwrites
- **Size Verification**: Ensures destination is large enough
- **Interactive Confirmations**: Required for destructive operations

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
- **smartmontools** (optional, for SMART data collection)
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

## Architecture

```
diskimager/
├── cmd/              # CLI commands
│   ├── image.go      # Forensic imaging
│   ├── restore.go    # Disk restoration
│   ├── convert.go    # Virtual disk conversion
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
│   ├── format/       # Format writers/readers
│   │   ├── raw/      # RAW format
│   │   ├── e01/      # E01 format (with Adler32)
│   │   └── virtual/  # VMDK/VHD formats
│   ├── storage/      # Storage backends
│   │   └── blob.go   # Cloud storage (S3/MinIO/GCS/Azure)
│   ├── smart/        # SMART data collection
│   ├── geometry/     # Disk geometry
│   └── tsk/          # Sleuth Kit bindings
└── config/           # Configuration
```

## Documentation

- **[README.md](README.md)** - This file
- **[MINIO.md](MINIO.md)** - MinIO and darkstorage.io guide
- **[BUGS.md](BUGS.md)** - Known issues
- **[BUGFIXES.md](BUGFIXES.md)** - Completed bug fixes
- **[ENHANCEME.md](ENHANCEME.md)** - Future enhancements
- **[CHANGELOG.md](CHANGELOG.md)** - Version history

## Support

- **Documentation**: See `*.md` files in repository
- **Issues**: https://github.com/afterdarksys/ads-diskimager/issues
- **darkstorage.io Support**: https://darkstorage.io/support

## License

[Specify your license here]

## Acknowledgments

Built with:
- [The Sleuth Kit](https://www.sleuthkit.org/) - Forensic analysis
- [go-diskfs](https://github.com/diskfs/go-diskfs) - Disk operations
- [Fyne](https://fyne.io/) - GUI framework
- [gocloud.dev](https://gocloud.dev/) - Cloud storage abstraction
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [smartmontools](https://www.smartmontools.org/) - SMART data collection

## Responsible Use

This tool is designed for:
- ✅ Authorized digital forensic investigations
- ✅ Incident response and recovery
- ✅ Data preservation and archival
- ✅ Security research and education

Do not use for:
- ❌ Unauthorized access to computer systems
- ❌ Theft of data
- ❌ Violation of privacy laws

---

**Version:** 2.0.0
**Last Updated:** 2026-03-28
**Status:** Production Ready (Enhanced Forensics-Grade)

**What's New in 2.0:**
- ✅ Restore capability with safety checks
- ✅ SMART data collection
- ✅ Write-blocker validation
- ✅ Disk geometry preservation
- ✅ Virtual disk format support (VMDK/VHD)
- ✅ Enhanced E01 format (Adler32 checksums)
- ✅ Secure audit logs (0600 permissions)
- ✅ Interactive safety confirmations
