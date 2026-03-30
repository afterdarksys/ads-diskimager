# Diskimager GUI Documentation

## Overview

Diskimager provides two professional GUI interfaces for forensic disk imaging operations:

1. **Native Fyne GUI** - Norton Ghost-style desktop application with wizard workflows
2. **Web UI** - Modern React-based web interface with REST API and real-time WebSocket updates

Both interfaces share the same backend imaging engine and provide professional forensic workflows.

---

## 1. Fyne Desktop GUI

### Features

- **Norton Ghost-Style Interface**: Professional wizard-based workflows
- **Imaging Wizard**: 6-step guided process for disk imaging
- **Restore Wizard**: 5-step guided process for image restoration
- **Dashboard**: Quick actions and system health monitoring
- **Dark/Light Theme Support**: Professional appearance
- **Real-Time Progress**: Speed graphs, ETA calculations, phase tracking

### Installation

The Fyne GUI is built into the main binary. No additional dependencies needed.

### Usage

```bash
# Launch the GUI
./diskimager ui

# Or build and run
go run . ui
```

### Imaging Wizard (6 Steps)

#### Step 1: Source Selection
- Visual disk picker with thumbnails
- Shows disk info: size, model, serial, health status
- File browser for selecting image files
- SMART health indicators

#### Step 2: Destination
- **Local Storage Tab**: Save to filesystem
- **Cloud Storage Tab**: S3, Azure Blob, Google Cloud Storage
- Cloud credential input
- Path validation

#### Step 3: Options
- **Format Selection**: Raw, E01, VMDK, VHD
- **Compression**: 0-9 levels with slider
- **Encryption**: AES-256 option
- **Chain of Custody**:
  - Case Number (required)
  - Examiner Name (required)
  - Evidence ID
  - Description

#### Step 4: Verification
- Pre-flight checks:
  - Source accessibility
  - Destination writable
  - Sufficient disk space
  - SMART status
- Safety validations
- Auto-run on entry

#### Step 5: Execution
- Real-time progress bar
- Speed indicator (MB/s)
- ETA calculation
- Phase tracking (Reading, Hashing, Writing, etc.)
- Operation log with timestamps
- Cancel button for emergency stop
- Bad sector counter

#### Step 6: Summary
- Operation results
- Hash values (SHA256)
- Time elapsed
- Generate forensic report button
- Case information summary

### Restore Wizard (5 Steps)

#### Step 1: Image Selection
- Browse for forensic image
- Format detection (E01, VMDK, VHD, Raw)
- Image information preview:
  - Size
  - Creation date
  - Format type
- Hash verification option

#### Step 2: Target Disk
- **WARNING ZONE**: Destructive operation warnings
- Visual disk selector
- System disk protection (cannot select /dev/disk0)
- Safety checks
- Disk size validation

#### Step 3: Partition Mapping
- Source partition layout preview
- Target disk layout preview
- Size compatibility check
- Sector-by-sector restore notice

#### Step 4: Verification
- Safety checks:
  - Image accessible
  - Target writable
  - Size compatibility
  - Not system disk
- Verify data integrity option
- **Confirmation Required**: "I understand this will ERASE ALL DATA"
- Auto-run checks on entry

#### Step 5: Execution
- **DO NOT POWER OFF** warning
- Real-time progress
- Speed tracking
- ETA display
- Operation log
- Emergency stop button
- Post-restore verification (optional)

### Dashboard

#### Quick Actions
- **Create Image**: Launch imaging wizard
- **Restore Image**: Launch restore wizard
- **Clone Disk**: Direct disk-to-disk (coming soon)
- **Find Files**: CLI feature reference
- **Analyze Image**: CLI feature reference

#### Recent Operations
- Last 10 operations
- Status indicators (✓ Completed, ✗ Failed)
- Timestamps
- Clear history button

#### System Health Panel
- SMART status for all disks
- Health indicators (Healthy, Warning, Failed)
- Temperature monitoring (when available)
- Refresh button

#### Settings
- Theme selection (Light/Dark/System)
- Default compression level
- Hash algorithm preferences
- Cloud credential management

### Keyboard Shortcuts

- `Ctrl+I`: Start imaging wizard
- `Ctrl+R`: Start restore wizard
- `Ctrl+Q`: Quit application
- `Esc`: Cancel wizard (with confirmation)

---

## 2. Web UI

### Features

- **Modern React Interface**: Professional TailwindCSS design
- **REST API**: Full programmatic access
- **WebSocket Support**: Real-time progress updates
- **Multi-User Ready**: Designed for remote management
- **Responsive Design**: Works on tablets
- **Live Monitoring**: Real-time speed graphs

### Installation

#### Backend Server

The web server is built into the main binary:

```bash
# Start the web server (default port 8080)
./diskimager web

# Custom port
./diskimager web --port 9000
```

#### Frontend Development

