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

**STATUS: COMPLETA** (commits `5dc3587`..A.3.3). Todos os breakdowns + effects-cru
migrados; só `WeaponFormulaCards`/`point-buy` ficam TS de propósito (ports novos).

**Passos:**
1. ✅ **FEITO** — Hook `useComputedSheet(char)` (`entities/character/computed-sheet.ts`)
   → `computeSheetV2` (engine), MODE-gated com o montador TS `assembleSheetV2` p/
   teste (reusa os breakdowns como oráculo, sem wasm no vitest). O oráculo de paridade
   (`parity-oracle.test.ts`) usa o MESMO `assembleSheetV2` — fonte única.
2. ✅ **FEITO** — 13 consumidores de breakdown lêem campos do `ComputedSheetV2`:
   `combat-magic-stats` (Combat/Saves/Magic — `WeaponFormulaCards` fica TS de propósito),
   `mobile-def-chip`, `character-stage`, `dossier-drawer`, `vitals-aside`, `sheet-header`
   (disp/fly; ainda lê `effects.flags` p/ fatigue), `expertise-row`, `expertises-panel`,
   `bag-panel`, `cast-spell-dialog`, `initiative-roll`, `sheet-search-index`,
   `stances-section` (tempHp), `use-power-dialog`. Helpers `expertiseFromSheet` /
   `requireExpertise` p/ perícias por nome. Valores idênticos (provados).
3. Expor no Go/WASM as regras que faltavam e trocar os consumidores de effects-cru:
   - 3.1 ✅ **FEITO** — CD de magia por classe aplicável (`spell-row computeBestCd`):
     novo campo `spellCdByAttribute` no `ComputedSheetV2` (Go + TS + oráculo regenerado,
     paridade E2E wasm provada). `spell-row`/`spellbook-panel` lêem o mapa; drop de
     `effects` desses dois.
   - 3.2 ✅ **FEITO** — `resolveConditionalDisplay` (posturas, `stances-section`):
     exposto via WASM (`resolveConditionalDisplay` em `cmd/wasm`), choke point MODE-gated
     `entities/character/conditional-display.ts` (`resolveStanceDisplay`), E2E ok.
   - 3.3 ✅ **FEITO** — `effect-source.equippedItemFlagEffects` (flags por-item,
     `effects-panel`): nova função Go `ComputeEquippedFlags(items)` (`engine/equipped_flags.go`)
     + WASM `computeEquippedFlags` + oráculo `equippedFlags` + teste de paridade Go
     (`equipped_flags_parity_test.go`, byte-equal) + E2E wasm (16/16, 4 chars com flags).
     `effect-source.ts` MODE-gated: prod chama o engine, o ramo TS (`tsEquippedFlags`) é
     o oráculo e some por DCE — levando junto o **último consumidor em prod de
     `computeItemEffects`**. Espelha o TS EXATO (só `catalog.modifiers` base; flags
     ordenadas por item p/ bater com `MarshalJSON`).
4. ✅ **DCE VERIFICADO** — os breakdowns sem consumidor em prod (`defenseTotal`,
   `displacementTotal`, `flySpeedTotal`, `inventorySlotsTotal`, `pmLimitTotal`,
   `bestBaseSpellCd`, `spellDCBonus`, `pmCostMod`, `characterDamageReduction`,
   `tempHpFromPowers`) saíram do bundle: labels só-de-breakdown (`"Bárbaro (p47)"`,
   `"Alma de Bronze (Fúria, p41)"`, …) dão **0** em `dist/assets/*.js` e vêm do wasm em
   runtime. `attributeTotal`/`expertiseTotalWithItems`/`statFor` PERMANECEM (usados por
   `WeaponFormulaCards`, deferido). `computeItemEffects` agora **DCE** (item 3.3):
   os dois consumidores restantes (`derived.ts resolveEffects` + `effect-source.ts
   tsEquippedFlags`) estão ambos no ramo `MODE==='test'`.

**"Tudo da ficha" — TAMBÉM FEITO** (eram os ports novos deferidos):
- ✅ **point-buy (criação)** → `engine/pointbuy.go` (`PointBuyStatusFor` = spent +
  warnings, strings pt-BR byte-iguais incl. U+2212) + WASM `pointBuyStatus` +
  choke point `features/character-build/point-buy.ts` + swap `atributos-step`.
  Go unit test + E2E. `pointBuySpent`/`pointBuyWarnings` DCE (0 no bundle).
- ✅ **weapon formula cards** → `engine/weapons.go` (`ComputeWeaponCards`: skill
  ranged?Pontaria:Luta, dano soma Força melee/thrown, crit) + WASM
  `computeWeaponCards` + oráculo `weaponCards` + teste paridade Go (byte-equal, 16
  chars) + E2E + choke point `entities/character/weapon-cards.ts` (`useWeaponCards`).
  `WeaponFormulaCards` agora só renderiza os números do Go; **`statFor` e
  `expertiseTotalWithItems` saíram do prod** (só nos montadores MODE-gated).
  `attributeTotal` permanece (usado por `level-vitals` p/ montar o input do vitals
  engine); `wieldedWeaponEntries` permanece (lookup de catálogo, via `hasWieldedWeapon`).

