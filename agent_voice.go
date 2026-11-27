package contracts

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"unicode"
)

// ReplyRule says when an agent answers on a chat platform. The zero value is
// [ReplyOnMention]. [ParseReplyRule] reads the text form and refuses any other
// text.
type ReplyRule uint8

// The rules an agent can follow on a chat platform.
const (
	ReplyOnMention ReplyRule = iota
	ReplyInChannels
	ReplyNever
)

// ErrUnknownReplyRule is returned for text, or a value, that names no reply rule.
// Match it with errors.Is.
var ErrUnknownReplyRule = errors.New("unknown reply rule")

// ReplyRules lists every reply rule, in the order of their values.
func ReplyRules() []ReplyRule {
	return []ReplyRule{ReplyOnMention, ReplyInChannels, ReplyNever}
}

// ParseReplyRule reads a rule from its text form: mention, channels or never.
// Any other text, the empty string included, returns an error matching
// [ErrUnknownReplyRule], which does not repeat the text.
func ParseReplyRule(text string) (ReplyRule, error) {
	for _, rule := range ReplyRules() {
		if rule.String() == text {
			return rule, nil
		}
	}
	return 0, ErrUnknownReplyRule
}

func (r ReplyRule) known() bool { return r <= ReplyNever }

// String writes the text form of the rule, or the raw number of a value that
// names no rule.
func (r ReplyRule) String() string {
	switch r {
	case ReplyOnMention:
		return "mention"
	case ReplyInChannels:
		return "channels"
	case ReplyNever:
		return "never"
	}
	return strconv.Itoa(int(r))
}

// MarshalText writes the text form of the rule. A value that names no rule
// returns an error matching [ErrUnknownReplyRule].
func (r ReplyRule) MarshalText() ([]byte, error) {
	if !r.known() {
		return nil, ErrUnknownReplyRule
	}
	return []byte(r.String()), nil
}

// UnmarshalText reads the text form of a rule, as [ParseReplyRule] does. It
// returns an error matching [ErrUnknownReplyRule] for any other text.
func (r *ReplyRule) UnmarshalText(text []byte) error {
	parsed, err := ParseReplyRule(string(text))
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}

const redactedBotToken = "[redacted]"

// ErrEmptyBotToken is returned by [NewBotToken] for an empty secret.
var ErrEmptyBotToken = errors.New("empty bot token")

// ErrInvalidBotToken is returned by [NewBotToken] for a secret that holds a
// control character or starts or ends with a space, as a line read with its
// line break would. The error does not repeat the secret.
var ErrInvalidBotToken = errors.New("invalid bot token")

// BotToken is the secret a gateway signs in with as one bot. The secret is
// unexported and reached only through [BotToken.Secret]. Every formatting verb
// prints [redacted], and the JSON form writes "[redacted]", or null for the zero
// value. The zero value holds no token. Build one with [NewBotToken].
type BotToken struct {
	secret *string
}

// NewBotToken builds a token. It returns [ErrEmptyBotToken] when secret is
// empty, and [ErrInvalidBotToken] when it holds a control character or a
// leading or trailing space.
func NewBotToken(secret string) (BotToken, error) {
	if secret == "" {
		return BotToken{}, ErrEmptyBotToken
	}
	if strings.TrimSpace(secret) != secret || strings.ContainsFunc(secret, unicode.IsControl) {
		return BotToken{}, ErrInvalidBotToken
	}
	return BotToken{secret: &secret}, nil
}

// Secret is the token itself, for the connection that sends it. Never log or
// store its result. It is empty for the zero value.
func (t BotToken) Secret() string {
	if t.secret == nil {
		return ""
	}
	return *t.secret
}

// Valid reports whether the token holds a secret. It is false for the zero
// value.
func (t BotToken) Valid() bool { return t.secret != nil }

// String renders a placeholder in place of the secret.
func (t BotToken) String() string { return redactedBotToken }

// GoString renders a placeholder in place of the secret for %#v.
func (t BotToken) GoString() string { return redactedBotToken }

// MarshalJSON writes a placeholder in place of the secret, and null for the
// zero value.
func (t BotToken) MarshalJSON() ([]byte, error) {
	if !t.Valid() {
		return []byte("null"), nil
	}
	return json.Marshal(redactedBotToken)
}

// AgentVoice is how one agent speaks on one gateway. Token is the agent's own
// bot on that gateway; its zero value means the agent speaks through the
// gateway's own bot, under its name. Reply is when the agent answers, and
// Channels lists the conversation ids of that gateway where [ReplyInChannels]
// answers every message.
type AgentVoice struct {
	Agent    string    `json:"Agent"`
	Token    BotToken  `json:"Token"`
	Reply    ReplyRule `json:"Reply"`
	Channels []string  `json:"Channels"`
}

// AgentDirectory is the optional capability of the [SessionControl] a gateway
// receives through [SessionControlReceiver], found by a type assertion.
// AgentVoices lists the agents of the daemon with how each speaks on the gateway
// of that Manifest Kind. The implementer sorts the list by agent name and tests
// it, makes the method safe for concurrent calls, and returns a fresh copy,
// Channels included, that the caller may keep and change. A gateway reads it
// again when it needs a fresh rule, since an agent's rule can change while the
// daemon runs; the tokens are those of the daemon's startup.
type AgentDirectory interface {
	AgentVoices(gateway string) []AgentVoice
}
