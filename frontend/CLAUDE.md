# Frontend guide

Adapts the root [CLAUDE.md](../CLAUDE.md) to this package. Root rules
apply; the notes below override or extend them for the frontend.

Stack: React 19, Vite (rolldown), TanStack Router (file-based) + Query,
Zustand, Shadcn/Radix, Tailwind v4, socket.io-client, sonner.

## Code style

- Root code-style rules apply (4–20 line functions, <500 line files, SRP,
  explicit types, no `any`, early returns, max 2 indent levels).
- Function components + hooks only. No class components.
- Follow the rules of hooks: no conditional hooks; stable deps arrays.
- Derive, don't duplicate: compute from query data in render, don't mirror
  server state into `useState`. Local UI state only in `useState`/Zustand.
- Props are the injection seam — pass data/handlers in, don't reach into
  singletons from a component.

## Architecture — Feature-Sliced Design + routes

Layers, dependencies point **downward only** (`routes → pages → features →
entities → shared`); never import sideways or upward except via a layer's
public surface.

- **routes/** — TanStack file routes, kept **thin**: `createFileRoute` +
  `beforeLoad`/`loader`/`validateSearch` only, delegating render to a
  `pages/*` component. No screen JSX or business logic here.
- **pages/** — screen composition. A component that leaves its Route's
  module uses `getRouteApi('<path>')`, not `Route.useX`.
- **features/** — user-facing use-cases (session-tracker, character-sheet,
  campaign-manage, gm-tools, …).
- **entities/** — light: per-domain query hooks (`entities/*/queries.ts`) +
  derived logic. Domain **types + the `api` client stay in `shared/api`**
  (documented deviation).
- **shared/** — `ui` (shadcn kit), `layout`, `api`, `realtime`, `lib`,
  `stores`. `app/` holds `main.tsx` bootstrap.

**Nested-route gotcha:** a `$id` route that has children (e.g.
`$id.sessions.$sid`, `$id.sheet`) must be a **layout** (`component: Outlet`)
with the detail screen in `$id.index.tsx` — otherwise the detail page
swallows the outlet and children never mount. See
[[reference_tanstack_nested_routes]] / root memory.

## Data + third-party boundaries

- Server state via TanStack Query only. Query hooks live in `entities/*`;
  the HTTP client is `shared/api/api.ts` — components never call `fetch`.
- Wrap third-party libs behind owned modules: socket.io → `shared/realtime`
  (`useSessionSocket`), toasts → `shared/ui/sonner`. Don't import
  `socket.io-client` / `sonner` directly in features/pages.
- Auth + cross-cutting UI state in Zustand stores under `shared/stores`.

## Styling

- Tailwind v4 + the Controlled Decay tokens in `src/index.css`
  (`--primary`, `--card`, `--hp-full`, `--font-display` = Cinzel, …). Use
  tokens (`bg-card`, `text-muted-foreground`, `[color:var(--primary)]`),
  not raw hex.
- Every screen must work at phone width: responsive grids (`sm:`/`lg:`),
  `flex-wrap` button rows, horizontal scroll for overflowing tab bars.

## Tests

- Run: `pnpm --filter frontend test` (vitest). Root `pnpm test` fans out.
- Every new function/hook gets a test; bug fixes get a regression test.
- Mock I/O with **named fake classes/objects** (fake socket, fake fetch),
  not inline stubs. React Testing Library for components.
- The security boundary lives on the server — don't rely on UI gating for
  correctness; still gate UI by role for UX.

## Typecheck + lint + format

- Typecheck: `pnpm --filter frontend typecheck` (tsgo). `noUnusedLocals` +
  `noUnusedParameters` are **on** — prune unused imports.
- Lint: `pnpm --filter frontend lint` (eslint) for this package's own
  script. **But CI also runs root `biome lint --write .`, and biome's
  `includes` cover `**/*.tsx` — so biome lints frontend too** (only its
  formatter is off). CI fails on any biome error or `--write` diff. Biome
  can't tell a custom component (`NumberInput`, `Combobox`) is a form
  control, so its recommended a11y rule `noLabelWithoutControl` errors on a
  `<label>` that wraps one — associate via `htmlFor`+`id`, not nesting.
  There is no prettier binary — match surrounding formatting.
- `routeTree.gen.ts` is generated + gitignored; CI regenerates on build.

## Reference

- Root [CLAUDE.md](../CLAUDE.md) and the [Tormenta 20 book](/t20-book.pdf).
- FSD: https://feature-sliced.design/
