# 09 — Derived-status widgets: Steam & GitHub (runtime seam)

**What to build:** The decorative live-status widgets that make the site feel
alive — Steam currently/recently playing, and recent GitHub activity. They are
plain client-side custom elements (not Astro `client:*` islands), hydrated via
native custom-element upgrade. Each **fetches `custodian` on `connectedCallback`**
(`GET /v1/integrations/{source}`) and **hides itself entirely when there is no
data** (empty payload → widget absent, never a hollow placeholder); a
failed/timed-out fetch also leaves it hidden rather than erroring. `persona` owns
the **staleness policy**: it applies its own "recently active → stale" threshold
to the fetch timestamp `custodian` carries, and the copy says **"recently
active"**, never "live"/"now" — idle and "source briefly unreachable" are
indistinguishable by design, so the honest statement is "nothing new since X".
The status surface is an instance of `<blank-badge>`. The GitHub activity feed
reads as visibly **observed**, distinct from the curated-projects showcase.

**Blocked by:** 02.

**Status:** ready-for-agent

- [ ] Each widget is a plain client-side custom element fetching `custodian` on `connectedCallback`
- [ ] An empty integration payload hides the widget entirely; a failed/timed-out fetch also stays hidden without erroring
- [ ] `persona`-owned staleness threshold applied to the fetch timestamp: fresh shown as current, past-threshold not presented as current
- [ ] Copy says "recently active", never "live"/"now"
- [ ] Status surface rendered as `<blank-badge>`; the observed feed is visibly distinct from the curated showcase
