package contracts

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"
)

// Errors returned by ValidateModels and NewGatewayCreds, matched with errors.Is.
var (
	ErrModelEmptyID        = errors.New("empty ID")
	ErrModelDuplicateID    = errors.New("duplicate ID")
	ErrModelEmptyLabel     = errors.New("empty label")
	ErrModelEmptyArg       = errors.New("empty argument")
	ErrModelControlChar    = errors.New("control character")
	ErrModelUnknownRoute   = errors.New("unknown route")
	ErrModelNegative       = errors.New("negative number")
	ErrModelNotFinite      = errors.New("number is not finite")
	ErrCredsMissing        = errors.New("gateway credentials: neither base URL nor token available")
	ErrCredsTokenNoURL     = errors.New("gateway credentials: token without a base URL")
	ErrCredsURLNoToken     = errors.New("gateway credentials: base URL without a token, refusing to run on the user's own subscription")
	ErrCredsURLControlChar = errors.New("gateway credentials: base URL contains a control character")
	ErrCredsTokenControl   = errors.New("gateway credentials: token contains a control character")
)

func hasControlChar(s string) bool {
	return strings.ContainsFunc(s, unicode.IsControl)
}

// Route says who serves a model. It is the only notion of "provider" the
// system carries, and it is binary: either the vendor CLI talks to its own
// service with the login present on the machine, or it talks to the Neublox
// gateway, which routes to the real upstream server-side. herrscher therefore
// never needs to know about Z.ai, Alibaba, or DeepSeek.
type Route string

const (
	// RouteNative is the route on which the CLI uses the user's own authentication.
	RouteNative Route = "native"
	// RouteGateway is the route on which the CLI is pointed at the Neublox gateway.
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
func (p RoutePolicy) Allows(route Route) bool {
	if p == PolicyGatewayOnly {
		return route == RouteGateway
	}
	return true
}

// ModelSpec is a model a backend declares it knows how to run. It lives in the
// Manifest. So it is readable without instantiating a backend, which the app
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
	// OutputPrice is the price in USD per million output tokens. Same rules as
	// InputPrice: for a gateway route it is OUR price, 0 = unknown.
	OutputPrice float64
	// CacheReadPrice is the price in USD per million tokens read from the
	// prompt cache. Same rules as InputPrice, 0 = unknown.
	CacheReadPrice float64
	// CacheWritePrice is the price in USD per million tokens written to the
	// prompt cache. Same rules as InputPrice, 0 = unknown.
	CacheWritePrice float64
	// ContextWindow is the size of the model's context window, in tokens. 0 =
	// unknown.
	ContextWindow int
	// Hidden keeps the model out of the selector without removing it: a retired
	// or historical model. Its ID still resolves a session or agent recorded
	// with it.
	Hidden bool
}

// ValidateModels checks the integrity of a backend's catalog. kind names the
// backend in error messages. Called by the host at startup: an inconsistent
// catalog must kill the daemon with a clear message, not silently produce a
// wrong selector. It returns the first failing model as an error wrapping
// [ErrModelEmptyID], [ErrModelDuplicateID], [ErrModelEmptyLabel],
// [ErrModelEmptyArg], [ErrModelControlChar], [ErrModelUnknownRoute],
// [ErrModelNegative] or [ErrModelNotFinite], to match with errors.Is.
func ValidateModels(kind string, models []ModelSpec) error {
	seen := make(map[string]bool, len(models))
	for index := range models {
		spec := &models[index]
		if err := validateModelIdentity(kind, index, spec, seen); err != nil {
			return err
		}
		if err := validateModelRequiredFields(kind, spec); err != nil {
			return err
		}
		if err := validateModelControlChars(kind, spec); err != nil {
			return err
		}
		if err := validateModelRoute(kind, spec); err != nil {
			return err
		}
		if err := validateModelNumbers(kind, spec); err != nil {
			return err
		}
	}
	return nil
}

func validateModelNumbers(kind string, spec *ModelSpec) error {
	for _, price := range []struct {
		name  string
		value float64
	}{
		{"InputPrice", spec.InputPrice},
		{"OutputPrice", spec.OutputPrice},
		{"CacheReadPrice", spec.CacheReadPrice},
		{"CacheWritePrice", spec.CacheWritePrice},
	} {
		if err := validateModelPrice(kind, spec.ID, price.name, price.value); err != nil {
			return err
		}
	}
	if spec.ContextWindow < 0 {
		return fmt.Errorf("backend %q: model %q, field ContextWindow: %w", kind, spec.ID, ErrModelNegative)
	}
	return nil
}

