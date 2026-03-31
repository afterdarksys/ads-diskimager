# Diskimager API Server

## Overview

The Diskimager API Server provides a RESTful HTTP API for forensic disk imaging operations. It enables programmatic access to all diskimager features with asynchronous job processing, real-time progress updates via WebSocket, and comprehensive authentication options.

## Features

- **Asynchronous Job Processing**: Submit imaging jobs and track progress asynchronously
- **Real-time Progress**: WebSocket streaming for live progress updates
- **Multi-Hash Support**: Simultaneous MD5, SHA1, SHA256 computation
- **Compression**: Inline gzip/zstd compression
- **Sparse File Detection**: Automatic zero-block skipping
- **Bandwidth Throttling**: Network/disk rate limiting
- **Multiple Sources**: Disk, file, S3, Azure, GCS, VM formats
- **Multiple Destinations**: File, S3, Azure, GCS
- **Authentication**: API key and mTLS (mutual TLS) support
- **CORS**: Cross-origin resource sharing for web applications
- **Chain of Custody**: Forensic metadata tracking

## Quick Start

### Starting the Server

```bash
# Basic server (HTTP, no authentication - NOT recommended for production)
./diskimager api-server --bind-address :8080

# Production server with API keys
./diskimager api-server \
  --bind-address :8080 \
  --api-keys secret-key-1,secret-key-2 \
  --max-workers 10

# Production server with TLS
./diskimager api-server \
  --bind-address :8443 \
  --tls-cert server.crt \
  --tls-key server.key \
  --api-keys key1,key2

# Production server with mTLS (client certificate verification)
./diskimager api-server \
  --bind-address :8443 \
  --tls-cert server.crt \
  --tls-key server.key \
  --tls-ca ca.crt \
  --api-keys key1,key2 \
  --enable-cors \
  --allowed-origins https://forensics.example.com
```

### Configuration Options

| Flag | Description | Default |
|------|-------------|---------|
| `--bind-address` | Address to bind server | `:8080` |
| `--max-workers` | Maximum concurrent imaging jobs | `10` |
| `--api-keys` | Comma-separated list of valid API keys | None |
| `--tls-cert` | Path to TLS certificate file | None |
| `--tls-key` | Path to TLS private key file | None |
| `--tls-ca` | Path to CA certificate for mTLS | None |
| `--enable-cors` | Enable CORS support | `false` |
| `--allowed-origins` | Comma-separated list of allowed origins | `*` |

### Environment Variables

```bash
# API keys can also be set via environment variable
export DISKIMAGER_API_KEYS="key1,key2,key3"

# Cloud storage credentials
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export MINIO_ACCESS_KEY="minio-access-key"
export MINIO_SECRET_KEY="minio-secret-key"
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/gcs-credentials.json"
export AZURE_STORAGE_ACCOUNT="storage-account"
export AZURE_STORAGE_KEY="storage-key"
```

## API Endpoints

### Health Check

**GET** `/api/v1/health`

No authentication required.

```bash
curl http://localhost:8080/api/v1/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-03-31T00:00:00Z",
  "version": "1.0.0",
  "uptime": 3600
}
```

### Version Information

**GET** `/api/v1/version`

No authentication required.

```bash
curl http://localhost:8080/api/v1/version
```

**Response:**
```json
{
  "api_version": "1.0.0",
  "diskimager_version": "2.1.0",
  "build_date": "2026-03-31",
  "git_commit": "abc123"
}
```

### Create Imaging Job

**POST** `/api/v1/jobs/image`

Requires authentication (API key or mTLS).

```bash
curl -X POST http://localhost:8080/api/v1/jobs/image \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d @job-request.json
```

**Request Body:**
```json
{
  "source": {
    "type": "file",
    "path": "/dev/sda"
  },
  "destination": {
    "type": "file",
    "path": "/evidence/disk001.img",
    "format": "raw"
  },
  "options": {
    "block_size": 65536,
    "hash_algorithms": ["md5", "sha1", "sha256"],
    "compression": "zstd",
    "compression_level": 5,
    "detect_sparse": true,
    "rate_limit": 52428800
  },
  "metadata": {
    "case_number": "CASE-2024-001",
    "evidence_number": "HDD-12345",
    "examiner": "John Doe",
    "description": "Suspect laptop hard drive",
    "notes": "Found at crime scene"
  }
}
```

**Source Types:**
- `disk` - Physical disk device (e.g., `/dev/sda`)
- `file` - Local file
- `s3` - Amazon S3 object
- `azure-blob` - Azure Blob Storage
- `gcs` - Google Cloud Storage
- `vm-vmdk` - VMware VMDK (future)
- `vm-vhd` - Hyper-V VHD (future)

