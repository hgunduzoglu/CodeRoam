//go:build darwin && cgo

package identity

/*
#include <errno.h>
#include <sys/acl.h>

static int coderoam_has_extended_acl(int fd) {
	errno = 0;
	acl_t acl = acl_get_fd_np(fd, ACL_TYPE_EXTENDED);
	if (acl == NULL) {
		if (errno == ENOENT) {
			return 0;
		}
		return errno == 0 ? -EIO : -errno;
	}
	acl_entry_t entry;
	int result = acl_get_entry(acl, ACL_FIRST_ENTRY, &entry);
	int saved_errno = errno;
	acl_free(acl);
	if (result == 0) {
		return 1;
	}
	if (result == -1 && saved_errno == EINVAL) {
		return 0;
	}
	return saved_errno == 0 ? -EIO : -saved_errno;
}
*/
import "C"

import (
	"syscall"
)

func hasExtendedACL(fd int, _ bool) (bool, error) {
	result := int(C.coderoam_has_extended_acl(C.int(fd)))
	if result < 0 {
		return false, syscall.Errno(-result)
	}
	return result == 1, nil
}
