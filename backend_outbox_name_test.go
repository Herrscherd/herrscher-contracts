package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateOutboundFileNamesTheHostOutbox(t *testing.T) {
	const outbox = "/srv/outbox"
	err := ValidateOutboundFile(OutboundFile{Path: "/elsewhere/fact.txt", Name: "fact.txt"}, outbox)
	if !errors.Is(err, ErrInvalidOutboundFile) {
		t.Fatalf("error = %v, want ErrInvalidOutboundFile", err)
	}
	if !strings.Contains(err.Error(), `outside the outbox "/srv/outbox"`) {
		t.Fatalf("message %q does not name the host outbox", err)
	}
}
