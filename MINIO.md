# MinIO and S3-Compatible Storage Support

The diskimager tool now supports imaging directly to MinIO and other S3-compatible storage services, including **darkstorage.io**.

## Supported Storage Backends

- **MinIO** (self-hosted or cloud)
- **darkstorage.io** (S3-compatible storage)
- **AWS S3** (with custom endpoints)
- Any S3-compatible service (Wasabi, DigitalOcean Spaces, etc.)

---

## URL Formats

### 1. MinIO Shorthand Format (Recommended)
```bash
minio://endpoint/bucket/path/to/file.img
```

**Examples:**
```bash
# darkstorage.io
minio://darkstorage.io/evidence-bucket/case001/disk.img

# Self-hosted MinIO
minio://minio.example.com:9000/forensics/disk001.img

# With HTTPS
minio://https://s3.darkstorage.io/my-bucket/evidence.e01
```

### 2. S3 Format with Custom Endpoint
```bash
s3://bucket/path/to/file.img?endpoint=https://endpoint-url
```

**Examples:**
```bash
# darkstorage.io using S3 format
s3://evidence-bucket/case001/disk.img?endpoint=https://darkstorage.io

# Self-hosted MinIO
s3://forensics/disk001.img?endpoint=https://minio.example.com:9000
```

### 3. Standard S3 (AWS)
```bash
s3://bucket/path/to/file.img
```

---

## Authentication

### Environment Variables

The tool looks for credentials in this order:

**MinIO-specific variables (checked first):**
```bash
export MINIO_ACCESS_KEY="your-access-key"
export MINIO_SECRET_KEY="your-secret-key"
```

**AWS-compatible variables (fallback):**
```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
```

### For darkstorage.io

```bash
# Set your darkstorage.io credentials
export MINIO_ACCESS_KEY="your-darkstorage-access-key"
export MINIO_SECRET_KEY="your-darkstorage-secret-key"

# Or use AWS variable names
export AWS_ACCESS_KEY_ID="your-darkstorage-access-key"
export AWS_SECRET_ACCESS_KEY="your-darkstorage-secret-key"
```

---

## Usage Examples

### Basic Imaging to darkstorage.io

```bash
# Set credentials
export MINIO_ACCESS_KEY="ds_abc123..."
export MINIO_SECRET_KEY="secret456..."

# Image directly to darkstorage.io
./diskimager image \
  --in /dev/sda \
  --out minio://darkstorage.io/evidence/case-2024-001/suspect-laptop.img \
  --hash sha256 \
  --case "CASE-2024-001" \
  --examiner "Detective Smith"
```

### E01 Format to MinIO

```bash
# Create compressed E01 image on MinIO
./diskimager image \
  --in /dev/sdb \
  --out minio://minio.lab.local:9000/forensics/evidence.e01 \
  --format e01 \
  --hash sha256 \
  --case "LAB-TEST-001" \
  --examiner "Forensic Team"
```

### Using S3 Format with darkstorage.io

```bash
./diskimager image \
  --in /dev/sdc \
  --out "s3://my-evidence/disk.img?endpoint=https://darkstorage.io" \
  --hash sha256
```

### Streaming from stdin to Cloud

```bash
# Pipe dd output directly to cloud storage
dd if=/dev/sda bs=64K | ./diskimager image \
  --in /dev/stdin \
  --out minio://darkstorage.io/cases/urgent/disk001.img \
  --hash sha256
```

---

## Features

### ✅ Supported
- Direct imaging to MinIO/S3-compatible storage
- All hash algorithms (MD5, SHA1, SHA256)
- Chain of custody metadata
- E01 format compression
- RAW format
- Progress reporting
- Bad sector handling
- Forensic audit logs (saved locally)

### ⚠️ Limitations
- **Resume not supported** for cloud storage targets
  - Reason: Cloud object storage doesn't support append operations
  - Workaround: Use local storage with resume, then upload completed image
- Network interruptions will require restart
- Large images may take significant time depending on bandwidth

---

## Configuration for darkstorage.io

### 1. Get Your Credentials

Log into your darkstorage.io account and navigate to:
- **Access Keys** section
- Generate a new access key pair
- Note: Keep your secret key secure!

### 2. Configure Environment

Create a `.env` file or export to shell:

```bash
# ~/.diskimager.env
export MINIO_ACCESS_KEY="ds_xxxxxxxxxxxxxxxx"
export MINIO_SECRET_KEY="yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"
```

Load before use:
```bash
source ~/.diskimager.env
```

### 3. Create Bucket

Ensure your bucket exists on darkstorage.io before imaging:
- Log into darkstorage.io web console
- Create bucket (e.g., `forensics-evidence`)
- Set appropriate permissions

### 4. Test Connection

```bash
# Quick test with small file
echo "test" > test.txt
./diskimager image \
  --in test.txt \
  --out minio://darkstorage.io/your-bucket/test/test.img \
  --hash sha256

# Check darkstorage.io console to verify file uploaded
```

---

## Best Practices

### For Forensic Use

1. **Local Audit Logs**
   - Audit logs are always saved locally (`.log` files)
   - Upload audit logs to cloud separately
   - Maintain local copies for chain of custody

2. **Hash Verification**
   - Hash is calculated before upload
   - Verify hash after upload using cloud provider tools
   - Document both hashes in chain of custody

3. **Bandwidth Considerations**
   - Large disk images (TB+) may take hours/days
   - Consider local imaging, then bulk upload
   - Use progress monitoring

