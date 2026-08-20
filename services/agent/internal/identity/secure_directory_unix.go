//go:build linux || darwin

package identity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type secureDirectory struct {
	fd   int
	path string
	dev  uint64
	ino  uint64
}

type unixStat = unix.Stat_t

func openIdentityDirectory(path string, create bool) (*secureDirectory, error) {
	if create {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return nil, fmt.Errorf("create agent identity directory: %w", err)
		}
	}
	return openValidatedDirectoryChain(path)
}

func openValidatedDirectoryChain(path string) (*secureDirectory, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == string(filepath.Separator) {
		return nil, ErrInsecureStorage
	}
	fd, err := unix.Open(
		string(filepath.Separator),
		unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0,
	)
	if err != nil {
		return nil, ErrInsecureStorage
	}
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("inspect identity root: %w", err)
	}
	if err := validateAncestorStat(fd, &stat); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}

	components := strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator))
	for index, component := range components {
		nextFD, err := unix.Openat(
			fd,
			component,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
			0,
		)
		_ = unix.Close(fd)
		if errors.Is(err, unix.ENOENT) {
			return nil, ErrIdentityNotFound
		}
		if err != nil {
			return nil, ErrInsecureStorage
		}
		fd = nextFD
		if err := unix.Fstat(fd, &stat); err != nil {
			_ = unix.Close(fd)
			return nil, fmt.Errorf("inspect identity path component: %w", err)
		}
		if index == len(components)-1 {
			if err := validateDirectoryStat(fd, &stat); err != nil {
				_ = unix.Close(fd)
				return nil, err
			}
		} else if err := validateAncestorStat(fd, &stat); err != nil {
			_ = unix.Close(fd)
			return nil, err
		}
	}
	return &secureDirectory{
		fd:   fd,
		path: path,
		dev:  uint64(stat.Dev),
		ino:  stat.Ino,
	}, nil
}

func validateAncestorStat(fd int, stat *unix.Stat_t) error {
	if stat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		(stat.Uid != 0 && stat.Uid != uint32(os.Geteuid())) {
		return ErrInsecureStorage
	}
	if stat.Mode&0o022 != 0 {
		rootOwnedSticky := stat.Uid == 0 && stat.Mode&unix.S_ISVTX != 0
		if !rootOwnedSticky {
			return ErrInsecureStorage
		}
	}
	hasACL, err := hasExtendedACL(fd, true)
	if err != nil || hasACL {
		return ErrInsecureStorage
	}
	return nil
}

func validateDirectoryStat(fd int, stat *unix.Stat_t) error {
	if stat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		stat.Mode&0o777 != 0o700 ||
		stat.Uid != uint32(os.Geteuid()) {
		return ErrInsecureStorage
	}
	hasACL, err := hasExtendedACL(fd, true)
	if err != nil || hasACL {
		return ErrInsecureStorage
	}
	return nil
}

func validateIdentityFileStat(fd int, stat *unix.Stat_t) error {
	if err := validateIdentityFileAccessStat(fd, stat); err != nil {
		return err
	}
	if stat.Size <= 0 || stat.Size > maxIdentityFileSize {
		return ErrInvalidIdentity
	}
	return nil
}

func validateIdentityFileAccessStat(fd int, stat *unix.Stat_t) error {
	if stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Mode&0o077 != 0 ||
		stat.Mode&0o400 == 0 ||
		stat.Uid != uint32(os.Geteuid()) {
		return ErrInsecureStorage
	}
	hasACL, err := hasExtendedACL(fd, false)
	if err != nil || hasACL {
		return ErrInsecureStorage
	}
	return nil
}

func (d *secureDirectory) close() {
	if d != nil && d.fd >= 0 {
		_ = unix.Close(d.fd)
		d.fd = -1
	}
}

func (d *secureDirectory) lockExclusive(ctx context.Context) error {
	const retryInterval = 5 * time.Millisecond
	for {
		err := unix.Flock(d.fd, unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			return fmt.Errorf("lock agent identity directory: %w", err)
		}
		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("lock agent identity directory: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

func (d *secureDirectory) ensurePathStillBound() error {
	reopened, err := openValidatedDirectoryChain(d.path)
	if err != nil {
		return ErrInsecureStorage
	}
	defer reopened.close()
	if reopened.dev != d.dev || reopened.ino != d.ino {
		return ErrInsecureStorage
	}
	return nil
}

func sameUnixFile(left unix.Stat_t, right unix.Stat_t) bool {
	return left.Dev == right.Dev && left.Ino == right.Ino
}

func (d *secureDirectory) entryStat(name string) (unix.Stat_t, bool, error) {
	var stat unix.Stat_t
	err := unix.Fstatat(d.fd, name, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if errors.Is(err, unix.ENOENT) {
		return unix.Stat_t{}, false, nil
	}
	if err != nil {
		return unix.Stat_t{}, false, fmt.Errorf("inspect agent identity entry: %w", err)
	}
	return stat, true, nil
}

func (d *secureDirectory) readEntry(name string) ([]byte, error) {
	fd, err := unix.Openat(d.fd, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil, ErrIdentityNotFound
	}
	if err != nil {
		return nil, ErrInsecureStorage
	}
	file := os.NewFile(uintptr(fd), "agent-identity")
	if file == nil {
		_ = unix.Close(fd)
		return nil, ErrInvalidIdentity
	}
	defer file.Close()

	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return nil, fmt.Errorf("inspect opened agent identity: %w", err)
	}
	if err := validateIdentityFileStat(fd, &stat); err != nil {
		return nil, err
	}
	encoded, err := io.ReadAll(io.LimitReader(file, maxIdentityFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("read agent identity: %w", err)
	}
	if len(encoded) == 0 || len(encoded) > maxIdentityFileSize {
		return nil, ErrInvalidIdentity
	}
	return encoded, nil
}

func (d *secureDirectory) writePending(name string, encoded []byte) error {
	fd, err := unix.Openat(
		d.fd,
		name,
		unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0o600,
	)
	if errors.Is(err, unix.EEXIST) {
		return ErrIdentityExists
	}
	if err != nil {
		return fmt.Errorf("create pending agent identity: %w", err)
	}
	defer unix.Close(fd)
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return fmt.Errorf("inspect pending agent identity: %w", err)
	}
	if err := validateIdentityFileAccessStat(fd, &stat); err != nil {
		return err
	}

	for remaining := encoded; len(remaining) > 0; {
		written, err := unix.Write(fd, remaining)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return fmt.Errorf("write pending agent identity: %w", err)
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		remaining = remaining[written:]
	}
	if err := unix.Fsync(fd); err != nil {
		return fmt.Errorf("sync pending agent identity: %w", err)
	}
	return nil
}

func (d *secureDirectory) publish(pendingName string, identityName string) error {
	if err := unix.Linkat(d.fd, pendingName, d.fd, identityName, 0); errors.Is(err, unix.EEXIST) {
		return ErrIdentityExists
	} else if err != nil {
		return fmt.Errorf("publish agent identity: %w", err)
	}
	return d.sync()
}

func (d *secureDirectory) remove(name string) error {
	if err := unix.Unlinkat(d.fd, name, 0); errors.Is(err, unix.ENOENT) {
		return nil
	} else if err != nil {
		return fmt.Errorf("remove pending agent identity: %w", err)
	}
	return nil
}

func (d *secureDirectory) sync() error {
	if err := unix.Fsync(d.fd); err != nil {
		return fmt.Errorf("sync agent identity directory: %w", err)
	}
	return nil
}
