package contracts

import (
	"context"
	"io/fs"
)

// PluginConfig is the neutral settings bag a factory receives at startup. The
// host fills it from env > config.json; a plugin reads only the keys it knows
// (a Discord gateway "token", a Claude backend "cmd"/"model", …).
type PluginConfig struct{ Settings map[string]string }

// Get returns a setting (empty if absent), nil-map safe.
func (c PluginConfig) Get(key string) string {
	if c.Settings == nil {
		return ""
	}
	return c.Settings[key]
}

// GatewaySet is the coherent channel a gateway plugin provides to the host: the
// outbound messaging port plus every port the daemon and bridge loops need,
// all built from one PluginConfig. Optional ports (Reader, Admin, Prober) may be
// nil; the host degrades. This is what lets "add a plugin = blank import +
// rebuild": the host instantiates a GatewaySet from the registry and drives it
// without any plugin-specific wiring.
type GatewaySet struct {
	Gateway Gateway
	Reader  ChannelReader
	Admin   ChannelAdmin
	Prober  Prober
}

// GatewayFactory and BackendFactory build a live plugin instance from runtime
// config. Registering a factory (not an instance) is what lets a plugin announce
// itself in init() before any token/command is known — the xcaddy pattern.
type (
	GatewayFactory func(ctx context.Context, cfg PluginConfig) (GatewaySet, error)
	BackendFactory func(ctx context.Context, cfg PluginConfig) (Backend, error)
	MemoryFactory  func(ctx context.Context, cfg PluginConfig) (Memory, error)
	// OrchestratorFactory builds a session-scoped Orchestrator. Unlike the other
	// factories it also receives the Memory port it composes (nil when no memory
	// is wired); the session name arrives via cfg (key "session").
	OrchestratorFactory func(ctx context.Context, cfg PluginConfig, mem Memory) (Orchestrator, error)
)

// Plugin is what a plugin declares about itself. Exactly one factory is non-nil,
// consistent with Manifest.Category.
type Plugin struct {
	Manifest     Manifest
	Gateway      GatewayFactory      // set iff Manifest.Category == CategoryGateway
	Backend      BackendFactory      // set iff Manifest.Category == CategoryBackend
	Memory       MemoryFactory       // set iff Manifest.Category == CategoryMemory
	Orchestrator OrchestratorFactory // set iff Manifest.Category == CategoryOrchestrator
	// Skills are the playbooks teaching an agent to use what this plugin
	// contributes, installed by the host only when the plugin is in the build —
	// so a Discord playbook never sits in the context of a machine that has no
	// Discord. A static field and not a method on the instance: a gateway missing
	// its credentials never instantiates, and it must still ship its playbook.
	// Nil when a plugin contributes none.
	Skills fs.FS
}

// CommandSource is an optional capability of a live plugin instance: the verbs
// it contributes to the daemon's own command registry. The host namespaces them
// under the plugin's Manifest Kind, so two gateways declaring the same path do
// not collide. A Cmd's Run may close over anything the plugin holds — the
// registry only ever sees a Cmd, which is what keeps the core agnostic.
type CommandSource interface {
	Commands() []Cmd
}

// Registry collects plugins and queries them by category. Plugins self-register
// into Default from their init(); the host queries it at startup. In Phase 1 the
// in-process registration becomes NATS self-registration with the same Manifest
// and the same query surface.
type Registry struct{ plugins []Plugin }

func (r *Registry) Register(p Plugin) { r.plugins = append(r.plugins, p) }
func (r *Registry) Plugins() []Plugin { return r.plugins }

func (r *Registry) byCategory(c Category) []Plugin {
	var out []Plugin
	for _, p := range r.plugins {
		if p.Manifest.Category == c {
			out = append(out, p)
		}
	}
	return out
}

func (r *Registry) Gateways() []Plugin      { return r.byCategory(CategoryGateway) }
func (r *Registry) Backends() []Plugin      { return r.byCategory(CategoryBackend) }
func (r *Registry) Memories() []Plugin      { return r.byCategory(CategoryMemory) }
func (r *Registry) Orchestrators() []Plugin { return r.byCategory(CategoryOrchestrator) }

// Default is the global registry plugins self-register into via init(). Precedent
// in the stdlib: image.RegisterFormat, database/sql.Register. A blank import of a
// plugin package (in the host's generated plugins.go) triggers its init().
var Default Registry

// Register adds a plugin to the Default registry.
func Register(p Plugin) { Default.Register(p) }