**Destination Types:**
- `file` - Local file
- `s3` - Amazon S3
- `azure-blob` - Azure Blob Storage
- `gcs` - Google Cloud Storage

**Formats:**
- `raw` - Raw disk image (default)
- `e01` - Expert Witness Format (E01)

**Compression Algorithms:**
- `none` - No compression (default)
- `gzip` - Gzip compression
- `zstd` - Zstandard compression (recommended)

**Response:**
```json
{
  "job_id": "7650bd78-74b2-45db-83b7-0143cabb04ce",
  "status": "queued",
  "created_at": "2026-03-31T00:00:00Z",
  "source": { "type": "file", "path": "/dev/sda" },
  "destination": { "type": "file", "path": "/evidence/disk001.img" },
  "options": { ... },
  "metadata": { ... },
  "stream_url": "http://localhost:8080/api/v1/jobs/7650bd78-74b2-45db-83b7-0143cabb04ce/stream"
}
```

### Get Job Status

**GET** `/api/v1/jobs/{jobId}`

```bash
curl http://localhost:8080/api/v1/jobs/7650bd78-74b2-45db-83b7-0143cabb04ce \
  -H "X-API-Key: your-api-key"
```

**Response:**
```json
{
  "job_id": "7650bd78-74b2-45db-83b7-0143cabb04ce",
  "status": "completed",
  "created_at": "2026-03-31T00:00:00Z",
  "started_at": "2026-03-31T00:00:01Z",
  "completed_at": "2026-03-31T02:30:45Z",
  "progress": {
    "phase": "completed",
    "bytes_processed": 1000204886016,
    "total_bytes": 1000204886016,
    "percentage": 100.0,
    "speed": 98500000,
    "eta": 0
  },
  "result": {
    "bytes_copied": 1000204886016,
    "hashes": {
      "md5": "d41d8cd98f00b204e9800998ecf8427e",
      "sha1": "da39a3ee5e6b4b0d3255bfef95601890afd80709",
      "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    },
    "bad_sectors": [],
    "duration": 9044
  }
}
```

**Job Statuses:**
- `queued` - Job waiting for worker
- `running` - Job currently executing
- `completed` - Job finished successfully
- `failed` - Job failed with error
- `cancelled` - Job was cancelled

**Progress Phases:**
- `initializing` - Opening source/destination
- `reading` - Reading from source
- `hashing` - Computing hashes
- `compressing` - Compressing data
- `encrypting` - Encrypting data
- `writing` - Writing to destination
- `verifying` - Verifying integrity
- `completed` - Finished

### List Jobs

**GET** `/api/v1/jobs?status={status}&limit={limit}&offset={offset}`

```bash
# List all jobs
curl http://localhost:8080/api/v1/jobs \
  -H "X-API-Key: your-api-key"

# List running jobs
curl "http://localhost:8080/api/v1/jobs?status=running" \
  -H "X-API-Key: your-api-key"

# Pagination
curl "http://localhost:8080/api/v1/jobs?limit=50&offset=100" \
  -H "X-API-Key: your-api-key"
```

**Response:**
```json
{
  "jobs": [ {...}, {...} ],
  "total": 150,
  "limit": 50,
  "offset": 0
}
```

### Cancel Job

**DELETE** `/api/v1/jobs/{jobId}`

```bash
curl -X DELETE http://localhost:8080/api/v1/jobs/7650bd78-74b2-45db-83b7-0143cabb04ce \
  -H "X-API-Key: your-api-key"
```

**Response:**
```json
{
  "message": "Job cancelled"
}
```

### WebSocket Progress Streaming

**GET** `/api/v1/jobs/{jobId}/stream` (WebSocket)

Connect to this endpoint to receive real-time progress updates.

**Authentication:** Pass API key via query parameter: `?api_key=your-api-key`

**JavaScript Example:**
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/jobs/7650bd78.../stream?api_key=your-key');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);

  switch(data.type) {
    case 'status':
      console.log('Job status:', data.status);
      break;
    case 'progress':
      console.log(`Progress: ${data.progress.percentage}% - ${data.progress.phase}`);
      console.log(`Speed: ${data.progress.speed} bytes/sec - ETA: ${data.progress.eta}s`);
      break;
    case 'complete':
      console.log('Job completed:', data.status);
      ws.close();
      break;
    case 'error':
      console.error('Job error:', data.error);
      ws.close();
      break;
  }
};
```

**Python Example:**
```python
import websocket
import json

