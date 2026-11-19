//go:build unix

package approvalsock_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/Herrscherd/herrscher-contracts/approvalsock"
)

func TestServeReturnsAtOnceWhenLockHeldAndContextEnds(t *testing.T) {
	t.Parallel()
	path := filepath.Join(privateDir(t), "a.sock")
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("open lock: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := lock.Close(); closeErr != nil {
			t.Errorf("close lock: %v", closeErr)
		}
	})
	err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX)
	if err != nil {
		t.Fatalf("hold lock: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	err = approvalsock.Serve(ctx, path, allowAll, logErrors(t))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Serve error = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("Serve took %v to return under a held lock", elapsed)
	}
	if _, statErr := os.Lstat(path); statErr == nil {
		t.Fatal("Serve bound a socket without holding the lock")
	}
}
