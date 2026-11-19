//go:build !unix

package approvalsock

import (
	"errors"
	"fmt"
)

func checkPrivateDirectory(string) error {
	return fmt.Errorf("approval socket needs a unix platform: %w", errors.ErrUnsupported)
}
