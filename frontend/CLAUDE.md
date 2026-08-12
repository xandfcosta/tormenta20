# Frontend guide

Adapts the root [CLAUDE.md](../CLAUDE.md) to this package. Root rules
apply; the notes below override or extend them for the frontend.

Stack: **SolidJS**, Vite (rolldown), TanStack Router (file-based) + Query,
Kobalte (headless UI), Tailwind v4, socket.io-client, solid-sonner.

Runs on **:5173**; the Go API (`engine-go`, :3001) is the backend.

## Code style

- Root code-style rules apply (4–20 line functions, <500 line files, SRP,
  explicit types, no `any`, early returns, max 2 indent levels).
- **The component body runs ONCE.** There is no re-render, so there are no
  hook rules, no dependency arrays and no `memo`. What re-runs is the
  reactive expression, not the function around it.
- **Never destructure props** — it freezes reactivity at the moment of the
  call. Read `props.x` at the point of use, or split with `splitProps`.
- Reactive reads belong *inside* JSX or a derived accessor
  (`const total = () => a() + b()`), never captured in a local const at
  setup time.
- `createComputed` — not `createEffect` — when derived state must settle in
  the SAME update; an effect runs after the body and one render too late.
- Props are the injection seam — pass data/handlers in, don't reach into
  singletons from a component. A component that performs an action takes it
  as a prop (`onConfirm`), so the page owns the api client.
- `create*` factories: one WITHOUT state between calls may be called per
  event; one that HOLDS state (a timer, a rollback snapshot, a draft) must
  be born once in the component body.

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
  character-build, gm-tools, …).