Fase A **100% concluída**: nenhuma regra de ficha é autorada em TS em prod.

**Padrão E2E wasm** (recriar em `$CLAUDE_JOB_DIR/tmp`): `wasm_exec.js` via
`vm.runInThisContext`, instanciar `t20.wasm`, `primeEngineCatalogs` a partir de
`engine-go/parity/_catalogs.json`, comparar `computeSheetV2`/`resolveConditionalDisplay`
contra o oráculo/valor esperado. Mapas do Go serializam com chaves ordenadas
(alfabético) — comparar deep/canônico, não string-equal.

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

**Stack de dados escolhida (SQL-first, sem mágica de ORM):**
- **`sqlc`** — modelagem+queries: você escreve o SQL (`query.sql`) sobre um
  `schema.sql`, e o sqlc gera código Go **type-safe** (structs + métodos). Zero
  reflection, controle total do SQL; relações/eager-load = JOINs escritos à mão
  (nada de `Preload` automático).
- **`goose`** (ou **Atlas** se quiser diff declarativo estilo `prisma migrate`) —
  migrations: arquivos `.sql` up/down versionados.
- **`modernc.org/sqlite`** — driver SQLite **Go puro (sem cgo)**, mantém o
  `engine-go` compilando/cross-compilando limpo como já é.
- Fluxo: `schema.sql` (fonte da verdade do DB, migrado do `schema.prisma`) →
  goose aplica → sqlc gera o acesso tipado. Alternativa considerada e preterida:
  **Ent+Atlas** (mais "Prisma-like", client tipado + eager-load ergonômico, mas
  mais mágica/código gerado).

**Porte (demais decisões a tomar na sessão nova):** router Go (net/http vs chi/echo),
JWT/sessão, WebSocket (gorilla/nhooyr/coder) p/ o realtime, servir catálogos,
portar o seed. Depois:
apontar `/api` (e o socket) do Vite pro server Go, e o `pnpm dev` raiz roda só
`frontend` + `engine-go` (server Go que serve API + WASM). O engine Go já existe —
a API é o grosso.

**Ordem sugerida:** Fase A (rápida, baixo risco, regras já provadas em Go) →
Fase B (grande). Ou B direto se a prioridade é a API — são independentes.

---

## Fase B — PROGRESSO DE IMPLEMENTAÇÃO (começar a próxima sessão aqui)

**Decisões travadas** (via AskUserQuestion, não re-perguntar):
- **Realtime**: lib Go **socket.io-compatível** (não reescrever em WebSocket cru) —
  precisa falar o protocolo Engine.IO/Socket.IO que o front (`socket.io-client`) usa.
- **Migrations**: **goose** (embarcado, roda no startup).
- **Cutover**: **big-bang** — construir o Go até paridade total ATRÁS do Nest, virar o
  proxy `/api` + `/socket.io` do Vite no fim, manter o Nest para rollback.
- **Restrição**: o Nest continua a autoridade até B.7; o app não muda até o cutover.
  Commits: uma linha, sem trailer de co-autor.

**Stack já montada** (`engine-go/`, módulo `t20engine`):
- Router **chi** + **chi/cors**; **golang-jwt/jwt v5** (HS256, `sub` assinado como
  NÚMERO p/ compat do cookie do Nest); **x/crypto/bcrypt** (cost 12);
  **modernc.org/sqlite** (driver Go puro, sem cgo — mantém o WASM limpo);
  **pressly/goose/v3** (migrations embarcadas em `db/migrations`, rodam no startup);
  **sqlc** (codegen commitado em `db/sqlcgen/`).
- **Regra de ouro das deps**: libs novas só em `api/`, `db/`, `catalog/`, `cmd/api`,
  `cmd/seed`. **NUNCA** em `engine/` — senão `GOOS=js GOARCH=wasm go build ./cmd/wasm`
  quebra. Os dois motores de `engine/` seguem stdlib-puro.
- **sqlc quirk**: colunas viram minúsculas → campos gerados tipo `Ownerid`, `Hpmax`,
  `Catalogspellid`. Structs sqlc = tipos internos de DB; os **DTOs à mão**
  (`character_dto.go`) carregam o contrato camelCase do JSON (espelha Nest DTO×Prisma).
  Args nomeados via `sqlc.arg('foo')` PRESERVAM camelCase → `Foo`; `?` posicional
  vira minúsculo. `sqlc.narg`=nullable. `sqlc.slice('ids')`=`[]int64` IN-clause.
- **writeJSON** usa `enc.SetEscapeHTML(false)` p/ paridade byte-a-byte com
  `JSON.stringify` (`<>&` em modifiers).
- **Catálogos**: servidos de JSON embarcado (`catalog/data/*.json`, `//go:embed`),
  exportados pelo harness do front `GEN_CATALOGS=1`
  (`frontend/src/entities/catalog/catalog-export.test.ts`).

