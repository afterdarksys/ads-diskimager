# Find Command - Forensic File Search & Extraction Guide

## Overview

The `find` command searches for files within disk images using pattern matching and can extract them into various archive formats. This is perfect for **targeted evidence collection** without modifying the original disk image.

---

## 🎯 Key Features

- ✅ **Non-destructive** - Operates on disk images only
- ✅ **Pattern matching** - Glob patterns (*.doc, *.exe, etc.)
- ✅ **Auto-detection** - Automatically detects filesystem type
- ✅ **Multiple formats** - Extract to tar.bz2, tar.gz, cpio.bz2
- ✅ **Hash calculation** - Optional SHA256 for each file
- ✅ **Metadata preservation** - Timestamps, permissions
- ✅ **TSK integration** - Uses The Sleuth Kit if available
- ✅ **Recursive search** - Search all subdirectories

---

## 📚 Quick Examples

### 1. Find All Office Documents (Print Only)

```bash
./diskimager find \
  --disk evidence.img \
  --patterns "*.doc,*.docx,*.xls,*.xlsx,*.ppt,*.pptx" \
  --output print
```

**Output:**
```
Auto-detecting filesystem...
✓ Detected filesystem: ntfs
Searching for patterns: *.doc, *.docx, *.xls, *.xlsx, *.ppt, *.pptx

✓ Found 127 file(s) in 3.2s

Files found:
--------------------------------------------------------------------------------
   1. /Users/John/Documents/Project_Plan.docx
   2. /Users/John/Documents/Budget_2024.xlsx
   3. /Users/John/Downloads/Presentation.pptx
...
--------------------------------------------------------------------------------
Total: 127 file(s)
```

---

### 2. Extract PDFs to Archive

```bash
./diskimager find \
  --disk evidence.img \
  --patterns "*.pdf" \
  --output evidence_pdfs.tar.bz2
```

**Result:**
- Creates `evidence_pdfs.tar.bz2` containing all PDFs
- Preserves directory structure
- Compressed with bzip2

---

### 3. Find Executables with Hashes

```bash
./diskimager find \
  --disk evidence.img \
  --patterns "*.exe,*.dll" \
  --output executables.tar.bz2 \
  --hash \
  --verbose
```

**Features:**
- Extracts all .exe and .dll files
- Calculates SHA256 hash for each file
- Creates .sha256 files alongside each executable
- Verbose output shows file details

**Output:**
```
Files found:
--------------------------------------------------------------------------------
   1. /Windows/System32/notepad.exe
      Size:     223232 bytes
      Modified: 2024-01-15T10:30:45Z
      SHA256:   a1b2c3d4e5f6...

   2. /Program Files/App/malware.exe
      Size:     45056 bytes
      Modified: 2024-03-10T15:22:11Z
      SHA256:   deadbeef1234...
--------------------------------------------------------------------------------
```

---

### 4. Extract Images (Multiple Patterns)

```bash
./diskimager find \
  --disk evidence.img \
  --patterns "*.jpg,*.jpeg,*.png,*.gif,*.bmp" \
  --output images.tar.gz \
  --format tar.gz
```

---

### 5. CPIO Archive Format

```bash
./diskimager find \
  --disk evidence.img \
  --patterns "*.log,*.txt" \
  --output logs.cpio.bz2 \
  --format cpio.bz2
```

**Why CPIO?**
- Standard for Unix backups
- Preserves file permissions better
- Compatible with `cpio -i` for extraction

---

## 🔧 Command Reference

### Required Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--disk` | Disk image to search | `--disk evidence.img` |
| `--patterns` | File patterns (comma-separated) | `--patterns "*.doc,*.pdf"` |

### Optional Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--fs` | `auto` | Filesystem type (auto, ext4, ntfs, fat32, apfs) |
| `--output` | `print` | Output destination (print or file path) |
| `--format` | auto | Archive format (tar, tar.bz2, tar.gz, cpio.bz2) |
| `--recursive` | `true` | Recursive search |
| `--verbose` | `false` | Show file details |
| `--hash` | `false` | Calculate SHA256 for each file |

---

## 📂 Supported Filesystems

### With TSK (The Sleuth Kit):
- ✅ **NTFS** - Windows filesystems
- ✅ **FAT12/FAT16/FAT32** - Legacy Windows, USB drives
- ✅ **exFAT** - Modern USB drives
- ✅ **Ext2/Ext3/Ext4** - Linux filesystems
- ✅ **HFS+** - macOS (older)
- ✅ **APFS** - macOS (modern)
- ✅ **ISO 9660** - CD-ROM images
- ✅ **UDF** - DVD images

