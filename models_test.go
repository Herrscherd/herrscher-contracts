package contracts

import "testing"

func TestRoutePolicyAllows(t *testing.T) {
	cases := []struct {
		policy RoutePolicy
		route  Route
		want   bool
	}{
		{PolicyAll, RouteNative, true},
		{PolicyAll, RouteGateway, true},
		{PolicyGatewayOnly, RouteGateway, true},
		{PolicyGatewayOnly, RouteNative, false},
		// The zero value must be permissive: a host that configures nothing
		// behaves as it did before this change.
		{RoutePolicy(""), RouteNative, true},
	}
	for _, c := range cases {
		if got := c.policy.Allows(c.route); got != c.want {
			t.Errorf("RoutePolicy(%q).Allows(%q) = %v, want %v", c.policy, c.route, got, c.want)
		}
	}
}

func TestValidateModelsRejectsEmptyID(t *testing.T) {
	err := ValidateModels("claude", []ModelSpec{{Label: "No id", Arg: "x", Route: RouteNative}})
	if err == nil {
		t.Fatal("ValidateModels accepted a model with an empty ID")
	}
}

func TestValidateModelsRejectsEmptyLabel(t *testing.T) {
	err := ValidateModels("claude", []ModelSpec{{ID: "a", Arg: "a", Route: RouteNative}})
	if err == nil {
		t.Fatal("ValidateModels accepted a model with an empty Label")
	}
}

func TestValidateModelsRejectsEmptyArg(t *testing.T) {
	err := ValidateModels("claude", []ModelSpec{{ID: "a", Label: "A", Route: RouteNative}})
	if err == nil {
		t.Fatal("ValidateModels accepted a model with an empty Arg")
	}
}

func TestValidateModelsRejectsDuplicateID(t *testing.T) {
	models := []ModelSpec{
		{ID: "a", Label: "A", Arg: "a", Route: RouteNative},
		{ID: "a", Label: "A bis", Arg: "a2", Route: RouteGateway},
	}
	if err := ValidateModels("claude", models); err == nil {
		t.Fatal("ValidateModels accepted a duplicate ID")
	}
}

func TestValidateModelsRejectsUnknownRoute(t *testing.T) {
	err := ValidateModels("claude", []ModelSpec{{ID: "a", Label: "A", Arg: "a", Route: Route("carrier-pigeon")}})
	if err == nil {
		t.Fatal("ValidateModels accepted an unknown route")
	}
}

func TestValidateModelsAcceptsValid(t *testing.T) {
	models := []ModelSpec{
		{ID: "a", Label: "A", Arg: "a", Route: RouteNative},
		{ID: "b", Label: "B", Arg: "b", Route: RouteGateway},
	}
	if err := ValidateModels("claude", models); err != nil {
		t.Fatalf("ValidateModels rejected a valid set: %v", err)
	}
}

func TestFilterModelsGatewayOnly(t *testing.T) {
	models := []ModelSpec{
		{ID: "n", Label: "N", Arg: "n", Route: RouteNative},
		{ID: "g", Label: "G", Arg: "g", Route: RouteGateway},
	}
	got := FilterModels(models, PolicyGatewayOnly)
	if len(got) != 1 || got[0].ID != "g" {
		t.Fatalf("FilterModels(gateway-only) = %+v, want only the gateway model", got)
	}
}

func TestFilterModelsAllKeepsEverything(t *testing.T) {
	models := []ModelSpec{
		{ID: "n", Label: "N", Arg: "n", Route: RouteNative},
		{ID: "g", Label: "G", Arg: "g", Route: RouteGateway},
	}
	if got := FilterModels(models, PolicyAll); len(got) != 2 {
		t.Fatalf("FilterModels(all) dropped models: %+v", got)
	}
}

