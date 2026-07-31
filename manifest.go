package contracts

type Category string

const (
	CategoryGateway      Category = "gateway"
	CategoryBackend      Category = "backend"
	CategoryMemory       Category = "memory"
	CategoryOrchestrator Category = "orchestrator"
)

// Capabilities are announced by a plugin. The degrading decorator reads them to
// rabat unsupported actions. This is the single source of truth (no separate
// Capabilities() method on the port).
type Capabilities struct {
	Reactions   bool
	SelectMenus bool
	Replies     bool
}

// Status is a plugin's own claim about its maturity. It exists so maturity is
// declared at the source rather than inferred by a reader from test coverage or
// the presence of a CI file — two signals that had drifted apart in practice.
type Status string

const (
	// StatusLive is the zero value: a plugin that declares nothing is assumed
	// ready to run, which is what every plugin predating this field was.
	StatusLive Status = ""
	// StatusWIP compiles and registers, but a port it advertises may be
	// unimplemented or unstable.
	StatusWIP Status = "wip"
	// StatusExperimental is complete but unproven — its shape may still change.
	StatusExperimental Status = "experimental"
	// StatusDeprecated still works but is scheduled for removal.
	StatusDeprecated Status = "deprecated"
)

// String renders a Status for display, spelling out the zero value.
func (s Status) String() string {
	if s == StatusLive {
		return "live"
	}
	return string(s)
}

// Manifest is what a plugin announces about itself. In Phase 1 this becomes the
// payload of the NATS self-registration; the shape stays identical.
type Manifest struct {
	Kind         string
	Category     Category
	Capabilities Capabilities
	// Status is the plugin's maturity claim. The zero value means live, so
	// adding this field left every existing manifest correct as written.
	Status Status
	// Config declares every setting the plugin reads, with the env var it binds
	// from and whether it is required. The host resolves a PluginConfig from this
	// (see Resolve) — it never needs to know a plugin's keys itself.
	Config []Setting
}
