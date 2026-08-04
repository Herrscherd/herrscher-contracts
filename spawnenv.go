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

// The environment variable NAMES that carry a gateway route from the host
// down to a vendor CLI. They live here, next to MergeEnv and the round-trip
// test, for exactly the same reason those functions do: the host WRITES these
// keys and a backend (or the vendor CLI itself) READS them, and nothing else
// ties the two sides together. Spelled as literals on both sides, a rename is
// green in every repo and only fails at run time — and for claude it fails
// SILENTLY: an unrecognised base-URL variable makes the child run natively, on
// the machine's own subscription, while the session is still marked gateway.
//
// Anything that writes or matches one of these names must use these constants.
const (
	// EnvAnthropicBaseURL redirects the claude CLI at the gateway. Alone, it
	// is the forbidden shape: gateway traffic billed to the user's own login.
	// It travels with EnvAnthropicAuthToken or not at all.
	EnvAnthropicBaseURL = "ANTHROPIC_BASE_URL"
	// EnvAnthropicAuthToken is the gateway credential. It takes precedence
	// over the subscription OAuth in the claude CLI's own precedence order,
	// which is what guarantees the session runs on the product's account.
	EnvAnthropicAuthToken = "ANTHROPIC_AUTH_TOKEN"
	// EnvOpenAIBaseURL is the codex counterpart of EnvAnthropicBaseURL. The
	// codex backend also treats its presence as the signal that this spawn is
	// on the gateway route and needs a generated CODEX_HOME.
	EnvOpenAIBaseURL = "OPENAI_BASE_URL"
	// EnvNeubloxToken is the gateway credential for codex. It is referenced by
	// env_key from the generated config.toml rather than written into it, so
	// the token never lands on disk.
	EnvNeubloxToken = "NEUBLOX_TOKEN"
)

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
