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
	named := labelDiscovered(discovered)
	if err := ValidateModels(kind, named); err != nil {
		return nil, fmt.Errorf("discovered catalog: %w", err)
	}
	merged := append([]ModelSpec{}, static...)
	for _, spec := range named {
		if isListed(merged, spec) {
			continue
		}
		merged = append(merged, withoutPrice(spec))
	}
	if err := ValidateModels(kind, merged); err != nil {
		return nil, fmt.Errorf("merged catalog: %w", err)
	}
	return merged, nil
}

func labelDiscovered(discovered []ModelSpec) []ModelSpec {
	named := make([]ModelSpec, len(discovered))
	for index, spec := range discovered {
		if spec.Label == "" {
			spec.Label = spec.ID
		}
		named[index] = spec
	}
	return named
}

func isListed(static []ModelSpec, spec ModelSpec) bool {
	for _, known := range static {
		if known.ID == spec.ID || (known.Route == spec.Route && known.Arg == spec.Arg) {
			return true
		}
	}
	return false
}

func withoutPrice(spec ModelSpec) ModelSpec {
	spec.InputPrice = 0
	return spec
}
