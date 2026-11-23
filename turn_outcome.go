package contracts

import "errors"

// TurnOutcome says how a turn ended when it did not complete: a failed backend or
// an interrupted turn. The zero value is a completed turn. [ParseTurnOutcome]
// reads the text form and refuses any text beyond the constants below.
type TurnOutcome string

// The outcomes a terminal reply can carry besides a completed turn.
const (
	TurnFailed      TurnOutcome = "failed"
	TurnInterrupted TurnOutcome = "interrupted"
)

// ErrUnknownTurnOutcome is returned for text, or a value, that names no turn outcome.
// Match it with errors.Is.
var ErrUnknownTurnOutcome = errors.New("unknown turn outcome")

// ParseTurnOutcome reads an outcome from its text form. The empty text is a
// completed turn, the zero TurnOutcome. Any other text that is not a constant
// returns an error matching [ErrUnknownTurnOutcome], which does not repeat the text.
func ParseTurnOutcome(text string) (TurnOutcome, error) {
	if outcome, ok := knownTurnOutcome(text); ok {
		return outcome, nil
	}
	return "", ErrUnknownTurnOutcome
}

func knownTurnOutcome(text string) (TurnOutcome, bool) {
	switch outcome := TurnOutcome(text); outcome {
	case "", TurnFailed, TurnInterrupted:
		return outcome, true
	}
	return "", false
}

// MarshalText writes the text form of the outcome. A value that names no turn
// outcome returns an error matching [ErrUnknownTurnOutcome].
func (o TurnOutcome) MarshalText() ([]byte, error) {
	if _, ok := knownTurnOutcome(string(o)); !ok {
		return nil, ErrUnknownTurnOutcome
	}
	return []byte(o), nil
}

// UnmarshalText reads the text form of an outcome, as [ParseTurnOutcome] does. A
// refused text is not quoted, so decoding never echoes a received value.
func (o *TurnOutcome) UnmarshalText(text []byte) error {
	parsed, err := ParseTurnOutcome(string(text))
	if err != nil {
		return err
	}
	*o = parsed
	return nil
}
