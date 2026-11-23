package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestRequireRepoHostRefusesAWrongHostWithoutEchoingTheRepository(t *testing.T) {
	host, err := ParseForgeHost("gitlab.example.com")
	if err != nil {
		t.Fatalf("ParseForgeHost: %v", err)
	}
	repo, err := ParseRepoRef("github.com/" + receivedSecret + "/hello")
	if err != nil {
		t.Fatalf("ParseRepoRef: %v", err)
	}
	err = RequireRepoHost(host, repo)
	if !errors.Is(err, ErrForgeWrongHost) {
		t.Fatalf("error = %v, want ErrForgeWrongHost", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received repository", err)
	}
}