4. **Metadata**
   - Always include case number, examiner, evidence number
   - E01 format embeds metadata in file
   - RAW format stores metadata in audit log

### Security

1. **Credentials**
   - Never hardcode credentials
   - Use environment variables
   - Rotate credentials regularly
   - Use least-privilege access

2. **Encryption**
   - E01 format provides compression (not encryption)
   - Consider client-side encryption for sensitive data
   - Use HTTPS endpoints (default)

3. **Network**
   - Use VPN for sensitive evidence
   - Avoid public networks
   - Monitor for interruptions

---

## Troubleshooting

### "credentials not found" Error

```bash
# Check environment variables are set
echo $MINIO_ACCESS_KEY
echo $MINIO_SECRET_KEY

# Re-export if empty
export MINIO_ACCESS_KEY="your-key"
export MINIO_SECRET_KEY="your-secret"
```

### "failed to open bucket" Error

**Possible causes:**
1. Incorrect endpoint URL
2. Bucket doesn't exist
3. Wrong credentials
4. Network connectivity issues

**Debug:**
```bash
# Test with curl
curl -v https://darkstorage.io

# Verify bucket exists via web console
# Check credentials are correct
```

### Slow Upload Speeds

**Solutions:**
1. Check network bandwidth
2. Increase block size: `--bs 1048576` (1MB)
3. Consider compression: `--format e01`
4. Use wired connection instead of WiFi

### Network Interruption

**Recovery:**
- Resume is NOT supported for cloud storage
- Image to local storage first with `--resume` capability
- Then upload completed image separately

**Workaround:**
```bash
# Step 1: Image locally with resume support
./diskimager image \
  --in /dev/sda \
  --out /local/storage/disk.img \
  --resume \
  --hash sha256

# Step 2: Upload completed image (separate tool)
aws s3 cp /local/storage/disk.img s3://bucket/path/ \
  --endpoint-url https://darkstorage.io
```

---

## Advanced Configuration

### Custom Region
```bash
# MinIO doesn't use regions, but SDK requires it
# Default: us-east-1 (works for all MinIO deployments)
```

### Path-Style Addressing
Automatically enabled for MinIO (required for compatibility).

### Timeouts
No timeout on upload (long-running operations supported).

---

## Performance Tips

### 1. Block Size Optimization
```bash
# Larger blocks = fewer API calls, faster upload
--bs 1048576  # 1MB (good for fast networks)
--bs 2097152  # 2MB (optimal for gigabit+)
```

### 2. Compression (E01)
```bash
# E01 format reduces upload size by 30-60%
--format e01
```

### 3. Local Buffer
```bash
# For very large images, buffer locally first
dd if=/dev/sda of=/tmp/buffer.img bs=1M status=progress
./diskimager image --in /tmp/buffer.img --out minio://... --format e01
```

---

## Integration Examples

### With existing forensic workflows

```bash
#!/bin/bash
# forensic-acquire.sh - Automated forensic acquisition to darkstorage.io

CASE_NUM="$1"
DEVICE="$2"
EXAMINER="$3"

if [ -z "$CASE_NUM" ] || [ -z "$DEVICE" ] || [ -z "$EXAMINER" ]; then
  echo "Usage: $0 <case-number> <device> <examiner>"
  exit 1
fi

# Load credentials
source ~/.diskimager.env

# Verify device is write-protected
if mountpoint -q "$DEVICE"; then
  echo "ERROR: Device is mounted! Unmount first."
  exit 1
fi

# Acquire to cloud
echo "Starting acquisition of $DEVICE for case $CASE_NUM"
./diskimager image \
  --in "$DEVICE" \
  --out "minio://darkstorage.io/cases/${CASE_NUM}/evidence-$(date +%Y%m%d).img" \
  --format e01 \
  --hash sha256 \
  --case "$CASE_NUM" \
  --examiner "$EXAMINER" \
  --evidence "PRIMARY_DRIVE"

echo "Acquisition complete. Check darkstorage.io for uploaded image."
```

---

## Comparison: Local vs Cloud Storage

| Feature | Local Storage | Cloud Storage (MinIO/darkstorage.io) |
|---------|---------------|--------------------------------------|
| Resume | ✅ Yes | ❌ No |
| Speed | ⚡ Fast | 🌐 Network-dependent |
| Capacity | 💾 Local disk limit | ☁️ Unlimited (pay-as-you-go) |
| Accessibility | 🏠 Single location | 🌍 Anywhere |
| Cost | 💰 Hardware upfront | 💵 Per GB/month |
| Redundancy | ⚠️ Manual backups | ✅ Automatic |
| Chain of Custody | 📁 Physical control | 🔐 Audit trail |

**Recommendation:**
- **Small to medium cases (< 500GB):** Direct to cloud
- **Large cases (> 500GB):** Local first, then upload
- **Mission critical:** Local + cloud redundancy

---

## Support

For darkstorage.io specific issues:
- Visit: https://darkstorage.io/support
- Documentation: https://docs.darkstorage.io

For diskimager issues:
- GitHub: https://github.com/afterdarksys/ads-diskimager/issues
- Documentation: See `ENHANCEME.md`, `BUGS.md`

---

## License & Compliance

The diskimager tool maintains forensic integrity for cloud storage:
- ✅ Cryptographic hash verification
- ✅ Chain of custody metadata
- ✅ Audit logging
- ✅ Bad sector handling
- ✅ Non-destructive reads

**Court admissibility:** Consult with your jurisdiction's requirements for digital evidence stored in cloud platforms.

---

*Last Updated: 2026-03-16*
*Version: 1.0.0 with MinIO support*