```bash
cd web/frontend

# Install dependencies
npm install

# Development mode with hot reload
npm run dev

# Production build
npm run build
```

### Architecture

```
┌─────────────────┐
│  React Frontend │
│   (Port 3000)   │
└────────┬────────┘
         │
         ├── REST API (/api/*)
         │   └── Gin Router (Port 8080)
         │
         └── WebSocket (/api/ws/jobs/:id)
             └── Gorilla WebSocket
```

### API Endpoints

#### Disk Management

```bash
# List available disks
GET /api/disks

Response:
[
  {
    "path": "/dev/disk1",
    "name": "Data HDD",
    "size": 2199023255552,
    "model": "WD Blue 2TB",
    "serial": "WD-67890",
    "health": "Healthy",
    "type": "HDD"
  }
]
```

#### Job Management

```bash
# Create imaging job
POST /api/jobs/image
Content-Type: application/json

{
  "type": "image",
  "source_path": "/dev/disk1",
  "dest_path": "evidence-001.e01",
  "format": "E01",
  "compression": 6,
  "metadata": {
    "case_number": "CASE-2026-001",
    "examiner": "John Doe",
    "evidence": "Suspect laptop HDD",
    "description": "Primary storage device from suspect's workstation"
  }
}

Response:
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "created"
}
```

```bash
# Create restore job
POST /api/jobs/restore
Content-Type: application/json

{
  "type": "restore",
  "source_path": "evidence-001.e01",
  "dest_path": "/dev/disk2",
  "format": "E01"
}
```

```bash
# List all jobs
GET /api/jobs

Response:
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "image",
    "status": "running",
    "status_str": "running",
    "phase": "reading",
    "progress": 45.2,
    "speed": 125829120,
    "eta": "15m30s",
    "bytes_total": 2199023255552,
    "bytes_done": 994341421056,
    "bad_sectors": 0,
    "errors": [],
    "created_at": "2026-03-29T14:30:00Z",
    "source_path": "/dev/disk1",
    "dest_path": "evidence-001.e01",
    "format": "E01"
  }
]
```

```bash
# Get specific job
GET /api/jobs/:id

# Cancel/Delete job
DELETE /api/jobs/:id
```

#### WebSocket Real-Time Updates

```javascript
// Connect to job progress stream
const ws = new WebSocket('ws://localhost:8080/api/ws/jobs/:id');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);

  if (data.type === 'progress') {
    console.log('Progress:', data.percentage + '%');
    console.log('Speed:', data.speed / (1024 * 1024), 'MB/s');
    console.log('Phase:', data.phase);
    console.log('ETA:', data.eta, 'seconds');
  }
};

// Send control commands
ws.send(JSON.stringify({ type: 'pause' }));
ws.send(JSON.stringify({ type: 'cancel' }));
```

### Web UI Components

#### Dashboard
- **Stats Cards**: Active jobs, available disks, completed jobs
- **Disk Selector**: Visual list of available disks with health status
- **Job Queue**: Real-time job list with progress bars
- **Progress Monitor**: Detailed view of selected job

#### Disk Selector
- Disk icons (HDD/SSD/USB)
- Health status colors
- Size and model information
- Click to select for operations
- Auto-refresh capability

#### Job Queue
- Job cards with status badges
- Real-time progress bars
- Speed indicators
- Delete/Cancel buttons
- Phase tracking
- Error display

#### Progress Monitor
- WebSocket connection indicator
- Real-time metrics:
  - Status, Phase, Progress, Speed
  - Bytes processed
  - Bad sector count
  - ETA
- **Speed Graph**: Live chart showing transfer speed over time
- Error list with details

### Development

#### Frontend Tech Stack
- **React 18**: UI framework
- **TypeScript**: Type safety
- **TailwindCSS**: Utility-first styling
- **Zustand**: Lightweight state management
- **Recharts**: Speed visualization
- **Axios**: HTTP client
- **Lucide React**: Icon library
- **Vite**: Build tool

#### Running in Development

Terminal 1 - Backend:
```bash
./diskimager web --port 8080
```

Terminal 2 - Frontend:
```bash
cd web/frontend
npm run dev
```

Access at: `http://localhost:3000`

#### Production Build

```bash
cd web/frontend
npm run build

# Serve static files (implementation needed)
# Would integrate with Go's http.FileServer
```

---

## Comparison: Fyne vs Web UI

| Feature | Fyne Desktop | Web UI |
|---------|--------------|--------|
| **Installation** | Single binary | Binary + Node.js (dev) |
| **Platform** | Windows, macOS, Linux | Browser-based |
| **Wizards** | Full 6/5-step wizards | Dashboard-based |
| **Real-Time** | Direct updates | WebSocket |
| **Multi-User** | Single user | Ready for multi-user |
| **Offline** | Yes | Requires server |
| **Remote Access** | No | Yes |
| **Performance** | Native | Near-native |
| **Deployment** | Desktop app | Web server |