func validateModelPrice(kind, modelID, name string, price float64) error {
	if math.IsNaN(price) || math.IsInf(price, 0) {
		return fmt.Errorf("backend %q: model %q, field %s: %w", kind, modelID, name, ErrModelNotFinite)
	}
	if price < 0 {
		return fmt.Errorf("backend %q: model %q, field %s: %w", kind, modelID, name, ErrModelNegative)
	}
	return nil
}

func validateModelIdentity(kind string, index int, spec *ModelSpec, seen map[string]bool) error {
	if strings.TrimSpace(spec.ID) == "" {
		return fmt.Errorf("backend %q: model #%d: %w", kind, index, ErrModelEmptyID)
	}
	if seen[spec.ID] {
		return fmt.Errorf("backend %q: model %q: %w", kind, spec.ID, ErrModelDuplicateID)
	}
	seen[spec.ID] = true
	return nil
}

func validateModelRequiredFields(kind string, spec *ModelSpec) error {
	if strings.TrimSpace(spec.Label) == "" {
		return fmt.Errorf("backend %q: model %q: %w", kind, spec.ID, ErrModelEmptyLabel)
	}
	if strings.TrimSpace(spec.Arg) == "" {
		return fmt.Errorf("backend %q: model %q: %w", kind, spec.ID, ErrModelEmptyArg)
	}
	return nil
}

func validateModelControlChars(kind string, spec *ModelSpec) error {
	for _, field := range []struct{ name, value string }{
		{"ID", spec.ID}, {"Label", spec.Label}, {"Arg", spec.Arg},
	} {
		if hasControlChar(field.value) {
			return fmt.Errorf("backend %q: model %q, field %s: %w", kind, spec.ID, field.name, ErrModelControlChar)
		}
	}
	for _, effort := range spec.Efforts {
		if hasControlChar(effort) {
			return fmt.Errorf("backend %q: model %q, effort %q: %w", kind, spec.ID, effort, ErrModelControlChar)
		}
	}
	return nil
}

func validateModelRoute(kind string, spec *ModelSpec) error {
	if spec.Route != RouteNative && spec.Route != RouteGateway {
		return fmt.Errorf("backend %q: model %q, route %q: %w", kind, spec.ID, spec.Route, ErrModelUnknownRoute)
	}
	return nil
}

// FilterModels keeps the models the policy allows. The result is always
// non-nil so callers can serialize it to JSON without producing `null`.
func FilterModels(models []ModelSpec, policy RoutePolicy) []ModelSpec {
	out := make([]ModelSpec, 0, len(models))
	for index := range models {
		if policy.Allows(models[index].Route) {
			out = append(out, models[index])
		}
	}
	return out
}

// GatewayCreds is the (base URL, token) pair a gateway route requires. Its
// fields are unexported, so no package outside this one can construct a
// half-filled value. NewGatewayCreds is the only path to one.
//
// The reason is legal, not aesthetic. A base URL with no token sends traffic
// to the gateway while billing it against the user's own claude.ai
// subscription. The shape Anthropic explicitly forbids third-party
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

// NewGatewayCreds builds a complete pair, or fails. It returns [ErrCredsMissing]
// when both values are blank, [ErrCredsTokenNoURL] when only the token is set,
// [ErrCredsURLNoToken] when only the base URL is set, and
// [ErrCredsURLControlChar] or [ErrCredsTokenControl] when a value contains a
// control character.
func NewGatewayCreds(baseURL, token string) (GatewayCreds, error) {
	baseURL, token = strings.TrimSpace(baseURL), strings.TrimSpace(token)
	switch {
	case baseURL == "" && token == "":
		return GatewayCreds{}, ErrCredsMissing
	case baseURL == "":
		return GatewayCreds{}, ErrCredsTokenNoURL
	case token == "":
		return GatewayCreds{}, ErrCredsURLNoToken
	case hasControlChar(baseURL):
		return GatewayCreds{}, ErrCredsURLControlChar
	case hasControlChar(token):
		return GatewayCreds{}, ErrCredsTokenControl
	}
	return GatewayCreds{baseURL: baseURL, token: token}, nil
}
