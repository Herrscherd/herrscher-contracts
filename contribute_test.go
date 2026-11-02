package contracts_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

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
	if bare := (contracts.Plugin{}); bare.Skills != nil {
		t.Fatal("contributing no skills must be the zero value")
	}
}

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
