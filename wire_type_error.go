package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const wireReason = "Reason"

func nameWrongJSONType(err, sentinel error, subject string, wireFields []string) error {
	typeErr, ok := errors.AsType[*json.UnmarshalTypeError](err)
	if !ok {
		return err
	}
	return wrongWireType(sentinel, wireFieldOf(typeErr.Field, wireFields, subject))
}

func wrongWireType(sentinel error, field string) error {
	return fmt.Errorf("%w: %s has the wrong JSON type", sentinel, field)
}

func wireFieldOf(path string, wireFields []string, subject string) string {
	topLevel, _, _ := strings.Cut(path, ".")
	for _, field := range wireFields {
		if strings.EqualFold(field, topLevel) {
			return field
		}
	}
	return subject
}
