# gixt Agent Prompt Utility Specification

Status: Proposed
Implementation branch: `feat/agent-prompt-utility`
Scope: GitHub issues #4–#9 plus proposed work packages WP7–WP8

## 1. Summary

gixt retrieves and composes versioned Gist artifacts through `cat`, and explicitly executes scripts through `run`.

```text
target(s) -> resolve each gist -> obtain revision -> select entry/entries -> cat to stdout | run one entry
```

Targets identify Gists; entries identify files. `cat` and `run` share resolution, acquisition, and selection, then apply different actions and security policies.

## 2. Product examples

```sh
gixt cat review-prompt | agent
gixt cat review.md style.md | agent
gixt run cleanup-script -- --dry-run
```

Plain text is the prompt format; no manifest or agent framework is in scope.

## 3. Goals

Composable byte-exact retrieval; explicit execution; deterministic resolution and selection; actionable errors; reuse existing cache, pin, and authentication behavior.

## 4. Non-goals

Prompt schemas/frameworks, fuzzy or silent ambiguous resolution, a dependency manager, a public Go library, and unrelated Gist mutations are out of scope.

## 5. Implementation context

- `resolveTarget` returns only an ID and filename selection lives in `runner`; consolidate both before the action so exact filename intent is preserved (`internal/cli/resolve.go:18`, `internal/runner/runner.go:18-24,65-87`).
- `runWithOptions` combines acquisition and execution; reuse its resolution/cache/acquisition path for `cat` instead of copying it (`internal/cli/run.go:82-172`).
- Root typo suggestions currently run before target resolution; suggest commands only after typed target-not-found (`internal/cli/cli.go:45-52`).
- Owner pagination already exists (`internal/gist/gist.go:210-240`). Reuse it; keep listing read-only and remove `list clear` to avoid shadowing a target named `clear`.
- Trust snapshots currently fetch revisions sequentially; release builds already inject a version but omit `-trimpath -s -w`. These are localized changes for issues #4 and #9.

## 6. Public CLI Design

### 6.1 Core commands

```text
gixt cat <target> [<target> ...] [--entry <filename>]
gixt run <target> [--entry <filename>] [-- <args>]
gixt list [owner]
gixt add <target> [--as <name>]
gixt add mine
gixt remove <target> [--yes]
gixt remove --owner <owner> [--yes]
```

`remove <target>` uses normal target resolution (ID, URL, owner/name, alias, or filename). `remove --owner` forgets all locally remembered Gists belonging to that owner. These forms are mutually exclusive. Owner-wide removal requires confirmation unless `--yes` is supplied.

Each positional operand to `cat` is resolved independently through the normal target resolver; operands may identify files from the same or different Gists, or Gists themselves. A Gist target without `--entry` selects all its files; an exact filename target selects that file. `--entry` is allowed only with a single target and selects one exact file from that Gist. To concatenate individual files across Gists, pass each filename as its own target, for example `gixt cat review.md style.md`. Both `cat` and `run` accept `-e, --entry <filename>`; for `cat`, it is limited to a single target.

`gixt add mine` is the canonical shorthand for registering all visible Gists owned by the authenticated user; it is equivalent to resolving the current login and running `gixt add owner <owner>`. It requires authentication, accepts no `--as` alias because it registers multiple Gists, and is intentionally named `mine` to match `trust mine`. The exact `mine` subcommand takes precedence over a target or alias with that name.

Advanced state and management commands may remain:

```text
gixt trust ...
gixt pin ...
gixt auth ...
gixt cache ...
gixt gist ...
gixt self ...
```

Documentation and root help should lead with `cat`, `run`, `list`, and `add`. `print` is an alias for `cat`. Advanced commands should not define the primary product mental model.

### 6.2 Bare-target behavior

Change the shorthand immediately:

```sh
gixt <target>
```

means:

```sh
gixt cat <target>
```

Execution always requires the explicit `gixt run <target>` command. Treat this as an intentional breaking API change and document it in the release notes and migration examples; do not preserve bare execution during a compatibility period.

`gixt print` is an alias for `gixt cat` and has identical arguments and output behavior. `run --view` remains available temporarily but is deprecated and must not be documented as the retrieval API.

## 7. Target and Entry Model

Introduce a shared result that preserves both identities:

```go
type ResolvedTarget struct {
    GistID        string
    RequestedFile string
}
```

`GistID` identifies exactly one Gist. `RequestedFile` is populated only when the user's target explicitly and exactly identifies a filename.

Recommended internal operations:

```go
func ResolveTarget(...) (ResolvedTarget, error)
func SelectEntry(files []string, requested string) (string, error)
```

The exact package and receiver shapes should follow existing project style. The important boundary is that selection happens once, outside `runner`, and is shared by `cat` and `run`.

## 8. Gist Resolution Rules

Resolve a target in this order:

1. Exact Gist ID or Gist URL.
2. Exact known alias.
3. `owner/name` against known entries, then live owner lookup when networking is allowed.
4. Exact full filename among known entries.
5. Filename stem among known entries.
6. Return a typed not-found or ambiguity error.

Rules:

- An alias identifies a Gist, never a file.
- Aliases should be globally unique when added.
- Filename matching may identify a Gist only when exactly one Gist remains.
- Owner qualification narrows Gist candidates but does not bypass same-owner collisions.
- Platform preference must not choose between different Gists.
- Offline mode must never initiate live owner lookup.
- Ambiguity errors must list useful disambiguators: aliases, `owner/filename`, or shortened/full IDs.

### 8.1 Exact filename propagation

If `script.zsh` uniquely resolves one known Gist because that Gist contains an exact file named `script.zsh`, return:

```go
ResolvedTarget{
    GistID:        "...",
    RequestedFile: "script.zsh",
}
```

A stem such as `script` may resolve a Gist but should not necessarily select an entry when multiple variants exist. Selection then follows the fallback rules unless an explicit `cat` entry operand or `run --entry` is supplied.

## 9. Entry Selection Rules

After exactly one Gist has resolved, select entries according to the action:

For `cat`:

1. Explicit `--entry <filename>` when `cat` has exactly one target.
2. Exact full filename carried by `ResolvedTarget.RequestedFile`.
3. All files in lexical order.

For `run`:

1. Explicit `--entry <filename>`.
2. Exact full filename carried by `ResolvedTarget.RequestedFile`.
3. Files matching `main.*`.
4. Files matching `index.*`.
5. Current-platform shell variant when candidates are variants of the same basename.
6. First filename in lexical order.

Rules:

- Explicit entries must match exact Gist filenames/paths, case-sensitively unless GitHub's filename semantics require otherwise.
- Missing explicit entries are errors and must show available filenames.
- Explicit entries always override a filename inferred from the target.
- `cat --entry` is valid only with one target; reject it when multiple targets are supplied.
- An exact filename target selects that file; an alias, ID, URL, or owner/Gist target with no `--entry` selects all files for `cat`.
- Selection and concatenation must be deterministic; sort the Gist file list before selecting all files or applying fallback selection.
- Selection returns filenames, not commands.

## 10. Shared artifact pipeline

Extract only the common path from `runWithOptions`: discover paths, resolve the target and pinned/ref revision, obtain files through existing cache/network behavior, select entries, and return cleanup responsibility. Keep the result type and helper boundaries minimal; this section requires shared behavior, not a prescribed `Artifact` struct or a generic pipeline framework.

`cat` applies the shared path independently to each target in command-line order and writes its selected content. `run` requires one selected entry, then builds and executes its command. Do not duplicate fetching or cache implementations for `cat`.

## 11. `gixt cat` Contract — Issue #6

```text
gixt cat <target> [<target> ...] [--entry <filename>] [flags]
```

Supported flags for the first version:

```text
--offline
--no-cache
--ref <sha>
```

Each target is resolved independently through the normal target resolver and processed in command-line order; targets may resolve to the same or different Gists. A single `--entry <filename>` is allowed only when exactly one target is supplied and must match an exact file in that Gist. Without `--entry`, a Gist target emits all its files in lexical order, while an exact filename target emits that file. Multiple files from one or more Gists can be named as separate targets, each resolved normally.

Behavior:

1. Resolve and obtain each target's artifact through the shared pipeline.
2. Select the single explicit entry, all files in lexical order, or the exact file inferred from the target.
3. Write the selected bytes directly to stdout, accepting binary content without prompting or filtering.
4. Concatenate entries and targets in order with no headers, separators, status messages, or added newlines.
5. Write diagnostics only to stderr.
6. Never perform execution trust checks or modify the trust store.
7. Honor pins when `--ref` is absent.
8. Honor cache and offline semantics consistently with `run`.

