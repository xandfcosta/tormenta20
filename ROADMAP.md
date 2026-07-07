# Roadmap

Living list of pending work. Ordered by leverage, not by required
sequence. Sourced from code TODOs, deferred design decisions, and
gaps observed after the D6 UI reskin sprint (2026-07-07).

Update this file whenever a bullet ships (link the PR) or a bullet
gets rescoped. Delete rather than mark "done" — closed work lives
in `git log`.

## Product features

### Spellbook + spell engine — residual
- SP1 shipped: read-only spellbook panel (browser over 199-entry catalog).
- SP2 shipped: CharacterSpell join + learn/unlearn/prepare persistence.
- SP3 shipped: cast engine (PM debit + augment validation +
  prepared-list gate for Clérigo/Druida).
- **Still pending**: Arcanista path distinction (Bruxo/Feiticeiro cast
  free, Mago prepares) — currently all Arcanistas cast free per MVP
  rule. Blocked on wiring class-choice data through to the cast check.
- **Still pending**: catalisadores instant-decrement replacement —
  needs per-school modifier targets to land so the item consumer can
  apply "reduz custo PM em magias de Evocação" etc. (memory:
  `spell_engine_deferred`).
- **Still pending**: item-based `pmLimit` stat gets checked frontend-
  side but not by the backend cast method (backend uses PDF base
  ½-level). Widen to compile the derived stat server-side or accept
  the conservative floor.

### Bestiário — coverage completo
- Expansion #1 (PR #238): 20 → 51 monstros (livro p286-298).
- Expansion #2: 51 → 80 monstros (livro p300-316). Cobertura Cap 7
  agora inclui todos os grupos: Masmorras, Ermos, Puristas, Reino dos
  Mortos, Duyshidakk, Sszzaazitas, Trolls Nobres, Dragões (5 idades +
  Enxame Kobold + Tirano do Terceiro), Tormenta (lefu completos +
  Sacerdote de Aharadak + Otyugh).
- Bestiário considerado feature-complete para o core do livro.

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
