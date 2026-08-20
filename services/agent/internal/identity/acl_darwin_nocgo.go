//go:build darwin && !cgo

package identity

import "errors"

func hasExtendedACL(_ int, _ bool) (bool, error) {
	return false, errors.New("agent identity: ACL inspection requires cgo on darwin")
}
