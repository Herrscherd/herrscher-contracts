package contracts

import (
	"context"
	"testing"
)

func TestRegistryFiltersByCategory(t *testing.T) {
	var r Registry
	if len(r.Plugins()) != 0 {
		t.Fatalf("fresh registry should be empty")
	}
	r.Register(Plugin{
		Manifest: Manifest{Kind: "discord", Category: CategoryGateway},
		Gateway:  func(context.Context, PluginConfig) (Gateway, error) { return nil, nil },
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
