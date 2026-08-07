# Coding Conventions

How code gets written in this repo. Language-agnostic principles — formatting, naming
and import order belong to each component's own linter, not here. These are the rules
that survive a stack change.

Two failure modes drive this doc: **comments that lie or clutter**, and **code that is
clever instead of clear**. Everything below is one of those two.

## Comments

The code is a description of the design **as it stands now**. It is not a changelog, a
debate transcript, or a scratchpad.

**Readable code dispels most comments before they're written.** A comment is not the
first tool for making a line understandable — a clearer name, a smaller function, an
early return usually is. Reach for those first; if they make the code obvious, the
comment was never needed. So most of the time the right response to "this needs a
comment" is to fix the code until it doesn't. The comments that remain are the ones no
amount of clarity could have carried — a *why*, a constraint, a warning. That is a
direct consequence of the section below: the clearer the code, the fewer comments it
can possibly need.

- **Comment the non-obvious, never the obvious.** If the comment restates what the line
  plainly does, delete it. A comment earns its place only by saying something the code
  cannot: a *why*, a constraint, a non-local consequence, a warning.
- **No history in the code.** Never write what the code *used to* do, what approach was
  *rejected*, what was *tried and dropped*, or what got *simplified away*. That belongs in
  the commit message, the PR description, or an ADR — where it is dated and attributed.
  A `// we considered X but…` comment is noise the moment it's written.
- **No decision-diary comments.** If a choice was hard enough to explain, it is an ADR
  (`docs/adr/`), and the comment — if any — is a one-line pointer to it, not the argument
  itself.
- **Prefer commenting the interface over the implementation.** Say what a function
  promises and requires; let the body speak for how.
- **A comment that can go stale will.** If the same fact lives in the code and a comment,
  keep it in the code.

If you find yourself writing a paragraph, ask whether it's really an ADR or a
`CONTEXT.md` entry trying to escape.

## Clarity over cleverness

> "Debugging is twice as hard as writing the code in the first place. Therefore, if you
> write the code as cleverly as possible, you are, by definition, not smart enough to
> debug it." — Kernighan

- **Code should mimic the logic it implements.** The shape of the code should track the
  shape of the problem — the domain terms from `CONTEXT.md`, the steps a reader would
  name if they described the task aloud. When the structure diverges from the mental
  model, that's a smell, not an optimisation.
- **Boring beats terse.** Do not collapse three legible steps into one dense expression to
  save lines. Named intermediate values, an early return, an ordinary loop — these are
  features. Clever one-liners are a cost paid by every future reader.
- **Optimise only what's measured.** Don't trade readability for a performance win you
  haven't proven you need. Correct-and-clear first; fast-and-obscure only with a reason on
  record.
- **Don't hide control flow.** Avoid indirection, metaprogramming, or abstraction whose
  only payoff is fewer characters. An abstraction has to earn its keep by removing real
  duplication or naming a real concept — not by looking sophisticated.
- **Write it the way the surrounding code is written.** Match the idiom, density, and
  naming already in the file. Consistency reads faster than personal style.

## The test

Before committing a line, one question: **would the next reader understand this faster
from the code than from an explanation?** If the answer needs a comment to become yes,
first try to make the code the explanation.

---

*For humans, not a task: the principles above are distilled from John Ousterhout,
*A Philosophy of Software Design* (comments and complexity); Kernighan & Plauger,
*The Elements of Programming Style*; and the Go proverbs ("Clear is better than clever").
These are named for provenance only — there is nothing here to go and read or fetch;
this doc is self-contained.*
