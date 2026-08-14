package contracts_test

import (
	"context"
	"errors"
	"io/fs"
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

// A plugin's skills are reachable without its port ever being built — a gateway
// with no token still ships its playbook. The two factories are independent, and
// this test fails the moment the host is asked to build one to get the other.
func TestPluginSkillsDoNotRequireBuildingThePort(t *testing.T) {
	built := false
	p := contracts.Plugin{
		Manifest: contracts.Manifest{Kind: "fake", Category: contracts.CategoryGateway},
		Gateway: func(context.Context, contracts.PluginConfig) (contracts.GatewaySet, error) {
			built = true
			return contracts.GatewaySet{}, errors.New("no token")
		},
		Skills: func(context.Context, contracts.PluginConfig) (fs.FS, error) {
			return fstest.MapFS{"demo/SKILL.md": &fstest.MapFile{Data: []byte("# demo")}}, nil
		},
	}
	tree, err := p.Skills(context.Background(), contracts.PluginConfig{})
	if err != nil {
		t.Fatalf("the skills factory must stand on its own: %v", err)
	}
	if _, err := tree.Open("demo/SKILL.md"); err != nil {
		t.Fatalf("the carried skill must be readable: %v", err)
	}
	if built {
		t.Error("reading a plugin's skills must not build its port")
	}
	// A plugin that contributes nothing leaves it nil, and that must stay legal.
	if bare := (contracts.Plugin{}); bare.Skills != nil {
		t.Fatal("contributing no skills must be the zero value")
	}
}

// A skills plugin declines when the tool its playbook describes is not on the
// machine, and the host installs nothing rather than teaching an agent about a
// capability it does not have.
func TestASkillsPluginMayDecline(t *testing.T) {
	p := contracts.Plugin{
		Manifest: contracts.Manifest{Kind: "superset", Category: contracts.CategorySkills},
		Skills: func(context.Context, contracts.PluginConfig) (fs.FS, error) {
			return nil, errors.New("no Superset install at /nowhere")
		},
	}
	if _, err := p.Skills(context.Background(), contracts.PluginConfig{}); err == nil {
		t.Fatal("a skills factory must be able to refuse")
	}
}
