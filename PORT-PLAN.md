# PORT-PLAN — engine único em Go/WASM (Inc.2 = port completo)

> Plano vivo do "um motor só". Substitui o **Passo 2–4 do HANDOFF.md**, cuja
> premissa estava errada (ver §1). Escopo escolhido pelo dono: **port completo**
> — portar todo o motor real do front (`derived.ts` + `computeItemEffects` +
> `activeItemsFor`) pra Go, atingir paridade real, trocar a UI e remover a
> derivação TS. Multi-sessão, TDD com paridade card-a-card.

## 1. A descoberta que muda tudo: existem DOIS motores

O HANDOFF assumia que `derived.ts` era a duplicação TS que o WASM elimina, e que
`GET /:id/sheet` servia de oráculo de paridade. **Ambas as premissas são falsas.**

Há duas engines distintas no t20-data:

| | `computeCharacterSheet` (MVP) | `computeItemEffects` (o real) |
|---|---|---|
| Arquivo | `t20-data/src/character-sheet.ts` | `t20-data/src/items/engine.ts` (343 ln) |
| Quem usa | backend `/sheet`, **engine-go/WASM atual**, página `ComputedSheetCards` (view read-only) | **`derived.ts`** → ficha interativa inteira |
| Input | `ActiveEffect[]` já achatados | `ActiveItem[]` crus (montados de itens/raça/origem/poderes) |
| Alvos de modifier | 6: attribute/defense/attack/damage/save/skill | **~28**: expertise, expertiseAll, defenseDexCap, flySpeed, inventorySlots, pmLimit, spellDC, pmCost, maxPv/maxPm, tempHp, catalyst, resistance, flag, maneuver… |
| Produz | totais simples | breakdowns por-contribuição, condicionais togglados, RD, PV temp, flags |

**Prova dura:** o mapper `characterToInput` (front **e** backend) **não popula
`activeEffects`**. Logo o `computeSheet` do WASM — e o `/sheet` do servidor —
ignoram por completo modificadores de equipamento, passivos de poder (além de
vitals), mods não-atributo de raça, condicionais, etc. A "paridade byte-equal"
passou porque **WASM e backend são o mesmo MVP reduzido**. `/sheet` tem MENOS
dados que a ficha interativa já mostra hoje — não serve de oráculo.

Conclusão: o motor real e feature-complete é **front-only** (`derived.ts` +
`computeItemEffects`). O WASM atual porta o motor pequeno. "Um motor só" =
portar o motor real inteiro pra Go.

## 2. Arquitetura-alvo (decisão: 2b — port completo com catálogos no engine)

Duas camadas no motor real:

- **Coleta (`activeItemsFor`, ~400 ln em `derived.ts`)** — LÊ catálogos. Monta
  `ActiveItem[]` a partir do `Character` cru + caches. Codifica regras do livro:
  penalidade de não-proficiência (p142), mirror weapon attack, homebrew
  (equilibrada/vestido), mods de atributo de raça, deformidade, perda de Carisma
  da Tormenta, `ownedClassPowers`, `originModifiers`.
- **Resolução (`computeItemEffects` + breakdowns, ~343+ ln)** — CATALOG-FREE.
  Opera só em `ActiveItem[]` → `ItemEffects` (stacking por bonusType, flags,
  condicionais) e daí os `*Total` com contribuições.

**Fronteira do WASM (endgame):** o engine detém os catálogos (primados **uma
vez** a partir do MESMO JSON que o front já busca em `ensureCatalogs`, via uma
`primeEngineCatalogs(json)` em Go) e computa **tudo** a partir do `Character`
cru. O front passa a chamar `computeSheetV2(characterToInputV2(char), conditionals)`
e recebe um `ComputedSheetV2` rico com todos os breakdowns. `derived.ts` +
`items/engine.ts` saem do bundle.

Por que catálogos no engine e não `ActiveItem[]` como input (2a)? Porque a coleta
tem regras de verdade; deixá-la em TS não remove `derived.ts` (só a metade math).
O dono pediu remover `derived.ts` e a camada `activeItemsFor` — logo catálogos
entram no engine. Custo assumido: structs Go espelham as SHAPES dos catálogos
(não os dados — dados vêm do JSON buscado). Regra nova = editar tipo TS (autoria)
+ struct Go (consumo). Mitigar com um teste de paridade de shape.

## 3. Harness de paridade (o backbone do TDD)

Oráculo = saída ATUAL do `derived.ts` nos 16 seed chars. Fixtures já existem:
`frontend/src/entities/character/__fixtures__/character-input-parity.json`
(`{slug, char: Character, expected: CharacterInput}` × 16).

Plano do harness (`frontend`, vitest ou script node com caches primados):
para cada `char`, dumpar sob `engine-go/parity/<slug>.json`:
1. `activeItemsFor(char)` — **input** da resolução (exige exportar `activeItemsFor`).
2. `computeItemEffects(activeItems)` → `ItemEffects` normalizado (flags como array
   ordenado, byTarget como objeto) — oráculo da **resolução**.
3. Cada breakdown: `defenseTotal`, `expertiseTotalWithItems` (todas as perícias),
   `displacementTotal`, `flySpeedTotal`, `inventorySlotsTotal`, `attributeTotal`×6,
   `pmLimitTotal`, `bestBaseSpellCd`, `spellDCBonus`, `pmCostMod`,
   `characterDamageReduction`, `tempHpFromPowers`, `useAllConditionals`.