### Without TSK (mount-based):
- ✅ **Ext2/Ext3/Ext4** - Linux (requires sudo)
- ✅ **NTFS** - Windows (requires ntfs-3g)
- ✅ **FAT32** - USB drives

**Recommendation:** Install The Sleuth Kit for best results:
```bash
# macOS
brew install sleuthkit

# Ubuntu/Debian
sudo apt-get install sleuthkit

# CentOS/RHEL
sudo yum install sleuthkit
```

---

## 🎓 Forensic Use Cases

### 1. Incident Response - Find Suspicious Files

```bash
# Find recently modified executables
./diskimager find \
  --disk compromised_system.img \
  --patterns "*.exe,*.dll,*.sys" \
  --output suspicious_binaries.tar.bz2 \
  --hash
```

**Then analyze hashes:**
```bash
# Extract archive
tar -xjf suspicious_binaries.tar.bz2

# Check hashes against VirusTotal
for f in *.exe.sha256; do
  hash=$(cat "$f")
  echo "Checking $hash..."
  # curl to VirusTotal API
done
```

---

### 2. eDiscovery - Collect Documents

```bash
# Find all emails and documents
./diskimager find \
  --disk employee_laptop.img \
  --patterns "*.pst,*.ost,*.eml,*.msg,*.doc*,*.xls*,*.pdf" \
  --output ediscovery_collection.tar.bz2 \
  --verbose
```

---

### 3. Malware Analysis - Extract IOCs

```bash
# Find all scripts and executables
./diskimager find \
  --disk malware_sample.img \
  --patterns "*.exe,*.dll,*.ps1,*.vbs,*.js,*.bat,*.cmd" \
  --output iocs.tar.bz2 \
  --hash
```

---

### 4. Data Recovery - Find Lost Files

```bash
# Find all photos
./diskimager find \
  --disk damaged_disk.img \
  --patterns "*.jpg,*.jpeg,*.png,*.raw,*.cr2,*.nef" \
  --output recovered_photos.tar.gz \
  --verbose
```

---

### 5. Compliance Audit - Find Sensitive Data

```bash
# Find potential PII files
./diskimager find \
  --disk audit_disk.img \
  --patterns "*ssn*,*tax*,*password*,*credential*" \
  --output potential_pii.tar.bz2 \
  --hash
```

---

## 🔍 Pattern Matching

### Glob Patterns

```bash
# Single pattern
--patterns "*.doc"

# Multiple patterns (comma-separated)
--patterns "*.doc,*.pdf,*.txt"

# Wildcard matching
--patterns "report*.xlsx"      # report2024.xlsx, report_final.xlsx

# Multiple extensions
--patterns "*.{doc,docx}"       # Both .doc and .docx

# Case-insensitive (on most systems)
--patterns "*.PDF,*.pdf"
```

### Complex Patterns

```bash
# Find all Office documents
--patterns "*.doc,*.docx,*.xls,*.xlsx,*.ppt,*.pptx"

# Find all images
--patterns "*.jpg,*.jpeg,*.png,*.gif,*.bmp,*.tiff"

# Find all archives
--patterns "*.zip,*.rar,*.7z,*.tar,*.gz,*.bz2"

# Find all code files
--patterns "*.py,*.java,*.c,*.cpp,*.js,*.go"
```

---

## 📦 Archive Formats

### TAR (Uncompressed)
```bash
--format tar
```
- **Pros:** Fast, standard, universally supported
- **Cons:** Large file size
- **Use When:** Speed is priority, will compress later

### TAR.BZ2 (Bzip2 Compression)
```bash
--format tar.bz2
```
- **Pros:** High compression ratio, good for documents
- **Cons:** Slower than gzip
- **Use When:** Storage/bandwidth limited, best compression needed
- **Typical Ratio:** 60-80% reduction

### TAR.GZ (Gzip Compression)
```bash
--format tar.gz
```
- **Pros:** Fast compression, universally supported
- **Cons:** Lower compression than bzip2
- **Use When:** Need balance of speed and size
- **Typical Ratio:** 50-70% reduction

### CPIO.BZ2 (CPIO with Bzip2)
```bash
--format cpio.bz2
```
- **Pros:** Better permission preservation, standard for backups
- **Cons:** Less common than tar
- **Use When:** Need exact permission/ownership preservation

---

## 🚀 Advanced Usage

### Workflow 1: Preview then Extract

```bash
# Step 1: Preview results
./diskimager find \
  --disk evidence.img \
  --patterns "*.exe" \
  --output print \
  --verbose

# Step 2: If results look good, extract
./diskimager find \
  --disk evidence.img \
  --patterns "*.exe" \
  --output executables.tar.bz2 \
  --hash
```

---

### Workflow 2: Multiple Targeted Extractions

