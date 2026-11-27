package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

const botTestSecret = "123456:AAbot-s3cr3t"

func TestReplyRuleRoundTripsThroughText(t *testing.T) {
	for _, rule := range ReplyRules() {
		text, err := rule.MarshalText()
		if err != nil {
			t.Fatalf("%v.MarshalText() error = %v", rule, err)
		}
		var back ReplyRule
		if err := back.UnmarshalText(text); err != nil || back != rule {
			t.Fatalf("UnmarshalText(%q) = %v, %v, want %v", text, back, err, rule)
		}
	}
}

func TestReplyRuleZeroValueIsOnMention(t *testing.T) {
	var rule ReplyRule
	if rule != ReplyOnMention || rule.String() != "mention" {
		t.Fatalf("zero ReplyRule = %v, want mention", rule)
	}
}

func TestParseReplyRuleRefusesUnknownTextWithoutEcho(t *testing.T) {
	for _, text := range []string{"", "Mention", "always-on"} {
		_, err := ParseReplyRule(text)
		if text == "" {
			echotest.RequireBareSentinel(t, err, ErrUnknownReplyRule)
			continue
		}
		echotest.RequireRefusalWithoutEcho(t, err, ErrUnknownReplyRule, text)
	}
}

func TestReplyRuleRefusesAnUnknownValueOnTheWire(t *testing.T) {
	if _, err := json.Marshal(ReplyRule(9)); !errors.Is(err, ErrUnknownReplyRule) {
		t.Fatalf("json.Marshal(ReplyRule(9)) error = %v, want ErrUnknownReplyRule", err)
	}
	if got := ReplyRule(9).String(); got != "9" {
		t.Fatalf("ReplyRule(9).String() = %q, want 9", got)
	}
}

func TestNewBotTokenRefusesAnEmptySecret(t *testing.T) {
	_, err := NewBotToken("")
	echotest.RequireBareSentinel(t, err, ErrEmptyBotToken)
}

func TestBotTokenNeverPrintsItsSecret(t *testing.T) {
	token, err := NewBotToken(botTestSecret)
	if err != nil {
		t.Fatalf("NewBotToken() error = %v", err)
	}
	voice := AgentVoice{Agent: "rin", Token: token}
	for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x"} {
		for _, value := range []any{token, voice, &voice} {
			if got := fmt.Sprintf(format, value); strings.Contains(got, botTestSecret) {
				t.Errorf("fmt.Sprintf(%q) = %q, leaks the secret", format, got)
			}
		}
	}
	data, err := json.Marshal(voice)
	if err != nil {
		t.Fatalf("json.Marshal(voice) error = %v", err)
	}
	if strings.Contains(string(data), botTestSecret) {
		t.Fatalf("json.Marshal(voice) = %s, leaks the secret", data)
	}
	if token.Secret() != botTestSecret || !token.Valid() {
		t.Fatalf("Secret() = %q, Valid() = %v", token.Secret(), token.Valid())
	}
}

func TestZeroBotTokenHoldsNothing(t *testing.T) {
	var token BotToken
	data, err := json.Marshal(token)
	if err != nil || string(data) != "null" {
		t.Fatalf("json.Marshal(zero) = %s, %v, want null", data, err)
	}
	if token.Valid() || token.Secret() != "" {
		t.Fatalf("zero token Valid() = %v, Secret() = %q", token.Valid(), token.Secret())
	}
}
