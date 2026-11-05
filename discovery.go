package contracts

import (
	"context"
	"fmt"
)

// DiscoveryAccess is what the host hands a discovery function: the gateway pair it holds, the route policy it enforces and the environment reader.
type DiscoveryAccess struct {
	Gateway GatewayCreds
	Policy  RoutePolicy
	Getenv  func(string) string
}

// ModelDiscoverer lists the models a vendor serves now; no listing on this machine means no models and no error.
type ModelDiscoverer func(ctx context.Context, access DiscoveryAccess) ([]ModelSpec, error)

// MergeDiscovered merges discovered models into a static catalog, keeping static entries and validating both lists.
func MergeDiscovered(kind string, static, discovered []ModelSpec) ([]ModelSpec, error) {
	named := prepareDiscovered(discovered)
	if err := ValidateModels(kind, named); err != nil {
		return nil, fmt.Errorf("discovered catalog: %w", err)
	}
	merged := append([]ModelSpec{}, static...)
	for index := range named {
		position, listed := listedPosition(merged, named[index])
		if !listed {
			merged = append(merged, named[index])
			continue
		}
		if named[index].ContextWindow != 0 {
			merged[position].ContextWindow = named[index].ContextWindow
		}
	}
	if err := ValidateModels(kind, merged); err != nil {
		return nil, fmt.Errorf("merged catalog: %w", err)
	}
	return merged, nil
}

func prepareDiscovered(discovered []ModelSpec) []ModelSpec {
	named := make([]ModelSpec, len(discovered))
	copy(named, discovered)
	for index := range named {
		if named[index].Label == "" {
			named[index].Label = named[index].ID
		}
		clearPrices(&named[index])
		if named[index].ContextWindow < 0 {
			named[index].ContextWindow = 0
		}
	}
	return named
}

func listedPosition(static []ModelSpec, spec ModelSpec) (int, bool) {
	for index := range static {
		if static[index].ID == spec.ID {
			return index, true
		}
	}
	for index := range static {
		if static[index].Route == spec.Route && static[index].Arg == spec.Arg {
			return index, true
		}
	}
	return 0, false
}

func clearPrices(spec *ModelSpec) {
	spec.InputPrice = 0
	spec.OutputPrice = 0
	spec.CacheReadPrice = 0
	spec.CacheWritePrice = 0
}
