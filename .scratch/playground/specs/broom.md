# Spec: broom

Status: ready-for-agent

The command-line tool used to configure, edit, and write content. `broom` is a
client of `custodian`'s API and has no storage or authority of its own — it only
moves content from the author's editor into `custodian`. It is the custodian's
own implement: humble on purpose.

This spec synthesises the resolved decision tickets `06` (name) and `17`
(language, UX & distribution), grounded in the `03` domain model and in the
contract `broom` is a client of: `custodian`'s API (`09`), auth model (`10`),
storage model (`08`, revised by `12`), and the monorepo tooling (`13`). Where a
later ticket revised an earlier one, the revised position is stated here as the
single source of truth. `broom` builds nothing of `custodian`'s; where this spec
touches the contract it records `broom`'s *use* of it, and the authoritative
contract lives in `specs/custodian.md`.

## Problem Statement

The author needs to write and maintain the site's content — long-form blog
posts and their images, a profile, and the credentials behind the live-status
widgets — without the two pains the deprecated site imposed: editing meant
rebuilding, and a secret was shipped to the browser. `custodian` solves both by
owning content behind an API, but an API is not an authoring experience. Someone
has to drive it.

The real risk this component carries: **it is the one most likely to be quietly
abandoned if the UX is poor.** A CMS the author finds annoying is a CMS the
author stops publishing through, which would defeat the entire rebuild. So the
authoring workflow and — especially — media ergonomics carry more weight than
any language or distribution question. The single most annoying part of most
self-built CMSes is uploading an image and then hand-pasting an opaque id into a
post body; `broom` must not have that.

`broom` must also never hold an AWS credential and never hold a third-party key.
Its only counterparty is `custodian`, over an authenticated API.

## Solution

`broom` is a single static Go binary that speaks only to `custodian`'s admin API
(`/admin/v1/*`) using the *same* generated OpenAPI client `custodian` publishes
(`09`/`13`). It is a pure API client: no local content repository, no renderer,
no preview host, no `blank`/`persona` dependency.

Its command surface is **noun-grouped verbs** (`logs`, `media`, `profile`,
`integration`) plus top-level auth verbs (`login`, `logout`). The authoring loop
is: `broom logs new` prompts for metadata, creates the post *immediately* as an
`unlisted` draft (the slug is the post — the invariant is *as long as there is a
slug, there is a post*), then launches `$EDITOR` on the body and `PATCH`es on
save. `custodian` is canonical; there is nothing to keep on disk, so there is no
local git repo of posts and `08`'s point-in-time recovery is the safety net.

Media is a deliberately separate gesture, run in a second terminal while the post
sits open in the editor: `broom media add <file>` runs `custodian`'s
presigned-upload flow and **prints and clipboard-copies legible markdown** —
`![](https://cdn.mihirsingh.dev/media/<key>)`. The author-given kebab-case key
is what makes the reference legible instead of opaque; that legibility is the
whole point.

There is **no local preview** — `unlisted` semantics make it redundant, since
every fetched log is public and link-reachable, so preview *is* edit → save →
refresh the unlisted slug on real `persona`. Distribution in v1 is **build from
source** via a root `just install` recipe; published install stories are fog.
Configuration is a **single XDG config file holding `url` + `token`**, with
`BROOM_URL`/`BROOM_TOKEN` env overrides and a baked-in default URL.

## User Stories

### Authentication

1. As the author, I want `broom login` to prompt for a token and verify it with a single authenticated call to `custodian`, so that I learn immediately whether the credential works rather than at first write.
2. As the author, I want a successful `login` to persist `url` + `token` to my config file with `0600` permissions, so that subsequent commands need no re-auth.
3. As the author, I want `broom logout` to remove the stored token, so that I can clear the credential from a machine.
4. As the author, I want any command to fail with a clear "not logged in / token rejected" message when no valid credential is available, so that auth failures are never mistaken for content bugs.
5. As the author, I want `BROOM_TOKEN` and `BROOM_URL` environment variables to override the config file when set, so that I can run an ad-hoc command against a different token or custodian without editing config.

### Authoring logs

