# Roadmap

Living list of pending work. Ordered by leverage, not by required
sequence. Sourced from code TODOs, deferred design decisions, and
gaps observed after the D6 UI reskin sprint (2026-07-07).

Update this file whenever a bullet ships (link the PR) or a bullet
gets rescoped. Delete rather than mark "done" — closed work lives
in `git log`.

## Product features

### Spellbook + spell engine — residual
- SP1/2/3 shipped: catalog browser, persistence, cast engine (PM debit
  + augment validation + prepared-list gate).
- SP4 shipped: Arcanista path distinction (Bruxo/Feiticeiro cast free,
  Mago prepares) + item-based pmLimit compiled into backend cast
  validation (equipped item catalog + improvements + material mods).
- **Still pending**: catalisadores instant-decrement replacement —
  needs per-school pmCost modifier target + ActiveEffect wiring so a
  drunk Alquimista's Fire can knock 2 PM off the next Evocação cast
  (memory: `spell_engine_deferred`). Non-trivial: schema + engine +
  UI to display buff + cast-time consumption.

### Bestiário — coverage completo
- Expansion #1 (PR #238): 20 → 51 monstros (livro p286-298).
- Expansion #2: 51 → 80 monstros (livro p300-316). Cobertura Cap 7
  agora inclui todos os grupos: Masmorras, Ermos, Puristas, Reino dos
  Mortos, Duyshidakk, Sszzaazitas, Trolls Nobres, Dragões (5 idades +
  Enxame Kobold + Tirano do Terceiro), Tormenta (lefu completos +
  Sacerdote de Aharadak + Otyugh).
- Bestiário considerado feature-complete para o core do livro.

## Infra / perf

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
