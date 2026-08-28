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
	// It travels with EnvAnthropicAPIKey or not at all.
	EnvAnthropicBaseURL = "ANTHROPIC_BASE_URL"
	// EnvAnthropicAPIKey plutôt qu'ANTHROPIC_AUTH_TOKEN : c'est le premier slot
	// de l'ordre de résolution des identifiants, donc celui qui prime le plus
	// sûrement sur une session OAuth d'abonnement — et « clé API longue durée »
	// est la bonne sémantique pour ce que Neublox émet.
	//
	// C'est un REMPLACEMENT. Poser les deux variables fait envoyer x-api-key et
	// Authorization ensemble, et l'API rejette la requête en 401.
	EnvAnthropicAPIKey = "ANTHROPIC_API_KEY"
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

// The environment variable NAMES that carry a session's approval policy from
// the host down to a backend that gates in-process. They live here for the same
// reason the gateway route names do: the host WRITES them and a backend READS
// them, and nothing else ties the two sides together.
const (
	// EnvApprovalsMode carries the session's approval mode: "ask" or "strict".
	// Absent or empty means ungated, which is how herrscher behaved before
	// approvals existed.
	EnvApprovalsMode = "HERRSCHER_APPROVALS_MODE"
	// EnvApprovalsSession is the session name a backend passes back when it
	// asks, without which there is nothing to ask about. It deliberately
	// reuses HERRSCHER_SESSION, which the supervisor already exports on every
	// bridge spawn: a second variable holding the same string could only ever
	// disagree with the first one.
	EnvApprovalsSession = "HERRSCHER_SESSION"
	// EnvApprovalsBin is the herrscher binary a backend asks with. It is the
	// same trusted binary the materialized hook invokes, and it is passed
	// rather than resolved from PATH so a session cannot be gated by a
	// different herrscher than the one supervising it.
	EnvApprovalsBin = "HERRSCHER_APPROVALS_BIN"
)

// approvalsModeBypass is the one mode that means "do not gate". It is spelled
// here rather than imported because contracts depends on nothing.
const approvalsModeBypass = "bypass"

// ApprovalsEnv encodes a session's gate for a backend child process. A bypass
// or empty mode returns nil: there is nothing to carry, and a variable set to a
// value meaning "ignore me" is a variable that will eventually be misread.
func ApprovalsEnv(session, mode, bin string) map[string]string {
	if session == "" || bin == "" || mode == "" || mode == approvalsModeBypass {
		return nil
	}
	return map[string]string{
		EnvApprovalsMode:    mode,
		EnvApprovalsSession: session,
		EnvApprovalsBin:     bin,
	}
}

// ApprovalsFromEnv decodes what ApprovalsEnv wrote. gated is false unless all
// three are present and the mode is not bypass, so a partial set fails open, in
// the same direction every other approval failure does.
func ApprovalsFromEnv(look func(string) string) (session, mode, bin string, gated bool) {
	if look == nil {
		return "", "", "", false
	}
	mode = strings.TrimSpace(look(EnvApprovalsMode))
	session = strings.TrimSpace(look(EnvApprovalsSession))
	bin = strings.TrimSpace(look(EnvApprovalsBin))
	gated = mode != "" && mode != approvalsModeBypass && session != "" && bin != ""
	return session, mode, bin, gated
}
