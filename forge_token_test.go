package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const forgeTestSecret = "ghp_s3cr3tValue"

func newTestForgeToken(t *testing.T) ForgeToken {
	t.Helper()
	token, err := NewForgeToken(forgeTestSecret, "octocat", TokenSourceApp)
	if err != nil {
		t.Fatalf("NewForgeToken() error = %v", err)
	}
	return token
}

func TestForgeTokenNeverPrintsItsSecret(t *testing.T) {
	token := newTestForgeToken(t)
	for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%d"} {
		if got := fmt.Sprintf(format, token); strings.Contains(got, forgeTestSecret) {
			t.Errorf("fmt.Sprintf(%q) = %q, leaks the secret", format, got)
		}
	}
}

func TestForgeTokenNeverPrintsItsSecretInsideAStruct(t *testing.T) {
	holder := struct{ Token ForgeToken }{Token: newTestForgeToken(t)}
	if got := fmt.Sprintf("%+v", holder); strings.Contains(got, forgeTestSecret) {
		t.Fatalf("%%+v of a struct holding a token = %q, leaks the secret", got)
	}
}

func TestForgeTokenJSONOmitsItsSecret(t *testing.T) {
	raw, err := json.Marshal(newTestForgeToken(t))
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(raw), forgeTestSecret) {
		t.Fatalf("JSON = %s, leaks the secret", raw)
	}
	if string(raw) != `{"login":"octocat","source":"app"}` {
		t.Fatalf("JSON = %s, want the login and the source", raw)
	}
}

func TestForgeTokenAccessorsReturnWhatWasBuilt(t *testing.T) {
	token := newTestForgeToken(t)
	if token.Secret() != forgeTestSecret || token.Login() != "octocat" || token.Source() != TokenSourceApp {
		t.Fatalf("accessors = %q, %q, %v, want the built values", token.Secret(), token.Login(), token.Source())
	}
}

func TestNewForgeTokenRefusesAnEmptySecret(t *testing.T) {
	_, err := NewForgeToken("", "octocat", TokenSourceEnv)
	if !errors.Is(err, ErrForgeNotConnected) {
		t.Fatalf("NewForgeToken(empty secret) error = %v, want ErrForgeNotConnected", err)
	}
}

func TestNewForgeTokenRefusesAnUnknownSource(t *testing.T) {
	_, err := NewForgeToken(forgeTestSecret, "octocat", TokenSource(0))
	if !errors.Is(err, ErrUnknownTokenSource) {
		t.Fatalf("NewForgeToken(zero source) error = %v, want ErrUnknownTokenSource", err)
	}
}
