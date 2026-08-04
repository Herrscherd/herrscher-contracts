package contracts

import (
	"fmt"
	"strings"
)

// Route says who serves a model. It is the only notion of "provider" the
// system carries, and it is binary: either the vendor CLI talks to its own
// service with the login present on the machine, or it talks to the Neublox
// gateway, which routes to the real upstream server-side. herrscher therefore
// never needs to know about Z.ai, Alibaba, or DeepSeek.
type Route string

const (
	// RouteNative: the CLI uses the user's own authentication.
	RouteNative Route = "native"
	// RouteGateway: the CLI is pointed at the Neublox gateway.
	RouteGateway Route = "gateway"
)

// RoutePolicy bounds the routes a host is willing to serve. The app's public
// build sets PolicyGatewayOnly: a native model is not hidden there, it is
// absent from the catalog, so it cannot be selected, persisted, or resumed.
type RoutePolicy string

const (
	// PolicyAll is the zero value: a host that configures nothing behaves as
	// it did before this change. The internal build stays on this value.
	PolicyAll RoutePolicy = ""
	// PolicyGatewayOnly exposes only models served by the gateway.
	PolicyGatewayOnly RoutePolicy = "gateway-only"
)

// Allows reports whether the policy accepts this route.
func (p RoutePolicy) Allows(r Route) bool {
	if p == PolicyGatewayOnly {
		return r == RouteGateway
	}
	return true
}

// ModelSpec is a model a backend declares it knows how to run. It lives in the
// Manifest — so it is readable without instantiating a backend, which the app
// needs in order to populate its selector before a session exists.
type ModelSpec struct {
	// ID is the stable identifier persisted in session state. It, not Label
	// or Arg, is what lets a resume recover the route.
	ID string
	// Label is what the user sees. No notion of vendor, provider, or
	// protocol should appear in it.
	Label string
	// Arg is the value passed to --model on the CLI. It often differs from
	// the ID (cursor encodes effort into it, the gateway renames it).
	Arg string
	// Efforts lists the accepted levels. Empty means no separate effort axis.
	Efforts []string
	Route   Route
	// InputPrice is the price shown in USD per million input tokens. For a
	// gateway route this is OUR price, not the upstream's. 0 = unknown, and
	// the cost is then displayed in tokens alone.
	InputPrice float64
}

// ValidateModels checks the integrity of a backend's catalog. kind names the
// backend in error messages. Called by the host at startup: an inconsistent
// catalog must kill the daemon with a clear message, not silently produce a
// wrong selector.
func ValidateModels(kind string, models []ModelSpec) error {
	seen := make(map[string]bool, len(models))
	for i, m := range models {
		if strings.TrimSpace(m.ID) == "" {
			return fmt.Errorf("backend %q: model #%d has an empty ID", kind, i)
		}
		if seen[m.ID] {
			return fmt.Errorf("backend %q: duplicate model ID %q", kind, m.ID)
		}
		seen[m.ID] = true
		if strings.TrimSpace(m.Label) == "" {
			return fmt.Errorf("backend %q: model %q has an empty Label", kind, m.ID)
		}
		if strings.TrimSpace(m.Arg) == "" {
			return fmt.Errorf("backend %q: model %q has an empty Arg", kind, m.ID)
		}
		if m.Route != RouteNative && m.Route != RouteGateway {
			return fmt.Errorf("backend %q: model %q has unknown route %q", kind, m.ID, m.Route)
		}
	}
	return nil
}

// FilterModels keeps the models the policy allows. The result is always
// non-nil so callers can serialize it to JSON without producing `null`.
func FilterModels(models []ModelSpec, p RoutePolicy) []ModelSpec {
	out := make([]ModelSpec, 0, len(models))
	for _, m := range models {
		if p.Allows(m.Route) {
			out = append(out, m)
		}
	}
	return out
}

// GatewayCreds is the (base URL, token) pair a gateway route requires. Its
// fields are unexported, so no package outside this one can construct a
// half-filled value — NewGatewayCreds is the only path to one.
//
// The reason is legal, not aesthetic. A base URL with no token sends traffic
// to the gateway while billing it against the user's own claude.ai
// subscription — the shape Anthropic explicitly forbids third-party
// developers from producing. Making that state unrepresentable is better than
// testing for it after the fact.
//
// The zero value stays constructible outside the package
// (`contracts.GatewayCreds{}`) but it is empty on BOTH sides, so it is never a
// half-pair.
type GatewayCreds struct {
	baseURL string
	token   string
}

// BaseURL is the gateway URL. Empty only for the zero value.
func (c GatewayCreds) BaseURL() string { return c.baseURL }

// Token is the authentication token. Empty only for the zero value.
func (c GatewayCreds) Token() string { return c.token }

// NewGatewayCreds builds a complete pair, or fails.
func NewGatewayCreds(baseURL, token string) (GatewayCreds, error) {
	baseURL, token = strings.TrimSpace(baseURL), strings.TrimSpace(token)
	switch {
	case baseURL == "" && token == "":
		return GatewayCreds{}, fmt.Errorf("gateway credentials: neither base URL nor token available")
	case baseURL == "":
		return GatewayCreds{}, fmt.Errorf("gateway credentials: token without a base URL")
	case token == "":
		return GatewayCreds{}, fmt.Errorf("gateway credentials: base URL without a token — refusing to run on the user's own subscription")
	}
	return GatewayCreds{baseURL: baseURL, token: token}, nil
}
