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
| `StageTrigger` | `compaction.go` | optional, on a `Compactor` | `TriggerPercent()` is the share of the window from which a custom stage wants to run, before the host's `AutoCompactPercent`. Zero means the host's threshold |
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

### Code forge

| Port | File | Kind | What it does |
|---|---|---|---|
| `Forge` | `forge.go` | required | `Manifest`, `Connection` (login, token source, permissions, rate limit), `Repositories` (the repositories the account can read, paged), the pull request verbs `OpenPullRequest`, `PullRequest`, `MergePullRequest`, `Checks`, `ReviewThreads`, `ReplyToThread`, the CI verb `JobLog` (the tail of a job's log, bounded by `maxBytes`) and `Issue`. One implementation per forge (GitHub, GitLab, Forgejo) |

### Implemented by the host, consumed by plugins

| Port | File | What it does |
|---|---|---|
| `Forge` | `forge.go` | the operations of one forge host: its `Manifest` and `Connection`, the repositories the account can read (`Repositories`, paged), the access a token has to a repository, pull requests (open, read, merge), checks and their job logs, review threads, and issues. Paged reads take a `PageRequest` and return a `Page`; the event stream of pull requests is the optional `PullRequestFeed` interface, found by a type assertion, so a plugin without it still satisfies `Forge`. The feed works in process only: the host compiles the plugin that serves a forge into its own binary, and no wire transport carries it. A method on a repository of another host returns `ErrForgeWrongHost`, which the `forgetest` package checks for every method |
| `ForgeCredentials` | `forge.go` | `Token(ctx, host)` returns the `ForgeToken` for a `ForgeHost`, `Invalidate(ctx, host)` drops it after a 401. The host owns the secrets; a forge plugin never stores a token |
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
| `ApprovalRequest`, `ApprovalAnswer`, `ApprovalDecision`, `ParseApprovalDecision`, `ErrInvalidDecision`, `ApprovalRefusal`, `ParseApprovalRefusal`, `ErrInvalidRefusal`, `ApprovalAsker`, `WithApprovalAsker`, `ApprovalAskerFrom` | `approval.go`, `approval_context.go` | a tool approval asked during a turn and its verdict, and the asker a backend carries on the context |
| `Serve`, `WithErrorHandler`, `Ask`, `ConnectionError`, `ErrUnreadableAnswer`, `ErrMissingRequestID`, `MaxConcurrentConnections`, `EnvApprovalSocket` | `approvalsock/` | the local Unix socket that carries one approval from the hook to the backend and back, its reported failures, and the environment variable that names its path |
| `CompactTrigger`, `CompactRequest`, `CompactResult` | `compaction.go` | why a compaction runs (`manual` or `automatic`), and what a stage receives and reports. The host reads only `Note` as text, so producers set `Note` (English) and `Notice` (the code) until the host reads the code |
| `StageTrigger`, `ValidateStageTriggerPercent`, `ErrStageTriggerPercentOutOfRange` | `compaction.go` | the threshold from which a custom stage asks to run, never later than the host's, and the check the host runs on it |
| `CompactStage`, `RunCompactStack`, `CompactStageBackend` | `compact_stack.go` | a compaction stack run in order, and the reserved name of the session's backend stage |
| `History`, `HistoryMessage`, `HistoryCall`, `HistoryRole`, `HistoryEdit`, `PruneAction`, `PruneOutcome`, `ValidateHistoryEdits` | `history.go` | a session's conversation as a stage reads it, and the edits that prune its tool calls |
| `ToolCall`, `ToolCallState`, `ToolCallStates`, `ParseToolCallState`, `ErrUnknownToolCallState`, `ErrInvalidToolCall` | `tool_call.go` | one transition of a tool call (`started`, `succeeded` or `failed`) under the same `ID` for its start and finish, the closed set of states, and the check that refuses a malformed call on the wire |
| `BackendKindTool`, `BackendKindToolDone`, `BackendEvent.ValidateToolCall` | `backend.go`, `tool_call.go` | the kinds that start and finish a tool call, and the check of the call's shape against its kind |
| `DecodeEvent`, `ErrInvalidEvent` | `event.go` | the entry point that decodes an `Event` from its JSON, and the refusal of a field of the wrong type, named by its wire name, and of an outcome that names no turn outcome |
| `TurnOutcome`, `TurnFailed`, `TurnInterrupted`, `ParseTurnOutcome`, `ErrUnknownTurnOutcome` | `turn_outcome.go` | how a turn ended when it did not complete (`failed` or `interrupted`), carried by `Event.Outcome` on the terminal reply. The empty outcome is a completed turn. Any other text is refused, and the refusal never repeats it |
| `ErrInvalidApproval` | `approval.go` | the refusal of an `ApprovalRequest` or `ApprovalAnswer` field of the wrong JSON type, named by its wire name, and of an `ApprovalAnswer` whose `Decision` names no decision |
| `HealthState`, `WorstHealth`, `ProbeAccess`, `ProbeReport`, `HealthProber` | `health.go` | the outcome of a health check (`ok`, `degraded`, `down`, `off`), the account it checks, and the plugin level function that runs it |
| `LimitWindow`, `Limit`, `LimitReader`, `ErrLimitsUnavailable`, `ModelUsage` | `limits.go` | the vendor usage limits of an account (`five-hour`, `weekly`, `monthly`), the plugin level function that reads them, and the usage of one model within a turn |
| `ContextPart`, `ContextMemory`, `ContextSkills`, `ContextDelegation`, `ContextCapabilities`, `ContextIdentity`, `InjectedPart` | `context_part.go` | the blocks the host writes into the context of every turn and the estimated size of each, carried by `Event.Injected` on the terminal reply |
| `ModelReadable` | `attachment_readable.go` | whether the host hands an inbound `Attachment` to a model (an image, a text file, a PDF, JSON, XML, TOML or YAML), read from its type or, without one, its extension. A gateway that stages files itself checks it before downloading |
| `OutboundFile`, `ValidateOutboundFile`, `ErrInvalidOutboundFile` | `backend.go` | a file a backend rendered to disk (`Path` inside the turn's `Prompt.Outbox`, `Name`, `ContentType`), and the check the host runs before a gateway gets it |
| `CompactStats`, `CompactOutcome`, `CompactOutcomes`, `ParseCompactOutcome`, `ErrUnknownCompactOutcome`, `ErrNegativeCompactCount` | `compact_stats.go` | what a compaction stage measured for `herrscher usage`, its outcome (`done`, `short`, `failed`, `dry_run`, `skipped`), and the check `CompactStats.Validate` runs on its outcome and counts |
| `ForgeToken`, `NewForgeToken`, `ErrEmptyForgeToken`, `TokenSource` (`TokenSourceSaved`, `TokenSourceApp`, `TokenSourceEnv`, `TokenSourceCLI`), `ErrUnknownTokenSource` | `forge_token.go`, `forge_states.go` | a credential for one forge host. Its secret is unexported; `fmt` and `String` print `[redacted]`, and JSON writes only the login and the source, or `null` for the zero token. `TokenSource` says where it came from and renders as `unknown` when it names no source; an empty secret is refused |
| `ForgeRepo`, `NewForgeRepo`, `ForgeRepoFields`, `ErrInvalidCloneURL` | `forge_repo.go` | a repository a forge lists for the account: its reference, its https clone URL, its description (invalid UTF-8 replaced, then cut to 1024 bytes on a rune boundary), whether it is private and whether it is archived, and its default branch as a `BranchName` (the zero value when the forge gives none). Built only by `NewForgeRepo` from a `ForgeRepoFields`, which refuses a repository off the forge host, and a clone URL that is not an https URL naming that same repository or that `ParseRepoRef` refuses (longer than 512 bytes). A plugin skips a repository it refuses, and a forge served under a base path is not supported |
| `BranchName`, `ParseBranchName`, `ErrInvalidBranchName` | `forge_refs.go` | a git branch name, checked against the rules of `git check-ref-format --branch`, plus two rules of this package: not `@` alone, and at most 255 bytes. Built only by `ParseBranchName`; the zero value names no branch, and `IsZero` reports it. The error does not repeat the text |
| `ForgeHost`, `ParseForgeHost`, `ErrInvalidForgeHost`, `ForgeRepoRef`, `ParseRepoRef`, `ErrInvalidRepoRef`, `RequireRepoHost`, `ErrForgeWrongHost` | `forge_host.go`, `forge_repo.go` | a checked forge host (an ASCII DNS name, lowercased, with an optional port that is the web and API port), and a checked repository (host, namespace, name), built only by `ParseRepoRef` from `host/namespace/name`, an https URL, an ssh:// URL or a `git@` address. An ssh:// port is dropped, and port 443 is dropped from every host, so `github.com:443` and `github.com` are the same host. `RequireRepoHost` refuses the zero host or repository, and one that belongs to another host. `==` on a `ForgeRepoRef` is case-sensitive, so match repositories with `SameRepo` |
| `PullRequestRef`, `NewPullRequestRef`, `IssueRef`, `NewIssueRef`, `JobRef`, `NewJobRef`, `ThreadRef`, `NewThreadRef`, `ErrInvalidRef`, `PullRequestDraft`, `PullRequest` (with `Checks` and `LatestReview`), `ReviewID`, `PullRequestFeed`, `PullRequestSignal`, `Check`, `ReviewThread`, `ReviewComment`, `Issue`, `IssueQuery`, `ForgeActor`, `ForgeConnection`, `ForgeRateLimit`, `Revision`, `CommitSHA`, `ParseCommitSHA`, `ErrInvalidCommitSHA` | `forge.go`, `forge_refs.go`, `forge_commit.go` | the values a `Forge` takes and returns. A reference is built only by its `New…` constructor, which refuses a zero repository or a non-positive number. A `Check`'s conclusion is read through `Check.Concluded`, and an `IssueQuery` reads its state through `IssueQuery.Listed`, where the zero state lists open issues. A `CommitSHA` is 40 or 64 lowercase hex digits. A `LatestReview` is a `ReviewID`, compared for equality only; an empty one proves nothing, since the forge reports it both when there is no review and when it reports no reviews, and a host cannot tell the two apart |
| `PullRequestState`, `PullRequestEventKind`, `ChecksState` (with `ChecksUnknown`, the zero value), `SummarizeChecks`, `MergeMethod`, `MergeState`, `CheckStatus`, `CheckConclusion` (with `ConclusionOther`), `AuthorAssociation`, `IssueState` (with `IssueStateUnknown`, the zero value), `RepoPermission` | `forge_states.go` | the closed sets of a forge: a pull request's state, a merge method and the merge state the forge reports, a check's status and conclusion, an author's association, an issue's state (unknown, open or closed), and an account's permission on a repository. Each `String` prints the raw number of a value it does not name, except `TokenSource` and the zero `IssueState`, which print `unknown`. `SummarizeChecks` reads a list of `Check`s as one `ChecksState`: an empty list is `none`, a failed check outranks an action-required one, an action-required one outranks a pending one, and a pending one outranks a pass. Its godoc maps each check conclusion to its state. A check whose conclusion is `action_required` reads as `action_required`, which a host shows as a check waiting for a human action, never as a failure; `other` reads as `failed` |
| `ErrForgeNotConnected`, `ErrForgeUnauthorized`, `ErrForgeForbidden`, `ErrForgeNotFound`, `ErrForgeNotModified`, `ErrFeedRefused`, `ForgeRateLimitedError`, `ForgeMergeRefusedError` | `forge_errors.go` | the failures a caller tells apart on a forge: no credential, refused credential, missing permission, nothing under the reference, an unchanged resource, a refused event feed, a rate limit that carries its reset time (or says it is unknown), and a merge the forge refused with the state it reports |
| `PageRequest`, `PageCursor`, `Page`, `MaxPageLimit`, `ErrInvalidPageLimit` | `forge_page.go` | a paged read: a cursor, a limit from 0 to `MaxPageLimit`, and the page that returns the items and the next cursor |
| `MaxJobLogBytes`, `ValidateLogLimit`, `ErrInvalidLogLimit` | `forge_refs.go` | the largest log tail `Forge.JobLog` returns, and the check of a requested size against it |
| `CheckRefusesForeignHost`, `CheckListsRepositories`, `CheckPullRequestFeed` | `forgetest/` | test support, imported from `_test.go` files only: `CheckRefusesForeignHost` calls every method of a `Forge` that takes a repository, pull request, job, thread or issue with a reference on a foreign host, and fails the test unless each returns `ErrForgeWrongHost`. `CheckListsRepositories` pages through `Repositories` one item at a time and fails unless a limit outside `MaxPageLimit` is refused with `ErrInvalidPageLimit`, a valid page returns no error, the listing ends with an empty `Next` within 100 pages, and every repository is on the host given. Each page gets ten seconds, a call that ignores its context fails at that deadline, and the whole walk gets two minutes, after which it fails naming the budget. `CheckPullRequestFeed` opens the `PullRequestFeed` of a pull request, cancels it, and fails unless the channel closes within ten seconds of the cancellation and every signal is well formed, with an error signal, if any, the last before the close. A refusal at open passes. A valid signal sent after the cancellation is accepted, and the check does not prove that no new signal follows once the cancellation has been observed nor that no send blocks, so a plugin proves those in its own tests, along with the path of a stream that fails mid-stream, which the check cannot make fail. Its tests run the exported check under `testing/synctest`, so the timeouts are checked against simulated time. A plugin runs both against its forge in its own tests |
| `RequireRefusalWithoutEcho`, `RequireBareSentinel`, `RequireNoSecretFragment`, `Probe` | `echotest/` | test support, imported from `_test.go` files only: the checks that a refusal does not echo the input it refuses, by a run of four bytes, and that a bare sentinel carries no text added to it |

## Registration and config

A plugin declares a `Plugin{Manifest, <one factory>}` in `init()` and calls
`Register`.

Exactly one of `GatewayFactory`, `BackendFactory`, `MemoryFactory`,
`OrchestratorFactory`, `CompactorFactory` or `ForgeFactory` is set, matching
`Manifest.Category`. A `ForgeFactory` receives the config, the `ForgeHost` it
serves and the host's `ForgeCredentials`, in that order after the context. The
host builds one forge per `ForgeHost`. The exception is `CategorySkills`, which has no port to
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

`ModelReadable(Attachment)` is the one rule for which inbound files reach a
model. The host drops any other file, so a gateway that downloads files itself
asks it first and tells the author when a file is refused.

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
outbox (or an outbox that is the root), a name that is blank, `.`, `..` or holds
a path separator or a NUL, or a content
type that does not parse. The check reads no file, so the host also resolves
symbolic links, sends only a regular file and bounds its size: a backend can then
never make a gateway send a file of the daemon it did not render.

A gateway that implements `FileSender` posts the text with the files attached, in
reply to the message. Through `Degrade`, a gateway without it receives a reply,
or a post when the platform has no replies, whose text lists the name and path of each file, so no file is dropped silently.

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

A stage that judges the context with a model also reports the size of the state it
sent in `StateTokens` (estimated tokens) and the reduction it applied to that
state in `StateReduction`, a short English text the plugin defines, such as
`texts abridged`. Both are omitted from the JSON when zero or empty, so stats
written before they existed read back unchanged. `CompactStats.Validate` refuses
an `Outcome` that names no outcome with `ErrUnknownCompactOutcome`, and any
negative count or `Duration` with `ErrNegativeCompactCount`; JSON encoding does
not validate them.

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

A custom stage may run before that threshold. A `Compactor` that also implements
`StageTrigger` returns from `TriggerPercent()` the share of the window, in
percent, from which it wants to run: for instance a pruning stage that loses no
text can run from 50 while the backend's summary waits for 80. Zero means the
host's `AutoCompactPercent`, and so does a `Compactor` that does not implement
`StageTrigger`. A value above the host's threshold does not delay the stage,
which then runs at the host's threshold. When `AutoCompactPercent` is zero, the
operator turned automatic compaction off and no stage turns it back on: the
stage then runs only on a manual compaction. The host checks the value with
`ValidateStageTriggerPercent` and refuses any value outside 0 to 100
(`ErrStageTriggerPercentOutOfRange`), without running the stage at that value.

`SessionInfo.ContextWindow` is the context window size, in tokens, that the host
resolved for the session's vendor. Zero means the host did not resolve it, as an
older daemon does, and a frontend falls back to its own size.

`SessionInfo.PullRequest` is the pull request the session opened: its `Number`,
`State` (a `PullRequestState`, which marshals to `"open"`, `"closed"` or
`"merged"`), `Draft` and `URL`. A session with no pull request has the zero value,
which is omitted from the wire (`omitzero`), so a message from a daemon older than
this field decodes to no pull request. Encoded on its own, the zero value is `null`,
and decoding `null` gives the zero value back. Any other value must have a `Number`
above zero and a known `State`, and the same rule applies when encoding and when
decoding: a value that breaks it fails with `ErrInvalidSessionPullRequest`. A field
of the wrong JSON type, such as a numeric `State` or a quoted `Number`, is not a
break of that rule: decoding returns the `encoding/json` error as is. Decoding
an unknown `State` matches both `ErrInvalidSessionPullRequest` and
`ErrUnknownPullRequestState`, and encoding one matches the same pair. Encoding
refuses an unknown state, so `json.Marshal` fails for the whole `SessionInfo` list
(fail closed) rather than send a pull request without its state. The producer must
only fill valid values, the states listed above with a positive `Number`: the host
guarantees it never fills an invalid one.

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
for claude, `CODEX_HOME` for codex, `GROK_HOME` for grok. `ProfileEnv` builds
that single entry.

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

`Event.Injected` sizes every block the host composes into the turn's context
(`memory`, `skills`, `delegation`, `capabilities`, `identity`), each an
`InjectedPart` with `0` for a block left empty, carried on the terminal reply
(`done`) like `Models`. A reply with the list measured its turn, a reply without
it did not. The sizes are estimates the host makes from the text it wrote, not
counts reported by the vendor. A reader keeps a part it does not know under its
wire name, so a host that injects a new block never makes a reader refuse the
event. A wrong JSON type in `injected` is refused like in any other field.

From v0.39.0 on, `Event.Outcome` (`TurnOutcome`) says how a turn ended when it
did not complete:
`failed` when the backend failed, `interrupted` when the turn was cut. It is
carried on the terminal reply (`done`) only. The zero value is a completed turn,
which is omitted on the wire, and a `null` outcome reads as absent, as it does
for every other field. `TurnOutcome` marshals and unmarshals through
`ParseTurnOutcome`, which refuses any other text with `ErrUnknownTurnOutcome`
and never repeats it.

Two kinds of reader meet an outcome, and they behave differently:

- A reader built on this version, which knows the field, refuses an unknown
  outcome value: `DecodeEvent` refuses the event with `ErrInvalidEvent`, and
  `json.Unmarshal` returns the bare `ErrUnknownTurnOutcome`. It never reads the
  turn as completed. This strictness is deliberate.
- A reader built before the field existed ignores the `outcome` key, because
  `encoding/json` does not know it, and reads a failed or interrupted turn as a
  completed one. No change in this package can make such a reader refuse it.

Deployment: in herrscher the bridge and the host ship in one binary and update
together, so the host reads the outcome from the bridge it runs with. A gateway
or an external consumer kept on an older version reads a failed or interrupted
turn as a success. Upgrade every consumer that reads `outcome` before a producer
starts to emit one.

A reader of the wire must decode each line, that is each `Event`, with
`DecodeEvent(data)`. It refuses a field of the wrong JSON type, a number out of
range, or an outcome that names no turn outcome, with `ErrInvalidEvent`. Its
message names the wire field, such as `t` or `coordination`, and never the Go
type or the value. A number is out of range when it is a nonzero integer written
with digits only, an optional minus sign in front, that its integer field cannot
hold, or when it is beyond `float64`. Any other number refused by an integer
field, such as `1.5`, `1.0` or `1e2`, has the wrong JSON type, and so has `-0`
for an unsigned field, while a signed field accepts it as zero. A field nested
in `coordination`, `todos`, `models` or `subagent` is named by its top-level
field. A `tool_call` refused inside the event keeps `ErrInvalidToolCall`, a
document that is not valid JSON is returned as its `*json.SyntaxError`, and a
`null` decodes to the zero `Event`.

A type that embeds `Event` must decode its `Event` with `DecodeEvent` too, then
read its own envelope fields apart. It never receives a promoted
`UnmarshalJSON`, because `Event` defines none. `json.Unmarshal` into an `Event`
keeps the error of `encoding/json`, which names the Go type, so use
`DecodeEvent` wherever the wire name matters.

`BackendEvent.ToolCall` and `Event.ToolCall` carry the lifecycle of one tool
call, so a host can build a per-turn timeline with each call's start, duration
and outcome. Every transition has an `ID`, which the backend takes from its own
protocol's call id and which is unique within one execution of the backend (a
`reset` starts a new one, see below), and `Tool` on every transition, so a
finished call can be labelled without its start.

- The start is a `tool` event (`BackendKindTool`) with `ToolCall.State`
  `started`. It also keeps `Tool` and `Detail` on the `BackendEvent`: that
  duplication is the compatibility shim for hosts that predate `ToolCall`, so
  they still show the call.
- The result is a `tool_done` event (`BackendKindToolDone`) with
  `ToolCall.State` `succeeded` or `failed`, under the same `ID`. Its `Tool` and
  `Detail` on the `BackendEvent` stay empty, so a host that only knows `tool`
  never prints the call twice.
- A backend that cannot follow a call to its end emits the start only.
- A call whose start comes before a `reset` and that no `tool_done` of the
  current execution closes is unfinished: the host forgets it at the `reset`,
  with no duration and no outcome.
- A `reset` opens a new execution of the backend within the same turn. `ID` is
  unique within one execution, so the new process may reuse an ID from before
  the reset. A `tool_done` whose `ID` matches no call started in the current
  execution is ignored for the timeline and closes nothing, even when that ID
  comes from before the reset.

Backends follow the rule that they run `BackendEvent.ValidateToolCall` in their
tests on every event they emit. It checks the shape each kind requires: a `tool`
event carries a `started` call whose `Tool` and `Detail` equal the event's, a
`tool_done` event carries a finished call with `Tool` and `Detail` empty on the
event and no `Detail` on the call, no other kind carries a call, and neither
kind carries a stamped `At`. A `tool` event without a call stays valid, and a
`tool_done` event without a call is refused.

The host must run `BackendEvent.ValidateToolCall` on every `BackendEvent` it
receives, before it copies the `ToolCall` to an `Event`. On a refusal the host
must drop the `ToolCall` from that event and log the refusal. The rest of the
event, its text and its tokens, must still flow.

`ToolCall.At` is stamped by the host alone. A backend leaves it zero, and the
host sets it when it receives the transition. It is the time that travels on
`Event.tool_call` and in the journal, so the timeline has one clock. Its zero
value on the wire is omitted. Its zone offset must be a whole number of minutes:
the wire form writes the offset to the minute, so `ToolCall.Validate` and
`ToolCall.MarshalJSON` refuse a call whose offset has seconds rather than drop
them. Its year must be from 0 to 9999, the years the wire form writes, and both
refuse a year outside that range the same way.

`ToolCall.Detail` is display text, a command, a path or a query, and it can hold
secrets. It is set on the start only. An exporter must not send it outside the
host unless the operator opts in.

Gateways are host plugins, so they sit inside the boundary. They already
render `Tool` and `Detail` as the status line, so `Event.ToolCall.Detail`
reaching a gateway changes nothing. The opt-in rule concerns exporters that send
data out of the host to a telemetry backend.

`ToolCall.Error` is a short human reason for a failed call. It is set only on a
`failed` transition and empty otherwise. It can carry the same content as
`Detail` (stderr, a command, a token), so an exporter does not send it outside
the host unless the operator opts in. The OpenTelemetry mapping is future work:
an exporter will map `error.type` to the closed value `tool_failed`, never to
the `Error` text. `Event.tool_call` is omitted when the backend reports no
lifecycle.

A failed call on the wire, pretty-printed for reading:

```json
{
  "t": "status",
  "tool_call": {
    "id": "c1",
    "tool": "Bash",
    "state": "failed",
    "error": "exit 1"
  }
}
```

A `ToolCall` is parsed once, at the boundary. Decoding refuses a call with no
`state`, an unknown state, an empty `id`, an empty `tool`, an `error` on a state
other than `failed`, or a `detail` on a finished call (`succeeded` or `failed`).
Each refusal matches `ErrInvalidToolCall` with `errors.Is`. A call whose fields
have the wrong JSON type, whose state text names no state, or whose `at` is not
an RFC 3339 time, is refused the same way. A state text that names no state also
matches `ErrUnknownToolCallState`. A field of the wrong JSON type, and an `at`
that is not an RFC 3339 time, wrap no other cause: the message names the field,
never its value. A document that is not valid JSON is not a refused call:
`encoding/json` returns its `*json.SyntaxError` before the call is decoded, and
that error does not match `ErrInvalidToolCall`. `ToolCall.Validate` applies the
same rules to a call built in Go, and `ToolCall.MarshalJSON` runs it, so an
invalid call never reaches the wire.

No refusal echoes the input it refuses. A message names the rule that failed or
the field that has the wrong JSON type, never the text of `detail`, `error`,
`state`, `at`, `tool` or `kind`, so a secret that lands in a malformed field
stays out of logs. A field is named in its wire spelling from a fixed list, so a
key the input wrote differently (`ID` for `id`) is never echoed either.
`ParseToolCallState` does not quote the text it was given either. A refusal may
give a number, such as an ordinal, a count, a length or an index: it names a
position or a size, not the input. Four exceptions apply. The first is the
`*json.SyntaxError` of a document that is not valid JSON: it is not a refusal of
the contract, so it may quote a character of the input. A logger that must keep
the input out logs only the type of the error. The second is the
`*json.MarshalerError` that `json.Marshal` wraps around a refusal of
`ToolCall.MarshalJSON`, whose message names the Go type `*contracts.ToolCall`,
or of `TurnOutcome.MarshalText`, whose message names `contracts.TurnOutcome`.
`errors.Is` still matches `ErrInvalidToolCall` or `ErrUnknownTurnOutcome`
through that wrap. A direct call to `ToolCall.MarshalJSON` or to
`ToolCall.Validate` gives the message without it. The third is a path the host
sets, not the input: the outbox that `ValidateOutboundFile` names, and the
socket path and directory of `approvalsock`, which an error may name so the
operator can find them. The fourth is a value fixed by the compiled code of the
host, not by the input: a refusal may name it, as `ValidateModels` names the
kind of a backend, which comes from the manifest of a plugin built into the
binary, and as `NewCatalog` names the key, the language or the file name of a
message or file it refuses. The sources a host passes to `NewCatalog` are the
locales embedded in the module or in a plugin built into the binary, so the
exception covers them; a caller that reads user files into `NewCatalog` is
outside it. A parameter name that `Catalog.Message` refuses is likewise a name
the calling code chose. The rule does not concern the test tooling packages
(`*test`, such as `forgetest` and `i18ntest`).

Unknown states are refused, not skipped. A reader that meets an unknown state
refuses the whole event, and the error wraps both `ErrInvalidToolCall` and
`ErrUnknownToolCallState`. Adding a state is therefore a breaking change of this
module, released only after every host reads it.

Tokens are not a field of the call. The contract carries no per-call token
count. Each `usage` event carries the output tokens of one assistant message,
and the host adds them up: the cumulative count of a turn is the sum of the
`OutTokens` of the `usage` events received so far in that turn. A `result` event
is authoritative: when it arrives, its `OutTokens` replaces that sum. A `reset`
event leaves the sum unchanged: the host keeps the output tokens already counted
in the turn, since they were spent, and adds the usage of the new process on
top. The contract says the output streamed so far is discarded, not that the
tokens spent on it are. The host attributes to each call the cumulative count at
its start transition, minus that count at the previous tool-call transition of
the same turn (the start or the end of any call), or minus zero for the first
call of the turn. A `tool_done` that the host ignores is not a transition, so
it moves no count. A per-call token field would duplicate that count and could
disagree with it, so none is added.

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

## Remote approvals

A backend that runs on another machine than the daemon can ask the daemon to
approve a tool while a turn runs. The request rises in the turn's stream and the
answer comes back down the same stream. Nothing that identifies the daemon's
session leaves the daemon's machine: no session name and no token travels, in
the request, in the answer or on the socket.

The hook that the CLI agent spawns reaches the backend process through a Unix
socket on the same machine. The backend passes the socket path to the hook in
`HERRSCHER_APPROVAL_SOCKET` (`approvalsock.EnvApprovalSocket`).

- `ApprovalRequest{ID, Tool, Subject}` asks about one tool. `ApprovalAnswer{ID,
  Decision, Reason, Refusal}` answers it. `Decision` is `allow` or `deny`, and
  `Allowed()` is true only for `allow` with no `Refusal`: an empty, unknown or
  malformed decision, or a refused one, is a denial.
- `Refusal` is an `ApprovalRefusal`, set only when the transport refused the
  approval itself, before any asker answered. The known codes are `invalid`,
  `too_large`, `too_many` (the transport already had its maximum of approvals in
  flight), `no_asker`, `cancelled`, `deadline`, `stream_closed`, `send_failed`
  and `unclassified`. They are short stable codes, not catalogue keys. The field
  is omitted from the JSON when empty, and an empty `Refusal` means the decision
  came from an asker or the daemon.
- `Refusal` always goes with `Decision` `deny`. `MarshalJSON` refuses to write a
  refused answer with any other decision: an `allow` with a `Refusal` returns
  `ErrRefusedAllow`, and an empty or unknown `Decision` with a `Refusal` returns
  `ErrInvalidDecision`. It never changes a value silently. `UnmarshalJSON` reads
  any `Refusal` as `deny`, whatever valid `Decision` it holds, or no `Decision`
  at all, so for answers written by this module, a reader from before v0.35.0,
  which decides on `Decision` alone, never allows a refused answer. A `Decision`
  missing from the JSON, as in `{"ID":"1","Refusal":"too_many"}`, reads as
  `deny`. A `Decision` that is an empty string or unknown fails to decode, as in
  v0.34. A `Decision` of `null` reads as missing. `RefusedAnswer(id, refusal)`
  builds such an answer and returns `ErrInvalidRefusal` for an empty or unknown
  code.
- A JSON `null` leaves an `ApprovalAnswer` unchanged. `UnmarshalJSON` does not
  inherit the caller's `json.Decoder` settings, so `DisallowUnknownFields` does
  not reject unknown keys inside an answer.
- A JSON `null` leaves an `ApprovalRequest` unchanged. `UnmarshalJSON` does not
  inherit the caller's `json.Decoder` settings either, so
  `DisallowUnknownFields` does not reject unknown keys inside a request.
- A field of the wrong JSON type in an `ApprovalRequest` or an `ApprovalAnswer`
  is refused with `ErrInvalidApproval`. Its message names the wire field, `ID`,
  `Tool`, `Subject`, `Decision`, `Reason` or `Refusal`, and never the Go type or
  the value.
- Reading is tolerant. A code this version does not know is kept and relayed
  unchanged, and `ApprovalRefusal.Known()` tells whether this version can
  translate it, so a reader shows a generic reason for it. Such an answer is
  still refused: `Allowed()` is false. An empty `Refusal` in JSON is read as no
  refusal, so `{"Decision":"allow","Refusal":""}` is an allowance. The known
  codes are checked by `ParseApprovalRefusal` and `RefusedAnswer`, not by the
  text marshaller, which writes the code it holds.
- `ID` correlates a request with its answer within one stream. It is not a
  secret. An answer whose ID is not the request's is refused.
- `WithApprovalAsker` puts an `ApprovalAsker` on the turn's context, and
  `ApprovalAskerFrom` reads it back. `Backend.Respond` does not change.
- `approvalsock.Serve` runs on the backend side with the asker it got from the
  daemon. `approvalsock.Ask` runs in the hook with the path it read from the
  environment. `Reason` is free text the daemon supplies, and the contracts
  never translate it.

The model is deliberately closed:

- **No token travels.** The socket is local to the machine and carries one JSON
  line each way per connection.
- **The directory is private.** `Serve` refuses a socket directory that is not
  owned by the current user with mode 0700, with `ErrInsecureDirectory`, and
  `Ask` applies the same check to the directory of the path before it dials. A
  wrong directory is never reached. `Serve` does not remove a file already at
  the path.
- **Binding is serialised.** While it binds, `Serve` holds an exclusive `flock`
  on a lock file `<path>.lock` beside the socket, inside the same private
  directory. Two `Serve` calls racing on one path never both bind: the loser
  finds the winner's socket alive and returns `ErrPathInUse`. A wait for the
  lock ends as soon as the context ends: `Serve` then returns the context error
  without waiting for the holder. The lock file is left in place, empty and
  owner-only, once `Serve` returns.
- **Orphan sockets are recovered, nothing else is.** A socket left by a killed
  run is detected with `Lstat` and a probe: when the probe is refused, `Serve`
  removes it and listens. A regular file, a symlink, or a socket that still
  answers returns `ErrPathInUse`, and the file is left untouched. If `Serve` is
  cancelled before the probe ends, it returns the context error and leaves the
  orphan in place, so a cancelled start never reads as a live socket.
- **The socket is private through its directory.** `Serve` does not change the
  mode of the socket file: it keeps the mode the umask gives it. The protection
  is the directory, which is owned by the current user with mode 0700. `Serve`
  checks it before the bind and checks with `SameFile` after the bind that it is
  still the directory that holds the socket, and it closes the socket if it is
  not (`ErrDirectoryChanged`). No other user can traverse the directory, so none
  can reach the socket. Nothing else should be placed in that directory.
- **The peer is checked on Linux.** `Ask` reads the peer credentials with
  `SO_PEERCRED` and refuses a listener that does not run as the current user
  (`ErrForeignPeer`). On other unix systems the private directory is the only
  peer guard. On Windows, `Serve` and `Ask` return an error matching
  `errors.ErrUnsupported`, and the package still builds there.
- **Everything is bounded.** A message is at most 64 KiB (a line of 65535 bytes
  is accepted, one of 65536 is refused), and a request not received within five
  seconds is dropped. `Ask` refuses an oversized request with
  `ErrRequestTooLarge` before it dials. At most `MaxConcurrentConnections` (16)
  connections are answered at once. A connection that arrives while all of them
  are taken is closed at once without an answer, and reported to the error
  handler as `OpBusy` with `ErrBusy`. The hook never receives `ErrBusy`: it sees
  its connection closed or reset, which `Ask` returns as a read or write error,
  and it reads that as a denial. The connection does not wait in the listen
  backlog. A request is answered within ten minutes: the asker's context ends
  then, and also when the hook closes its connection, so an asker that waits
  for a human stops when nobody is waiting for it. A malformed, oversized or
  silent request closes its connection without reaching the asker.
- **The client keeps its write side open until the answer.** While the asker
  runs, `Serve` watches the connection. A client that half-closes its write side
  after its request, or that closes or resets the connection, counts as
  departed: it cancels the asker's context without a report, and `Serve` writes
  no answer after that context ends, so a client that half-closed receives
  nothing. The asker then returns a denial, which the hook reads as a refusal.
  `Ask` keeps its write side open while it waits, so the daemon sees the request
  until it is answered. Any other read failure on that connection is reported as
  `OpRead`.
- **No failure goes unseen, and none stalls the server.** `Serve` requires
  `WithErrorHandler`, and it reports every connection failure, every refused
  connection and every failed accept to it as a `*ConnectionError` carrying an
  `Op` (`OpAccept`, `OpRead`, `OpAnswer`, `OpClose` or `OpBusy`). `Serve` calls
  the handler from one goroutine of its own, fed by a queue of 32 failures, so a
  slow handler never delays the accept loop or a connection. A failure that
  finds the queue full is counted, and the handler then receives a
  `*ReportsDroppedError` with that count once the queue has room, matching
  `ErrReportsDropped`: a drop is never silent. `Serve` returns only after the
  handler has returned for every queued failure. The handler must return
  quickly. It must not panic either: `Serve` does not recover from a panic in
  the handler, by choice, because a panic is a bug that has to be seen.
- **An unsendable answer is never sent.** An answer whose decision is unknown
  does not marshal: `json.Marshal` returns `ErrInvalidDecision`, `Serve` reports
  it as `OpAnswer` and closes the connection without writing. The hook reads
  that as a refusal, never as a malformed answer. `ApprovalDecision` marshals
  and unmarshals through the same check.
- **Decisions are parsed at the boundary.** `ParseApprovalDecision` accepts only
  `allow` and `deny`, and `ApprovalDecision` marshals and unmarshals through the
  same check. `Ask` refuses a decoded answer whose decision is unknown or
  missing with `ErrInvalidDecision`. An answer line that is not one JSON object
  decoding as an approval answer, or that exceeds 64 KiB, matches
  `ErrUnreadableAnswer`, and neither error repeats the text received. A
  connection closed or reset before the answer is a read or write error that
  matches neither sentinel. `Allowed()` still denies anything else.
- **Failure is a denial.** `Ask` returns an error for every way the daemon may
  not be reached: an open directory, an absent socket, a foreign peer, an
  oversized request, a deadline, an unreadable answer, an unknown decision, a
  mismatched ID, a request with no ID. The caller turns every error into a
  denial. An asker that cannot reach the daemon returns a denial too, never an
  allowance. `Ask` waits as long as its context allows, so the caller sets the
  deadline. Its close error is reported once, and a deadline is never mistaken
  for a failure to close.

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
  `CodeHealthEnvMissing`, `CodeHealthRemedyRun`, `CodeHealthRemedyEnvRestart` and
  `CodeHealthRepoAccessUnreadable`, and the account codes `CodeAccountFallback`
  and `CodeAccountNoQuota`, and the transcript notice codes `CodeTurnInterrupted`
  and `CodeResumeLost`.
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
  shown as it is, `CodeHealthRepoAccessUnreadable` takes `Repo` and `Reason`, and
  the others carry no argument.
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
- `Notice` and `Arg` have a JSON form owned by contracts, the form compactwire
  uses for a compaction notice. A notice without arguments is `{"code":47}`,
  such as `CodeTurnInterrupted`. Otherwise it is
  `{"code":1,"args":[{"name":"Name","kind":1,"value":"lead"}]}`, where `kind` is
  1 for `Text`, 2 for `Count` and 3 for `Ref`, and `Arg` encodes alone in the
  same shape. The decoder reads that form and the v0.30.0 form
  (`{"Code":1,"Args":[{}]}`), ignores unknown fields and keeps a code outside
  `Codes` for `Renderer` to refuse. A value of kind 1 to 3 that is absent,
  `null` or of the wrong JSON type is refused with `ErrNoticeArgs`; `null` is
  reserved for the zero argument (kind 0). A wrong JSON type or a number out of
  range in a notice, `args` itself included, is refused with `ErrNoticeArgs` by
  the rule of `DecodeEvent`, named `code` or `args`, the wire field, with
  `has the wrong JSON type` or `is out of range`, and never the Go type or the
  value. A value whose kind is unknown is kept as its JSON, and it encodes back
  in an equivalent compact form: spaces are removed, and the HTML characters
  `<`, `>` and `&` are written in their escaped form (`\u003c`, `\u003e` and
  `\u0026`), as `encoding/json` does. `Renderer` refuses such an argument. A
  `null` argument, an argument that is not an object, a name or a kind of the
  wrong type, a kind outside `uint8`, a code outside `uint16` and an argument
  list that is not a list are all refused with `ErrNoticeArgs`. A malformed JSON
  text is not a shape error: `json.Unmarshal` returns it as the
  `*json.SyntaxError` of `encoding/json`, and `errors.Is(err, ErrNoticeArgs)` is
  false for it. A `json.Decoder` reports a stream cut short as
  `io.ErrUnexpectedEOF` before the notice is decoded, so that error is not a
  shape error either. A `null` notice changes nothing, as in v0.30.0: decoding
  it into a `Notice` leaves the `Notice` as it was. A `null` argument is refused
  with `ErrNoticeArgs`, since an argument has no absent value. A notice takes at
  most 16 arguments and a name at most 256 bytes of the decoded name, not of its
  escaped form. A value takes at most 64 KiB of JSON as it is encoded, and the
  raw JSON of a value is refused above 256 KiB before it is compacted.
  `encoding/json` reads the whole JSON notice once, argument list included,
  before these limits apply. The size of a value is measured on that encoded
  form, escapes included, on both encoding and decoding: a value that would
  exceed it is refused with `ErrNoticeLimit` when it is encoded and when it is
  decoded, so a relay never receives a notice it cannot send back.
- Compatibility with v0.30.0. The keys move from `Code` and `Args` to `code` and `args`.
  The compatibility holds only for a case-insensitive reader, such as Go's `encoding/json`,
  which reads `code` and `args` into a v0.30.0 `Code` and `Args`. A case-sensitive reader
  no longer reads the code. A notice without arguments comes back exact. A notice with
  arguments keeps its code and its number of arguments, but each argument is an empty `Arg`,
  as v0.30.0 already produces: `Renderer` refuses it with `ErrNoticeArgs` when its code has
  parameters, and renders a code without parameters as before. compactwire keeps its own
  frame in herrscher and is unchanged by this version. The shape model in
  `notice_json_test.go` mirrors that frame; it is a model of the shape, not the compactwire
  codec.
- `ScrollbackLine`, one entry of a session's scrollback as
  `SessionControl.Scrollback` returns it, carries a `Notice` when the entry is a
  notice rather than a reply, such as `CodeTurnInterrupted` for a turn that ended
  without a reply. The gateway renders `Notice` with `Renderer` in the session
  language. `Role` does not tell a notice from a reply. `Text` is the fallback for
  clients that predate the field, and for a notice that `Renderer` refuses. `Notice`
  is omitted from the JSON when it is nil, so those clients read the same format as
  before. A notice with arguments reaches the gateway, and the `session scrollback`
  command, with them, in the form described above. The rule for the gateway: when `Notice` is set and
  `Renderer` renders it, the gateway shows that text; when `Notice` is nil, or
  `Renderer` refuses it with `ErrUnknownCode` (an unknown code), `ErrNoticeArgs`
  (malformed arguments) or `ErrMissingMessage` (no English message), or with any
  other error of `Render`, for instance a catalogue that does not load, the gateway
  shows `Text`. No helper in contracts makes this choice for a `ScrollbackLine`:
  each gateway applies the rule.
- `ScrollbackLine.TurnID` carries the id of the turn that produced the entry, the
  same value as `Event.TurnID`. One turn produces several lines (a `user` line, one or
  more `assistant` replies, or the `notice` of `CodeTurnInterrupted`), and neither
  `Event` nor `ScrollbackLine` records a position within the turn. So a `TurnID` does
  not identify a line, and a `(TurnID, Role)` pair does not either. A gateway that
  replays a scrollback and then receives live events for the same session matches them
  with this rule, comparing whole lines on (`TurnID`, `Role`, `Text`):
  1. Find the longest suffix of the scrollback that equals a prefix of the live
     lines, line by line.
  2. Discard that prefix of the live lines: the scrollback already shows it.
  3. When either side of a comparison has an empty `TurnID`, compare `Role` and
     `Text` only.
  The rule needs no rank within the turn: `assistant a3` arriving live after the
  scrollback held `user q`, `assistant a1` and `assistant a2` matches nothing in the
  suffix, so it is shown.
- Live events become lines as follows. A `reply` event with non-empty text, and not
  reporting a host error, becomes one `assistant` line that carries the event's
  `TurnID`. No other event becomes a line: a `chunk` event is a piece of a stream and
  never a line. Only lines produced this way are matched against the scrollback.
- An empty `TurnID` on either side is a residual. When a scrollback line or a live
  `reply` has no `TurnID`, the comparison falls back to (`Role`, `Text`), so a live
  `assistant` line with the same text as a scrollback line is discarded as a repeat,
  even when it is a new reply. There are two cases:
  - A scrollback line without a `TurnID`, for example `assistant "ok"`, followed by a
    live `assistant "ok"` with `TurnID` T1: the live line is discarded. This is kept on
    purpose: the scrollback can hold lines from before the field existed, and the
    gateway cannot tell those apart from a repeat. It applies only while such a line is
    among the last lines of the scrollback that the live batch aligns with. For example,
    with a scrollback of `user "q"` with `TurnID` T0 followed by `assistant "ok"`
    without a `TurnID`, the live `assistant "ok"` with `TurnID` T1 is discarded; with a
    scrollback of `assistant "ok"` without a `TurnID` followed by `user "q"` with
    `TurnID` T0, the same live line is shown.
  - A live `reply` without a `TurnID`, for example `assistant "ok"`, after a scrollback
    of `assistant "ok"` with `TurnID` T0: the live line is discarded. `Event.TurnID` is
    omitted when empty, so the contract guarantees nothing here. This case does not end
    with the scrollback: it lasts until the emitter sets a `TurnID` on every `reply`.
- The key is `TurnID` in JSON, not `turn_id` as on `Event`, on purpose.
  `ScrollbackLine` keys every field by its Go name (`Role`, `Text`, `Notice`), a
  convention pinned by `TestScrollbackLineJSONKeepsGoFieldNames`, while `Event` keys its
  fields in snake_case. A client maps the event's `turn_id` onto the line's `TurnID`.
- `TurnID` is omitted from the JSON when empty. An entry that no turn produced, and a
  line written before the field existed, both read with an empty `TurnID`.
- `CodeTurnInterrupted` is a transcript notice, not a health code. It reports a
  session turn that ended without a reply, whatever the cause, and names no cause.
  It takes no argument.
- `CodeResumeLost` reports that the earlier conversation of a session could not be
  resumed, so the turn runs on a new session without its context. It takes no
  argument. This version defines the contract only: the code, its texts in the shared
  catalog, and `BackendKindNotice`, the `Kind` of a `BackendEvent` whose `Notice` field
  carries the notice.
- One route per notice: a notice reaches a gateway only through `NoticeSink.Notify`.
  `Event` has no notice field and no notice type.
- `ErrNoticeFallbackPosted` is returned by a `NoticeSink` that posted a plain-text
  fallback instead of the notice. The caller neither retries nor posts the notice
  again through `Degrade`.
- A code of success, such as `CodeSessionCreated`, is a notice like any other: it
  is never `CodeUnexpected`.
- `Arg` is built by `Text(name, value)`, `Count(n)` (named `Count`, it selects
  the plural form) and `Ref(name, id)` (the identifier of a platform object, which
  a gateway renders as a mention). Its zero value is malformed, and a decoded `Arg`
  can be malformed too: `Renderer` is the safety net that refuses what its code
  cannot show. `Arg` has no
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
