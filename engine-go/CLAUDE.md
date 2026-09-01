# Motor Go

Adapta o [CLAUDE.md da raiz](../CLAUDE.md) a este pacote. As regras da raiz
valem; o que está aqui estende ou sobrepõe.

`engine-go` é o app inteiro: a API HTTP na :3001, o motor de regras, e as CENAS
em `.templ` servidas com Datastar — mais a folha e as ilhas de JS delas, em
`api/piloto/src`. Um processo serve tudo.

Ele já foi só o backend, com uma SPA em SolidJS ao lado e o mesmo motor
compilado para WASM rodando no navegador. Os dois saíram na ALE-272: não há
`STATIC_DIR`, não há `dist` para servir, e a regra tem um lugar só.

Desde a ALE-273 esse processo também sobe por `docker compose up -d --build`,
com o banco em bind mount no hospedeiro. **O compose não trouxe um segundo
runtime**: continua sendo UM serviço, e a decisão do ALE-101 está inteira. O
proxy que normalmente viria junto foi considerado e recusado — ele compraria só
a compressão, que mora em `plataforma.Gzip` por ~150 linhas e sem um segundo
lugar onde o SSE pode ser bufferizado por engano.

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

## O gerador de tipos da fronteira SAIU

Aqui morava o `go generate ./engine`, que escrevia
`frontend/src/shared/api/engine-types.ts` — as formas que atravessavam a
fronteira do WASM —, mais o `TestGeneratedTypesAreCurrent` que falhava apontando
a primeira linha divergente.

Ele existia porque havia DOIS lados. Com a SPA e o WASM apagados (ALE-272, fatia
10c) não há fronteira a manter em dia: o motor é chamado de dentro do mesmo
processo, pelo tipo Go de verdade. O `cmd/tsgen`, o `engine/tsgen.go` e o guarda
saíram juntos.

A lição que fica, porque ela vale para o próximo gerador: **um tipo com
`MarshalJSON` próprio precisa declarar a forma de FIO**, e o emissor recusava com
panic quem não declarasse — refletir a struct em memória produz um tipo que
mente (o `ItemEffects` guarda flags num Set e serializa um array).

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
nomeando como faltante justamente a que a migração acabou de derrubar. Ver a
seção abaixo.

## A PRÉ-COMPRESSÃO do build saiu com a SPA

Aqui morava a explicação do `spaHandler` escolhendo o irmão `.br`/`.gz` que o
`postbuild` do front gerava (ALE-153): o `net/http` não comprime nada sozinho, e
o servidor mandava os 3,7 MB crus do `t20.wasm` para um navegador que pedia
`gzip, br`.

Com a SPA apagada (ALE-272, fatia 10c) não há `dist` para servir nem asset
pesado para pré-comprimir — os estáticos do piloto são a folha, quatro ilhas de
JS e duas fontes, todos embutidos no binário. O que sobrou é a compressão do que
o servidor RENDERIZA, logo abaixo.

Duas coisas que aquele caminho ensinou e que valem para qualquer arquivo servido
daqui: o `Content-Type` tem de sair do nome ORIGINAL (adivinhado pela extensão
do irmão comprimido, o wasm virava `application/octet-stream` e o
`instantiateStreaming` recusava), e `gzip;q=0` é uma RECUSA — um
`strings.Contains` a leria como aceitação, e é por isso que
`plataforma.AcceptsEncoding` existe.

## E o que o servidor RENDERIZA sai comprimido na hora

As duas compressões convivem e a divisão é por NATUREZA do conteúdo, não por
preguiça de unificar. Asset é imutável e se comprime UMA vez no build, com
brotli -q11 e os ~8s que ele custa; cena renderizada não existe antes da
requisição, então para ela a escolha real é gzip na hora ou nada.

Era **nada** até a ALE-273, e a conta é maior do que parece porque todo comando
da ficha responde redesenhando a cena INTEIRA: a aba de Combate viaja 44,7 KB
crus, 5,6 KB em gzip, e vai de novo a cada toque no PV. Numa LAN isso não
aparece; no telefone do jogador com dados móveis, são 44 KB por toque.

Quem faz é o `plataforma.Gzip`, montado na borda do mux em `cmd/api`. Ele decide
pelo `Content-Type` que o handler escreveu, e pula o que já chega com
`Content-Encoding` — que é exatamente o caso dos irmãos pré-comprimidos da
seção acima, e recomprimi-los produziria bytes maiores gastando CPU.

