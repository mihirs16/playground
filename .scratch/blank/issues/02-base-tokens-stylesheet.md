# 02 — Base / tokens stylesheet

**What to build:** The single base/tokens stylesheet a consumer imports once to get the
whole aesthetic without writing CSS. It exposes the design tokens as documented CSS
custom properties — the monotone ink ramp, the `4·8·16·24·40·64·96px` spacing scale,
the `20/16/15px` (h1/h3/body) type scale, and an independent dark-theme token layer
(ink → `#e9e9e9`, paper → `#0b0b0b`) — and defaults them on `:root` so they cross the
shadow boundary by inheritance. Importing it restyles global `<a>` as ink underlines
(no yellow) and global headings as centered, standing alone, lowercased, one weight
(400), separated by whitespace only, with colour balanced against size (h1 `#5F5F5F`,
h3 `#5C5C5C`, body full ink). `#FBFF20` is exposed strictly as a background-highlight
token — never text or underline. Long-form body prose is Roboto Serif, justified,
block-centered at ~64ch, and keeps its authored casing while all chrome and headings
are lowercased. A single `600px` breakpoint. This ticket establishes the
custom-property theming channel the components inherit through. No `h3:after` hairline,
no Poppins.

**Blocked by:** 01.

**Status:** ready-for-agent

- [ ] Ink ramp, spacing scale, type scale, and dark-theme layer exposed as documented CSS custom properties, defaulted on `:root`
- [ ] Global `<a>` renders as an ink underline; headings render centered, lowercased, one weight, whitespace hierarchy
- [ ] Colour-by-size: h1 computes to `#5F5F5F`, h3 to `#5C5C5C`, body to full ink
- [ ] `#FBFF20` available only as a background-highlight token
- [ ] Body prose is Roboto Serif, justified, ~64ch, keeps authored casing; chrome/headings lowercased
- [ ] Single `600px` breakpoint; verified via computed styles on plain elements; demo page
