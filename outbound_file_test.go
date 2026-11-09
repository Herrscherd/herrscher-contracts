package contracts

import (
	"errors"
	"testing"
)

func TestValidateOutboundFileAcceptsAnAbsoluteNamedFile(t *testing.T) {
	file := OutboundFile{Path: "/w/out/pixel.png", Name: "pixel.png", ContentType: "image/png"}
	if err := ValidateOutboundFile(file); err != nil {
		t.Fatalf("ValidateOutboundFile(%+v) = %v, want nil", file, err)
	}
}

func TestValidateOutboundFileRefusesEachMalformedFile(t *testing.T) {
	cases := map[string]OutboundFile{
		"relative path":      {Path: "out/pixel.png", Name: "pixel.png"},
		"empty path":         {Name: "pixel.png"},
		"no name":            {Path: "/w/out/pixel.png"},
		"slash in name":      {Path: "/w/out/pixel.png", Name: "out/pixel.png"},
		"backslash in name":  {Path: "/w/out/pixel.png", Name: `out\pixel.png`},
		"name is only a dir": {Path: "/w/out/pixel.png", Name: "../"},
	}
	for name, file := range cases {
		if err := ValidateOutboundFile(file); !errors.Is(err, ErrInvalidOutboundFile) {
			t.Fatalf("%s: ValidateOutboundFile(%+v) error = %v, want ErrInvalidOutboundFile", name, file, err)
		}
	}
}
