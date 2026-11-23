package contracts

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestWireNameListsAreTheJSONTagsOfTheirStructs(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		typeName string
		listed   []string
	}{
		{name: "Event", file: "event.go", typeName: "Event", listed: eventWireFields()},
		{name: "ApprovalRequest", file: "approval.go", typeName: "ApprovalRequest", listed: approvalRequestWireFields()},
		{name: "ApprovalAnswer", file: "approval.go", typeName: "ApprovalAnswer", listed: approvalAnswerWireFields()},
		{name: "noticeWire", file: "notice_json.go", typeName: "noticeWire", listed: noticeWireFields()},
		{name: "noticeWireIn", file: "notice_json.go", typeName: "noticeWireIn", listed: noticeWireFields()},
		{name: "ToolCall", file: "tool_call.go", typeName: "ToolCall", listed: toolCallWireFields()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := jsonWireNames(t, tt.file, tt.typeName)
			if !sameNames(tags, tt.listed) {
				t.Fatalf("json tags of %s = %v, listed = %v", tt.typeName, tags, tt.listed)
			}
		})
	}
	t.Run("malformed tags", func(t *testing.T) {
		for _, tag := range []string{``, `xml:"name"`, `json:"name`} {
			if name, ok := parseJSONWireName(tag); ok {
				t.Fatalf("parseJSONWireName(%q) = %q, want no name", tag, name)
			}
		}
		if name, ok := parseJSONWireName(`json:"kind,omitempty"`); !ok || name != "kind" {
			t.Fatalf("parseJSONWireName of a well-formed tag = %q, %v; want kind, true", name, ok)
		}
	})
}

func TestWireNameListsAreAllMarshaled(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		typeName string
		filled   any
	}{
		{name: "Event", file: "event.go", typeName: "Event", filled: richEvent()},
		{name: "ApprovalRequest", file: "approval.go", typeName: "ApprovalRequest", filled: ApprovalRequest{ID: "1", Tool: "bash", Subject: "ls"}},
		{name: "ApprovalAnswer", file: "approval.go", typeName: "ApprovalAnswer", filled: ApprovalAnswer{ID: "1", Decision: DecisionDeny, Reason: "no", Refusal: RefusalTooMany}},
		{name: "noticeWire", file: "notice_json.go", typeName: "noticeWire", filled: noticeWire{Code: 1, Args: []Arg{Text("name", "lead")}}},
		{name: "noticeWireIn", file: "notice_json.go", typeName: "noticeWireIn", filled: noticeWireIn{Code: 1, Args: json.RawMessage("[]")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := marshaledKeys(t, tt.filled)
			for _, tag := range jsonWireNames(t, tt.file, tt.typeName) {
				if _, ok := keys[tag]; !ok {
					t.Fatalf("the filled %s marshals no %q key", tt.typeName, tag)
				}
			}
		})
	}
}

func sameNames(got, want []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(want)))
}

func marshaledKeys(t *testing.T, value any) map[string]json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var keys map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &keys); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", encoded, err)
	}
	return keys
}

func jsonWireNames(t *testing.T, file, typeName string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("ParseFile(%s) error = %v", file, err)
	}
	var names []string
	for _, field := range structFields(t, parsed, typeName) {
		names = append(names, jsonWireName(t, field))
	}
	return names
}

func structFields(t *testing.T, parsed *ast.File, typeName string) []*ast.Field {
	t.Helper()
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatalf("%s is not a struct", typeName)
			}
			return structType.Fields.List
		}
	}
	t.Fatalf("type %s not found", typeName)
	return nil
}

func jsonWireName(t *testing.T, field *ast.Field) string {
	t.Helper()
	if field.Tag == nil {
		t.Fatal("a field has no struct tag")
	}
	tag, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		t.Fatalf("struct tag %s: %v", field.Tag.Value, err)
	}
	name, ok := parseJSONWireName(tag)
	if !ok {
		t.Fatalf("struct tag %s has no well-formed json key", tag)
	}
	return name
}

func parseJSONWireName(tag string) (string, bool) {
	_, afterKey, found := strings.Cut(tag, `json:"`)
	if !found {
		return "", false
	}
	quoted, _, closed := strings.Cut(afterKey, `"`)
	if !closed {
		return "", false
	}
	name, _, _ := strings.Cut(quoted, ",")
	return name, true
}
