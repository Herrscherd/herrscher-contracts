package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestValidateOutboundFileRefusesWithoutEchoingTheFile(t *testing.T) {
	const outbox = "/srv/outbox"
	tests := []struct {
		name string
		file OutboundFile
	}{
		{
			name: "relative path",
			file: OutboundFile{Path: receivedProbe + "/fact.txt", Name: "fact.txt"},
		},
		{
			name: "path outside the outbox",
			file: OutboundFile{Path: "/" + receivedProbe + "/elsewhere", Name: "fact.txt"},
		},
		{
			name: "name with a separator",
			file: OutboundFile{Path: outbox + "/fact.txt", Name: receivedProbe + "/x"},
		},
		{
			name: "name that is a dot",
			file: OutboundFile{Path: outbox + "/" + receivedProbe, Name: "."},
		},
		{
			name: "content type that does not parse",
			file: OutboundFile{Path: outbox + "/fact.txt", Name: "fact.txt", ContentType: receivedProbe + "; ;"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutboundFile(tt.file, outbox)
			if !errors.Is(err, ErrInvalidOutboundFile) {
				t.Fatalf("error = %v, want ErrInvalidOutboundFile", err)
			}
			echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
		})
	}
}
