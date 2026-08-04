# herrscher-contracts

**The port package.** Every Herrscher plugin implements interfaces declared here,
and the core consumes only these. It is types and interfaces plus a few thin
helpers — no runtime, no daemon, no plugin logic. Zero dependencies: Go standard
library only, Go 1.25.

```go
import contracts "github.com/Herrscherd/herrscher-contracts"
```

## Ports

Required ports must be satisfied for a plugin of that category to work.
Optional ones are capabilities: the host type-asserts and degrades when absent.

| Port | File | Kind | What it does |
|------|------|------|--------------|
| **Model edge** | | | |
| `Backend` | `backend.go` | required | `Respond(ctx, Prompt, onEvent)` → reply; `Close`. |
| `ResumeAware` | `backend.go` | optional | Exposes an opaque `ResumeToken()` the host persists and feeds back. |
| `ChoiceAware` | `backend.go` | optional | `PendingChoice()` — the turn is blocked on an interactive pick. |
| `ChoiceInjector` | `backend.go` | optional | `InjectChoice` answers that pick out of band. |
| `SkillNative` | `skill.go` | optional | `NativeSkills()` — backend loads skills itself; host skips injection. |
| **Channel edge** | | | |
| `Gateway` | `gateway.go` | required | `Manifest`, `Post`, `Reply`, `React`, `Menu`. |
| `ChannelReader` | `host.go` | optional | `Enabled`, `DefaultChannel`, `EnsureChannel`, `Read`, `Unreact`, `UpsertStatusMessage`. |
| `ChannelAdmin` | `host.go` | optional | `Kind`, `CreateUnder`, `ForumPost`, `Archive`, `Send`, `ChannelRef`. |
| `MenuRouter` | `host.go` | optional | `RouteMenu` — menu picks return to a named route, not the channel. |
| `Prober` | `host.go` | optional | `Probe` round-trip latency for liveness. |
| `EventSink` | `event.go` | optional | `Emit(Event)` — gateway renders the live turn stream itself. |
| `RoutedEventSink` | `event.go` | optional | `EmitTo(Conversation, Event)` — multi-session variant, preferred over `EventSink`. |
| `Foreground` | `foreground.go` | optional | `RunForeground` — gateway owns the main thread (TUI); at most one. |
| `SessionControlReceiver` | `session_control.go` | optional | `BindSessionControl` — receives the host's session-control handle. |
| **Memory** | | | |
| `Memory` | `memory.go` | required | `Recall`, `Record`, `Search`, `Links`, `Unlink`, `Close`. Passive verbs only. |
| `Provisioner` | `memory.go` | optional | `EnsureProject`, `EnsureAgent` — create scope-root nodes. |
| `Locator` | `memory.go` | optional | `Locate` — openable URIs (`Location{Obsidian, File}`) for a node. |
| `Deleter` | `memory.go` | optional | `Delete` — forget a node by key; idempotent. |
| **Conversation policy** | | | |
| `Orchestrator` | `orchestrator.go` | required | `Context`, `Observe`, `Consolidate`, `Close`. Session-scoped. |
| `CurationHook` | `memory.go` | required | `Consolidate` — embedded in `Orchestrator`. |
| `TurnReactor` | `turn_reactor.go` | optional | `React` — handles in-band memory markers and strips them. |
| **Implemented by the host, consumed by plugins** | | | |
| `SessionControl` | `session_control.go` | host | `Dispatch`, `Create`, `Close`, `Sessions`, `Scrollback`, `Resume`, `Interrupt`. |
| `Coordinator` | `coordinator.go` | host | `Handoff`, `Delegate`, `Report`, `Merge`, `Seal`, `FanOut`, `Route`. |
| `RosterProvider` | `roster.go` | host | `Agents()` — the agents a session may delegate to. |
| `Liveness` | `liveness.go` | host | `HeartbeatAck` — sink for a gateway's keepalive. |
| **Model routing (types & helpers, not ports)** | | | |
| `ModelSpec`, `Route`, `RoutePolicy` | `models.go` | types | The model catalog a backend declares, its route, and the policy that bounds it; plus `ValidateModels`, `FilterModels`, `GatewayCreds`/`NewGatewayCreds`. |
| `MergeEnv`, `ParseEnvSetting`, `EncodeEnvSetting` | `spawnenv.go` | helpers | Encode/decode the per-session environment the host injects at spawn, and the `Env*` key constants both sides must use. |

