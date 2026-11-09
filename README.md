# herrscher-contracts

**The port package.** Every Herrscher plugin implements interfaces declared here,
and the core consumes only these.

It is types and interfaces plus a few thin helpers. No runtime, no daemon, no
plugin logic. The root package depends on the Go standard library and on the
`i18n` sub-package, which brings go-i18n and x/text (see below). Go 1.27.

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
| `Compactor` | `compaction.go` | optional | `Compact(ctx, CompactRequest, onEvent)` shortens a session's context. A backend's is its default, a plugin's is a custom stage |
| `HistoryEditor` | `history.go` | optional | `ReadHistory(ctx)` returns the conversation the model sees, `PruneHistory(ctx, edits)` prunes tool calls from it in place, all or nothing |
| `FileProducer` | `backend.go` | optional | `TakeFiles()` hands over the files the last `Respond` rendered, once |

### Channel edge

| Port | File | Kind | What it does |
|---|---|---|---|
| `Gateway` | `gateway.go` | required | `Manifest`, `Post`, `Reply`, `React`, `Menu` |
| `GatewayCloser` | `gateway.go` | optional | `Close(ctx)` stops the plugin's goroutines and waits for them within `ctx` |
| `ChannelReader` | `host.go` | optional | `Enabled`, `DefaultChannel`, `EnsureChannel`, `Read`, `Unreact`, `UpsertStatusMessage` |
| `ChannelAdmin` | `host.go` | optional | `Kind`, `CreateUnder`, `ForumPost`, `Archive`, `Send`, `ChannelRef` |
| `MenuRouter` | `host.go` | optional | `RouteMenu`: menu picks return to a named route, not to the channel |
| `Prober` | `host.go` | optional | `Probe` round-trip latency, for liveness |
| `EventSink` | `event.go` | optional | `Emit(Event)`: the gateway renders the live turn stream itself |
| `RoutedEventSink` | `event.go` | optional | `EmitTo(Conversation, Event)`, the multi-session variant, preferred over `EventSink` |
| `Foreground` | `foreground.go` | optional | `RunForeground`: the gateway owns the main thread (a TUI). At most one |
| `NoticeSink` | `gateway.go` | optional | `Notify(ctx, conv, lang, Notice)`: the gateway renders a notice itself, in the given language, with English as the fallback. Without it, `Degrade` renders the text |
| `FileSender` | `gateway.go` | optional | `SendFiles(ctx, conv, replyTo, text, files)` posts text with files attached. Without it, `Degrade` replies with a text that lists the names and paths |
| `SessionControlReceiver` | `session_control.go` | optional | `BindSessionControl` receives the host's session-control handle |
| `CatalogSource` | `catalog_source.go` | optional, deprecated | `Catalog() fs.FS` on a live instance; use the descriptor field `Plugin.Catalog` instead |

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
| `SessionControl` | `session_control.go` | `Dispatch`, `Create` and `Close` (return a `Result`), `Sessions`, `Scrollback`, `Resume`, `Interrupt` |
| `Coordinator` | `coordinator.go` | `Handoff`, `Delegate`, `Report`, `Merge`, `Seal`, `FanOut`, `Route` |
| `RosterProvider` | `roster.go` | `Agents()`, the agents a session may delegate to |
| `Liveness` | `liveness.go` | `HeartbeatAck`, the sink for a gateway's keepalive |

### Types and helpers, not ports

