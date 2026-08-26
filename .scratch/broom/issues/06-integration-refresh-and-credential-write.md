# 06 — Integrations: `refresh` and credential write

**What to build:** The terminal gestures for third-party integrations.
`broom integration refresh [name]` forces custodian to poll a source (`steam`,
`github`) immediately and prints the fresh record, so a newly rotated Steam key or
a fixed GitHub PAT can be verified without waiting for the next automatic tick.
Writing a third-party integration credential through broom is an authenticated
admin API call, so key rotation is a terminal gesture that takes effect on the
next poll with no custodian restart — the credential lives in custodian, never
stored by broom.

**Blocked by:** 01.

**Status:** done

- [x] `integration refresh [name]` forces custodian's manual `/refresh` and prints the fresh record
- [ ] ~~A credential written through broom is an authed admin API call; broom never stores it locally~~
- [x] Both exercised through the fake custodian, asserting the requests issued

**Note (2026-08-26):** Credential write dropped. Custodian resolves each source's
credential from its deploy-time environment (see `poller.go`: keys are read
env-at-startup) and exposes no credential-write endpoint — the only integration
admin path is `.../refresh`. So key rotation is a custodian deploy concern, not a
broom terminal gesture. `integration refresh` is implemented: named to poll one
source, or bare to fan out over all known sources (`steam`, `github`), with a
failed poll surfaced as an error.

Also added `integration get [name]` over custodian's public read
(`GET /v1/integrations/{source}`): prints the last stored record without forcing
a poll, including the empty-but-present shape for a source never polled.
