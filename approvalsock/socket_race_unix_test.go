//go:build unix

package approvalsock_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Herrscherd/herrscher-contracts/approvalsock"
)

func TestServeRacingStartsLetOneWin(t *testing.T) {
	for range 20 {
		path := filepath.Join(privateDir(t), "race.sock")
		ctx, cancel := context.WithCancel(t.Context())
		results := make(chan error, 2)
		start := make(chan struct{})
		for range 2 {
			go func() {
				<-start
				results <- approvalsock.Serve(ctx, path, allowAll, logErrors(t))
			}()
		}
		close(start)
		var first error
		select {
		case first = <-results:
		case <-time.After(5 * time.Second):
			cancel()
			t.Fatal("neither racing Serve returned an error")
		}
		if !errors.Is(first, approvalsock.ErrPathInUse) {
			cancel()
			t.Fatalf("first Serve error = %v, want ErrPathInUse", first)
		}
		waitForServer(t, path)
		cancel()
		if second := <-results; second != nil {
			t.Fatalf("winning Serve returned %v after cancel", second)
		}
	}
}