**A armadilha mora no SSE, e ela não deixa erro para trás.** A resposta de todo
comando do Datastar é `text/event-stream` — ela usa o envelope de SSE para
mandar UM remendo e fechar. Então "não comprimir SSE" pularia justamente o que
se quer comprimir; e comprimir SEM repassar o `Flush` prende o quadro no buffer
interno do `gzip.Writer`, e o fluxo AO VIVO da Mesa para de atualizar. Nada
falha, nada loga, e o sintoma — "o tempo real quebrou" — não aponta para um
middleware de compressão. Por isso o `Flush` esvazia o gzip ANTES de quem está
embaixo, e `TestOFluxoAoVivoAtravessaOGzip` mede um quadro chegando com a
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
decidir no `Flush` também**, e `TestOFlushAntesDoWriteJaDecideOEnvelope` repete a
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

O tempo real era socket.io e subia para `wss://` numa conexão HTTP/1.1 à parte,
porque o Go não anuncia o RFC 8441. Com SSE (ALE-253) ele é um `GET` como
qualquer outro e viaja pelo mesmo h2 — uma conexão a menos e uma exceção a menos
para lembrar.

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

## O tempo real é SSE, e é uma rota como as outras

`GET /campaigns/{campaignId}/sessions/{id}/events` devolve `text/event-stream` e
fica aberto; tudo que SOBE é `POST`/`PATCH`/`DELETE` numa rota própria
(`mountLiveRoutes`). Era socket.io até a ALE-253, e a troca não foi de
biblioteca: as 37 mensagens que subiam pelo socket eram TODAS mutação, e mutação
é uma requisição. Bidirecional não era requisito, era hábito.

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
`SessionStore.Assinar`, o `BoardStore.Assinar`, o `CharacterWatch.Assinar` e um
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
  permitidos do `aovivo` e do `tabuleiro`, cujos guardas de fronteira avisam que
  acrescentar import à lista transforma a porta em enfeite. O que impede a porta
  dos fundos é o `TestOVocabularioNaoImportaNinguem`: enquanto o vocabulário for
  folha, depender dele não cria fronteira errada — e no dia em que alguém
  importar a ficha ali "para enriquecer o evento", os dois contextos passam a
  alcançar a ficha de graça com o guarda de lá verde.

Um barramento tipado é o caso em que a PORTA não serve, e vale saber por quê:
porta é interface declarada no consumidor, e ela só casa com tipos do consumidor
— um vocabulário por contexto, que é o problema de novo. O que legitima o
compartilhamento é ele ser *shared kernel*: pequeno, sem dependências, e de todos
porque não é de ninguém.

## Catálogos

O catálogo viaja COMPRIMIDO (`writeCatalogJSON`, ALE-159): `spells` sozinho são
179 KB crus e 40 KB em gzip. Isso importava quando a SPA os buscava por HTTP
(ALE-107) e eles entravam em toda carga fria; hoje as cenas leem o embutido
direto, e a rota `GET /catalog/:nome` está entre as que perderam consumidor com
a SPA. Comprimido UMA vez e guardado, não por requisição — o conteúdo vem de
`go:embed` e não muda enquanto o binário for o mesmo. A leitura do `Accept-Encoding` é a do `plataforma.AcceptsEncoding`, uma
só, para o tratamento de `q=0` não divergir em duas cópias.

`catalog/data/*.json` é embutido no binário. **Este é o único lugar onde
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

**Regra que só o front sabia é fronteira ABERTA, e a progressão de círculo era
uma.** Em que nível cada classe destrava cada círculo vivia só no
`SPELL_PROGRESSION` da SPA, então o servidor não tinha como perguntar — e o
`validateAugments` aceitava qualquer `requiresCircle`, que são 126 dos 486
aprimoramentos do catálogo. A tabela virou o campo `spellcasting` das cinco
classes conjuradoras em `classes.json` (ALE-272), MOVIDA e não retranscrita: um
teste do front comparava as duas cópias campo a campo, e morreu com a SPA — a
tabela do catálogo ficou como fonte única. O sintoma que essa família produz é
sempre o mesmo: a tela tranca e o servidor não.

