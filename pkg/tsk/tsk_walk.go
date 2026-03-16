package tsk

/*
#cgo CFLAGS: -I/usr/local/opt/sleuthkit/include
#cgo LDFLAGS: -L/usr/local/opt/sleuthkit/lib -ltsk
#include <tsk/libtsk.h>

// Helper wrapper declared in gateway.c
extern TSK_WALK_RET_ENUM walk_cb_gateway(TSK_FS_FILE *a_fs_file, const char *a_path, void *a_ptr);
*/
import "C"
import (
	"errors"
	"io"
	"runtime/cgo"
	"unsafe"
)

// File wraps TSK_FS_FILE to allow reading.
// Note: The pointer is only valid during the callback!
type File struct {
	fsFile *C.TSK_FS_FILE
}

// Name returns the file name.
func (f *File) Name() string {
	if f.fsFile == nil || f.fsFile.name == nil {
		return ""
	}
	return C.GoString(f.fsFile.name.name)
}

// Size returns the file size.
func (f *File) Size() int64 {
	if f.fsFile == nil || f.fsFile.meta == nil {
		return 0
	}
	return int64(f.fsFile.meta.size)
}

// IsDir checks if it is a directory.
func (f *File) IsDir() bool {
	if f.fsFile == nil || f.fsFile.meta == nil {
		return false
	}
	return (f.fsFile.meta._type & C.TSK_FS_META_TYPE_DIR) != 0
}

// ReadAt reads data from the file at offset.
func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	if f.fsFile == nil {
		return 0, errors.New("invalid file handle")
	}
	
	// tsk_fs_file_read(file, offset, buf, len, flags)
	// returns bytes read or -1 on error.
	
	if len(p) == 0 {
		return 0, nil
	}
	
	cBuf := (*C.char)(unsafe.Pointer(&p[0]))
	bytesRead := C.tsk_fs_file_read(f.fsFile, C.TSK_OFF_T(off), cBuf, C.size_t(len(p)), C.TSK_FS_FILE_READ_FLAG_NONE)
	
	if bytesRead == -1 {
		return 0, errors.New("tsk_fs_file_read error")
	}
	
	if bytesRead < C.ssize_t(len(p)) {
		// EOF or short read
		if bytesRead == 0 {
			return 0, io.EOF
		}
		return int(bytesRead), nil // Caller handles short reads or we return EOF next time
	}
	
	return int(bytesRead), nil
}

//export goWalkCb
func goWalkCb(a_fs_file *C.TSK_FS_FILE, a_path *C.char, a_ptr unsafe.Pointer) C.TSK_WALK_RET_ENUM {
	h := cgo.Handle(a_ptr)
	// Value is func(*File, string) error
	callback := h.Value().(func(*File, string) error)

	path := C.GoString(a_path)
	file := &File{fsFile: a_fs_file}

	if err := callback(file, path); err != nil {
		// Stop walking on error
		return C.TSK_WALK_ERROR
	}
	return C.TSK_WALK_CONT
}

// Walk iterates over the filesystem starting at root.
func (fs *Filesystem) Walk(callback func(file *File, path string) error) error {
	h := cgo.NewHandle(callback)
	defer h.Delete()

	// TSK_FS_DIR_WALK_FLAG_RECURSE | TSK_FS_DIR_WALK_FLAG_NOORPHAN | TSK_FS_DIR_WALK_FLAG_ALLOC | TSK_FS_DIR_WALK_FLAG_UNALLOC
	// For now, let's just do RECURSE + NOORPHAN (standard logical walk)
	flags := C.TSK_FS_DIR_WALK_FLAG_ENUM(C.TSK_FS_DIR_WALK_FLAG_RECURSE | C.TSK_FS_DIR_WALK_FLAG_NOORPHAN)

	// Root inode is usually fsInfo->root_inum.
	rootInum := fs.fsInfo.root_inum

	ret := C.tsk_fs_dir_walk(fs.fsInfo, rootInum, flags, (C.TSK_FS_DIR_WALK_CB)(C.walk_cb_gateway), unsafe.Pointer(h))
	
	if ret == 1 {
		return errors.New("tsk_fs_dir_walk failed")
	}
	return nil
}
