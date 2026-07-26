# CLI: language, UX and distribution

Type: grilling
Status: open
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
