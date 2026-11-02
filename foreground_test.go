package contracts

import (
	"context"
	"testing"
)

func TestForegroundIsOptionalCapability(_ *testing.T) {
	var _ Foreground = fgStub{}
}

type fgStub struct{}

func (fgStub) RunForeground(context.Context, context.CancelFunc) error { return nil }
