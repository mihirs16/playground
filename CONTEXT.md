# Context

The glossary for **playground** — a polyglot monorepo and digital space for learning, experimenting and practising smaller software concepts. It replaces a deprecated Next.js site that used Notion as a lite CMS.

Use these terms exactly. Where a synonym is listed as avoided, avoid it.

## The components

The five names are a deliberate set, not five independent choices. Each is a single evocative English noun, and the relationships between them are legible from the names alone.

### `custodian`

The API that keeps the playground. It owns all content, serves it to `persona` at runtime, accepts writes from `broom`, and is the only component holding third-party credentials.

The keeper of the playground — hence `broom`, its implement.

_Avoid_: "the backend", "the CMS", "the API" unqualified.

### `broom`

The command-line tool used to configure, edit and write content. It is a client of `custodian`'s API and has no storage or authority of its own.

The custodian's own implement — humble on purpose, because it only moves content from your editor into `custodian`.

_Avoid_: "the admin tool", "the CLI" unqualified.

### `blank`

The Web Components UI library, published to public npm with semver. Consumed by `persona` and intended for other personal projects.

Named for the aesthetic it enforces: monotone text, a single highlight yellow, and a great deal of blank space.

_Avoid_: "the design system", "the component library" unqualified.

### `persona`

The public website. Bakes profile content at build time and loads blogs and live status from `custodian` at runtime. Built with `blank`.

The face presented to the world.

_Avoid_: "the frontend", "the site" unqualified, "the portfolio".

### `deed`

The infrastructure-as-code component. Terraform (or OpenTofu — undecided) that provisions everything the other four run on: the EC2 instance, the S3 buckets, the CloudFront distribution, the IAM roles, DNS.

The authoritative declared record of the ground the playground sits on — a deed is what says the land is yours, and like Terraform it is a declaration rather than a set of instructions.

Unlike the other four, `deed` doesn't run or serve anything; it describes. It also reaches beyond the playground: it will own provisioning for other personal projects, which gives it the same property `blank` has — **its interface must not become playground-shaped.**

_Avoid_: "the Terraform repo", "the infra" unqualified, "IaC" unqualified.

## The content buckets

`custodian` owns three kinds of content. The distinction is by **who writes it and how often**, and it drives the storage model, the API shape, and how `persona` fetches each.

### Authored content

Written by hand through `broom`. **Logs** (see below) and their uploaded **media**. Changes often, and is the reason `custodian` exists.

### Derived content

Fetched from third-party APIs and cached by `custodian`. Never written by hand. Currently: Steam currently-playing and GitHub activity. Modelled as **integrations** (see below).

Regenerable, so its durability requirements are weaker than authored content — but it is also the content most likely to be stale or unavailable, so it carries a fetch timestamp.

### Profile content

Authored, but changes roughly yearly: about, experience, skills, resume link. Lives in `custodian` rather than as repo files, deliberately — leaving it in the repo would partly recreate the rebuild-to-edit problem the whole project exists to escape.

`persona` bakes this at build time, so editing it does require a rebuild. That is an accepted trade, not an oversight.

## Curated projects vs activity

**Two distinct concepts**, deliberately not merged:

- A **curated showcase** — a project list you author and order yourself. Editorial. Authored content.
- An **activity feed** — what you are demonstrably working on, observed from GitHub. Not editorial. Derived content.

They have different lifecycles, different trust levels, and render differently. Do not collapse them into one "projects" concept, which is what the deprecated site's single Notion database did.

The curated showcase is not its own first-class type — it is a **profile record** keyed `curated-projects` (see the domain model below). The activity feed is an **integration**.

## The authored-content domain model

Settled by ticket `03`. Every term below is ubiquitous language — use it exactly.

**"Content" means all three buckets** (authored, derived, profile), not just authored.

### `log`

A long-form, blog-like authored writing piece. The one first-class authored-structured type — everything else profile-ish is a keyed record (below).

- **Fields**: title, subtitle, slug, cover image, reading time, tags, created date, updated date, body, state. (No description/excerpt yet — deferred to the blog-list UI design.)
- **State**: `listed` or `unlisted`. Every log is always reachable by its slug link; state only controls whether it appears in the blog index. `unlisted` covers both never-published drafts and taken-down logs — there is no separate hidden, un-shareable state. Publishing = listing; unpublishing (listed → unlisted) is allowed. So a draft is shareable by link before it is ever listed.
- **Slug**: author-chosen. **Frozen while listed, mutable while unlisted** — an unlist re-opens it for editing; you don't care about link stability once it's off the index.
- **Body**: **GFM core** — CommonMark plus tables, strikethrough, task lists, autolinks. **No footnotes** (parser-portability footgun). No custom/novel syntax. The `blank` highlight-yellow emphasis is plain `**strong**` (nothing else is bolded). A **blockquote** (`>`) carries both quotations *and* margin/aside notes — `persona` decides which side to float it; the author gets no left/right control.

### `snippet`

A short-form, note-like writing piece. **Future feature — not v1.** Likely to arrive as a summary *derived from* a log rather than a separately authored type. Recorded so nobody builds it prematurely.

### `media`

A first-class, independent content type — not owned by any log. Full CRUD, list, and search through `custodian`/`broom`.

- **Fields**: url, filename, content-type, size, upload date, caption, attribution.
- **URL** is a generic, public, domain-owned CDN URL (`cdn.mihirsingh.dev/…`) — **not** scoped to a log (no `logs/<slug>/` in the path). The author pastes it into a log body as ordinary markdown, so the body stays portable to any markdown platform.
- Logs reference media only as **URL text** `custodian` does not parse, so there is no referential integrity. Deleting media can orphan a link. Mitigation is an active **`broom` reference-check**: because log bodies are pure-text markdown, `broom` searches all logs for a media URL before deleting it. Orphaning is otherwise the author's responsibility.

### `profile`

A table of keyed records — one **row per key**, each with an `id` and an opaque JSON **`body`** column. Keys: `about` (body is markdown), `experience`, `skills`, `resume-link`, `curated-projects`. `custodian` does **not** validate the body shape; the shape is a convention shared between `persona` and `custodian`. The no-guardrail trade (a typo breaks the live site) is accepted because profile is low-churn and single-author.

### `integration`

The derived/third-party content type — one **record per source** (Steam, GitHub), each with an opaque JSON **`body`** and a fetch timestamp. Mirrors `profile`'s id-plus-body shape. `custodian` stores what it fetched without understanding the payload's internal structure.

## Not yet settled

Recorded so nobody assumes these are decided:

- **Language and framework for every component except `blank`** (TypeScript by necessity) **and `deed`** (HCL, though Terraform vs OpenTofu is open — ticket `18`). Polyglot is *incidental* here: best tool per component, never variety for its own sake.

## Settled since

- **Where everything runs** — AWS, eu-west-1. See [ADR-0001](docs/adr/0001-hosting-and-deployment-posture.md).
- **The authored-content domain model** — ticket `03` (above): logs, media, profile records, integrations, and the ubiquitous language for all three buckets.