### 11.1 Binary content and separators

`cat` accepts and writes binary content byte-for-byte by default. It must not prompt, detect/reject binary data, or alter bytes. This allows pipelines such as `gixt cat script | bash`; `gixt` itself only writes the content, while the downstream program decides what to do with it.

Do not insert a line break or any other separator between files. Unix `cat` concatenates bytes exactly, and an automatic delimiter would corrupt binary data and change text content. Users who need boundaries can add them explicitly in the shell or in the source files.

### 11.2 `--view` migration

- Keep `run --view` temporarily for compatibility.
- Mark it deprecated in help and documentation.
- `cat` streams the selected file or files; `print` remains an alias.
- `gist show` remains the metadata/file-list inspection command.
- Remove `--view` in the next intentional breaking release.

## 12. `gixt run` Contract — Issues #6 and #7

Add `--entry/-e` to the explicit `run` command. Bare-target dispatch is `cat`, not a run shorthand.

Change command construction from:

```go
BuildCommand(dir string, files []string, userArgs []string, python string)
```

to:

```go
BuildCommand(dir string, entry string, userArgs []string, python string)
```

`runner` should only determine how to execute an already-selected file:

1. Explicit Python override.
2. Shebang.
3. Extension mapping.

Trust remains run-only and keyed by Gist ID plus commit. Selecting a different entry from an already-trusted commit does not require a second approval because the approval already covers the revision and its file set.

## 13. Root Dispatch and Typo Handling — Issue #5

Exact Cobra commands always win. Any other root token is treated as the first target for `cat`:

1. Attempt normal target resolution and cat the result.
2. If it resolves, cat it even when its name resembles a command.
3. If resolution returns typed `ErrTargetNotFound`, compute a command suggestion.
4. If a likely command exists, return the suggestion.
5. Otherwise return the original not-found error.

Do not replace these errors with typo suggestions:

- ambiguity;
- authentication failure;
- network failure;
- rate limiting;
- malformed local state;
- cache miss in offline mode;
- permission failure.

Introduce typed errors rather than matching message strings, for example:

```go
type TargetNotFoundError struct { ... }
type AmbiguousTargetError struct { ... }
```

Acceptance examples:

```text
gixt runc       # retrieves a valid target named runc to stdout
gixt remvoe     # suggests remove only if target remvoe is genuinely absent
gixt script     # reports ambiguity, never suggests a command
gixt owner/tool # preserves network/auth errors
```

## 14. `gixt list [owner]` — Issue #8

`list` becomes a read-only command:

```text
gixt list                 # local remembered entries
gixt list <owner>         # remote public/private-visible Gists for owner
gixt list <owner> --limit 30
gixt list <owner> --all
```

Rules:

- Default remote limit: 30 entries.
- `--limit` must be positive and sets the maximum result count.
- `--all` follows pagination until exhausted.
- `--limit` and `--all` are mutually exclusive.
- Output uses the same columns for local and remote entries where practical.
- Listing a remote owner does not remember, cache, or trust their Gists.
- API and status diagnostics go to stderr; table/data goes to stdout.

The existing `ListForOwner` implementation should be reused, with pagination adapted so callers can stop at the requested limit without fetching five fixed pages unnecessarily.

### 14.1 Removing remembered Gists

Remove `gixt list clear`; the `list` namespace is for listing, and a subcommand named `clear` can shadow a valid Gist alias or target named `clear`.

Use the existing `gixt remove` command for known-store changes:

- `gixt remove <target>` forgets one locally remembered Gist after normal target resolution.
- `gixt remove --owner <owner>` forgets all locally remembered Gists for that owner.
- `--owner` and `<target>` are mutually exclusive.
- Owner-wide removal requires a TTY confirmation unless `--yes` is supplied; refuse in non-interactive mode without `--yes`.
- Confirmation must state that local remembered entries, aliases, and pins for that owner will be removed.
- These operations do not delete remote GitHub Gists or clear cache/trust approvals.

The explicit `--owner` form avoids ambiguity with aliases or filenames and matches the existing owner-based registration workflow.

