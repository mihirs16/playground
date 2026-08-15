# 01 — Walking skeleton: config, generated client, test harness & auth

**What to build:** A booting `broom` binary that stands up the whole shape the
rest of the work hangs off. The noun-grouped command tree (`logs`, `media`,
`profile`, `integration`, plus top-level `login`/`logout`) exists with subcommands
stubbed. The *same* generated OpenAPI Go client custodian publishes is vendored
in and wired as the transport, so broom holds no hand-rolled HTTP. Configuration
resolves from a single XDG config file holding `url` + `token` together, with a
baked-in default URL and `BROOM_URL`/`BROOM_TOKEN` environment overrides that
take precedence over the file. The one test seam is established — an in-process
fake custodian (`net/http/httptest`) that speaks the same OpenAPI contract — and
the baseline error renderer for custodian's RFC 9457 `problem+json` (render the
`detail`, branch on the stable `code`, list field-errors) that also distinguishes
a network / custodian-down failure from a rejected request. The first real
commands land: `broom login` prompts for a token, verifies it with a single
authenticated call, and on success persists `url` + `token` to the config file at
`0600`; `broom logout` removes the stored token. Any command with no valid
credential fails with a clear "not logged in / token rejected" message.

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

- [ ] `go build` produces a single static binary; command tree present with subcommands stubbed
- [ ] Vendored generated OpenAPI Go client wired as the transport
- [ ] Config resolves `url` + `token` from a single XDG file (`0600`); baked-in default URL; `BROOM_URL`/`BROOM_TOKEN` override the file
- [ ] `httptest` fake custodian harness stands up the real command wiring against a fake API boundary
- [ ] `problem+json` renderer shows `detail`, branches on `code`, lists field-errors, and distinguishes custodian-down from custodian-said-no
- [ ] `login` verifies via one authed call and writes `0600` config; `logout` clears the token
- [ ] Missing/rejected credential fails with a clear not-logged-in message, never mistaken for a content bug
