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

### Bestiário expansion — residual
- Expansion #1 shipped: 20 → 51 monstros (livro p286-298). Cobertura
  atual: Masmorras (Glop, Rato Gigante, Orcs, Guerreiro de Chifres,
  Centopeia-Dragão, Aranha Gigante, Gárgula, Mantícora, Golem de Ferro),
  Ermos (Bandido, Guarda de Cidade, Centauro, Gnoll, Lobo, Cão do
  Inferno, Serpe, Basilisco, Ogro, Urso-Coruja, Grifo), Puristas
  (Recruta, Soldado, Sargento-Mor, Capelão, Capitão-Baluarte,
  Cavaleiro do Leopardo, Colosso Supremo), Reino dos Mortos (Zumbi,
  Turba Zumbi, Esqueleto, Esqueleto de Elite, Aparição, Necromante,
  Falange, Vampiro), Dragões (Filhote/Jovem/Adulto/Rei).
- **Ainda pendentes (livro p299-323)**: Trolls Nobres (Fintroll,
  Ganchador, Troll das Cavernas), Sszzaazitas (Cascavel, Jiboia, Naja,
  Nagah Mística/Guardião, Sucuri, Cultista, Lagash), Duyshidakk
  (Hobgoblin, Arauto de Thwor, Engenho de Guerra, Sombra de Thwor,
  Devorador de Medos), Dragões idades intermediárias (Adulto Extremo,
  Venerável), Tormenta (Otyugh, Uktril, Maníaco Lefou, Geraktril,
  Reishid, Sacerdote de Aharadak, Thuwarokk).

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
