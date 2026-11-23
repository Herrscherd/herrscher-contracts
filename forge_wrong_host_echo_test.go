package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestRequireRepoHostRefusesAWrongHostWithoutEchoingTheRepository(t *testing.T) {
	host, err := ParseForgeHost("gitlab.example.com")
	if err != nil {
		t.Fatalf("ParseForgeHost: %v", err)
	}
	repo, err := ParseRepoRef("github.com/" + receivedProbe + "/hello")
	if err != nil {
		t.Fatalf("ParseRepoRef: %v", err)
	}
	err = RequireRepoHost(host, repo)
	if !errors.Is(err, ErrForgeWrongHost) {
		t.Fatalf("error = %v, want ErrForgeWrongHost", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
}
