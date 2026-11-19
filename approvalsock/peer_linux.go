package approvalsock

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

func checkPeer(connection net.Conn) error {
	unixConnection, isUnix := connection.(*net.UnixConn)
	if !isUnix {
		return fmt.Errorf("%w: connection is not a unix socket", ErrForeignPeer)
	}
	raw, err := unixConnection.SyscallConn()
	if err != nil {
		return fmt.Errorf("read approval socket peer: %w", err)
	}
	var credentials *syscall.Ucred
	var credErr error
	if err = raw.Control(func(fd uintptr) {
		credentials, credErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return fmt.Errorf("read approval socket peer: %w", err)
	}
	if credErr != nil {
		return fmt.Errorf("read approval socket peer credentials: %w", credErr)
	}
	if int(credentials.Uid) != os.Getuid() {
		return fmt.Errorf("%w: uid %d", ErrForeignPeer, credentials.Uid)
	}
	return nil
}