### 14.2 Owner registration progress

`gixt add owner <owner>` should report progress to stderr as each Gist-list page is received, without contaminating stdout. Since the total page count is not known in advance, report the page number and cumulative number of Gists found, for example:

```text
Loading Gists: page 2 (200 found)
```

Do not save a partial owner refresh: collect all pages successfully, then replace that owner's known entries and persist once. On a page failure, leave the known store unchanged. Keep progress local to this page-fetch operation; do not introduce a general progress/spinner framework.

### 14.3 Registering the authenticated user

`gixt add mine` is a shorthand for `gixt add owner <authenticated-login>`:

- Resolve the login through the existing authenticated-user API.
- Reuse owner registration pagination, progress, pin preservation, and atomic replacement.
- Require authentication and return the existing actionable authentication error when unavailable.
- Do not assign aliases automatically; each remembered Gist remains addressable by ID, filename, or a later explicit `--as` registration.
- Reject positional arguments and `--as` for this form.

This is part of the owner-management work package because it adds a convenience entry point while reusing the existing owner-registration semantics.

### 14.4 Human-readable list output

The default `gixt list` and `gixt list <owner>` output should be a compact, deterministic table suitable for terminals and readable in logs:

```text
OWNER       NAME       ID        UPDATED                  DESCRIPTION
leolaurindo wp1-test   21306350  2026-09-24T06:55:35Z     gixt WP1 target and entry selection test
```

Requirements:

- Use the same columns and ordering for local and remote rows where data exists: owner, name, shortened ID, updated time, and description.
- Use the alias as `NAME`, falling back to the first lexical filename.
- Shorten IDs consistently using the existing cache display convention; `gist show` remains the full metadata view.
- Align columns and keep output deterministic; do not use color, spinners, or terminal-only control sequences.
- Keep table data on stdout and API/status diagnostics on stderr.
- Preserve the existing listing semantics, limits, pagination, and read-only behavior; this work package changes presentation only.

## 15. Trust Progress — Issue #4

Issue #4 is specifically about `gixt trust mine`: it lists the authenticated user's Gists, fetches each current revision, and approves that Gist ID/revision for execution. Progress applies to those revision snapshots only—not to every Gist registration/remembering operation such as `gixt add`.

`gixt trust mine` must provide concise progress on stderr without changing stdout.

Initial implementation:

```text
Snapshotting gists: 2/10
```

Requirements:

- Progress goes only to stderr.
- Non-TTY output uses stable line-oriented progress rather than a spinner.
- Fetch all required revisions before persisting trust changes.
- If any fetch fails, do not partially update the trust store.

Fetch Gist revisions with a bounded worker pool of at most 3 concurrent requests; do not parallelize the paginated `/gists` listing. The current list-item data does not include the revision history needed for trust approval, so fetch each Gist's detail once and reuse that response.

- Keep the number of API calls to the required list pages plus one detail fetch per Gist.
- Update progress on stderr as revision fetches complete; preserve input ordering in final diagnostics.
- Respect GitHub primary/secondary rate-limit responses and `Retry-After`/reset headers when supplied; avoid aggressive retries.
- Stop or cancel outstanding work on a fatal error where practical.
- Collect every successful revision in memory and persist once, only after all required fetches succeed.
- On any failure, leave the trust store unchanged.
- Keep the worker pool and progress reporting local to `trust mine`; do not build a general concurrency/progress framework or reuse these workers for unrelated commands.
- Do not use unbounded goroutines because GitHub secondary rate limits apply.

## 16. Release Binary Stripping — Issue #9

Change the release build to preserve version injection while stripping paths and debug tables:

```sh
go build -trimpath \
  -ldflags "-s -w -X github.com/leolaurindo/gixt/internal/version.Version=${VERSION}" \
  -o dist/gixt${{ matrix.ext }} ./cmd/gixt
```

Acceptance criteria:

- All existing release targets still build.
- `gixt --version` still prints the tag version.
- Release archives and checksums are still generated.
- Compare representative binary size before and after and record the result in the pull request.

## 17. Command Simplification Policy

New behavior should not create a new command whenever a shared concept exists.

Rules:

