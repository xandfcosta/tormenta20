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

## Interaction &amp; navigation (one philosophy, all scenes)

- **One design language across every scene** (grimório reskin, ALE-55): same
  tokens, framing, focus/selection treatment, and SFX vocabulary. A new screen
  adopts the existing scene grammar — it doesn't invent its own look.
- **One keyboard-navigation grammar, everywhere.** Scenes are **regions**
  (rail / header / content …) navigated with a shared primitive
  (`shared/lib/spatial-nav` geometry + `shared/ui/scene-nav` hook): arrows move
  the focus cursor **by layout** (2-D in a grid, 1-D in a list) and cross to a
  neighbouring region at the edge; **PageUp/PageDown** switch section (bumpers);
  **Enter** activates; **Esc** goes up one level (item → rail → leave scene). Do
  not hand-roll per-scene `keydown` handlers — declare regions via
  `data-nav-region` / `data-nav-layout` and let the hook drive.
- **Keyboard never removes the mouse.** Arrow/gamepad navigation is **additive
  progressive enhancement** over real, clickable/tappable controls — every item
  stays a native `a`/`button` (or has a click handler) and keeps its `hover`
  state. Pointer input must work fully on its own; never gate a click behind a
  keyboard step, and never trap focus.
- Keyboard nav is enabled only on laptop/desktop (`≥xl` + `pointer: fine`); it
  degrades cleanly to tap below that. Shortcut hints are `xl:`-gated too.

## Tests

- Run: `pnpm --filter frontend test` (vitest). Root `pnpm test` fans out.
- Every new function/hook gets a test; bug fixes get a regression test.
- Mock I/O with **named fake classes/objects** (fake socket, fake fetch),
  not inline stubs. React Testing Library for components.
- The security boundary lives on the server — don't rely on UI gating for
  correctness; still gate UI by role for UX.
- **Responsive validation is mandatory for any screen/layout change.** Before
  calling a UI task done, validate it at all six form factors — a scene must
  fill the space it's given (no card floating in wasted margin) *and* stay
  usable when cramped:
  - **Desktop** — 1920×1080
  - **Laptop** — 1440×900
  - **Tablet landscape** — 1024×768 · **Tablet portrait** — 768×1024
  - **Mobile landscape** — 844×390 · **Mobile portrait** — 390×844

  Grimório scenes fill their `SceneShell` width (no narrow `max-w` cap that
  leaves empty margin on desktop); reserve max-width only for genuine reading
  measure (prose), not for whole scenes.

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