A fatia 7 fechou a terceira: a compatibilidade entre **melhoria/material** e o
item que os recebe (`appliesTo`) vivia no `familyFor` do TypeScript, e o
`handleAddItem` carregava a dívida escrita em comentário. Agora ela é
`aMelhoriaCabeNoItem`, e o filtro do diálogo é conveniência sobre a mesma regra.

E a fatia 8 fechou a QUARTA, que era a maior: as regras de ESCOLHA de poder —
quantas vagas o nível abre (uma por nível a partir do 2º, p33), quantos
benefícios a origem dá, quais caminhos e quais deuses cada classe aceita — eram
363 linhas de `shared/rules/abilities-*.ts`, e o `handleUpdateAbilities` gravava
os cinco blobs sem conferir NADA. Um pedido montado à mão punha vinte poderes num
personagem de nível 1 e o motor somava os modificadores de todos. A validação é
`aFichaComEscolhasValidas`, ela roda nas DUAS portas (o endpoint JSON e os
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

## O fixture do piloto prima o catálogo DE VERDADE

O `novoPiloto` primava `{"items":[]}`, e isso fazia regra sumir do TESTE sem
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

1. **O molde** (`api/bancada_test.go`): o `TestMain` migra UM banco e cada teste
   o copia. `db.Open` continua sendo o mesmo de produção, com o mesmo
   `assertSchema` — o goose encontra a última versão e não tem o que fazer.
2. **`PRAGMA synchronous=OFF`** no banco de teste. Durabilidade é o que um banco
   que morre no fim do caso não tem o que proteger. Fica no helper e **nunca** no
   `db.Open`: em produção essa linha é perda de dados do mestre.

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

## Testes

As faixas, o vermelho antes de confiar e o que não merece teste estão no
[CLAUDE.md da raiz](../CLAUDE.md). O que é deste pacote:

- `go test ./...` — sem flag, sem setup.
- **Teste de regra vive junto da regra e cita a página.** Ele não é a mesma coisa
  que o ORÁCULO: o oráculo prende a ficha inteira de ponta a ponta e acusa
  qualquer número que mude sem ter sido pedido; o teste de regra explica POR QUE
  aquele número é aquele, com a página do livro do lado. Nenhum substitui o
  outro.

  > Aqui morava a frase "teste de paridade prova que os dois motores concordam".
  > Ela ficou **falsa sem ninguém mexer nela**: o `t20-data` foi aposentado (ver
  > "Regenerar oráculo"), não há segundo motor, e o mesmo arquivo passou a dizer
  > as duas coisas. É o defeito que a seção "Documentação" da raiz descreve, e
  > ele sobreviveu a uma épica inteira.

- **Sabotar é a forma barata de provar que o teste mede o que diz medir**, e no
  Go ela é barata mesmo: inverta o operador, rode o caso, confirme o vermelho,
  reverta. Foi assim que se descobriu que um teste de PV passava por acidente —
  em todos os casos a primeira classe também era a maior.

## templ — as armadilhas que já custaram tempo

As cenas do `piloto/` são `.templ` compiladas para `.go` por `go tool templ
generate`. O `.templ` e o `_templ.go` andam juntos e o CI recusa o par
desencontrado, no mesmo molde do `TestGeneratedTypesAreCurrent`. O que segue foi
todo descoberto errando — está aqui para ninguém redescobrir:

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
- **Regenerar não basta: o servidor precisa REINICIAR.** O `go run ./cmd/api`
  compila uma vez, então depois do `templ generate` o processo continua servindo
  o HTML antigo. Isto já produziu uma medição de layout inteira contra a página
  velha — 74px de deslocamento "que não sumiam" depois do conserto, porque o
  conserto não estava no ar.
- **Classe nova exige `scripts/build-piloto-css.sh`.** O scanner do Tailwind lê
  `../*.templ`, e só ele — por isso `classesDoBotao` mora no `.templ` e não no
  `.go`. Classe que não passou pelo scanner simplesmente não existe na folha, e
  o elemento aparece sem estilo em vez de dar erro.
- **E TOKEN inventado tem o mesmo fim, com o script rodado.** `text-grimorio-ink`
  parece irmão de `text-grimorio-gold` e não é: `grimorio-ink` não está na
  paleta, o Tailwind não emite regra para o que não conhece, e o elemento fica
  com a cor HERDADA — o crachá de contagem dos Efeitos saiu dourado sobre
  dourado, 1,53:1, e atravessou uma fatia inteira (ALE-272). O
  `TestTodaTintaDaCasaExisteNaFolha` varre `piloto_*.templ` e `piloto_*.go` e
  cobra cada token da casa contra a folha compilada. A paleta mora no
  `@theme` do `api/piloto/src/index.css`; conferir lá antes de inventar o nome.

## Datastar: dez armadilhas que não deixam erro para trás

As três primeiras foram descobertas na ALE-203, a quarta na ALE-205, a quinta na
ALE-235, quatro na ALE-272 e a última na ALE-275; nenhuma delas escreve uma linha
no console — a oitava escreve UMA, e no lugar que ninguém olha. Estão aqui porque
o sintoma de cada uma aponta para o lugar errado.

### `data-show` + `data-attr:style` no MESMO nó CONGELA a aba

O `data-show` escreve `el.style.display`; o `data-attr:style` reescreve o
atributo `style` INTEIRO, apagando o `display` que o outro acabou de pôr — que
faz o outro pôr de novo. O renderizador entra em laço.

O sintoma é o pior possível: a aba para de responder a TUDO. Sem console, sem
exceção, sem conseguir sequer navegar para fora — a ferramenta de medir some
junto, e o que sobra é "o navegador travou", que não aponta para nada.

**Quem ESCONDE é um nó, quem POSICIONA é outro.** O
`TestNenhumNoTemDataShowEDataAttrStyleJuntos` varre o HTML servido e recusa a
combinação.

### O sinal é um PROXY: ler um índice que não existe o CRIA

`$lista[0]` não é "o primeiro item": o Datastar registra o caminho e o cria
vazio. Com uma reserva de doze rótulos no ar, `$reguapontos[i]` encheu o sinal
de strings vazias entre os pontos de verdade — pingos na origem do plano e o
servidor medindo zero.

**Guardar o sinal numa constante NÃO resolve** (a constante continua sendo o
proxy). O que resolve é COPIAR: `const lista = [...$reguapontos]`. O
`TestNenhumaExpressaoIndexaOSinalDaLista` afirma a regra pelo que PODE vir
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

Na SPA, animação de entrada era `animate-in` + `<Show keyed>`: o nó era
RECONSTRUÍDO a cada troca e a animação vinha de graça (ALE-97). Nas cenas
desenhadas pelo servidor — campanhas, personagens, e toda cena com cursor — nada
nunca monta: o servidor manda todos os itens e o cursor só alterna `data-show`.

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
(`oPostDaFicha`), e um guarda de varredura lê o HTML de cada aba e falha se algum
comando sair sem ele (`TestNenhumComandoDaFichaPerdeAAba`). Sinal do cliente
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
desfazia o dele. `TestNenhumFocoPedeAoServidorSemGuardaDeTeclado` varre a FONTE
inteira, e não uma cena servida, porque enumerar cena por cena deixaria a
próxima nascer sem medição.

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
(`comandoDaFicha` + `fichaView.Recusa`). A consequência para os testes é a parte
que importa: o status deixou de distinguir "gravou" de "recusou", então **o que
os guardas afirmam é a FRASE**, com `aRecusaDaCena`. A API JSON continua com os
status dela; quem desenha página responde página.

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
data-on-signal-patch="@get('/piloto/personagens/13?tab=' + $fichatab + '&embutida=1')"
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
trinta mutações pelo `comandoDaFicha`, e a linha esquecida numa delas seria uma
ficha que não atualiza só naquele gesto. É a mesma lição do gancho que nascia
desligado, na seção do SSE.

**A segunda metade é o que o desenho quase perdeu:** o servidor NÃO sabe em que
aba a pessoa está, porque a aba viaja na query dos comandos da ficha e o stream
abriu antes de qualquer clique. Quem a guarda é um sinal que o clique na aba
escreve (`$fichatab`), e o repedido a concatena. Sem isso o remendo devolve a
aba padrão e tira o jogador de onde ele estava — provado por sabotagem, e é a
mesma família do `?tab=` perdido logo acima.

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

É a mesma família do `<Show keyed>` sem parâmetro que mordeu do lado da SPA
(ALE-208): **estado de um item sobrevivendo à troca por outro**. Muda o vestido,
não o defeito. E como o `data-show` é a forma que esta migração multiplica, vale
conferir a cada cena nova: existe algum nó compartilhado que recebe escrita
depois da renderização? Se existe, quem troca de item tem de apagá-lo.
