package contracts

import (
	"context"
	"testing"
)

// capStub prouve que Locator et Deleter sont implémentables.
type capStub struct{}

func (capStub) Locate(context.Context, string) (Location, error) { return Location{}, nil }
func (capStub) Delete(context.Context, string) error             { return nil }

var (
	_ Locator = capStub{}
	_ Deleter = capStub{}
)

func TestLocationZeroValue(t *testing.T) {
	var l Location
	if l.Obsidian != "" || l.File != "" {
		t.Fatalf("zero Location should be empty, got %+v", l)
	}
}