6. As the author, I want `broom logs new` to prompt me for metadata (title, subtitle, tags, and the optional plain-prose `description`), so that a new post is described before I start writing.
7. As the author, I want `broom logs new` to create the post immediately as an `unlisted` draft the instant metadata is entered, so that the slug-is-the-post invariant holds and there is never an in-limbo post with no server record.
8. As the author, I want `broom logs new` to then launch `$EDITOR` on the (initially empty) body and `PATCH` the body to custodian when I save and close, so that I write in my own editor and publishing the body is a single save.
9. As the author, I want an empty body to be allowed — including abort-without-writing — so that a freshly created post is a valid empty `unlisted` draft rather than an error.
10. As the author, I want `broom logs edit <slug>` to pull the current body into a temp file, open `$EDITOR`, and `PATCH` on save, so that editing an existing post round-trips identically to creating one.
11. As the author, I want `broom logs list [--listed|--unlisted]` to show my posts of any state (including drafts the public index hides), so that I can find and manage work in progress.
12. As the author, I want `broom logs publish <slug>` and `broom logs unpublish <slug>` to toggle a post's state via a `PATCH`, so that going live or pulling back is one command with no bespoke endpoint.
13. As the author, I want `broom logs rm <slug>` to delete a post, so that I can remove a draft or a retired post entirely.
14. As the author, I want a slug-collision error on create to be surfaced legibly (from custodian's `slug_conflict` code) with a prompt to choose another slug, so that I recover in-flow rather than reading a raw HTTP error.
15. As the author, I want an attempt to rename a `listed` post's slug to be reported clearly (custodian's `slug_frozen_while_listed`), so that I understand published links are deliberately frozen.

### Media