1. `cat` and `run` are the only artifact actions in the core workflow.
2. `list` is read-only; `remove --owner` owns confirmed bulk removal from the known store.
3. `add` and `remove` own known-store mutations.
4. `gist show` owns metadata inspection.
5. `cache` exposes only explicit cache maintenance, not normal retrieval.
6. `trust` exposes execution approval state only.
7. Flags shared by `cat` and `run` must use the same names and semantics; `--entry` selects one file in either command, while multiple files for `cat` are supplied as independently resolved target operands.
8. Do not add a separate `prompt` namespace; prompts are plain-text artifacts.
9. Do not restore a manifest system for this feature.

## 18. stdout, stderr, and Exit Codes

### stdout

Reserved for consumable output:

- raw selected file content from `cat`;
- executed process stdout from `run`;
- structured/table output from explicit listing and inspection commands.

### stderr

Reserved for:

- diagnostics;
- progress;
- warnings;
- trust prompts;
- deprecation messages;
- errors.

### Exit behavior

- `cat` returns non-zero for resolution, acquisition, selection, or output errors; binary content is not an error.
- `run` continues to propagate execution failures and signals through the existing runner behavior.
- A missing explicit entry is distinct from a missing target in diagnostics.
- Ambiguity is never treated as not-found for typo suggestions.

## 19. Documentation Plan

Update:

- `README.md`: new positioning, prompt-first examples, explicit `run` examples.
- `COMMANDS.md`: exact implemented command inventory.
- `docs/cli-usage.md`: target versus entry mental model.
- Add `docs/targets-and-entries.md` only if the explanation cannot remain concise in `cli-usage.md`.
- `docs/index.md`: link prompt retrieval examples.
- `CHANGELOG.md`: compatibility notes and deprecations.

Documentation must include:

1. Alias identifies a Gist; entry identifies a file.
2. How exact filename targets behave.
3. How to resolve collisions with `--as` and `--entry`.
4. Why `cat` does not require trust (`print` is an alias).
5. How stdout/stderr make piping safe.
6. Bare invocation now retrieves content; execution requires `gixt run` and this is an intentional breaking change.
7. Binary output is passed through byte-for-byte by default.
8. Offline and pinned prompt retrieval examples.
9. The difference between `add mine`, `add owner`, `list`, and `trust mine`.
10. The human-readable list table and its stable stdout/stderr behavior.

`OVERHAUL.md` describes an older, broader proposal and conflicts with parts of the current implemented tree. Mark it historical or remove it once this specification is accepted so two active-looking designs do not coexist.

## 20. Test Plan

### 20.1 Resolution tests

Add coverage for:

- ID and URL resolution;
- exact alias precedence;
- unique exact filename resolution with `RequestedFile`;
- stem resolution without incorrect entry inference;
- owner-qualified lookup;
- same-owner filename collisions;
- cross-owner filename collisions;
- aliases colliding with filenames;
- offline owner lookup refusal;
- platform variants not selecting between different Gists;
- typed not-found and ambiguity errors.

The current resolution suite has only narrow offline coverage (`internal/cli/resolve_test.go`). Expand it before changing root dispatch.

### 20.2 Entry-selection tests

Test:

- explicit entry precedence;
- inferred exact filename precedence;
- missing explicit entry with available-file diagnostics;
- `main.*` preference;
- `index.*` preference;
- platform shell preference;
- lexical fallback;
- deterministic behavior from unsorted input;
- empty Gist behavior.

Move/adapt existing runner selection tests rather than duplicating them.

### 20.3 Cat tests

Test:

- one selected file reaches stdout byte-for-byte;
- multiple positional targets are resolved independently and concatenated in command-line order, whether they resolve to the same or different Gists;
- filename targets use normal resolution and preserve ambiguity errors;
- a Gist target without `--entry` contributes all its files in lexical order;
- `--entry` selects one exact file with a single target;
- `--entry` is rejected with multiple targets;
- exact filename targets select that entry across same- and different-Gist combinations;
- no separator or extra newline is added;
- diagnostics do not reach stdout;
- explicit and inferred entries;
- pin and `--ref` behavior;
- offline cached retrieval;
- no-cache cleanup;
- no trust prompt or trust-store mutation;
- binary content passes through byte-for-byte by default;
- broken stdout/write error where testable.

### 20.4 Run regression tests

