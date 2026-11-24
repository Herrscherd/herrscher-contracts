//go:build unix

package approvalsock_test

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"syscall"
	"testing"

	contracts "github.com/Herrscherd/herrscher-contracts"
	"github.com/Herrscherd/herrscher-contracts/approvalsock"
)

func TestAskReportsAnAnswerResetWithoutSentinel(t *testing.T) {
	t.Parallel()
	path := resetServer(t)
	_, err := approvalsock.Ask(t.Context(), path, contracts.ApprovalRequest{ID: "15"})
	if err == nil {
		t.Fatal("Ask accepted a connection reset without an answer")
	}
	if !errors.Is(err, syscall.ECONNRESET) && !errors.Is(err, syscall.EPIPE) {
		t.Fatalf("Ask error = %v, want syscall.ECONNRESET or syscall.EPIPE for a connection reset without an answer", err)
	}
}

func resetServer(t *testing.T) string {
	t.Helper()
	path := filepath.Join(privateDir(t), "reset.sock")
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "unix", path)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	served := make(chan error, 1)
	go func() { served <- resetOnce(listener) }()
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close reset listener: %v", err)
		}
		if err := <-served; err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("reset server: %v", err)
		}
	})
	return path
}

func resetOnce(listener net.Listener) error {
	connection, err := listener.Accept()
	if err != nil {
		return err
	}
	unixConnection, ok := connection.(*net.UnixConn)
	if !ok {
		return fmt.Errorf("accepted %T, want *net.UnixConn", connection)
	}
	if err := setLingerZero(unixConnection); err != nil {
		return err
	}
	return unixConnection.Close()
}

func setLingerZero(connection *net.UnixConn) error {
	rawConnection, err := connection.SyscallConn()
	if err != nil {
		return err
	}
	var sockErr error
	if err := rawConnection.Control(func(fd uintptr) {
		sockErr = syscall.SetsockoptLinger(int(fd), syscall.SOL_SOCKET, syscall.SO_LINGER, &syscall.Linger{Onoff: 1, Linger: 0})
	}); err != nil {
		return err
	}
	return sockErr
}
