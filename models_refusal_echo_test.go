package contracts

import (
	"errors"
	"math"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestValidateModelsRefusalsDoNotEchoTheirInput(t *testing.T) {
	modelWith := func(mutate func(*ModelSpec)) ModelSpec {
		spec := ModelSpec{ID: receivedProbe, Label: "Label", Arg: "arg", Route: RouteNative}
		if mutate != nil {
			mutate(&spec)
		}
		return spec
	}
	tests := []struct {
		name    string
		models  []ModelSpec
		wantErr error
	}{
		{"contextWindow", []ModelSpec{modelWith(func(s *ModelSpec) { s.ContextWindow = -1 })}, ErrModelNegative},
		{"inputPriceNotFinite", []ModelSpec{modelWith(func(s *ModelSpec) { s.InputPrice = math.NaN() })}, ErrModelNotFinite},
		{"outputPriceNegative", []ModelSpec{modelWith(func(s *ModelSpec) { s.OutputPrice = -1 })}, ErrModelNegative},
		{"duplicateID", []ModelSpec{modelWith(nil), modelWith(nil)}, ErrModelDuplicateID},
		{"emptyLabel", []ModelSpec{modelWith(func(s *ModelSpec) { s.Label = " " })}, ErrModelEmptyLabel},
		{"emptyArg", []ModelSpec{modelWith(func(s *ModelSpec) { s.Arg = " " })}, ErrModelEmptyArg},
		{"controlCharField", []ModelSpec{modelWith(func(s *ModelSpec) { s.ID += "\x01" })}, ErrModelControlChar},
		{"controlCharEffort", []ModelSpec{modelWith(func(s *ModelSpec) { s.Efforts = []string{receivedProbe + "\x01"} })}, ErrModelControlChar},
		{"unknownRoute", []ModelSpec{modelWith(func(s *ModelSpec) { s.Route = Route(receivedProbe) })}, ErrModelUnknownRoute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModels("kind", tt.models)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
		})
	}
}
