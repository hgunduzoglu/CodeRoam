//go:build linux

package identity

import (
	"errors"

	"golang.org/x/sys/unix"
)

func hasExtendedACL(fd int, directory bool) (bool, error) {
	attributes := []string{"system.posix_acl_access"}
	if directory {
		attributes = append(attributes, "system.posix_acl_default")
	}
	for _, attribute := range attributes {
		size, err := unix.Fgetxattr(fd, attribute, nil)
		if err == nil {
			return size >= 0, nil
		}
		if errors.Is(err, unix.ENODATA) {
			continue
		}
		if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EOPNOTSUPP) {
			continue
		}
		return false, err
	}
	return false, nil
}
