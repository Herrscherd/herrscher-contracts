package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNoticeLimit reports a notice whose argument count, argument name or argument size exceeds its JSON limits.
var ErrNoticeLimit = errors.New("notice exceeds the wire limits")

const (
	maxNoticeArgs      = 16
	maxNoticeArgBytes  = 1 << 16
	maxNoticeNameBytes = 256
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

// UnmarshalJSON reads the form MarshalJSON writes and the capitalised v0.30.0 form; null leaves the notice unchanged, as in v0.30.0.
func (n *Notice) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var envelope noticeWireIn
	if err := json.Unmarshal(data, &envelope); err != nil {
		return shapeError(err)
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
	if len(a.name) > maxNoticeNameBytes {
		return nil, fmt.Errorf("argument name is longer than %d bytes: %w", maxNoticeNameBytes, ErrNoticeLimit)
	}
	value, err := encodeValue(a.value)
	if err != nil {
		return nil, fmt.Errorf("argument %q: %w", a.name, err)
	}
	return json.Marshal(argWire{Name: a.name, Kind: uint8(a.kind), Value: value})
}

// UnmarshalJSON reads an argument; a null argument, a wrong JSON type or a malformed shape is refused with ErrNoticeArgs.
func (a *Arg) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return fmt.Errorf("notice argument is null: %w", ErrNoticeArgs)
	}
	var frame argWire
	if err := json.Unmarshal(data, &frame); err != nil {
		return shapeError(err)
	}
	if len(frame.Name) > maxNoticeNameBytes {
		return fmt.Errorf("argument name is longer than %d bytes: %w", maxNoticeNameBytes, ErrNoticeLimit)
	}
	decoded := Arg{name: frame.Name, kind: ArgKind(frame.Kind)}
	if err := decodeValue(&decoded, frame.Value); err != nil {
		return fmt.Errorf("argument %q: %w", frame.Name, err)
	}
	if _, err := encodeValue(decoded.value); err != nil {
		return fmt.Errorf("argument %q: %w", frame.Name, err)
	}
	*a = decoded
	return nil
}

func encodeValue(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(encoded) > maxNoticeArgBytes {
		return nil, fmt.Errorf("value of %d bytes is longer than %d: %w", len(encoded), maxNoticeArgBytes, ErrNoticeLimit)
	}
	return encoded, nil
}

func shapeError(err error) error {
	if _, ok := errors.AsType[*json.SyntaxError](err); ok {
		return err
	}
	return fmt.Errorf("%w: %w", ErrNoticeArgs, err)
}

func decodeValue(arg *Arg, raw json.RawMessage) error {
	if isNullOrAbsent(raw) {
		if arg.kind == ArgText || arg.kind == ArgCount || arg.kind == ArgRef {
			return fmt.Errorf("%w: a value is required for kind %d", ErrNoticeArgs, arg.kind)
		}
		return nil
	}
	switch arg.kind {
	case ArgText, ArgRef:
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return fmt.Errorf("%w: %w", ErrNoticeArgs, err)
		}
		arg.value = text
	case ArgCount:
		var count int
		if err := json.Unmarshal(raw, &count); err != nil {
			return fmt.Errorf("%w: %w", ErrNoticeArgs, err)
		}
		arg.value = count
	default:
		arg.value = append(json.RawMessage(nil), raw...)
	}
	return nil
}

func isNullOrAbsent(raw []byte) bool {
	return len(raw) == 0 || string(raw) == "null"
}

func decodeNoticeArgs(raw json.RawMessage) ([]Arg, error) {
	if isNullOrAbsent(raw) {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	open, err := decoder.Token()
	if err != nil {
		return nil, shapeError(err)
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
