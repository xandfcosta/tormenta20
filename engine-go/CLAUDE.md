# Motor Go

Adapta o [CLAUDE.md da raiz](../CLAUDE.md) a este pacote. As regras da raiz
valem; o que está aqui estende ou sobrepõe.

`engine-go` é o app inteiro: a API HTTP na :3001, o motor de regras, e as CENAS
em `.templ` servidas com Datastar — mais a folha e as ilhas de JS delas, em
`serve/web/assets/src`, e o kit de apresentação em `serve/web/ui`. Um processo serve tudo, e
desde a ALE-273 ele também sobe por `docker compose up -d --build`, com o banco
em bind mount. **O compose não trouxe um segundo runtime**: continua sendo UM
serviço. O proxy que normalmente viria junto foi considerado e recusado — ele
compraria só a compressão, que mora em `httpio.Gzip`, e traria um segundo lugar
onde o SSE pode ser bufferizado por engano.

## O mapa das pastas

Cinco grupos, e a pergunta que cada um responde:

```
engine-go/
├── cmd/          os binários: o servidor, a seed, o gerador do oráculo
├── domain/       O QUE O JOGO É — nada aqui sabe de HTTP
│   ├── engine/   as regras do livro, PURAS (só a stdlib)
│   ├── catalog/  o livro transcrito, embutido por go:embed
│   ├── book/     o catálogo tipado, lido por treze famílias
│   ├── sheet/  board/  live/   o domínio COM estado
│   └── campaign/ account/ creature/ search/ markdown/
├── app/          O QUE UM GESTO FAZ, do pedido à gravação
│   ├── session/  o ciclo da sessão, a trava de acesso e o STORE da fila
│   ├── boards/   o store dos tabuleiros abertos, com as abas e os lugares
│   ├── initiative/ quem entra na fila, e com que números
│   ├── character/ o herói: nascer (`Births`) e jogar (`Plays`)
│   └── rest/     o que expira e o que recupera quando a cena ou o dia acaba
├── serve/        O QUE RESPONDE HTTP
│   ├── api/      a RAIZ DE COMPOSIÇÃO: monta o roteador e cumpre as portas
│   └── web/      as quinze cenas, cada uma com a porta dela
│       └── assets/  o FRONT que não é `.templ`: a folha, as ilhas de JS,
│                    as fontes e o favicon — e o `go:embed` deles
├── infra/        O QUE NÃO É DOMÍNIO
│   ├── db/       migrações e as consultas do sqlc
│   ├── config/   o que o ambiente diz, lido no boot
│   ├── httpio/   a borda HTTP: responder, ler corpo, comprimir
│   ├── wire/     a fronteira SEM transporte — é o que o domain alcança
│   └── events/   o barramento tipado
├── convention/   os guardas que não são de pacote nenhum
├── parity/       os dezoito oráculos de ficha
└── scripts/      a folha e as ilhas de JS
```

**A seta só aponta para BAIXO**: `serve` → `app` → `domain` + `infra`, e o
`infra` não conhece ninguém. Quem garante são os `boundary_test.go` de cada
pacote — e, entre os GRUPOS, o `TestNoLayerImportsUpwards`, que varre a árvore em
vez de uma lista, para o pacote que nascer amanhã já nascer medido. As listas de
permitidos são argumentadas linha a linha: acrescentar uma entrada é decisão, não
conveniência.

### O `app/` é novo, e o que ele NÃO é (ALE-344)

Ele não é "a pasta para onde mudo o que estava no `api`". O que entra aqui é o
gesto INTEIRO — quem pode, o que decide, o que grava —, e o sinal de que uma
coisa pertence a ele é ter as três. Um repasse de uma linha não vira caso de uso
por mudar de pasta.

**A orquestração já existia, com outro nome.** O `apply` do store dos
tabuleiros carrega o estado, chama a regra PURA do `board_state.go` e devolve o
quadro: isso é um caso de uso, e ele passou anos arquivado dentro de `domain/`.
Os dois stores mudaram de lugar na ALE-344, e o número que fecha a divisão é o
do `domain/board`: **1.802 linhas, ZERO toques de persistência.**

**O que sobra em `domain/` é a regra, e ela é PÚBLICA.** As mutações que os
stores chamam — `AddToken`, `AdvanceTurn`, `PatchEntryVitals` e as irmãs — eram
privadas enquanto o chamador morava no mesmo pacote. Hoje são a API dos dois
pacotes puros, e é assim que se reconhece um: recebe estado, devolve estado, não
trava nada e não grava nada.

**As recusas daqui são TIPADAS** (`ErrNotFound`, `ErrForbidden`, `ErrRefused`) e
nunca um número de HTTP. Um caso de uso que devolvesse 403 não poderia ser
chamado de outro transporte — que é a única coisa que esta camada compra.

> **O `character.Plays` é a exceção declarada, e a razão é a TELA** (ALE-347).
> Quem embrulha uma recusa com `%w: app.ErrRefused` coloca "recusado pela regra"
> no fim da frase que o `Error()` devolve — e a cena da ficha mostra esse texto
> CRU ao jogador, porque o Datastar não desenha corpo de resposta 4xx e toda
> recusa dela tem de voltar como frase na cena. Os gestos da ficha têm UM
> transporte, e nenhum chamador lê o tipo: o embrulho custaria o sufixo na tela
> e não compraria nada. Quando o segundo transporte chegar, é o embrulho que
> desce — uma linha por recusa. O que NÃO pode voltar é o número de HTTP dentro
> da regra, e esse saiu: o `applySpellBuffEffect` devolvia `(efeito, int, error)`
> e montava um erro de campo com status 400 lá dentro.

**Onde procurar:**

| a pergunta | a pasta |
|---|---|
| quanto de PV este herói tem | `domain/engine` |
| como a ficha é montada do banco | `domain/sheet` |
| o que a tela desenha | `serve/web/<cena>` |
| a folha, uma ilha de JS, uma fonte | `serve/web/assets` |
| quem cumpre o que a cena pede | `serve/api` |
| onde o dado é gravado | `infra/db` |
| por que a suíte reprovou uma convenção | `convention` |

> Os TESTES moram ao lado do código, e isso é do Go: teste de caixa branca
> precisa do mesmo diretório para alcançar o que não é exportado. É por isso que
> `serve/api/` é de longe o diretório mais populoso do repositório, e que a maior
> parte das entradas dele é teste — o `ls` é a fonte, e o parágrafo abaixo diz
> por que um número escrito aqui não sobreviveria.

**Cada cena é um pacote em `web/`**, e o `api` guarda a composição do roteador, as
portas que cada cena pede e as regras que não são de tela nenhuma. Ver "Como uma
CENA é construída".

**No `serve/api`, o nome do arquivo diz o ADAPTADOR que o possui**, e o assunto
vem depois do prefixo: `table_*` é do `tableRules`, `sheet_*` do `sheetRules`,
`campaign_*` e `account_*` dos outros dois. **Um arquivo, um dono** — o
`character.go` tinha TRÊS donos, e o arquivo dos membros tinha dois — ele foi
repartido um por dono, e nenhum dos dois nomes dizia qual (ALE-330). A metade da
mesa desceu para o `app/initiative` na ALE-344; a de campanha é o
`campaign_members.go`.

Quem cobra é o `TestEveryAdapterFileCarriesItsPrefix`: ele lê o RECEPTOR dos
métodos e falha com o nome do arquivo e o do dono. Duas coisas que ele NÃO
conta, e as duas são de propósito — o `*Server`, que é fiação e está em metade
do diretório, e os hosts de cena (`tableHost`, `hubHost`), que já têm a
convenção deles em `*_deps.go` e convivem com um adaptador no mesmo arquivo.

> Aqui morava a contagem: quantos métodos o `*Server` tinha, quantos arquivos o
> `api` tinha, quantas rotas sobraram. **Dois dos números estavam errados** — o
> guia dizia oito métodos onde havia onze, e 36 arquivos onde havia 54. Número
> escrito à mão sobre coisa que muda envelhece sozinho, e o `grep` é a fonte
> (ALE-325).

**As cenas atendem na RAIZ**, sem prefixo. Duas consequências que o `git grep`
não mostra e que já morderam:

- **O `alvoOriginal` lê `RequestURI` e não `URL.Path`**, porque o segundo
  descarta a QUERY.
- **Endereço interno não soma prefixo.** Um `"/" + rota` vira `//livro`, que não
  é uma barra a mais: `//algo` é URL relativa a PROTOCOLO, e o navegador a lê
  como o HOST `algo`. Quem denuncia é um teste de cena; o compilador não tem
  como.

## Ambientes

`LoadConfig` lê `.env.<APP_ENV>` deste diretório antes do env do processo —
**o processo vence o arquivo**, sempre (ALE-119). O `.env.development` é
versionado porque nada nele é segredo; o `.env.production` é do dono da mesa e
não entra no git.

Configuração nova entra em `infra/config/config.go` **e** nos dois arquivos `.env` — um
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

**Não há segunda implementação para conferir contra**: o oráculo é o Go
descrevendo o Go, e **um bug no motor vira a nova verdade em silêncio**. A
mitigação não é técnica, é de processo, e é a linha acima.

O que ele protege é a ficha inteira de 18 personagens, ponta a ponta, acusando
qualquer número que mude sem ter sido pedido.

E a paridade nunca teria salvado: quando havia dois motores, eles estiveram
**perfeitamente de acordo** durante meses enquanto AMBOS erravam a RD do
Guerreiro, que não existe no livro. Paridade prova concordância; nunca prova
correção.

## Regras vêm do livro, com página

Todo teste de regra cita a página do [Tormenta 20](/t20-book.pdf) e, quando
existe, o **exemplo trabalhado** do próprio livro — a Samira da p173, a Bola de
Fogo do arcanista de 11º nível na p171, o clérigo/druida da p226. Um exemplo
trabalhado é melhor que uma asserção inventada: ele separa leituras possíveis da
mesma frase.

**Confira a citação antes de escrever.** O offset é `página do PDF = página do
livro + 6`. Onze citações erradas já foram corrigidas neste repositório, três
delas escritas no mesmo dia em que eu as "verifiquei".

## O sqlc trunca SQL por causa de comentário acentuado

Comentário em `infra/db/query.sql` é **ASCII**. O sqlc mede a query em bytes e conta o
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

Saída prática: **tabela nova em vez de coluna nova**. Foi o que a `00005` fez
com o tabuleiro, e de quebra "sessão sem tabuleiro" passou a ser dito pelo
schema (a linha existe ou não existe) em vez de por convenção sobre um JSON
vazio.

E a `00010` mostrou a segunda metade da armadilha (ALE-205): **trocar a chave
primária de uma tabela também não passa pelo gerador.** No SQLite isso é um
rebuild com `CREATE` + `INSERT SELECT` + `RENAME TO`, e o `RENAME` some do
catálogo do mesmo jeito que o `ADD COLUMN`. O caminho que funciona é criar a
tabela NOVA com outro nome, copiar, e derrubar a velha — foi assim que o
`session_boards` (1:1 com a sessão) virou `open_boards` (uma linha por tabuleiro
aberto).

**E o guarda de schema precisou aprender o `DROP`**: a lista de esperadas lia só
os `CREATE`, então toda tabela que qualquer migração já tivesse criado era
exigida para sempre — o servidor recusaria subir sobre um banco CORRETO,
nomeando como faltante justamente a que a migração acabou de derrubar. Ver "O
boot confere o SCHEMA".

## O que o servidor RENDERIZA sai comprimido na hora

Cena renderizada não existe antes da requisição, então a escolha real para ela é
gzip na hora ou nada — o `net/http` não comprime nada sozinho.

**`gzip;q=0` é uma RECUSA**, e um `strings.Contains` a leria como aceitação. É
por isso que o `httpio.AcceptsEncoding` existe em vez de uma busca por
substring.

Era **nada** até a ALE-273, e a conta é maior do que parece porque todo comando
da ficha responde redesenhando a cena INTEIRA: a aba de Combate viaja 44,7 KB
crus, 5,6 KB em gzip, e vai de novo a cada toque no PV. Numa LAN isso não
aparece; no telefone do jogador com dados móveis, são 44 KB por toque.

Quem faz é o `httpio.Gzip`, montado na borda do mux em `cmd/api`. Ele decide
pelo `Content-Type` que o handler escreveu, e pula o que já chega com
`Content-Encoding`: recomprimir produz bytes maiores gastando CPU.

**A armadilha mora no SSE, e ela não deixa erro para trás.** A resposta de todo
comando do Datastar é `text/event-stream` — ela usa o envelope de SSE para
mandar UM remendo e fechar. Então "não comprimir SSE" pularia justamente o que
se quer comprimir; e comprimir SEM repassar o `Flush` prende o quadro no buffer
interno do `gzip.Writer`, e o fluxo AO VIVO da Mesa para de atualizar. Nada
falha, nada loga, e o sintoma — "o tempo real quebrou" — não aponta para um
middleware de compressão. Por isso o `Flush` esvazia o gzip ANTES de quem está
embaixo, e `TestTheLiveStreamCrossesTheGzip` mede um quadro chegando com a
conexão ainda aberta.

**E há uma segunda metade, que custou 27 casos vermelhos no e2e com os
unitários TODOS verdes.** O `datastar-go` monta o fluxo nesta ordem: escreve o
`Content-Type`, chama `Flush()` para MANDAR OS CABEÇALHOS, e só então escreve o
primeiro remendo. Um envelope que decide comprimir apenas no `Write` chega
tarde — os cabeçalhos já foram sem `Content-Encoding` e o corpo sai comprimido
mesmo assim, então o navegador lê bytes de gzip como se fossem texto. Nenhuma
requisição falha, nenhum status muda: o que se vê é que os remendos param de ser
aplicados, e a busca não filtra, a seta não anda, o diálogo não abre.

Os guardas unitários estavam verdes porque escreviam o cabeçalho ANTES de
esvaziar, que não é o que a biblioteca faz. **Um envelope de resposta tem de
decidir no `Flush` também**, e `TestAFlushBeforeTheWriteAlreadyDecidesTheEnvelope` repete a
ordem do Datastar de propósito. A regra geral, que vale para qualquer middleware
que se escreva aqui: *quem esvazia compromete os cabeçalhos*.

O envelope também expõe `Unwrap() http.ResponseWriter`, que é o contrato do
`http.ResponseController` desde o Go 1.20 — sem ele, um `SetWriteDeadline` ou um
`Hijack` de qualquer camada acima responde `ErrNotSupported`, e quem chamou
conclui que o ambiente não suporta fluxo.

## O `synchronous` do SQLite vale 139ms POR TOQUE

O padrão do SQLite é `FULL`, que faz `fsync` a cada commit, e num prato girante
isso é uma rotação de disco por escrita. Medido (ALE-273): o comando que muda o
PV e redevolve a cena levava **121ms** de servidor e **139ms** até o número
mudar na tela. Com `synchronous=NORMAL`, no MESMO disco, viraram **1,7ms** e
**12ms**.

O controle que isolou a causa foi trocar só o LUGAR do arquivo — disco → tmpfs,
binário idêntico —, e o número do tmpfs bateu com o do pragma. Antes disso a
suspeita natural era o tamanho do HTML, e ela já estava descartada por outra
medição: **o tempo do POST era o mesmo em abas de 15 KB e de 39 KB.**