16. As the author, I want `broom media add <file> [--key k]` to run custodian's reserve → upload → confirm flow over a presigned S3 `PUT`, so that bytes go straight to S3 and `broom` never needs an AWS credential.
17. As the author, I want `broom media add` to print and clipboard-copy legible markdown `![](https://cdn.mihirsingh.dev/media/<key>)` on success, so that I paste a readable reference into the open post body instead of an opaque id.
18. As the author, I want to supply my own kebab-case `--key`, so that the reference in my prose is meaningful; and I want to omit it and have custodian mint a random kebab key when I do not care.
19. As the author, I want a duplicate key to be reported clearly (custodian's `media_key_taken`), so that I never silently overwrite an existing asset.
20. As the author, I want `broom media ls` to list and search existing media, so that I can find and reuse an asset rather than re-uploading it.
21. As the author, I want `broom media rm <key>` to delete a media record, and I want `broom` to first scan my posts for references to that key and warn me before deleting, so that I don't orphan an image a live post still points at.

### Profile

22. As the author, I want `broom profile get <key>` to fetch a profile record (`about`, `experience`, `skills`, `resume-link`, `curated-projects`) and show its raw JSON, so that I can inspect the current value.
23. As the author, I want `broom profile edit <key>` to open the record's raw JSON in `$EDITOR` and `PUT`-upsert it on save, so that I control the profile shape by convention with `persona`, with no schema imposed by `broom`.

### Integrations

24. As the author, I want `broom integration refresh [name]` to force custodian to poll a source (`steam`, `github`) immediately and print the fresh record, so that I can verify a newly rotated Steam key or a fixed GitHub PAT without waiting for the next tick.
25. As the author, I want to write a third-party integration credential through `broom` (an authed admin API call), so that key rotation is a terminal gesture that takes effect on the next poll with no custodian restart.

### Errors and output

26. As the author, I want `broom` to render custodian's RFC 9457 `problem+json` errors as a legible message — the `detail` string, branching on the stable `code`, and listing any field-errors — so that failures read as guidance, not as raw HTTP.
27. As the author, I want a network or custodian-down failure to be reported as clearly distinct from a rejected request, so that "custodian is unreachable" and "custodian said no" are never conflated.

### Distribution

28. As the author, I want to build and install `broom` from source with a root `just install` recipe, so that getting it onto a machine needs only the repo and the Go toolchain, with no published artifact.

## Implementation Decisions

### Name (`06`)

- The CLI is **`broom`** — the custodian's own implement, so `broom` and
  `custodian` read as a deliberate set (`broom` exists only to drive
  `custodian`'s API). Humble on purpose, five characters, natural in every
  command position, and it shadows no standard Unix command (no PATH hazard).
- **Namespace note carried from `06`:** the bare npm name `broom` is squatted by
  an abandoned ~2012 package, so *if* `broom` ever ships via npm it must use a
  scope (`@broom/cli` and `@mihirs16/broom` are both free). This does not bind
  v1, which builds from source (Go module paths are repo-namespaced, a
  non-issue).

### Language and client (`17`, depends on `07`/`09`/`13`)

- **Go**, compiled to a single static binary. Chosen because `broom` consumes
  the **same generated OpenAPI client** `custodian` already produces from its one
  `.yaml` (`09`), fanned out by `13`'s `just gen` into a vendored,
  committed, CI-drift-checked Go client. Client-and-types sharing was decisive
  over Rust (a second client + unverified crates.io name) and TypeScript (drags
  in Node, kills the single-binary story, and `persona`'s language was not even
  settled at the time). One language across backend + tooling.
- `broom` is a **pure API client**: it talks only to `custodian`'s admin surface,
  holds no AWS credential, holds no third-party key, and depends on neither
  `blank` nor `persona`.

### Authoring workflow (`17`, invariants from `09`/`10`/`08`)

- **`broom logs new`** → interactive metadata prompts → **creates the post
  immediately as `unlisted`** (slug is identity per `09`; invariant: *as long as
  there is a slug, there is a post*) → launches **`$EDITOR`** on the body →
  **`PATCH` on save**.
- **`broom logs edit <slug>`** round-trips an existing body through a temp file
  identically: pull → `$EDITOR` → `PATCH` on save.
- **`custodian` is canonical — there is no local git repo of posts.** The
  `$EDITOR`-launching gesture was chosen *over* standing local files precisely
  because canonical-is-custodian means there is nothing to keep on disk. `08`'s
  Litestream PITR + 30-day S3 versioning is the safety net, not a git working
  copy.
- **Empty bodies are allowed.** Every post is born `unlisted`; aborting the
  editor without saving leaves a valid empty `unlisted` post.
- **State transitions are `PATCH {state}` under the hood** — `logs publish` →
  `listed`, `logs unpublish` → `unlisted` (`10`'s two-state model; there is no
  third preview state). `broom` exposes them as verbs; there are no bespoke
  publish/unpublish endpoints on custodian.
- **Slug rename** is possible only while `unlisted` and is a server-performed
  move (`09`); `broom` surfaces the frozen-while-listed rejection legibly.

### Media ergonomics (`17`, flow from `09`, revises `08`)

- **A wholly separate gesture** from authoring, run in a second terminal while
  the post sits open in the editor: **`broom media add <file> [--key k]`**.
- It runs `09`'s **reserve → upload → confirm** flow over a presigned S3 `PUT`
  (`custodian` reserves a `pending` record with key-uniqueness enforced, returns
  a presigned URL, `broom` `PUT`s the bytes to S3, then confirms so custodian
  `HEAD`s S3 and flips the record to `available`). **`broom` holds no AWS
  credentials** — it only talks to custodian and to the presigned URL custodian
  hands back.
- On success it **prints and clipboard-copies legible markdown**
  `![](https://cdn.mihirsingh.dev/media/<key>)`. The **author-given kebab key**
  is the ergonomic win — a legible reference instead of an opaque id, which is
  the single-most-annoying-part-of-self-built-CMSes problem solved by
  legibility. Omitting `--key` lets custodian mint a random kebab key.
- **Revises `08`: the media URL gains a `/media/` path prefix**
  (`cdn.mihirsingh.dev/media/<key>`, was bare `cdn.mihirsingh.dev/<key>`),
  namespacing the single CloudFront distribution's root so other path prefixes
  stay free. (Already reflected in `specs/custodian.md`.)
- **`broom media rm <key>`** performs a **`broom`-side reference scan** of post
  bodies before delete, as a courtesy warning — custodian does not parse bodies
  for URLs, and S3 versioning is the real safety net (`08`).

### Command surface (`17`)

Noun-grouped verbs throughout; `login`/`logout` stay top-level as auth verbs with
no noun object.

```
broom login / logout                             # 10: verify token via one authed call

broom logs new                                   # prompts metadata → unlisted post → $EDITOR body
broom logs edit <slug>                           # pull body → $EDITOR → PATCH on save
broom logs list [--listed|--unlisted]
broom logs rm <slug>
broom logs publish <slug> / unpublish <slug>     # state toggle (listed/unlisted, 10)

broom media add <file> [--key k]                 # reserve→upload→confirm, print + clipboard ![](…/media/k)
broom media ls
broom media rm <key>                             # broom ref-check before delete (08)

broom profile edit <key> / get <key>             # 03: raw JSON in $EDITOR → PUT-upsert, opaque
broom integration refresh [name]                 # manual /refresh (09/12); polling is otherwise automatic
```

- **`profile edit <key>`** opens the record's **raw JSON** in `$EDITOR` and
  `PUT`-upserts it (`03`); the body is opaque — `broom` imposes no schema, the
  shape is convention between the author and `persona`.
- **`integration refresh`** forces custodian's manual `/refresh` (`09`/`12`);
  routine polling is otherwise automatic on custodian's 5-minute tick and needs
  no `broom` involvement.

### No local preview (`17`)

- **`broom` ships no renderer and no preview host.** `<blank-markdown>` *could*
  have been reused for a `broom`-hosted local page without API drift, but the
  `unlisted` semantics make a local preview **redundant**: every fetched log is
  public and link-reachable (`10`), so preview = edit → save → refresh the
  `unlisted` slug on real `persona`. This keeps `broom` a pure API client with no
  `blank`/`persona` dependency.

### Configuration (`17`, extends `10`)

- **A single XDG config file holds `url` + `token` together.** `10` settled the
  token half (a `0600` file, `BROOM_TOKEN` override); `17` adds the URL beside
  it.
- **Baked-in default URL**, so a fresh install needs only a token. **`BROOM_URL`
  mirrors `BROOM_TOKEN`** as an env override.
- **No staging / named-environment profiles in v1** — there is no staging
  custodian in any decision; the env overrides cover ad-hoc use. Named profiles
  (`broom --env staging`) are fog.

### Distribution (`17`, uses `13`)

- **Build from source via a root `just install` recipe** (`13`'s thin root
  justfile delegating to the Go toolchain).
- **No published distribution in v1.** Homebrew tap / GoReleaser-built GitHub
  Release binaries / `go install` on a public module path are all deferred to
  fog until a *second machine or another user* actually needs `broom` installed
  rather than built, at which point the npm-scope note from `06` applies only if
  npm is the chosen channel.

## Testing Decisions

**What makes a good `broom` test:** it asserts on **what `broom` sends to
`custodian` and what it does with the response** — the HTTP requests issued
(method, path, headers, body), and the observable local effects (the file handed
to `$EDITOR`, the config written, the text printed to stdout, the string placed
on the clipboard) — never on internal function calls or private structs. `broom`
is a thin client; its contract is *the sequence of API calls a workflow makes*
plus *its editor/clipboard/stdout side effects*.

**The single seam: the `custodian` API boundary.** Tests drive `broom`'s
commands against a **fake custodian** — an in-process `httptest` server that
speaks the same OpenAPI contract (ideally stood up from the same `.yaml`, so
contract drift is caught) — and assert on the requests it receives and the
responses `broom` acts on. This exercises the real generated client and the real
command wiring, faking only the thing that leaves the machine.

**The three local edges are faked** (injected at the boundary), because they
touch the developer's environment and must be deterministic in a test:

1. **`$EDITOR`** — replaced by a scripted fake that writes known body/JSON
   content (or exits without writing, for the empty-body and abort cases), so
   tests exercise `logs new`/`logs edit`/`profile edit` end to end.
2. **The clipboard** — a fake sink lets tests assert `media add` copies the exact
   legible `![](…/media/<key>)` string it also prints.
3. **The presigned S3 `PUT`** — the fake custodian hands back a presigned URL
   pointing at the test server (or a fake S3), so the reserve → upload → confirm
   sequence is driven deterministically without real S3.

**Workflows under test** (through the one seam): the full `logs new` loop
(metadata prompt → immediate `unlisted` create → `$EDITOR` → `PATCH` on save,
including the empty-body/abort path); `logs edit` round-trip; publish/unpublish
as `PATCH {state}`; the `media add` reserve→upload→confirm sequence with key and
random-key variants; the `media rm` reference-scan warning; `profile
get`/`edit` opaque JSON round-trip; `integration refresh`; `login` verifying via
one authed call and writing `0600` config; the `BROOM_TOKEN`/`BROOM_URL` override
precedence; and `problem+json` error rendering (branching on `code`, listing
field-errors, and distinguishing custodian-down from custodian-said-no).

**Prior art:** there is no existing `broom` test suite — this spec's
implementation establishes the pattern. It should follow idiomatic Go
`net/http/httptest` tests, using the generated OpenAPI Go client (shared with
custodian per `13`) as the transport under test so the tests exercise the
published contract.

## Out of Scope

- **Building `custodian`, `persona`, `blank`, or `deed`.** This spec is `broom`
  only; where it touches `custodian` it records `broom`'s *use* of the contract,
  not custodian's implementation (which lives in `specs/custodian.md`).
- **A local content repository / git working copy of posts** — `custodian` is
  canonical; `broom` keeps no posts on disk.
- **A local renderer or preview server** — dropped deliberately; `unlisted`-on-
  real-`persona` is the true-fidelity preview.
- **Published distribution** — Homebrew tap, GoReleaser GitHub-Release binaries,
  `go install` on a public path (and the npm scope from `06` if npm is ever the
  channel) — all fog until a second machine or another user needs `broom`
  installed rather than built.
- **Staging / named-environment profiles** (`broom --env staging`) — fog; there
  is no staging custodian, and env overrides cover ad-hoc use.
- **OS keychain integration for the token** — `10` settled on a `0600` file, not
  a keychain; a keychain is not v1.
- **Media pipeline beyond raw upload** — optimisation, responsive variants,
  format conversion — is custodian/`blank`/`persona` territory, not `broom`'s;
  `broom` uploads raw bytes and prints a reference.
- **Authoring the OpenAPI `.yaml` or the generated client** — that is
  custodian/`13` work; `broom` *consumes* the generated client.
- **The `snippet` short-form type, LaTeX/math bodies, and any non-GFM-core
  syntax** — future features, not v1; `broom` sends body text verbatim and adds
  no syntax of its own.

## Further Notes

- **The abandonment risk is the design driver.** `17` names `broom` the component
  most likely to be quietly abandoned if the UX is poor — a CMS you find annoying
  is one you stop publishing through, defeating the rebuild. Every friction point
  removed (immediate-`unlisted` create so there is never a limbo draft, `$EDITOR`
  so the author writes where they already write, legible media keys, no preview
  ceremony) serves that one goal. Weight authoring and media ergonomics above
  language and distribution when trade-offs arise.
- **The slug-is-the-post invariant, stated once:** `broom logs new` creates the
  server record *before* the editor opens, so there is no window in which the
  author has written a post that custodian does not know about. Every post has a
  slug; every slug has a post.
- **`broom` never holds a secret that isn't its own.** No AWS credential (media
  goes via custodian's presigned URL), no third-party key (those live in
  custodian's SQLite per `10`/`19`, written *through* `broom`'s authed API call
  but never *stored* by `broom`). `broom`'s only secret is its own bearer token,
  in a `0600` file.
- **Two things share the same underlying `PATCH`:** editing a body and toggling
  state are both `PATCH /admin/v1/logs/{slug}` on the wire (`09`); `broom` merely
  presents them as distinct verbs (`logs edit` vs `logs publish`). Don't invent
  separate custodian endpoints for them.
- **The `/media/` prefix originated here.** `17` (this component's UX work) is
  what revised `08`'s bare `cdn.mihirsingh.dev/<key>` to
  `cdn.mihirsingh.dev/media/<key>`; the change is already carried in
  `specs/custodian.md`, noted here so the provenance isn't lost.
