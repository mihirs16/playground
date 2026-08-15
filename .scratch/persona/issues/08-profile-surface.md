# 08 — Profile surface

**What to build:** A reader learns who the author is. Profile content — about,
experience, skills, resume link, and curated projects — is baked at build time
from `custodian`'s public read surface (`GET /v1/profile/{key}`), so all content
stays in one owned, API-backed store. The `about` markdown renders as prose so it
reads like the rest of the site. The curated project showcase is presented as a
deliberately authored, **ordered editorial** list — visibly distinct from the
observed GitHub activity feed, so a reader is never misled into thinking an
observed feed was hand-curated. The surface is assembled from the agreed page
furniture (hero, about, experience card, curated-project card, skills, footer),
and `persona` owns its accessibility floor: heading order, alt text, and the
actual `aria-label` values it passes to `blank`'s icon buttons.

**Blocked by:** 02.

**Status:** ready-for-agent

- [ ] Profile (about, experience, skills, resume link, curated projects) is baked at build from `custodian`'s public API
- [ ] `about` markdown renders as prose
- [ ] Curated projects render as a deliberately ordered, editorial list, visibly distinct from an observed feed
- [ ] Assembled from the agreed page furniture
- [ ] Heading order, alt text, and `aria-label` values passed to `blank`'s icon buttons are correct (a11y floor)
