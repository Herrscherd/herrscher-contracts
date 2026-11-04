package contracts

import (
	"errors"
	"testing"
)

func TestMergeDiscoveredKeepsStaticEntryForKnownID(t *testing.T) {
	static := []ModelSpec{{ID: "m-1", Label: "Model One", Arg: "m-1", Route: RouteNative, InputPrice: 5}}
	discovered := []ModelSpec{{ID: "m-1", Label: "renamed by vendor", Arg: "m-1", Route: RouteNative}}

	merged, err := MergeDiscovered("v", static, discovered)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || merged[0].Label != "Model One" || merged[0].InputPrice != 5 {
		t.Fatalf("static entry not kept: %+v", merged)
	}
}

func TestMergeDiscoveredDropsSameArgOnSameRoute(t *testing.T) {
	static := []ModelSpec{{ID: "claude-haiku-4-5", Label: "Haiku 4.5", Arg: "claude-haiku-4-5-20251001", Route: RouteNative}}
	discovered := []ModelSpec{{ID: "claude-haiku-4-5-20251001", Label: "Haiku 4.5", Arg: "claude-haiku-4-5-20251001", Route: RouteNative}}

	merged, err := MergeDiscovered("v", static, discovered)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 {
		t.Fatalf("same model listed twice: %+v", merged)
	}
}

func TestMergeDiscoveredNewModelHasDerivedLabelAndNoPrice(t *testing.T) {
	discovered := []ModelSpec{
		{ID: "new-with-label", Label: "New One", Arg: "new-with-label", Efforts: []string{"low"}, Route: RouteNative, InputPrice: 7},
		{ID: "new-without-label", Arg: "new-without-label", Route: RouteNative},
	}

	merged, err := MergeDiscovered("v", nil, discovered)
	if err != nil {
		t.Fatal(err)
	}
	if merged[0].Label != "New One" || merged[1].Label != "new-without-label" {
		t.Fatalf("labels: %+v", merged)
	}
	for _, spec := range merged {
		if spec.InputPrice != 0 {
			t.Fatalf("new model %q carries a price %v", spec.ID, spec.InputPrice)
		}
	}
	if len(merged[0].Efforts) != 1 || merged[0].Efforts[0] != "low" {
		t.Fatalf("vendor efforts lost: %+v", merged[0])
	}
}

func TestMergeDiscoveredRejectsInvalidListAsAWhole(t *testing.T) {
	static := []ModelSpec{{ID: "kept", Label: "Kept", Arg: "kept", Route: RouteNative}}
	discovered := []ModelSpec{
		{ID: "ok", Label: "OK", Arg: "ok", Route: RouteNative},
		{ID: "bad", Label: "Bad", Arg: "bad", Route: "tunnel"},
	}

	merged, err := MergeDiscovered("v", static, discovered)
	if !errors.Is(err, ErrModelUnknownRoute) {
		t.Fatalf("err = %v, want ErrModelUnknownRoute", err)
	}
	if merged != nil {
		t.Fatalf("partial merge returned: %+v", merged)
	}
}

func TestMergeDiscoveredRejectsMalformedEntryWithStaticID(t *testing.T) {
	static := []ModelSpec{{ID: "kept", Label: "Kept", Arg: "kept", Route: RouteNative}}
	discovered := []ModelSpec{{ID: "kept", Label: "Kept", Arg: "kept", Route: "tunnel"}}

	merged, err := MergeDiscovered("v", static, discovered)
	if !errors.Is(err, ErrModelUnknownRoute) {
		t.Fatalf("err = %v, want ErrModelUnknownRoute", err)
	}
	if merged != nil {
		t.Fatalf("malformed entry silently accepted: %+v", merged)
	}
}

func TestGatewayOnlyDropsNativeDiscoveredModel(t *testing.T) {
	static := []ModelSpec{{ID: "gw-x", Label: "X", Arg: "x", Route: RouteGateway}}
	discovered := []ModelSpec{
		{ID: "native-y", Label: "Y", Arg: "native-y", Route: RouteNative},
		{ID: "gw-z", Label: "Z", Arg: "z", Route: RouteGateway},
	}

	merged, err := MergeDiscovered("v", static, discovered)
	if err != nil {
		t.Fatal(err)
	}
	kept := FilterModels(merged, PolicyGatewayOnly)
	if len(kept) != 2 || kept[0].ID != "gw-x" || kept[1].ID != "gw-z" {
		t.Fatalf("gateway-only kept %+v", kept)
	}
}
