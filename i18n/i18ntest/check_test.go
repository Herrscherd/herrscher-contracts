package i18ntest

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
)

type recorder struct {
	failures []string
	fatal    []string
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.fatal = append(r.fatal, fmt.Sprintf(format, args...))
}

func (r *recorder) reported(fragment string) bool {
	return slices.ContainsFunc(r.failures, func(failure string) bool {
		return strings.Contains(failure, fragment)
	})
}

func TestCheckAcceptsCompleteCatalog(t *testing.T) {
	rec := &recorder{}
	Check(rec, os.DirFS("../testdata/complete"))
	if len(rec.failures) != 0 || len(rec.fatal) != 0 {
		t.Fatalf("Check() reported %v, fatal %v", rec.failures, rec.fatal)
	}
}

func TestCheckRefusesViolations(t *testing.T) {
	cases := []struct {
		name string
		dir  string
		want string
	}{
		{"missing key in a language", "incomplete", `message "session.archived" missing in fr`},
		{"missing plural form", "plural_forms", `in fr lacks the "one" plural form`},
		{"different fields", "params", "uses fields [Title], English uses [Name]"},
		{"template function", "functions", `function "printf" not defined`},
		{"control structure", "control", "only {{.Name}} fields are allowed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recorder{}
			Check(rec, os.DirFS("../testdata/"+tc.dir))
			if !rec.reported(tc.want) {
				t.Fatalf("Check(%s) reported %v, want a failure containing %q", tc.dir, rec.failures, tc.want)
			}
		})
	}
}

func TestCheckStopsWhenCatalogCannotBeRead(t *testing.T) {
	rec := &recorder{}
	Check(rec, os.DirFS("../testdata/complete"), os.DirFS("../testdata/duplicate"))
	if len(rec.fatal) != 1 || len(rec.failures) != 0 {
		t.Fatalf("Check() fatal = %v, failures = %v, want one fatal error", rec.fatal, rec.failures)
	}
}
