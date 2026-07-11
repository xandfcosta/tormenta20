# Backend guide — DDD + TDD

Adapts the root [CLAUDE.md](../CLAUDE.md) to this package. Root rules
apply; the notes below override or extend them. This service is built
**domain-first** and **test-first** — those two disciplines drive every
other decision here.

Stack: NestJS 11, Prisma 7 (better-sqlite3 / SQLite), JWT (passport-jwt,
cookie), Socket.IO gateway, class-validator + zod, jest / ts-jest.

## Domain-Driven Design

- **Bounded contexts = Nest modules.** `auth`, `users`, `campaigns`,
  `campaign-members`, `sessions`, `characters`, `realtime`. A module owns
  its domain; cross-context calls go through another module's **service**,
  never its Prisma rows.
- **Layering per context:** controller/gateway (transport) → service
  (application + domain logic) → Prisma (persistence). Business rules live
  in **services**. Controllers only parse/validate/authorize-route and
  delegate; gateways do the same for WS.
- **Ubiquitous language.** Name types and methods after the tabletop
  domain and the Tormenta 20 book: `Campaign`, `Session`, `CampaignMember`,
  `Character`, `ActiveEffect`, `initiative`, `PV`/`PM`, `descanso`,
  `catalisador`. A rule from the book cites its page in a comment.
- **Aggregates.** `Character` is an aggregate root (items, spells, effects,
  expertises, classes). All mutations funnel through `CharactersService` so
  invariants hold — never write child rows from another context.
- **Invariants belong in the domain**, enforced in services and covered by
  tests: e.g. class levels sum ≤ 20, equip caps (4 vested / 2 hands), one
  player character per user per campaign, PV/PM clamped to max.
- **Authorization is a domain rule**, not middleware glue. Encapsulate it:
  `resolveAccess` (GM = owner, player = owns a member character),
  owner-or-campaign-GM `findOne`, WS `assertGm` / `assertVitalsEditable`.
  Reads are member-aware; writes stay GM/owner-scoped.
- **Guard the persistence boundary.** Don't leak Prisma types across a
  context's public surface; return domain shapes. Detect Prisma errors
  structurally (`isPrismaUniqueViolation`, code `P2002`), not by importing
  runtime types.
- **Exceptions carry the offending value + expected shape** (root rule) and
  use Nest's typed exceptions (`ForbiddenException`, `ConflictException`
  with `fieldErrors`).

## Test-Driven Development

- **Red → green → refactor.** Write the failing test first, make it pass,
  then clean up. Every new function gets a test; **every bug fix starts
  with a regression test** that reproduces it.
- Run: `pnpm --filter backend test` (jest). Root `pnpm test` fans out.
- **Named fake classes for all I/O** — `FakePrisma`, fake `CampaignsService`,
  fake socket — injected via the constructor. No inline stubs, no real DB.
  Constructor DI exists precisely to make this substitution clean.
- Tests are **F.I.R.S.T** and colocated as `*.spec.ts` next to the unit.
- **Cover the domain rules exhaustively:** every authorization branch
  (owner / GM / player / stranger), every invariant, both sides of each
  guard. A rule without a test is not done.
- Gateway handlers: drive with a fake socket (`socket.data.user`/`role`) +
  fake services; assert both the return value and the room broadcast.
- When a spec asserts a rule from the book, comment the page/rule so the
  expected numbers are traceable.

## Nest conventions

- Dependencies via **constructor injection** only (never global/import).
  This is both a root rule and the seam TDD relies on.
- One provider = one responsibility; keep services focused, split god files
  (root: <500 lines).
- DTOs validate input at the edge (class-validator / zod); services trust
  validated input but re-check **domain** invariants.
- Structured logging via Nest `Logger` (JSON-ish, contextual) — never
  `console.log`. Fire-and-forget side effects (persistence write-through)
  log failures with the offending id, don't crash the request.

## Typecheck + lint

- Typecheck: `pnpm --filter backend typecheck` (tsgo, `tsconfig.build.json`
  — excludes `*.spec.ts`; ts-jest type-checks specs at test time).
- Lint/format: `pnpm exec biome lint --write .` from root (backend **is**
  biome-scoped, unlike frontend). CI fails on any lint-produced diff.
- Emit (nest build) + tests (jest) still run on classic `tsc`/ts-jest;
  tsgo is typecheck-only until its programmatic API lands (TS 7.1).

## Reference

- Root [CLAUDE.md](../CLAUDE.md) and the [Tormenta 20 book](/t20-book.pdf)
  (rules source of truth — cite pages for domain invariants).
