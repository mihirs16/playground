# 04 — Media: `add`, `ls`, `rm`

**What to build:** The media workflow, a deliberately separate gesture run in a
second terminal while a post sits open in the editor. `broom media add <file>
[--key k]` runs custodian's reserve → upload → confirm flow over a presigned S3
`PUT`: custodian reserves a `pending` record and hands back a presigned URL, broom
`PUT`s the bytes straight to S3, then confirms so custodian `HEAD`s S3 and flips
the record to `available`. broom holds no AWS credential — it talks only to
custodian and to the presigned URL. On success it prints **and** clipboard-copies
legible markdown `![](https://cdn.mihirsingh.dev/media/<key>)`, so the author
pastes a readable reference rather than an opaque id — the whole ergonomic point.
The author supplies a kebab-case `--key` for a meaningful reference, or omits it
and custodian mints a random kebab key. A duplicate key is reported clearly from
custodian's `media_key_taken`, never a silent overwrite. `broom media ls` lists
and searches existing media so an asset can be reused instead of re-uploaded.
`broom media rm <key>` first scans the author's post bodies for references to that
key and warns before deleting, so a live post is not left pointing at an orphaned
image (custodian does not parse bodies for URLs; this scan is broom's courtesy).
This slice establishes the clipboard fake and the presigned-`PUT` fake.

**Blocked by:** 02 — the `media rm` reference scan reads post bodies.

**Status:** ready-for-agent

- [ ] `media add` runs reserve → upload → confirm; bytes go to the presigned URL, broom holds no AWS credential
- [ ] On success, prints and clipboard-copies the exact `![](https://cdn.mihirsingh.dev/media/<key>)` string
- [ ] Author `--key` used verbatim; omitted `--key` accepts a custodian-minted random kebab key
- [ ] Duplicate key surfaces `media_key_taken` legibly, no silent overwrite
- [ ] `media ls` lists and searches media
- [ ] `media rm <key>` scans post bodies for references and warns before deleting
- [ ] Clipboard and presigned-`PUT` faked in tests; reserve → upload → confirm driven deterministically through the fake custodian
