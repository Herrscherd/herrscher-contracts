package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNoticeLimit reports a notice whose argument count or argument size exceeds its JSON limits.
var ErrNoticeLimit = errors.New("notice exceeds the wire limits")

const (
	maxNoticeArgs     = 16
	maxNoticeArgBytes = 1 << 16
)

type argWire struct {
	Name  string          `json:"name"`
	Kind  uint8           `json:"kind"`
	Value json.RawMessage `json:"value"`
}

type noticeWire struct {
	Code uint16 `json:"code"`
	Args []Arg  `json:"args,omitempty"`
}

type noticeWireIn struct {
	Code uint16          `json:"code"`
	Args json.RawMessage `json:"args"`
}

// MarshalJSON encodes the notice as {"code":N,"args":[...]}, args omitted when empty.
func (n Notice) MarshalJSON() ([]byte, error) {
	if len(n.Args) > maxNoticeArgs {
		return nil, fmt.Errorf("notice code %d takes at most %d arguments: %w", uint16(n.Code), maxNoticeArgs, ErrNoticeLimit)
	}
	return json.Marshal(noticeWire{Code: uint16(n.Code), Args: n.Args})
}

// UnmarshalJSON reads the form MarshalJSON writes and the capitalised v0.30.0 form.
func (n *Notice) UnmarshalJSON(data []byte) error {
	var envelope noticeWireIn
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("notice: %w", err)
	}
	args, err := decodeNoticeArgs(envelope.Args)
	if err != nil {
		return err
	}
	*n = Notice{Code: Code(envelope.Code), Args: args}
	return nil
}

// MarshalJSON encodes the argument as {"name":...,"kind":...,"value":...}.
func (a Arg) MarshalJSON() ([]byte, error) {
	value, err := json.Marshal(a.value)
	if err != nil {
		return nil, fmt.Errorf("argument %q: %w", a.name, err)
	}
	if len(value) > maxNoticeArgBytes {
		return nil, fmt.Errorf("argument %q: %w", a.name, ErrNoticeLimit)
	}
	return json.Marshal(argWire{Name: a.name, Kind: uint8(a.kind), Value: value})
}

// UnmarshalJSON reads an argument and refuses a value of the wrong JSON type with ErrNoticeArgs.
func (a *Arg) UnmarshalJSON(data []byte) error {
	var frame argWire
	if err := json.Unmarshal(data, &frame); err != nil {
		return fmt.Errorf("notice argument: %w", err)
	}
	if len(frame.Value) > maxNoticeArgBytes {
		return fmt.Errorf("argument %q: %w", frame.Name, ErrNoticeLimit)
	}
	kind := ArgKind(frame.Kind)
	value, err := decodeArgValue(kind, frame.Value)
	if err != nil {
		return fmt.Errorf("argument %q: %w", frame.Name, err)
	}
	*a = Arg{name: frame.Name, kind: kind, value: value}
	return nil
}

func decodeArgValue(kind ArgKind, raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	switch kind {
	case ArgText, ArgRef:
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNoticeArgs, err)
		}
		return text, nil
	case ArgCount:
		var count int
		if err := json.Unmarshal(raw, &count); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNoticeArgs, err)
		}
		return count, nil
	}
	return nil, nil
}

func decodeNoticeArgs(raw json.RawMessage) ([]Arg, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	open, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("notice arguments: %w", err)
	}
	if open != json.Delim('[') {
		return nil, fmt.Errorf("notice arguments are not a list: %w", ErrNoticeArgs)
	}
	var args []Arg
	for decoder.More() {
		if len(args) == maxNoticeArgs {
			return nil, fmt.Errorf("notice takes at most %d arguments: %w", maxNoticeArgs, ErrNoticeLimit)
		}
		var arg Arg
		if err := decoder.Decode(&arg); err != nil {
			return nil, err
		}
		args = append(args, arg)
	}
	return args, nil
}
