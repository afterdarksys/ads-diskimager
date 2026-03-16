# Disk Imager - Bug Report

## Critical Bugs

### 1. Resume Functionality Hash Mismatch (CRITICAL)
**Location:** `cmd/image.go:119`
**Severity:** CRITICAL - Breaks forensic integrity validation

**Issue:**
When resuming an imaging session, the hash calculated only covers the resumed portion of the file, not the entire file from the beginning. This makes the reported hash useless for forensic verification.

**Current Behavior:**
```go
// cmd/image.go:119
if res != nil {
    res.BytesCopied += existingBytesCopied  // Adds existing bytes to count
}
// But the hash in res.Hash only covers the newly copied data!
```

**Test Case:**
```bash
# Create initial image
echo "test data for resume" > input.txt
./diskimager image --in input.txt --out output.img --hash sha256
# Reports hash: 93789f5d4194c79b1abc2078c909f9742aec96b5bdb32cc056e9b43fa72998b6

# Add more data and resume
echo "additional data" >> input.txt
./diskimager image --in input.txt --out output.img --hash sha256 --resume
# Reports hash: 90ab62ce16754127a216e7eaf2cb27459161ef53cc8913cc7133192014dcee6d

# Actual file hash
sha256sum output.img
# Actual hash: e4a01f5c6cada5b75e6b0b79bfca1e15fa4aee94a59817309812950b4a24f973
# MISMATCH!
```

**Impact:**
- Forensic chain of custody is broken
- Cannot verify image integrity
- Evidence may be inadmissible in court

**Fix Required:**
When resuming, the hash needs to be calculated incrementally from the existing file, or the tool should note that resume invalidates the hash and require re-hashing the entire file afterward.

---

### 2. Bad Sector Handling Not Seekable-Safe
**Location:** `imager/imager.go:119-130`
**Severity:** HIGH - Fails on non-seekable sources

**Issue:**
The bad sector recovery logic assumes the source is seekable (`io.Seeker`). If the source is stdin, a pipe, or network stream, it will fail with "unrecoverable read error" even though zero-padding was applied.

**Code:**
```go
if seeker, ok := cfg.Source.(io.Seeker); ok {
    _, seekErr := seeker.Seek(int64(remaining), io.SeekCurrent)
    if seekErr != nil {
        return res, fmt.Errorf("failed to seek past bad sector: %w", seekErr)
    }
    continue
} else {
    // Stream is broken - abort
    return res, fmt.Errorf("unrecoverable read error at offset %d: %w", res.BytesCopied, err)
}
```

**Impact:**
- Cannot image from stdin/pipes with bad sector handling
- Remote streaming with bad sectors will fail
- Limits use cases for forensic acquisition

**Fix Required:**
For non-seekable streams, after zero-padding, mark the position as recovered and attempt to continue reading. The stream will naturally skip past the bad data.

---

### 3. E01 Missing Done Section
**Location:** `pkg/format/e01/writer.go:154-172`
**Severity:** MEDIUM - E01 files may not be recognized by standard tools

**Issue:**
The E01 writer doesn't write a proper "done" section marker that compliant EWF parsers expect. Standard forensic tools (FTK, EnCase) may reject the file.

**Code:**
```go
func (w *Writer) Close() error {
    // Writes table magic/section
    tableMagic := "table2"
    w.file.Write([]byte(tableMagic))
    // ... writes chunk info ...
    return w.file.Close()  // Missing "done" section!
}
```

**Impact:**
- Generated E01 files may be rejected by industry-standard tools
- Interoperability issues with forensic software
- Evidence may not be processable in standard workflows

**Fix Required:**
After writing the table section, write a proper "done" section with the EWF done marker.

---

### 4. E01 Writer Missing Error Checks
**Location:** `pkg/format/e01/writer.go:160-171`
**Severity:** MEDIUM - Silent data corruption possible

**Issue:**
The Close() method doesn't check for write errors when writing the table section. If disk is full or write fails, the E01 file will be corrupted silently.

**Code:**
```go
// No error checking on these writes:
w.file.Write([]byte(tableMagic))
binary.Write(w.file, binary.LittleEndian, uint32(w.chunkCount))
for _, offset := range w.offsetList {
    binary.Write(w.file, binary.LittleEndian, uint32(offset))
}
```

**Impact:**
- Silent corruption of E01 files
- No indication of write failure
- Incomplete forensic images that appear complete

**Fix Required:**
Check all write errors and return them. Consider using `defer` to ensure proper cleanup on error.

---

## Medium Priority Bugs

### 5. E01 Offset Overflow (32-bit limitation)
**Location:** `pkg/format/e01/writer.go:168`
**Severity:** MEDIUM - Files larger than 4GB will corrupt

**Issue:**
EWF1 format uses 32-bit offsets, limiting addressable space to 4GB. Code casts 64-bit offsets to 32-bit without checking for overflow.

**Code:**
```go
for _, offset := range w.offsetList {
    binary.Write(w.file, binary.LittleEndian, uint32(offset)) // Truncates > 4GB
}
```

**Impact:**
- Files larger than 4GB will have corrupted offset tables
- Unable to read chunks correctly
- Silent data corruption

**Fix Required:**
Either:
1. Add validation to reject files > 4GB for EWF1
2. Implement EWF2 format (64-bit offsets)
3. Implement split file support