O que se perde numa queda de energia são os commits ainda não sincronizados. O
banco NÃO corrompe, e essa é a garantia do WAL: o arquivo principal nunca fica
meio-escrito, porque o conteúdo novo mora no `-wal` e a recuperação relê os
quadros até o último commit válido, cada um com checksum. É a diferença entre
`NORMAL` e `OFF` — e `OFF` continua sendo só do banco de teste, onde não há o
que proteger.

Para quem vem do MariaDB: `FULL` ≈ `innodb_flush_log_at_trx_commit=1`, `NORMAL`
≈ `=2`. E o WAL do SQLite é parente do **redo log do InnoDB**, não do binlog —
log físico de páginas, para recuperação e concorrência, e não log lógico de
eventos para replicação.

## HTTPS termina NESTE processo (opcional)

`TLS_CERT_FILE` + `TLS_KEY_FILE` preenchidos e o `ListenAndServeTLS` entra no
lugar do `ListenAndServe` (ALE-118). Vazios nos dois — o padrão — nada muda.
**Meio par derruba o boot** (`Config.validateTLS`, e ele roda em desenvolvimento
também): cair para HTTP em silêncio é o pior dos mundos, porque quem escreve
meio par liga `COOKIE_SECURE=true` junto e aí o navegador descarta o cookie de
sessão — o login não conclui e não há erro em lugar nenhum.

Terminar aqui, e não num proxy na frente, é o que mantém a decisão de um
processo só. Quem terminar TLS fora deixa os dois caminhos vazios.

O esquema do log vem da config (`Config.Scheme`), e isso não é cosmético: aquela
linha É o endereço que o mestre repassa para a mesa.

**Medido, não suposto (ALE-118):** com TLS o Chrome negocia **h2** para a
página. A pré-compressão (`.br`) atravessa o TLS e o h2 intacta.

O tempo real é SSE, então ele é um `GET` como qualquer outro e viaja pelo mesmo
h2 — sem conexão à parte e sem exceção para lembrar.

O que este repositório **não** decide é de onde vem o certificado — as duas
saídas e o preço de cada uma estão no README, e nenhuma delas se executa de
dentro do repositório.

## O boot confere o SCHEMA, não o `goose_db_version`

A migração pode CONSTAR aplicada sem a tabela existir. Aconteceu: a tabela do
tabuleiro (então `session_boards`) sumiu do banco de desenvolvimento com a 00005
marcada, o goose disse "no migrations to run", e o tabuleiro passou um dia
vivendo só em memória — cada gravação falhando numa linha de log que ninguém lê
(ALE-154).

Por isso o `db.Open` roda `assertSchema` DEPOIS de migrar e **recusa subir**
nomeando as tabelas que faltam. A lista de esperadas é lida das próprias
migrações embutidas, nunca escrita à mão: lista à mão envelhece em silêncio, que
é como este repositório já perdeu o `TurnsTaken` e o `creatureId` no mesmo dia.

Ela lê os DOIS verbos da seção `Up`, `CREATE TABLE` e `DROP TABLE`, na ordem dos
arquivos (ALE-205). Só com o primeiro, o schema nunca podia PERDER uma tabela — a
derrubada legítima da `00010` virava "falta a `session_boards`" num banco que
estava certo.

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
`WriteTimeout` de propósito** (ALE-157): ele mataria o fluxo SSE, que é conexão
longa por natureza, e um download grande numa rede ruim. É o timeout que parece
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

## O tempo real é SSE, e o canal MUDOU de dono

**Bidirecional não é requisito, é hábito.** O que sobe é sempre MUTAÇÃO, e
mutação é uma requisição; o que desce é um `text/event-stream` que fica aberto.
Foi assim que um socket bidirecional virou uma rota por comando mais um `GET`
longo, e o argumento continua valendo para o próximo canal. A Mesa em
Datastar tem fluxo PRÓPRIO (`/campanhas/{campanha}/sessoes/{sessao}/fluxo`, em
`web/table/stream.go`), ele assina o `events.Bus` e não o `SSEHub`, e os comandos
dela são rotas da CENA. Nenhuma linha do que este arquivo descrevia como "a rota
de eventos" existe.

> **E o `SSEHub` ficou sem ouvinte.** Medido na ALE-277: em produção ninguém
> chama `SSEHub.Add` — só testes —, porque a única rota que abria conexão era a
> `/events`. Quem EMITE continua lá, e emitir para zero ouvinte não estoura nada.
>
> Ele segue de pé, e o que MUDOU é que apagá-lo deixou de ser perigoso: a
> gravação saiu de dentro do publicador na ALE-288 — ver "A gravação saiu de
> dentro da publicação", abaixo. A `PresenceRegistry` também deixou de estar
> vazia: quem a preenche agora é o fluxo da própria Mesa (ALE-287).

Três coisas sumiram junto, e todas eram exceção:

- **A política de origem duplicada.** O socket tinha caminho próprio no mux,
  fora do `Router()`, e por isso precisava do `guardSocketOrigin` repetindo o
  CORS por conta (ALE-158). O `/events` está debaixo do `cors.Handler` do
  roteador — a política é uma só.
- **O token na query string.** O `EventSource` não deixa pôr cabeçalho, mas manda
  COOKIE, e o `extractToken` lê o cookie antes do header. Então o fluxo entra
  debaixo do `requireAuth` sem gambiarra.
- **A detecção de queda por biblioteca.** O `r.Context()` é cancelado quando o
  cliente vai embora. A batida de 25s (`sseHeartbeat`) NÃO é para isso — é um
  comentário SSE (`: ping`) para atravessar intermediário que fecha conexão
  ociosa.

O `emit` do hub **nunca bloqueia**: fila cheia descarta o quadro daquele leitor.
Quem perdeu reconecta e busca o estado por HTTP, que é o mesmo caminho da
primeira carga — nenhum estado vive só no fio.

**Uma armadilha que custou uma rodada:** o `characterChanged` era um GANCHO
preenchido pelo `SocketHandler()`. Apagar o gateway deixou o gancho sem quem o
ligasse, o `go test ./...` seguiu VERDE — havia até um teste afirmando que nulo
era caminho normal — e quem acusou foi o e2e de dois clientes. Campo que precisa
ser preenchido por outro arquivo para o recurso existir é recurso que nasce
desligado.

## O barramento de eventos: o que acontece na mesa é TIPADO

O `events.Bus` (ALE-279) entrega, dentro do processo, o que aconteceu numa mesa.
Ele substituiu quatro mecanismos com a mesma forma e nenhum nome em comum — o
`Assinar` de cada um dos dois stores, o `CharacterWatch.Assinar` e um
`Emit` —, os três primeiros `chan struct{}`: diziam QUE algo mudou e nunca O
QUÊ, e o `select` do stream da Mesa tinha um `case` para cada um só para juntar
de volta o que estava separado por acidente de onde o estado mora.

**Não é event sourcing, e a diferença é a decisão inteira.** O banco continua
sendo o estado. As regras do livro são conta e não fluxo — empilhamento de
modificador, PV/PM, círculo de magia são função pura de `sheet → números`, com o
oráculo como rede —, então o `engine/` não participa disto e reconstruir ficha a
partir de eventos compraria versionamento de evento e snapshot sem nada em troca.

Três coisas que este desenho pede, e uma que ele proíbe:

- **Nomear o evento é obrigação de COMPILAÇÃO.** O `apply` dos dois stores recebe
  o evento por parâmetro, então não dá para mutar sem dizer o que aconteceu.
  Antes o aviso era uma linha dentro do funil e um comentário prometia que
  ninguém escapava — promessa que vale enquanto ninguém escrever a mutação de
  fora.
- **Publicar acontece FORA da trava.** O barramento é folha e poderia ser chamado
  de dentro, mas quem acorda agora sabe o que houve e pode ler o estado na hora;
  publicar sob a trava faria esse leitor esperar pelo escritor no instante em que
  foi acordado para ler. Onde o método usa `defer Unlock`, o corpo vira
  `…Locked` e um invólucro fino publica.
- **A fila cheia DESCARTA e CONTA.** `chan struct{}` colapsa — dois "mudou"
  pendentes não dizem mais que um —, e evento tipado não colapsa, então a fila
  tem dezesseis lugares. O contrato de quem escuta continua o mesmo: o evento é a
  notícia, a verdade está no store. O que mudou é que o descarte deixou de ser
  invisível (`Subscription.Dropped`), pela mesma razão que o medidor de contraste
  devolve o denominador.
- **`events/` não importa NADA do projeto.** Ele teve de entrar na lista de
  permitidos do `aovivo` e do `board`, cujos guardas de fronteira avisam que
  acrescentar import à lista transforma a porta em enfeite. O que impede a porta
  dos fundos é o `TestVocabularyImportsNothing`: enquanto o vocabulário for
  folha, depender dele não cria fronteira errada — e no dia em que alguém
  importar a ficha ali "para enriquecer o evento", os dois contextos passam a
  alcançar a ficha de graça com o guarda de lá verde.

Um barramento tipado é o caso em que a PORTA não serve, e vale saber por quê:
porta é interface declarada no consumidor, e ela só casa com tipos do consumidor
— um vocabulário por contexto, que é o problema de novo. O que legitima o
compartilhamento é ele ser *shared kernel*: pequeno, sem dependências, e de todos
porque não é de ninguém.

## Catálogos

**As cenas leem o catálogo EMBUTIDO direto** — ele não viaja por HTTP, e não há
rota que o sirva.

A regra que governa quem o lê: **uma vez e guardado**, não por requisição, porque
o conteúdo vem de `go:embed` e não muda enquanto o binário for o mesmo. É o que
o `race_traits.go` faz com `sync.Once`.

`domain/catalog/data/*.json` é embutido no binário. **Este é o único lugar onde
catálogo é autorado** — mudar uma magia é editar um arquivo só, e a cena, o
motor e os testes leem o mesmo arquivo.

**E há DUAS fontes do mesmo `items.json` no processo, com durabilidades
diferentes.** O `catalog.Resource` é `go:embed` — existe sempre que o binário
existe. O `s.catalogs` é primado de um arquivo por caminho de configuração, e o
`primeCatalogs` diz em log o que faz quando ele falta: "mutation validators
disabled". Regra que se desliga sozinha quando um arquivo some não é regra, e a
bancada mostrou o preço na ALE-272 — um escudo foi VESTIDO num teste porque o
catálogo do fixture está vazio. **Validação de regra lê o embutido**; o
`s.catalogs` fica para o motor, que é primado pelo mesmo caminho que o oráculo.

**Regra que só a TELA sabe é fronteira aberta, e o sintoma é sempre o mesmo: a
tela tranca e o servidor não.** Três já morderam — a progressão de círculo (em
que nível cada classe destrava cada círculo), a compatibilidade entre
melhoria/material e o item que os recebe, e o limite de nome de campanha.

Hoje as três moram no catálogo ou no domínio: `spellcasting` em `classes.json`,
`aMelhoriaCabeNoItem`, `campaign.Description`. O filtro que a tela aplica é
conveniência sobre a mesma regra, nunca a regra.

E a QUARTA era a maior: as regras de ESCOLHA de poder — quantas vagas o nível
abre (uma por nível a partir do 2º, p33), quantos benefícios a origem dá, quais
caminhos e quais deuses cada classe aceita — só a tela sabia, e a gravação
aceitava os cinco blobs sem conferir NADA. Um pedido montado à mão punha vinte
poderes num personagem de nível 1, e o motor somava os modificadores de todos. A
validação é
`sheet.WithChoicesValid`, ela roda nas DUAS portas (o endpoint JSON e os
comandos da ficha), e é **estrita**: a escrita tem de deixar a ficha VÁLIDA, e
não só "não piorar". Decisão do dono, com a razão registrada — o projeto ainda
não foi usado numa mesa real, então não há ficha antiga fora da conta para
proteger. O preço apareceu na hora: dois testes tinham fixtures ilegais pelo
livro (um poder inventado, e um personagem sem classe com um poder escolhido).

O que protege dado transcrito é **validação de schema**
(`catalog/rules_tables_test.go`), não um `expect` por campo: o risco é typo, não
regressão. O que ela cobre é o que quebra tela — perícia que não existe, faixa de
rolagem com buraco, termo de devoto apontando para raça inexistente.

**E o que ela NÃO cobre é número errado.** Medido na ALE-151: 44 dos 80 verbetes
do bestiário tinham atributo ou resistência trocados, vários com o número do
verbete VIZINHO — a Serpe carregava as resistências do Ogro, e o schema estava
verde o tempo todo, porque um inteiro no lugar de outro inteiro é um schema
válido. Contra isso só serve conferir contra o livro:

```
python3 scripts/audit-bestiary.py            # relatório
python3 scripts/audit-bestiary.py --aplicar  # escreve as correções
```

Rodar é **ato deliberado**, como o `genoracle`, e pela mesma razão: a ferramenta
PROPÕE lendo o PDF, e é o diff revisado contra o livro que decide. Ela só aceita
um bloco contíguo e completo cuja Defesa e Pontos de Vida já batam com o
catálogo — essas duas âncoras é que dizem "é esta criatura".

Uma armadilha que ela documenta e que vale para qualquer extração deste PDF: o
`pdftotext -layout` junta colunas VIZINHAS na mesma linha de texto, então uma
linha de atributos aparece colada à criatura errada. Ler por coordenada, nunca
por layout.

### E o buraco INVERSO: o valor existe, mas não se confere que ele APONTA

O schema pega FORMA e não VALOR — é o parágrafo acima. O `seed-data.json` tinha o
buraco virado do avesso: os valores eram strings bem formadas, e ninguém conferia
se elas achavam alguma coisa no catálogo. Só `create.items[].catalogId` passava
por lookup; raça, origem, classe, deus, poder concedido, tamanho e magia entravam
crus (ALE-226).

**Seed que mente é caro porque não quebra**: `machado-de-batalha` no lugar de
`machado-batalha` produz um personagem sem a arma, com a ficha abrindo normal. E
o e2e roda contra a seed, então o combatente sem arma vira um teste que mede o
ambiente em vez do app.

Três coisas que a correção ensinou, e valem para qualquer validador de
referência:

- **A mensagem sugere o VIZINHO**, porque errar id é erro de digitação e
  digitação erra por pouco. O teto de distância — um terço do comprimento — é o
  que separa sugestão de chute: sugestão errada é pior que nenhuma, porque quem
  lê a segue.
- **Ela junta TUDO antes de falhar.** Parar no primeiro erro faz quem escreveu
  cinco ids errados rodar o gerador cinco vezes.
- **O validador afirma o próprio DENOMINADOR.** O `catalog` carrega os embeds com
  `sync.Once` e engole erro de leitura e de parse: com o embed quebrado toda
  lista vem vazia, e um validador ingênuo acusaria TODOS os nomes do arquivo —
  culpando quem escreveu o seed por um defeito do build. Lista vazia é falha de
  carga, e a mensagem diz isso.

E a referência que escapou vale registro: a CHAVE do `classChoices` é um nome de
classe (`{"Arcanista": {…}}`). Referência escondida em chave de objeto não se
parece com referência, e ela só apareceu quando o
`TestEveryCreateFieldOfTheSeedIsClassified` obrigou a classificar campo por
campo. **Guarda que força a varredura acha o que a leitura não acha.**

## O fixture do app prima o catálogo DE VERDADE

O `newSceneFixture` primava `{"items":[]}`, e isso fazia regra sumir do TESTE sem
sumir da produção — duas vezes na mesma épica, e nenhuma delas apareceu como
erro:

- a fatia 7 mediu um escudo sendo **VESTIDO**, porque o eixo de equipar não
  achava o item no catálogo vazio e devolvia "sem opinião";
