package contracts

import (
	"errors"
	"math"
	"testing"
)

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
		{RoutePolicy(""), RouteNative, true},
	}
	for _, c := range cases {
		if got := c.policy.Allows(c.route); got != c.want {
			t.Errorf("RoutePolicy(%q).Allows(%q) = %v, want %v", c.policy, c.route, got, c.want)
		}
	}
}

func TestValidateModelsRejects(t *testing.T) {
	cases := []struct {
		name    string
		models  []ModelSpec
		wantErr error
	}{
		{"empty ID", []ModelSpec{{Label: "No id", Arg: "x", Route: RouteNative}}, ErrModelEmptyID},
		{"empty label", []ModelSpec{{ID: "a", Arg: "a", Route: RouteNative}}, ErrModelEmptyLabel},
		{"empty arg", []ModelSpec{{ID: "a", Label: "A", Route: RouteNative}}, ErrModelEmptyArg},
		{"control character", []ModelSpec{{ID: "a", Label: "A\n", Arg: "a", Route: RouteNative}}, ErrModelControlChar},
		{"duplicate ID", []ModelSpec{
			{ID: "a", Label: "A", Arg: "a", Route: RouteNative},
			{ID: "a", Label: "A bis", Arg: "a2", Route: RouteGateway},
		}, ErrModelDuplicateID},
		{"unknown route", []ModelSpec{{ID: "a", Label: "A", Arg: "a", Route: Route("carrier-pigeon")}}, ErrModelUnknownRoute},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateModels("claude", c.models)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("ValidateModels error = %v, want %v", err, c.wantErr)
			}
		})
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

func TestGatewayCredsRefuses(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		token   string
		wantErr error
	}{
		{"nothing", "", "", ErrCredsMissing},
		{"base URL without token", "https://gw.example", "", ErrCredsURLNoToken},
		{"blank token", "https://gw.example", "   ", ErrCredsURLNoToken},
		{"token without base URL", "", "tok", ErrCredsTokenNoURL},
		{"blank base URL", "  ", "tok", ErrCredsTokenNoURL},
		{"control character in base URL", "https://gw\x07.example", "tok", ErrCredsURLControlChar},
		{"control character in token", "https://gw.example", "to\nk", ErrCredsTokenControl},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewGatewayCreds(c.baseURL, c.token)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("NewGatewayCreds error = %v, want %v", err, c.wantErr)
			}
		})
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
				t.Fatal("NewGatewayCreds accepted a control character. It would forge a variable in the child")
			}
		})
	}
}

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

func TestValidateModelsRejectsNegativeNumbers(t *testing.T) {
	cases := []struct {
		name string
		spec ModelSpec
	}{
		{"negative InputPrice", ModelSpec{ID: "x", Label: "X", Arg: "x", Route: RouteNative, InputPrice: -1}},
		{"negative OutputPrice", ModelSpec{ID: "x", Label: "X", Arg: "x", Route: RouteNative, OutputPrice: -1}},
		{"negative CacheReadPrice", ModelSpec{ID: "x", Label: "X", Arg: "x", Route: RouteNative, CacheReadPrice: -1}},
		{"negative CacheWritePrice", ModelSpec{ID: "x", Label: "X", Arg: "x", Route: RouteNative, CacheWritePrice: -1}},
		{"negative ContextWindow", ModelSpec{ID: "x", Label: "X", Arg: "x", Route: RouteNative, ContextWindow: -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateModels("k", []ModelSpec{tc.spec})
			if !errors.Is(err, ErrModelNegative) {
				t.Fatalf("ValidateModels error = %v, want %v", err, ErrModelNegative)
			}
		})
	}
}