---

### 6. io.ReadFull Semantic Issue
**Location:** `imager/imager.go:74`
**Severity:** LOW - Potential edge case issues

**Issue:**
Using `io.ReadFull` changes the semantics from regular Read(). `io.ReadFull` tries to fill the entire buffer and returns `io.ErrUnexpectedEOF` if it hits EOF before filling the buffer. The current code handles this, but it's more complex than needed.

**Current:**
```go
nr, err := io.ReadFull(cfg.Source, buf)
if err == io.EOF {
    break
}
if err == io.ErrUnexpectedEOF {
    break  // Last partial block
}
```

**Recommendation:**
Use regular `Read()` instead of `ReadFull()` for simpler semantics, unless block-aligned reads are required for bad sector handling on block devices.

---

### 7. Verify Command Incomplete for E01
**Location:** `cmd/verify.go:38-44`
**Severity:** LOW - Documented limitation

**Issue:**
The verify command documents that it doesn't properly verify E01 files - it hashes the container file, not the reconstructed raw image data.

**Code Comment:**
```go
// In a real E01 integration, we would need to read and uncompress the chunks
// using the EWF logic to reconstruct the raw image hash or verify internal Adler32s.
```

**Impact:**
- E01 verification is incomplete
- Cannot verify forensic image integrity properly
- Users may think verification passed when it only verified the container

**Fix Required:**
Implement E01 reader to decompress chunks and hash the reconstructed raw data.

---

## Low Priority / Enhancement Issues

### 8. Progress Reporting Not Synchronized
**Location:** `cmd/image.go:192`
**Severity:** LOW - UI issue only

**Issue:**
Progress reporting uses modulo arithmetic without synchronization. Multiple threads or rapid updates could cause missed or duplicate progress messages.

**Code:**
```go
if pr.BytesRead%(1024*1024*10) == 0 {
    fmt.Printf("\rCopied: %d bytes", pr.BytesRead)
}
```

**Impact:**
- Minor UI glitches in progress reporting
- No functional impact on imaging

**Fix Required:**
Add proper synchronization or use atomic operations for BytesRead.

---

### 9. GUI Progress Update Inefficiency
**Location:** `cmd/ui.go:67`
**Severity:** LOW - Performance issue

**Issue:**
GUI progress updates check modulo on every read, which is wasteful.

**Code:**
```go
if pr.CopiedBytes%(1024*1024*5) == 0 {
    if pr.TotalBytes > 0 {
        pr.ProgressBar.SetValue(float64(pr.CopiedBytes) / float64(pr.TotalBytes))
    }
}
```

**Impact:**
- Unnecessary CPU cycles on modulo operation
- Could impact performance on very fast reads

**Fix Required:**
Use a threshold approach: track last reported value and only update when difference exceeds threshold.

---

### 10. Missing Input Validation
**Location:** Multiple locations
**Severity:** LOW - Usability issue

**Issues:**
- No validation that hash algorithm is supported before starting long operations
- No check for disk space before imaging
- No warning if source device is mounted
- No validation of block size (could be 0 or negative theoretically)

**Impact:**
- Poor user experience
- Wasted time on operations that will fail
- Potential data safety issues

**Fix Required:**
Add pre-flight validation checks before starting imaging operations.

---

### 11. Race Condition in GUI
**Location:** `cmd/ui.go:100-169`
**Severity:** LOW - Thread safety issue

**Issue:**
GUI updates from goroutine without proper Fyne thread synchronization. Fyne requires UI updates to happen on the main thread.

**Code:**
```go
go func() {
    // ... imaging work ...
    statusLabel.SetText("Success! ...") // Unsafe UI update from goroutine
}()
```

**Impact:**
- Potential crashes on some platforms
- UI corruption
- Undefined behavior

**Fix Required:**
Use `fyne.CurrentApp().SendNotification()` or channels to marshal UI updates to main thread.

---

## Test Coverage Gaps

### Missing Tests
1. Resume functionality with hash validation
2. Bad sector handling with non-seekable streams
3. E01 writer error handling
4. E01 files > 4GB
5. Progress reporting accuracy
6. Concurrent operations safety

---

## Summary

**Critical Issues:** 2
**High Priority:** 1
**Medium Priority:** 4
**Low Priority:** 4

**Recommended Fix Priority:**
1. Resume hash mismatch (CRITICAL)
2. Bad sector handling for streams (HIGH)
3. E01 error checking (MEDIUM)
4. E01 done section (MEDIUM)
5. E01 offset overflow (MEDIUM)
6. Input validation (LOW)
7. GUI thread safety (LOW)

---

## Testing Performed

```bash
# Basic imaging
./diskimager image --in test.txt --out test.img --hash sha256
# ✓ Works

# Resume functionality
./diskimager image --in test.txt --out test.img --resume
# ✗ Hash mismatch bug confirmed

# E01 format
./diskimager image --in test.txt --out test.e01 --format e01
# ✓ Creates file, but missing done section

# Progress reporting
dd if=/dev/zero bs=1M count=25 | ./diskimager image --in /dev/stdin --out test.img
# ✓ Progress prints correctly

# Unit tests
go test ./...
# ✓ All tests pass (but missing coverage for above bugs)
```

---

*Report Generated: 2026-03-16*
*Testing Environment: macOS 24.6.0, Go 1.25.2*