Test that the refactor preserves:

- Python override;
- shebang resolution;
- extension mapping;
- forwarded arguments;
- isolate versus current-directory execution;
- trust behavior;
- timeout behavior;
- cache pruning;
- `--add` and `--as` after successful execution;
- selected filename execution.

### 20.5 Dispatch tests

Test:

- exact commands win;
- command-like valid targets resolve;
- typo suggestion only after target not-found;
- ambiguity remains ambiguity;
- network/auth/offline errors are preserved.

### 20.6 List tests

Test:

- local listing;
- owner listing;
- default limit;
- custom limit;
- all-pages behavior;
- mutually exclusive flags;
- listing does not mutate the local store;
- `add owner` reports page progress on stderr and leaves the known store unchanged if a page fails;
- `add mine` resolves the authenticated owner, applies the same progress and atomic persistence rules, and rejects missing authentication;
- `remove --owner` warns and requires confirmation, refusing in non-interactive mode unless `--yes` is supplied;
- owner removal leaves GitHub Gists, cache, and trust approvals untouched;
- `remove <target>` resolves aliases, IDs, URLs, owner/name, and filenames normally, including a target named `clear`.

### 20.7 Trust and release tests

Test:

- progress is on stderr;
- no partial trust persistence on failure;
- trust revision fetches never exceed three concurrent workers;
- concurrency preserves ordering, responds appropriately to rate limits, and does not persist partial trust on failure;
- tagged version injection after stripped build.