## Registration & config

A plugin declares a `Plugin{Manifest, <one factory>}` in `init()` and calls
`Register`. Exactly one of `GatewayFactory` / `BackendFactory` / `MemoryFactory`
/ `OrchestratorFactory` is set, matching `Manifest.Category`. A gateway factory
returns a `GatewaySet{Gateway, Reader, Admin, Prober}` — one channel, optional
ports nil. `Manifest.Config []Setting` declares each env-bound setting;
`Resolve` builds a validated `PluginConfig` and fails startup naming every
missing required key. `Manifest.Status` is the plugin's own maturity claim
(`StatusLive` — the zero value — `StatusWIP`, `StatusExperimental`,
`StatusDeprecated`); it is descriptive only and never affects discovery.
`Manifest.AttachmentHosts []string` names the hosts a gateway's attachment URLs
may point at; the host pins its downloads to that allowlist, so a gateway that
declares none has none downloaded.

## Model routing

A backend declares the models it knows how to run in `Manifest.Models`
(`[]ModelSpec{ID, Label, Arg, Efforts, Route, InputPrice}`), so the catalog is
readable without instantiating anything. `ID` is what the host persists in
session state — `Arg` is only what goes on the CLI's `--model`, and the two
often differ. `ValidateModels(kind, models)` is what the host calls at startup:
an empty ID/Label/Arg, a duplicate ID, or an unknown route is a startup
failure, not a silently wrong selector.

`Route` is binary: `RouteNative` (the vendor CLI uses the login present on the
machine) or `RouteGateway` (it is pointed at the product's own paid account).
`RoutePolicy` bounds which of them a build serves — `PolicyAll` (the zero
value, unchanged behavior) or `PolicyGatewayOnly`, the public build. Under
`gateway-only` a native model is not hidden, it is **absent**: `FilterModels`
drops it, so it cannot be listed, selected, persisted, or resumed.
`RoutePolicy.Allows(Route)` is the single predicate.

`GatewayCreds` carries the (base URL, token) pair a gateway route needs. Its
fields are **unexported** and `NewGatewayCreds` is the only constructor, which
refuses a blank half: a base URL without a token would send traffic to the
gateway while billing it to the user's own subscription — the shape Anthropic
forbids third-party developers from producing. Making that state
unrepresentable is stronger than testing for it afterwards. The zero value
stays constructible but is empty on both sides, so it is never a half-pair.

`MergeEnv`, `ParseEnvSetting` and `EncodeEnvSetting` carry the environment the
host injects into a spawned child. They live here, not in each backend,
because the host encodes and the backends decode: split across repos, the two
halves could drift with no test able to see it. `MergeEnv` **replaces** an
inherited key rather than appending a second entry for it. The variable names
themselves are constants for the same reason — `EnvAnthropicBaseURL`,
`EnvAnthropicAuthToken`, `EnvOpenAIBaseURL`, `EnvNeubloxToken` — since a
rename spelled as a literal on both sides compiles green everywhere and fails
only at run time, silently, by running natively on the user's own login.

## Helpers

`Degrade(Gateway)` wraps a gateway so callers always invoke the rich method:
`Reply`→`Post`, `Menu`→numbered-list `Post`, `React`→no-op, and
`BindSessionControl` still forwards. `MemoryScope{Project, Agent}` plus
`RecordShared` / `RecordPrivate` / `RecallScoped` / `RecallRelevant` implement
shared-vs-private memory as a policy over the plain `Memory` verbs.
`EnforceBudget`, `NextState`, and `Score` back per-node size limits, node
lifecycle states, and relevance ranking.

## Further reading

- [Herrscher docs](https://github.com/Herrscherd/herrscher-docs)
