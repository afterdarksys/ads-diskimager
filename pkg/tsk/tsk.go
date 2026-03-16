package tsk

/*
#cgo CFLAGS: -I/usr/local/opt/sleuthkit/include
#cgo LDFLAGS: -L/usr/local/opt/sleuthkit/lib -ltsk
#include <tsk/libtsk.h>
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

// Image wraps TSK_IMG_INFO
type Image struct {
	imgInfo *C.TSK_IMG_INFO
}

// OpenImage opens a disk image file.
func OpenImage(path string) (*Image, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	// TSK_IMG_OPEN_SING -> open single image
	// 0 -> type detection (autodetect)
	// 0 -> sector size (0 = autodetect)
	imgInfo := C.tsk_img_open_sing(cPath, C.TSK_IMG_TYPE_DETECT, 0)
	if imgInfo == nil {
		// TSK error handling is global/thread-local usually, tricky to get exact string without more CGO
		return nil, errors.New("failed to open image with libtsk")
	}

	return &Image{imgInfo: imgInfo}, nil
}

// Close closes the image.
func (img *Image) Close() {
	if img.imgInfo != nil {
		C.tsk_img_close(img.imgInfo)
		img.imgInfo = nil
	}
}

// Filesystem wraps TSK_FS_INFO
type Filesystem struct {
	fsInfo *C.TSK_FS_INFO
}

// OpenFS opens a filesystem at a specific offset (in bytes).
func (img *Image) OpenFS(offset int64) (*Filesystem, error) {
	// tsk_fs_open_img(img_info, offset, type)
	fsInfo := C.tsk_fs_open_img(img.imgInfo, C.TSK_OFF_T(offset), C.TSK_FS_TYPE_DETECT)
	if fsInfo == nil {
		return nil, fmt.Errorf("failed to open filesystem at offset %d", offset)
	}

	return &Filesystem{fsInfo: fsInfo}, nil
}

// Close closes the filesystem.
func (fs *Filesystem) Close() {
	if fs.fsInfo != nil {
		C.tsk_fs_close(fs.fsInfo)
		fs.fsInfo = nil
	}
}
