package contracts

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestRegistryFiltersByCategory(t *testing.T) {
	var r Registry
	if len(r.Plugins()) != 0 {
		t.Fatalf("fresh registry should be empty")
	}
	r.Register(Plugin{
		Manifest: Manifest{Kind: "discord", Category: CategoryGateway},
		Gateway:  func(context.Context, PluginConfig) (GatewaySet, error) { return GatewaySet{}, nil },
	})
	r.Register(Plugin{
		Manifest: Manifest{Kind: "claude", Category: CategoryBackend},
		Backend:  func(context.Context, PluginConfig) (Backend, error) { return nil, nil },
	})

	if got := r.Gateways(); len(got) != 1 || got[0].Manifest.Kind != "discord" {
		t.Fatalf("Gateways() did not isolate the gateway plugin: %+v", got)
	}
	if got := r.Backends(); len(got) != 1 || got[0].Manifest.Kind != "claude" {
		t.Fatalf("Backends() did not isolate the backend plugin: %+v", got)
	}
}

func TestDefaultRegistryRegister(t *testing.T) {
	before := len(Default.Plugins())
	Register(Plugin{Manifest: Manifest{Kind: "x", Category: CategoryGateway}})
	if len(Default.Plugins()) != before+1 {
		t.Fatalf("Register did not append to Default")
	}
}

func TestPluginConfigGet(t *testing.T) {
	var zero PluginConfig
	if zero.Get("missing") != "" {
		t.Fatalf("nil-map Get should be empty")
	}
	c := PluginConfig{Settings: map[string]string{"token": "abc"}}
	if c.Get("token") != "abc" {
		t.Fatalf("Get returned wrong value")
	}
}

func TestRegistryIsolatesMemory(t *testing.T) {
	var r Registry
	r.Register(Plugin{
		Manifest: Manifest{Kind: "obsidian", Category: CategoryMemory},
		Memory:   func(context.Context, PluginConfig) (Memory, error) { return nil, nil },
	})
	r.Register(Plugin{
		Manifest: Manifest{Kind: "claude", Category: CategoryBackend},
		Backend:  func(context.Context, PluginConfig) (Backend, error) { return nil, nil },
	})

	got := r.Memories()
	if len(got) != 1 || got[0].Manifest.Kind != "obsidian" {
		t.Fatalf("Memories() did not isolate the memory plugin: %+v", got)
	}
	if len(r.Backends()) != 1 {
		t.Fatalf("Backends() should still see exactly one backend")
	}
}

// A plugin may contribute skills and nothing else. The category is what makes it
// findable and countable next to the gateways and the backends, and the absence
// of a port factory is what makes it legal.
func TestRegistryIsolatesSkills(t *testing.T) {
	var r Registry
	r.Register(Plugin{
		Manifest: Manifest{Kind: "superset", Category: CategorySkills},
		Skills: func(context.Context, PluginConfig) (fs.FS, error) {
			return fstest.MapFS{"demo/SKILL.md": &fstest.MapFile{Data: []byte("# demo")}}, nil
		},
	})
	r.Register(Plugin{
		Manifest: Manifest{Kind: "claude", Category: CategoryBackend},
		Backend:  func(context.Context, PluginConfig) (Backend, error) { return nil, nil },
	})

	got := r.Skills()
	if len(got) != 1 || got[0].Manifest.Kind != "superset" {
		t.Fatalf("Skills() did not isolate the skills plugin: %+v", got)
	}
	if got[0].Gateway != nil || got[0].Backend != nil || got[0].Memory != nil || got[0].Orchestrator != nil {
		t.Error("a skills plugin sets no port factory")
	}
	if len(r.Backends()) != 1 {
		t.Fatalf("Backends() should still see exactly one backend")
	}
}

// Skills are orthogonal to the category: a gateway carries the playbook that
// teaches an agent to use it, and stays a gateway.
func TestAGatewayMayAlsoCarrySkills(t *testing.T) {
	var r Registry
	r.Register(Plugin{
		Manifest: Manifest{Kind: "chat", Category: CategoryGateway},
		Gateway:  func(context.Context, PluginConfig) (GatewaySet, error) { return GatewaySet{}, nil },
		Skills: func(context.Context, PluginConfig) (fs.FS, error) {
			return fstest.MapFS{"chat/SKILL.md": &fstest.MapFile{Data: []byte("# chat")}}, nil
		},
	})
	if len(r.Skills()) != 0 {
		t.Error("Skills() reports the skills category, not every plugin that carries a playbook")
	}
	gws := r.Gateways()
	if len(gws) != 1 || gws[0].Skills == nil {
		t.Fatalf("a gateway carrying skills is still a gateway that carries skills: %+v", gws)
	}
}
