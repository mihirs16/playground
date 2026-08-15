# custodian: storage model

Type: grilling
Status: resolved
Blocked by: 03

## Question

Where does authored content physically live, and where does media live?

The main fork:

- **Git-backed markdown** — content is files in a repo that custodian reads and writes. Version history and diffs for free, editable in your own editor as a fallback, no migrations, and content survives custodian dying. Costs: concurrent writes are awkward, querying and pagination mean reading the tree or maintaining an index, and every write is a commit.
- **Database-backed** — SQLite (a single file, trivial to back up, plausible even on a container with a volume) or Postgres (managed, more moving parts). Real queries, real indexes, clean pagination. Costs: migrations, backups you own, and content that only exists inside custodian.

Then, separately:

- ~~**Media** — object storage or local disk~~ **Settled by `01`: S3, eu-west-2.** What remains is the key scheme — content-addressed or path-based? — and whether originals are kept alongside derivatives.
- **Derived-data cache** — do the cached Steam and GitHub payloads live in the same store, or somewhere ephemeral (in-memory, Redis)? They're regenerable, so durability requirements differ from authored content.
- **Backups** — what's the actual recovery story for authored content, given the blogs are the thing you'd most hate to lose?

## Context

Still blocked on `03` — the domain model has to say what an entity *is* before deciding where it sits.

**`01` narrowed this substantially without closing it.** What it locked is the **posture**, not the engine:

- State lives **on the instance's own disk**, with durability rented to **S3** via continuous replication. Not a managed database, not Postgres on the box.
- **Blobs go to S3, records go to a queryable store.** Blog markdown counts as a *record*, because listing S3 gives you keys, not "published posts newest-first" — an S3-only content store means hand-rolling an index object and racing it on every write.
- `01`'s working assumption is **SQLite**, and `07` now inherits "must embed SQLite in-process" as a constraint. This ticket can still overturn the engine, but doing so reopens `07`.

**The git-backed fork is narrowed, not dead** — and it's still worth weighing rather than defaulting to a database out of habit. A pure git-backed store is hard to reconcile with `01`'s replicate-to-S3 durability model, but a **hybrid** fits cleanly: markdown in git as the source of truth, an index in SQLite for querying. That gets version history, diffs, and editor-as-fallback while keeping pagination sane. Its cost is two sources of truth to keep in step.

The concurrency objection to git barely applies to a single writer, which is what made it attractive in the first place.

Note the interaction with `09`: if content is git-backed, the write path is inherently slower and possibly asynchronous, which the API contract has to reflect honestly rather than pretending writes are instant.

## Blocks

`09` API contract

## Answer

The engine question stays closed the way `01`/`07` left it — this ticket settles *how custodian uses* that engine, plus keys, cache placement, and recovery.

**1. Source of truth: SQLite, sole.** The `log` body (GFM-core text) is a column in SQLite. No git backing, no dual source of truth. The git-hybrid fork was weighed and dropped: it adds git *on top of* `07`'s already-embedded `modernc.org/sqlite` rather than instead of it, forces a second replication path alongside `01`'s "replicate one file to S3" durability model, and partly walks back toward the file-based, rebuild-to-edit content that custodian exists to escape. Does not reopen `07`.

**2. Version history: deferred (additive fog).** Edits overwrite in place; a `log` row carries `created_at`/`updated_at`. A `log_revision` table (append a body snapshot per save, current-pointer on the row) is purely additive, so shipping without it costs nothing structurally. Single author + low churn + whole-DB PITR (see 5) make per-post revisions a nice-to-have, not v1. Graduated to fog.

**3. Media key: a single kebab-case string.** Author-provided, or randomly generated (kebab) when omitted. Uniqueness is **enforced by custodian** — a duplicate key returns an error, never a silent overwrite. Filename / content-type / size already live on the `media` record (`03`), so the key stays a bare slug: the public URL is extension-free (`cdn.mihirsingh.dev/<key>`) and content-type is served from record metadata via CDN headers. Content-addressing was rejected — its dedup win is irrelevant for one author, and its immutability fights the plausible "swap the cropped version, keep the URL" operation. Future derivatives slot under the key (`<key>-800w` / `<key>/800w`); not foreclosed. "Originals alongside derivatives?" defers to the media-pipeline fog.

**4. Derived data (`integration`) lives in the same SQLite store.** No Redis, no in-memory tier — either would violate `01`'s "self-manage the compute, rent the durability, keep it one boring thing." The two regenerable JSON rows (Steam, GitHub) ride along in S3 replication for a rounding-error cost, and persona's read path stays uniform across all three buckets. A stale `integration` row restored from backup self-heals on the next poll.

**5. Recovery: point-in-time restore + S3 versioning, 30-day retention.** `01`'s continuous replication is upgraded from a plain mirror to **WAL shipping (Litestream-style)**, giving per-timestamp restore — so "I ran a bad write an hour ago" is recoverable, not just "the instance died." **S3 bucket versioning on both** the DB-replica bucket and the media bucket makes bad object deletes/overwrites undoable (reinforcing that `broom`'s pre-delete reference scan is a courtesy, not the safety net). Window: **30-day PITR + 30-day non-current-version lifecycle expiry** on both buckets — directly answering "the blogs are what you'd most hate to lose."

**⚠️ Open caveat carried on this decision:** verify the final S3 cost (Standard storage + versioning non-current copies + WAL segment history at 30-day retention) fits `01`'s ~£10/mo envelope. Near-certainly pennies at blog scale, but confirm against real numbers when the buckets are configured — flagged in fog, not blocking.

**Ripples:** `09` (API contract) was blocked by `03` and `08` — both now resolved, so it joins the frontier. The write path is fully synchronous (SQLite in-process), so `09` need not model async/slow writes. The media-pipeline fog no longer waits on `08`; it now waits on `blank`/persona rendering needs.

---

**Later refinement (from [`12`](12-derived-data-freshness-and-caching.md)):** point 4's `integration` storage shape is revised from a **single upserted row per source** to an **append-on-change timeseries** (one row per distinct polled state) so future timelines/sparklines are buildable. Each poll compares against the latest row: changed → insert a timestamped row; identical/idle → insert nothing. The store is still the same SQLite DB riding S3 replication; data is kept **indefinitely** (append-on-change keeps it tiny) while the recovery window stays this ticket's **30 days**. Because history is append-only and never edited in place, the "bad write aged out of the 30-day PITR window" risk is a near-non-issue; adds only a rounding-error to the open S3-cost caveat above.