| Name | File | What it is |
|---|---|---|
| `ModelSpec`, `Route`, `RoutePolicy` | `models.go` | the model catalog a backend declares, its route, and the policy that bounds it, plus `ValidateModels`, `FilterModels`, `GatewayCreds` and `NewGatewayCreds` |
| `MergeEnv`, `ParseEnvSetting`, `EncodeEnvSetting` | `spawnenv.go` | encode and decode the per-session environment the host injects at spawn, and the `Env*` key constants both sides must use |
| `WithPrincipal`, `PrincipalFrom` | `principal.go` | who is asking, carried on the context |
| `CompactTrigger`, `CompactRequest`, `CompactResult` | `compaction.go` | why a compaction runs (`manual` or `automatic`), and what a stage receives and reports. The host reads only `Note` as text, so producers set `Note` (English) and `Notice` (the code) until the host reads the code |
| `CompactStage`, `RunCompactStack`, `CompactStageBackend` | `compact_stack.go` | a compaction stack run in order, and the reserved name of the session's backend stage |
| `History`, `HistoryMessage`, `HistoryCall`, `HistoryRole`, `HistoryEdit`, `PruneAction`, `PruneOutcome`, `ValidateHistoryEdits` | `history.go` | a session's conversation as a stage reads it, and the edits that prune its tool calls |
| `HealthState`, `WorstHealth`, `ProbeAccess`, `ProbeReport`, `HealthProber` | `health.go` | the outcome of a health check (`ok`, `degraded`, `down`, `off`), the account it checks, and the plugin level function that runs it |
| `LimitWindow`, `Limit`, `LimitReader`, `ErrLimitsUnavailable`, `ModelUsage` | `limits.go` | the vendor usage limits of an account (`five-hour`, `weekly`, `monthly`), the plugin level function that reads them, and the usage of one model within a turn |
| `OutboundFile`, `ValidateOutboundFile`, `ErrInvalidOutboundFile` | `backend.go` | a file a backend rendered to disk (`Path` inside the turn's `Prompt.Outbox`, `Name`, `ContentType`), and the check the host runs before a gateway gets it |
| `CompactStats`, `CompactOutcome`, `CompactOutcomes`, `ParseCompactOutcome`, `ErrUnknownCompactOutcome` | `compact_stats.go` | what a compaction stage measured for `herrscher usage`, and its outcome (`done`, `short`, `failed`, `dry_run`, `skipped`) |

## Registration and config

A plugin declares a `Plugin{Manifest, <one factory>}` in `init()` and calls
`Register`.

Exactly one of `GatewayFactory`, `BackendFactory`, `MemoryFactory`,
`OrchestratorFactory` or `CompactorFactory` is set, matching
`Manifest.Category`. The exception is `CategorySkills`, which has no port to
build and so declares no port factory. A gateway factory returns a
`GatewaySet{Gateway, Reader, Admin, Prober}`: one channel, with the optional
ports left nil.

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

## Files a backend renders

A backend that implements `FileProducer` hands over the files it wrote to disk
while it answered, such as a generated image. The host gives each turn an
`Outbox`, an absolute directory in `Prompt.Outbox`, and the backend writes its
files there. Each `OutboundFile` carries the absolute `Path` of the file, the
`Name` a gateway shows, and its `ContentType`. The host reads `TakeFiles` once
after each `Respond`, and a file taken is not returned by a later call.

The host checks each file with `ValidateOutboundFile(file, outbox)`, which
matches `ErrInvalidOutboundFile` for a path that is not clean or lies outside the
outbox, a name that is empty, `.`, `..` or holds a path separator, or a content
type that does not parse. The check reads no file, so the host also resolves
symbolic links, sends only a regular file and bounds its size: a backend can then
never make a gateway send a file of the daemon it did not render.

A gateway that implements `FileSender` posts the text with the files attached, in
reply to the message. Through `Degrade`, a gateway without it receives a reply
whose text lists the name and path of each file, so no file is dropped silently.

## Compaction

A backend that implements `Compactor` compacts its own sessions: for Claude that
is `/compact`. A plugin of `CategoryCompactor` offers a custom stage: its
`CompactorFactory` builds the `Compactor`, it answers no prompt and is never a
session vendor, and `Registry.Compactors` lists it. The manifest declares a
compactor with `Capabilities.Compactor`, so a stack can be checked before any
plugin is instantiated.

A session's compaction is a stack of `CompactStage`s that the host runs in its
configured order with `RunCompactStack`:

- each stage receives the usage the previous one left, its `TokensAfter` when
  `TokensMeasured` is true, and the usage it was given otherwise. The first
  measured stage sets `UsageMeasured` to whether the request has a `TokenLimit`
  (the request's own flag is normalised the same way on entry), and `TokensUsed`
  and `TokenLimit` are read only while it is on;
- the context is checked before each stage, so a cancelled context stops the
  stack before the next stage runs;
- a stage whose result is `Done` stops the stack, and its result is returned;
- a failing stage, or one whose `Compactor` is nil (`ErrNilCompactor`), stops the
  stack, and its error is returned wrapped with the stage's name, so `errors.Is`
  still matches the cause. The stages before it keep their effect, and the result
  they leave in `StageStats`;
- the stage named `CompactStageBackend` is the session's own backend compaction;
- a nil `onEvent` is replaced by one that drops the events;
- the stats of each stage are kept: the returned result carries in `StageStats`
  the `Stats` of every stage that ran and reported some, in order, with `Stage`
  set to the stage's name when the stage left it empty.

A stage reports what it did in `CompactResult.Stats` (`CompactStats`): its
`Outcome` (`CompactOutcome`, which marshals as `done`, `short`, `failed`,
`dry_run` or `skipped`), the calls it kept, truncated, pruned or pinned, the
characters and the estimated tokens it removed, and the model requests it made.
`Outcome` must be set: its zero value refuses to marshal. The host reads the stats for `herrscher usage`.

The host calls `Compact` only while it holds the turn of the session, under the
same exclusion as `Respond`. The host renders the outcome (the note, the tokens
before and after) to the gateways, not the stage. Progress goes through the
`onEvent` callback, as for `Respond`.

### History edited in place

A backend that implements `HistoryEditor`, and declares it with
`Capabilities.HistoryEditor`, lets a stage shorten the conversation without a
summary. The host passes it to the stages it runs as `CompactRequest.History`,
nil when the backend has none, so a stage that needs it skips or fails on its own
terms. `ReadHistory` returns the messages the model sees on its next turn, oldest
first, each assistant message with its tool calls and their results.
`PruneHistory` takes `HistoryEdit`s naming calls by `HistoryCall.ID`:
`PruneResult` keeps the call, the first `KeepChars` runes of its result and the
last `KeepTail` runes, with a note in place of the runes between them. `PruneCall`
keeps only a trace of the call, and refuses a `KeepTail`. Text the operator or the agent wrote
is never an edit target. The edits apply all or none, after
`ValidateHistoryEdits`, and an edit naming no call of the history fails with
`ErrUnknownHistoryCall`. `PruneOutcome` counts the runes of tool inputs and
results before and after, so a stage can judge the saving without a tokenizer.

`CompactTrigger` marshals to its text form (`"manual"`, `"automatic"`), and
refuses to marshal a value that names no trigger.

`SessionInfo.AutoCompactPercent` is the share of the context window, in percent,
at which the session compacts itself without being asked. Zero means it never
does, and 1 to 100 is the threshold. The host checks it with
`ValidateAutoCompactPercent` where it produces the value; JSON decoding does not.

`SessionInfo.ContextWindow` is the context window size, in tokens, that the host
resolved for the session's vendor. Zero means the host did not resolve it, as an
older daemon does, and a frontend falls back to its own size.

## Model routing

A backend declares the models it knows how to run in `Manifest.Models`
(`[]ModelSpec{ID, Label, Arg, Efforts, Route, InputPrice, OutputPrice,
CacheReadPrice, CacheWritePrice, ContextWindow, Hidden}`), so the catalog is
readable without instantiating anything.

`ID` is what the host persists in session state. `Arg` is only what goes on the
CLI's `--model`, and the two often differ.

`ValidateModels(kind, models)` is what the host calls at startup. An empty
`ID`, `Label` or `Arg`, a duplicate `ID`, or an unknown route is a startup
failure, not a silently wrong selector.

The prices are USD per million tokens and `ContextWindow` is in tokens. Zero
means unknown, and a gateway route carries OUR prices, as `InputPrice` does. A
negative number is refused with `ErrModelNegative`, and a NaN or infinite price
with `ErrModelNotFinite`.

`Hidden` marks a retired or historical model. It shares the ID space with the
visible models, so a hidden and a visible model cannot have the same `ID`. The
contract exposes two views of a catalog, and a host uses each for one job:

- `FilterModels(models, policy)` is the resolution view. It keeps the hidden
  models, so a session or agent recorded with one still finds its route by `ID`.
- `SelectableModels(models, policy)` is the selector view. It applies the same
  policy and also leaves out the hidden models. It never resolves an `ID`.

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

### Discovery

A backend that can list its models on the current machine sets
`Plugin.Discover`, a `ModelDiscoverer`. It receives a `DiscoveryAccess` (the
gateway pair the host holds, the route `Policy` the host enforces, and a
`Getenv` for the vendor's own key or settings) and returns what the vendor
serves now. A backend must honour `Policy`: under `PolicyGatewayOnly` it
neither runs a native listing (CLI or vendor API) nor returns native models,
and under `PolicyAll` it lists its native models next to the gateway ones. A vendor with no listing on
this machine returns no models and no error, and the static catalog stands
alone. An error is a failed attempt: the host logs it and keeps its last list.

`MergeDiscovered(kind, static, discovered)` is the only way discovered models
enter a catalog. It validates the discovered list as a whole, so one malformed
entry rejects the lot and the static catalog is kept. A discovered model that
the static list already names, by `ID` first and then by route and `Arg`, is dropped: the
static entry keeps its label, price and efforts. A new model keeps the vendor's
label, or takes its `ID`. The four prices (`InputPrice`, `OutputPrice`,
`CacheReadPrice`, `CacheWritePrice`) are always reset to zero on a discovered
model, since a gateway price is ours and the discovery does not know it. They are
discarded before validation, so a bad price on a discovered model never rejects
the list. The `ContextWindow` is kept: a new model takes it, and a static model
takes it when it is non-zero, while its label and prices stay as they are. A
negative window is read as unknown (0) and does not reject the list. A
static `Hidden` model that discovery lists again stays hidden. The host then
applies `FilterModels` as it does to a static list, so under `gateway-only` a
discovered native model stays absent.

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

### Account profiles

A profile is a second login the user owns for one vendor (a personal and a work
account, say), each kept in its own directory. The host points the vendor CLI at
that directory with one variable, given by `ProfileHomeEnv`: `CLAUDE_CONFIG_DIR`
for claude, `CODEX_HOME` for codex. `ProfileEnv` builds that single entry.

Cursor has no profile. Its CLI offers no variable that relocates its login on
every platform, so `ProfileHomeEnv` refuses it with `ErrNoProfileHome`.

`ParseProfileName` is the only way to a `ProfileName`. It refuses the reserved
name `native`, which means no override at all, so the zero value of a profile
field is the native login.

### Health and limits

`Plugin.Probe` is a `HealthProber` and `Plugin.Limits` a `LimitReader`. Like
`Discover`, they are plugin level functions: the host calls them on its own,
without a live instance, so a backend that is not logged in can still say so.
Each is nil when the plugin offers none. Any category may set `Probe`; only a
backend sets `Limits`.

Both receive a `ProbeAccess`: the `Getenv` of the account checked (its profile
environment merged over the host's), the `Account` (empty for the native login),
the plugin's own `Config`, as its factory would receive it, and `Deep`, which
asks for a real, minimal round trip instead of a cheap check.

A probe reports a `ProbeReport` rather than an error: its `State`, the
`Latency` it measured (zero when it did not), and a `Detail` and a `Remedy`
that are `Notice` values, codes the client renders, never sentences. The
`CodeHealth` codes cover the usual findings (not logged in, rate limited,
credentials refused, unreachable, a missing variable) and remedies. `Remedy`
is the zero `Notice` when there is nothing to do. `HealthOff` means not set up
or not logged in: it never fails a check on its own. `WorstHealth` ranks the
states `down`, then `degraded`, then `ok`, then `off`, and returns `off` for an
empty list.

A `LimitReader` returns each window the vendor reports for the account, scoped
to the whole account (`Scope` empty) or to a sub-scope the vendor names, such
as a model family. `ResetsAt` and `Span` are zero when the vendor gives none.
It never spends a turn to read them: a vendor that exposes none returns `ErrLimitsUnavailable`.

`Event.Models` and `BackendEvent.Models` split the usage of a turn by model,
each a `ModelUsage`, so a ledger can tell the main model from the one that ran
the subagents. They are empty when the backend gives no breakdown, and the
totals stand alone.

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
no-op, `Notify` to a `Post` of the notice rendered as text, `SendFiles` to a `Reply`
whose text lists each file's name and path, and `BindSessionControl` still
forwards.

`MemoryScope{Project, Agent}` plus `RecordShared`, `RecordPrivate`,
`RecallScoped` and `RecallRelevant` implement shared-versus-private memory as a
policy over the plain `Memory` verbs.

`EnforceBudget`, `NextState` and `Score` back per-node size limits, node
lifecycle states, and relevance ranking.

## Translated messages (`i18n`)

The `i18n` sub-package is the runtime of herrscher's translated texts. It is the
only part of the module with dependencies beyond the standard library:
`github.com/nicksnyder/go-i18n/v2` and `golang.org/x/text`. The root package
imports it for `Renderer`, so those dependencies reach every module that imports
contracts.

- `Language` is the closed set of supported languages, `English` and `French`.
  `ParseLanguage` accepts a BCP 47 tag or a POSIX locale in any letter case
  (`fr`, `fr_FR.UTF-8`, `en_US`, `C`, `POSIX`). `String` renders a supported
  language as its tag, and any other value as `Language(n)`. The zero value is
  not a language: it means unknown. A language crosses the wire as its number
  (`English` is 1, `French` is 2), so a value is frozen like a `Code`.
- The session language reaches the plugins in two places. `SessionInfo.Language`
  is what a gateway reads from `SessionControl.Sessions()`, to render the notices
  of the session's conversation in it. `Prompt.Language` is what the host sets on
  each turn, so a backend answers in it, and sees a change made by `/set lang`
  from the next turn on. A zero `Language` means the host does not know it, and
  the reader falls back to the operator's default. A backend that reads it
  decides how to apply it; a backend that ignores it is unchanged.
- `FromEnv` reads `LC_ALL`, `LC_MESSAGES` and `LANG` through an injected
  `getenv`, as POSIX does: the first one that is set and not empty decides. If
  its value is not a supported language, `FromEnv` reports false without looking
  at the variables after it. `Resolve` returns the first supported candidate, or
  English when none is.
- French messages use the straight apostrophe (') and the « » guillemets, as the
  French texts of the herrscher repositories do.
- A catalog is a set of JSON files, one per language, named after it
  (`en.json`, `fr.json`), in the go-i18n v2 format:
  `{"id": "session.archived", "other": "Session archived"}`. A placeholder is
  `{{.Name}}`.
- A message that is plural in English has `one` and `other` forms, and so does
  each other language. The form is chosen by the language's CLDR rules from the
  `Count` parameter, which is a numeric string, a signed integer, or an unsigned integer up to
  `math.MaxInt64`. Any other type, a float included, is refused with `ErrParamType`.
- `NewCatalog(logger, sources...)` builds a catalog from several `fs.FS`, usually
  one embedded directory per package. A key defined twice across the sources is
  refused with `i18n.ErrDuplicateKey`. A template that does not parse, including a
  call to a template function, is refused at construction.
- `Catalog.Message(lang, key, params)` renders a message. A `{{.Name}}` field with
  no entry in `params` is an error. Every entry of `params` must be a scalar: a
  string, a bool, an integer (`int`, `int8` to `int64`, `uint`, `uint8` to `uint64`) or
  a float (`float32`, `float64`), of exactly these types: a named type such as `Language`
  is refused, so the caller passes its underlying value. Any other value,
  nil untyped or typed, pointer, slice, map, struct or func, is refused with
  `ErrParamType` even when the message does not use that field, because the template
  would render it as text such as `<nil>` or `[]`. A key missing in a language renders in English, and a
  key missing everywhere returns the key itself. Each of these cases logs one
  warning per key and language, for at most 1024 keys; past that, each further
  missing key is logged every time it is asked for. An unsupported language, or a
  `Count` that cannot be read, returns an error.
- The restriction to plain fields is applied by `i18ntest.Check` only: a message
  may not hold control structures, named templates or template functions. The
  rendering does not enforce it, so the guard must run in the test suite of each
  package that owns messages.
- `i18ntest.Check(t, sources...)` is that guard. Every English key must exist in
  each supported language with its plural forms, each language must use the same
  fields as English, and a key defined in a non-English language but absent from
  English is refused.
- `Plugin.Catalog` is the `fs.FS` of a plugin's translated texts, in the format
  above, set in its descriptor so the host reads it before any instance exists.
  The host builds its catalog from its own sources and those of the plugins, so
  a key defined in both is refused with `i18n.ErrDuplicateKey`. A plugin without
  texts leaves it nil. `CatalogSource` is its deprecated instance-side form.
- `i18ntest.CheckDeclared(t, declared, sources...)` ties a catalog to the codes
  that use it. `declared` maps each key to the names of its fields. Each declared
  key must exist in every language with exactly those fields, and a key the
  sources define without a declaration is refused.

### Codes, notices and results

Messages a human reads are named by a code, not by a sentence. The host emits
`Notice{Code, Args}` and the front end renders it.

- `Code` is a closed `uint16` enum: `CodeSessionNotFound`, `CodeSessionExists`,
  `CodeUnknownAgent`, `CodeSessionCreated`, `CodeSessionClosed`, `CodeUnexpected`,
  `CodeSettingInvalid`, `CodeSettingMissing`, `CodeNodeMerged`, `CodeCompactUnavailable`,
  `CodeCompacted`, `CodeAddressPlaintextRefused`, the compaction reasons
  `CodeCompactTimeoutInvalid`, `CodeCompactInstructionRefused` and `CodeCompactNoThread`,
  and the session, agent and host codes `CodeSessionLimit`, `CodeNoHome`,
  `CodeNoHostRegistered`, `CodeServiceControlUnavailable`, `CodeModelRequired`,
  `CodeExplicitCmdRefused`, `CodeInvalidValue`, `CodeSharedAgent`,
  `CodeUnknownApprovalMode`, `CodeAgentExists`, `CodeAgentInvalidName`,
  `CodeAgentNotFound`, `CodeUnsupportedHomeType`, the generic `CodeEntityNotFound` and
  `CodeEntityUnknown`, `CodeScheduleNotFound`, `CodeChannelKindUnder`,
  `CodeChannelKindHome`, `CodeInvalidValueHint`, `CodeSessionCloseDirty`, and the
  health codes `CodeHealthConnected`, `CodeHealthSignedInAs`, `CodeHealthNotLoggedIn`,
  `CodeHealthRateLimited`, `CodeHealthCredentialsRefused`, `CodeHealthUnreachable`,
  `CodeHealthEnvMissing`, `CodeHealthRemedyRun` and `CodeHealthRemedyEnvRestart`.
  `Code.Key()` returns its catalog key (`session.not_found`, ...) through an
  exhaustive switch; a value outside the list returns its raw number, `code.N`.
  `Codes()` lists every code, and a test scans the whole `uint16` range, so a
  constant named in `Key` or `params` but missing from `Codes()` fails the suite.
- `Code` declares its parameters, which are `Name` for the session and agent
  codes, `Detail` for `CodeUnexpected`, `Setting` and `Value` for
  `CodeSettingInvalid`, `Settings` for `CodeSettingMissing`, `Reason` for
  `CodeCompactUnavailable`, `Address` for `CodeAddressPlaintextRefused`, and none
  for `CodeNodeMerged`, `CodeCompacted`, `CodeNoHome`, `CodeNoHostRegistered`,
  `CodeServiceControlUnavailable`, `CodeModelRequired`, `CodeExplicitCmdRefused` and
  the compaction reasons, which carry no argument. `CodeSessionLimit` takes `Count`,
  a `Count` argument that selects the plural form. `CodeInvalidValue` takes `Field`
  and `Value`, `CodeSharedAgent` takes `Agent`, `CodeUnknownApprovalMode` takes
  `Mode`, `CodeAgentExists`, `CodeAgentInvalidName` and `CodeAgentNotFound` take
  `Name`, and `CodeUnsupportedHomeType` takes `Type`. `CodeEntityNotFound` and
  `CodeEntityUnknown` take `Kind` and `Name`, where `Kind` is shown as it is: the
  French texts read `{Kind} « Name » introuvable` and
  `valeur inconnue pour {Kind} : « Name »`. `CodeScheduleNotFound` takes `Name`,
  `CodeChannelKindUnder` and `CodeChannelKindHome` take `Channel`, a `Ref`, and
  `Kind`, and `CodeInvalidValueHint` takes `Field`, `Value` and `Hint`; `Hint` is a
  text the caller composes and is shown as it is, so it is not translated.
  Among the health codes, `CodeHealthSignedInAs` takes `Name`,
  `CodeHealthUnreachable` takes `Reason`, `CodeHealthEnvMissing` and
  `CodeHealthRemedyEnvRestart` take `Name`, `CodeHealthRemedyRun` takes `Command`,
  shown as it is, and the others carry no argument.
  `CodeSessionCloseDirty` takes `Name`, the session whose worktree holds
  uncommitted changes: a close refused for that reason is a normal outcome, not
  `CodeUnexpected`, and its text tells the operator to commit or to close with
  `force:true`. Each parameter declares its kind, and every one is a `Text` except
  `Count`. `Settings` is a text the caller composes, for instance the names of the missing settings
  joined with `, `: it is not a list, and the text is shown as it is.
  `Detail` holds the technical text of a failure with no code of its own. It is
  never translated: a gateway renders it in a code block, and `Degrade` renders it
  as plain text. `Reason` is an ordinary `Text` argument, the technical reason of a
  compaction that cannot run. It sits inside a sentence, so a gateway must
  neutralise it like every other `Text` argument. The Discord gateway does so from
  v0.24.0 on, for each `Text` of a notice.
- `ErrNoticeFallbackPosted` is returned by a `NoticeSink` that posted a plain-text
  fallback instead of the notice. The caller neither retries nor posts the notice
  again through `Degrade`.
- A code of success, such as `CodeSessionCreated`, is a notice like any other: it
  is never `CodeUnexpected`.
- `Arg` is built only by `Text(name, value)`, `Count(n)` (named `Count`, it selects
  the plural form) and `Ref(name, id)` (the identifier of a platform object, which
  a gateway renders as a mention). Its zero value is malformed. `Arg` has no
  float constructor, so a decimal number is never rendered in Go format. `Render`
  checks that the name matches the kind: `Count` must be an `ArgCount`, and an
  `ArgCount` must be named `Count`. It also checks each argument against the kind
  its code declares for that name: a `Ref` or a `Count` given for a `Text`
  parameter is refused with `ErrNoticeArgs`.
- `Renderer` turns a `Notice` into text in a language, through the shared catalog
  `locales/en.json` and `locales/fr.json`, embedded in the module. `NewRenderer()`
  builds the catalog on first use. `Render` refuses a notice whose code is
  unknown, whose arguments are not exactly the names its code declares, or whose
  code has no English message, the fallback language. The last one returns `ErrMissingMessage`:
  a raw key is never returned to a human. The other two return `ErrUnknownCode`
  and `ErrNoticeArgs`.
- `NoticeError` is an error that reaches a human: it implements `Notice() Notice`.
  `NoticeOf(err)` returns the notice of the first `NoticeError` in the chain, or
  `CodeUnexpected` with the error text as `Detail`. `errors.Is` and
  `errors.AsType` keep matching the sentinels and types, not the text.
- `Result` is what `Cmd.Run` and `SessionControl.Dispatch` return. It carries a
  notice (`NoticeResult`) for the human, or typed data (`DataResult`) for a caller
  that renders it itself, such as `--json`. `DataResult(nil)` equals `Result{}`:
  it carries neither. When `Dispatch` returns an error, its `Result` is ignored.
  `Cmd.Help`, `Param.Help` and the help of a `Builder` are catalog keys: the
  module that declares the command renders them, in its own catalog.
- `NoticeSink` is the optional port of a gateway that renders a notice itself,
  in the language it receives, with its own markup. A message missing from that
  language renders in English. Without it, `Degrade` renders the notice as plain
  text in the language and posts it through `Post` as it is, with the same
  English fallback, so no emoji or markup is added on that path. The `Text`
  arguments are not escaped there: the gateway's `Post` must neutralise them. A
  gateway that renders `CodeUnexpected` puts its `Detail` in a code block,
  untranslated; `Degrade` keeps the plain text.

Log lines, CLI syntax and flags, protocol markers and the prompts sent to agents
are not translated. They stay in English.

## Migration

The next release is one breaking release: every change below lands together.

- `Cmd.Run` returns `(Result, error)` instead of `(string, error)`. Return
  `NoticeResult(notice)` for a message to a human, or `DataResult(v)` for data
  that a caller renders itself. `Builder.Do` takes the same signature.
- `SessionControl.Dispatch`, `Create` and `Close` return `(Result, error)` instead
  of `(string, error)`. Read the notice with `Result.Notice()`, or the data with
  `Result.Data()`. When the error is not nil, the `Result` is ignored.
- `Builder.Help`, `Builder.Param`, `Builder.ValueParam` and `Cmd.Help` take a
  catalog key, not a sentence. The module that declares the command renders the
  key in its own catalog. The Go compiler does not tell a key from a sentence:
  both are strings, so a sentence left in place compiles. The CLI guard catches
  it.
- `NoticeSink` is additive and optional. A gateway that implements
  `Notify(ctx, conv, lang, notice)` receives the notice and its language. Without
  it, `Degrade` renders the notice as text.

Messages that have no code yet, successes included, get codes from the shared
catalog as their module migrates to codes. Until a message has its code, its text
is not in the shared catalog. A success is never `CodeUnexpected`: it has its own
code, such as `CodeSessionCreated` or `CodeSessionClosed`. Texts specific to one
module (Discord, the terminal UI, the CLI) stay out of the shared catalog: the
module builds its own catalog with `i18n.NewCatalog`, and checks it with
`i18ntest`.

## Further reading

- [Herrscher docs](https://github.com/Herrscherd/herrscher-docs), page
  `reference/ports`
