package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateOutboundFileRefusesWithoutEchoingTheFile(t *testing.T) {
	const outbox = "/srv/outbox"
	tests := []struct {
		name string
		file OutboundFile
	}{
		{
			name: "relative path",
			file: OutboundFile{Path: "rel/" + receivedSecret, Name: "fact.txt"},
		},
		{
			name: "path outside the outbox",
			file: OutboundFile{Path: "/elsewhere/" + receivedSecret, Name: "fact.txt"},
		},
		{
			name: "name with a separator",
			file: OutboundFile{Path: outbox + "/fact.txt", Name: receivedSecret + "/x"},
		},
		{
			name: "name that is a dot",
			file: OutboundFile{Path: outbox + "/" + receivedSecret, Name: "."},
		},
		{
			name: "content type that does not parse",
			file: OutboundFile{Path: outbox + "/fact.txt", Name: "fact.txt", ContentType: receivedSecret + "; ;"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutboundFile(tt.file, outbox)
			if !errors.Is(err, ErrInvalidOutboundFile) {
				t.Fatalf("error = %v, want ErrInvalidOutboundFile", err)
			}
			if strings.Contains(err.Error(), receivedSecret) {
				t.Fatalf("message %q echoes the received file", err)
			}
		})
	}
}
