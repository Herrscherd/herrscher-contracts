//go:build !unix

package approvalsock

import (
	"context"
	"errors"
	"fmt"
)

func lockSocket(context.Context, string) (func() error, error) {
	return nil, fmt.Errorf("approval socket needs a unix platform: %w", errors.ErrUnsupported)
}

func restrictSocket(string) error {
	return fmt.Errorf("approval socket needs a unix platform: %w", errors.ErrUnsupported)
}
