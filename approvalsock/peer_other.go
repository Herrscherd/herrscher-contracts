//go:build !linux

package approvalsock

import "net"

func checkPeer(net.Conn) error {
	return nil
}
