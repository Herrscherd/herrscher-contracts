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

func TestForgeTokenPrintsTheRedactedMarkerForTheTextVerbs(t *testing.T) {
	token := newTestForgeToken(t)
	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		if got := fmt.Sprintf(format, token); got != "[redacted]" {
			t.Errorf("fmt.Sprintf(%q) = %q, want [redacted]", format, got)
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
	if !token.Valid() {
		t.Fatalf("Valid() = false for a built token, want true")
	}
}

func TestZeroForgeTokenIsNotValidAndHasNoSecret(t *testing.T) {
	var token ForgeToken
	if token.Valid() {
		t.Fatalf("Valid() = true for the zero token, want false")
	}
	if token.Secret() != "" {
		t.Fatalf("Secret() of the zero token = %q, want empty", token.Secret())
	}
	if got := token.String(); got != "[redacted]" {
		t.Fatalf("zero token prints %q, want [redacted]", got)
	}
}

func TestNewForgeTokenRefusesAnEmptySecret(t *testing.T) {
	token, err := NewForgeToken("", "octocat", TokenSourceEnv)
	if !errors.Is(err, ErrEmptyForgeToken) {
		t.Fatalf("NewForgeToken(empty secret) error = %v, want ErrEmptyForgeToken", err)
	}
	if token.Valid() {
		t.Fatalf("a refused token is valid, want the zero value")
	}
}

func TestNewForgeTokenRefusesAnUnknownSource(t *testing.T) {
	_, err := NewForgeToken(forgeTestSecret, "octocat", TokenSource(0))
	if !errors.Is(err, ErrUnknownTokenSource) {
		t.Fatalf("NewForgeToken(zero source) error = %v, want ErrUnknownTokenSource", err)
	}
}

func TestNewForgeTokenNamesTheRefusedSourceNumber(t *testing.T) {
	_, err := NewForgeToken(forgeTestSecret, "octocat", TokenSource(9))
	if err == nil || err.Error() != "token source 9: unknown forge token source" {
		t.Fatalf("NewForgeToken(source 9) error = %v, want the source number", err)
	}
}

func TestNewForgeTokenAcceptsEveryKnownSource(t *testing.T) {
	sources := map[TokenSource]string{
		TokenSourceSaved: "saved",
		TokenSourceApp:   "app",
		TokenSourceEnv:   "env",
		TokenSourceCLI:   "cli",
	}
	for source, name := range sources {
		token, err := NewForgeToken(forgeTestSecret, "octocat", source)
		if err != nil {
			t.Errorf("NewForgeToken(source %s) error = %v, want none", name, err)
			continue
		}
		if token.Source() != source {
			t.Errorf("Source() = %v, want %v", token.Source(), source)
		}
		raw, err := json.Marshal(token)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		if want := fmt.Sprintf(`{"login":"octocat","source":%q}`, name); string(raw) != want {
			t.Errorf("JSON for source %s = %s, want %s", name, raw, want)
		}
	}
}

func TestZeroForgeTokenMarshalsToNull(t *testing.T) {
	var token ForgeToken
	raw, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("json.Marshal(zero token) error = %v", err)
	}
	if string(raw) != "null" {
		t.Fatalf("JSON of the zero token = %s, want null", raw)
	}
}

func TestZeroForgeTokenJSONInsideAStructIsNull(t *testing.T) {
	raw, err := json.Marshal(struct {
		Token ForgeToken `json:"Token"`
	}{})
	if err != nil {
		t.Fatalf("json.Marshal(struct) error = %v", err)
	}
	if string(raw) != `{"Token":null}` {
		t.Fatalf("JSON of a struct holding the zero token = %s, want {\"Token\":null}", raw)
	}
}

func TestUnknownTokenSourceRendersAsUnknown(t *testing.T) {
	for _, source := range []TokenSource{0, 9, -1} {
		if got := source.String(); got != "unknown" {
			t.Errorf("TokenSource(%d).String() = %q, want unknown", int(source), got)
		}
	}
}
