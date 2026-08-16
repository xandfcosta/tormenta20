# Motor Go

Adapta o [CLAUDE.md da raiz](../CLAUDE.md) a este pacote. As regras da raiz
valem; o que está aqui estende ou sobrepõe.

`engine-go` é o backend: API HTTP na :3001, o motor de regras, e o mesmo motor
compilado para WASM que roda no navegador. Um processo serve a SPA, a API e o
socket em produção (`STATIC_DIR`).

## Ambientes

`LoadConfig` lê `.env.<APP_ENV>` deste diretório antes do env do processo —
**o processo vence o arquivo**, sempre (ALE-119). O `.env.development` é
versionado porque nada nele é segredo; o `.env.production` é do dono da mesa e
não entra no git.

Configuração nova entra em `api/config.go` **e** nos dois arquivos `.env` — um
default que só existe no Go é um default que ninguém descobre. Se a variável
puder derrubar produção em silêncio (chave de assinatura, origem liberada),
ela também entra em `Config.Validate`, que roda antes de o servidor escutar.

## Regenerar oráculo é ato deliberado

Os JSONs em `parity/` são a rede de regressão da ficha inteira e o teste mais
valioso do repositório: um único `lenda-nv20-maximo.json` fixa `pvMax 277`, a
carta do Machado com ataque 21 e o breakdown, as 29 perícias com composição.

```
cd engine-go && go run ./cmd/genoracle
```

> **O diff de um oráculo é revisado contra o LIVRO, nunca aceito porque "o teste
> ficou verde".**

**Esse momento já chegou.** O `t20-data` foi aposentado, então não há segunda
implementação: o oráculo é o Go descrevendo o Go, e **um bug no motor vira a nova
verdade em silêncio**. A mitigação não é técnica, é de processo, e é a linha
acima.

O que o oráculo ainda protege é enorme — a ficha inteira de 18 personagens, ponta
a ponta, acusando qualquer número que mude sem ter sido pedido. O que ele deixou
de provar é que DOIS motores concordam.

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

## O sqlc trunca SQL por causa de comentário acentuado

Comentário em `db/query.sql` é **ASCII**. O sqlc mede a query em bytes e conta o
comentário em runas, então cada letra acentuada acima de uma query corta um
caractere do SQL gerado — **em silêncio**. Na ALE-120 um comentário com três
acentos gerou `WHERE id = ? AND usedAt IS N`, que ainda compilava.

## O sqlc não enxerga `ALTER TABLE ADD COLUMN` de outro arquivo

Medido no sqlc **v1.31.1** (ALE-124): uma coluna adicionada por `ALTER TABLE …
ADD COLUMN` num arquivo de migração NOVO não entra no catálogo — toda query que
a cita falha com `column "x" does not exist`, e o erro aponta a *query*, não a
migração. Dentro do MESMO arquivo (`CREATE TABLE` + `ALTER` juntos) funciona, e
as colunas assim adicionadas na `00002` continuam valendo porque… continuam
valendo: regenerar não as perde. Não perca tempo com a forma do comentário nem
com a seção `Down` — foi tudo testado.

Saída prática: **tabela 1:1 em vez de coluna nova**. Foi o que a `00005` fez com
o tabuleiro (`session_boards`), e de quebra "sessão sem tabuleiro" passou a ser
dito pelo schema (a linha existe ou não existe) em vez de por convenção sobre um
JSON vazio.

## Catálogos

`catalog/data/*.json` é embutido no binário e servido por `GET /catalog/:nome`.
**Este é o único lugar onde catálogo é autorado** — mudar uma magia é editar um
arquivo só. O front os busca por HTTP e o `test-setup` do vitest lê os mesmos
arquivos, então uma edição vale para os dois lados na hora.

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
