# Handoff — migração de dados + engine (decouple do front → WASM)

> Documento de passagem de sessão. Resume onde o projeto está, o que foi feito e
> o próximo passo concreto. Detalhes finos estão nos arquivos de memória
> `project_front_decouple_catalog` e `project_fase3_rtt_bench`.

## TL;DR do estado

1. **Decouple de DADOS — 100% concluído e commitado.** O front NÃO empacota mais
   nenhum catálogo estático (spells, itens, bestiário, raças, origens, classes,
   poderes, condições, tormenta, divine powers, activation registry). **~490KB raw
   fora do bundle.** Tudo é servido por `GET /catalog/*` e cacheado no front.
2. **Fase 3 (thin-client vs WASM) — medida e decidida: WASM.** Compute é grátis
   (0.016–0.28ms/sheet); só a rede diferencia. Escolhido WASM (fonte única Go,
   instant, offline; custo 0.95MB gz de load único).
3. **WASM — fundação + bridge prontos e PROVADOS.** engine-go compila pra wasm,
   roda no browser, e computa **idêntico ao servidor**. O mapper
   `Character → CharacterInput` tem **paridade byte-equal** com o backend nos 16
   seed chars. Falta a troca de shape na UI (o grosso do Inc.2).

## O que rodar (dev)

```bash
pnpm dev        # sobe TUDO: t20-data(watch) + backend(:3000) + frontend(:5173)
                # + engine-go via air (rebuilda o wasm em frontend/public/engine/
                #   + server Go :3002) — hot-reload ao editar .go
pnpm dev:app    # só backend + frontend (sem Go/wasm)
```

- **air** precisa estar instalado: `go install github.com/air-verse/air@latest`
  (fica em `~/go/bin` / mise go bin; já está no PATH via mise nesta máquina).
- O **wasm** (`frontend/public/engine/t20.wasm`, ~0.95MB gz) é gitignored; é
  gerado por `frontend/scripts/build-engine-wasm.sh` (o air roda isso). Rode o
  script uma vez (ou `pnpm dev`) antes de builds que dependam do wasm.
- Seed/login: `mestre@tormenta.com` / `123456` (GM, 10 chars) e
  `jogador@tormenta.com` / `123456` (6 chars). DB em `backend/dev.db` (seeded).

## Commits desta sessão (todos em `main`)

```
8377b41 chore(dev): pnpm dev sobe front+back+go(air); air rebuilda wasm + server :3002
906f431 feat(frontend): WASM Inc.2 — mapper Character→CharacterInput + paridade 16 seed chars
e828433 feat(frontend): WASM Inc.1 — engine-go→wasm build pipeline + engine-wasm wrapper
f882d93 feat(frontend): decouple catálogos finais via cache — t20-classes (40KB) eliminado
48da1bd feat(backend): catalog resources origens/tormenta-powers/divine-powers/activations
6ee9002 feat(t20-data): decouple catálogos finais data-free + parametrizações
6a616a4 feat(frontend): A spells via cache — SPELL_CATALOG ~163KB fora (206→43KB core)
2ab9ed8 feat(backend): A spells — validateApplyBuff recebe hasBuff
42f37f9 feat(t20-data): A spells — rules/spells validators data-free
c5f6039 feat(frontend): B.3 abilities via cache — t20-abilities 149→2.28KB
db2656a feat(backend): B.3 catalog resources abilities
e10568b feat(t20-data): B.3 split abilities logic + vital-grants DI
```

Verificação global: **FE 583 / BE 492 / t20-data 4637** testes verdes; typechecks
limpos; biome 0 erros. Front build sem chunks de catálogo (t20-classes/spells/
bestiary/items eliminados; sobra ~29KB de LÓGICA de regra compartilhada).

## Arquitetura do decouple (padrão a seguir)

Cada catálogo tem um `*-cache` no front (`frontend/src/shared/lib/`), primado por
`ensureCatalogs` (`entities/catalog/ensure-catalogs.ts`, chamado no
`__root.tsx` beforeLoad) e por `test-setup.ts`:

- `catalog-cache` (itens), `abilities-cache` (raças/origens/class-powers/
  general-powers/deuses/granted-powers), `spell-cache`, `racas-cache`
  (RACAS+ORIGENS), `rules-catalog-cache` (CONDITIONS+TORMENTA_POWERS),
  `divine-powers-cache`, `activation-cache`.
- Lógica de regra PURA fica compartilhada em t20-data (o front value-importa —
  tree-shake). Helpers co-locados com dados foram split pra módulos data-free
  (padrão item-classify). Funções-engine que o front importa e que leem dados
  foram parametrizadas (validateCast/validateLearnSpell, collectVitalGrants com
  resolver injetado, deformidadeAvailablePowers, getActivation → cache).
- **Lição:** sempre re-grepar o BUNDLE buildado por strings de dados — âncoras
  via helper co-locado ou função-engine transitiva não aparecem num sweep só do
  fonte.

## WASM — o que já existe

- `engine-go/` — port Go do `computeCharacterSheet` (compila; `go build ./...` OK).
  `cmd/wasm` (browser) + `cmd/server` (:3002 POST /sheet). `bench/payloads/*` (16
  seed chars, input CharacterInput) + `bench/expected/*` (oráculo ComputedSheet).
- `frontend/src/shared/lib/engine-wasm.ts` — `ensureEngine()` (lazy-load do
  wasm via <script> glue + instantiateStreaming, cache once) e
  `computeSheet(input: CharacterInput): ComputedSheet`. **DORMENTE** (nada importa
  ainda). Provado no browser: computa deep-equal ao oráculo (lenda-nv20/kharvos/
  curandeira); load+instantiate ~40-50ms local.
- `frontend/src/entities/character/to-character-input.ts` — `characterToInput(c:
  Character): CharacterInput`, espelho 1:1 do backend `toCharacterInput`. Teste
  de paridade (`to-character-input.test.ts` + `__fixtures__/`) verde nos 16 chars.

## PRÓXIMO PASSO — WASM Inc.2 (a troca de shape na UI, NÃO iniciada)

Este é o refactor grande. Plano:

1. **Wire do engine no boot:** `ensureEngine()` no `__root.tsx` beforeLoad
   (paralelo ao `ensureCatalogs`). Wire o build do wasm no `predev`/`prebuild`
   do frontend (hoje o air cobre em dev; CI/build precisam do script).
2. **Trocar consumidores da UI:** hoje o sheet lê a saída de
   `entities/character/derived.ts` `characterEffects` (shape do FRONT:
   attributes/expertises/activeItems/breakdowns). O WASM devolve `ComputedSheet`
   (shape t20-data/Go: attributes/vitals/defense/skills/…). Os dois DIVERGEM —
   mapear ComputedSheet → o que `features/character-sheet/computed-sheet.tsx`
   (`ComputedSheetCards`) + cards precisam. **Faseado por consumidor**, com
   paridade visual (comparar contra `GET :id/sheet`, que já é server-computed).
3. **Otimismo:** `level-vitals.ts`/`draft-vitals.ts`/equip-preview/cast passam a
   `computeSheet(characterToInput(hipótese))` — instant, sem round-trip. O
   `collectVitalGrants` + `frontVitalResolver` (da B.3) provavelmente somem (o
   engine faz tudo).
4. **Remover/encolher** `derived.ts` (+ parte do `effect-source.ts`) — a
   duplicação TS que motivou o projeto. Ao fim, o front tem UM motor (Go/WASM).

Cuidados: `computeSheet` é síncrono só APÓS `ensureEngine()`; o mapper exige os
caches primados (loader gate). Testar paridade visual antes de remover o derive TS.

## Referências

- Memória: `project_front_decouple_catalog` (todo o decouple, fases, âncoras),
  `project_fase3_rtt_bench` (números da Fase 3, decisão WASM, plano Inc.2).
- Regras adaptadas: `frontend/CLAUDE.md` (FSD), `backend/CLAUDE.md` (DDD/TDD).
- Bench Go-vs-Node: `bench/` + `engine-go/cmd/server`.
