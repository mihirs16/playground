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

**Status:** ready-for-agent

- [ ] `integration refresh [name]` forces custodian's manual `/refresh` and prints the fresh record
- [ ] A credential written through broom is an authed admin API call; broom never stores it locally
- [ ] Both exercised through the fake custodian, asserting the requests issued