Before each implementation PR is considered complete, run:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./cmd/gixt
sh -n docs/install.sh
```

Run the PowerShell installer syntax check in CI on Windows.

### 20.8 List presentation tests

Test:

- local and remote listings use the same column order where practical;
- headers and rows remain aligned for short and long owners, names, and descriptions;
- aliases and lexical filename fallbacks populate the `NAME` column correctly;
- IDs use the existing shortened display convention;
- rows are deterministic and contain no color or terminal control sequences;
- table data stays on stdout and diagnostics stay on stderr;
- presentation changes do not mutate the known store or alter list limits and pagination.

## 21. Ordered Implementation Plan

### 21.1 Work-package progress

Keep this ledger current in the same PR that changes a work-package status. It is the quick status view; the commit or PR reference provides traceability without requiring checkout inspection.

| Work package | Status | Evidence or next step |
|---|---|---|
| WP1 — Target and entry foundation | Done | Merged to `dev` in PR #12 (`99cdb74`) |
| WP2 + WP3 — `cat` retrieval and root dispatch | Done | Merged to `dev` in PR #13 (`760129e`) |
| WP4 + WP7 — Owner management and `add mine` | In progress | WP4 implemented on `feat/owner-management`; WP7 pending |
| WP5 — Trust snapshot progress and concurrency | Planned | Implement on `feat/trust-mine` |
| WP6 — Release binary size | Planned | Implement on `feat/strip-release-binaries` |
| WP8 — Human-readable list output | Planned | Implement on `feat/list-table` after WP4 + WP7 |

### 21.2 Branch and PR Work Packages

Keep this file as the single source of truth. Assign agents a work-package ID and its in-scope sections; do not copy the specification into branch-specific documents. Each feature branch should start from the latest `development` and open a PR back to `development`, following the promotion flow in section 22.

| Work package | Suggested branch | Scope and sections | Depends on |
|---|---|---|---|
| WP1 — Target and entry foundation (#7) | `feat/target-entry` | Shared target resolution and typed errors (§7–9), run entry selection (§12), tests (§20.1–20.2) | None; do first |
| WP2 + WP3 — `cat` retrieval and root dispatch (#6, #5) | `feat/cat-dispatch` | Shared artifact pipeline and `cat` behavior, independently resolved targets, single-target `--entry`, byte-exact multi-target output, bare-target dispatch, command-like target handling, and typo suggestions (§6.2, §10–13), tests (§20.3, §20.5) | WP1 |
| WP4 + WP7 — Owner management and `add mine` (#8) | `feat/owner-management` | Remote `list <owner>`, pagination, confirmed `remove --owner`, `add owner` page progress, and authenticated-user registration shorthand (§6.1, §14, §14.3), tests (§20.6) | WP1 recommended for the final normal-resolution contract used by `remove <target>` |
| WP5 — Trust snapshot progress and concurrency (#4) | `feat/trust-mine` | Progress, bounded revision-fetch workers, rate-limit handling, atomic persistence (§15), tests (§20.7) | None |
| WP6 — Release binary size (#9) | `feat/strip-release-binaries` | Release build flags and version/size validation (§16, §20.7) | None |
| WP8 — Human-readable list output | `feat/list-table` | Compact deterministic table presentation for local and remote listings without changing listing semantics (§14.4), tests (§20.8) | WP4 + WP7; presentation-only branch |

Recommended merge order is WP1 → combined WP2 + WP3. WP4 + WP7 can follow WP1 and proceed in parallel with the combined cat/dispatch branch. WP8 follows WP4 + WP7 and remains presentation-only. WP5 and WP6 can proceed independently. Each branch should start from the latest `development` and open one PR back to `development`. There is no separate WP3 branch: root dispatch is implemented in the same `feat/cat-dispatch` branch as `cat`. Agents should avoid shared-file conflicts and rebase on the latest `development` before opening or updating their PRs. Each PR should implement only its assigned behavior plus directly required tests/docs; leave unrelated work for its own package.

For agent handoff, use a prompt like: “Implement WP4 + WP7 from `SPEC.md` §14 and §14.3 and §20.6. Follow the dependency and scope in §21.2; do not implement other work packages. Run the required checks listed in §20.”

#### Keep network helpers use-case-specific

- Owner-add progress reports the page and cumulative count for `gixt add owner`; reuse the existing pager and keep the progress callback local to this path.
- Trust progress and the three-worker limit apply only to `gixt trust mine` revision fetching.
- Do not add generic progress, spinner, retry, or concurrency frameworks for these features. Handle GitHub rate limits where the trust fetches occur, and avoid adding concurrency to owner pagination or `cat` without a separate measured need.

Section 22 is repository workflow guidance, not another issue work package. If its workflow-file changes are needed, make them a separate infrastructure PR; GitHub rulesets/branch-protection settings require repository-admin configuration outside the code PR.

### 21.3 Execution order

- Implement WP1 first, then implement the combined WP2 + WP3 branch in dependency order. Include the bare-command breaking change and migration notes in that combined branch.
- WP4 + WP7 follows WP1. WP8 follows WP4 + WP7. WP5 and WP6 are independent and may run in parallel.
- Each work-package branch must pass its listed tests. Before promotion, run the full checks in §20.

## 22. Git and CI Workflow

- Standard promotion: `feat/* -> development -> main`; occasional direct commits to `development` may go to `main` through a PR.
- Disable CI on every push. Run `.github/workflows/ci.yml` for PRs targeting `development` or `main`, plus `workflow_dispatch`; avoid path filters that leave required checks pending.
- Protect `main`: require PRs and CI, block direct/force pushes and deletion, and require the branch to be up to date. Protect `development` against force pushes/deletion and require CI on feature PRs; direct commits may remain available for the expedited flow.
- Keep release packaging in `.github/workflows/release.yml` on `v*` tags (plus manual dispatch). Merge and test the promotion to `main`, create an annotated tag on that commit, then push the tag.
- Re-run CI on the `development -> main` PR: its merge-result SHA may differ from the feature PR. Optimize away per-push CI, not validation of a different commit.

GitHub rulesets/branch-protection settings require repository-admin configuration; they are not implemented by editing workflow YAML alone.

## 23. Completion checklist

- Target resolution, exact filename propagation, and deterministic entry selection are shared by `cat` and `run`; ambiguity is never silently resolved.
- `cat` resolves each target independently, emits exact bytes without separators, and never executes or prompts. Bare targets use `cat`; execution requires `run`.
- Owner listing is read-only; `remove --owner` only changes local known entries; `add mine` reuses atomic owner registration; owner-add progress and trust snapshots persist only after successful completion.
- Local and remote listings use deterministic human-readable tables without changing their read-only semantics.
- `trust mine` uses at most three revision-fetch workers and respects rate limits; release binaries are stripped without losing version injection.
- CLI docs and changelog explain the command behavior and breaking bare-target change.
- The WP-specific tests in §20 and full checks (`go test ./...`, `go vet ./...`, `go build ./cmd/gixt`) pass.
