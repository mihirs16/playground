# 10 — Wire the real S3 ObjectStore (AWS SDK v2 + instance-profile chain)

**What to build:** the media flow that is green against the fake becomes real in
production. Replace the stubbed `errNotImplemented` S3 edge with a concrete
`ObjectStore` backed by AWS SDK for Go v2, authenticated through the IMDS
instance-profile credential chain — no long-lived AWS credentials on disk
(ADR-0001). `PresignPut` returns a working presigned S3 `PUT` for the media key
and content-type with the configured expiry; `HeadObject` reports real object
presence so `confirm` only flips a record to `available` when bytes actually
landed; `HeadBucket` reaches the real bucket so it can serve as an input to the
health gauge. The injected interface and its fake are unchanged, so every
existing media and health test keeps passing untouched — this ticket only
supplies the real implementation the fake was always standing in for. Verified
against a real bucket by an operator smoke (reserve → upload to the presigned
URL → confirm → `available`).

**Blocked by:** 04 (media reserve/confirm flow against the fake — done).

**Status:** ready-for-agent

- [ ] `ObjectStore` implemented with AWS SDK v2 using the IMDS instance-profile chain; no static AWS creds
- [ ] `PresignPut` yields a working presigned S3 `PUT` honouring key, content-type, and expiry
- [ ] `HeadObject` reflects real object presence; `confirm` flips to `available` only when bytes exist
- [ ] `HeadBucket` reaches the real bucket and feeds the health gauge
- [ ] Existing media + health tests pass unchanged against the fake (interface untouched)
- [ ] Operator smoke against a real bucket: reserve → upload via presigned URL → confirm → `available`
