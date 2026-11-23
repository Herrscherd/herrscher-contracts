package contracts

import (
	"errors"
	"testing"
)

func TestPageRequestAcceptsTheLimitRange(t *testing.T) {
	for _, limit := range []int{0, 1, 100} {
		if err := (PageRequest{Limit: limit}).Validate(); err != nil {
			t.Errorf("Validate(limit %d) error = %v, want none", limit, err)
		}
	}
}

func TestPageRequestRefusesALimitOutsideTheRange(t *testing.T) {
	for _, limit := range []int{-1, 101} {
		err := (PageRequest{Limit: limit}).Validate()
		if !errors.Is(err, ErrInvalidPageLimit) {
			t.Errorf("Validate(limit %d) error = %v, want ErrInvalidPageLimit", limit, err)
		}
	}
}

func TestPageCursorEmptyIsTheFirstPage(t *testing.T) {
	first := PageRequest{}
	if first.Cursor != "" || first.Limit != 0 {
		t.Fatalf("zero PageRequest = %+v, want the first page at the default size", first)
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("zero PageRequest Validate() error = %v, want none", err)
	}
}