- a fatia 8 mediu a distribuição de atributo do humano passando com **três vezes
  o mesmo atributo**, porque a raça não estava lá e a validação tratava
  desconhecido como completo.

Nos dois casos o guarda ficou verde afirmando o contrário do que mede. O fixture
agora lê o mesmo `parity/_catalogs.json` que o servidor lê, como o
`newTestServer` do `character_cast_rules_test.go` já fazia.

**A regra geral: catálogo vazio no fixture é validação desligada em silêncio.**
Quando uma checagem responde "não sei" para o que não está no catálogo — e é o
que quase todas fazem, porque recusar o desconhecido travaria fichas antigas —,
primar vazio é escolher que ela nunca rode.

## A bancada dos testes: um molde migrado, copiado por teste

`newTestServer` abre um SQLite de VERDADE por teste — é o que faz este pacote
provar composição em vez de mock. O que custava caro era migrar: `db.Open` roda
as migrações todas, e um teste por banco dava ~3.400 migrações com `fsync`.

O preço de um `fsync` é o do dispositivo onde o `TMPDIR` cai, e a diferença não
é de grau (ALE-260, medido nesta máquina):

|                     | tmpfs (`/tmp`) | disco girante (`/mnt/HD`) |
| ------------------- | -------------- | ------------------------- |
| migrar do zero      | 7,1 ms         | **2.102 ms**              |
| copiar o molde      | ~0,1 ms        | 0,076 ms                  |
| reabrir já migrado  | ~1 ms          | 1 ms                      |

No prato girante cada migração custava ~49 ms — uma rotação por `fsync` — e a
suíte do `api/` levava **15m01s** com apenas 10 s de CPU: 99% de espera. O
sintoma mente, porque aparece como "os testes estão lentos", que é a conclusão
que faz alguém cortar teste ou subir o timeout em vez de consertar a bancada.

Duas mudanças, as duas só no teste:

1. **O molde** (`db/testdb`, extraído do `api` na ALE-281): o `TestMain` migra
   UM banco e cada teste o copia. `db.Open` continua sendo o mesmo de produção,
   com o mesmo `assertSchema` — o goose encontra a última versão e não tem o que
   fazer.
2. **`PRAGMA synchronous=OFF`** no banco de teste. Durabilidade é o que um banco
   que morre no fim do caso não tem o que proteger. Fica no helper e **nunca** no
   `db.Open`: em produção essa linha é perda de dados do mestre.
3. **A cópia FALHA ALTO** (ALE-268), e as três defesas dela existem porque a
   cópia era a parte silenciosa do arranjo: o erro do `Close` era descartado num
   `defer`, não havia `Sync`, e ninguém conferia se o arquivo chegou inteiro.
   Hoje o `Close` volta o erro, o `Sync` força os bytes ao disco, e o TAMANHO é
   comparado com o do molde.

   **O sintoma que isso conserta apontava para o lugar errado**: uma cópia
   parcial produz um SQLite truncado, que se comporta exatamente como um banco
   sem as tabelas — o CI reprovou uma vez com `no such table: session_boards`
   junto de um `disk I/O error`, e "no such table" é a frase de um banco NÃO
   MIGRADO. Quem investigasse iria caçar migração. E repare que o item 2 acima é
   o que remove a barreira que tornaria isso barulhento sozinho: a decisão certa
   para velocidade é a que cala o erro.

   **O preço foi medido e não estimado**: `./api/ ./db/...` foi de 10,34 s para
   10,96 s, ~6%, e é o `Sync` por banco de teste. Frequência do defeito em
   2026-09-08: uma ocorrência em 100 corridas de CI.

Resultado: `api/` de **15m01s para 24 s** no disco girante, e a suíte Go inteira
de 6,8 s para **4,7 s** em tmpfs. Os 178 casos que o pacote tinha na época
continuaram rodando, zero pulados — a aceleração não veio de cortar teste, que é
a primeira suspeita quando uma suíte encolhe de quinze minutos para vinte e
quatro segundos.

**Por que o molde e não exportar `TMPDIR` para um tmpfs.** A variável funciona —
mas só para quem lembrar, e só na máquina de quem lembrou. Quando isto foi
medido havia uma sessão vizinha rodando com o `TMPDIR` no prato girante sem saber
que pagava quinze minutos por corrida. O molde tira o disco da conta em vez de
pedir que alguém escolha o disco certo.

**Ela virou PACOTE (`db/testdb`) na ALE-281, e o motivo é de forma.** Arquivo
`_test.go` não exporta nada para fora do pacote, e o `api` está se dividindo em
um pacote por cena (ALE-278) — a primeira cena a sair encontraria a bancada
inalcançável e escreveria a própria, que é como um fixture nasce com catálogo
vazio e desliga validação em silêncio. Cada pacote que usa declara uma linha:

```go
func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }
```

O que NÃO foi junto é a montagem do servidor: ela precisa do tipo `api.Server`, e
um pacote de bancada que importasse o `api` seria importado de volta pelos testes
dele — ciclo. O molde é a parte cara e a parte compartilhável; o fixture é de
cada pacote.

**O preço de dividir foi medido antes de dividir, com controle** (o mesmo `go
test -run` que não casa com nada, num pacote SEM banco, para descontar a subida
do processo): **248 ms por pacote**, ou ~3,7 s na corrida inteira com quinze. O
número é maior que os 7,1 ms de "migrar do zero" da tabela acima porque ele
inclui ligar um binário que carrega o SQLite e as migrações — a tabela mede o
`fsync`, esta medição mede o pacote.

## Testes

As faixas, o vermelho antes de confiar e o que não merece teste estão no
[CLAUDE.md da raiz](../CLAUDE.md). O que é deste pacote:

- `go test ./...` — sem flag, sem setup.
- **Teste de regra vive junto da regra e cita a página.** Ele não é a mesma coisa
  que o ORÁCULO: o oráculo prende a ficha inteira de ponta a ponta e acusa
  qualquer número que mude sem ter sido pedido; o teste de regra explica POR QUE
  aquele número é aquele, com a página do livro do lado. Nenhum substitui o
  outro.

- **Sabotar é a forma barata de provar que o teste mede o que diz medir**, e no
  Go ela é barata mesmo: inverta o operador, rode o caso, confirme o vermelho,
  reverta. Foi assim que se descobriu que um teste de PV passava por acidente —
  em todos os casos a primeira classe também era a maior.

  > **E às vezes não é preciso sabotar: a árvore de ONTEM é o caso vermelho.** Os
  > dois guardas do `convention` foram provados assim — `git archive` do commit
  > anterior, o guarda copiado por cima, `go test`. Deu 775 reprovados num e 128
  > no outro. É melhor que sabotagem por dois motivos: o caso negativo é real e
  > não inventado, e não há como a sabotagem sair inerte — que já enganou duas
  > vezes nesta épica.

## `convention`: os guardas que não são de pacote nenhum

`convention/` (ALE-282) não tem código de produção. Ele existe porque duas frases
do CLAUDE.md da raiz se contradizem quando a regra é sobre TODOS os pacotes:
"regra mecanizável vira guarda" e "o guarda mora no pacote que a possui". Pôr uma
regra do repositório no `api` seria escolher um dono arbitrário — e o `api` está
sendo dividido em um pacote por cena (ALE-278), então o guarda mudaria de casa
junto com a próxima fatia.

São **23** hoje (`grep -h '^func Test' convention/*_test.go | wc -l`), e aqui dizia OITO
até a ALE-317 — o número envelheceu sozinho ao longo de sete issues, que é
exatamente o que a seção "Como uma convenção passa a valer" do guia da raiz
prevê para número escrito à mão sobre família que cresce. **O `grep` é a fonte.**

O que vale saber não é a contagem, é por que cada um veio parar aqui, e são três
razões. Uns nasceram lá porque a regra é do REPOSITÓRIO e não de um pacote;
outros MUDARAM de casa porque mediam o próprio diretório e encolheram quando uma
cena virou pacote; e três são a família das citações.

- **`TestEveryTestNameIsEnglish`** varre todo `*_test.go` e recusa nome com
  palavra em português. Ele é o que impede o 774º — a regra de idioma sempre
  disse "nome de teste" com todas as letras, e ainda assim 773 dos 1.051 casos
  estavam em português, porque *convenção escrita e não varrida é aplicada
  exatamente aos arquivos que alguém apontou*.
- **`TestNoCitationNamesAMissingTest`** recusa comentário ou `.md` que nomeie um
  teste inexistente. A varredura mediu **136** dessas antes de começar: um `.md`
  fica errado sem ninguém mexer nele, e nada no repositório acusava.
- **`TestNoCitationNamesAMissingFile`** faz o mesmo para CAMINHO DE ARQUIVO, e
  mediu **41** (ALE-285). A origem é sempre a mesma: um renome ou uma mudança de
  pacote leva o arquivo e deixa para trás os comentários que apontavam para ele —
  "a regra mora no ..." nomeando um arquivo que já se chama outra coisa.

  Ele só olha `.go` e `.templ`, e a restrição é o que o torna possível: com as
  extensões todas ele acusa **325**, e a esmagadora maioria (181 `.ts`, 20 `.js`,
  17 `.tsx`) é PROCEDÊNCIA — um comentário dizendo de onde a regra veio, apontando
  para um arquivo que nunca vai voltar a existir. Está certo, e um guarda com 325
  exceções é um guarda que alguém apaga. Nas extensões do stack VIVO a
  ambiguidade some: um `.go` citado que não existe é endereço velho, não
  história.

  A primeira coisa que ele pegou foi a **própria docstring**, que nomeava os
  arquivos mortos como exemplo. Declará-los teria resolvido e seria errado: a
  lista é por nome-base, então perdoar um nome ali perdoaria também o próximo
  comentário que voltasse a apontar para ele. A prosa passou a nomeá-los sem a
  extensão.

- **`TestNoCitationNamesAMissingSymbol`** (ALE-286) fecha a TERCEIRA forma da
  família: o comentário que nomeia uma FUNÇÃO que sumiu. Ele mediu **466**, e o
  que o torna possível é o ruído ser resolvido por REGRA — cinco naturezas
  colhidas mecanicamente (chave de fio, chave de catálogo, literal de string, a
  API CHAMADA em qualquer `.X(`, e os identificadores do `.ts`) mais dois
  julgamentos (a CAIXA, que identifica procedência, e a procedência DECLARADA no
  bloco). Uma sonda ingênua acusa 1.371; com as regras, 122.

  A lista de lápides dele tem 66 nomes e quatro grupos, e o grupo que ensina é o
  do NOME HIPOTÉTICO: `detalheAberto` nunca existiu — ele é o exemplo do que
  aconteceria se alguém escrevesse um sinal em camelCase, e um guarda que o
  proibisse tiraria do repositório justamente a explicação que impede o defeito.

- **`TestEveryMarkerColorCanBePainted`** e **`TestNoSceneCommandUsesTheDefaultTab`**
  moram aqui porque um lia um caminho relativo ao `api` e o outro varria um glob
  de nome; quando a Mesa virou pacote, o arquivo sumiu debaixo de um e o glob
  deixou de casar para o outro. **Os dois falharam ALTO porque os dois tinham
  piso** — o irmão do foco, que não tinha, teria passado verde medindo metade.

- **`TestNoFocusAsksTheServerWithoutAKeyboardGuard`** chegou na ALE-278, e não
  por ser regra de repositório desde sempre: ele já existia dentro da cena do
  bestiário, e foi a divisão em pacotes que o obrigou a mudar de casa. A
  história inteira está em "Dois pedidos de UM gesto", mais abaixo; o que
  interessa aqui é a forma, porque ela vale para o próximo: **um guarda que
  varre `.` mede o diretório em que ele por acaso mora**, e mover o arquivo
  encolhe a varredura sem mudar uma linha do guarda nem acender nada.

- **`TestNoHandwrittenFocusRing`** (ALE-317) é regra do repositório desde o
  primeiro dia, e a razão de ele existir não é estilo: a receita de foco é
  GLOBAL e não layerada, então `focus-visible:outline-*` numa `class=` é
  decoração — 242 sítios a escreviam, e apagar os 242 não muda um pixel. O que
  vale copiar dele é o TERCEIRO controle: ele lê o `index.css` e afirma que a
  regra global continua lá. Um guarda que proíbe a cópia porque existe o
  original tem de falhar quando o original sumir, senão ele passa a cobrar uma
  regra cuja razão morreu — e aí a proibição deixa o app sem realce nenhum.

**A lista de marcadores tem uma fresta declarada**, e ela é deliberada: nome
PRÓPRIO do livro passa. `TestBolaDeFogoWorkedExample` e
`TestEspecializacaoEmArmadura` são o nome da magia e do poder, não prosa em
português — a alternativa seria uma lista que cobra a tradução de um nome
próprio, que é pior.

**E a lápide continua valendo.** `// Aqui morava o TestX, que prendia…` é boa
prática aqui: ela diz por que uma garantia SAIU, que é o que o `git log` esconde
de quem lê o arquivo. O guarda só pede que ela seja DECLARADA em `tombstones` —
apagar um teste é um ato, e o ato aparece numa linha.

## templ — as armadilhas que já custaram tempo

As cenas são `.templ` compiladas para `.go` por `go tool templ generate`. O
`.templ` e o `_templ.go` andam juntos, e o CI recusa o par desencontrado. O que
segue foi todo descoberto errando — está aqui para ninguém redescobrir:

- **`else if` NÃO existe numa lista de atributos, e o templ não reclama.** Ele
  fecha o primeiro `if`, escreve a palavra ` else` como TEXTO dentro das aspas do
  elemento e abre um `if` INDEPENDENTE — então os DOIS ramos saem, e o HTML tem
  um atributo chamado `else` mais o seu repetido duas vezes. Nada estoura:
  atributo repetido não existe no DOM, o navegador guarda o PRIMEIRO e descarta o
  resto em silêncio, e o segundo ramo fica morto sem deixar rastro. **A escolha
  entre dois atributos volta para o Go**, onde `else if` é `else if` — exclusão
  por CONSTRUÇÃO, e não dois blocos que se prometem exclusivos.
- **Comentário NÃO vive na lista de atributos de um elemento.** `// ...` entre
  dois atributos derruba o parser, e a mensagem aponta OUTRA linha — nunca a do
  comentário. Já aconteceu **oito** vezes; as três últimas foram na mesma sessão,
  por quem não leu esta linha antes de escrever.
  **A mensagem varia com o que envolve o elemento, e é por isso que ninguém
  reconhece o sintoma na segunda vez:** solto dá `malformed open element` na
  linha do `<`; dentro de um `for` dá `for: expected nodes, but none were found`;
  dentro de um `if` dá `if: expected nodes, but none were found` — as duas
  apontando a linha do BLOCO, dezenas de linhas acima. Se um `for`/`if` que não
  mudou começou a reclamar de "expected nodes", procure o comentário que você
  acabou de pôr entre dois atributos.
  O comentário vai ACIMA do `templ`, ou acima do elemento inteiro.
  **E o pior não é o erro: é que `go build` fica VERDE por cima dele**, porque o
  `_templ.go` antigo continua no lugar. `templ generate && go build` numa linha
  só esconde isso — o build sucede sobre código gerado velho, e a página serve o
  HTML de antes. Leia a saída do `templ generate`, não a do `go build`.
- **`@componente()` tem de COMEÇAR a linha.** No meio de um texto ele vira texto
  literal, e a página mostra `@tecla("⏎")` escrito na tela — sem erro nenhum.
