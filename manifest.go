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
	// Models declares the models this backend knows how to run. Empty for
	// non-backend plugins. The host aggregates this field across all compiled
	// backends to form the catalog offered to the user — without ever
	// instantiating a backend, which the app needs before a session exists.
	Models []ModelSpec
	// AttachmentHosts are the hostnames this gateway's Attachment URLs may point
	// at — its own CDN, and nothing else. The host downloads those URLs, so it
	// pins them to an allowlist rather than fetching whatever a remote author put
	// in a message; declaring the list here is how it gets one without knowing
	// what a "Discord CDN" is. Empty means the gateway hands over no https
	// attachments, which is the safe default: nothing is downloaded.
	AttachmentHosts []string
}
