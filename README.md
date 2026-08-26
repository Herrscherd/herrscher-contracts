# herrscher-contracts

**The port package.** Every Herrscher plugin implements interfaces declared here,
and the core consumes only these.

It is types and interfaces plus a few thin helpers. No runtime, no daemon, no
plugin logic. Zero dependencies: Go standard library only, Go 1.25.

```go
import contracts "github.com/Herrscherd/herrscher-contracts"
```

## Ports

A **required** port must be satisfied for a plugin of that category to work. An
**optional** one is a capability: the host type-asserts for it and degrades when
it is absent.

### Model edge

| Port | File | Kind | What it does |
|---|---|---|---|
| `Backend` | `backend.go` | required | `Respond(ctx, Prompt, onEvent)` returns a reply, plus `Close` |
| `ResumeAware` | `backend.go` | optional | exposes an opaque `ResumeToken()` the host persists and feeds back |
| `ChoiceAware` | `backend.go` | optional | `PendingChoice()`: the turn is blocked on an interactive pick |
| `ChoiceInjector` | `backend.go` | optional | `InjectChoice` answers that pick out of band |
| `SkillNative` | `skill.go` | optional | `NativeSkills()`: the backend loads skills itself, so the host skips injection |

### Channel edge

| Port | File | Kind | What it does |
|---|---|---|---|
| `Gateway` | `gateway.go` | required | `Manifest`, `Post`, `Reply`, `React`, `Menu` |
| `ChannelReader` | `host.go` | optional | `Enabled`, `DefaultChannel`, `EnsureChannel`, `Read`, `Unreact`, `UpsertStatusMessage` |
| `ChannelAdmin` | `host.go` | optional | `Kind`, `CreateUnder`, `ForumPost`, `Archive`, `Send`, `ChannelRef` |
| `MenuRouter` | `host.go` | optional | `RouteMenu`: menu picks return to a named route, not to the channel |
| `Prober` | `host.go` | optional | `Probe` round-trip latency, for liveness |
| `EventSink` | `event.go` | optional | `Emit(Event)`: the gateway renders the live turn stream itself |
| `RoutedEventSink` | `event.go` | optional | `EmitTo(Conversation, Event)`, the multi-session variant, preferred over `EventSink` |
| `Foreground` | `foreground.go` | optional | `RunForeground`: the gateway owns the main thread (a TUI). At most one |
| `SessionControlReceiver` | `session_control.go` | optional | `BindSessionControl` receives the host's session-control handle |

### Memory

| Port | File | Kind | What it does |
|---|---|---|---|
| `Memory` | `memory.go` | required | `Recall`, `Record`, `Search`, `Links`, `Unlink`, `Close`. Passive verbs only |
| `Provisioner` | `memory.go` | optional | `EnsureProject`, `EnsureAgent` create scope-root nodes |
| `Locator` | `memory.go` | optional | `Locate` returns openable URIs (`Location{Obsidian, File}`) for a node |
| `Deleter` | `memory.go` | optional | `Delete` forgets a node by key, idempotently |

### Conversation policy

| Port | File | Kind | What it does |
|---|---|---|---|
| `Orchestrator` | `orchestrator.go` | required | `Context`, `Observe`, `Consolidate`, `Close`. Session-scoped |
| `CurationHook` | `memory.go` | required | `Consolidate`, embedded in `Orchestrator` |
| `TurnReactor` | `turn_reactor.go` | optional | `React` handles in-band memory markers and strips them |

### Implemented by the host, consumed by plugins

| Port | File | What it does |
|---|---|---|
| `SessionControl` | `session_control.go` | `Dispatch`, `Create`, `Close`, `Sessions`, `Scrollback`, `Resume`, `Interrupt` |
| `Coordinator` | `coordinator.go` | `Handoff`, `Delegate`, `Report`, `Merge`, `Seal`, `FanOut`, `Route` |
| `RosterProvider` | `roster.go` | `Agents()`, the agents a session may delegate to |
| `Liveness` | `liveness.go` | `HeartbeatAck`, the sink for a gateway's keepalive |

### Types and helpers, not ports

| Name | File | What it is |
|---|---|---|
| `ModelSpec`, `Route`, `RoutePolicy` | `models.go` | the model catalog a backend declares, its route, and the policy that bounds it, plus `ValidateModels`, `FilterModels`, `GatewayCreds` and `NewGatewayCreds` |
| `MergeEnv`, `ParseEnvSetting`, `EncodeEnvSetting` | `spawnenv.go` | encode and decode the per-session environment the host injects at spawn, and the `Env*` key constants both sides must use |
| `WithPrincipal`, `PrincipalFrom` | `principal.go` | who is asking, carried on the context |

## Registration and config

A plugin declares a `Plugin{Manifest, <one factory>}` in `init()` and calls
`Register`.