---

## Use Cases

### Fyne Desktop GUI
- **Forensic labs**: Dedicated imaging workstations
- **Field work**: Laptop-based evidence collection
- **Offline operations**: No network required
- **Quick operations**: Fast startup, native performance

### Web UI
- **Remote management**: Manage imaging servers remotely
- **Lab monitoring**: Monitor multiple imaging operations
- **Integration**: API for custom workflows
- **Multi-user environments**: Team collaboration
- **Batch operations**: Queue multiple jobs

---

## Architecture Details

### Shared Components

Both UIs use the same underlying systems:

```
┌──────────────────────────────────────────────┐
│              User Interface Layer            │
│  ┌─────────────────┐  ┌──────────────────┐  │
│  │   Fyne GUI      │  │    Web UI        │  │
│  │  (Native)       │  │  (React+API)     │  │
│  └────────┬────────┘  └────────┬─────────┘  │
├───────────┴───────────────────┴──────────────┤
│           Backend Components                  │
│  ┌───────────────────────────────────────┐  │
│  │   pkg/progress/progress.go            │  │
│  │   - Unified Progress Tracker          │  │
│  │   - Real-time updates                 │  │
│  │   - Speed calculations                │  │
│  └───────────────────────────────────────┘  │
│  ┌───────────────────────────────────────┐  │
│  │   imager/imager.go                    │  │
│  │   - Core imaging engine               │  │
│  │   - Format writers (E01, Raw, VMDK)   │  │
│  │   - Hash verification                 │  │
│  └───────────────────────────────────────┘  │
│  ┌───────────────────────────────────────┐  │
│  │   pkg/storage/storage.go              │  │
│  │   - Local, S3, Azure, GCS             │  │
│  │   - Unified destination handling      │  │
│  └───────────────────────────────────────┘  │
└───────────────────────────────────────────────┘
```

### Progress Tracking

Both UIs use `/Users/ryan/development/experiments-no-claude/go/diskimager/pkg/progress/progress.go`:

- **Atomic operations**: Thread-safe counters
- **Phase tracking**: Initializing, Reading, Hashing, Writing, etc.
- **Speed calculation**: Real-time MB/s
- **ETA estimation**: Dynamic time remaining
- **Bad sector counting**: Error tracking
- **Status management**: Pending, Running, Completed, Failed

---

## Best Practices

### For Forensic Use

1. **Always fill in chain of custody**:
   - Case number (required)
   - Examiner name (required)
   - Evidence description
   - Notes

2. **Verify hashes**:
   - SHA256 automatically generated
   - MD5 optional
   - Compare with source

3. **Run pre-flight checks**:
   - Verify disk health (SMART)
   - Check available space
   - Ensure write permissions

4. **Monitor operations**:
   - Watch for bad sectors
   - Check error logs
   - Verify completion status

5. **Generate reports**:
   - Use report generation feature
   - Include in case files
   - Document metadata

### For Developers

1. **API Integration**:
   - Use REST API for automation
   - WebSocket for monitoring
   - Implement error handling

2. **Custom Workflows**:
   - Script with API endpoints
   - Monitor via WebSocket
   - Handle job lifecycle

3. **Testing**:
   - Test with various disk sizes
   - Verify error handling
   - Check cancellation behavior

---

## Troubleshooting

### Fyne GUI

**Issue**: GUI doesn't start
```bash
# Check Fyne dependencies
go list -m fyne.io/fyne/v2

# Rebuild
go build -o diskimager
```

**Issue**: Slow rendering
- Lower compression level
- Reduce update frequency
- Check system resources

### Web UI

**Issue**: Cannot connect to API
```bash
# Verify server is running
curl http://localhost:8080/health

# Check firewall settings
# Verify port is not in use
```

**Issue**: WebSocket connection fails
- Check CORS settings
- Verify WebSocket upgrade
- Check network proxy settings

**Issue**: Frontend build fails
```bash
cd web/frontend
rm -rf node_modules package-lock.json
npm install
npm run build
```

---

## Future Enhancements

### Planned Features

1. **Authentication**: User login and permissions
2. **Audit Trail**: Complete operation logging
3. **Scheduling**: Queue jobs for specific times
4. **Notifications**: Email/Slack alerts on completion
5. **Templates**: Save imaging configurations
6. **Batch Operations**: Multiple disks simultaneously
7. **Clone Wizard**: Direct disk-to-disk cloning
8. **Analysis Integration**: Link to forensic analysis tools

### Coming Soon

- Docker container for web UI
- Mobile app (read-only monitoring)
- Integration with FTK/EnCase
- Advanced reporting templates
- Cloud backup scheduling

---

## Support

For issues or questions:
- Check `/Users/ryan/development/experiments-no-claude/go/diskimager/README.md`
- Review API documentation in code comments
- Examine example usage in test files

---

## License

© 2026 AfterDark Systems