- **Valor CONSTANTE de atributo sai literal; só o DINÂMICO é escapado.** Uma
  aspa simples num literal fica aspa simples; a mesma string vinda de variável
  vira `&#39;`. Importa quando se afirma HTML em teste.
- **NUNCA passe formatador em arquivo GERADO** (ALE-278). `goimports -w web/x/*.go`
  alcança os `_templ.go` e junta no bloco de imports os dois de runtime que o
  `templ generate` emite SOLTOS no topo. São doze linhas sem uma de
  comportamento, e a CI reprova em "Fail if the generated templates were stale"
  com a suíte local INTEIRA verde.
  **Gerado reformatado é gerado desencontrado por definição**, e é a metade
  desta armadilha que o item acima NÃO cobre: lá o `_templ.go` está ATRASADO e
  `templ generate` conserta; aqui ele está em dia no conteúdo e diferente na
  FORMA, então regenerar e compilar não denuncia nada. Só o `git diff` DEPOIS de
  regenerar:

  ```
  go tool templ generate && git diff --quiet -- 'engine-go/**/*_templ.go'
  ```

  Esse é o passo da CI, e ele tem de entrar no roteiro de toda fatia que mover
  arquivo de pacote — porque mover pacote é exatamente quando o `goimports` é
  preciso. Se ele for, restrinja o alvo: `_templ.go` só se corrige regerando.
- **Regenerar não basta: o servidor precisa REINICIAR.** O `go run ./cmd/api`
  compila uma vez, então depois do `templ generate` o processo continua servindo
  o HTML antigo. Isto já produziu uma medição de layout inteira contra a página
  velha — 74px de deslocamento "que não sumiam" depois do conserto, porque o
  conserto não estava no ar.
- **Classe nova exige `scripts/build-css.sh`.** Classe que não passou pelo
  scanner simplesmente não existe na folha, e o elemento aparece sem estilo em
  vez de dar erro.
  **Aqui morava "o scanner lê `../*.templ`, e só ele", e é falso** — medido na
  ALE-278: tirar os DOIS `@source` do `app.src.css` não muda um byte da folha
  compilada (105.328 com, 105.328 sem), porque a detecção automática do Tailwind
  v4 varre da pasta da folha até a raiz do projeto respeitando o `.gitignore`. As
  linhas ficaram como declaração de intenção; quem depurar "classe sumiu" não
  deve perder tempo nelas. O suspeito é o TOKEN que não existe na paleta.

  **E porque ele varre a árvore inteira, a PROSA entra na folha — então compile
  por ÚLTIMO.** O scanner não sabe distinguir uma classe aplicada de uma classe
  citada num `.md` ou numa docstring, e não há como pedir que ele ignore: um
  guia que explica uma receita faz o Tailwind emitir as utilidades daquela
  receita, mesmo sem nenhum consumidor. Isso custou uma CI vermelha na ALE-317:
  a varredura apagou 242 sítios, a folha foi compilada, e só DEPOIS os dois
  guias ganharam a explicação — o passo "Fail if the stylesheet was stale"
  reprovou com uma linha de diferença. A ordem do roteiro é **código → prosa →
  `build-css.sh` → `git add`**, e o mesmo vale quando a mudança APAGA
  classe: o artefato guarda o estado de antes.
- **Regra da casa que precisa GANHAR de um utilitário mora em `@layer utilities`,
  e não em `components`.** No Tailwind v4 a CAMADA decide antes da
  especificidade: `utilities` vence `components` mesmo quando o seletor de baixo
  tem duas classes contra uma. A ALE-218 escreveu `.notas-flutuam .notas-coluna
  { position: absolute }` em `components` e ela perdeu para o `lg:static` que a
  coluna traz do próprio `class=` — com o sintoma pedindo a conclusão errada: o
  mapa crescia **18px em vez de 728**, porque só a divisa (que não tinha
  concorrente) tinha flutuado. "Cresceu um pouco" parece regra aplicada com
  cálculo errado; era regra DESCARTADA, e a outra metade da tela mentiu por ela.
  E o conserto tentador — tirar o `lg:static` do elemento — **troca o defeito de
  lado**: sem ele o `fixed` da classe base passou a ganhar e a coluna virou tela
  cheia em toda largura. Quem tem de mudar é a camada.
- **`inset-0` põe `left: 0`, e `left` ganha de `right`.** Posicionar um painel
  pela direita não é só escrever `right: 0`: com os dois lados definidos e a
  largura fixa, o navegador resolve pelo `left` e o painel vai para o lado
  errado. Toda regra que ancora à direita sobre uma classe base com `inset`
  escreve `left: auto` junto — e o comentário fica, porque a linha parece
  redundante para quem não viu o painel do lado errado do mapa (ALE-218).
- **E TOKEN inventado tem o mesmo fim, com o script rodado.** `text-grimorio-ink`
  parece irmão de `text-grimorio-gold` e não é: `grimorio-ink` não está na
  paleta, o Tailwind não emite regra para o que não conhece, e o elemento fica
  com a cor HERDADA — o crachá de contagem dos Efeitos saiu dourado sobre
  dourado, 1,53:1, e atravessou uma fatia inteira. O
  `TestEveryHouseTintExistsInTheStylesheet` cobra cada token contra a folha
  compilada; a paleta mora no `@theme` do `serve/web/assets/src/index.css`, e é
  lá que se confere antes de inventar um nome.
  **Ele varre o DIRETÓRIO e não um padrão de nome, e isso custou duas vezes**: o
  glob era um padrão de NOME, e ele deixou de casar duas vezes: quando o kit
  mudou de arquivo, e quando os arquivos perderam um prefixo. Nas duas o guarda
  seguiria VERDE
  medindo menos. Padrão de nome acopla o guarda à nomenclatura; o diretório é o
  terreno, e ele não muda de nome sozinho.
  **E o que ele mede deixou de ser só a paleta `grimorio-*` na ALE-276**: os
  PAPÉIS semânticos — `destructive`, `primary`, `muted`, `card`, `popover` e os
  outros — não estavam em `asPaletasDaCasa`, então `text-destructive-foreground`
  foi escrito em três sítios com o token nunca declarado, e o guarda passou por
  cima. O botão "Excluir a sessão" herdou o `--foreground` do diálogo e saiu a
  4,33:1. A varredura foi de 21 tintas para 43, e o piso do denominador subiu de
  20 para 40 junto — um piso que o conjunto ANTIGO satisfazia não denunciaria a
  volta dele.

- **E a CLASSE DE ESCOPO não é uma tinta, então o guarda de tinta não a via.**
  `scene-grimorio` é a condição para os tokens existirem — o `@custom-variant
  dark` do `index.css` é `&:is(.dark *, .scene-grimorio, …)`. Sem ela, nenhum
  token resolve e o app inteiro sai sem cor, sem contraste e com outro realce de
  foco. Uma varredura de identificadores trocou `grimorio` por `grimoire` com
  `\bgrimorio\b`, e **o hífen é fronteira de palavra**: `scene-grimorio` foi
  junto (ALE-283). O `go build`, o `templ generate`, o `go vet`, a suíte Go
  inteira e o próprio guarda de tinta ficaram VERDES; quem acusou foi o e2e, com
  oito casos de leiaute e contraste. O `TestEveryScopeClassExistsInTheStylesheet`
  fecha o buraco, e ele lê só o que está dentro de `class=` — a primeira versão
  varria o arquivo inteiro e reprovava `scene-title`, `scene-shell` e
  `scene-content`, que são valores de `data-slot` e não classes.

  > **A lição maior é sobre renome em massa, e vale para a próxima varredura:**
  > `\bnome\b` alcança muito mais que identificador Go. Numa entrada só de mapa,
  > `grimorio` estragou TRÊS coisas de naturezas diferentes — os tokens da
  > paleta, a ROTA `/grimorio` (endereço que alguém favorita) e o nome do arquivo
  > `grimorio.js` no disco. Renome de símbolo Go se faz com `gopls rename`, que é
  > semântico; troca textual fica só para o que o `gopls` não alcança — o
  > componente `templ` —, e aí o alvo tem de ser um nome COMPOSTO e único.

## Como uma CENA é construída

As doze cenas saíram do `api` uma a uma entre a ALE-272 e a ALE-278, e o que
sobrou dessa travessia é um formato. A crônica de cada fatia está nas issues; o
que vale para a próxima cena está aqui.

**O formato é uma PORTA declarada pela cena.** A interface `Deps` mora em
`web/<cena>/deps.go`, e o `*api.Server` a cumpre. A direção é o desenho inteiro:
quem escolhe o que atravessa a fronteira é o CONSUMIDOR, não o objeto que tem
tudo. O `api` monta com `cena.Routes(r, cena.New(s.cenaHost()))`, e é nessa linha
que o compilador cobra quando a porta deixa de ser cumprida.

**O CASO DE USO não entra pela porta: ele entra por PARÂMETRO** (ALE-344,
ALE-347). Uma porta existe para desviar de um ciclo — a cena precisa do `api`,
que importa a cena —, e o `app/` está ABAIXO das duas: não há ciclo, então não
há interface a declarar. Três cenas já montam assim, e a assinatura diz o que
elas fazem:

```go
table.New(s.tableHost(), s.sessionLifecycle(), s.restParty(), s.initiativeQueue())
forge.New(s.sceneCore(), s.characterBirths())
sheetui.New(s.sheetHost(), s.characterPlays())
```

O efeito é a porta ENCOLHER em vez de crescer: a da ficha saiu de dezoito
métodos para oito quando os gestos dela viraram `character.Plays`. Uma entrada
que vira caso de uso SAI da `Deps` — ela não ganha um adaptador novo.

Cada `web/*` tem um `boundary_test.go` que recusa import do hospedeiro. Ele não
existe para pegar o ciclo — esse o compilador já pega —, existe para pegar o
atalho: um import DIRETO de `catalog` contornando a camada tipada, que não é
ciclo, não é erro, e é a divisão vazando por baixo.

### As regras da porta

- **Ela pede o MENOR tipo que resolve.** `CurrentUserID(r) int64`, e não o
  usuário inteiro: o tipo do usuário é do hospedeiro, e uma porta que devolve
  tipo do hospedeiro não é porta, é o hospedeiro com outro nome.
- **Ela pede a PERGUNTA, não o objeto.** Foi a lição que valeu para as dez cenas
  seguintes — `PodeEditar(...) bool` em vez de entregar o que permitiria decidir.
- **Quem CLASSIFICA o erro é o hospedeiro; quem escolhe a FRASE é a cena.** O
  texto que aparece na tela é da cena, sempre; a natureza do erro é de quem
  conhece o banco.
- **Ela é fina quando a cena não precisa do servidor**, não quando alguém foi
  disciplinado. Porta larga é sintoma de cena fazendo trabalho de outra — a
  pergunta é essa, não "dá para cortar um método?".
- **Conserta-se a porta, não o chamador.** Quando o teste do hospedeiro obriga a
  abrir um método, o que está errado é a porta.
- **Um contrato que já existe ganha da regra.** Quando a assinatura vem de uma
  interface de terceiro, ela manda.
- **O dublê da porta precisa RENDERIZAR.** Um dublê que devolve struct e não
  desenha prova que o handler chamou alguma coisa, e nada sobre a tela.

### Onde um pedaço de código mora

**O destino de uma função é a DEPENDÊNCIA dela.** Não é o assunto, não é quem a
chama hoje: é do que ela precisa. O `bcrypt` não atravessa para uma cena; um
ajudante que só fala de `sql.Null*` vai para o `dbvalue`; uma regra que lê o
livro vai para o `book`.

**A linha divisória dentro de um arquivo é a mesma que entre arquivos**, e
**o menor não é o mais estreito, é o que não se repete**: uma cópia de sete
linhas declarada no comentário dela custa menos que um método a mais na porta.

**A cena sai; a bancada que a exercita fica.** Mover a cena não move os testes de
integração dela — eles batem no roteador de verdade, que continua no `api`.
## Código órfão, bancada mentirosa, e raciocínio envelhecido

Três famílias que apareceram juntas entre a ALE-287 e a ALE-292, e que se
confundem porque as três produzem VERDE sobre nada. As investigações estão nas
issues; o que se aplica está aqui.

### Órfão é dívida OU capacidade esperando gesto

Uma função sem chamador pode ser das duas coisas, e **o que as separa não é a
contagem de chamadores — é ler o que a coisa FAZ.** Numa varredura de cinco
órfãos, três eram dívida e dois eram funcionalidade perdida: a camada de baixo
inteira no ar, com teste, e nenhuma rota chegando nela.

E a segunda metade da regra: **a capacidade esperando gesto pode NÃO precisar do
que a esperava.** Quando o gesto finalmente chegou, ele não usou nada dos três
órfãos guardados para ele. Guardar código por "vai ser útil" é apostar no desenho
que você ainda não fez.

### A bancada que escreve o que a produção não escreve

**Código alcançável por ninguém e teste que arranja o próprio estado são a mesma
doença**: os dois parecem cobertura.

- **Quando o ARRANJO de um teste usa uma porta que nenhum caminho de produção
  usa, o verde é sobre o arranjo.** Uma mesa inteira passava verde chamando um
  `Join` que ninguém em produção chamava desde a ALE-272.
- **Quando a bancada escreve uma coluna que a produção nunca escreve, o verde é
  sobre a bancada.** O fixture fornecia o que a tela não fornecia, e toda mesa
  aberta pela tela nascia com a coluna nula.

O controle é o mesmo nos dois casos: **percorra o caminho de produção para
montar o estado**, e não a porta mais curta que chega ao mesmo dado.

### O RACIOCÍNIO guardado envelhece como o código guardado

Uma docstring **estava certa quando foi escrita, continuou plausível por duas
épicas, e estava errada no dia em que alguém a leu para decidir.** É a mesma
família do `.md` que apodrece, e ela é pior: ninguém relê docstring de função que
não vai tocar.

Duas armadilhas de quem constrói a sonda que caça isso:

- **A leitura tem de usar a MESMA varredura que a trava.** Uma sonda que casa
  `.Simbolo(` perde todo nome que colide com a biblioteca padrão; casando também
  `Simbolo(`, ela mascara nome sobrecarregado. O resultado vira PISO, e o
  silêncio dele não é evidência.
- **Foi o controle positivo que reprovou a sonda**, não a leitura dela. Sem um
  caso conhecido plantado, a lista curta passaria por resposta.

### Guarda de leiaute precisa do ESTADO, não só da cena

A cena estava na lista de visitas e o ESTADO não: nenhum guarda visitava um mapa
COM marcador, então os ícones espremidos e o número de dois dígitos do acervo
atravessaram todas as medições. **Visitar a cena não é visitar a tela** — é a
mesma família que a raiz cataloga em "um guarda só mede o que ele VISITA", com o
dado no lugar da navegação.

> E uma armadilha de acessibilidade que saiu daí: **esconder texto por CSS o tira
> da árvore de acessibilidade**, então um rótulo escondido por `display:none` não
> é lido por ninguém — nem pelo leitor de tela, nem pelo `getByLabel`.
## Apagar código morto: as armadilhas do RECORTE

A ALE-277 e a ALE-278 apagaram 104 manipuladores sem chamador e repartiram o
`*Server`, que tinha 89 métodos exportados existindo para cumprir a UNIÃO das
portas das onze cenas — quando **67 das 76 assinaturas tinham exatamente UMA cena
pedindo**. Hoje ele tem onze. A crônica está nas issues; o que morde ao recortar
está aqui.

**O compilador pega três das quatro**, e a quarta é a que importa:

