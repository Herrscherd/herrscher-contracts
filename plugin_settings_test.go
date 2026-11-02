package contracts

import (
	"errors"
	"slices"
	"testing"
)

func TestResolveBindsEnvAndDefaults(t *testing.T) {
	env := map[string]string{"DISCORD_BOT_TOKEN": "abc"}
	getenv := func(k string) string { return env[k] }

	cfg, err := Resolve([]Setting{
		{Key: "token", Env: "DISCORD_BOT_TOKEN", Required: true},
		{Key: "channel", Env: "DISCORD_CHANNEL_ID", Required: false},
		{Key: "stream", Env: "DCTL_STREAM", Default: "true"},
	}, getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Get("token") != "abc" {
		t.Fatalf("token = %q", cfg.Get("token"))
	}
	if cfg.Get("channel") != "" {
		t.Fatalf("absent optional must be empty, got %q", cfg.Get("channel"))
	}
	if cfg.Get("stream") != "true" {
		t.Fatalf("default not applied: %q", cfg.Get("stream"))
	}
}

func TestResolveMissingRequiredFails(t *testing.T) {
	_, err := Resolve([]Setting{
		{Key: "token", Env: "DISCORD_BOT_TOKEN", Required: true},
		{Key: "other", Env: "OTHER", Required: true},
	}, func(string) string { return "" })
	missing, ok := errors.AsType[*MissingSettingsError](err)
	if !ok {
		t.Fatalf("error %v is not a *MissingSettingsError", err)
	}
	want := []string{"token (set DISCORD_BOT_TOKEN)", "other (set OTHER)"}
	if !slices.Equal(missing.Missing, want) {
		t.Fatalf("Missing = %q, want %q", missing.Missing, want)
	}
	if _, err := Resolve([]Setting{
		{Key: "k", Env: "K", Required: true, Default: "d"},
	}, func(string) string { return "" }); err != nil {
		t.Fatalf("default should satisfy required: %v", err)
	}
}
