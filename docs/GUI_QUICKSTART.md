# GUI Quick Start Guide

## Getting Started in 5 Minutes

### Option 1: Fyne Desktop GUI (Recommended for Forensic Labs)

```bash
# Build
go build -o diskimager

# Launch
./diskimager ui
```

**What you'll see:**
1. Professional dashboard with quick action cards
2. Click "Create Image" to start the 6-step imaging wizard
3. Click "Restore Image" to start the 5-step restore wizard

**First Imaging Job:**
1. Click "Create Image" button
2. Select source disk from the list
3. Choose destination (local file or cloud)
4. Enter case information (required):
   - Case Number: e.g., "CASE-2026-001"
   - Examiner: Your name
5. Run pre-flight checks (auto-runs)
6. Watch real-time progress with speed graph
7. Generate forensic report when complete

### Option 2: Web UI (For Remote Management)

**Terminal 1 - Start Backend:**
```bash
go build -o diskimager
./diskimager web --port 8080
```

**Terminal 2 - Start Frontend (Development):**
```bash
cd web/frontend
npm install
npm run dev
```

**Access:** `http://localhost:3000`

**What you'll see:**
1. Modern web dashboard
2. Available disks on the left
3. Job queue in the center
4. Real-time progress monitor at bottom

**Create Your First Job via API:**
```bash
curl -X POST http://localhost:8080/api/jobs/image \
  -H "Content-Type: application/json" \
  -d '{
    "source_path": "/dev/disk1",
    "dest_path": "evidence.e01",
    "format": "E01",
    "compression": 6,
    "metadata": {
      "case_number": "CASE-2026-001",
      "examiner": "John Doe",
      "evidence": "Suspect laptop",
      "description": "Primary storage device"
    }
  }'
```

---

## Key Features at a Glance

### Fyne Desktop Features
- ✅ 6-step imaging wizard
- ✅ 5-step restore wizard
- ✅ Real-time progress with speed graphs
- ✅ Norton Ghost-style interface
- ✅ Chain of custody forms
- ✅ SMART health monitoring
- ✅ Dark/Light themes
- ✅ Report generation

### Web UI Features
- ✅ REST API for automation
- ✅ WebSocket real-time updates
- ✅ Modern React interface
- ✅ Multi-job queue management
- ✅ Speed visualization charts
- ✅ Remote monitoring
- ✅ Browser-based access

---

## Common Workflows

### Workflow 1: Image a Disk (Fyne GUI)

1. Launch GUI: `./diskimager ui`
2. Click **"Create Image"**
3. **Step 1**: Select source disk
4. **Step 2**: Choose destination
   - Local: Browse for output file
   - Cloud: Enter S3/Azure/GCS credentials
5. **Step 3**: Configure options
   - Format: E01 (recommended for forensics)
   - Compression: Level 6
   - Enter case details (required)
6. **Step 4**: Review checks (auto-runs)
7. **Step 5**: Monitor progress
   - Watch speed graph
   - Check for bad sectors
   - View operation log
8. **Step 6**: Generate report

**Time:** 15-20 minutes for 100GB disk at 100MB/s

### Workflow 2: Restore an Image (Fyne GUI)

1. Launch GUI: `./diskimager ui`
2. Click **"Restore Image"**
3. **Step 1**: Select E01/Raw/VMDK image
4. **Step 2**: Select target disk
   - ⚠️ WARNING: All data will be erased!
5. **Step 3**: Review partition mapping
6. **Step 4**: Run safety checks
   - Confirm: "I understand this will ERASE ALL DATA"
7. **Step 5**: Monitor restoration
   - Real-time progress
   - Optional verification

### Workflow 3: Monitor Jobs Remotely (Web UI)

1. Start server: `./diskimager web`
2. Open browser: `http://localhost:8080`
3. View dashboard:
   - Active jobs counter
   - Available disks
   - Job queue with progress bars
4. Click any job for details:
   - Real-time speed graph
   - Phase tracking
   - Error logs

### Workflow 4: Automate with API

