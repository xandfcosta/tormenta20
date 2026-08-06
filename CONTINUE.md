# CONTINUE — comece por aqui (sessão nova)

> Ponto único de entrada depois de um `/clear`. Aponte a sessão nova para este
> arquivo. Detalhe completo em **`PORT-PLAN.md`**.

## Em uma frase

Migração **"1 motor só"**: portar o motor real de derivação de ficha
(`frontend/.../derived.ts` + `t20-data/src/items/engine.ts`) para **Go/WASM**,
com paridade TDD contra os 16 personagens semente, e no fim remover a derivação
TS. Escopo escolhido pelo dono: **port completo** (não o MVP).

## Estado atual (slices 1–3 + coleta/catálogos — DONE, verde e verificado)

- **Slice 1** — resolução core em Go: `engine-go/engine/itemeffects.go`
  (port fiel de `items/engine.ts`, catalog-free) + `itemeffects_test.go` (~44 casos).
- **Slice 3** — harness TDD: `activeItemsFor` exportado;
  `frontend/src/entities/character/parity-oracle.test.ts` gera 16 oráculos em
  `engine-go/parity/*.json`; `engine-go/engine/itemeffects_parity_test.go` prova
  `ComputeItemEffects` **byte-equal ao TS nos 16 seed chars**.
- **Slice 2 (task #4) — coleta + catálogos → Go [NOVO]**: portado `activeItemsFor`
  inteiro + `PrimeEngineCatalogs`. Arquivos Go novos: `catalogs.go` (shapes +
  store + lookups), `character.go` (Character cru), `collect_rules.go` (raceMods/
  originMods/resolveAtributoDeltas/requiredProficiency/ownership), `collect_parse.go`
  (parsers das colunas JSON), `collect_entities.go` (raça/origem/classe/tormenta),
  `collect.go` (`(*Catalogs).ActiveItemsFor` + itens/overlays/penalidades/homebrew).
  Harness agora dumpa `char` em cada oráculo + `engine-go/parity/_catalogs.json`;
  `collect_parity_test.go` prova `ActiveItemsFor` **byte-equal ao TS nos 16 seeds**.
  Detalhes/cuidados do port abaixo em "Cuidados".
- Nada removido ainda — `derived.ts` intacto. Sem regressão.

## Próximo passo — task #5: breakdowns → Go

Portar os `*Total` de `derived.ts` (defenseTotal, expertiseTotalWithItems,
displacementTotal, flySpeedTotal, inventorySlotsTotal, attributeTotal×6,
pmLimitTotal, bestBaseSpellCd, spellDCBonus, pmCostMod, characterDamageReduction,
tempHpFromPowers) + RD/tempHp, emitindo um `ComputedSheetV2`. Oráculo já existe
no harness (PORT-PLAN.md §3 lista os breakdowns; hoje o oráculo só dumpa
`activeItems`+`itemEffects` — estender `parity-oracle.test.ts` p/ dumpar os
breakdowns esperados). Depois: #6 fronteira WASM + boot/build, #7 troca da UI
(paridade visual), #8 remoção. Ordem/detalhe: `PORT-PLAN.md §4`. Consumidores da
UI a cobrir: `PORT-PLAN.md §5`.

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