- Recortar função de UMA LINHA procurando `}` na coluna zero come as vizinhas.
- Receptor colide com variável existente no destino.
- `s.` dentro de uma string de formato vira recorte errado.
- **E a que ele NÃO pega: a regra que só aquele teste guardava.** Método sem
  chamador não quebra compilação, e teste que o exercitava some junto com ele.
  Antes de apagar, pergunte o que o teste afirmava — não se alguém chama.

**Cena se testa pelo `WebRouter()`**, não pelo `Router()` da API JSON. Foi o que
sobreviveu à saída da API: dois guardas que varriam rotas de `/characters`
morreram com o terreno, e a invariante mudou de casa para o
`TestNoSheetWriteAcceptsAStranger`. **Guarda de varredura vale o que vale o
terreno que ele varre.**
## `campaign`: a mesma regra recusando com DUAS frases

O que é um nome válido e o que é uma descrição válida saíram do `api` na
ALE-278, com a forma exata do `account` — e pelo mesmo defeito, que vale
registrar porque agora ele tem DOIS casos e um padrão.

A cena escrevia "O nome é obrigatório e cabe em 120 caracteres." e a rota JSON
respondia `err.Error()`, que era uma frase em inglês. Duas frases para uma regra
é o que quebra quando alguém mexe no limite: uma das duas fica para trás, e é
sempre a que ninguém está olhando.

**O padrão, agora com dois casos:** quando uma regra de produto tem um consumidor
de TELA e um de API, a frase divide antes da conta. No `account` a divergência
era a mesma (inglês na rota JSON, português na porta); aqui é idêntica. O que os
dois pacotes fazem é a mensagem morar COM a regra, em pt-BR, porque quem lê é
uma pessoa.

### A extração desfez um vazamento de TIPO

O `campaignDescription` devolvia `sql.NullString`. Uma regra de produto
carregando o tipo do banco é a fronteira no lugar errado, e o conserto é o que
também torna os dois caminhos iguais: a regra devolve TEXTO, quem grava traduz
vazio para NULL.

### O guarda afirmava uma garantia que ele não tinha

Escrevi na docstring dele que "`database/sql` é a tentação NOMEADA aqui".
Sabotei com `var _ = sql.NullString{}`: o build passou **e o guarda passou** —
ele só olhava `t20engine/*`, e `database/sql` é biblioteca padrão.

É literalmente o **comentário não é correção** do CLAUDE.md acontecendo dentro
de um guarda. Ele ganhou uma lista de recusa da biblioteca padrão com UMA
entrada, e a razão de ser uma só está escrita: esta é a tentação MEDIDA — a
versão anterior devolvia `sql.NullString` de verdade. **Lista de perigo
imaginado envelhece; lista de defeito acontecido, não.**

> A lição de instrumento é a mais transferível daqui: um guarda de fronteira que
> filtra por PREFIXO do módulo é cego para tudo que não tem esse prefixo. Se a
> prosa dele nomear uma tentação de fora do módulo, ela precisa de uma segunda
> lista — ou a prosa está mentindo.

## `account`: a regra de conta, e a cópia que divergiu na FRASE

O que é um e-mail, o que é uma senha aceitável, e a forma dos dois pedidos que
criam sessão. Ele saiu do `api` junto com a porta (ALE-278), e o motivo não foi
arrumação: eram DUAS cópias da mesma regra, e elas já tinham divergido.

O `api` tinha `validateRegister`/`validateLogin`/`validatePassword` com as
mensagens em pt-BR, e `ValidateRegister`/`ValidateLogin`/`ValidatePassword` com
as mesmas regras em inglês. **Não era código morto esperando limpeza**: a
`ValidatePassword` inglesa era chamada pela rota JSON que redefine a senha, e a
portuguesa pela tela da porta. A mesma regra recusava com dois textos, e um deles
na língua que a regra de idioma proíbe para o que um humano lê.

As outras duas grafias inglesas eram dívida de verdade — a `ValidateLogin` sem
chamador nenhum, a `ValidateRegister` com exatamente um: um teste, que afirmava
as frases inglesas. **Mudar o mínimo da senha na cópia viva deixava aquele teste
VERDE**, porque ele prendia a outra.

É a mesma forma do `search` e pelo mesmo motivo declarado lá — função pura
hospedada em pacote grande vira cópia na mão de quem não pode importar o pacote.
A diferença é onde a cópia errou: lá foi na conta, aqui na frase. As duas
compilam, e nenhuma das duas aparece num diff.

Ele NÃO vai para `plataforma` de propósito: aquele pacote é infraestrutura sem
domínio, e "a senha precisa ter ao menos 8 caracteres" é regra de PRODUTO.

## `search`: a busca vira pacote, e a cópia que quebrou o acento morre

O casamento e a pontuação de busca são 271 linhas puras — `strings`, `unicode` e
a normalização de acento — que moravam no `api` por história e não por
dependência. Elas viraram `search` na ALE-278.

**O que decidiu não foi a pureza, foi o histórico.** Quando o catálogo tipado
saiu para o `book`, ele precisou do `Fold` (que desacentua) e não podia importar
o `api`. Escrevi uma cópia de nove linhas e escrevi ERRADO — chamei a função que
só faz `ToLower` —, e `book.KeyOfName("Atuação")` passou a devolver "atuação" com
acento: a classe deixou de ligar a perícia que treina, sem erro, sem panic, sem
log.

Um pacote apaga a cópia E a razão de haver duas. O guarda de fronteira dele tem
lista VAZIA, e o comentário diz por que isso importa mais aqui do que na média:
no dia em que o `search` alcançar catálogo, banco ou HTTP, o próximo que precisar
do `Fold` vai copiar de novo — e a próxima cópia vai estar errada de outro jeito.

A regra tem guarda nas DUAS camadas, e é deliberado: `search.TestFoldDropsAccentsAndCase`
prende a função, `book.TestTheAddressKeyDropsAccents` prende o efeito no endereço.
Foi exatamente entre as duas que a cópia divergiu.

> Provar o guarda custou TRÊS sabotagens, e as duas primeiras foram inertes — uma
> não achou a linha, a outra virou erro de compilação ("declared and not used").
> Verde depois de sabotar só significa alguma coisa quando a sabotagem CHEGOU.

## `book`: o catálogo TIPADO, lido por treze famílias

A raça, a classe, a perícia, o deus, a condição, a magia, o poder, o item, a
origem, o efeito e a escola de magia moram em `book` desde a ALE-278 — mais os
leitores que os montam do `catalog` e o maquinário de ELOS, que transforma um
texto em citação clicável consultando o catálogo de condições.

Ele é a segunda camada compartilhada a sair, e a que de fato travava a divisão:
a forja tentou sair primeiro e não conseguiu porque precisa de `racaDoLivro` e
`classeDoLivro` para desenhar as cartas.

**O BESTIÁRIO chegou depois, e o atraso foi um defeito de FERRAMENTA.** O extrator
que separava as declarações tratava o parêntese de um bloco `var ( … )` como se
fosse RECEPTOR de método, e engolia a declaração seguinte — que era justamente o
carregador das criaturas. Ninguém percebeu porque o resultado foi um pacote que
compilava com uma coisa a menos, e só apareceu quando o buscador tentou sair e
esbarrou no verbete.

A lição é sobre a ferramenta e não sobre o bestiário: **um extrator que erra por
omissão produz um resultado que compila.** O que o teria pego na hora é o que o
guarda de tinta faz — contar quantos itens saíram e comparar com quantos foram
pedidos.

**O guarda de fronteira dele é o mais importante da série, e a razão é
aritmética:** treze famílias leem o livro, então quase todo pacote de cena que
nascer vai importá-lo. No dia em que ele importar o `api`, todas as cenas
alcançam HTTP de graça — com o guarda de cada uma continuando verde, porque cada
guarda só olha os imports dele.

### O defeito que a extração produziu, e como ele apareceu

O `book` é FOLHA, então ele não pode importar o `dobra` do `busca.go` — a função
que desacentua para comparação. Escrevi uma cópia de nove linhas, e escrevi
ERRADO: chamei o `dobraSimples`, que é só `ToLower`, achando que ele desacentuava.

O sintoma não parece o defeito. `KeyOfName("Atuação")` passou a devolver
`"atuação"` em vez de `"atuacao"`, e a consequência foi a classe deixar de LIGAR
a perícia que treina: um elo apontando para um endereço que não existe. Sem erro,
sem panic, sem nada no log.

Quem acusou foi um teste de CENA dois pacotes acima
(`TestTheClassLinksTheExpertisesItTrains`). A regra passou a ter guarda onde ela
mora, no próprio `book` — e ele nasceu vermelho por sabotagem.

**A lição para as próximas extrações: função copiada por causa de fronteira é
lugar de defeito silencioso.** Ela compila, tem o nome certo, e faz outra coisa.
Quando copiar, copie o CORPO do original — não uma reescrita de memória.

## `sheet` e `creature`: a ficha e a criatura, fora do `api`

O `CharacterDTO` e os seus seis irmãos moram em `sheet`; o bloco de criatura do
livro mora em `creature` (ALE-278). Eles saíram do `api` porque a medição da
divisão apontou para eles: **toda cena usa de 15 a 60 símbolos de outras
famílias**, e os DTOs estão em quase todas as listas — enquanto eles fossem do
`api`, nenhuma cena podia sair sem levar o `api` junto.

**A pergunta que decidiu foi "eles carregam HTTP?", e a resposta foi medida antes
de mover:** o `character_dto.go` importava `sqlcgen`, `engine` e `plataforma` e
não tocava `*Server` nem `http`; o `creature_block.go` importava `fmt` e
`strings` e mais nada. Os dois eram forma de dado hospedada por acidente de
história.

O que NÃO saiu junto e vale saber por quê: os quatro handlers do estado de jogo.
O arquivo que os hospedava misturava a FORMA (dois structs sem dependência) com
o encanamento que a grava — os structs viajam dentro do `CharacterDTO`, então
foram; os handlers ficaram. O que restava dele — zerar os usos "1/cena" e baixar
as posturas — desceu para o `app/rest` na ALE-344, e o arquivo deixou de
existir.

**E o `sheet` ganhou a CONSTRUÇÃO junto, na terceira camada.** `Load` monta o
agregado a partir das linhas do banco, `Compute` o passa pelo motor, e
`LoadAndCompute` é a soma — as três eram métodos do `api.Server` e viraram
funções com as dependências por parâmetro, porque o que elas usavam dele eram as
`queries` e os `catalogs` e nada mais. Oito cenas leem isso.

A medição que decidiu foi refeita DEPOIS das duas primeiras camadas, e ela disse
duas coisas. A primeira: das 36 coisas que as cenas usam da família `character`,
o núcleo é esse punhado. A segunda, que mudou o plano: **os treze arquivos
`character*.go` misturam handler HTTP com domínio** — nenhum é puro, então não há
arquivo a mover, só função.

> A primeira versão dessa medição disse que os treze eram limpos, e era mentira
> do INSTRUMENTO: `\\*Server\\b` dentro de uma f-string vira barra invertida
> literal e não casa com nada. "Nenhum toca HTTP" era a resposta que eu queria, e
> ela quase passou. A segunda versão tem CONTROLE — ela afirma primeiro que a
> sonda enxerga HTTP num arquivo que sabidamente o tem.

Os métodos do `Server` ficaram como invólucros de uma linha. Eles somem quando
cada cena receber as dependências dela por construtor; o que interessa agora é
que a lógica passou a ser alcançável sem o `api`.

Os dois pacotes têm guarda de fronteira, e a razão é a que o `events` já
documenta: **cada cena que se mudar vai importá-los.** No dia em que o `creature`
alcançar o catálogo, todas as cenas alcançam junto — de graça, e com o guarda de
fronteira de cada uma continuando VERDE, porque cada guarda só olha os imports
dele. A lista do `creature` é vazia; a do `sheet` tem QUATRO, e a quarta é a
única desta série que mudou o que um pacote é — o `book`, na fatia da ficha. Ver
a seção do `web/sheetui`, mais abaixo.

## `web/ui`: o kit de apresentação, e o que ele NÃO pode saber

O botão, o campo, a moldura, o rótulo de seção, a caixa rolável, o ícone e a
CASCA moram em `web/ui` desde a ALE-278 (fatia 4). Eles são o que 35 famílias de
arquivo liam do `api`, e sair de lá é o que permite as cenas se dividirem em um
pacote cada.

**A linha divisória não é tamanho, é DEPENDÊNCIA.** O que ficou no `api` foi o
agrupamento do LIVRO e dos ELOS — `aPaginaDoLivro`, `pedacoDoTexto`,
`eloParaOAcervo`, os dois diálogos — porque o `trecho` que eles desenham nasce de
uma consulta ao catálogo de efeitos e de escolas de magia. Levá-los faria o
pacote de apresentação importar catálogo, que é o contrário do que a divisão
existe para conseguir.

**A casca RECEBE o que não pode conhecer.** Duas dependências a prendiam ao
`api`, e as duas viraram campo de `ui.Page`:

- `Asset func(string) string` — o endereço versionado dos estáticos, que são
  `go:embed` do `api`;
- `Overlays []templ.Component` — o livro, o verbete e o buscador, que leem
  catálogo. A casca só reserva o lugar.

Quem preenche é o `scene_core.go`, o ÚNICO lugar do projeto que monta uma
página. Pôr esses campos em cada `ui.Page{…}` seria repetir dezoito vezes o que
não varia.

**A armadilha desta fatia foi o renomeador, e vale para a próxima.** Exportar um
símbolo é escrever um nome novo, então o kit inteiro passou para inglês — e um
`re.sub` por palavra trocou 250 arquivos, inclusive um comentário que virou "o
diálogo Int". Os nomes do kit são palavras COMUNS em português (`campo`,
`tamanho`, `variante`, `botao`, `layout`), e elas aparecem em comentário, em
string, em nome de parâmetro de outro componente e dentro de identificador
hifenizado — `data-nav-layout` virou `data-nav-ui.Layout`, que o HTML aceita sem
reclamar e que só custa a navegação por teclado daquela cena, em silêncio.

O que funcionou: um renomeador que pula comentário e string, mais o COMPILADOR
como rede para o resto — em Go, `ui.Field := …` não é declaração válida, então
todo parâmetro e toda variável local com nome colidente vira erro. O que o
compilador não pega é o hifenizado; esse se acha com um `grep` por `-ui.` depois.

### O `extra` do botão NÃO ganha da base, e quem decide é a FOLHA (ALE-316)

`ButtonClasses(v, t, extra)` concatena, e **concatenar não resolve conflito**: o
`Join` não é `tailwind-merge`. Duas classes do mesmo eixo na mesma tag não se
somam — vence a que a folha compilada declara por ÚLTIMO. Então um `extra` cuja utilidade venha ANTES da base perde,
e perde calado.

Medido nos offsets da folha de hoje:

| eixo | ordem na folha | consequência |
|---|---|---|
| fonte | `.text-base 58566` · `.text-lg 58664` · `.text-sm 58756` · `.text-xs 58940` | a MENOR sempre vence |
| `gap` | `.gap-1 41290` · `.gap-1.5 41316` · `.gap-2 41357` | o MAIOR vence |
| hover | `.hover:text-destructive-ink 72192` · `.hover:text-foreground 72256` | o `foreground` vence |

As três linhas produziram um defeito cada. A da fonte era latente: o primeiro
botão de ícone a pedir `text-lg` sairia pequeno. A do `gap` estava **VIVA** — o
`gap-2` da base vencia o `gap-1.5` do `SizeSmall`, e o botão pequeno nunca teve
o respiro que alguém escreveu para ele. A do hover é a razão de o ghost que
apaga ser VARIANTE (`VariantGhostDanger`) e não um `extra`.

