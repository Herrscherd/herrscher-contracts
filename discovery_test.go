package contracts

import (
	"errors"
	"math"
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

func TestMergeDiscoveredDropsDuplicateAmongDiscovered(t *testing.T) {
	discovered := []ModelSpec{
		{ID: "dup-a", Label: "A", Arg: "same-arg", Route: RouteNative},
		{ID: "dup-b", Label: "B", Arg: "same-arg", Route: RouteNative},
	}

	merged, err := MergeDiscovered("v", nil, discovered)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || merged[0].ID != "dup-a" {
		t.Fatalf("same route and arg listed twice: %+v", merged)
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

func TestMergeDiscoveredNewModelKeepsContextWindowButNoPrices(t *testing.T) {
	discovered := []ModelSpec{
		{ID: "new", Arg: "new", Route: RouteNative, ContextWindow: 200000, InputPrice: 1, OutputPrice: 2, CacheReadPrice: 3, CacheWritePrice: 4},
	}

	merged, err := MergeDiscovered("v", nil, discovered)
	if err != nil {
		t.Fatal(err)
	}
	got := merged[0]
	if got.ContextWindow != 200000 {
		t.Fatalf("ContextWindow = %d, want 200000 from the vendor listing", got.ContextWindow)
	}
	if got.InputPrice != 0 || got.OutputPrice != 0 || got.CacheReadPrice != 0 || got.CacheWritePrice != 0 {
		t.Fatalf("discovered model carries a price: %+v", got)
	}
}

func TestMergeDiscoveredKeepsHiddenStaticEntryWhenStillListed(t *testing.T) {
	static := []ModelSpec{{ID: "old", Label: "Old", Arg: "old", Route: RouteNative, Hidden: true}}
	discovered := []ModelSpec{{ID: "old", Arg: "old", Route: RouteNative}}

	merged, err := MergeDiscovered("v", static, discovered)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || !merged[0].Hidden {
		t.Fatalf("hidden static model came back visible: %+v", merged)
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

func TestMergeDiscoveredTakesContextWindowForStaticModel(t *testing.T) {
	static := []ModelSpec{{ID: "claude-x", Label: "Claude X", Arg: "claude-x", Route: RouteNative, InputPrice: 5}}
	discovered := []ModelSpec{{ID: "claude-x", Arg: "claude-x", Route: RouteNative, ContextWindow: 200000, OutputPrice: 9}}

	merged, err := MergeDiscovered("claude", static, discovered)
	if err != nil {
		t.Fatal(err)
	}
	got := merged[0]
	if got.ContextWindow != 200000 {
		t.Fatalf("ContextWindow = %d, want 200000 from the discovered listing", got.ContextWindow)
	}
	if got.Label != "Claude X" || got.InputPrice != 5 || got.OutputPrice != 0 {
		t.Fatalf("discovery changed more than the window: %+v", got)
	}
}

func TestMergeDiscoveredKeepsStaticContextWindowWhenDiscoveredUnknown(t *testing.T) {
	static := []ModelSpec{{ID: "claude-x", Label: "Claude X", Arg: "claude-x", Route: RouteNative, ContextWindow: 100000}}
	discovered := []ModelSpec{{ID: "claude-x", Arg: "claude-x", Route: RouteNative}}

	merged, err := MergeDiscovered("claude", static, discovered)
	if err != nil {
		t.Fatal(err)
	}
	if merged[0].ContextWindow != 100000 {
		t.Fatalf("ContextWindow = %d, want the static 100000 kept when discovery says nothing", merged[0].ContextWindow)
	}
}

func TestMergeDiscoveredIgnoresInvalidDiscardedPrices(t *testing.T) {
	discovered := []ModelSpec{
		{ID: "bad-price", Arg: "bad-price", Route: RouteNative, OutputPrice: math.NaN(), CacheReadPrice: -1},
		{ID: "good", Arg: "good", Route: RouteNative, ContextWindow: 8000},
	}

	merged, err := MergeDiscovered("v", nil, discovered)
	if err != nil {
		t.Fatalf("MergeDiscovered rejected the list over prices it discards: %v", err)
	}
	if len(merged) != 2 || merged[0].ID != "bad-price" || merged[1].ID != "good" {
		t.Fatalf("MergeDiscovered = %+v, want both models", merged)
	}
}
