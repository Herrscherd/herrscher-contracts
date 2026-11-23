package contracts

import (
	"errors"
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
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
