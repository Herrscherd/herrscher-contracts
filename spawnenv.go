package contracts

import (
	"fmt"
	"sort"
	"strings"
)

// The environment variable NAMES that carry a gateway route from the host
// down to a vendor CLI. They live here, next to MergeEnv and the round-trip
// test, for exactly the same reason those functions do: the host WRITES these
// keys and a backend (or the vendor CLI itself) READS them, and nothing else
// ties the two sides together. Spelled as literals on both sides, a rename is
// green in every repo and only fails at run time. And for claude it fails
// SILENTLY: an unrecognised base-URL variable makes the child run natively, on
// the machine's own subscription, while the session is still marked gateway.
//
// Anything that writes or matches one of these names must use these constants.
const (
	// EnvAnthropicBaseURL redirects the claude CLI at the gateway. Alone, it
	// is the forbidden shape: gateway traffic billed to the user's own login.
	// It travels with EnvAnthropicAPIKey or not at all.
	EnvAnthropicBaseURL = "ANTHROPIC_BASE_URL"
	// EnvAnthropicAPIKey rather than ANTHROPIC_AUTH_TOKEN: it is the first slot in
	// the credential resolution order, so it reliably takes precedence over a
	// subscription OAuth session. A long-lived API key is also the right meaning
	// for what Neublox issues.
	//
	// It is a REPLACEMENT. Setting both variables sends x-api-key and Authorization
	// together, and the API rejects the request with a 401.
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
// An empty extra returns base unchanged. That is the native route's
// non-regression case.
func MergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	out := make([]string, 0, len(base)+len(extra))
	for _, entry := range base {
		if k, _, found := strings.Cut(entry, "="); found {
			if _, override := extra[k]; override {
				continue
			}
		}
		out = append(out, entry)
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
// by newlines. Only the FIRST '=' separates the key from the value. Tokens
// are often base64-encoded and contain one.
func ParseEnvSetting(setting string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(setting, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || key == "" {
			continue
		}
		out[key] = value
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
	var builder strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&builder, "%s=%s\n", k, env[k])
	}
	return builder.String()
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

const approvalsModeBypass = "bypass"

// ApprovalsEnv encodes a session's gate for a backend child process. It returns
// nil when there is nothing to carry: the mode is bypass, or the mode, the session
// or the bin is empty. A variable set to a value meaning "ignore me" is a variable
// that will eventually be misread.
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

// ApprovalsFromEnv decodes what ApprovalsEnv wrote. Gated is false unless all
// three are present and the mode is not bypass, so a partial set fails open, in
// the same direction every other approval failure does.
func ApprovalsFromEnv(look func(string) string) ApprovalsGate {
	if look == nil {
		return ApprovalsGate{}
	}
	mode := strings.TrimSpace(look(EnvApprovalsMode))
	session := strings.TrimSpace(look(EnvApprovalsSession))
	bin := strings.TrimSpace(look(EnvApprovalsBin))
	return ApprovalsGate{
		Session: session,
		Mode:    mode,
		Bin:     bin,
		Gated:   mode != "" && mode != approvalsModeBypass && session != "" && bin != "",
	}
}

// ApprovalsGate is the approvals route a spawn decoded from its environment.
type ApprovalsGate struct {
	Session string
	Mode    string
	Bin     string
	Gated   bool
}
