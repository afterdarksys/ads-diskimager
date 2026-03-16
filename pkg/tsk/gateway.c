#include <tsk/libtsk.h>
#include "_cgo_export.h"

TSK_WALK_RET_ENUM walk_cb_gateway(TSK_FS_FILE *a_fs_file, const char *a_path, void *a_ptr) {
    return goWalkCb(a_fs_file, (char*)a_path, a_ptr);
}
