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

- ⏳ **B.6 — Realtime (EM ANDAMENTO, o mais difícil)**. Gateway socket.io do Nest em
  `backend/src/realtime/`. Front consome em `frontend/src/shared/realtime/realtime.ts`
  (`io(REALTIME_ORIGIN, {withCredentials, transports:['websocket','polling'], autoConnect:false})`,
  namespace default `/`). **Protocolo**: socket.io-client **^4.8.3** → **Engine.IO v4
  (`EIO=4`)**. **Fase de mapeamento CONCLUÍDA** — detalhes abaixo.

  **FASE 0 — domínio transport-agnostic (pré-requisito do gateway).** Auditoria do
  pacote `api/` achou regra de negócio trançada com resposta HTTP (recebe `w`/`r`,
  escreve `writeError`/`writeJSON` no meio da regra) — o gateway WS não consegue reusar.
  Padrão-alvo (precedente `authorizedCharacter`): domínio devolve `(valor, status int,
  error)`; o HTTP mapeia p/ `writeError`, o WS p/ emit/WsException. Refactor sem mudança
  de comportamento (smoke-tests HTTP + `go test ./api/` como rede).
  - 🔴 **Leak (extrair núcleo):** authz `campaignAccess`/`ownedSession`/`ownedCampaign`
    (o 1º ainda lê `currentUser(r)` dentro da regra); apply-effect `applyManualPool`/
    `applyPool`/`applySpellBuff` (regra inteira — supremacia de pool, plan de displacement,
    buff de magia — tecida com `writeJSON`; `applySpellBuff` é o que o WS `apply-effect` chama).
  - 🟡 **Smell leve (só usa `r.Context()`):** `insertCharacter`/`healVitals`/`syncLevelVitals`/
    `equipLimitCheck`/`classDTOs` → trocar `r *http.Request` por `ctx context.Context`.
  - 🟢 **Transporte legítimo (não mexer):** `writeJSON`/`writeError`/`decodeJSON`/`intParam`/
    `issueSession`/`extractToken`/`currentUser`.
  - **Slices:** ✅ **0.1 authz** (`resolveRole` member-aware = owner→gm / membro→player / 403;
    `loadOwnedCampaign`; `loadSessionInCampaign` compartilhado por `ownedSession` owner-only
    e o futuro `sessionForCaller` member-aware do WS). ✅ **0.2 apply-effect** — núcleo
    `applySpellBuffEffect(ctx, charID, spellID, scope) (EffectDTO, status, error)` extraído
    (o WS `apply-effect` chama direto); introduzido o padrão de erro tipado **`fieldError`**
    + **`writeDomainError`** (domínio devolve error, HTTP renderiza o envelope rico; WS usa
    `.Error()`). **Deferido**: `applyManualPool`/`applyPool` (pools tempHp) — **sem consumidor
    WS**, múltiplas formas de resposta (cleared/superseded/displaced); extrair só se surgir
    consumidor ou como cleanup puro. ⏳ **0.3 smells de assinatura** (`r *http.Request` →
    `ctx`) — só quando um helper de B.6 precisar (ex.: computar vitais no descanso).

  **FASE 1 — helpers de domínio do gateway (transport-agnostic, sem rota HTTP).**
  ✅ **1.1 combatant/session** (`members.go`/`sessions.go`, todos `(valor, status, error)`):
  `sessionForCaller(ctx,user,camp,sess) (Session, role, status, err)` = `resolveRole` +
  `loadSessionInCampaign` (o `findOneForCaller` do WS, member-aware); `resolveCombatant(ctx,
  caller,camp,char) (combatant, status, err)` — char→camp→membership→authz (404/404/400/403),
  devolve `{name,hp,mp}`; `listPlayerCombatants(camp)` (role=player + vitais, p/ populate);
  `listMemberCharacterIds(camp)` (p/ descanso). Reusam sqlc existente (`GetCharacter`,
  `GetCampaign`, `IsCharacterMember`, `ListMembers`) — **zero query nova**. **Harness de teste
  DB-backed novo** (`realtime_domain_test.go`: `newTestServer` = `db.Open` temp + `NewServer`
  cat-less + seeds) cobrindo cada branch de authz — reusável nas próximas slices.
  ✅ **1.2 descanso** (`character_effects.go`, todos `(…, status, error)`, authz owner-or-GM via
  `authorizedCharacter`): `endScene` (deleta efeitos scope=scene), `endDay` (scope IN scene,day),
  `restVitals(condition)` (PV/PM += floor(level × `restMultiplier`), clamp ao max, persiste via
  `SetVitalsCurrent`; `restMultiplier`={ruim .5, normal 1, confortavel 2, luxuosa 3}; condition
  desconhecida→normal). **2 queries sqlc novas** (`DeleteEffectsByScope`, `DeleteSceneAndDayEffects`
  — IN estático, sem slice) regeneradas com `go run …/sqlc@latest generate` (v1.31.1; +23 linhas,
  0 removidas). Testes DB cobrindo seletividade de escopo + clamp + fallback + authz 403.
  **Nota**: os helpers recebem `ctx` direto — o *smell 0.3* não apareceu aqui (`restVitals` usa o
  row do `authorizedCharacter`, não `r`). ⏳ **1.3 ws-auth** (verify JWT, reusar do HTTP).

  **Auth (handshake, `ws-auth.ts`):** token via `handshake.auth.token` → `Authorization:
  Bearer` → **cookie `t20_session`** (nome de `COOKIE_NAME`), nessa ordem. `jwt.verify`
  (mesmo segredo do HTTP, `sub` = id) → `auth.findById(sub)` (nega usuário revogado).
  Falha: `socket.emit('unauthorized', {message})` + `disconnect(true)`. O user resolvido
  vai em `socket.data.user`; o **role** (`gm|player`) é resolvido POR-mensagem em
  `assertSessionAccess` e guardado em `socket.data.role`.

  **Estado (`SessionRuntimeState`)**: `{ initiative: InitiativeEntry[], round, turnIndex }`.
  `InitiativeEntry = {id(uuid), label, initiative, type:'character'|'npc', characterId?,
  hpCurrent?, hpMax?, mpCurrent?, mpMax?}`. Invariantes a portar EXATO (`session-state.service.ts`):
  lista **sempre ordenada DESC** por `initiative`, desempate `label.localeCompare`;
  `turnIndex` preserva quem está no turno após re-sort/insert/remove; `nextTurn` faz wrap
  → `round++`; `removeEntry` ajusta `turnIndex`; clamp de vitais `[0, max]`; `INITIATIVE_MAX_ENTRIES=50`.
  Persistência: `runtimeState` (coluna JSON **já existe** no schema, default correto) —
  gravação **fire-and-forget** com flag `dirty` + retry na próxima mutação → emite
  `persistence-warning` só quando a flag vira. Hidratação no 1º `load(sessionId)`.

  **Mensagens cliente→servidor** (payload sempre inclui `{campaignId, sessionId}` exceto
  onde nota): `join-session`(ack `{joined}`), `leave-session`(`{sessionId}` só; ack `{left}`),
  `get-session-state`(ack=state; refaz hpMax/mpMax do DB), `initiative-add`(`{entry}`, GM),
  `initiative-self`(`{characterId, initiative}`, upsert por characterId, **não** GM-gated),
  `initiative-update`(`{entryId, patch}`, GM), `initiative-remove`(`{entryId}`, GM),
  `initiative-next-turn`(GM), `initiative-reset`(GM), `initiative-populate`(GM),
  `session-rest`(`{scope:'scene'|'day', condition?}`, GM), `apply-effect`(`{entryId, spellId,
  scope?}`, GM), `vitals-patch`(`{entryId, patch:{hpCurrent?,mpCurrent?}}`),
  `vitals-delta`(`{entryId, hpDelta?, mpDelta?}`). Mutações vitais: player só edita o
  PRÓPRIO personagem (`assertVitalsEditable`); NPC = GM-only. Handlers com ack retornam o
  novo `state` (ou `{applied…}`, `{rested…}`).
  **Emits servidor→cliente:** `unauthorized{message}`, `session-state`(state, broadcast p/
  sala), `session-rest{sessionId,scope,condition}`, `effect-applied{sessionId,characterId,
  spellId}`, `presence{sessionId,users:PresenceUser[]}`, `persistence-warning{sessionId,dirty}`.
  **Presence** (`presence-registry.ts`): mapa sessionId→(socketId→user); roster **dedupe por
  userId** (multi-aba colapsa; GM vence). Broadcast em join/leave/disconnect.

  **⚠️ GAP DE DOMÍNIO DESCOBERTO** — o gateway chama métodos de serviço do Nest que **nunca
  viraram rota HTTP**, logo B.3/B.4 **não** os portaram. Precisam ser portados p/ Go como
  helpers (reusando sqlc + engine já existentes):
  1. `sessions.findOneForCaller` → resolver **role** (gm/player) member-aware. Go tem
     `campaignAccess`(bool) + `ownedSession`(owner); falta a variante que devolve role.
  2. `members.resolveCombatant(caller, campaign, charId)` → membership + owner-or-GM + stats
     `{name,hpCurrent,hpMax,mpCurrent,mpMax}` (usado por `initiative-add/self`).
  3. `members.listPlayerCombatants(campaign)` → PCs role=player com vitais (`initiative-populate`).
  4. `members.listMemberCharacterIds(campaign)` → ids (`session-rest`).
  5. `effects.endScene` / `endDay` / `restVitals(condition)` → mecânica de descanso T20
     (expira efeitos por escopo; restaura PV/PM). **Nenhuma existe em Go** (só há
     adjust/delete/`applyEffect`). `applyEffect` já existe (`apply_effect.go`).
  6. `characters.assertOwner` — Go tem `authorizedCharacter`; derivar a variante owner-only.

  **Lib Go (o risco central) — ✅ RESOLVIDA + VALIDADA POR SPIKE.** Escolhida:
  **`github.com/zishang520/socket.io/servers/socket/v3` v3.0.4** (port fiel do server JS).
  **Spike** (server Go mínimo + `socket.io-client` **4.8.3** real do front, node_modules)
  provou os 3 cenários: handshake por `auth.token`; handshake por **cookie `t20_session`**;
  no-auth rejeitado (`unauthorized` + disconnect). Ack round-trip (`{joined:"session:42"}`)
  e broadcast de sala (`session-state`) chegaram no cliente. **API mapeada** (usar direto no
  gateway):
  - `io := socket.NewServer(nil, nil)`; montar `http.Handle("/socket.io/", io.ServeHandler(nil))`.
  - `io.On("connection", func(clients ...any){ s := clients[0].(*socket.Socket); … })`.
  - **Ack** = `type Ack = func([]any, error)`; é o ÚLTIMO arg do handler quando o cliente
    passa callback: `cb := args[len(args)-1].(socket.Ack); cb([]any{payload}, nil)`.
  - **Handshake**: `s.Handshake().Auth["token"].(string)` + `.Headers` (`map[string]any`,
    valor string|[]string; ler `"Cookie"`). Extração espelha `ws-auth.ts` (auth→Bearer→cookie).
  - Per-socket: `s.SetData(x)` / `s.Data()` (guardar `{user,role}`). Salas: `s.Join(socket.Room(...))`,
    `s.Leave`, broadcast `io.To(room).Emit(ev, payload)`, direto `s.Emit`, `s.Disconnect(true)`.
  - Deps puxadas: `zishang520/socket.io/v3`, `parsers/socket/v3`, `servers/engine/v3`,
    `golang.org/x/{crypto,net,sys,text}`. **Regra de ouro**: SÓ em `api/`/`cmd/api`, **nunca**
    em `engine/` (senão quebra `GOOS=js` do WASM). Spike descartável em
    `$SCRATCH/sio-spike` (não commitado).
  **PRÓXIMOS PASSOS de implementação** (lib desbloqueada): (1) portar os 6 helpers de domínio;
  (2) `session-state` runtime (mapa em memória + invariantes de sort/turn/clamp + persist
  fire-and-forget na coluna `runtimeState`); (3) presence-registry (dedupe por userId);
  (4) ws-auth (reusar o verify JWT do HTTP); (5) os ~14 handlers + emits; (6) montar no
  `cmd/api`. Testar com o mesmo padrão do spike (client socket.io-client real).
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