**O conserto geral é de forma: nenhum eixo mora na BASE se um TAMANHO pode
querer outro valor.** A fonte e o `gap` desceram para o mapa de tamanhos, e aí
não há duas classes para disputar. O que sobra na base é o que nenhum tamanho
contradiz — forma, foco, transição, desabilitado.

Para quem for escrever a próxima variante: o teste é `grep -bo '\.classe'` na
folha compilada, dos dois lados do conflito. Não adianta raciocinar pela ordem
do `class=`, que o navegador ignora.

## O anel de foco é UMA receita, e o repouso dela existe para não animar (ALE-318)

A regra é global, mora no `index.css` desde a ALE-173 e **não é layerada** —
então ela ganha de todo utilitário do Tailwind, que é layerado. Consequência que
vale saber antes de escrever qualquer botão: `focus-visible:outline-*` escrito à
mão não faz efeito nenhum dentro de uma cena, porque a regra de cima já decidiu.
Eram **242 sítios** copiando a mesma tripla — 240 em 43 `.templ` e 2 em `.go` de
produção —, e apagar os 242 não mudou um pixel (ALE-317). A issue os contava como
147 porque a primeira medição olhou só `<button>`, e a tripla também estava em
`<a>`, `<input>`, `<summary>` e `<label>`: *uma medição parcial não é um número
menor, é um número de outra pergunta.* Quem impede o 243º é o
`TestNoHandwrittenFocusRing`.

O que MUDA o pixel é o REPOUSO, e foi só medindo que isso apareceu. O `@layer
base` traz o `* { border-color; outline-color }` do shadcn, e o `outline-color`
vinha a 50%; a largura vinha de `medium`, que o navegador computa como **3px**; o
afastamento vinha de `0`. Nada disso pinta — o `outline-style` em repouso é
`none`. Mas os três são o ponto de PARTIDA da transição, e o `transition-colors`
e o `transition-all` do Tailwind v4 incluem `outline-*`: o anel era ALCANÇADO em
150ms em vez de desenhado. Medido quadro a quadro num botão do kit: ouro a 0,50
no primeiro quadro, 0,73 aos 54ms, cheio aos 154ms — e quem tabula na
autorrepetição do teclado (~33ms por parada) nunca via o anel inteiro. O meio do
caminho mede 3,25:1 contra os 10,46:1 do final, raspando o piso de 3:1 do WCAG
1.4.11.

**O conserto é escrever o repouso igual ao destino**, no mesmo `*`: sem
diferença, não há o que interpolar. A duplicação do `2px`/`1px` com a regra de
foco é deliberada, e quem cobra que os dois lados não se separem é o
`e2e/tests/support/focus.ts`.

Três coisas que essa medição deixou, e nenhuma delas é sobre contorno:

- **Ler o computado durante uma transição devolve o valor de PARTIDA**, e
  escolher outro instante só troca o erro de lugar. O medidor da casa pergunta ao
  navegador se existe transição (`getAnimations()` devolve uma `CSSTransition`
  por propriedade, com o nome dela) em vez de inferir pela aparência. A história
  inteira está na seção "O INSTRUMENTO MENTE COM CARA DE RESULTADO" do
  [CLAUDE.md da raiz](../CLAUDE.md).
- **A carta de rádio é a única exceção legítima, e ela é do RÓTULO.** A forja e o
  "entrar na mesa" escondem o `<input>` com `sr-only` e desenham o realce no
  `<label>`, com `has-[:focus-visible]:outline-*`. A regra global não alcança —
  ela casa `:is(a, button, input, …)` —, e uma regra `label:has(:focus-visible)`
  no lugar dela desenharia DOIS anéis concêntricos nos dez rótulos do app que
  envolvem um campo VISÍVEL. As três plaquetas eram a segunda aparência de foco
  do repositório (afastamento de 2px), e quem as achou foi o medidor subindo a
  árvore a partir do rádio invisível — um sweep de focáveis nunca olha para lá.
- **Guarda que aproxima um seletor mede outro seletor.** A primeira versão do
  medidor isentava do anel tudo que estivesse dentro de `[data-nav-region]`,
  porque o trilho fala por brilho; o `index.css` isenta `[data-nav-region] :is(a,
  button, [data-nav-item])`. Os rádios `sr-only` das cartas e o contêiner rolável
  estão dentro do trilho e não são item dele — 37 nós acusados de uma vez, na
  folha da forja. A condição do guarda passou a COPIAR o seletor.

## As notas numa janela própria, e o pacto que as torna únicas (ALE-218)

As notas da sessão têm endereço:
`GET /campanhas/{campanha}/sessoes/{sessao}/notas` desenha o
MESMO painel da coluna, sem o mapa em volta. É o último dos quatro lugares que a
ALE-218 decidiu — lado a lado, empilhado, flutuando e a janela —, e o único que
sai do leiaute da página.

**Ela é cena e não painel mudado de lugar**, e isso foi medido e não escolhido:
o nó adotado por outro documento perde o Datastar (ver a armadilha do mesmo nome
na seção do Datastar). O corpo é reusado — `notesBody`, `modesRange`,
`autosaveState` são os mesmos —, e o que a cena NÃO tem diz o desenho: sem o
alternador de flutuar (não há mapa embaixo), sem a divisa de largura (quem
redimensiona é o sistema), sem o ✕ (fechar é a janela).

**As duas não podem estar abertas ao mesmo tempo, e a razão é o BANCO.** As duas
escrevem `sessions.notes` com autosave de 1,2s, e as notas não são região do
stream de propósito — são de um leitor só. Com as duas no ar, quem salva por
último apaga o parágrafo do outro, sem aviso, com as DUAS faixas dizendo "Salvo".

O pacto é `localStorage` mais o evento `storage`, e não um canal de difusão:
o evento chega a toda janela da origem sem ninguém segurar um objeto vivo, e é o
mesmo mecanismo que as outras três preferências das notas já usam. A chave
guarda o ID DA SESSÃO, porque uma janela aberta na sessão 4 não tem o que dizer
sobre a 7. Três detalhes que custaram pensamento:

- **Quem fecha a coluna é o ANÚNCIO da janela, não o clique.** A diferença
  aparece quando o navegador bloqueia o pop-up: ali nada foi tomado, e a coluna
  fica onde estava em vez de sumir sobre uma janela que não abriu. O bloqueio é
  dito em PALAVRAS — `window.open` devolvendo nulo é a única pista que existe.
- **`pagehide` e não `beforeunload`**: aquele pede confirmação em alguns
  caminhos, e `unload` é pulado quando a página vai para o cache de ida e volta.
- **A chave pode ficar PRESA** se a janela morrer sem `pagehide` (uma queda do
  navegador). O conserto é o próprio gesto: com as notas "numa janela", o botão
  do trilho chama `window.open` com o mesmo NOME — que acha a janela viva ou
  abre outra. O estado preso se desfaz clicando onde a pessoa já ia clicar.

**E o `window.open` entrou no guarda de endereço** (`TestEveryAddressAPostWritesExistsInTheRouter`,
ALE-308) em vez de ganhar um próprio: o defeito é o mesmo que o do `@post` com
caminho morto, e o extrator já sabia parar na primeira vírgula de topo, então o
nome da janela e as `features` ficam de fora sozinhos. O método é GET porque é
navegação — e isso não é detalhe: perguntar ao chi por um POST em
`/campanhas/1/sessoes/4/notas` responderia "existe" pela rota de SALVAR, e o guarda ficaria
verde sobre um endereço de página que não existisse.

## Onde a coordenada de um gesto do tabuleiro viaja

**No CORPO, e não no caminho** (ALE-305). O `@post` do Datastar aceita
`{payload: …}`, e o payload é calculado no instante do clique — não vira sinal,
que é estado compartilhado e viajaria em toda requisição. O formato é um só para
o tabuleiro inteiro:

```json
{"from":{"X":2,"Y":2},"to":{"X":8,"Y":5}}        // o traço e o retângulo
{"kind":"dificil","erase":true,"from":…,"to":…}  // com a espécie do pincel
{"shape":"esfera","size":"6","from":…,"to":…}    // o gabarito: origem e mira
{"from":{"X":7,"Y":3}}                           // um lugar só: parada, marcador
{"delta":{"X":3,"Y":-2},"marked_tokens":"a,b"}   // o arrasto do grupo marcado
```

As chaves são INGLESAS porque campo JSON é fronteira; só a ROTA saiu dessa lista
(ver o `CLAUDE.md` da raiz). Um tipo só — o `strokeBody` — para todos os gestos,
porque um formato só é um formato só para aprender; um por gesto é como nasce a
terceira grafia do mesmo par de números.

### O que o corpo quebrado produz, e as DUAS famílias de recusa (ALE-311)

O corpo que chega quebrado de verdade é `{"from":{"X":undefined}}` — o `payload`
é calculado no instante do gesto, e um sinal indefinido no meio da expressão
manda `undefined`, que não é JSON. O que não pode acontecer é o servidor decidir
sozinho que o gesto foi na origem.

**As nove rotas que lêem pontos se dividem em duas famílias, e o contrato de
recusa é DIFERENTE em cada uma:**

- as que respondem **só sinais** (`marcar-area`, `gabarito`, `regua`) recusam em
  **400**, porque não há cena para redesenhar;
- as que são **comando** (terreno, retângulo, peça avulsa, grupo) recusam em
  **200** com a frase no `$command_error` — porque o Datastar DESCARTA o remendo
  de toda resposta não-2xx, e uma recusa em 4xx ali não apareceria na tela.

A primeira versão do guarda reprovou quatro rotas por medir o STATUS, que é o
contrato errado para metade delas. **O que as duas famílias têm em comum é a
única coisa que importa para quem está na mesa: a frase CHEGA** — e é por isso
que o `TestEveryGestureThatReadsPointsRefusesABrokenBody` prende a frase.

> **Corpo VAZIO não é recusa, e isso é desenho.** `{}` decodifica para (0,0) em
> silêncio, porque `from` ausente e `from` em (0,0) são indistinguíveis num
> struct de inteiros. Prender isso exigiria ponteiro em todo campo de coordenada.
>
> A consequência para quem escreve teste é a que morde: **(0,0) é o VALOR-ZERO,
> então um caso ancorado na origem não consegue detectar um `from` que parou de
> ser lido.** O `TestTheEraserStrokeClearsTheWholeSegment` apagava de (0,0) a
> (6,6) e era estruturalmente incapaz de medir o que veio medir. Todo caso novo
> sai da origem de propósito, e vários afirmam que (0,0) NÃO foi tocado.

### Quando o corpo JÁ É o formulário, o payload lista os sinais

**`/pecas/nova` também foi** (ALE-306), e o caminho dela é o que ensina.

Eu a deixei de fora na ALE-305 dizendo que era impossível: *"o `payload`
SUBSTITUI os sinais, então a peça nasceria sem nome"*. A primeira metade é
verdade e a conclusão não segue — **o payload pode carregar os sinais**, basta
nomeá-los:

```js
@post('…/pecas/nova', {payload: {from: {X: cx, Y: cy},
  new_token_name: $new_token_name, new_token_size: $new_token_size,
  new_token_look: $new_token_look}})
```

O obstáculo real era outro, e mais raso: **o corpo não se lê duas vezes.** O
`ReadSignals` do datastar-go copia `r.Body` inteiro num buffer, então um segundo
leitor pega vazio. O conserto é um struct só, lido uma vez — e some o leitor
separado.

**O preço, que é o que decide:** a expressão passa a listar cada sinal pelo nome,
uma grafia a mais num lugar que um `grep` de `$nome` não distingue de leitura
qualquer. Quem paga é o `TestEveryPayloadKeyMatchesTheSignalItReads`: quando a
chave é ela mesma um nome de sinal, ela tem de carregar aquele sinal — então
`new_token_size: $new_token_look` reprova, e `kind: $tool` passa, porque `kind`
não é sinal nenhum.

#### O guarda tinha três cegos, e o denominador contava duas vezes (ALE-310)

- **Janela de 400 CARACTERES.** Ela sangrava de um `payload:` para o seguinte, e
  os "nove pares" que o guarda reportava eram OITO distintos, com `marked_tokens`
  contado duas vezes. Hoje a janela é o objeto `{…}` casado por BALANÇO DE
  CHAVES.
- **O piso era sobre PARES.** `pares < 1` sobre nove: trocar `payload:` por
  `payload :` — que o JavaScript aceita — em três sítios derrubava a conta para
  UM e o guarda passava. Ele só afirmava "o regex ainda casa em algum lugar do
  repositório". Hoje o piso é sobre SÍTIOS de payload, que são quinze, e a âncora
  aceita o espaço.
- **Payload que é VARIÁVEL.** `{payload: traco}` contribuía zero pares em
  silêncio. Hoje o `const traco = {…}` da linha de cima é resolvido, e o spread
  (`{...traco, kind: $tool}`) traz as chaves de origem junto.

**E o TERCEIRO CANAL foi fechado.** O guarda prendia `chave == $sinal` — o lado
do CLIENTE — e nada prendia `chave == o que o servidor lê`. Uma chave renomeada
de um lado só chega ao servidor e cai no chão: o gesto responde 200 com o
valor-zero, sem erro em lugar nenhum. Agora toda chave de payload tem de ter uma
tag `json:"chave"` ou um campo exportado de mesmo nome.

> **E o "vermelho contra a árvore de ontem" deste guarda nunca foi detecção.**
> Rodado contra a árvore pré-307 — a que TINHA o defeito de coordenada no
> caminho — ele passa, com 13 sítios e 6 pares casados. O FAIL que ele produz nas
> árvores mais antigas é o PISO tropeçando, porque o piso é calibrado para a
> contagem de hoje. Um piso calibrado falha em qualquer árvore anterior,
> inclusive numa perfeitamente sadia: **"vermelho contra ontem" só é evidência
> quando o motivo do vermelho é o defeito, e não o denominador.**

#### A CAIXA da chave conta, e ela vinha errada (ALE-313)

O cliente escrevia `{X: cx, Y: cy}` e o `engine.Square` declara `json:"x"` e
`json:"y"`, em MINÚSCULAS. Isso funcionava **por acidente**: o `encoding/json`
casa campo sem diferenciar caixa **só quando não há correspondência exata**.

É o mesmo mecanismo da armadilha do camelCase em nome de sinal, que esta casa
já pagou no construtor de encontros — duas grafias para um conceito, seguradas
por uma tolerância da biblioteca. O modo de falhar também é o mesmo: no dia em
que o struct ganhar um campo com correspondência EXATA para `X`, a exata vence e
a coordenada pousa no campo errado, sem erro em lugar nenhum.

**Qual lado muda foi decidido pelo DADO, e não por gosto.** O tabuleiro gravado
em `campaign_places` e `open_boards` carrega `"x": 3, "y": 0` — minúsculo. Mudar
as tags do servidor reescreveria todo lugar do acervo; mudar o cliente é
expressão. Foram 34 chaves em 13 linhas de `web/table`, todas dentro de literal
de string — os `boardSquare{X: …}` em volta são campo de struct Go e ficam.

Quem cobra é o `TestEveryPayloadKeyMatchesTheSignalItReads`, que passou a exigir
a grafia EXATA: **a tag GANHA do nome do campo**, porque um campo tagueado não se
lê pelo nome dele. Somar os dois faria o guarda aceitar `X` num campo que só
responde por `x`.

