//go:build unix

package approvalsock

import (
	"fmt"
	"os"
	"syscall"
)

const privateDirectoryMode = 0o700

func checkPrivateDirectory(dir string) (os.FileInfo, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("stat approval socket directory: %w", err)
	}
	owner, isUnix := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || !isUnix || info.Mode().Perm() != privateDirectoryMode || int(owner.Uid) != os.Getuid() {
		return nil, fmt.Errorf("%w: %s", ErrInsecureDirectory, dir)
	}
	return info, nil
}
