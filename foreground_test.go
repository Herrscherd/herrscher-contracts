package contracts

import (
	"context"
	"testing"
)

func TestForegroundIsOptionalCapability(t *testing.T) {
	// A type implementing Foreground must satisfy the interface; the compiler
	// proves the method set. This test documents the contract shape.
	var _ Foreground = fgStub{}
}

type fgStub struct{}

func (fgStub) RunForeground(context.Context, context.CancelFunc) error { return nil }