> **AQUI O NAVEGADOR NÃO É TESTEMUNHA, e é o inverso do achado da ALE-311.**
> Lá, trocar `from` por `origin` deixava a suíte de Go verde e o Playwright
> acusava com três falhas. Aqui não: devolvida UMA linha para `{X: …}`, o
> `go test ./api/` passa, o app FUNCIONA e o Playwright passaria também —
> porque a tolerância de caixa do `encoding/json` é exatamente o que faz a
> grafia errada continuar funcionando. **Um defeito que só existe como risco
> latente não tem testemunha em tempo de execução; quem o prende é o guarda de
> TEXTO**, e é por isso que ele existe.

> **E o guarda contaminou a si mesmo com a prosa que o explica.** Ao apertar a
> regra, `x: $rect_to_x` passou a reprovar como "chave que é sinal e carrega
> outro" — porque a colheita de sinais lia o corpo CRU dos arquivos, e o
> `data-show="$x"` que o cabeçalho do `overlay_flash_test.go` usa como EXEMPLO
> entrava na lista como se fosse sinal da árvore. O mesmo arquivo já tirava
> comentário na varredura de payload e não na de sinais.

### DESLOCAMENTO é coordenada, e o guarda não sabia disso

**`/grupo/mover/{dx}/{dy}` também foi** (ALE-307), e ela é a que denuncia o
guarda. O guarda de rota nasceu na mesma issue, para impedir que a família
voltasse a escapar — e ele passou VERDE por cima dela, porque eu escrevi o padrão
com os nomes que tinha na frente (`x`, `y`, `x2`, `mx`) e o arrasto do grupo chama
os seus de `dx` e `dy`.

É a terceira vez nesta família que uma varredura mede a grafia comum e a incomum
sobra: a `colar` escapou de um `grep` por `base+"…"`, o `/pecas/nova` escapou por
um argumento errado, e agora o guarda que existia para fechar as duas escapou
pelo NOME do parâmetro. **O que faz de um número "coordenada" não é ele ser
absoluto — é ele vir do PONTEIRO**, e por isso o endereço só se escreve
concatenando (`'…/grupo/mover/' + dx + '/' + dy`), que é a forma que esta seção
inteira existe para apagar.

A conversão levou junto **cinco leitores mortos**: `urlRect`, `tracoDaURL`,
`quadradoDaURL`, `urlSquareSecond` e `quadradoDoCaminho` — 86 linhas lendo
`chi.URLParam(r, "x")` com nenhuma rota registrando `{x}` desde a ALE-305, e
nenhum chamador. Função de pacote sem uso não é erro em Go; quem as achou foi
perguntar quem chama, e não o compilador.

#### E o guarda foi REFEITO, porque ele era duas coisas estreitas (ALE-310)

Ele morava na `convention` e lia o CÓDIGO-FONTE, procurando oito literais de
coordenada na mesma LINHA de um `r.Get|Post|…(`. As duas metades falhavam, e as
duas falhas são reusáveis:

- **Lista de PROIBIDOS subconta em silêncio.** Nove grafias alternativas
  sabotadas passaram verdes — entre elas `{col}`/`{lin}`, que são os nomes que o
  `board_view.go` escreve em toda peça, e `{cx}`/`{cy}`, que são os nomes que o
  cliente usa hoje. Hoje é uma lista de PERMITIDOS em
  `api/testdata/route_params.txt` com os 39 parâmetros da árvore, e parâmetro
  novo REPROVA até alguém escrever a linha — que é o ato de declarar que ele não
  vem do ponteiro.
- **Parser que lê a LINHA não lê o ARGUMENTO.** Quatro portas ficavam de fora,
  três delas forma que este repositório usa: a coordenada declarada no `base :=`
  (dominante em `web/table`, 30+ rotas), o registro quebrado em duas linhas, e o
  `r.Route`/`r.Handle`, fora do conjunto de verbos. Sabotando um `r.Route` o
  guarda passava **e o denominador SUBIA**, porque ele contava o `r.Post` filho e
  ignorava o pai.

**O conserto não foi um parser melhor: foi perguntar ao ROTEADOR.** O `chi.Walk`
devolve o padrão já montado — o `base :=` juntado, o pai concatenado com o filho,
a continuação lida como uma linha só —, e não há linha para ler errado. O guarda
mudou de casa junto, para `api/`, porque é lá que o roteador se monta.

E o denominador passou a ser EXATO: **212 rotas de verdade**, contra as 193 que o
regex estimava. Aquelas 193 incluíam dez linhas de `r.Header.Get(` — sete delas a
MESMA linha (`if r.Header.Get("datastar-request")`) que toda cena nova copia, o
que fazia o piso crescer a cada cena, na direção que o afrouxa.

### O PASSO da ficha FICA no caminho, e a razão não é a que estava escrita

`/personagens/{id}/vitais/{qual}/{passo}`, `/nivel/{classe}/{passo}` e
`/poderes/classe/{classe}/{escolha}/{valor}` continuam com o dado no caminho.
**Não por descuido — por medição.**

A razão que estava escrita no `web/sheetui/routes.go` era *"o valor é do botão
que foi clicado, e não de um sinal da página que quatro botões disputariam"*, e
ela é o mesmo não-sequitur do `/pecas/nova`: o `payload` é calculado por
chamada, então quatro botões não disputam nada — cada um escreve o seu literal.

**O obstáculo real é o REDESENHO.** Toda mutação da ficha passa pelo
`sheetCommand`, que lê os sinais UMA vez e os entrega ao `s.Load(…)` que redesenha
a cena inteira. São 29 campos: as buscas, os filtros, os aumentos, o construtor
de item. Como o `payload` SUBSTITUI os sinais, um `{step: -1}` chegaria com os
outros 28 zerados — e a tela voltaria sem o filtro que a pessoa estava usando.

Medido, postando `/vitais/pv/-1` com e sem os sinais no corpo:

| aba | com os sinais | só o passo |
|---|---|---|
| `expertises`, filtrando por "Perce" | 19.091 bytes, acha o termo | 24.122 bytes, lista inteira de volta |
| `abilities`, filtrando por "Ataque" | 23.247 bytes, acha o termo | 23.389 bytes, filtro perdido |

Converter custaria listar os 29 sinais no payload de cada botão de passo — o
preço da seção anterior, multiplicado por vinte e nove. **A divisão que vale não
é "caminho para o que o gesto decidiu, corpo para o estado"**: é que uma
coordenada vem do ponteiro e obriga a concatenar o endereço, e um passo é um
literal que o `templ` escreve na renderização, num endereço constante por botão.


## Datastar: doze armadilhas que não deixam erro para trás

As três primeiras foram descobertas na ALE-203, a quarta na ALE-205, a quinta na
ALE-235, quatro na ALE-272, a décima na ALE-275, a décima primeira na ALE-296 e a
décima segunda na ALE-218; nenhuma delas escreve uma linha no console — a oitava
escreve UMA, e no lugar que ninguém olha. Estão aqui porque o sintoma de cada uma
aponta para o lugar errado.

### `data-show` esconde TARDE: o nó pinta antes de o Datastar chegar

O `data-show` só é avaliado quando o runtime carrega e processa o DOM. Até lá o
nó está no documento com as classes que ele tem — e se elas o fazem visível, ele
PINTA. Um diálogo com `fixed inset-0 bg-black/60` cobre a janela inteira e
depois some.

**Medido** (ALE-296), com uma sonda de `requestAnimationFrame` instalada antes de
qualquer script da página: 30 quadros nas Perícias, 13 na Mochila, 10 no Combate,
9 nas Magias. A 60fps é meio segundo de pano preto, e o dono relatou como *"um
dialog de 1 frame que some"* — a impressão de quem vê subestima, porque um
piscar não se cronometra a olho.

**O conserto é `style="display:none"` estático ao lado do `data-show`**: ele é
lido na análise do HTML, antes de qualquer script, e o Datastar o troca por vazio
ao mostrar. **Não confundir com o `data-attr:style` da armadilha seguinte** — a
diferença é que aquele reescreve o atributo a cada avaliação e este é lido uma
vez.

Eram DEZESSEIS nós: seis sobreposições (uma delas o `templ overlay` da casa, com
dez call sites) e dez conteúdos escondidos por sinal puro. O
`TestNoDataShowNodeIsBornVisible` varre as duas formas, e a segunda é a que
decide sozinha: `data-show="$x"` sem negação e sem operador quer dizer "escondido
até ficar verdadeiro", e todos os sinais assim deste repositório nascem `false`.
Ele erra para o lado seguro — nascer escondido e aparecer um quadro depois é
sempre melhor que pintar e sumir.

### `ReadSignals` vem ANTES do `NewSSE`, e a ordem errada passa VERDE no teste

O `NewSSE` ASSUME a resposta e fecha o corpo do pedido. Um `ReadSignals` depois
dele encontra corpo fechado e o gesto chega sem sinal nenhum — o `datastar-go`
chega a perguntar de volta *"are you sure you created the SSE ***AFTER*** the
ReadSignals?"*.

**O que faz esta valer uma seção é como ela escapa:** o `httptest.NewRequest`
não reproduz esse ciclo de vida, então **o teste de handler passa VERDE** e o
defeito só aparece no navegador. E num `GET` ela não morde — o corpo é vazio dos
dois jeitos —, o que faz o defeito nascer no dia em que alguém trocar o gesto
para `POST`, longe de onde a ordem foi escrita.

O corolário: **o corpo do pedido se lê UMA vez.** O `ReadSignals` copia o
`r.Body` inteiro num buffer, e um segundo leitor pega vazio — sem erro.

### `data-show` + `data-attr:style` no MESMO nó CONGELA a aba

O `data-show` escreve `el.style.display`; o `data-attr:style` reescreve o
atributo `style` INTEIRO, apagando o `display` que o outro acabou de pôr — que
faz o outro pôr de novo. O renderizador entra em laço.

O sintoma é o pior possível: a aba para de responder a TUDO. Sem console, sem
exceção, sem conseguir sequer navegar para fora — a ferramenta de medir some
junto, e o que sobra é "o navegador travou", que não aponta para nada.

**Quem ESCONDE é um nó, quem POSICIONA é outro.** O
`TestNoNodeHasDataShowAndDataAttrStyleTogether` varre o HTML servido e recusa a
combinação.

### O sinal é um PROXY: ler um índice que não existe o CRIA

`$lista[0]` não é "o primeiro item": o Datastar registra o caminho e o cria
vazio. Com uma reserva de doze rótulos no ar, `$reguapontos[i]` encheu o sinal
de strings vazias entre os pontos de verdade — pingos na origem do plano e o
servidor medindo zero.

**Guardar o sinal numa constante NÃO resolve** (a constante continua sendo o
proxy). O que resolve é COPIAR: `const lista = [...$reguapontos]`. O
`TestNoExpressionIndexesTheListSignal` afirma a regra pelo que PODE vir
depois de `$lista` — `=` ou `]` —, e não por uma forma errada conhecida: a
primeira versão dele procurava `$lista[` e passava verde sobre a segunda forma.

### Sequência de comandos NÃO cabe num ternário

`evt.shiftKey ? (stmt; stmt) : (stmt; stmt)` é erro de SINTAXE em JavaScript. O
Datastar engole o erro de parse e o handler inteiro vira nada — não só o ramo
novo: o gesto que já funcionava para junto. Use `if (…) { … } else { … }`.

E `setPointerCapture` vai por ÚLTIMO na expressão: ele LANÇA quando o ponteiro
não está mais ativo, e no meio ele engole o resto do gesto.

### O servidor escrevendo num SINAL pelo stream: uma vez, nunca a cada quadro

O stream da Mesa remenda HTML, e desde a ALE-205 ele também escreve num SINAL —
`PatchSignals` — num caso só: o "mostrar à mesa" leva quem foi puxado para a
superfície do Tabuleiro, porque a superfície é sinal do navegador e o servidor
não a alcança de outro jeito.

**A armadilha é a cadência, e ela não parece uma armadilha:** o estado do puxão
vive enquanto a pessoa não escolhe outra aba, então a leitura ingênua manda o
sinal em TODO quadro. O resultado é uma trava disfarçada — quem tenta voltar
para a Mesa é devolvido ao mapa um segundo depois, para sempre, sem erro em
lugar nenhum, e conclui que o botão está quebrado.

O que resolve é a memória do que já foi empurrado ser da CONEXÃO (uma variável
do laço do stream), e não do servidor: duas abas da mesma pessoa merecem o
empurrão cada uma. Vale para qualquer sinal que o servidor venha a escrever
daqui: **remendo de HTML é idempotente, remendo de sinal não é** — o HTML
descreve o estado, o sinal muda a decisão de quem está do outro lado.

### Animação de ENTRADA numa cena que nunca monta: a classe substitui o mount

Nas cenas desenhadas pelo servidor — campanhas, personagens, e toda cena com
cursor — **nada nunca monta**: o servidor manda todos os itens e o cursor só
alterna `data-show`. Então não há remontagem de nó para disparar `animate-in`.

**O que substitui o mount é a CLASSE entrando num nó que não a tinha** (ALE-235).
O item que sai perde a classe, o que entra ganha, e são elementos DIFERENTES —
então não existe o caso que não replica, que é "a mesma animação, já concluída,
no mesmo nó". Dispensa morph, reflow forçado e id que muda a cada troca, que são
as saídas que a issue previa e que ninguém precisou escrever.

Duas coisas para quem repetir a receita:

- **O gesto que move o cursor tem de ser IDEMPOTENTE.** Um clique num item do
  trilho dispara `focusin` E `click`, os dois com a mesma expressão. Se ela
  calcula alguma coisa a partir do estado que ela mesma escreve (a DIREÇÃO, a
  partir do índice anterior), a segunda passagem lê o valor já atualizado e
  produz o resultado errado — sempre o mesmo, silenciosamente. A guarda é um
  `if` no começo da expressão.
- **E isso não aparece numa sonda que clica por `element.click()`**, porque ela
  não move o foco: só o gesto de verdade dispara os dois eventos. Mesma família
  do evento de ponteiro sintético, logo abaixo.

**Que a receita é GERAL, e não um remendo de uma cena, foi medido na ALE-297**:
as campanhas largaram o livro de couro, viraram palco e herdaram a entrada
inteira — as duas classes, o atraso da placa e o movimento reduzido — **sem uma
linha nova de CSS**. O que a segunda cena custou foi outra coisa, e ela vale
mais que a receita: enquanto o vizinho, o facho, a entrada e o gesto eram
privados de `web/characters`, eles tinham exatamente um chamador POSSÍVEL.
Copiá-los seria a saída óbvia e teria produzido duas gramáticas de cursor
divergindo em silêncio; hoje moram em `web/ui/stage.go` e o guarda de gesto
varre as duas cenas.

### O `@post` que redesenha a CENA precisa carregar o estado que está na URL

Um comando do Datastar responde com um remendo da cena inteira, e o handler
descobre o que desenhar lendo a própria requisição. O que está na URL da PÁGINA
— `?tab=`, `?aba=`, `?entrada=` — **não vai junto**: o `@post` manda o endereço
que está escrito nele, e mais nada.

O sintoma não parece um bug de estado. Na ficha, mexer no PV com a Mochila aberta
devolvia a cena desenhada na PRIMEIRA aba, porque o resolvedor não achou `tab` na
query e caiu no padrão — a tela parecia ter se fechado sozinha. Nada falha, nada
loga: o servidor desenhou uma cena perfeitamente válida, só que de outra seção.

