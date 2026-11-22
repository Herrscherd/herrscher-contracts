package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const wireReason = "Reason"

func nameWrongJSONType(err, sentinel error, subject string, wireFields []string) error {
	typeErr, ok := errors.AsType[*json.UnmarshalTypeError](err)
	if !ok {
		return err
	}
	return wireTypeError(sentinel, wireFieldOf(typeErr.Field, wireFields, subject), typeErr)
}

func nameWrongJSONTypeOf(err, sentinel error, field string) error {
	return nameWrongJSONType(err, sentinel, field, nil)
}

func wireTypeError(sentinel error, field string, typeErr *json.UnmarshalTypeError) error {
	if isOutOfRange(typeErr) {
		return wrongWireRange(sentinel, field)
	}
	return wrongWireType(sentinel, field)
}

func isOutOfRange(typeErr *json.UnmarshalTypeError) bool {
	number, isNumber := strings.CutPrefix(typeErr.Value, "number ")
	if !isNumber {
		return false
	}
	_, parseErr := strconv.ParseInt(number, 10, 64)
	return parseErr == nil || errors.Is(parseErr, strconv.ErrRange)
}

func wrongWireType(sentinel error, field string) error {
	return fmt.Errorf("%w: %s has the wrong JSON type", sentinel, field)
}

func wrongWireRange(sentinel error, field string) error {
	return fmt.Errorf("%w: %s is out of range", sentinel, field)
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