Exactly one of `GatewayFactory`, `BackendFactory`, `MemoryFactory` or
`OrchestratorFactory` is set, matching `Manifest.Category`. The exception is
`CategorySkills`, which has no port to build and so declares no port factory. A
gateway factory returns a `GatewaySet{Gateway, Reader, Admin, Prober}`: one
channel, with the optional ports left nil.

`Manifest.Config []Setting` declares each env-bound setting. `Resolve` builds a
validated `PluginConfig` and fails startup naming every missing required key.

`Manifest.Status` is the plugin's own maturity claim: `StatusLive` (the zero
value), `StatusWIP`, `StatusExperimental` or `StatusDeprecated`. It is
descriptive only and never affects discovery.

`Manifest.AttachmentHosts []string` names the hosts a gateway's attachment URLs
may point at. The host pins its downloads to that allowlist, so a gateway that
declares none has none downloaded.

### Skills are orthogonal to the category

`Plugin.Skills` is a `SkillsFactory`. Any plugin may carry the playbooks that
teach an agent to use what it contributes, and a plugin of `CategorySkills`
carries nothing else.

The host calls it apart from the port factory, so a gateway that never
instantiates for want of a token still ships its playbook. A factory that returns
an error installs nothing, which is how a plugin declines on a machine that lacks
the tool its playbook describes.

## Model routing

A backend declares the models it knows how to run in `Manifest.Models`
(`[]ModelSpec{ID, Label, Arg, Efforts, Route, InputPrice}`), so the catalog is
readable without instantiating anything.

`ID` is what the host persists in session state. `Arg` is only what goes on the
CLI's `--model`, and the two often differ.

`ValidateModels(kind, models)` is what the host calls at startup. An empty
`ID`, `Label` or `Arg`, a duplicate `ID`, or an unknown route is a startup
failure, not a silently wrong selector.

`Route` is binary. `RouteNative` means the vendor CLI uses the login present on
the machine. `RouteGateway` means it is pointed at the product's own paid
account.

`RoutePolicy` bounds which of them a build serves: `PolicyAll` (the zero value,
unchanged behaviour) or `PolicyGatewayOnly`, the public build. Under
`gateway-only` a native model is not hidden, it is **absent**. `FilterModels`
drops it, so it cannot be listed, selected, persisted or resumed.
`RoutePolicy.Allows(Route)` is the single predicate.

`GatewayCreds` carries the base URL and token pair a gateway route needs. Its
fields are **unexported** and `NewGatewayCreds` is the only constructor, which
refuses a blank half. A base URL without a token would send traffic to the
gateway while billing it to the user's own subscription, the shape Anthropic
forbids third-party developers from producing. Making that state unrepresentable
is stronger than testing for it afterwards. The zero value stays constructible
but is empty on both sides, so it is never a half-pair.

### The spawn environment

`MergeEnv`, `ParseEnvSetting` and `EncodeEnvSetting` carry the environment the
host injects into a spawned child. They live here, and not in each backend,
because the host encodes and the backends decode. Split across repos, the two
halves could drift with no test able to see it.

`MergeEnv` **replaces** an inherited key rather than appending a second entry for
it.

The variable names are constants for the same reason: `EnvAnthropicBaseURL`,
`EnvAnthropicAuthToken`, `EnvOpenAIBaseURL`, `EnvNeubloxToken`. A rename spelled
as a literal on both sides compiles green everywhere and fails only at run time,
silently, by running natively on the user's own login.

## Who is asking

`WithPrincipal(ctx, "chat:1234")` marks a context as carrying the caller's
identity, and `PrincipalFrom(ctx)` reads it back. A gateway sets it from the
identity its platform already gives it, prefixed with its own kind. The daemon
reads it to decide what that caller may run.

The key is unexported, so `WithPrincipal` is the only way to write one. A context
value keyed by a string would let any package in the process name a caller, which
is the opposite of what naming a caller is for.

It is additive. A gateway built before this existed names nobody, and the
daemon's answer to an unnamed caller is to leave the decision where it already
was, so an old plugin keeps working exactly as it did. An empty principal is not
a name: it yields the same context back, so a gateway with no identity to offer
does not have to know whether passing `""` would be worse than not calling at
all.

## Helpers

`Degrade(Gateway)` wraps a gateway so callers always invoke the rich method:
`Reply` falls back to `Post`, `Menu` to a numbered-list `Post`, `React` to a
no-op, and `BindSessionControl` still forwards.

`MemoryScope{Project, Agent}` plus `RecordShared`, `RecordPrivate`,
`RecallScoped` and `RecallRelevant` implement shared-versus-private memory as a
policy over the plain `Memory` verbs.

`EnforceBudget`, `NextState` and `Score` back per-node size limits, node
lifecycle states, and relevance ranking.

## Further reading

- [Herrscher docs](https://github.com/Herrscherd/herrscher-docs), page
  `reference/ports`
