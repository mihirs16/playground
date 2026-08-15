# 04 — Data buckets: media + SQLite-backup S3 with `prevent_destroy` and instance-profile access

**What to build:** The durable homes for the data ADR-0001 says to rent because
"if it's gone, it's gone," carrying the one safety invariant that must survive any
future state re-partitioning. `deed` provisions the media bucket and the SQLite
backup bucket (the Litestream target), each with
`lifecycle { prevent_destroy = true }` so no component destroy can silently take
unrecoverable data regardless of how the state files are later split, and grants
the box's instance profile the read/write access it needs to serve media and back
up SQLite. With this in place custodian's media reserve/confirm flow and its
Litestream backup have real homes.

**Blocked by:** 02.

**Status:** ready-for-agent

- [ ] Media bucket and SQLite-backup bucket provisioned
- [ ] `prevent_destroy = true` present on both data buckets, independent of state layout
- [ ] Instance profile granted the read/write access needed for media and SQLite backup
