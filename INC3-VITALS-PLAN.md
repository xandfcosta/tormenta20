# INC3-VITALS-PLAN — motor de vitals (PV/PM) único em Go/WASM

> Continuação da Inc.2 (derive de item-effects, COMPLETA). Agora: portar o
> **motor de vitals** (PV/PM máximos) do front pro Go, tirando a duplicação TS
> (`frontVitalResolver` + `collectVitalGrants` + `multiclassPvPool/MpPool`).
> Mesmo método da Inc.2: oráculo de paridade nos 16 seeds, choke-point,
> MODE-gate p/ DCE do bundle.

## O que são os vitals
PV/PM MÁXIMOS = `multiclassPvPool(classes, con) + collectVitalGrants(...).pv`
(e idem PM). `collectVitalGrants` soma os modifiers `maxPv`/`maxPm` que o
personagem possui (raça, poderes de classe, poderes gerais, poder concedido/
deus, origem), avaliando o `scale` (flat/level/levelStep/attribute) com a regra
"atributo entra uma vez só" (p225).

## Duplicação hoje
- **Front TS**: `t20-data collectVitalGrants` (catalog-driven via resolver) +
  `frontVitalResolver` (lê caches) + `multiclassPvPool/MpPool` + `CLASS_VITALS`.
  Consumidores: `optimisticLevelVitals` (level-vitals.ts, otimismo de level-up)
  e `deriveDraftVitals` (draft-vitals.ts, preview do wizard).
- **Go MVP** (`vitals.go`): pools + `collectVitalGrants` com TABELA HARDCODED
  (~19 entradas maxPv/maxPm). Catalog-FREE (serve o /sheet do backend). NÃO
  reusar aqui: usa `racas[slug]` (front passa NOME) + dados hardcoded.

## Alvo: vitals catalog-driven no engine Inc.2
Reusa a coleta da Inc.2 (raceModifiers/ownedClassPowers/generalPower/
originModifiers via catálogos primados) + o pool math do vitals.go. Adiciona só
o catálogo de **granted powers** (poder concedido, p/ `godPower` → Bênção do
Mana). Recebe os `attrTotals` REAIS do front (com mods de item), não os
reduzidos do MVP.

## Slices — TODAS ✅ DONE (2026-08-06)
1. **Oráculo** ✅: harness dumpa `vitals {pvMax,pmMax}` por seed (via
   `buildVitalContext`+`computeVitalPools`) + `grantedPowers` no `_catalogs.json`.
2. **Go** ✅: `GrantedPower` no Catalogs + `vitals_v2.go` `(*Catalogs).ComputeVitals`
   catalog-driven (coleta maxPv/maxPm de raça/classe/geral/deus/origem via lookups
   da Inc.2 + evalModifierScale + dedupe de atributo + pools do vitals.go).
   `vitals_v2_parity_test.go` byte-equal nos 16 seeds (inclui Anão/Elfo + Bênção
   do Mana). Adicionado `GodPower` ao `Character` Go.
3. **WASM + front** ✅: `computeVitals(ctx)` no cmd/wasm + `engine-wasm.ts`
   (`VitalContext`/`VitalPools`). Choke point `entities/character/vital-pools.ts`
   `computeVitalPools(ctx)` MODE-gated. `level-vitals.buildVitalContext` +
   `draft-vitals` chamam o choke point; `frontVitalResolver` só no ramo test.
   **DCE provado**: `godPowerModifiers` (método do frontVitalResolver) AUSENTE de
   `dist/assets`; `computeVitals` PRESENTE. E2E node (`wasm-vitals-proof.cjs`):
   computeVitals byte-equal ao oráculo 16/16. FE 585 verde, build/typecheck/biome
   limpos, Go verde.

## Estado: Inc.3 COMPLETA — vitals rodam 100% no Go/WASM em produção
Removida a duplicação TS de vitals do bundle (frontVitalResolver +
collectVitalGrants + pools). `vital-resolver.ts` fica como oráculo de teste
(DCE em prod). O MVP `vitals.go` (hardcoded, catalog-free) segue servindo o
`/sheet` do backend — engine separado do front, fora de escopo.

## Verificação (rodar por slice)
```
cd engine-go && go test ./... && gofmt -l engine/
pnpm --filter frontend test && pnpm --filter frontend typecheck
GEN_ORACLE=1 pnpm --filter frontend test parity-oracle   # regenera oráculo
node "$CLAUDE_JOB_DIR/tmp/wasm-effects-proof.cjs"          # E2E do wasm
```