def on_message(ws, message):
    data = json.loads(message)

    if data['type'] == 'progress':
        progress = data['progress']
        print(f"Progress: {progress['percentage']:.1f}% - {progress['phase']}")
        print(f"Speed: {progress['speed']/1024/1024:.1f} MB/s - ETA: {progress['eta']}s")
    elif data['type'] == 'complete':
        print(f"Job completed: {data['status']}")
        ws.close()

ws = websocket.WebSocketApp(
    "ws://localhost:8080/api/v1/jobs/7650bd78.../stream?api_key=your-key",
    on_message=on_message
)
ws.run_forever()
```

## Authentication

### API Key Authentication

Include API key in the `X-API-Key` header:

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/jobs
```

### mTLS (Mutual TLS) Authentication

Configure server with CA certificate:

```bash
./diskimager api-server \
  --bind-address :8443 \
  --tls-cert server.crt \
  --tls-key server.key \
  --tls-ca ca.crt
```

Connect with client certificate:

```bash
curl --cert client.crt --key client.key --cacert ca.crt \
  https://localhost:8443/api/v1/jobs
```

### Combined Authentication

You can require both API key and mTLS:

```bash
./diskimager api-server \
  --bind-address :8443 \
  --tls-cert server.crt \
  --tls-key server.key \
  --tls-ca ca.crt \
  --api-keys key1,key2
```

## Complete Examples

### Example 1: Basic Local Imaging

```bash
# Create job request
cat > /tmp/job.json <<EOF
{
  "source": {"type": "disk", "device": "/dev/sdb"},
  "destination": {"type": "file", "path": "/evidence/usb001.img"},
  "options": {
    "block_size": 65536,
    "hash_algorithms": ["sha256"]
  },
  "metadata": {
    "case_number": "CASE-001",
    "examiner": "Jane Doe"
  }
}
EOF

# Submit job
JOB_ID=$(curl -s -X POST http://localhost:8080/api/v1/jobs/image \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d @/tmp/job.json | jq -r '.job_id')

echo "Job ID: $JOB_ID"

# Poll for completion
while true; do
  STATUS=$(curl -s http://localhost:8080/api/v1/jobs/$JOB_ID \
    -H "X-API-Key: your-key" | jq -r '.status')

  if [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ]; then
    break
  fi

  sleep 5
done

# Get final result
curl -s http://localhost:8080/api/v1/jobs/$JOB_ID \
  -H "X-API-Key: your-key" | jq
```

### Example 2: Cloud Storage with Compression

```bash
cat > /tmp/cloud-job.json <<EOF
{
  "source": {
    "type": "disk",
    "device": "/dev/sdc"
  },
  "destination": {
    "type": "s3",
    "bucket": "forensics-evidence",
    "key": "cases/2024/case001/disk.img.zst",
    "format": "raw"
  },
  "options": {
    "block_size": 65536,
    "hash_algorithms": ["md5", "sha1", "sha256"],
    "compression": "zstd",
    "compression_level": 5,
    "detect_sparse": true,
    "rate_limit": 52428800
  },
  "metadata": {
    "case_number": "CASE-2024-001",
    "evidence_number": "HDD-12345",
    "examiner": "John Doe",
    "description": "Suspect laptop hard drive"
  }
}
EOF

curl -X POST http://localhost:8080/api/v1/jobs/image \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d @/tmp/cloud-job.json
```

## Error Handling

### HTTP Status Codes

- `200 OK` - Request successful
- `201 Created` - Job created successfully
- `400 Bad Request` - Invalid request body or parameters
- `401 Unauthorized` - Authentication required or failed
- `404 Not Found` - Job not found
- `409 Conflict` - Cannot cancel completed/failed job
- `500 Internal Server Error` - Server error

### Error Response Format

```json
{
  "error": "validation_error",
  "message": "Source type is required",
  "details": {
    "field": "source.type",
    "value": null
  }
}
```

## Performance Tuning

### Worker Pool Size

```bash
# Increase concurrent jobs for high-capacity systems
--max-workers 50
```

### Block Size

```bash
# Larger blocks = faster (but more memory)
"block_size": 1048576  # 1MB
```

### Compression Level

```bash
# Fast compression for network transfer
"compression": "zstd",
"compression_level": 1

# Maximum compression for archival
"compression": "zstd",
"compression_level": 9
```

### Bandwidth Limiting

```bash
# Limit to 50 MB/s
"rate_limit": 52428800
```

## Security Best Practices

