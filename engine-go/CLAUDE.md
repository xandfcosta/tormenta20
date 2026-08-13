# Motor Go

Adapta o [CLAUDE.md da raiz](../CLAUDE.md) a este pacote. As regras da raiz
valem; o que está aqui estende ou sobrepõe.

`engine-go` é o backend: API HTTP na :3001, o motor de regras, e o mesmo motor
compilado para WASM que roda no navegador. Um processo serve a SPA, a API e o
socket em produção (`STATIC_DIR`).

## Regenerar oráculo é ato deliberado

Os JSONs em `parity/` são a rede de regressão da ficha inteira e o teste mais
valioso do repositório: um único `lenda-nv20-maximo.json` fixa `pvMax 277`, a
carta do Machado com ataque 21 e o breakdown, as 29 perícias com composição.

```
GEN_ORACLE=1 pnpm --filter frontend test parity-oracle
```

> **O diff de um oráculo é revisado contra o LIVRO, nunca aceito porque "o teste
> ficou verde".**

Enquanto o `t20-data` existe, o oráculo tem duas testemunhas: ele é gerado pela
implementação TS de referência (`tsCharacterEffects`, `tsVitalPools`,
`tsEquippedItemFlags` — chamadas EXPLICITAMENTE pelo harness, nunca por `if` de
ambiente) e conferido contra o Go. Quando o TS morrer, o oráculo passa a ser o Go
descrevendo o Go, e **um bug no motor vira a nova verdade em silêncio**. A
mitigação não é técnica, é de processo, e é a linha acima.

Aprendido do jeito difícil: a paridade entre os dois motores esteve **perfeita**
durante meses enquanto AMBOS erravam a RD do Guerreiro, que não existe no livro
(ALE-111). Paridade prova concordância; nunca prova correção.

## Regras vêm do livro, com página

Todo teste de regra cita a página do [Tormenta 20](/t20-book.pdf) e, quando
existe, o **exemplo trabalhado** do próprio livro — a Samira da p173, a Bola de
Fogo do arcanista de 11º nível na p171, o clérigo/druida da p226. Um exemplo
trabalhado é melhor que uma asserção inventada: ele separa leituras possíveis da
mesma frase.

**Confira a citação antes de escrever.** O offset é `página do PDF = página do
livro + 6`. Onze citações erradas já foram corrigidas neste repositório, três
delas escritas no mesmo dia em que eu as "verifiquei".

## O gerador de tipos da fronteira

```
go generate ./engine          # escreve frontend/src/shared/api/engine-types.ts
```

O arquivo gerado é commitado e `TestGeneratedTypesAreCurrent` falha apontando a
primeira linha divergente se alguém mudar uma struct sem regenerar.

Gera-se **só a fronteira** (o que o WASM devolve e recebe), não os catálogos: o
Go serve catálogo como bytes crus e não tem struct para a maioria deles; as três
que existem são subconjuntos deliberados. Ver ALE-108.

Um tipo com `MarshalJSON` próprio **precisa** declarar sua forma de fio em
`tsWireOverrides` — o emissor recusa com panic caso contrário, porque refletir a
struct em memória produziria um tipo que mente (o `ItemEffects` guarda flags num
Set e serializa um array).

## Catálogos

`catalog/data/*.json` é embutido no binário e servido por `GET /catalog/:nome`.
Quatro tabelas são **autoradas aqui** (perícias por classe, termos de devoto,
tabelas do Improviso, desenho de masmorra); o resto ainda é despejado do
`t20-data`.

O que protege dado transcrito é **validação de schema**
(`catalog/rules_tables_test.go`), não um `expect` por campo: o risco é typo, não
regressão. O que ela cobre é o que quebra tela — perícia que não existe, faixa de
rolagem com buraco, termo de devoto apontando para raça inexistente.

## Testes

- `go test ./...` — sem flag, sem setup.
- Teste de regra vive junto da regra e cita a página; teste de paridade prova que
  os dois motores concordam. **São coisas diferentes e nenhum substitui o outro.**
- Correção de bug nasce **vermelha**. Sabotar a implementação depois de escrever
  o teste é a forma barata de provar que ele mede o que diz medir — foi assim que
  se descobriu que um teste de PV passava por acidente, porque em todos os casos
  a primeira classe também era a maior.
