# 11 — Wire the real Steam/GitHub SourceClient (HTTP adapters + shaping)

**What to build:** the poller that is green against the fake fetches real live
status in production. Replace the stubbed `errNotImplemented` `SourceClient` with
concrete, hand-written **code-defined adapters** for Steam and GitHub (`24`):
each owns its endpoint, its per-call credential (read from the process
environment at startup, the same secret class as the admin token hash), and its
response-shaping into persona's predictable per-widget shape — Steam's nested
2-week aggregate and GitHub's filtered feed both collapse to the shape custodian
serves. GitHub is polled with `If-None-Match` and its ETag round-trips so a `304`
is a real "no change". A network failure is returned as an error, never an
empty result, so an unreachable source keeps last-known-good and never flips the
health gauge. The injected interface and its fake are unchanged, so the poller's
changed/unchanged/unreachable/idle/startup tests keep passing untouched — this
ticket supplies the real implementation the fake stood in for. Verified by an
operator refresh (`POST /admin/v1/integrations/{source}/refresh`) returning a
fresh record for each source against live credentials.

**Blocked by:** 06 (integration poller & timeseries read against the fake).

**Status:** in-review

- [x] Concrete Steam and GitHub adapters implement `SourceClient`; credentials read from env at startup
- [x] Each adapter shapes its raw response into persona's per-widget shape (Steam 2-week aggregate, GitHub filtered feed)
- [x] GitHub polled with `If-None-Match`; real ETag round-trips and a `304` is treated as no change
- [x] A network failure is an error (not an empty result); unreachable source never flips the health gauge
- [x] Existing poller tests pass unchanged against the fake (interface untouched)
- [ ] Operator smoke: `POST .../refresh` returns a fresh record for both Steam and GitHub against live credentials (requires live credentials — operator verification)