```bash
# Extract documents
./diskimager find \
  --disk evidence.img \
  --patterns "*.doc,*.docx,*.pdf" \
  --output documents.tar.bz2

# Extract images
./diskimager find \
  --disk evidence.img \
  --patterns "*.jpg,*.png" \
  --output images.tar.bz2

# Extract emails
./diskimager find \
  --disk evidence.img \
  --patterns "*.pst,*.eml,*.msg" \
  --output emails.tar.bz2
```

---

### Workflow 3: Hash Verification Chain

```bash
# Step 1: Extract with hashes
./diskimager find \
  --disk evidence.img \
  --patterns "*.exe" \
  --output malware.tar.bz2 \
  --hash

# Step 2: Extract archive
tar -xjf malware.tar.bz2

# Step 3: Verify hashes
for f in *.exe; do
  echo "File: $f"
  cat "$f.sha256"
  sha256sum "$f"
  echo "---"
done
```

---

## ⚠️ Important Notes

### TSK vs Mount-Based Extraction

**The Sleuth Kit (TSK) - RECOMMENDED:**
- ✅ Non-destructive (no mounting needed)
- ✅ Works on any system
- ✅ No sudo required
- ✅ Handles deleted files
- ✅ More filesystem types

**Mount-Based:**
- ⚠️ Requires sudo (root access)
- ⚠️ Mounts image (potential modification)
- ⚠️ Limited filesystem support
- ⚠️ Cannot access deleted files

**Install TSK for best experience:**
```bash
brew install sleuthkit  # macOS
apt install sleuthkit   # Ubuntu/Debian
```

---

### Forensic Best Practices

1. **Always work on copies, never originals**
   ```bash
   # Create working copy first
   cp original_evidence.img working_copy.img

   # Use working copy for search
   ./diskimager find --disk working_copy.img ...
   ```

2. **Document your searches**
   ```bash
   # Save search results
   ./diskimager find \
     --disk evidence.img \
     --patterns "*.exe" \
     --output print \
     --verbose > search_results.txt
   ```

3. **Hash everything**
   ```bash
   # Always use --hash for evidence collection
   ./diskimager find \
     --disk evidence.img \
     --patterns "*.doc" \
     --output documents.tar.bz2 \
     --hash
   ```

4. **Preserve chain of custody**
   - Document date/time of extraction
   - Note which patterns were searched
   - Save search output logs
   - Include in case notes

---

## 🐛 Troubleshooting

### Error: "fls: command not found"
**Solution:** Install The Sleuth Kit
```bash
brew install sleuthkit  # macOS
```

### Error: "Failed to detect filesystem"
**Solution:** Specify filesystem manually
```bash
./diskimager find \
  --disk evidence.img \
  --fs ntfs \
  --patterns "*.doc"
```

### Error: "mount failed (may need sudo)"
**Solution 1:** Use TSK instead (install sleuthkit)

**Solution 2:** Run with sudo
```bash
sudo ./diskimager find \
  --disk evidence.img \
  --patterns "*.doc" \
  --output docs.tar.bz2
```

### Error: "No files found matching patterns"
**Troubleshooting:**
```bash
# 1. Verify filesystem detection
./diskimager find --disk evidence.img --patterns "*" --output print | head -20

# 2. Try different patterns
--patterns "*"              # Find all files
--patterns "*.DOC,*.doc"    # Try case variations

# 3. Check verbose output
--verbose
```

---

## 📊 Performance Tips

### Large Images (> 100GB)
```bash
# Use specific patterns (not wildcards)
--patterns "report.docx"  # Better than "*.docx"

# Extract to fast storage
--output /fast/ssd/output.tar.bz2
```

### Many Small Files
```bash
# Use tar (no compression) for speed
--format tar

# Or use gzip instead of bzip2
--format tar.gz
```

### Network Storage
```bash
# Compress heavily for network transfer
--format tar.bz2  # Best compression

# Or extract locally first, then transfer
--output /local/fast/output.tar.bz2
```

---

## 🎯 Summary

**The `find` command is perfect for:**
- ✅ Targeted evidence collection
- ✅ eDiscovery document gathering
- ✅ Malware sample extraction
- ✅ Data recovery from images
- ✅ Compliance auditing
- ✅ Incident response

**Key Advantages:**
- Non-destructive (works on images)
- Fast pattern-based search
- Multiple archive formats
- Hash verification support
- TSK integration for robustness

---

## 📚 See Also

- [README.md](README.md) - Main documentation
- [STATE_OF_THE_ART_FEATURES.md](STATE_OF_THE_ART_FEATURES.md) - Advanced features
- [The Sleuth Kit Documentation](https://www.sleuthkit.org/sleuthkit/docs.php)

---

**Version:** 2.0.0
**Last Updated:** 2026-03-28
