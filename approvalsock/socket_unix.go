//go:build unix

package approvalsock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func lockSocket(ctx context.Context, path string) (func() error, error) {
	file, err := openLock(path)
	if err != nil {
		return nil, err
	}
	if err = acquireLock(ctx, file); err != nil {
		return nil, errors.Join(err, file.Close())
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

func acquireLock(ctx context.Context, file *os.File) error {
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EINTR) {
			return fmt.Errorf("lock approval socket: %w", err)
		}
		if !pause(ctx, lockRetryDelay) {
			return fmt.Errorf("lock approval socket: %w", ctx.Err())
		}
	}
}
