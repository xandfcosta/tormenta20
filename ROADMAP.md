# Roadmap

Living list of pending work. Ordered by leverage, not by required
sequence. Sourced from code TODOs, deferred design decisions, and
gaps observed after the D6 UI reskin sprint (2026-07-07).

Update this file whenever a bullet ships (link the PR) or a bullet
gets rescoped. Delete rather than mark "done" — closed work lives
in `git log`.

## Product features

### Spellbook + spell engine
- `t20-data/spells.ts` (256 LOC) pins **mechanics only**: PM cost per
  círculo, save CD formula, execução/alcance/duração/área taxonomy,
  augment validation. No catalog, no consumer.
- Char sheet shows Limit PM / CD Magia / Custo PM but has no magias
  UI (no spellbook tab, no cast action, no prepared list).
- Catalisadores currently ship as instant-decrement (memory:
  `spell_engine_deferred`) — waiting on engine + per-school modifier
  targets.
- Cap 4 magias catalog residual (~150 spells to encode).
- Needed pieces: catalog file, spellbook panel under `character-sheet/`,
  cast mutation with PM debit, augment picker UI, prepared-list gate
  per class (Mago/Feiticeiro/Bardo/etc.).

### Bestiário expansion
- `t20-data/bestiary.ts` currently 20 monsters (549 LOC).
- Book has ~150+ criaturas across Cap 8.
- Structure already handles ataques + habilidades especiais + tipo
  taxonomy; expansion is pure data entry against existing type shape.

### Encounter → session tracker bridge
- `/gm/encounters` builds an encounter (partyLevel, monsters, ND, XP).
- No path to push composition into an active session's initiative
  tracker — GM currently re-adds each entry manually via WS.
- Needed: "Enviar para sessão" action + campaign/session picker +
  batched `initiative-add` per monster (respect `INITIATIVE_MAX_ENTRIES=50`).

### GM invite / deep-link flow
- Backend memory (`campaign-members.service.ts:63`): "GM invites →
  player accepts" pattern deliberately out of scope. Currently only
  self-join by campaign ID (`/campaigns/join` post-OC1).
- Needed: invite token endpoint, deep-link route
  (`/campaigns/join?token=...`), auto-fill campaign ID + validate.

## Persistence residual

### WS→DB current-vitals write-through
- `backend/src/realtime/session-state.service.ts:338` TODO.
- P3 shipped end-of-session batch commit (`session end` writes hp/mp
  to Character rows).
- P5 shipped `hpMax` auto-refresh from DB on `get-session-state`.
- Real-time `hpCurrent`/`mpCurrent` write-through on every `patchVitals`
  still deferred — page refresh mid-combat loses tracker state.
- Design decision needed: debounced write? per-event? env flag like
  `WS_VITALS_WRITETHROUGH=1` used in P3?

## Infra / perf

### Bundle code-split
- Build shows chunks > 500KB warning:
  - `index-CiZOYcYl.js` 424KB (main)
  - `_id-C9jE9sJZ.js` 119KB (char sheet)
- TanStack Router already lazy per-route. Sub-tree splitting inside
  char sheet subfiles could help; also lucide-react tree-shake audit.

### E2E coverage
- Playwright ripped (PR #220) — pixel-diff too fragile.
- Current coverage: 186 vitest (structural DOM snapshots via
  testing-library) + 295 backend jest.
- Gap: full flow (login → create character → join campaign →
  session → tracker mutations) has no automated regression net.
- Options: Playwright with structural (`toHaveText`, `toBeVisible`)
  instead of screenshot; or accept and rely on unit + manual.

## Char sheet split — housekeeping

Not urgent — all files under 500 LOC rule. Largest remaining
sub-files (for future splits when they grow):

| File | Lines |
|------|-------|
| `expertise-row.tsx` | 498 |
| `class-abilities.tsx` | 485 |
| `effects-panel.tsx` | 474 |
| `catalog-picker-dialog.tsx` | 471 |
