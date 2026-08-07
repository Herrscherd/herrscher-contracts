package contracts_test

import (
	"context"
	"testing"
	"testing/fstest"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

// contributor is a plugin-side type proving both contribution points are
// satisfiable from outside the package, which is the only place they are ever
// implemented.
type contributor struct{}

func (contributor) Commands() []contracts.Cmd {
	return []contracts.Cmd{contracts.New("channel", "read").Help("read a channel").Do(
		func(context.Context, contracts.Input) (string, error) { return "", nil },
	)}
}

func (contributor) Delete(context.Context, string, string) error       { return nil }
func (contributor) Edit(context.Context, string, string, string) error { return nil }

var (
	_ contracts.CommandSource = contributor{}
	_ contracts.MessageEditor = contributor{}
)

// A plugin carries its skills as a plain fs.FS, so they exist without the
// plugin ever being instantiated — a gateway with no token still ships its
// playbook.
func TestPluginCarriesSkillsWithoutInstantiating(t *testing.T) {
	p := contracts.Plugin{
		Manifest: contracts.Manifest{Kind: "fake", Category: contracts.CategoryGateway},
		Skills:   fstest.MapFS{"demo/SKILL.md": &fstest.MapFile{Data: []byte("# demo")}},
	}
	if p.Skills == nil {
		t.Fatal("a plugin must be able to carry skills")
	}
	if _, err := p.Skills.Open("demo/SKILL.md"); err != nil {
		t.Fatalf("the carried skill must be readable: %v", err)
	}
	// A plugin that contributes nothing leaves it nil, and that must stay legal.
	if bare := (contracts.Plugin{}); bare.Skills != nil {
		t.Fatal("contributing no skills must be the zero value")
	}
}
