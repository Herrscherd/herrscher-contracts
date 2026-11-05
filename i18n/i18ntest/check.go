package i18ntest

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"text/template/parse"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

// TB is the part of testing.TB that Check uses. A test passes its *testing.T, and
// a test of this package passes a recorder.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

var errOnlyFields = errors.New("only {{.Name}} fields are allowed in a message")

const (
	oneForm   = "one"
	otherForm = "other"
)

type fieldSets map[i18n.Language]map[string]map[string]struct{}

// Check runs the catalog guards over sources, the file systems given to
// i18n.Messages. Every key of English must be defined in each supported language,
// with the plural forms that language needs when the English message is plural.
// Every language must use the same {{.Name}} fields as English, and a message
// holds nothing but such fields. Every key a language defines must also exist in
// English, so an orphan key is refused. Each violation is reported with Errorf,
// and a source that cannot be read stops the test with Fatalf.
func Check(reporter TB, sources ...fs.FS) {
	reporter.Helper()
	messages, err := i18n.Messages(sources...)
	if err != nil {
		reporter.Fatalf("read catalogs: %v", err)
		return
	}
	fields := collectFields(reporter, messages)
	for _, key := range sortedKeys(messages[i18n.English]) {
		checkKey(reporter, messages, fields, key)
	}
	checkOrphans(reporter, messages)
}

func checkOrphans(reporter TB, messages map[i18n.Language]map[string]*goi18n.Message) {
	reporter.Helper()
	for _, lang := range i18n.Languages() {
		for _, key := range sortedKeys(messages[lang]) {
			if _, defined := messages[i18n.English][key]; !defined {
				reporter.Errorf("message %q in %s is not defined in English", key, lang)
			}
		}
	}
}

func collectFields(reporter TB, messages map[i18n.Language]map[string]*goi18n.Message) fieldSets {
	reporter.Helper()
	fields := make(fieldSets, len(i18n.Languages()))
	for _, lang := range i18n.Languages() {
		fields[lang] = make(map[string]map[string]struct{}, len(messages[lang]))
		for _, key := range sortedKeys(messages[lang]) {
			names, err := messageFields(messages[lang][key])
			if err != nil {
				reporter.Errorf("message %q in %s: %v", key, lang, err)
				continue
			}
			fields[lang][key] = names
		}
	}
	return fields
}

func checkKey(reporter TB, messages map[i18n.Language]map[string]*goi18n.Message, fields fieldSets, key string) {
	reporter.Helper()
	source := messages[i18n.English][key]
	for _, lang := range i18n.Languages() {
		target, defined := messages[lang][key]
		if !defined {
			reporter.Errorf("message %q missing in %s", key, lang)
			continue
		}
		for _, form := range requiredForms(lang, source) {
			if formText(target, form) == "" {
				reporter.Errorf("message %q in %s lacks the %q plural form", key, lang, form)
			}
		}
		sourceNames, sourceOK := fields[i18n.English][key]
		targetNames, targetOK := fields[lang][key]
		if sourceOK && targetOK && !maps.Equal(sourceNames, targetNames) {
			reporter.Errorf("message %q in %s uses fields %v, English uses %v", key, lang, sortedNames(targetNames), sortedNames(sourceNames))
		}
	}
}

func isPlural(message *goi18n.Message) bool {
	return message.Zero+message.One+message.Two+message.Few+message.Many != ""
}

func requiredForms(lang i18n.Language, source *goi18n.Message) []string {
	if !isPlural(source) {
		return []string{otherForm}
	}
	switch lang {
	case i18n.English, i18n.French:
		return []string{oneForm, otherForm}
	}
	return []string{otherForm}
}

func formText(message *goi18n.Message, form string) string {
	forms := map[string]string{oneForm: message.One, otherForm: message.Other}
	return forms[form]
}

func messageFields(message *goi18n.Message) (map[string]struct{}, error) {
	names := make(map[string]struct{})
	for _, text := range []string{message.Zero, message.One, message.Two, message.Few, message.Many, message.Other} {
		if text == "" {
			continue
		}
		if err := collectTemplateFields(text, message.LeftDelim, message.RightDelim, names); err != nil {
			return nil, err
		}
	}
	return names, nil
}

func collectTemplateFields(text, leftDelim, rightDelim string, names map[string]struct{}) error {
	trees, err := parse.Parse("message", text, leftDelim, rightDelim)
	if err != nil {
		return fmt.Errorf("template: %w", err)
	}
	for _, tree := range trees {
		if err := collectNodeFields(tree.Root, names); err != nil {
			return err
		}
	}
	return nil
}

func collectNodeFields(node parse.Node, names map[string]struct{}) error {
	switch typed := node.(type) {
	case *parse.ListNode:
		for _, child := range typed.Nodes {
			if err := collectNodeFields(child, names); err != nil {
				return err
			}
		}
		return nil
	case *parse.TextNode:
		return nil
	case *parse.ActionNode:
		return collectPipeField(typed.Pipe, names)
	}
	return fmt.Errorf("%s: %w", node, errOnlyFields)
}

func collectPipeField(pipe *parse.PipeNode, names map[string]struct{}) error {
	if len(pipe.Decl) > 0 || len(pipe.Cmds) != 1 || len(pipe.Cmds[0].Args) != 1 {
		return fmt.Errorf("%s: %w", pipe, errOnlyFields)
	}
	field, isField := pipe.Cmds[0].Args[0].(*parse.FieldNode)
	if !isField || len(field.Ident) != 1 {
		return fmt.Errorf("%s: %w", pipe, errOnlyFields)
	}
	names[field.Ident[0]] = struct{}{}
	return nil
}

func sortedKeys[V any](entries map[string]V) []string {
	return slices.Sorted(maps.Keys(entries))
}

func sortedNames(names map[string]struct{}) []string {
	return slices.Sorted(maps.Keys(names))
}
