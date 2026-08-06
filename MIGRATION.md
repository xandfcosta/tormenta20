# MIGRATION — TS → Go (ponto único de entrada)

> Documento **único** do restante da migração de TypeScript para Go. Substitui e
> consolida os antigos `HANDOFF.md`, `PORT-PLAN.md`, `CONTINUE.md`,
> `INC3-VITALS-PLAN.md` (apagados). Comece uma sessão nova por aqui.
> Contexto histórico fino nas memórias `project_fase3_rtt_bench` e
> `project_front_decouple_catalog`.

## Estado atual (o que já é Go × o que ainda é TS)

**Já roda 100% em Go/WASM no browser (produção):**
- **Derive de item-effects** (Inc.2): coleta (`activeItemsFor` + catálogos) +
  resolução non-stacking (`computeItemEffects`). Front chama `computeEffects`
  (WASM) no choke point `derived.ts resolveEffects`. A coleta/resolução TS é
  **test-only** (gated por `import.meta.env.MODE === 'test'`) → **DCE** do bundle.
- **Vitals PV/PM** (Inc.3): `computeVitals` (WASM) via choke point
  `entities/character/vital-pools.ts computeVitalPools`. Pipeline TS test-only → DCE.

**Motor Go** (`engine-go/`, módulo `t20engine`):
- Catalog-driven (Inc.2/3): `engine/collect*.go`, `itemeffects.go`, `breakdowns*.go`,
  `vitals_v2.go`, `catalogs.go`. Exposto via `cmd/wasm` (`primeEngineCatalogs`,
  `computeEffects`, `computeSheetV2`, `computeVitals`) + wrapper
  `frontend/src/shared/lib/engine-wasm.ts`.
- MVP catalog-free (`engine/compute.go`, `vitals.go`) → serve o `/sheet` do
  bench/backend. **Engine separado**, fora do escopo do decouple do front.
- Paridade: oráculos em `engine-go/parity/*.json` (16 seeds), gerados por
  `frontend/.../parity-oracle.test.ts`. Testes `*_parity_test.go` batem byte-equal.

**Ainda em TS (o que falta migrar) — duas frentes:**

## Fase A — consolidar as regras de derivação DUPLICADAS (escolhida pelo dono)

As ~14 fórmulas de breakdown (`derived.ts`) **já estão portadas em Go**
(`ComputeSheetV2`, provado byte-equal), mas a UI ainda roda as **cópias TS**. Meta:
a UI **consome** o `ComputedSheetV2` do Go; nenhuma regra de ficha é autorada em TS.

**Passos:**
1. Hook `useComputedSheet(char)` em `entities/character` → `computeSheetV2(char,
   activeConditionals)` (engine), MODE-gated com montador TS p/ teste (reusa os
   breakdowns como oráculo, sem wasm no vitest). Padrão idêntico a `resolveEffects`.
   `ComputedSheetV2` já está COMPLETO (tem `attribute`, `attackAll`, `damageAll`).
2. Trocar os ~16 consumidores de breakdown p/ ler campos do `ComputedSheetV2`:
   `combat-magic-stats`, `mobile-def-chip`, `character-stage`, `dossier-drawer`,
   `vitals-aside`, `sheet-header`, `expertise-row`, `expertises-panel`, `spell-row`,
   `bag-panel`, `cast-spell-dialog`, `initiative-roll`, `sheet-search-index`,
   `stances-section`, `use-power-dialog`, `spellbook-panel`. Ex.: `defenseTotal(c,eff)`
   → `sheet.defense`; `expertiseTotalWithItems(c,luta,eff)` → `sheet.expertises.find`.
   Valores idênticos (provados) → sem regressão visual.
3. Expor no Go/WASM as regras que já estão em Go mas ainda não expostas, e trocar
   os últimos consumidores de effects-cru:
   - `resolveConditionalDisplay` (posturas, `stances-section`) → já existe em Go
     (`ResolveConditionalDisplay`); expor via WASM + consumir.
   - CD de magia por classe aplicável (`spell-row computeBestCd`) → usa `spellSaveDc`
     (já em Go); expor uma função ou os CDs por-atributo no `ComputedSheetV2`.
   - `effect-source.equippedItemFlagEffects` (flags por-item, `effects-panel`) →
     nova função Go `computeEquippedFlags(char)` (a resolução já é `ComputeItemEffects`;
     falta a proveniência por-item).