Testes Go leem `engine-go/parity/*.json` e comparam (semântico, key-order-independent,
como `parity_test.go` já faz). Fixtures são commitados; o harness re-gera quando a
regra TS muda (enquanto `derived.ts` ainda existir como referência).

**Slice 1 não precisa do harness**: os ~44 casos de `items/__tests__/engine*.test.ts`
são catalog-free e inline — portados pra table-tests Go dão verde imediato.

## 4. Ordem das slices (tasks #2–#8)

1. **Resolução core → Go** (task #2, EM ANDAMENTO). `items/engine.ts` inteiro:
   tipos (Modifier/ModifierTarget/BonusType/ModifierCondition/ActiveItem/
   ItemEffects/Contribution/AggregatedStat/ConditionalEffect) + `targetKey`,
   `resolveStack`, `computeItemEffects`, `statFor`, `conditionalId`,
   `applyActiveConditionals`, `resolveConditionalDisplay`. Testes = port de
   engine*.test.ts. **Catalog-free, autocontido — começa já.**
2. **Harness de paridade** (task #3). Destrava #4/#5.
3. **Coleta + catálogos → Go** (task #4, ✅ DONE). `ActiveItemsFor` +
   `PrimeEngineCatalogs` portados (catalogs.go/character.go/collect*.go). Bate
   `activeItems` byte-equal contra o oráculo nos 16 seeds
   (`collect_parity_test.go`). Harness dumpa `char` + `_catalogs.json`.
4. **Breakdowns → Go** (task #5, ✅ DONE). Todos os `*Total` + RD/tempHp num
   `ComputedSheetV2` (`sheet_rules.go`/`breakdowns.go`/`breakdowns_magic.go`).
   `(*Catalogs).ComputeSheetV2(char, conditionals)` bate byte-equal contra o
   oráculo `sheetV2` nos 16 seeds (`sheetv2_parity_test.go`).
5. **Fronteira WASM + boot/build** (task #6). Expõe `computeSheetV2`; prima
   catálogos; `ensureEngine()` no `__root.tsx` beforeLoad; build do wasm no
   `predev`/`prebuild` do frontend (hoje só o air cobre — CI/build precisam).
6. **Troca da UI** (task #7). Substitui os 26 consumidores + 3 hooks por hooks
   engine-backed, faseado, com paridade visual. Inclui otimismo (level/draft).
7. **Remoção** (task #8). Deleta `derived.ts` (breakdowns + coleta) e o import de
   `items/engine.ts` do bundle. Verde em FE/BE/t20-data + typecheck + biome +
   bundle sem regressão.

## 5. Consumidores de `derived.ts` (mapa — o que a troca precisa cobrir)

26 importadores reais. Contratos principais a preservar: os 3 hooks
(`useCharacterEffects`, `useAllConditionals`, `useFuriaActive`) leem o store
Zustand de condicionais → precisam de wrapper hook-shaped em volta da chamada
do engine; e os retornos `{total, contributions[]}` das funções de breakdown.

Consumidor mais pesado: `combat-magic-stats.tsx` (DEF, CD, RD, limites PM, save
DCs). Outros: `sheet-header` (movimento + `optimisticLevelVitals`), `cast-spell-dialog`
(gate de PM), `stances-section`/`use-power-dialog` (posturas + tempHp),
`expertise-row`/`expertises-panel`, `bag-panel` (slots + proficiência),
`effects-panel`/`effects-count-badge` (condicionais), `sheet-search-index`,
`mobile-def-chip`, `character-select/*`, `session-tracker/initiative-roll`.

Otimismo (não depende do `/sheet`): `entities/character/level-vitals.ts`
(`optimisticLevelVitals`) e `features/character-build/draft-vitals.ts`
(`deriveDraftVitals`) — ambos já rodam `collectVitalGrants` + `frontVitalResolver`
em TS. Passam a `computeSheetV2(hipótese)` — instant, sem round-trip; o
`frontVitalResolver` provavelmente some.

## 6. Estado atual do engine-go (ponto de partida)

- `engine-go/` módulo `t20engine` (go 1.26). `engine/` porta só o MVP
  `computeCharacterSheet` (attributes/vitals/defense/saves/skills/attacks/
  deslocamento/buffs). `cmd/wasm` (browser) + `cmd/server` (:3002).
- `bench/payloads/*` (16 CharacterInput) + `bench/expected/*` (oráculo MVP).
  `engine/parity_test.go` compara semântico. **Esse oráculo é do MVP — não
  cobre o motor real; NÃO reusar como oráculo das slices novas.**
- Front: `shared/lib/engine-wasm.ts` (`ensureEngine`/`computeSheet`, DORMENTE),
  `entities/character/to-character-input.ts` (mapper, paridade byte-equal).

## 7. Riscos / cuidados

- **Duplicação de shape de catálogo** (TS autoria ↔ Go consumo). Teste de shape.
- **`computeSheetV2` é síncrono só após `ensureEngine()`** + catálogos primados.
  Loader gate obrigatório antes de qualquer consumidor renderizar.
- **Paridade visual antes de deletar** o derive TS (task #8 só depois de #7 ok).
- **Não regredir** a ficha: o `/sheet`/`ComputedSheetCards` (view MVP) continua
  válido e separado; a troca é na ficha interativa (`character-sheet*.tsx`).
