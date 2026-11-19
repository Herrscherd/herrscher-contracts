//go:build unix

package approvalsock

import (
	"fmt"
	"os"
	"syscall"
)

const privateDirectoryMode = 0o700

func checkPrivateDirectory(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat approval socket directory: %w", err)
	}
	owner, isUnix := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || !isUnix || info.Mode().Perm() != privateDirectoryMode || int(owner.Uid) != os.Getuid() {
		return fmt.Errorf("%w: %s", ErrInsecureDirectory, dir)
	}
	return nil
}
