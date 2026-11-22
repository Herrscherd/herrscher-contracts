package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func nameWrongJSONType(err, sentinel error, subject string, wireFields []string) error {
	typeErr, ok := errors.AsType[*json.UnmarshalTypeError](err)
	if !ok {
		return err
	}
	return wrongJSONTypeError(sentinel, subject, wireFields, typeErr)
}

func wrongJSONTypeError(sentinel error, subject string, wireFields []string, err *json.UnmarshalTypeError) error {
	field, known := canonicalWireField(wireFields, err.Field)
	if !known {
		return wrongWireType(sentinel, subject)
	}
	return wrongWireType(sentinel, field)
}

func wrongWireType(sentinel error, field string) error {
	return fmt.Errorf("%w: %s has the wrong JSON type", sentinel, field)
}

func canonicalWireField(wireFields []string, name string) (string, bool) {
	topLevel, _, _ := strings.Cut(name, ".")
	for _, field := range wireFields {
		if strings.EqualFold(field, topLevel) {
			return field, true
		}
	}
	return "", false
}