func TestValidateModelsRejectsNotFinitePrices(t *testing.T) {
	cases := []struct {
		name string
		spec ModelSpec
	}{
		{"NaN InputPrice", ModelSpec{ID: "x", Label: "X", Arg: "x", Route: RouteNative, InputPrice: math.NaN()}},
		{"NaN OutputPrice", ModelSpec{ID: "x", Label: "X", Arg: "x", Route: RouteNative, OutputPrice: math.NaN()}},
		{"NaN CacheReadPrice", ModelSpec{ID: "x", Label: "X", Arg: "x", Route: RouteNative, CacheReadPrice: math.NaN()}},
		{"NaN CacheWritePrice", ModelSpec{ID: "x", Label: "X", Arg: "x", Route: RouteNative, CacheWritePrice: math.NaN()}},
		{"infinite OutputPrice", ModelSpec{ID: "x", Label: "X", Arg: "x", Route: RouteNative, OutputPrice: math.Inf(1)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateModels("k", []ModelSpec{tc.spec})
			if !errors.Is(err, ErrModelNotFinite) {
				t.Fatalf("ValidateModels error = %v, want %v", err, ErrModelNotFinite)
			}
		})
	}
}

func TestValidateModelsAcceptsKnownNumbersAndZeroUnknown(t *testing.T) {
	models := []ModelSpec{
		{ID: "priced", Label: "Priced", Arg: "priced", Route: RouteNative, InputPrice: 3, OutputPrice: 15, CacheReadPrice: 0.3, CacheWritePrice: 3.75, ContextWindow: 200000},
		{ID: "unknown", Label: "Unknown", Arg: "unknown", Route: RouteNative},
	}
	if err := ValidateModels("k", models); err != nil {
		t.Fatalf("ValidateModels rejected known numbers or unknown zeros: %v", err)
	}
}

func TestValidateModelsAcceptsHiddenModel(t *testing.T) {
	models := []ModelSpec{
		{ID: "live", Label: "Live", Arg: "live", Route: RouteNative},
		{ID: "retired", Label: "Retired", Arg: "retired", Route: RouteNative, Hidden: true},
	}
	if err := ValidateModels("k", models); err != nil {
		t.Fatalf("ValidateModels rejected a hidden model: %v", err)
	}
}

func TestValidateModelsRefusesHiddenModelSharingAnID(t *testing.T) {
	models := []ModelSpec{
		{ID: "a", Label: "A", Arg: "a", Route: RouteNative},
		{ID: "a", Label: "A retired", Arg: "a2", Route: RouteNative, Hidden: true},
	}
	if err := ValidateModels("k", models); !errors.Is(err, ErrModelDuplicateID) {
		t.Fatalf("ValidateModels error = %v, want %v", err, ErrModelDuplicateID)
	}
}

func TestFilterModelsKeepsHiddenModelForResolution(t *testing.T) {
	models := []ModelSpec{
		{ID: "live", Label: "Live", Arg: "live", Route: RouteNative},
		{ID: "retired", Label: "Retired", Arg: "retired", Route: RouteNative, Hidden: true},
	}
	got := FilterModels(models, PolicyAll)
	if len(got) != 2 || got[1].ID != "retired" {
		t.Fatalf("FilterModels dropped the hidden model needed to resume it: %+v", got)
	}
}

func TestSelectableModelsExcludesHiddenModel(t *testing.T) {
	models := []ModelSpec{
		{ID: "live", Label: "Live", Arg: "live", Route: RouteNative},
		{ID: "retired", Label: "Retired", Arg: "retired", Route: RouteNative, Hidden: true},
	}
	got := SelectableModels(models, PolicyAll)
	if len(got) != 1 || got[0].ID != "live" {
		t.Fatalf("SelectableModels = %+v, want only the visible model", got)
	}
}

func TestSelectableModelsStillAppliesPolicy(t *testing.T) {
	models := []ModelSpec{
		{ID: "native", Label: "Native", Arg: "native", Route: RouteNative},
		{ID: "gateway", Label: "Gateway", Arg: "gateway", Route: RouteGateway},
		{ID: "gateway-hidden", Label: "Old", Arg: "old", Route: RouteGateway, Hidden: true},
	}
	got := SelectableModels(models, PolicyGatewayOnly)
	if len(got) != 1 || got[0].ID != "gateway" {
		t.Fatalf("SelectableModels(gateway-only) = %+v, want only the visible gateway model", got)
	}
}

func TestSelectableModelsReturnsEmptyNotNil(t *testing.T) {
	models := []ModelSpec{{ID: "retired", Label: "Retired", Arg: "retired", Route: RouteNative, Hidden: true}}
	if got := SelectableModels(models, PolicyAll); got == nil || len(got) != 0 {
		t.Fatalf("SelectableModels = %#v, want a non-nil empty slice", got)
	}
}
