# herrscher-contracts

**The ports.** This module is the shared vocabulary of the Herrscher platform: the
interfaces and the neutral data types that every other repo agrees on. It is the
narrow waist that lets the core stay ignorant of *which* chat platform delivers a
message and *which* model answers it.

- **Zero dependencies** — only the Go standard library.
- **Zero logic** — type definitions plus one thin decorator (`Degrade`).
- **Consumer-defined ports** — the interfaces are written from the point of view of
  the code that *calls* them (the core), not the code that implements them. Adapters
  satisfy them structurally; they never import each other.

> Part of the [Herrscher](../herrscher-host/README.md) family:
> **contracts** · [core](../herrscher-core/README.md) ·
> [claude-backend](../herrscher-claude-backend/README.md) ·
> [discord-gateway](../herrscher-discord-gateway/README.md) ·
> [host](../herrscher-host/README.md)

---

## Why a separate repo

Hexagonal architecture lives or dies on one rule: **the domain depends on
abstractions, never on concretions.** If `core` imported the Discord adapter or the
Claude backend, swapping either would mean editing the core. Instead, all three
depend *only* on `contracts`:

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
| `BackendEvent` | One intermediate occurrence in a turn. `Kind` is `"tool"`, `"text"`, `"result"`, or `"reset"`; carries `Tool`, `Detail`, `Cost`, `IsError`. |
| `PendingChoice` | An interactive selection a backend is blocked on (`Question` + `[]Choice`). |

Two optional capability interfaces let a backend pause on an interactive choice
(e.g. a tool-permission prompt), serialized with its normal turns:

```go
type ChoiceAware interface {
    PendingChoice() (PendingChoice, bool)
}
type ChoiceInjector interface {
    InjectChoice(ctx context.Context, value string) (string, error)
}
```

### Gateway port — the channel edge (`gateway.go`)

A chat-platform plugin. The manager always calls the rich method; a decorator
(`Degrade`) downgrades any action the plugin doesn't advertise.

```go
type Gateway interface {
    Manifest() Manifest
    Post(ctx context.Context, conv Conversation, text string) (MessageID, error)
    Reply(ctx context.Context, conv Conversation, replyTo MessageID, text string) (MessageID, error)
    React(ctx context.Context, conv Conversation, msg MessageID, emoji string) error
    Menu(ctx context.Context, conv Conversation, replyTo MessageID, prompt string, opts []Choice) error
}
```

---

## Host-facing ports (`host.go`)

The always-on daemon needs more than a Gateway. These ports split the platform into
small, independently-implementable surfaces:

| Port | Method(s) | Role |
|------|-----------|------|
| `CommandSource` | `Run`, `Commands() <-chan InboundCommand` | Connects to the platform and streams inbound slash/component/autocomplete commands. |
| `CommandResponder` | `Defer`, `Respond`, `Edit`, `Autocomplete`, `AckComponent` | Replies to an interaction by its opaque `ResponseToken`. |
| `CommandRegistrar` | `Register` | Publishes the slash-command tree. |
| `Prober` | `Probe() (latencyMS, err)` | Cheap round-trip for health. |
| `StatusReporter` | `Upsert(prevID, content)` | Maintains a self-updating status message. |
| `Platform` | `Read`, `EnsureChannel`, `Unreact`, `SendSelectMenu`, … | The bridge's view of a channel: read history, post menus, manage reactions. |

`Liveness` (`liveness.go`) is a transport-level keepalive sink (`HeartbeatAck`).

---

## Commands (`command.go`)

A platform-agnostic model of a slash command, a clicked component, or an
autocomplete request.

```go
type Command struct {
    Invoker string
    Data    CommandData
}

type CommandData struct {
    Name     string
    Options  []Option   // recursive: subcommands & groups nest here
    CustomID string     // component interactions
    Values   []string   // component selected value(s)
}
```

`CommandData` carries helpers so the core can read the option tree without knowing
the platform's wire format:

- `Opt(name) (string, bool)` — recursive string lookup
- `OptBool(name) bool` — recursive bool lookup
- `Focused() (name, value, ok)` — the option being autocompleted
- `Subcommand() (string, []Option)` — the first subcommand and its options

`InboundCommand{Kind, Command, Token}` bundles one command with its
gateway-private `ResponseToken`. `CommandKind` is `KindSlash`, `KindComponent`, or
`KindAutocomplete`. `OptionType` distinguishes `OptSubcommand` from
`OptSubcommandGroup`.

---

## Conversation & messages

`Conversation{Gateway, ID}` (`conversation.go`) is an opaque, comparable address
into a chat platform — usable as a map key (`Conversation → SessionID`).
`SessionID` and `MessageID` are string aliases. `Choice{Label, Value}` and
`Attachment{Filename, URL, ContentType, Size}` are the neutral selection and file
primitives. `Message` (`message.go`) is the inbound envelope with full author
metadata and attachments.

---

## Capabilities & graceful degradation

A plugin announces what it can do (`manifest.go`):

```go
type Manifest struct {
    Kind         string
    Category     Category      // "gateway" | "backend" | "memory" | "orchestrator"
    Capabilities Capabilities
}

type Capabilities struct {
    Reactions   bool
    SelectMenus bool
    Replies     bool
}
```

`Degrade(g Gateway) Gateway` (`degrade.go`) wraps a Gateway so the manager can
*always* call the richest method while the fallback lives in one place, never in the
domain:

| Called | If capability missing |
|--------|----------------------|
| `Reply` | falls back to `Post` |
| `React` | becomes a no-op |
| `Menu` | falls back to a numbered-list `Post` |
| `Post` | always passes through |

---

## Plugin registry (`registry.go`)

A minimal in-process registry — `RegisterGateway`, `Gateways()` — that stands in for
what becomes NATS self-registration in a later phase. The `Manifest` shape is
deliberately identical so the migration is a transport change, not a contract change.

---

## Layout

| File | Contents |
|------|----------|
| `backend.go` | `Backend`, `Prompt`, `BackendEvent`, `PendingChoice`, `ChoiceAware`, `ChoiceInjector` |
| `gateway.go` | `Gateway`, `Inbound` |
| `host.go` | `CommandSource`, `CommandResponder`, `CommandRegistrar`, `Prober`, `StatusReporter`, `Platform`, `Channel` |
| `command.go` | `Command`, `CommandData` (+ helpers), `Option`, `CommandResponse`, `InboundCommand`, `AutocompleteChoice`, kinds & option-type constants |
| `conversation.go` | `Conversation`, `Choice`, `Attachment`, `SessionID`, `MessageID` |
| `message.go` | `Message` |
| `manifest.go` | `Manifest`, `Capabilities`, `Category` |
| `degrade.go` | `Degrade` decorator |
| `registry.go` | in-process `Registry` |
| `liveness.go` | `Liveness` |

---

## Using it

```go
import "github.com/Akayashuu/herrscher-contracts"
```

You don't run this module — you implement its interfaces. A backend implements
`contracts.Backend`; a platform adapter implements `contracts.Gateway` and the
host-facing ports; the core consumes all of them. See the
[core README](../herrscher-core/README.md) for who calls what.

Go 1.23. No build step, no dependencies.
