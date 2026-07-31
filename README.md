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

## Registration & config

A plugin declares a `Plugin{Manifest, <one factory>}` in `init()` and calls
`Register`. Exactly one of `GatewayFactory` / `BackendFactory` / `MemoryFactory`
/ `OrchestratorFactory` is set, matching `Manifest.Category`. A gateway factory
returns a `GatewaySet{Gateway, Reader, Admin, Prober}` — one channel, optional
ports nil. `Manifest.Config []Setting` declares each env-bound setting;
`Resolve` builds a validated `PluginConfig` and fails startup naming every
missing required key.

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
