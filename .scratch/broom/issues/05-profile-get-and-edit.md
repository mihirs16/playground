# 05 — Profile: `get` and `edit`

**What to build:** Inspecting and editing profile records. `broom profile get
<key>` fetches a profile record (`about`, `experience`, `skills`, `resume-link`,
`curated-projects`) and shows its raw JSON, so the current value can be inspected.
`broom profile edit <key>` opens the record's raw JSON in `$EDITOR` and
`PUT`-upserts it on save. The body is opaque — broom imposes no schema; the shape
is convention between the author and `persona`. Reuses the `$EDITOR` seam
established for authoring.

**Blocked by:** 02 — reuses the `$EDITOR` seam.

**Status:** ready-for-agent

- [ ] `profile get <key>` fetches a record and shows its raw JSON
- [ ] `profile edit <key>` opens raw JSON in `$EDITOR` and `PUT`-upserts on save
- [ ] broom imposes no schema on the body — it is round-tripped opaquely
- [ ] Round-trip exercised through the fake custodian and the faked `$EDITOR`