```bash
# Create job
JOB_ID=$(curl -s -X POST http://localhost:8080/api/jobs/image \
  -H "Content-Type: application/json" \
  -d '{
    "source_path": "/dev/disk1",
    "dest_path": "s3://evidence-bucket/case-001/disk.e01",
    "format": "E01",
    "compression": 6,
    "metadata": {
      "case_number": "CASE-2026-001",
      "examiner": "Automation Script"
    }
  }' | jq -r .id)

# Monitor progress
watch curl -s http://localhost:8080/api/jobs/$JOB_ID | jq .

# Wait for completion
while true; do
  STATUS=$(curl -s http://localhost:8080/api/jobs/$JOB_ID | jq -r .status)
  if [ "$STATUS" = "completed" ]; then
    echo "Job completed!"
    break
  fi
  sleep 5
done
```

---

## Tips & Tricks

### Fyne GUI Tips

1. **Keyboard Shortcuts:**
   - `Ctrl+I`: Quick imaging
   - `Ctrl+R`: Quick restore
   - `Ctrl+Q`: Quit

2. **Speed up pre-flight checks:**
   - Checks auto-run when you enter Step 4
   - No need to click button

3. **Save time on case info:**
   - Settings → Configure defaults
   - Pre-fill examiner name

4. **View logs:**
   - Execution step shows real-time log
   - Scroll to see all messages

### Web UI Tips

1. **Real-time updates:**
   - WebSocket connects automatically
   - Green dot = live connection

2. **Multiple jobs:**
   - Queue as many as needed
   - They run sequentially

3. **Speed graph:**
   - Shows last 20 data points
   - Updates every 500ms
   - Helps identify bottlenecks

4. **API automation:**
   - All operations available via REST
   - Perfect for scripting

---

## Troubleshooting Quick Fixes

### Fyne GUI Issues

**Black screen / Won't start:**
```bash
# macOS: Install Xcode command line tools
xcode-select --install

# Linux: Install dependencies
sudo apt-get install libgl1-mesa-dev xorg-dev
```

**Slow performance:**
- Lower compression level (0-3)
- Close other applications
- Check disk speed

### Web UI Issues

**API not responding:**
```bash
# Check if running
curl http://localhost:8080/health

# Should return: {"status":"ok","time":"..."}
```

**Frontend can't connect:**
```bash
# Check vite.config.ts proxy settings
# Should proxy /api to http://localhost:8080
```

**WebSocket fails:**
- Check firewall
- Ensure backend is on port 8080
- Try: `ws://localhost:8080/api/ws/jobs/test`

---

## Performance Benchmarks

### Expected Speeds

| Source | Destination | Speed | Notes |
|--------|-------------|-------|-------|
| SSD → Local SSD | 200-400 MB/s | Fast | Best case |
| HDD → Local HDD | 80-150 MB/s | Normal | Typical forensic imaging |
| Disk → E01 (compress 6) | 60-120 MB/s | Normal | With compression |
| Disk → S3 | 20-100 MB/s | Variable | Network dependent |
| USB → Local | 30-80 MB/s | Slow | USB 3.0 limitation |

### Optimization Tips

1. **Use Raw format** for fastest imaging (no compression overhead)
2. **Compression level 0** disables compression
3. **Direct-attached storage** faster than network
4. **SSD targets** significantly faster than HDD
5. **Disable antivirus** temporarily during imaging

---

## Next Steps

### Learn More
- Read full documentation: `GUI_DOCUMENTATION.md`
- Check API examples in code
- Review test files for usage patterns

### Advanced Features
- Cloud storage integration
- Batch operations
- Custom report templates
- API automation scripts

### Get Help
- Review error messages in logs
- Check SMART status for disk health
- Verify file permissions
- Test with small files first

---

## Production Checklist

Before deploying in production:

- [ ] Test imaging with known good disks
- [ ] Verify hash calculations match
- [ ] Test restore on non-critical disk
- [ ] Configure cloud credentials (if using)
- [ ] Set up default case number format
- [ ] Train users on wizard workflows
- [ ] Create standard operating procedures
- [ ] Test cancellation / interruption handling
- [ ] Verify report generation
- [ ] Document local storage paths

---

## Support Matrix

| OS | Fyne GUI | Web Server | Web Client |
|----|----------|------------|------------|
| macOS | ✅ | ✅ | ✅ |
| Linux | ✅ | ✅ | ✅ |
| Windows | ✅ | ✅ | ✅ |
| iOS | ❌ | ❌ | 🔶 (browser) |
| Android | ❌ | ❌ | 🔶 (browser) |

✅ = Fully Supported
🔶 = Partial Support
❌ = Not Supported

---

**You're ready to start! Choose your interface and begin imaging.**
