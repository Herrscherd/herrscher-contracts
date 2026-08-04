package contracts

import (
	"fmt"
	"sort"
	"strings"
)

// These three functions carry the environment variables a host injects into a
// backend's child process at spawn time.
//
// They live in contracts, not in each backend, because the host ENCODES and
// the backends DECODE: splitting the two halves across repos would let them
// drift apart without any test suite able to see it.
//
// The transport goes through PluginConfig.Settings, which is a
// map[string]string. There is no typed channel down to the plugin, and adding
// one would force a signature change on every factory.

// MergeEnv overlays extra onto base, in the "K=V" format of os.Environ(). An
// injected key REPLACES the inherited one rather than being appended
// alongside it: an exec.Cmd with two entries for the same variable has
// platform-dependent behavior, and a stale value inherited from the daemon
// would silently hijack the session.
//
// An empty extra returns base unchanged — that is the native route's
// non-regression case.
func MergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	out := make([]string, 0, len(base)+len(extra))
	for _, e := range base {
		if k, _, found := strings.Cut(e, "="); found {
			if _, override := extra[k]; override {
				continue
			}
		}
		out = append(out, e)
	}
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, k+"="+extra[k])
	}
	return out
}

// ParseEnvSetting decodes the value of the "env" setting: K=V pairs separated
// by newlines. Only the FIRST '=' separates the key from the value — tokens
// are often base64-encoded and contain one.
func ParseEnvSetting(s string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, found := strings.Cut(line, "=")
		if !found || k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

// EncodeEnvSetting is the inverse of ParseEnvSetting. Keys are sorted so the
// value is deterministic: without that, two identical spawns would carry
// different strings, and the function would not be testable.
func EncodeEnvSetting(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, env[k])
	}
	return b.String()
}
