# 04 — Media reserve → confirm flow (presigned S3, custodian off the byte path)

**What to build:** broom can upload media to S3 without holding any AWS
credentials, and custodian never touches the bytes. `POST /admin/v1/media`
reserves a `pending` media record with a single kebab-case key — author-provided
or, when omitted, a random kebab-case key custodian generates — enforcing key
uniqueness at reserve time (duplicate → `media_key_taken`, never a silent
overwrite) and returning `{ upload_url, url, expires_at }`, where `upload_url`
is a presigned S3 `PUT` and `url` is the extension-free public CDN URL.
`POST /admin/v1/media/{key}/confirm` makes custodian `HEAD` S3 and only then flip
the record to `available`, upholding the invariant that every `available` record
has real bytes behind it. `GET /admin/v1/media` lists and searches media so an
existing asset can be found and reused; `DELETE /admin/v1/media/{key}` removes a
record (any pre-delete reference scan is broom's courtesy — custodian does not
parse bodies for URLs). The S3 edge is exercised through its injected fake.

**Blocked by:** 03.

**Status:** done

- [x] `POST /admin/v1/media` reserves a `pending` record and returns `{ upload_url, url, expires_at }`
- [x] Key uniqueness enforced at reserve; duplicate → `media_key_taken`
- [x] Omitted key → custodian generates a random kebab-case key
- [x] `confirm` does an S3 `HEAD` before flipping to `available`; missing bytes → not flipped
- [x] `GET /admin/v1/media` lists and searches; `DELETE` removes a record
- [x] Public `url` is extension-free; content-type carried in record metadata
- [x] Flow exercised end-to-end against the fake S3 (reserve → simulated upload → confirm → HEAD → available)
