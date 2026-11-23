package contracts

import (
	"encoding/json"
	"fmt"
)

const redactedForgeToken = "[redacted]"

// ForgeToken is the credential a host holds for one forge host, with the account
// it belongs to and where it came from. The secret is unexported and reached only
// through [ForgeToken.Secret]. Every formatting verb and the JSON form print a
// placeholder and the account, never the secret: the secret sits behind a pointer,
// which fmt never follows inside a struct. Build one with [NewForgeToken].
type ForgeToken struct {
	secret *string
	login  string
	source TokenSource
}

// NewForgeToken builds a token. It returns [ErrForgeNotConnected] when secret is
// empty, and an error matching [ErrUnknownTokenSource] when source names no source.
func NewForgeToken(secret, login string, source TokenSource) (ForgeToken, error) {
	if secret == "" {
		return ForgeToken{}, ErrForgeNotConnected
	}
	if !source.known() {
		return ForgeToken{}, fmt.Errorf("%s: %w", source, ErrUnknownTokenSource)
	}
	return ForgeToken{secret: &secret, login: login, source: source}, nil
}

// Secret is the token itself, for the one request that sends it. Never log or
// store its result. It is empty for the zero value.
func (t ForgeToken) Secret() string {
	if t.secret == nil {
		return ""
	}
	return *t.secret
}

// Login is the account the token authenticates, as the forge names it.
func (t ForgeToken) Login() string { return t.login }

// Source is where the token came from.
func (t ForgeToken) Source() TokenSource { return t.source }

// String renders a placeholder in place of the secret.
func (t ForgeToken) String() string { return redactedForgeToken }

// GoString renders a placeholder in place of the secret for %#v.
func (t ForgeToken) GoString() string { return redactedForgeToken }

// MarshalJSON writes the account and the source of the token, never its secret.
func (t ForgeToken) MarshalJSON() ([]byte, error) {
	return json.Marshal(forgeTokenWire{Login: t.login, Source: t.source.String()})
}

type forgeTokenWire struct {
	Login  string `json:"login"`
	Source string `json:"source"`
}
