## Code style

- Functions: 4-20 lines. Split if longer.
- Files: under 500 lines. Split by responsibility.
- One thing per function, one responsibility per module (SRP).
- Names: specific and unique. Avoid `data`, `handler`, `Manager`.
  Prefer names that return <5 grep hits in the codebase.
- Types: explicit. No `any`, no `Dict`, no untyped functions.
- No code duplication. Extract shared logic into a function/module.
- Early returns over nested ifs. Max 2 levels of indentation.
- Exception messages must include the offending value and expected shape.

## Comments

- Keep your own comments. Don't strip them on refactor — they carry intent and provenance.
- Write WHY, not WHAT. Skip `// increment counter` above `i++`.
- Docstrings on public functions: intent + one usage example.
- Reference issue numbers / commit SHAs when a line exists because of a specific bug or upstream constraint.

## Tests

**The goal is confidence that the app produces the results we want — not
coverage of every small piece of code.** A test earns its place by protecting an
outcome someone would notice breaking. Everything below follows from that.

- **Prefer INTEGRATION.** The default test mounts a real page (or hits a real
  handler through the real router) with the I/O faked at the edge, and asserts
  what the user or the caller actually gets. That band catches composition bugs
  — which is where the defects have actually been — and it is where new coverage
  should go first.
- **Unit-test what carries a RULE**, not what carries plumbing. Modifier
  stacking, PV/PM, rounding, limits, optimistic rollback, wire formats: yes.
  Getters, one-line formatters, a `Set` + `sort` the assertion re-implements,
  and anything the typechecker already guarantees: no.
- **E2E is the smallest possible set.** A Playwright test must justify itself
  with a mechanism only a real browser has — animation timeline, real layout and
  overflow, virtualized lists that measure zero in jsdom, a socket across two
  servers. "It's a user journey" is NOT a justification: journeys are cheaper
  and steadier as integration tests. E2E is the most expensive and most fragile
  thing in this repo; spend it deliberately.
- **Push each guarantee to the cheapest layer that can hold it.** A server rule
  belongs in a handler test, not in an assertion that a button is missing — UI
  gating is UX, the security boundary is the server. Don't assert the same rule
  in three layers; pick the one that owns it.
- **Bug fixes get a regression test, and it must be proven RED first.** A test
  that was never seen failing is a guess. When a fix and its test land together,
  the commit says how the test was proven to fail.
- **Delete tests that cost more than they protect**: assertions on class names
  and DOM shape nobody promised, tests that re-derive the expected value by
  running the implementation, tests over code that is dead or gated out of the
  bundle. Green tests over unused code are the worst kind of debt — they charge
  maintenance and protect nothing.
- Data transcribed from the book (catalogs) is validated by SCHEMA at the dump,
  not by an `expect` per field repeating the same number. Pin the *exception*
  (the trap in the table), never the whole table.
- Mock external I/O (API, DB, filesystem) with named fake classes, not inline
  stubs. Tests must be F.I.R.S.T: fast, independent, repeatable,
  self-validating, timely.
- Write the test first when you can state the expected result first; when
  chasing a bug, reproduce it in the browser or a handler first, then encode it.
- **Cutting order: write the replacement, watch it go green, THEN delete.**
  Deleting first opens a blind window.

## Dependencies

- Inject dependencies through constructor/parameter, not global/import.
- Wrap third-party libs behind a thin interface owned by this project.

## Structure

- Follow the framework's convention (Rails, Django, Next.js, etc.).
- Prefer small focused modules over god files.
- Predictable paths: controller/model/view, src/lib/test, etc.

## Formatting

- Use the language default formatter (`cargo fmt`, `gofmt`, `prettier`, `black`, `rubocop -A`). Don't discuss style beyond that.

## Logging

- Structured JSON when logging for debugging / observability.
- Plain text only for user-facing CLI output.

## Reference
- Use [Tormenta 20 book](/t20-book.pdf) as reference for rules

## Package guides
- Frontend has its own adapted rules: [frontend/CLAUDE.md](frontend/CLAUDE.md)
  (FSD layers, thin TanStack routes, styling tokens, vitest/tsgo/eslint).
  When working under `frontend/`, follow it in addition to this file.
- The backend is `engine-go/` (Go): HTTP API on :3001, the rules engine, and
  the same engine compiled to WASM for the browser. One process serves the SPA,
  the API and the socket in production (`STATIC_DIR`) — there is no nginx and no
  compose. The NestJS backend was removed once nothing consumed it.
