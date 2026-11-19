package approvalsock

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	lockSuffix     = ".lock"
	lockMode       = 0o600
	lockRetryDelay = 10 * time.Millisecond
)

func bindSocket(ctx context.Context, path string, directory os.FileInfo) (net.Listener, error) {
	release, err := lockSocket(ctx, path)
	if err != nil {
		return nil, err
	}
	listener, err := bindLocked(ctx, path, directory)
	if err != nil {
		return nil, errors.Join(err, release())
	}
	if releaseErr := release(); releaseErr != nil {
		return nil, errors.Join(releaseErr, listener.Close())
	}
	return listener, nil
}

func bindLocked(ctx context.Context, path string, directory os.FileInfo) (net.Listener, error) {
	err := clearStaleSocket(ctx, path)
	if err != nil {
		return nil, err
	}
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on approval socket: %w", err)
	}
	err = checkSameDirectory(filepath.Dir(path), directory)
	if err != nil {
		return nil, errors.Join(err, listener.Close())
	}
	return listener, nil
}

func clearStaleSocket(ctx context.Context, path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect approval socket path: %w", err)
	}
	if info.Mode().Type() != fs.ModeSocket {
		return fmt.Errorf("%w: %s is not a socket", ErrPathInUse, path)
	}
	var dialer net.Dialer
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	connection, err := dialer.DialContext(probeCtx, "unix", path)
	if err == nil {
		return errors.Join(fmt.Errorf("%w: a live socket answers at %s", ErrPathInUse, path), connectionError(OpClose, connection.Close()))
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("probe approval socket: %w", ctxErr)
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		return fmt.Errorf("%w: probe %s: %w", ErrPathInUse, path, err)
	}
	if err = os.Remove(path); err != nil {
		return fmt.Errorf("remove stale approval socket: %w", err)
	}
	return nil
}

func checkSameDirectory(dir string, before os.FileInfo) error {
	after, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat approval socket directory: %w", err)
	}
	if !os.SameFile(before, after) {
		return fmt.Errorf("%w: %s", ErrDirectoryChanged, dir)
	}
	return nil
}

func nextAcceptDelay(delay time.Duration) time.Duration {
	if delay == 0 {
		return minAcceptDelay
	}
	return min(delay*backoffFactor, maxAcceptDelay)
}

func pause(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
