# custodian: auth model

Type: grilling
Status: open
Blocked by: 09

## Question

How does the cli authenticate to custodian, and what exactly is public?

- **Write auth** — a long-lived personal token, an OAuth device flow, mTLS, or something else? There is exactly one human writer, so the simplest thing that is genuinely safe probably wins.
- **Token storage on the client** — where does the cli keep credentials? OS keychain, a config file with restricted permissions, or an environment variable? What does `login` / `logout` look like, if those exist at all?
- **Is the read path fully public?** Blogs and profile presumably yes. Drafts presumably not — which means at least two trust levels, and a preview mechanism if persona is to render unpublished work.
- **Rotation and revocation** — what happens when a token leaks, and how would you know?
- **Rate limiting and abuse** — the read API is public and on the critical path for page rendering. What stops it being hammered?
- **Third-party secrets** — where do the Steam and GitHub API keys live, and who can reach them? These are custodian-side only and must never be exposed to persona's client bundle.

## Context

Blocked on `09` because auth attaches to a contract — whether there's one API or two changes the answer substantially.

**The cautionary fact this ticket exists to avoid repeating**: the deprecated site set `NEXT_PUBLIC_NOTION_KEY`, and the `NEXT_PUBLIC_` prefix inlines a value into the client bundle. The Notion integration token was therefore shipped to every visitor's browser. The same failure mode is available here — persona fetches from custodian at runtime, in the browser, so anything persona needs to send is public by construction. That constraint should shape the design rather than be patched later.

Corollary worth stating explicitly in the spec: because persona's Steam and GitHub reads happen client-side, custodian must proxy them. The third-party keys stay server-side, always.
