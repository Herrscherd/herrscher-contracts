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
		// Le zéro value doit être permissif : un host qui ne configure rien se
		// comporte comme avant ce changement.
		{RoutePolicy(""), RouteNative, true},
	}
	for _, c := range cases {
		if got := c.policy.Allows(c.route); got != c.want {
			t.Errorf("RoutePolicy(%q).Allows(%q) = %v, want %v", c.policy, c.route, got, c.want)
		}
	}
}

func TestValidateModelsRejectsEmptyID(t *testing.T) {
	err := ValidateModels("claude", []ModelSpec{{Label: "Sans id", Arg: "x", Route: RouteNative}})
	if err == nil {
		t.Fatal("ValidateModels accepted a model with an empty ID")
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
}

func TestGatewayCredsAcceptsPair(t *testing.T) {
	c, err := NewGatewayCreds("https://gw.example", "tok")
	if err != nil {
		t.Fatalf("NewGatewayCreds rejected a complete pair: %v", err)
	}
	if c.BaseURL != "https://gw.example" || c.Token != "tok" {
		t.Fatalf("NewGatewayCreds mangled its inputs: %+v", c)
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
