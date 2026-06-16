package contracts

import "context"

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
)

// Plugin is what a plugin declares about itself. Exactly one factory is non-nil,
// consistent with Manifest.Category.
type Plugin struct {
	Manifest Manifest
	Gateway  GatewayFactory // set iff Manifest.Category == CategoryGateway
	Backend  BackendFactory // set iff Manifest.Category == CategoryBackend
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

func (r *Registry) Gateways() []Plugin { return r.byCategory(CategoryGateway) }
func (r *Registry) Backends() []Plugin { return r.byCategory(CategoryBackend) }

// Default is the global registry plugins self-register into via init(). Precedent
// in the stdlib: image.RegisterFormat, database/sql.Register. A blank import of a
// plugin package (in the host's generated plugins.go) triggers its init().
var Default Registry

// Register adds a plugin to the Default registry.
func Register(p Plugin) { Default.Register(p) }
