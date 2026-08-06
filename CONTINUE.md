# CONTINUE — comece por aqui (sessão nova)

> Ponto único de entrada depois de um `/clear`. Aponte a sessão nova para este
> arquivo. Detalhe completo em **`PORT-PLAN.md`**.

## Em uma frase

Migração **"1 motor só"**: portar o motor real de derivação de ficha
(`frontend/.../derived.ts` + `t20-data/src/items/engine.ts`) para **Go/WASM**,
com paridade TDD contra os 16 personagens semente, e no fim remover a derivação
TS. Escopo escolhido pelo dono: **port completo** (não o MVP).

## Estado atual (slices 1–3 de 8 — DONE, verde e verificado)

- **Slice 1** — resolução core em Go: `engine-go/engine/itemeffects.go`
  (port fiel de `items/engine.ts`, catalog-free) + `itemeffects_test.go` (~44 casos).
- **Slice 3** — harness TDD: `activeItemsFor` exportado;
  `frontend/src/entities/character/parity-oracle.test.ts` gera 16 oráculos em
  `engine-go/parity/*.json`; `engine-go/engine/itemeffects_parity_test.go` prova
  `ComputeItemEffects` **byte-equal ao TS nos 16 seed chars**.
- Nada removido ainda — `derived.ts` intacto. Sem regressão.

## Próximo passo — Slice 2 (task #4): coleta + catálogos → Go

Portar `activeItemsFor` (a camada que LÊ catálogos) para Go + primar o engine com
o JSON de catálogo já buscado (`primeEngineCatalogs`). **Alvo já existe**: bater os
`activeItems` gerados em Go contra `engine-go/parity/*.json`. Depois: #5 breakdowns,
#6 fronteira WASM + boot/build, #7 troca da UI (paridade visual), #8 remoção.
Ordem e detalhe fino: `PORT-PLAN.md §4`. Consumidores da UI a cobrir: `PORT-PLAN.md §5`.

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
