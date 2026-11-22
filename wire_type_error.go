package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	wireReason  = "Reason"
	float64Bits = 64
)

func nameWrongJSONType(err, sentinel error, subject string, wireFields []string) error {
	typeErr, ok := errors.AsType[*json.UnmarshalTypeError](err)
	if !ok {
		return err
	}
	return wireTypeError(sentinel, wireFieldOf(typeErr.Field, wireFields, subject), typeErr)
}

func nameWrongArgJSONType(err, sentinel error) error {
	typeErr, ok := errors.AsType[*json.UnmarshalTypeError](err)
	if !ok {
		return err
	}
	return wireTypeError(sentinel, noticeWireArgs, typeErr)
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
	if isPlainNonzeroInteger(number) {
		return true
	}
	_, err := strconv.ParseFloat(number, float64Bits)
	return errors.Is(err, strconv.ErrRange)
}

func isPlainNonzeroInteger(number string) bool {
	digits := strings.TrimPrefix(number, "-")
	return strings.Trim(digits, "0123456789") == "" && strings.Trim(digits, "0") != ""
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
