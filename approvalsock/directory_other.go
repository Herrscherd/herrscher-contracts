//go:build !unix

package approvalsock

import (
	"errors"
	"fmt"
	"os"
)

func checkPrivateDirectory(string) (os.FileInfo, error) {
	return nil, fmt.Errorf("approval socket needs a unix platform: %w", errors.ErrUnsupported)
}