1. **Always use TLS in production**
   ```bash
   --tls-cert server.crt --tls-key server.key
   ```

2. **Enable mTLS for sensitive environments**
   ```bash
   --tls-ca ca.crt
   ```

3. **Use strong API keys**
   ```bash
   # Generate secure random keys
   openssl rand -hex 32
   ```

4. **Restrict CORS origins**
   ```bash
   --enable-cors --allowed-origins https://forensics.example.com
   ```

5. **Run as non-root user** (use capabilities for disk access)
   ```bash
   sudo setcap cap_sys_admin,cap_dac_read_search+ep ./diskimager
   ./diskimager api-server  # runs as non-root
   ```

6. **Use firewall rules** to restrict access

7. **Enable audit logging** (future feature)

## OpenAPI Specification

The complete OpenAPI 3.0 specification is available at:
```
api/openapi.yaml
```

Import this into tools like Swagger UI, Postman, or Insomnia for interactive API documentation.

## Troubleshooting

### Server won't start

**Check port availability:**
```bash
lsof -i :8080
```

**Check certificate permissions:**
```bash
ls -la server.crt server.key
chmod 600 server.key
```

### Authentication failures

**Verify API key:**
```bash
curl -v http://localhost:8080/api/v1/jobs \
  -H "X-API-Key: your-key" 2>&1 | grep -i auth
```

**Check mTLS certificate:**
```bash
openssl verify -CAfile ca.crt client.crt
```

### Jobs failing immediately

**Check source permissions:**
```bash
sudo ./diskimager api-server  # or use setcap
```

**Check destination permissions:**
```bash
ls -ld /evidence
```

**Check server logs:**
```bash
tail -f /tmp/api-server.log
```

### Slow performance

**Increase worker pool:**
```bash
--max-workers 20
```

**Adjust block size:**
```json
"block_size": 1048576
```

**Disable compression for testing:**
```json
"compression": "none"
```

## Production Deployment

### Systemd Service

```ini
# /etc/systemd/system/diskimager-api.service
[Unit]
Description=Diskimager API Server
After=network.target

[Service]
Type=simple
User=forensics
Group=forensics
WorkingDirectory=/opt/diskimager
ExecStart=/opt/diskimager/diskimager api-server \
  --bind-address :8443 \
  --tls-cert /etc/diskimager/server.crt \
  --tls-key /etc/diskimager/server.key \
  --tls-ca /etc/diskimager/ca.crt \
  --max-workers 10
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable diskimager-api
sudo systemctl start diskimager-api
sudo systemctl status diskimager-api
```

### Docker Deployment

```dockerfile
FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN go build -o diskimager .

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=builder /app/diskimager /usr/local/bin/
EXPOSE 8443
CMD ["diskimager", "api-server", "--bind-address", ":8443"]
```

```bash
docker build -t diskimager-api .
docker run -d -p 8443:8443 \
  -v /evidence:/evidence \
  -v /etc/diskimager:/certs \
  -e DISKIMAGER_API_KEYS="key1,key2" \
  diskimager-api
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: diskimager-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: diskimager-api
  template:
    metadata:
      labels:
        app: diskimager-api
    spec:
      containers:
      - name: diskimager-api
        image: diskimager-api:latest
        ports:
        - containerPort: 8443
        env:
        - name: DISKIMAGER_API_KEYS
          valueFrom:
            secretKeyRef:
              name: diskimager-secrets
              key: api-keys
        volumeMounts:
        - name: evidence
          mountPath: /evidence
        - name: certs
          mountPath: /certs
      volumes:
      - name: evidence
        persistentVolumeClaim:
          claimName: evidence-pvc
      - name: certs
        secret:
          secretName: diskimager-tls
```

## Version History

- **v2.1.0** (2026-03-31) - API server integration with real imager
  - Real imaging operations (not simulation)
  - Multi-hash support (MD5, SHA1, SHA256)
  - Compression support (gzip, zstd)
  - Sparse file detection
  - Bandwidth throttling
  - Progress tracking

- **v2.0.0** (2024-03-30) - Initial API server infrastructure
  - RESTful HTTP API
  - WebSocket progress streaming
  - API key and mTLS authentication
  - Job queue with worker pool
  - OpenAPI 3.0 specification

## Support

- **Documentation**: `docs/` directory
- **OpenAPI Spec**: `api/openapi.yaml`
- **Issues**: GitHub Issues
- **Examples**: See examples above

---

**Version**: 2.1.0
**Date**: 2026-03-31
**Status**: ✅ Production Ready