**O remédio é o servidor escrever o `?` no comando**, já que é ele quem sabe o
estado ao renderizar o botão: uma função só monta todo `@post` da cena
(`sheetPost`), e um guarda de varredura lê o HTML de cada aba e falha se algum
comando sair sem ele (`TestNoSheetCommandLosesTheTab`). Sinal do cliente
resolveria também, e é pior: some no F5, que é justamente o que o endereço na URL
existe para sobreviver.

Ele ficou ESCONDIDO uma fatia inteira porque todas as abas desenhavam o mesmo
aviso de "ainda vive na ficha antiga" — o salto não tinha aparência. O primeiro
painel de verdade o denunciou no primeiro clique da bancada.

### Dois pedidos de UM gesto: quem CHEGA por último manda

O clique do mouse **também foca**. Um nó que pede ao servidor no foco e no
clique manda dois pedidos por um gesto só, e se os dois remendam a mesma cena
eles disputam os sinais que ela redeclara a cada remendo — a ordem de CHEGADA
não é a de saída, e quem chega por último ganha.

No bestiário o pedido do foco existe para a prévia da seta e não leva `abrir=1`.
Chegando por último, ele fechava a ficha que o clique tinha acabado de abrir: a
criatura ficava escolhida, o diálogo não aparecia, e nada falhava em lugar
nenhum. **O CI pegou duas vezes seguidas o que a bancada nunca reproduziu** — a
máquina rápida entregava as respostas na ordem de saída, e a carregada não.
Reproduzido de propósito atrasando só a resposta do pedido sem `abrir`.

É a mesma família do `data-show` com `data-attr:style`: duas escritas no mesmo
lugar sem ordem garantida. A diferença é que esta atravessa a rede, então ela
some da bancada e mora no CI.

**O remédio é não mandar o segundo pedido:** pedido disparado por FOCO é
afordância de TECLADO, e por isso ele pede `:focus-visible`
(`el.matches(':focus-visible') && (…)`). Medido no navegador: o clique dá
`false`, o Tab dá `true`, e o foco PROGRAMÁTICO do driver de setas também dá
`true` — a prévia da seta fica inteira e o mouse deixa de mandar o pedido que só
desfazia o dele. `TestNoFocusAsksTheServerWithoutAKeyboardGuard` varre a FONTE
inteira, e não uma cena servida, porque enumerar cena por cena deixaria a
próxima nascer sem medição.

**E "a fonte inteira" precisou ser defendida na ALE-278**, porque ele quase
deixou de ser verdade sem ninguém mexer numa linha dele. O guarda morava no
pacote do bestiário e varria `*.templ` do PRÓPRIO diretório — o que era a fonte
inteira enquanto todas as cenas eram um pacote só. Quando o bestiário virou
`web/master`, mudá-lo de casa junto o teria deixado medindo QUATRO arquivos e
ignorando três (campanhas, o tabuleiro da mesa e personagens), com o terminal
dizendo verde: dos quatro `.templ` com `data-on:focus`, um foi com ele e três
ficaram. Ele foi para o `convention/` e passou a CAMINHAR a árvore, que é o
mesmo conserto que o guarda de tinta levou — e ganhou um piso de arquivos
VISITADOS, porque o denominador antigo (`achados > 0`) teria passado verde na
mudança: havia um foco com `@get` no diretório novo, e ele bastava.

### Resposta que não é 2xx: o remendo é DESCARTADO e a recusa some

O cliente do Datastar não aplica o remendo de uma resposta de erro. Um handler
que responde `http.Error(w, msg, 400)` — que é o certo numa API JSON — deixa a
tela EXATAMENTE como estava: o gesto não acontece, nada muda, e a única marca é
uma linha no console do navegador ("Failed to load resource: 400").

Isso atravessou três fatias da ficha sem ninguém ver, porque as recusas eram
raras e pareciam "o botão não fez nada". O caso que denunciou foi gastar mais
dinheiro do que se tem: o diálogo fechava, o saldo continuava igual, e não havia
uma palavra na tela.

**Numa cena servida, a recusa é CONTEÚDO.** Ela volta 200 com a cena inteira
redesenhada — que é o que mostra que nada mudou — mais a frase num `role="alert"`
(`sheetCommand` + `sheetui.View.Recusa`). A consequência para os testes é a parte
que importa: o status deixou de distinguir "gravou" de "recusou", então **o que
os guardas afirmam é a FRASE**, com `sceneRefusal`. A API JSON continua com os
status dela; quem desenha página responde página.

**E o caso EXTREMO desta armadilha é o 404: um endereço que não existe.** O
Datastar descarta o remendo, o handler nunca roda, e não há nem a recusa nem a
cena redesenhada — o gesto simplesmente não acontece. Foi assim que o `‹` da
mesa viveu morto uma issue inteira: a ALE-304 traduziu as rotas para português,
o botão continuou postando em `initiative/previous-turn`, e ninguém viu porque
não há o que ver.

O que tornava o defeito invisível ao `grep` é que o endereço era montado em duas
metades — `"initiative/"+route` no helper, `"previous-turn"` no chamador —, e
nenhuma das duas é um caminho que se possa procurar. **O endereço só existe
RESOLVIDO**, depois que o `templ` juntou a base, o id e o verbo; ler isso do
código-fonte é o parser que a ALE-307 já mostrou não saber ler `base :=`.

Quem cobra é o `TestEveryAddressAPostWritesExistsInTheRouter` (ALE-308), e ele
RENDERIZA: tira todo `@post`/`@get` do HTML servido de 29 cenas e pergunta ao
chi com `Match` se a rota existe. `Match` e não um pedido de verdade, porque um
POST em `/personagens/spliced` levaria 404 do HANDLER ("personagem não existe")
e o guarda leria isso como rota faltando.

**Duas coisas nele valem para quem escrever o próximo guarda de cena.** A
primeira: ele precisa da mesa VIVA. O `‹` nasce `disabled` quando `PodeAvancar`
é falso (`SceneActive && len(Initiative) > 0`), então na bancada recém-montada o
endereço morto não está no HTML — a primeira versão passou VERDE sobre o defeito
que ela veio pegar. A segunda: expressão que o extrator não souber resolver
REPROVA, em vez de sair da conta em silêncio.

### `contentType: 'form'` valida o formulário ANTES de mandar

`@post(url, {contentType: 'form'})` manda o `<form>` mais próximo em vez dos
sinais, e é o que permite uma cena servida não ter sinal nenhum: os controles
já são o estado, e uma segunda cópia deles num `data-signals` só cria a pergunta
de qual das duas vale. A folha da forja (ALE-272, fatia 9) é assim inteira — o
`@post` que redesenha o equipamento e o `submit` que forja leem o MESMO
formulário, pelo mesmo `r.ParseForm()`.

**A armadilha é a linha `if (!form.noValidate && !form.checkValidity())`**, que
o cliente roda antes de qualquer coisa: com um campo `required` em branco ele
chama `reportValidity()` e **não manda o pedido**. Na forja isso apareceu como o
primeiro clique numa carta de classe não fazendo nada além de abrir o balão
"preencha este campo" — e a pessoa escolhe a linhagem antes de batizar o herói,
então o balão aparecia sempre.

O conserto não é `novalidate`, que desligaria a validação nativa do `submit`
também: é **o campo não ser `required` no HTML** e a recusa ser do servidor, que
já é a autoridade. Vale a regra geral: **num formulário que também é remendado,
validação nativa de campo brigará com o redesenho.**

### `data-on-signal-patch` SEM filtro escuta o remendo dos OUTROS sinais

O stream pode acordar um pedaço da tela escrevendo um sinal, e quem reage é o
`data-on-signal-patch`. É assim que a ficha dentro da sessão se mantém em dia
(ALE-275): ela não é região do stream — sete painéis computados a cada tique
seria o gasto mais caro da página —, então o servidor manda um carimbo de uma
linha e o CLIENTE repede a ficha.

**A armadilha é que ele dispara em QUALQUER remendo de sinal**, e esta cena tem
outro: o "mostrar à mesa" escreve `superficie` para levar o jogador ao mapa.
Sem filtro, um puxão do mestre repediria a ficha — trabalho invisível, disparado
por um gesto que não tem nada a ver, e que só aparece como tráfego.

O filtro mora num atributo IRMÃO e não no valor:

```
data-on-signal-patch-filter="{include: /^fichaversao$/}"
data-on-signal-patch="@get('/personagens/13?tab=' + $fichatab + '&embutida=1')"
```

**E o servidor precisa mandar UMA vez por mudança**, nunca por quadro — é a
mesma regra do puxão, e pelo mesmo motivo: remendo de sinal não é idempotente. A
memória do que já foi avisado é da CONEXÃO (`announceSheetChange`).

**Quem diz "mudou" é um EVENTO, e não a comparação de um carimbo a cada tique.**
A primeira versão lia o `updatedAt` do personagem em todo quadro do batimento —
uma consulta por segundo por jogador conectado, quase sempre para descobrir que
nada mudou —, e a decisão do dono foi que toda escrita dentro da sessão já é um
evento que dá para escutar. O stream só lê o carimbo quando o evento diz que a
ficha mexeu; ver "O barramento de eventos" acima.

- **O interesse é por PERSONAGEM, não por sessão.** A pergunta é sobre uma ficha,
  e a mesma ficha pode estar em duas mesas — pendurar isso no `SessionStore`
  obrigaria quem PUBLICA a saber em quais mesas o personagem está.
- **Quem não tem ficha nesta mesa simplesmente não pede o interesse dela.** Aqui
  morava "os outros recebem um canal NULO, e canal nulo num `select` nunca
  dispara" — era o truque certo enquanto havia um canal por store, e ele deixou
  de ser necessário na ALE-279: não pedir diz a mesma coisa sem exigir que quem
  lê conheça o truque.

Quem publica é o GATEWAY (`characterChanged`) e não cada comando: passam mais de
trinta mutações pelo `sheetCommand`, e a linha esquecida numa delas seria uma
ficha que não atualiza só naquele gesto. É a mesma lição do gancho que nascia
desligado, na seção do SSE.

**A segunda metade é o que o desenho quase perdeu:** o servidor NÃO sabe em que
aba a pessoa está, porque a aba viaja na query dos comandos da ficha e o stream
abriu antes de qualquer clique. Quem a guarda é um sinal que o clique na aba
escreve (`$fichatab`), e o repedido a concatena. Sem isso o remendo devolve a
aba padrão e tira o jogador de onde ele estava — provado por sabotagem, e é a
mesma família do `?tab=` perdido logo acima.

### O morph REMOVE o nó que o seu JS pendurou na linha

Uma animação que precisa de um elemento próprio — um véu, um brilho, um
espaçador — não pode ser pendurada de dentro do observador que reage ao remendo.
**O `MutationObserver` roda como MICROTAREFA, no meio do morph**, e o morph
reconcilia os filhos daquele nó logo em seguida: o que você acabou de criar não
está no HTML que veio do servidor, então ele some.

O que torna isso caro é que a animação foi PEDIDA. Um guarda que conte chamadas
de `el.animate` fica verde; a sonda que olhe o DOM no instante do `animate()`
também. Medido na ALE-174, com o mesmo nó consultado em quatro instantes:

    ao animar: ligado · microtask: ligado · raf1: DESLIGADO · +300ms: DESLIGADO

O conserto é um `requestAnimationFrame` antes de pendurar — o quadro seguinte já
tem o morph assentado. Animação que mexe no PRÓPRIO nó (escala, sombra, opacidade
do elemento que já existe) não precisa disso: o morph reusa o nó e a animação
sobrevive à reconciliação de atributos.

**E a mesma reconciliação é o que dispensa guarda contra repetição**: o morph não
toca atributo que já bate, então um observador de `aria-current` (ou de qualquer
atributo de estado) só acorda quando o valor MUDA de verdade. Escrevi um `if
(oldValue === 'true') continue` por medo do pisca-pisca e ele nunca foi
verdadeiro — provado removendo-o e medindo zero disparos extras.

### O nó que muda de DOCUMENTO perde o Datastar, e continua desenhado

Uma janela própria — Document Picture-in-Picture, um `<iframe>` de outro
documento — pede o nó vivo emprestado, e a leitura natural é `appendChild` do
painel lá dentro. O nó VAI: ele é adotado, aparece inteiro, com as classes, o
texto e os `data-*` no lugar. **O que não vai é a reatividade.** O runtime do
Datastar mora no documento de origem; o nó adotado sai da árvore que ele observa
e nada mais o alimenta.

Medido na ALE-218, movendo a coluna das notas para uma janela de PiP, com o
MESMO `input` sintético dos dois lados: **1 POST antes de mover e ZERO depois**,
com a faixa de estado continuando a dizer "Salvo" — ela é escrita por sinal, e
o sinal parou. Não há exceção, não há console, e a tela está perfeita.

O controle importava aqui mais que de costume: a primeira medição comparou um
`fill()` de verdade antes com um evento sintético depois, e responderia sobre o
INSTRUMENTO. Com o mesmo evento nos dois lados, a diferença é a janela.

**O que funciona é a janela receber um DOCUMENTO**, com o runtime dela — medido
na mesma sessão: 18 de 19 nós com `data-show` escondidos pelo Datastar daquele
documento. Ou seja: superfície que vai para uma janela própria é CENA com
endereço, e não painel mudado de lugar. Ver "As notas numa janela própria".

## O evento de ponteiro SINTÉTICO destrói o que ele mede

`element.dispatchEvent(new PointerEvent(...))` com um `pointerId` inventado faz
o `setPointerCapture` lançar `NotFoundError` — e a exceção leva junto as
escritas de sinal do Datastar daquele handler. O gesto parece MORTO quando ele
está inteiro.

Aconteceu duas vezes na mesma sessão: primeiro concluí "o retângulo não
funciona", depois "o pincel também parou". A segunda conclusão estava certa por
outra razão (o ternário acima) e a primeira estava errada — e as duas vieram do
mesmo probe.

**Gesto de ponteiro se mede com entrada REAL.** O Playwright (`page.mouse.down`,
`page.keyboard.down('Shift')`) tem pointer de verdade; a automação do Chrome MCP
NÃO aplica `modifiers` ao arrasto (medido: `shiftKey` chega `false`), então para
modificador o caminho é o Playwright.

## Remendo em nó COMPARTILHADO exige limpeza no gesto que troca de item

Uma cena em Datastar desenha **todos** os itens e alterna qual aparece com
`data-show`. Isso é seguro enquanto o conteúdo de cada item vier inteiro do
servidor — e deixa de ser no instante em que algo é escrito, DEPOIS da
renderização, num nó que os itens dividem.

O caso que ensinou: o diálogo de redefinição de senha é UM só, reaproveitado por
todas as linhas de jogador, e o token chega por remendo do servidor num `<div>`
de id fixo. Gerar o link da Ana, fechar, e abrir a caixa da Bia mostrava **o link
da Ana sob o nome da Bia** — e quem estiver com pressa entrega a chave da conta
errada.

**A regra: quem LIMPA é o gesto que TROCA de item, não o gesto que gera.** Quem
gera não sabe que vai haver um próximo; quem troca sabe que houve um anterior.

```
data-on:click="$alvoId = el.dataset.id;
               document.getElementById('reset-link').innerHTML = '';
               $redefinir.showModal()"
```

A família é **estado de um item sobrevivendo à troca por outro**, e o `data-show`
a multiplica. Vale conferir a cada cena nova: existe algum nó compartilhado que
recebe escrita depois da renderização? Se existe, quem troca de item tem de
apagá-lo.
