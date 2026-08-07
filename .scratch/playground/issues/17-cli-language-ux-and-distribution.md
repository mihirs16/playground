# CLI: language, UX and distribution

Type: grilling
Status: resolved
Blocked by: 06, 07, 09

## Question

What language is the cli, what does using it feel like, and how does it get onto a machine?

- **Language** — Go and Rust both give a single distributable binary with no runtime to install. TypeScript/Node means `npx` and a Node dependency, but shares the API client with persona if persona is also TypeScript. If custodian is Go, the cli sharing its client and types is a strong argument.
- **Command surface** — what are the verbs? Sketch the real sessions: publishing a new post with two images, fixing a typo in a published post, updating your Steam link or profile, uploading media and getting a reference to paste into markdown.
- **Authoring workflow** — the important UX question. Do you write markdown in your own editor and the cli uploads a file, or does the cli open `$EDITOR`, or is it interactive prompts? File-based is the likely fit given the content is markdown on disk, and it composes with git and your existing editor.
- **Media ergonomics** — how do you upload an image and get its reference into a post body without copy-pasting an opaque id? This is the single most annoying part of most self-built CMSes, and worth designing deliberately.
- **Local preview** — can the cli render a draft locally, or does that belong to persona's dev server?
- **Distribution** — Homebrew tap, `go install`, GitHub Releases with binaries, npm? What's the actual install story for your own machines. **Note from `06`**: the bare npm name `broom` is squatted by an abandoned 2012 package, so npm distribution would need a scope (`@broom/cli` and `@mihirs16/broom` are both free). Homebrew and Go module paths are clear. crates.io was never verified — check by hand if this turns out to be Rust.
- **Config** — where the custodian URL and credentials live (coordinate with `10`), and how multiple environments are handled if there's ever a staging custodian.

## Context

Blocked on `06` (the name appears throughout), `07` (the client-sharing argument depends on custodian's language), and `09` (the cli is a client of the contract, so it can't be designed before the contract exists).

This is the component most likely to be quietly abandoned if the UX is poor — a CMS you find annoying is a CMS you stop publishing through, which would defeat the entire rebuild. The authoring workflow and media ergonomics questions deserve more weight than the language question.

## Answer

**Language: Go.** Consumes the *same* generated OpenAPI client as custodian (`09`'s one `.yaml`, fanned out by `13`'s `just gen`) — client-and-types sharing was decisive over Rust (second client, unverified crates.io name) and TS (drags in Node, kills the single-binary story, and persona's language isn't even decided). Single static binary, one language across backend + tooling.

**Authoring workflow.** `broom logs new` → interactive metadata prompts → creates the post immediately as **`unlisted`** (slug is identity per `09`; invariant: *as long as there's a slug, there's a post*) → launches **`$EDITOR`** on the body → **`PATCH` on save**. `broom logs edit <slug>` round-trips an existing body through a temp file identically. **Custodian is canonical — there is no local git repo of posts**; `08`'s Litestream PITR + 30-day S3 versioning is the safety net. **Empty bodies are allowed** (every post is born unlisted); abort-without-save leaves a valid empty unlisted post. The `$EDITOR`-launching gesture was chosen over standing files precisely because custodian-is-canonical means there's nothing to keep on disk.

**Media ergonomics.** A wholly separate gesture, run in another terminal session while the post sits open in the editor: `broom media add <file> [--key k]` runs `09`'s reserve→upload→confirm over presigned S3 `PUT`, then **prints and clipboard-copies legible markdown** `![](https://cdn.mihirsingh.dev/media/<key>)`. The **author-given kebab key** is what makes the reference legible instead of an opaque id — the single most-annoying-part-of-self-built-CMSes problem, solved by legibility. broom holds no AWS creds (only talks to custodian). **Revises `08`: the media URL gains a `/media/` path prefix** (`cdn.mihirsingh.dev/media/<key>`, was bare `cdn.mihirsingh.dev/<key>`), namespacing the single CloudFront distribution's root so other path prefixes stay free.

**Command surface** — noun-grouped throughout:

```
broom login / logout                       # 10: verify token via one authed call

broom logs new                             # prompts metadata → unlisted post → $EDITOR body
broom logs edit <slug>                     # pull body → $EDITOR → PATCH on save
broom logs list [--listed|--unlisted]
broom logs rm <slug>
broom logs publish <slug> / unpublish <slug>   # state toggle (listed/unlisted, 10)

broom media add <file> [--key k]           # reserve→upload→confirm, print + clipboard ![](…/media/k)
broom media ls
broom media rm <key>                        # broom ref-check before delete (08)

broom profile edit <key> / get <key>       # 03: raw JSON in $EDITOR → PUT-upsert, opaque
broom integration refresh [name]           # manual /refresh (09/12); polling is otherwise automatic
```

`login`/`logout` stay top-level (auth verbs, no noun object).

**Local preview: none in broom.** `<blank-markdown>` (`14`) *could* have been reused for a broom-hosted local page without drift, but the **`unlisted` semantics make it redundant**: every fetched log is public and link-reachable (`10`), so preview = edit → save → refresh the unlisted slug on real persona. broom ships no renderer/preview host, staying a pure API client with no `blank`/persona dependency.

**Distribution: build from source via a root `just install` recipe** (`13`'s thin root justfile). No published distribution yet — Homebrew tap / GoReleaser / GitHub Release binaries / `go install` publishing all deferred to fog until a second machine or another user needs them.

**Config: a single XDG config file holds `url` + `token` together** (extends `10`, which settled the token half — `0600`, `BROOM_TOKEN` override). Baked default URL so a fresh install needs only a token; `BROOM_URL` mirrors `BROOM_TOKEN` as an env override. **No staging / named-environment profiles in v1** — there is no staging custodian in any decision; the env overrides cover ad-hoc use, and named profiles (`broom --env staging`) are fog.

Feeds a small revision back to `08` (media URL `/media/` prefix). Published-distribution work graduates to fog.
