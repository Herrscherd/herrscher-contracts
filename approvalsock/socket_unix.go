//go:build unix

package approvalsock

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

func lockSocket(path string) (func() error, error) {
	file, err := openLock(path)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return nil, errors.Join(fmt.Errorf("lock approval socket: %w", err), file.Close())
	}
	return func() error {
		unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		if unlockErr != nil {
			unlockErr = fmt.Errorf("unlock approval socket: %w", unlockErr)
		}
		return errors.Join(unlockErr, file.Close())
	}, nil
}

func openLock(path string) (*os.File, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("open approval socket directory: %w", err)
	}
	file, openErr := root.OpenFile(filepath.Base(path)+lockSuffix, os.O_CREATE|os.O_RDWR, lockMode)
	if joined := errors.Join(openErr, root.Close()); joined != nil {
		return nil, fmt.Errorf("open approval socket lock: %w", joined)
	}
	return file, nil
}

func restrictSocket(path string) error {
	before, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect approval socket: %w", err)
	}
	if before.Mode().Type() != fs.ModeSocket {
		return fmt.Errorf("%w: %s is not a socket", ErrDirectoryChanged, path)
	}
	if err = os.Chmod(path, socketMode); err != nil {
		return fmt.Errorf("restrict approval socket: %w", err)
	}
	after, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect approval socket: %w", err)
	}
	if !os.SameFile(before, after) {
		return fmt.Errorf("%w: %s was replaced while restricted", ErrDirectoryChanged, path)
	}
	return nil
}
