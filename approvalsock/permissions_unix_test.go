//go:build unix

package approvalsock_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/Herrscherd/herrscher-contracts/approvalsock"
)

func TestServeSocketIsOwnerOnlyUnderOpenUmask(t *testing.T) {
	previous := syscall.Umask(0)
	t.Cleanup(func() { syscall.Umask(previous) })
	path := filepath.Join(privateDir(t), "a.sock")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- approvalsock.Serve(ctx, path, allowAll, logErrors(t)) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Serve after cancel: %v", err)
		}
	})
	waitForServer(t, path)
	waitForSocketMode(t, path, fs.FileMode(0o600))
}

func waitForSocketMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	var got fs.FileMode
	for range 200 {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat socket: %v", err)
		}
		got = info.Mode().Perm()
		if got == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("socket mode = %o, want %o", got, want)
}
