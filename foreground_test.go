package contracts

import "context"

var _ Foreground = fgStub{}

type fgStub struct{}

func (fgStub) RunForeground(context.Context, context.CancelFunc) error { return nil }