- **entities/** — light: per-domain query options (`entities/*/queries.ts`) +
  derived logic. Domain **types + the `api` client stay in `shared/api`**
  (documented deviation).
- **shared/** — `ui` (the Kobalte-based kit), `layout`, `api`, `realtime`,
  `lib`, `stores`. `app/` holds `main.tsx` bootstrap.

**Nested-route gotcha:** a `$id` route that has children (e.g.
`$id.sessions.$sid`, `/gm/$tool`, `/characters/new/$step`) must be a
**layout** (rendering an `Outlet`) with the detail screen in `$id.index.tsx`
— otherwise the detail page swallows the outlet and children never mount.
See [[reference_tanstack_nested_routes]] / root memory.

**A stepped scene reads its step from the URL**, never from a signal: deep
links, the browser Back button and the progress rail cannot then disagree.

## Data + third-party boundaries

- Server state via TanStack Query only. Query options live in `entities/*`;
  the HTTP client is `shared/api/api.ts` — components never call `fetch`.
- Solid Query has **no `useQueries`**: fan out with
  `queryClient.ensureQueryData` inside one query and derive from the cache.
- Wrap third-party libs behind owned modules: socket.io → `shared/realtime`,
  toasts → `shared/ui/sonner`, fuzzy search → `shared/lib/fuzzy-filter`.
  Don't import `socket.io-client` / `solid-sonner` directly in features.
- **Catalogs are fetched, never bundled.** The `__root` primes them
  (`ensureCatalogs` + `ensureEngineCatalogs`) and the caches are read
  INSIDE components — a module-level const would freeze an empty list.
  `public/engine/*.wasm` is a build artifact: run
  `scripts/build-engine-wasm.sh`.
- Optimistic mutations live OUTSIDE the component (`xActions(queryClient,
  id)` + pure transformations). See `expertise-mutations.ts`.

## Rendering — the traps that cost real time

- **`For` reconciles by REFERENCE.** Any DERIVED array recreates its rows on
  every change, and the field being typed into loses focus after one digit.
  Rule of thumb: `For` for a list of stable identities, **`Index` for a list
  of values that change** (the child receives an accessor). Worth a
  `toHaveFocus()` test — a value assertion will not catch it.
- **`Show` without `keyed` only watches truthiness**: swapping one non-null
  value for another REUSES the node, so a mount-triggered CSS animation never
  restarts and an `animationend` machine hangs.
- With `Show when={x}{(v) => …}` the `v` is an **accessor**; without `keyed`
  it is not the value.
- A registry of steps/blocks holds the **component**, never `render(value)` —
  a function called with a value captures that value and never sees the next
  edit. Render through `<Dynamic component={} />`. Make the registry a
  **total `Record`**, not `Partial`: a new entry then fails to compile
  instead of showing an empty stage.
- **Virtualized lists** (`@tanstack/solid-virtual`): Solid assigns `ref`
  BEFORE attributes, so measuring in the same pass reads an empty
  `data-index` and the core falls to index −1 — measure in a
  `queueMicrotask`. Rows reconcile by index, so `Show` without `keyed` keeps
  whichever item painted first. jsdom sees NEITHER bug (every element
  measures 0 and no row renders), so unit tests pass green over a broken
  screen — cover these in e2e.
- **`{label} {value}` in an `sr-only` line renders TWO text nodes** and the
  announced string comes out split. Use one interpolation.

## Kobalte (headless UI)

- No `asChild`. A link that looks like a button is `<A class={buttonVariants()}>`.
- Kobalte ships **English labels** (`Dialog.CloseButton` → `aria-label="Dismiss"`)
  that OVERRIDE the visible text and any inner `sr-only`. The app is pt-BR:
  pass an explicit `aria-label`, even when the word is written inside.
- `Select` composes the trigger's accessible name as `"<label> <value>"`.
- Data attributes: `data-[expanded]` / `data-[closed]` / `data-[selected]`
  (Radix's `data-state` vocabulary does not exist here).
- No `TooltipProvider`; delay is `openDelay` per tooltip.
- `Combobox` filters nothing by itself, defaults to `triggerMode: 'input'`
  (clicking does not open), needs `allowsEmptyCollection`, and requires
  `translations` in pt-BR. Kit: `shared/ui/picker-combobox` (it ACTS and
  clears itself); for a field holding a value use `Select`.
- A toast fired from inside a modal is NOT announced — the modal marks its
  siblings `aria-hidden` and the sonner region is a sibling. Errors from an
  action taken inside a dialog go INLINE (`DialogInlineError`).
- **`modal={false}` still dismisses on outside interaction.** A panel meant
  to share the screen must also prevent `onInteractOutside` — see
  `shared/ui/side-panel`.

## Styling

- Tailwind v4 + the Controlled Decay / grimório tokens in `src/index.css`
  (`--primary`, `--card`, `--hp-full`, `--grimorio-gold`, Cinzel…). Use
  tokens (`bg-card`, `text-muted-foreground`, `[color:var(--primary)]`),
  not raw hex.
- `--hp-hurt` is amber at hue 70, a hair from the gold at 85: an over-budget
  or overweight bar painted with it reads as "full". Use `--hp-critical`.
- Media queries switch on **WIDTH only**. On a phone the virtual keyboard
  changes viewport HEIGHT, and a height-driven switch rebuilds the component
  mid-typing and loses what was being searched.
- One DOM tree per component — a rail and its phone bar are the same list
  switching by class, not two `Show` branches that drift apart.
- **A new file's arbitrary values are missing from the CSS until the dev
  server restarts**, and the symptom lies: standard utilities on the same
  element still apply, so the screen looks almost right. Diagnose with
  `[...document.styleSheets].some(s => [...s.cssRules].some(r =>
  r.cssText.includes('<the arbitrary value>')))`. Check the PID —
  `pkill -f vite` may not kill it, and the old server keeps serving old CSS.

## Interaction & navigation (one philosophy, all scenes)

- **One design language across every scene** (grimório): same tokens,
  framing, focus/selection treatment and SFX vocabulary. A new screen adopts
  the existing scene grammar — it doesn't invent its own look.
- **One keyboard-navigation grammar, everywhere.** Scenes are **regions**
  (rail / header / content …) navigated with a shared primitive
  (`shared/lib/spatial-nav` geometry + `shared/lib/scene-nav`): arrows move
  the focus cursor **by layout** (2-D in a grid, 1-D in a list) and cross to
  a neighbouring region at the edge; **PageUp/PageDown** switch section;
  **Enter** activates; **Esc** goes up one level. Do not hand-roll per-scene
  `keydown` handlers — declare regions via `data-nav-region` /
  `data-nav-layout` and let the hook drive.
- The driver stands down while a **modal** overlay is open. A non-modal
  panel opts out with `data-nav-inline`, or opening it would kill the
  scene's arrows.
- **Keyboard never removes the mouse.** Arrow navigation is additive
  progressive enhancement over real, clickable controls — every item stays a
  native `a`/`button` and keeps its `hover`. Never gate a click behind a
  keyboard step, never trap focus. The mirror of that rule: **a gesture is
  never the only path** — anything draggable is redundant with a button.
- Keyboard nav is enabled only on laptop/desktop (`≥xl` + `pointer: fine`);
  it degrades cleanly to tap below that.

## Tests

- Run: `pnpm --filter frontend test` (vitest). Root `pnpm test` fans out.
- Every new function gets a test; bug fixes get a regression test.
- Mock I/O with **named fake classes** (`FakeFetch`, `FakeStorage`,
  `FakeRealtime`), not inline stubs. `@solidjs/testing-library` for
  components.
- Shared fixtures: `entities/character/__fixtures__/character.ts`
  (`makeCharacter`) and `shared/test/fake-storage`. A PARTIAL character
  blows up far from the cause — use the fixture.
- This jsdom has **no `AnimationEvent` constructor**: use `Event` +
  `defineProperty` to set `animationName`.
- The security boundary lives on the server — don't rely on UI gating for
  correctness; still gate UI by role for UX.
- **The E2E suite is unstable in PARALLEL** — run `-j 1` before
  investigating a failure. A spec that writes to the same seed needs
  `test.describe.configure({ mode: 'serial' })` and a self-cleaning setup.
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
  measure (prose), not for whole scenes. Column counts are **not monotonic**
  in width when the scene changes layout: below `lg` a catalogue owns the
  whole stage and needs MORE columns than at `lg`, where it shares with a
  detail pane.

## Typecheck + lint + format

- Typecheck: `pnpm --filter frontend typecheck` (tsc). `noUnusedLocals` +
  `noUnusedParameters` are **on** — prune unused imports.
- Lint: CI runs root `biome lint --write .`, and biome's `includes` cover
  `**/*.tsx`, so it lints this package (only its formatter is off). CI fails
  on any biome error or `--write` diff. Biome can't tell a custom component
  (`NumberInput`, `Select`) is a form control, so `noLabelWithoutControl`
  errors on a `<label>` that wraps one — associate via `for`+`id`, not
  nesting. `useSemanticElements` rejects `role="checkbox"`/`"radio"` on a
  `<button>`: the house pattern for a rich toggle is `aria-pressed`.
- **`aria-label` on a `<span>` is ignored** (no role to carry it). Use
  `aria-hidden` + an `sr-only` line. Biome catches it
  (`useAriaPropsSupportedByRole`).
- `routeTree.gen.ts` is generated + gitignored; CI regenerates on build.

## Reference

- Root [CLAUDE.md](../CLAUDE.md) and the [Tormenta 20 book](/t20-book.pdf).
- FSD: https://feature-sliced.design/
