# CONTINUE — comece por aqui (sessão nova)

> Ponto único de entrada depois de um `/clear`. Aponte a sessão nova para este
> arquivo. Detalhe completo em **`PORT-PLAN.md`**.

## Em uma frase

Migração **"1 motor só"**: portar o motor real de derivação de ficha
(`frontend/.../derived.ts` + `t20-data/src/items/engine.ts`) para **Go/WASM**,
com paridade TDD contra os 16 personagens semente, e no fim remover a derivação
TS. Escopo escolhido pelo dono: **port completo** (não o MVP).

## Estado atual (tasks #2–#8 — DONE; "1 motor só" em produção)

- **Slice 1** — resolução core em Go: `engine-go/engine/itemeffects.go`
  (port fiel de `items/engine.ts`, catalog-free) + `itemeffects_test.go` (~44 casos).
- **Slice 3** — harness TDD: `activeItemsFor` exportado;
  `frontend/src/entities/character/parity-oracle.test.ts` gera 16 oráculos em
  `engine-go/parity/*.json`; `engine-go/engine/itemeffects_parity_test.go` prova
  `ComputeItemEffects` **byte-equal ao TS nos 16 seed chars**.
- **Slice 2 (task #4) — coleta + catálogos → Go**: portado `activeItemsFor`
  inteiro + `PrimeEngineCatalogs`. Arquivos Go: `catalogs.go` (shapes +
  store + lookups), `character.go` (Character cru), `collect_rules.go` (raceMods/
  originMods/resolveAtributoDeltas/requiredProficiency/ownership), `collect_parse.go`
  (parsers das colunas JSON), `collect_entities.go` (raça/origem/classe/tormenta),
  `collect.go` (`(*Catalogs).ActiveItemsFor` + itens/overlays/penalidades/homebrew).
  Harness dumpa `char` em cada oráculo + `engine-go/parity/_catalogs.json`;
  `collect_parity_test.go` prova `ActiveItemsFor` **byte-equal ao TS nos 16 seeds**.
- **Task #5 — breakdowns → Go [NOVO]**: portados TODOS os `*Total` de `derived.ts`
  num único `ComputedSheetV2` (defense/displacement/flySpeed/inventorySlots/
  attribute×6/pmLimit/bestBaseSpellCd/spellDCBonus/pmCostMod/RD/tempHp/expertise-
  por-perícia). Arquivos Go: `sheet_rules.go` (trainingBonusForLevel/spellSaveDc/
  classSpellcastingAttribute/barbaroRd/guerreiroRd — puros de t20-data),
  `breakdowns.go` (structs + `(*Catalogs).ComputeSheetV2` + defense/movimento/
  attribute/expertise), `breakdowns_magic.go` (pm/spell/RD/tempHp). Harness dumpa
  `sheetV2` por seed (chama as funções reais do derived.ts; `tempHpFuria` usa
  furiaActive=true p/ exercitar Alma de Bronze); `sheetv2_parity_test.go` prova
  `ComputeSheetV2` **byte-equal ao TS nos 16 seeds**.
- **Task #6 — fronteira WASM + boot/build [NOVO]**: `cmd/wasm/main.go` expõe
  `primeEngineCatalogs` + `computeSheetV2` (além do MVP `computeCharacterSheet`).
  Front: `engine-wasm.ts` ganhou `primeEngineCatalogs`/`computeSheetV2` +
  `shared/lib/computed-sheet-v2.ts` (tipo `ComputedSheetV2` espelha o struct Go).
  Boot: `ensureEngineCatalogs(qc)` em `ensure-catalogs.ts` (warma o wasm + prima
  com o MESMO JSON de `ensureCatalogs`) chamado no `__root.tsx` beforeLoad em
  paralelo — **best-effort** (try/catch; NÃO load-bearing até #7). Build: `predev`/
  `prebuild` do frontend rodam `build-engine-wasm.sh`; CI ganhou `setup-go` +
  step `go test ./...`. **Prova E2E**: harness node (build real do wasm → prime →
  `computeSheetV2(char)`) bate byte-equal ao oráculo `sheetV2` nos 16 seeds.
- **Task #7 — troca da UI via CHOKE-POINT [NOVO]** (estratégia escolhida pelo dono
  — ver AskUserQuestion): em vez de reescrever os 26 consumidores, roteou-se o
  motor pesado num ÚNICO ponto. `cmd/wasm` expõe `computeEffects(char, conds)`
  (ItemEffects resolvido); `engine-wasm.ts` ganhou `computeEffects` (rebuild do
  flags array→Set). Em `derived.ts`, `resolveEffects()` chama o engine quando
  `areEngineCatalogsPrimed()`, senão cai no derive TS (fallback p/ testes/erro).
  `characterEffects`/`useCharacterEffects`/`useAllConditionals`/`useFuriaActive`
  passam por ele → **os 23 consumidores + os breakdowns rodam sobre effects do Go,
  ZERO mudança de call-site, paridade garantida**. Prova E2E node:
  `computeEffects(char)` byte-equal ao oráculo `itemEffects` nos 16 seeds. Os
  breakdowns continuam TS (puros sobre effects).
- **Task #8 — remoção / engine obrigatório [NOVO]**: `resolveEffects` agora é
  gated por `import.meta.env.MODE === 'test'`. Em produção/dev SÓ o engine roda
  (throw se não primado); o ramo TS (coleta `activeItemsFor` inteira +
  `computeItemEffects` + `applyActiveConditionals`) é TEST-ONLY → o build estático
  faz **dead-code-elimination** dele. **Provado**: os 4 strings exclusivos da
  coleta ("sem proficiência: -5…", "Homebrew: esotérico vestido", "poder(es) da
  Tormenta (p136)", "anula desbalanceada") estão AUSENTES de `dist/assets` (os
  breakdowns "Penalidade de armadura"/"cannot-apply-dex" continuam). `ensure-
  catalogs`/`__root` atualizados (engine load-bearing; falha vira erro só nas
  páginas de ficha). A coleta TS fica no repo como ORÁCULO de teste (o harness
  `parity-oracle.test.ts` ainda regenera; os testes vitest usam o derive TS, sem
  wasm). FE 585 verde, build verde, DCE provado.

## Estado: migração "1 motor só" COMPLETA em produção

O derive da ficha roda 100% no motor Go/WASM em produção; o bundle não carrega
mais a coleta TS. Restam APENAS caudas opcionais (não bloqueiam nada):
- `effect-source.ts equippedItemFlagEffects` ainda usa `computeItemEffects`
  por-item (lista de flags por item no tab Efeitos, com proveniência) → o engine
  agrega flags e perde a proveniência por-item, então migrar exigiria uma função
  nova no engine. `computeItemEffects` (resolução pura, ~100 ln) segue no bundle
  só por isso. Baixa prioridade.
- Otimismo `optimisticLevelVitals`/`deriveDraftVitals` usa `collectVitalGrants`+
  `frontVitalResolver` (caminho de vitals, NÃO este derive) — pode virar
  `computeSheetV2(hipótese)` um dia. Independente.
- Breakdowns `*Total` seguem TS finos sobre os effects do Go (decisão do #7).
  Migrar p/ `ComputedSheetV2` removeria `statFor` do bundle — ganho marginal.
- Deletar de vez a coleta TS (e o oráculo) exigiria carregar wasm no vitest
  (padrão em `wasm-effects-proof.cjs`); hoje NÃO vale (perde a rede de paridade).

## Verificação (rodar antes/depois de cada slice)

```bash
cd engine-go && go test ./... && gofmt -l engine/        # Go: verde + formatado
pnpm --filter frontend test                              # FE: 584 verdes
pnpm --filter frontend typecheck                         # tsgo limpo
pnpm exec biome lint frontend/ engine-go/ 2>/dev/null    # (biome cobre FE .ts/.tsx)
# regenerar oráculos quando a regra TS mudar:
GEN_ORACLE=1 pnpm --filter frontend test parity-oracle
```

## Cuidados (não repetir descobertas)

- Existem **DOIS motores**: MVP (`computeCharacterSheet`, o que o WASM porta hoje)
  vs. o real (`computeItemEffects`, ~28 alvos, breakdowns/condicionais/RD/tempHp).
  `characterToInput` NÃO popula `activeEffects` → `GET :id/sheet` NÃO é oráculo do
  motor real (tem menos dados que a UI). Ver `PORT-PLAN.md §1`.
- `computeSheetV2` só é síncrono após `ensureEngine()` + catálogos primados
  (loader gate). Paridade **visual** antes de deletar o derive TS (slice 8 só após #7).
- **Do port da coleta (task #4):** dois `resolveAtributoMod` coexistem — o do MVP
  (`races.go`, mapa hardcoded, `map[string]int`) e o novo `resolveAtributoDeltas`
  (`collect_rules.go`, catalog-driven, `[]attrDelta` ORDENADO). A ordem importa:
  `raceAttributeMods` emite 1 modifier por entrada na ORDEM da chave do objeto TS
  (`Object.entries`) → `atributoMod.mods/variants` usam `orderedInts` (decoder que
  preserva ordem; mapa Go embaralha). Idem: `parseChoiceSet` devolve `orderedSet`
  (lista+set) porque `generalPowerActiveItem`/`originPickedPowerIds` iteram na ordem
  de inserção. `Modifier.UnmarshalJSON` arredonda amount fracionário (só
  `botas-reforcadas` +1.5m, não equipada por nenhum seed) — engine é int-modelada.
  `effectSourceName` no Go cobre item + manual-temp-hp (spell/activation catalogs
  não primados; nenhum seed precisa). Regenerar oráculo+catálogos: comando abaixo.
