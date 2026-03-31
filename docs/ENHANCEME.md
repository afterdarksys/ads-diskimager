# diskimager Enhancement Roadmap

This document outlines potential improvements and features for the diskimager forensics tool, organized by category and priority.

## Critical Forensic Features

### 1. Bad Sector Handling
**Priority: HIGH** (mentioned in TODO at `imager/imager.go:75-79`)

- Implement retry logic with progressively smaller block sizes
- Zero-fill unreadable sectors and log their locations
- Continue imaging despite read errors (crucial for damaged disks)
- Track error map for later analysis

### 2. E01/EWF Format Support
**Priority: HIGH**

- Add Expert Witness Format (E01) output - industry standard
- Supports compression, encryption, and splitting
- Better for court evidence and tool compatibility
- Consider using `libewf` bindings

### 3. Verification Command
**Priority: HIGH**

- Separate `verify` command to check image integrity
- Compare source hash against stored hash
- Verify image file structure and metadata

### 4. Chain of Custody Tracking
**Priority: HIGH**

- Record examiner name, case number, evidence number
- Digital signatures for authenticity
- Timestamps for every operation
- Tamper-evident audit logs

## Performance & Reliability

### 5. Resume Capability
**Priority: HIGH**

- Save progress checkpoints during imaging
- Resume interrupted transfers from last checkpoint
- Critical for large disk images (multi-TB)

### 6. Compression Support
**Priority: MEDIUM**

- Inline compression (gzip, lz4, zstd)
- Significant storage savings for sparse disks
- Configurable compression levels

### 7. Multi-threaded Hashing
**Priority: MEDIUM**

- Calculate multiple hashes concurrently (MD5+SHA256)
- Parallel compression and hashing
- Better CPU utilization

### 8. Split Image Support
**Priority: MEDIUM**

- Split large images into multiple files
- Useful for FAT32 filesystems or network transfers
- Standard sizes (650MB for CD, 2GB, 4GB)

## Operational Improvements

### 9. Write-Blocking Validation
**Priority: HIGH**

- Detect and warn if source device isn't write-protected
- Check for mounted filesystems on source
- Prevent accidental data modification

### 10. Enhanced Metadata Collection
**Priority: MEDIUM**

- Disk geometry (heads, cylinders, sectors)
- SMART data from physical drives
- Partition table backup
- System information (hostname, OS, examiner)
- GPS coordinates (for field collection)

### 11. Better Progress Reporting
**Priority: MEDIUM**

- ETA calculation
- Transfer speed (MB/s)
- Percentage complete
- Estimated time remaining

### 12. Structured Logging
**Priority: MEDIUM**

- Use `logrus` or `zap` (already imported)
- Log levels (debug, info, warn, error)
- Structured JSON logs for automation
- Separate audit log from operational log

## Analysis Enhancements

### 13. Timeline Generation
**Priority: MEDIUM**

- MAC times (Modified, Accessed, Changed) extraction
- Timeline correlation across partitions
- Export to Plaso/log2timeline format

### 14. File Carving Integration
**Priority: MEDIUM**

- Recover deleted files from unallocated space
- Integration with `photorec` or `scalpel`
- Signature-based file recovery

### 15. Registry Analysis (Windows)
**Priority: LOW**

- Parse Windows registry hives
- Extract user accounts, installed software
- Recent activity analysis

### 16. YARA Scanning
**Priority: MEDIUM**

- Scan images for malware signatures
- Custom rule support
- Threat intelligence integration

## Server & Distribution

### 17. Database Backend
**Priority: MEDIUM**

- Track all images, cases, and evidence
- PostgreSQL or SQLite for metadata
- Query capabilities for case management

### 18. REST API
**Priority: MEDIUM**

- RESTful endpoints for automation
- Job queue for async operations
- Status polling for long-running tasks

### 19. Web UI
**Priority: LOW**

- Browser-based interface for non-technical users
- Case management dashboard
- Real-time progress monitoring

### 20. Container Support
**Priority: MEDIUM**

- Dockerfile for easy deployment
- Docker Compose for server setup
- Kubernetes manifests for scale

## Security Enhancements

### 21. Encryption at Rest
**Priority: MEDIUM**

- AES-256 encryption for stored images
- Key management integration (Vault, KMS)
- Encrypted transport (already has mTLS)

### 22. Role-Based Access Control
**Priority: LOW**

- User authentication and authorization
- Examiner, admin, viewer roles
- Audit trail for access

### 23. Secure Deletion
**Priority: LOW**

- DOD 5220.22-M compliant wiping
- Multiple overwrite passes
- Verification of deletion

## Quick Wins (Easy to Implement)

### 24. Image Deduplication
**Priority: LOW**

- Block-level dedup for storage efficiency
- Useful for similar systems (enterprise imaging)

### 25. Benchmark Mode
**Priority: LOW**

- Test read/write performance
- Identify optimal block size
- Hardware capability assessment

### 26. Dry Run Mode
**Priority: LOW**

- Simulate operations without writing
- Estimate time and storage requirements

### 27. Multiple Hash Algorithms Simultaneously
**Priority: MEDIUM**

- Calculate MD5, SHA1, SHA256 in one pass
- Common forensic requirement

### 28. Source Validation
**Priority: HIGH**

- Pre-imaging hash calculation
- Compare source disk before/after imaging
- Prove write-blocking worked

## Documentation & Compliance

### 29. NIST/ISO Compliance
**Priority: MEDIUM**

- Align with NIST SP 800-86 guidelines
- ISO 27037 compliance documentation
- ACPO principles adherence

### 30. Report Templates
**Priority: LOW**

- Customizable report generation
- PDF export with embedded signatures
- Case summary templates

## Top 5 Priorities

If implementing features incrementally, these should be tackled first:

1. **Bad Sector Handling** - Essential for real forensics work on damaged media
2. **E01/EWF Format Support** - Industry standard compatibility and legal admissibility
3. **Resume Capability** - Critical for large images and network interruptions
4. **Verification Command** - Prove integrity and build trust in the tool
5. **Chain of Custody** - Legal requirement for evidence handling

## Implementation Notes

- Many features can be implemented as separate commands to maintain CLI simplicity
- Consider feature flags for experimental features
- Maintain backwards compatibility with existing audit logs and image formats
- Focus on forensic soundness over convenience in core imaging functionality
- Performance optimizations should never compromise data integrity
