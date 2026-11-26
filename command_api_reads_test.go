package contracts

import (
	"context"
	"testing"
)

func TestReadsDeclaresACommandReadOnly(t *testing.T) {
	run := func(context.Context, Input) (Result, error) { return Result{}, nil }
	if New("x", "list").Reads().Do(run).ReadOnly != true {
		t.Fatal("Reads did not mark the command read only")
	}
	if New("x", "add").Do(run).ReadOnly {
		t.Fatal("a command is read only without saying so")
	}
}