4. Depois disso: `defenseTotal`/`statFor`/etc. ficam sem consumidor em prod →
   **DCE**. `statFor` remanescente é só LOOKUP (não regra). Verificar DCE por strings
   (padrão: os breakdowns somem do `dist/assets`; ver método na Fase A abaixo).

**NÃO nesta fase** (ainda TS-only, portar depois se quiser "tudo da ficha"):
montagem de ataque/dano de arma (`WeaponFormulaCards`: melee soma Força, crítico)
e `point-buy` (criação). São ports novos, não duplicações.

## Fase B — a API: NestJS → Go (a frente grande)

**É AQUI que "a UI usar as rotas em Go" e "`pnpm dev` rodar só o server Go"
acontecem.** Hoje é impossível: o server Go (`cmd/server`) só tem `/health` +
`/sheet` (compute). A API real é o **NestJS** (`backend/`, porta :3000), e a UI a
consome via `/api` (proxy do Vite → :3000; socket.io idem).

**O que o NestJS faz (inventário a portar):**
- **HTTP API** — 8 controllers: `auth` (JWT), `users`, `campaigns` +
  `campaign-invites`, `campaign-members`, `sessions`, `catalog` (serve os catálogos
  que o front prima), `characters` (CRUD + itens/perícias/efeitos/spells + `/sheet`
  server-computed via mapper).
- **DB** — Prisma 7 + SQLite (`@prisma/adapter-better-sqlite3`). Models: User,
  Campaign, CampaignMember, Session, Character, CharacterSpell, ActiveEffect,
  CharacterItem, CharacterRace, CharacterClass, CharacterExpertise.
- **Realtime** — socket.io gateway (`realtime/`): presence-registry, session-state,
  ws-auth. Front usa via `shared/realtime` (`useSessionSocket`).
- **Seed** — `backend/src/seed.ts` (16 seed chars; ver memória `project_seed`).

**Porte (decisões a tomar na sessão nova):** router Go (net/http vs chi/echo),
acesso a SQLite (database/sql + modernc/mattn, ou ORM), JWT/sessão, WebSocket
(coturn/gorilla/nhooyr) p/ o realtime, servir catálogos, portar o seed. Depois:
apontar `/api` (e o socket) do Vite pro server Go, e o `pnpm dev` raiz roda só
`frontend` + `engine-go` (server Go que serve API + WASM). O engine Go já existe —
a API é o grosso.

**Ordem sugerida:** Fase A (rápida, baixo risco, regras já provadas em Go) →
Fase B (grande). Ou B direto se a prioridade é a API — são independentes.

## Verificação (rodar por slice)

```bash
cd engine-go && go test ./... && gofmt -l engine/ cmd/   # Go verde + formatado
pnpm --filter frontend test                              # FE (usa derive TS em test)
pnpm --filter frontend typecheck                         # tsgo limpo
pnpm exec biome lint --write .                           # == CI (falha se gerar diff)
GEN_ORACLE=1 pnpm --filter frontend test parity-oracle   # regenera oráculos
# build + prova de DCE (regra saiu do bundle):
pnpm --filter frontend build
grep -rl "<string única da regra TS>" frontend/dist/assets/   # deve dar 0
```

Padrão de prova E2E do WASM (não commitado, recriar em `$CLAUDE_JOB_DIR/tmp`):
carrega `frontend/public/engine/{wasm_exec.js,t20.wasm}`, `primeEngineCatalogs`
a partir de `engine-go/parity/_catalogs.json`, e compara `computeX(char)` contra o
oráculo. Ver histórico em `project_fase3_rtt_bench`.

## Cuidados
- **MODE-gate**: o ramo TS (`import.meta.env.MODE === 'test'`) é o ORÁCULO de teste
  e some do bundle por DCE. Vitest NÃO carrega wasm — usa o derive TS. Não quebre isso.
- **Engine load-bearing**: em prod, `resolveEffects`/`computeVitalPools` dão throw se
  o engine não primou; `ensureEngineCatalogs` (best-effort no `__root` beforeLoad)
  warma o wasm. Falha do wasm = erro só nas páginas de ficha, não no app todo.
- **Dois motores Go**: o MVP catalog-free (`compute.go`/`vitals.go`) serve o `/sheet`
  do backend; o real catalog-driven (`collect*`/`breakdowns*`/`vitals_v2`) serve o
  front. Não unificar sem querer.
- **`build-engine-wasm.sh`**: `predev`/`prebuild` do frontend + CI (`setup-go`) já
  buildam o wasm. Air (`engine-go dev`) rebuilda no hot-reload.