// Serialized, a nil slice becomes `null` and an empty slice becomes `[]`. The
// `models list --json` CLI must always emit `[]`, never `null` — this is the
// boundary where the non-nil guarantee matters, so it is the one to test.
func TestFilterModelsReturnsEmptyNotNil(t *testing.T) {
	models := []ModelSpec{{ID: "n", Label: "N", Arg: "n", Route: RouteNative}}
	got := FilterModels(models, PolicyGatewayOnly)
	if got == nil {
		t.Fatal("FilterModels returned nil when everything was filtered out; want an empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("FilterModels(gateway-only) kept a native model: %+v", got)
	}
}

func TestGatewayCredsRefusesPartial(t *testing.T) {
	if _, err := NewGatewayCreds("https://gw.example", ""); err == nil {
		t.Fatal("NewGatewayCreds accepted a base URL without a token — the forbidden shape")
	}
	if _, err := NewGatewayCreds("", "tok"); err == nil {
		t.Fatal("NewGatewayCreds accepted a token without a base URL")
	}
	if _, err := NewGatewayCreds("  ", "tok"); err == nil {
		t.Fatal("NewGatewayCreds accepted a blank base URL")
	}
	// A whitespace-only token is the dangerous variant: it looks present to any
	// naive `!= ""` check, yet it does not authenticate.
	if _, err := NewGatewayCreds("https://gw.example", "   "); err == nil {
		t.Fatal("NewGatewayCreds accepted a blank token — the forbidden shape in disguise")
	}
}

func TestGatewayCredsAcceptsPair(t *testing.T) {
	c, err := NewGatewayCreds("https://gw.example", "tok")
	if err != nil {
		t.Fatalf("NewGatewayCreds rejected a complete pair: %v", err)
	}
	if c.BaseURL() != "https://gw.example" || c.Token() != "tok" {
		t.Fatalf("NewGatewayCreds mangled its inputs: %q %q", c.BaseURL(), c.Token())
	}
}

// The zero value is constructible outside the package
// (`contracts.GatewayCreds{}`) but it is empty on BOTH sides, so never a
// half-pair.
func TestGatewayCredsZeroValueIsEmptyOnBothSides(t *testing.T) {
	var c GatewayCreds
	if c.BaseURL() != "" || c.Token() != "" {
		t.Fatalf("zero GatewayCreds carried data: %q %q", c.BaseURL(), c.Token())
	}
}

func TestManifestCarriesModels(t *testing.T) {
	m := Manifest{Kind: "claude", Category: CategoryBackend, Models: []ModelSpec{
		{ID: "x", Label: "X", Arg: "x", Route: RouteNative},
	}}
	if len(m.Models) != 1 || m.Models[0].ID != "x" {
		t.Fatalf("Manifest.Models did not round-trip: %+v", m.Models)
	}
}

// A newline in either half is not a cosmetic problem: the value travels to the
// child through the newline-delimited "env" setting, where an embedded newline
// forges a second variable (and truncates the token). Reject it at the only
// constructor, next to the blank guard.
func TestGatewayCredsRejectsControlCharacters(t *testing.T) {
	cases := []struct {
		name           string
		baseURL, token string
	}{
		{"newline in token", "https://gw.example", "tok\nANTHROPIC_BASE_URL=http://evil"},
		{"newline in base URL", "https://gw.example\nX=1", "tok"},
		{"carriage return in token", "https://gw.example", "tok\rmore"},
		{"NUL in token", "https://gw.example", "tok\x00more"},
		{"escape in base URL", "https://gw.example\x1b[2J", "tok"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewGatewayCreds(tc.baseURL, tc.token); err == nil {
				t.Fatal("NewGatewayCreds accepted a control character — it would forge a variable in the child")
			}
		})
	}
}

// A malformed catalog must fail at registration, not at spawn: Arg and Efforts
// reach the child's argv, ID is persisted, and any of them reaching the "env"
// transport with a newline forges a second variable.
func TestValidateModelsRejectsControlCharacters(t *testing.T) {
	cases := []struct {
		name string
		spec ModelSpec
	}{
		{"newline in Arg", ModelSpec{ID: "x", Label: "X", Arg: "x\nOPENAI_BASE_URL=http://evil", Route: RouteNative}},
		{"newline in ID", ModelSpec{ID: "x\ny", Label: "X", Arg: "x", Route: RouteNative}},
		{"newline in Label", ModelSpec{ID: "x", Label: "X\ny", Arg: "x", Route: RouteNative}},
		{"newline in an effort", ModelSpec{ID: "x", Label: "X", Arg: "x", Efforts: []string{"low\nhigh"}, Route: RouteNative}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateModels("k", []ModelSpec{tc.spec}); err == nil {
				t.Fatal("ValidateModels accepted a control character in a field that reaches the child")
			}
		})
	}
}
