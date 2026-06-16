package contracts

import (
	"context"
	"testing"
)

func TestBuilderProducesCmd(t *testing.T) {
	c := New("session", "create").
		Help("start a session").
		Param("name", "session name", true).
		Param("shared", "use main checkout", false).
		Do(func(ctx context.Context, in Input) (string, error) {
			return "ok " + in.Get("name"), nil
		})

	if got := c.Path; len(got) != 2 || got[0] != "session" || got[1] != "create" {
		t.Fatalf("path = %v, want [session create]", got)
	}
	if c.Help != "start a session" {
		t.Fatalf("help = %q", c.Help)
	}
	if len(c.Params) != 2 || c.Params[0].Name != "name" || !c.Params[0].Required {
		t.Fatalf("params = %+v", c.Params)
	}
	if c.Params[1].Required {
		t.Fatal("shared must be optional")
	}
	out, err := c.Run(context.Background(), Input{Args: map[string]string{"name": "x"}})
	if err != nil || out != "ok x" {
		t.Fatalf("run = %q, %v", out, err)
	}
}

func TestInputAccessors(t *testing.T) {
	in := Input{Args: map[string]string{"name": "x", "flag": "true"}, Rest: []string{"a"}}
	if v, ok := in.Lookup("name"); !ok || v != "x" {
		t.Fatalf("Lookup name = %q,%v", v, ok)
	}
	if _, ok := in.Lookup("missing"); ok {
		t.Fatal("missing must report ok=false")
	}
	if in.Get("name") != "x" {
		t.Fatal("Get")
	}
	if !in.Bool("flag") || in.Bool("name") {
		t.Fatal("Bool: flag true, name not a bool")
	}
	if len(in.Rest) != 1 || in.Rest[0] != "a" {
		t.Fatal("Rest")
	}
}
