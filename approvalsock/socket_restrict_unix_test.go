//go:build unix

package approvalsock

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestRestrictSocketRefusesRegularFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "a.sock")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := restrictSocket(path); !errors.Is(err, ErrSocketReplaced) {
		t.Fatalf("restrictSocket error = %v, want ErrSocketReplaced", err)
	}
}

func TestRestrictSocketNeverFollowsSymlink(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatalf("chmod target: %v", err)
	}
	path := filepath.Join(dir, "a.sock")
	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := restrictSocket(path); !errors.Is(err, ErrSocketReplaced) {
		t.Fatalf("restrictSocket error = %v, want ErrSocketReplaced", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("symlink target mode = %v, want 0644 untouched", info.Mode().Perm())
	}
}

func TestRestrictSocketSetsOwnerOnly(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "a.sock")
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := listener.Close(); closeErr != nil {
			t.Errorf("close listener: %v", closeErr)
		}
	})
	err = restrictSocket(path)
	if err != nil {
		t.Fatalf("restrictSocket: %v", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if info.Mode().Perm() != socketMode {
		t.Fatalf("socket mode = %v, want %v", info.Mode().Perm(), os.FileMode(socketMode))
	}
}
