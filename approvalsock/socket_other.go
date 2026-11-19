//go:build !unix

package approvalsock

import (
	"errors"
	"fmt"
)

func lockSocket(string) (func() error, error) {
	return nil, fmt.Errorf("approval socket needs a unix platform: %w", errors.ErrUnsupported)
}

func restrictSocket(string) error {
	return fmt.Errorf("approval socket needs a unix platform: %w", errors.ErrUnsupported)
}