**Slices — estado:**
- ✅ **B.0** — fundação: `cmd/api`, chi, config, `/health`, middleware JWT.
- ✅ **B.1** — data layer: `schema`/goose migrations, sqlc, modernc/sqlite. Seed via
  `cmd/seed` (HTTP-driven, `seed-data.json`; senha ≥8 chars: "mestre123456").
  **Roster completo**: espelha o seed do Nest (`backend/src/seed.ts`) — 3 contas
  (`mestre`/`jogador`/`teste` @t20.local), 15 chars diversos. O driver enriquece o
  create body a partir do catálogo (nome+slots de item via `catalog.LookupItem`) e
  injeta vitals (9999 → o engine cura p/ o max real) + as 10 perícias treinadas
  padrão nos não-`simple`; HP danificado (`hpFraction`) via PATCH `/vitals` pós-cura,
  efeito de cena via `consume` do `cosmetico`. Ids inválidos do seed antigo
  corrigidos (`machado-batalha`, não `-de-`; `misseis-magicos` não existe no catálogo →
  usa o ARCANE_SPELLBOOK do Nest). **Não é idempotente** — rode em DB limpo (`rm -f $DB*`).
- ✅ **B.2** — auth: register/login/logout/me, bcrypt + cookie JWT.
- ✅ **B.3 — COMPLETA** — domínio Characters: CRUD + ~30 rotas de mutação. Todas as
  rotas em `api/server.go`. Cobre: vitals/damage (roteamento temp-HP + `planDamage`),
  level/class-level (recompute via `VitalsForCharacter`), abilities, proficiencies,
  items (add/patch/equip/delete/**consume**), conditions (catálogo), expertises
  (add/update/delete), spells (learn/unlearn/prepared/**cast**), **active-effects**
  (`POST` = spell buff + temp-HP manual com supremacia vale-o-maior; `PATCH`/`DELETE`).
  **Deferidos documentados (sem consumidor vivo no front):** ramo power-grant do
  applyEffect (`501`; precisa registry de ativação + compute de atributo server-side)
  e `GET /sheet`. Paridade verificada em smoke-tests + engine de vitals integrado.
- ✅ **B.4** — campaigns, members, invites, sessions, users. Helpers de authz
  (`ownedCampaign`, `campaignAccess`, `ownedSession`); roles player/gm; 409 de PC
  prévio; ciclo de sessão planned→active→ended→reopen.
- ✅ **B.5** — catalog endpoint: serve 15 resources + `options`. Retro-desbloqueou o
  cast (catálogo de magias) e o consume (specs de consumível) do B.3.

- ⏳ **B.6 — Realtime (PRÓXIMO, o mais difícil)**. Gateway socket.io do Nest em
  `backend/src/realtime/` (`realtime.gateway.ts`, `session-state.service.ts`,
  `presence-registry.ts`, `ws-auth.ts`). Front consome em
  `frontend/src/shared/realtime/realtime.ts` (`io(...)`, namespace default).
  **Auth do socket**: mesmo cookie JWT do HTTP (`ws-auth.ts`), emite `unauthorized`
  no handshake ruim. **Salas**: `sessionRoom(sessionId)`.
  **Mensagens cliente→servidor (`@SubscribeMessage`, ~18):** `join-session`,
  `leave-session`, `get-session-state`, `initiative-add`, `initiative-self`,
  `initiative-update`, `initiative-remove`, `initiative-next-turn`,
  `initiative-reset`, `initiative-populate`, `session-rest`, `apply-effect`,
  `vitals-patch`, `vitals-delta`.
  **Emits servidor→cliente:** `unauthorized`, `session-state` (broadcast do estado
  do tracker), `session-rest` (`{scope:'scene'|'day'}`), `effect-applied`,
  `presence` (`{sessionId, users}`), `persistence-warning` (`{sessionId, dirty}`).
  Muitos handlers usam ACK-callback (o front passa callback em `join-session`).
  **Risco**: achar/validar a lib Go socket.io-compatível que fale Engine.IO v4
  (o `socket.io-client` do front). Começar mapeando payloads exatos de cada msg em
  `realtime.gateway.ts` + o formato do `SessionRuntimeState` (front
  `SessionRuntimeState`/`PresenceUser`) antes de escrever qualquer Go.
- ⏳ **B.7 — Cutover**. Virar o proxy `/api` + `/socket.io` do Vite p/ o server Go;
  `pnpm dev` roda front + `cmd/api` (que serve API + WASM). Nest fica p/ rollback.

**Como rodar/testar a API Go hoje** (padrão dos smoke-tests desta fase):
```bash
cd engine-go && go build ./...
DB=/tmp/t20.db; rm -f $DB*; JAR=/tmp/t20.cookies
PORT=3036 DATABASE_URL="file:$DB" JWT_SECRET=x go run ./cmd/api &   # migra no startup
curl -s -c $JAR -X POST localhost:3036/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"gm@test.com","password":"password123"}'
# depois -b $JAR nas rotas autenticadas. gofmt -w + go vet ./api/ antes de commitar.
```

---

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
