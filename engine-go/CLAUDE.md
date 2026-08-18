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

## A SPA sai PRÉ-COMPRIMIDA do build

O `net/http` não comprime nada sozinho, e por isso o servidor mandava os 3,7 MB
crus do `t20.wasm` para um navegador que pedia `gzip, br` — 4,9 MB de carga fria
que comprimidos são 1,1 MB (ALE-153).

Quem comprime é o BUILD (`frontend/scripts/precompress-dist.sh`, rodado pelo
`postbuild`), gerando `.br` e `.gz` ao lado de cada asset; o `spaHandler` só
escolhe a variante pelo `Accept-Encoding`. Comprimir na requisição seria gastar
CPU da máquina do mestre a cada jogador que entra — brotli -q11 custa ~8s UMA vez
no build.

Duas armadilhas que os testes de `cmd/api` congelam: o `Content-Type` tem de sair
do nome ORIGINAL (adivinhado pela extensão do irmão, o wasm vira
`application/octet-stream` e o `instantiateStreaming` recusa), e `gzip;q=0` é uma
RECUSA — um `strings.Contains` a leria como aceitação.

Sem os irmãos comprimidos o app continua funcionando, só mais pesado: ausência é
caminho normal, não erro.

## O boot confere o SCHEMA, não o `goose_db_version`

A migração pode CONSTAR aplicada sem a tabela existir. Aconteceu: a
`session_boards` sumiu do banco de desenvolvimento com a 00005 marcada, o goose
disse "no migrations to run", e o tabuleiro passou um dia vivendo só em memória
— cada gravação falhando numa linha de log que ninguém lê (ALE-154).

Por isso o `db.Open` roda `assertSchema` DEPOIS de migrar e **recusa subir**
nomeando as tabelas que faltam. A lista de esperadas é lida das próprias
migrações embutidas, nunca escrita à mão: lista à mão envelhece em silêncio, que
é como este repositório já perdeu o `TurnsTaken` e o `creatureId` no mesmo dia.

Consequência a saber: um banco alterado por fora (um `goose down` parcial, um
backup anterior restaurado) agora **não sobe**. É deliberado — gravar no vazio
em silêncio é pior.

## Transação pega a trava no BEGIN (`_txlock=immediate`)

A conexão abre com `_txlock=immediate`, e não com o `DEFERRED` padrão (ALE-156).

O motivo: travas de unicidade decididas no CÓDIGO (pergunta ao banco, depois
escreve) não sobrevivem a dois pedidos simultâneos, porque as duas perguntas
acontecem antes de qualquer escrita. Quem torna o resultado **correto** é
refazer a checagem DENTRO da transação; o `immediate` é o que o torna
**honesto** — medido, sem ele um dos perdedores recebe 500 (o SQLite recusa a
escrita sobre snapshot mudado, o que está certo, mas chega ao jogador como erro
do servidor), e com ele o perdedor espera, relê e recebe o 409 que descreve o
que houve.

Custo: os escritores serializam entre si. As oito transações do app são todas de
escrita e leitura fora de transação continua livre (WAL), então numa mesa
doméstica isso é de graça; o `busy_timeout(5000)` cobre a espera.

## Timeouts, encerramento e backup

O servidor tem `ReadHeaderTimeout` (5s) e `IdleTimeout` (120s), e **não tem
`WriteTimeout` de propósito** (ALE-157): ele mataria o socket.io, que é conexão
longa por natureza, e o download do wasm numa rede ruim. É o timeout que parece
obrigatório e é justamente o errado aqui — há teste afirmando a AUSÊNCIA dele.

Um sinal encerra com ordem (`signal.NotifyContext` + `Shutdown`, janela de 10s).
Antes, um Ctrl-C no meio de um `VACUUM INTO` morria no meio e o
`defer database.Close()` nunca rodava — defer não roda quando o processo morre
por sinal.

Corpo de requisição tem teto de 1 MB no `decodeJSON`, com **413 próprio**: dizer
"JSON inválido" para um JSON válido manda procurar defeito de sintaxe onde o
problema é tamanho.

O backup automático (`BACKUP_EVERY`, `BACKUP_KEEP`) usa o mesmo `VACUUM INTO` do
manual e poda os mais antigos. Zero em qualquer um dos dois desliga. A poda só
alcança o que a listagem reconhece como backup — arquivo estranho na pasta não é
candidato.

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
