# herrscher-contracts

**The ports.** This module is the shared vocabulary of the Herrscher platform: the
interfaces and the neutral data types that every other repo agrees on. It is the
narrow waist that lets the core stay ignorant of *which* chat platform delivers a
message and *which* model answers it.

- **Zero dependencies** — only the Go standard library.
- **Almost zero logic** — type definitions plus two thin helpers (`Degrade`, `Resolve`).
- **Consumer-defined ports** — the interfaces are written from the point of view of
  the code that *calls* them (the core), not the code that implements them. Adapters
  satisfy them structurally; they never import each other.

> Part of the Herrscher family: **contracts** ·
> [herrscher](https://github.com/Herrscherd/herrscher) (the umbrella binary: core +
> daemon + CLI + bridge) · [discord-gateway](https://github.com/Herrscherd/herrscher-discord-gateway).

---

## Why a separate repo

Hexagonal architecture lives or dies on one rule: **the domain depends on
abstractions, never on concretions.** If the core imported the Discord adapter or a
model backend, swapping either would mean editing the core. Instead every plugin and
the core depend *only* on `contracts`:

```
core ───► contracts ◄─── claude-backend
                 ▲
                 └────── discord-gateway
```

Because `contracts` imports nothing, the dependency arrows can never form a cycle,
and the core holds zero knowledge of any specific model or platform.

---

## The two edges

Herrscher routes between two edges. Each is a port in this package.

### Backend port — the model edge (`backend.go`)

Turns one inbound prompt into a reply, optionally streaming intermediate progress.
The host neither knows nor cares which model answers.

```go
type Backend interface {
    Respond(ctx context.Context, p Prompt, onEvent func(BackendEvent)) (string, error)
    Close() error
}
```

| Type | Purpose |
|------|---------|
| `Prompt` | Platform-neutral input: `Content`, `Author`, `MessageID`, `ChannelID`, `Attachments []string` (local file paths). |
| `BackendEvent` | One intermediate occurrence in a turn: `Kind` (`"tool"`/`"text"`/`"result"`/`"reset"`), plus `Tool`, `Detail`, `Cost`, `IsError`. |
| `PendingChoice` | An interactive selection a backend is blocked on (`Question` + `[]Choice`). |

Two optional capability interfaces let a backend pause on an interactive choice
(e.g. a tool-permission prompt), serialized with its normal turns:

```go
type ChoiceAware interface  { PendingChoice() (PendingChoice, bool) }
type ChoiceInjector interface { InjectChoice(ctx context.Context, value string) (string, error) }
```

### Gateway port — the channel edge (`gateway.go`)

A chat-platform plugin. The manager always calls the rich method; the degrading
decorator downgrades any action the plugin doesn't advertise.

```go
type Gateway interface {
    Manifest() Manifest
    Post(ctx context.Context, conv Conversation, text string) (MessageID, error)
    Reply(ctx context.Context, conv Conversation, replyTo MessageID, text string) (MessageID, error)
    React(ctx context.Context, conv Conversation, msg MessageID, emoji string) error
    Menu(ctx context.Context, conv Conversation, replyTo MessageID, prompt string, opts []Choice) error
}
```

`Inbound` is a message arriving from a gateway into the manager (conversation,
author, text, attachments, message id).

---

## Session control — the operator seam (`session_control.go`)

This is how a gateway drives the running daemon's session lifecycle **without the
core ever learning the gateway's command surface**. The hub implements
`SessionControl`; a gateway receives it (via `SessionControlReceiver`) and calls it
from its own command handlers, formatting its platform input (e.g. a Discord slash
interaction) into a **neutral argv**.

```go
type SessionControl interface {
    Dispatch(ctx context.Context, args []string) (string, error) // {"session","create","--name","foo"}
    Sessions() []SessionInfo
}

type SessionControlReceiver interface {
    BindSessionControl(SessionControl)
}
```

`Dispatch` runs one operator command over the same vocabulary the CLI uses and, for
lifecycle changes, brings the affected sessions live (or tears them down) in the
hub. `Sessions()` returns `[]SessionInfo` snapshots so a gateway can enumerate them
(e.g. to autocomplete a session name). After the hub is built, the host calls
`BindSessionControl` once on any Gateway that implements the receiver — the host
never knows what the gateway does with it.

---

## The session event bus (`event.go`)

`Event` is one message on the session bus: the bridge (a pure backend runner) emits
turn events (`human`/`status`/`chunk`/`reply`/`reset`) for the hub to fan out, and
the hub injects `input`/`pick` down to the bridge. One `Event` encodes to exactly one
JSON line. The terminal reply carries the turn `Cost` (USD) so a renderer can show it.

```go
type EventSink interface { Emit(Event) }
```

`EventSink` is an optional gateway capability: a gateway that renders the live turn
stream itself (the terminal TUI does) implements it and the hub fans every event to
it; a gateway without it is driven by the host's default renderer over the
Gateway/ChannelReader ports.

---

## Host-facing channel ports (`host.go`)

The always-on daemon needs more than outbound messaging. These ports split the
channel into small, independently-implementable surfaces; optional ones may be nil
and the host degrades.

| Port | Method(s) | Role |
|------|-----------|------|
| `Prober` | `Probe` | Cheap round-trip for health latency. |
| `ChannelReader` | `Enabled`, `DefaultChannel`, `EnsureChannel`, `Read`, `Unreact`, `UpsertStatusMessage` | Read/lifecycle side: presence, default conversation, bootstrap, history, reaction removal, the single self-updating status message. |
| `MenuRouter` | `RouteMenu` | Optional: post a menu whose picks route back to a named neutral route (e.g. a session), not the channel it lives in. |
| `ChannelAdmin` | `Kind`, `CreateUnder`, `ForumPost`, `Archive`, `Send` | Create/archive session channels and post into them. |

`Channel{ID, Name}` identifies a conversation a gateway can create or reuse.

---

## The command API (`command_api.go`)

`Cmd` is the one neutral command concept the platform exposes: a namespaced `Path`,
its `Params`, and an opaque `Run` handler. A command is declared once and a *format*
(the CLI today, a gateway binding later) resolves an invocation to it; the registry
that holds the `Cmd` stays agnostic of whatever `Run` closes over.

```go
cmd := contracts.New("session", "create").
    Help("create a session").
    Param("name", "session name", true).
    Do(func(ctx context.Context, in contracts.Input) (string, error) { ... })
```

`Input{Args, Rest}` is the parsed, format-agnostic invocation handed to a handler
(`Lookup`, `Get`, `Bool` accessors). `Param` declares one input; required params
missing at dispatch are an error.

---

## Conversation & messages

`Conversation{Gateway, ID}` (`conversation.go`) is an opaque, comparable address
into a chat platform — usable as a map key. `SessionID` and `MessageID` are string
aliases. `Choice{Label, Value}` and `Attachment{Filename, URL, ContentType, Size}`
are the neutral selection and file primitives. `Message` (`message.go`) is the
inbound envelope with author metadata and attachments.

---

## Memory & orchestration (`memory.go`, `orchestrator.go`)

`Memory` is the persistent-recall port: a storage-neutral knowledge graph of `Node`s
(stable `Key`, `Kind`, markdown `Title`/`Body`, typed `Links`, flat `Meta`) with
passive verbs `Recall`/`Record`/`Search`/`Links`/`Close`. The structural spine is
`KindOrganization → KindProject → KindRepo`/`KindServer` plus `KindAgent` (a
durable companion that anchors per-agent private memory), `KindDomain` (a
transverse area-of-concern root grouping projects topically), and documentary
kinds.

### Memory scope — shared vs private (P1, `memory_scope.go`)

`MemoryScope{Project, Agent}` is the sharing **policy over the existing graph** —
not a new port. A game's durable memory hangs under the shared `Project` node
(every agent of the game recalls it); an agent's learned skills hang under its
own `Agent` node (private to that agent). Composable helpers build on the plain
`Memory` verbs:

| Helper | Effect |
|--------|--------|
| `RecordShared(ctx, m, scope, n)` | upsert `n`, link it under the project root (visible to all agents). |
| `RecordPrivate(ctx, m, scope, n)` | upsert `n`, link it under the agent root; falls back to **shared** when `scope.Agent == ""` so a fact is never dropped. |
| `RecallScoped(ctx, m, scope, depth)` | merge the shared + private subgraphs, de-duplicated by node `Key` and by edge value; shared-only when there is no agent. |

`Orchestrator` is the conversation-policy port: session-scoped, the host drives it
around each turn (`Context` primes the next prompt, `Observe` records the finished
turn) and it owns the curation strategy over `Memory`. Proactive curation lives
behind `CurationHook.Consolidate` — implemented by the Orchestrator, never by the
passive `Memory` port.

---

## Capabilities & graceful degradation

A plugin announces what it can do (`manifest.go`):

```go
type Manifest struct {
    Kind         string
    Category     Category       // "gateway" | "backend" | "memory" | "orchestrator"
    Capabilities Capabilities   // Reactions, SelectMenus, Replies
    Config       []Setting      // every setting the plugin reads
}
```

`Degrade(g Gateway) Gateway` (`degrade.go`) wraps a Gateway so the manager can
*always* call the richest method while the fallback lives in one place:

| Called | If capability missing |
|--------|----------------------|
| `Reply` | falls back to `Post` |
| `React` | becomes a no-op |
| `Menu` | falls back to a numbered-list `Post` |
| `Post` | always passes through |

The wrapper also **forwards `BindSessionControl`** to the underlying gateway when it
implements `SessionControlReceiver`, so the session-control seam survives
degradation.

---

## Plugins, config & registry (`registry.go`, `plugin_settings.go`)

A gateway plugin hands the host a `GatewaySet{Gateway, Reader, Admin, Prober}` — one
coherent channel built from one `PluginConfig`. Optional ports may be nil. This is
what makes "add a plugin = blank import + rebuild": the host instantiates the set
from the registry and drives it with no plugin-specific wiring.

Plugins register a **factory** (not an instance) so they can announce themselves in
`init()` before any token is known (the xcaddy pattern):

```go
type GatewayFactory      func(ctx, cfg PluginConfig) (GatewaySet, error)
type BackendFactory      func(ctx, cfg PluginConfig) (Backend, error)
type MemoryFactory       func(ctx, cfg PluginConfig) (Memory, error)
type OrchestratorFactory func(ctx, cfg PluginConfig, mem Memory) (Orchestrator, error)
```

`Registry` collects `Plugin`s and queries them by category (`Gateways`/`Backends`/
`Memories`/`Orchestrators`); plugins self-register into the global `Default` from
their `init()` and the host queries it at startup. That same `Manifest` and query
surface now also project **out-of-process**: the released
[`herrscher-transport`](https://github.com/Herrscherd/herrscher-transport) module
lets a plugin announce itself over NATS and answer port calls over gRPC, so a
category can run as a separate process without changing `contracts` — in-process
stays the default (the `memory` port is carried first).

Each plugin declares its config surface as `[]Setting` (neutral `Key`, the `Env` it
binds from, `Required`, `Default`). `Resolve(settings, getenv)` builds a validated
`PluginConfig` — a required value still empty fails the daemon at startup with a
message naming every missing key, rather than surfacing deep inside the plugin.

---

## Layout

| File | Contents |
|------|----------|
| `backend.go` | `Backend`, `Prompt`, `BackendEvent`, `PendingChoice`, `ChoiceAware`, `ChoiceInjector` |
| `gateway.go` | `Gateway`, `Inbound` |
| `session_control.go` | `SessionControl`, `SessionInfo`, `SessionControlReceiver` |
| `event.go` | `Event`, `EventSink` |
| `host.go` | `Channel`, `Prober`, `ChannelReader`, `MenuRouter`, `ChannelAdmin` |
| `command_api.go` | `Cmd`, `Param`, `Input` (+ accessors), `Builder`, `New` |
| `conversation.go` | `Conversation`, `Choice`, `Attachment`, `SessionID`, `MessageID` |
| `message.go` | `Message` |
| `memory.go` | `Memory`, `Node`, `Link`, `Query`, `Subgraph`, `NodeKind`, `CurationHook` |
| `orchestrator.go` | `Orchestrator` |
| `manifest.go` | `Manifest`, `Capabilities`, `Category` |
| `plugin_settings.go` | `Setting`, `Resolve` |
| `registry.go` | `PluginConfig`, `GatewaySet`, factories, `Plugin`, `Registry`, `Default`, `Register` |
| `degrade.go` | `Degrade` decorator (incl. `BindSessionControl` forwarding) |
| `liveness.go` | `Liveness` keepalive sink |

---

## Using it

```go
import contracts "github.com/Herrscherd/herrscher-contracts"
```

You don't run this module — you implement its interfaces. A backend implements
`contracts.Backend`; a platform adapter provides a `GatewaySet` (a `Gateway` plus the
host-facing ports); the core consumes all of them.

Go 1.25. No build step, no dependencies.

```bash
go build ./...
go vet ./...
go test ./...
```
