package virtual

import (
	"fmt"
	"os"
	"path/filepath"
)

// VMDKWriter creates VMware VMDK disk images
type VMDKWriter struct {
	descriptorFile *os.File
	flatFile       *os.File
	diskSize       int64
	bytesWritten   int64
}

// NewVMDKWriter creates a new VMDK writer
// Creates both descriptor file (.vmdk) and flat file (-flat.vmdk)
func NewVMDKWriter(basePath string, diskSize int64) (*VMDKWriter, error) {
	// Remove .vmdk extension if present
	basePath = removeExtension(basePath, ".vmdk")

	// Create flat file (actual data)
	flatPath := basePath + "-flat.vmdk"
	flatFile, err := os.OpenFile(flatPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create flat file: %w", err)
	}

	// Create descriptor file
	descriptorPath := basePath + ".vmdk"
	descriptorFile, err := os.Create(descriptorPath)
	if err != nil {
		flatFile.Close()
		return nil, fmt.Errorf("failed to create descriptor: %w", err)
	}

	w := &VMDKWriter{
		descriptorFile: descriptorFile,
		flatFile:       flatFile,
		diskSize:       diskSize,
	}

	// Write descriptor header immediately
	if err := w.writeDescriptor(filepath.Base(flatPath)); err != nil {
		w.Close()
		return nil, err
	}

	return w, nil
}

// Write implements io.Writer
func (w *VMDKWriter) Write(p []byte) (n int, err error) {
	n, err = w.flatFile.Write(p)
	w.bytesWritten += int64(n)
	return n, err
}

// Close finalizes the VMDK files
func (w *VMDKWriter) Close() error {
	// Close flat file first
	flatErr := w.flatFile.Close()

	// Close descriptor
	descErr := w.descriptorFile.Close()

	if flatErr != nil {
		return flatErr
	}
	return descErr
}

// writeDescriptor writes the VMDK descriptor file
func (w *VMDKWriter) writeDescriptor(flatFileName string) error {
	// Calculate disk geometry (CHS addressing)
	sectors := w.diskSize / 512
	cylinders := sectors / (255 * 63)
	if cylinders == 0 {
		cylinders = 1
	}

	descriptor := fmt.Sprintf(`# Disk DescriptorFile
version=1
encoding="UTF-8"
CID=fffffffe
parentCID=ffffffff
createType="monolithicFlat"

# Extent description
RW %d FLAT "%s" 0

# The Disk Data Base
ddb.virtualHWVersion = "8"
ddb.geometry.cylinders = "%d"
ddb.geometry.heads = "255"
ddb.geometry.sectors = "63"
ddb.adapterType = "ide"
`, sectors, flatFileName, cylinders)

	_, err := w.descriptorFile.WriteString(descriptor)
	return err
}

// VHDWriter creates Microsoft VHD disk images (fixed format)
type VHDWriter struct {
	file         *os.File
	diskSize     int64
	bytesWritten int64
}

// NewVHDWriter creates a new VHD writer (fixed disk format)
func NewVHDWriter(path string, diskSize int64) (*VHDWriter, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create VHD file: %w", err)
	}

	return &VHDWriter{
		file:     file,
		diskSize: diskSize,
	}, nil
}

// Write implements io.Writer
func (w *VHDWriter) Write(p []byte) (n int, err error) {
	n, err = w.file.Write(p)
	w.bytesWritten += int64(n)
	return n, err
}

// Close finalizes the VHD with footer
func (w *VHDWriter) Close() error {
	// Write VHD footer (512 bytes at end)
	footer := w.createVHDFooter()
	if _, err := w.file.Write(footer); err != nil {
		w.file.Close()
		return err
	}

	return w.file.Close()
}

// createVHDFooter creates a VHD fixed disk footer
func (w *VHDWriter) createVHDFooter() []byte {
	footer := make([]byte, 512)

	// Cookie "conectix"
	copy(footer[0:8], []byte("conectix"))

	// Features (0x00000002 = none)
	footer[8] = 0x00
	footer[9] = 0x00
	footer[10] = 0x00
	footer[11] = 0x02

	// File Format Version (1.0)
	footer[12] = 0x00
	footer[13] = 0x01
	footer[14] = 0x00
	footer[15] = 0x00

	// Data Offset (0xFFFFFFFF for fixed disk)
	footer[16] = 0xFF
	footer[17] = 0xFF
	footer[18] = 0xFF
	footer[19] = 0xFF
	footer[20] = 0xFF
	footer[21] = 0xFF
	footer[22] = 0xFF
	footer[23] = 0xFF

	// Current Size (big-endian)
	size := uint64(w.diskSize)
	footer[48] = byte(size >> 56)
	footer[49] = byte(size >> 48)
	footer[50] = byte(size >> 40)
	footer[51] = byte(size >> 32)
	footer[52] = byte(size >> 24)
	footer[53] = byte(size >> 16)
	footer[54] = byte(size >> 8)
	footer[55] = byte(size)

	// Original Size (same as current)
	copy(footer[56:64], footer[48:56])

	// Disk Geometry (CHS)
	cylinders := w.diskSize / (255 * 63 * 512)
	if cylinders > 65535 {
		cylinders = 65535
	}
	if cylinders == 0 {
		cylinders = 1
	}

	footer[64] = byte(cylinders >> 8)
	footer[65] = byte(cylinders)
	footer[66] = 255 // heads
	footer[67] = 63  // sectors

	// Disk Type (0x00000002 = fixed) - correct position at offset 60
	footer[60] = 0x00
	footer[61] = 0x00
	footer[62] = 0x00
	footer[63] = 0x02

	// Checksum calculation (ones complement of sum of footer)
	// VHD spec: checksum field is at offset 64-67, must be zeroed before calculation
	// Set checksum field to zero first
	footer[64] = 0x00
	footer[65] = 0x00
	footer[66] = 0x00
	footer[67] = 0x00

	checksum := uint32(0)
	for i := 0; i < 512; i++ {
		checksum += uint32(footer[i])
	}
	checksum = ^checksum // One's complement

	// Write checksum in big-endian format (VHD spec requires big-endian)
	footer[64] = byte(checksum >> 24)
	footer[65] = byte(checksum >> 16)
	footer[66] = byte(checksum >> 8)
	footer[67] = byte(checksum)

	return footer
}

// removeExtension removes specific extension from path
func removeExtension(path, ext string) string {
	if len(path) > len(ext) && path[len(path)-len(ext):] == ext {
		return path[:len(path)-len(ext)]
	}
	return path
}
