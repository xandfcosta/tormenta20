# Motor Go

Adapta o [CLAUDE.md da raiz](../CLAUDE.md) a este pacote. As regras da raiz
valem; o que está aqui estende ou sobrepõe.

`engine-go` é o app inteiro: a API HTTP na :3001, o motor de regras, e as CENAS
em `.templ` servidas com Datastar — mais a folha e as ilhas de JS delas, em
`api/piloto/src`, e o kit de apresentação em `web/ui`. Um processo serve tudo.

**Cada cena é um pacote em `web/`** desde a ALE-278, e a última saiu com a Mesa:
o `api` era 188 arquivos e 105 mil linhas quando a divisão começou, e hoje são
36 arquivos de produção e **nenhum `.templ`**. O que sobrou lá é a composição do
roteador, as portas que cada cena pede, e as regras que não são de tela nenhuma.

**E o `*Server` deixou de ser porta** (fatia 6): ele tinha 89 métodos
exportados para cumprir a união das onze portas e hoje tem OITO. Cada cena
recebe um adaptador com o que ela pede, e as regras ganharam quatro tipos com
nome do que decidem — conta, campanha, ficha e mesa ao vivo.

**A "API JSON" não sobrou.** Ela saiu na ALE-277: das 76 rotas do `Router()`
ficaram SETE, e dos 113 manipuladores ficaram nove. O resto não tinha
consumidor — nem cena, nem e2e, nem `fetch` de ilha — desde que a SPA foi
apagada (ALE-272), e um handler sem chamador não quebra compilação, não aparece
em diff e ainda cobra revisão a cada renome. O que ficou é o que alguma coisa
de fato chama: `/health`, o CRUD de campanha que a lista consome, a criação de
sessão, a lista de personagens e o `PATCH` de condições.

Ele já foi só o backend, com uma SPA em SolidJS ao lado e o mesmo motor
compilado para WASM rodando no navegador. Os dois saíram na ALE-272: não há
`STATIC_DIR`, não há `dist` para servir, e a regra tem um lugar só.

**As cenas atendem na RAIZ**, e `/piloto/*` responde 404 (ALE-280). O prefixo era
o nome daquela migração — a SPA de um lado, as cenas de outro — e sobreviveu a
ela no lugar mais caro, que é o endereço que o jogador favorita. O corte foi
SECO por decisão do dono: o app nunca foi usado numa mesa real, então não há link
de jogador a proteger, e um desvio a menos é uma exceção a menos no mux.

Duas consequências que o `git grep` não mostra e que já morderam uma vez:

- **`http.StripPrefix` saiu junto**, e com ele a razão de o `alvoOriginal` ler
  `RequestURI`. A linha ficou porque `URL.Path` descarta a QUERY — mas o motivo
  escrito nela era o outro, e uma explicação que aponta para um mecanismo que não
  existe mais é pior que nenhuma.
- **Endereço interno deixou de somar prefixo.** O `"/" + rotaDoLivro` virava
  `//livro`, que não é uma barra a mais: `//algo` é URL relativa a PROTOCOLO, e o
  navegador a lê como o HOST `algo`. Quem denunciou foi um teste de cena; o
  compilador não tem como.

E três entradas saíram de `legacy_addresses.go` — `/admin`, `/grimorio` e
`/redefinir-senha` eram escritas iguais pela SPA e pelo piloto, então sem prefixo
o desvio virou origem igual a destino. Isso não é linha inútil, é laço: o padrão
literal ganha do `"/"` das cenas no `ServeMux`, e a tela nunca apareceria.

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

## O tempo real é SSE, e o canal MUDOU de dono

Era socket.io até a ALE-253, e a troca não foi de biblioteca: as 37 mensagens que
subiam pelo socket eram TODAS mutação, e mutação é uma requisição. Bidirecional
não era requisito, era hábito. O que desceu virou
`GET /campaigns/{campaignId}/sessions/{id}/events`, um `text/event-stream` que
fica aberto; o que subia virou uma rota por comando (`mountLiveRoutes`).

**As duas pontas dessa troca saíram na ALE-277**, e o parágrafo acima fica porque
o argumento continua sendo o certo — o que mudou é quem o exerce. A Mesa em
Datastar tem fluxo PRÓPRIO (`/mesa/{campanha}/{sessao}/stream`, em
`web/table/stream.go`), ele assina o `events.Bus` e não o `SSEHub`, e os comandos
dela são rotas da CENA. Nenhuma linha do que este arquivo descrevia como "a rota
de eventos" existe.

> **E o `SSEHub` ficou sem ouvinte.** Medido na ALE-277: em produção ninguém
> chama `SSEHub.Add` — só testes —, porque a única rota que abria conexão era a
> `/events`. Quem EMITE continua lá (o `live_publish.go`, o `session-rest` da
> Mesa, o `character-changed` do servidor), e emitir para zero ouvinte não
> estoura nada. Não foi apagado nesta issue por uma razão específica: o
> `publishBoardState` e o `publishSessionState` fazem DUAS coisas, e a segunda é
> GRAVAR — desmontar o hub sem separar a persistência primeiro trocaria uma
> emissão inútil por uma escrita perdida. A mesma medição vale para a
> `PresenceRegistry`: a Mesa LÊ o elenco (`Presence().Roster`), e quem o
> preenchia era o handshake da `/events`, então a lista está vazia desde a
> ALE-272.

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

O catálogo viajava COMPRIMIDO (`writeCatalogJSON`, ALE-159): `spells` sozinho são
179 KB crus e 40 KB em gzip. Isso importava quando a SPA os buscava por HTTP
(ALE-107) e eles entravam em toda carga fria. **Hoje as cenas leem o embutido
direto**, e a rota `GET /catalog/:nome` foi apagada na ALE-277 com o
`api/catalog.go` inteiro, por ter perdido o consumidor junto com a SPA — o
`plataforma.AcceptsEncoding` ficou, porque quem serve arquivo estático ainda lê
`Accept-Encoding`.

O que a decisão de então deixou e continua valendo é o outro lado dela: ler UMA
vez e guardar, não por requisição, porque o conteúdo vem de `go:embed` e não muda
enquanto o binário for o mesmo. É o que o `race_traits.go` faz com `sync.Once`.

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

1. **O molde** (`db/testdb`, extraído do `api` na ALE-281): o `TestMain` migra
   UM banco e cada teste o copia. `db.Open` continua sendo o mesmo de produção,
   com o mesmo `assertSchema` — o goose encontra a última versão e não tem o que
   fazer.
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

  > Aqui morava a frase "teste de paridade prova que os dois motores concordam".
  > Ela ficou **falsa sem ninguém mexer nela**: o `t20-data` foi aposentado (ver
  > "Regenerar oráculo"), não há segundo motor, e o mesmo arquivo passou a dizer
  > as duas coisas. É o defeito que a seção "Documentação" da raiz descreve, e
  > ele sobreviveu a uma épica inteira.

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

OITO moram lá hoje, e a lista cresceu por duas razões diferentes — vale saber
qual é qual. Três nasceram lá porque a regra é do REPOSITÓRIO e não de um
pacote; três MUDARAM de casa porque mediam o próprio diretório e encolheram
quando uma cena virou pacote; e as duas últimas são a família das citações.

- **`TestEveryTestNameIsEnglish`** varre todo `*_test.go` e recusa nome com
  palavra em português. Ele é o que impede o 774º — a regra de idioma sempre
  disse "nome de teste" com todas as letras, e ainda assim 773 dos 1.051 casos
  estavam em português, porque *convenção escrita e não varrida é aplicada
  exatamente aos arquivos que alguém apontou*.
- **`TestNoCitationNamesAMissingTest`** recusa comentário ou `.md` que nomeie um
  teste inexistente. A varredura mediu **136** dessas antes de começar: um `.md`
  fica errado sem ninguém mexer nele, e nada no repositório acusava.
- **`TestNoCitationNamesAMissingFile`** faz o mesmo para CAMINHO DE ARQUIVO, e
  mediu **41** (ALE-285). A origem delas não foi a SPA: foram as varreduras de
  idioma. A ALE-282 e a ALE-283 renomearam arquivos para inglês, a ALE-278 moveu
  famílias inteiras de pacote, e os comentários que apontavam para eles ficaram
  falando dos nomes velhos — "a regra mora no ..." apontando para quatro arquivos
  que já se chamavam outra coisa.

  Ele só olha `.go` e `.templ`, e essa restrição é o que o torna possível: com as
  extensões todas ele acusa **325**, e a maioria esmagadora (181 `.ts`, 20 `.js`,
  17 `.tsx`) é a PROCEDÊNCIA que esta casa valoriza — `api/auth.go` citando o
  `auth-user.type.ts` do Nest diz de onde a regra veio e está certo. Um guarda com
  325 exceções é um guarda que alguém apaga; com as extensões do stack VIVO ele
  tem 41 defeitos e vinte lápides declaradas.

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
  chegaram na ALE-278, pela mesma razão do de baixo e no mesmo dia: o primeiro
  lia `piloto/piloto.src.css`, um caminho relativo ao `api`; o segundo varria
  `piloto_*.go` do próprio diretório. Quando a Mesa virou `web/table`, o arquivo
  sumiu debaixo de um e o glob deixou de casar para o outro. **Os dois falharam
  ALTO porque os dois tinham piso** — e o irmão do foco, que não tinha, teria
  passado verde medindo metade.

- **`TestNoFocusAsksTheServerWithoutAKeyboardGuard`** chegou na ALE-278, e não
  por ser regra de repositório desde sempre: ele já existia dentro da cena do
  bestiário, e foi a divisão em pacotes que o obrigou a mudar de casa. A
  história inteira está em "Dois pedidos de UM gesto", mais abaixo; o que
  interessa aqui é a forma, porque ela vale para o próximo: **um guarda que
  varre `.` mede o diretório em que ele por acaso mora**, e mover o arquivo
  encolhe a varredura sem mudar uma linha do guarda nem acender nada.

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
- **Classe nova exige `scripts/build-piloto-css.sh`.** Classe que não passou pelo
  scanner simplesmente não existe na folha, e o elemento aparece sem estilo em
  vez de dar erro.
  **Aqui morava "o scanner lê `../*.templ`, e só ele", e é falso** — medido na
  ALE-278: tirar os DOIS `@source` do `piloto.src.css` não muda um byte da folha
  compilada (105.328 com, 105.328 sem), porque a detecção automática do Tailwind
  v4 varre da pasta da folha até a raiz do projeto respeitando o `.gitignore`. As
  linhas ficaram como declaração de intenção; quem depurar "classe sumiu" não
  deve perder tempo nelas. O suspeito é o TOKEN que não existe na paleta.
- **E TOKEN inventado tem o mesmo fim, com o script rodado.** `text-grimorio-ink`
  parece irmão de `text-grimorio-gold` e não é: `grimorio-ink` não está na
  paleta, o Tailwind não emite regra para o que não conhece, e o elemento fica
  com a cor HERDADA — o crachá de contagem dos Efeitos saiu dourado sobre
  dourado, 1,53:1, e atravessou uma fatia inteira (ALE-272). O
  `TestEveryHouseTintExistsInTheStylesheet` varre `piloto_*.templ`, `piloto_*.go` e
  o `web/` INTEIRO, e cobra cada token contra a folha compilada. A paleta mora
  no `@theme` do `api/piloto/src/index.css`; conferir lá antes de inventar o nome.
  **O `web/` entrou na ALE-278 e mostra a forma da falha desta família**: o
  kit mudou de nome de arquivo, o padrão `piloto_*` deixou de casar com ele, e o
  guarda teria seguido verde medindo as cenas e ignorando o botão, o campo e a
  casca — os arquivos onde uma tinta errada aparece em TODA tela. Ele é varrido
  por caminhada e não por lista de pacotes desde que a terceira cena (`web/grimoire`)
  caiu fora da lista enumerada: enumerar faria a PRÓXIMA cena nascer sem medição.
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

## `web/forge`: a primeira cena a sair, e o formato que as próximas seguem

A forja tentou sair primeiro e não conseguiu — ela precisava de `racaDoLivro`,
de `CharacterDTO` e do kit, e todos eram do `api`, que a importaria de volta para
montar rota. Depois de três camadas fora (`sheet`, `book`, `web/ui`), ela saiu.

**O formato é uma PORTA declarada pela cena.** `forge.Deps` está em `web/forge`,
e o `*api.Server` a cumpre — a direção importa: quem escolhe o que atravessa a
fronteira é o CONSUMIDOR, e não o objeto que tem tudo. O `api` monta com
`forge.Routes(r, forge.New(s))`, e é nessa linha que o compilador cobra quando a
porta deixa de ser cumprida.

A medição que disse que isso ia dar certo: o `Server` tem 14 campos e 415
métodos, mas cada cena toca **um a três campos**. A forja toca `queries` e
`catalogs`, mais quatro métodos. A porta é fina porque o objeto é grande por
acumulação, não por acoplamento.

Duas decisões que vale copiar na próxima cena:

- **A porta pede o MENOR tipo que resolve.** Ela devolve `CurrentUserID(r) int64`
  e não o usuário inteiro, porque o tipo do usuário é do `api` — e uma porta que
  devolve tipo do hospedeiro não é porta, é o hospedeiro com outro nome.
- **Uma cópia, declarada.** O `oPassoDaURL` são sete linhas de parse que a ficha
  também tem. Pô-lo na porta seria mais acoplamento que duplicação, e a cópia diz
  isso no comentário dela.

### O guarda de fronteira da cena pega o que o compilador não pega

Importar o `api` de dentro da cena o COMPILADOR já recusa — é ciclo. O guarda
existe para o resto, e ele provou o valor na primeira execução: `items.go`
importava `t20engine/catalog` DIRETO, contornando a camada tipada. Nenhum ciclo,
nenhum erro, só a divisão vazando por baixo.

**A regra que aquele achado deixou: o destino de uma função é a DEPENDÊNCIA
dela.** O índice de itens de origem lia catálogo, então era do livro — mesmo que
só a forja o usasse.

## `web/door`: a segunda cena, e as três coisas que a porta ensinou

A porta saiu depois da forja, do hub e do grimório, e ela é a primeira cuja
extração DOEU — a forja "não vazava nenhum símbolo", e esta alcançava o `bcrypt`,
o `db` e dois sentinelas de erro do `api`. As três saíram, e nenhuma por regra de
estilo:

- **O `bcrypt` não atravessa.** A cena gerava o hash da senha nova com o
  `bcryptCost` do `api`, o que a obrigaria a receber uma constante de custo
  criptográfico por uma porta, para fazer trabalho que não é dela. O
  `ResetPassword` faz o caminho inteiro do outro lado; a cena pergunta se deu
  certo.
- **Duas perguntas em sequência viram uma.** A cena chamava `usableReset` e
  depois `GetUserByID`, carregando a linha do banco no meio só para ter o
  `Userid`. O que a tela mostra é o e-mail — então a porta pede
  `ResetLinkOwner(ctx, token) (email, ok)`, e a linha não atravessa.
- **Quem CLASSIFICA o erro é o hospedeiro; quem escolhe a FRASE é a cena.** Os
  sentinelas de convite recusado e convite gasto são valores do `api`. Se a cena
  os lesse, alcançaria o hospedeiro; se o `api` devolvesse a frase pronta, a voz
  da porta passaria a morar nele. O meio-termo é um `RefusalMotive` declarado
  pela CENA: o hospedeiro diz qual dos três casos é, a cena diz o que o jogador
  lê.

**E a lição que valeu para as dez cenas seguintes: a porta pede a PERGUNTA, não
o objeto.** `HasSession(r) bool` no lugar de `sessionUser(r) (AuthUser, error)`,
pelo mesmo motivo que a forja pede `CurrentUserID` e não o usuário — o tipo do
usuário é do `api`, e uma porta que devolve tipo do hospedeiro não é porta.

### O renome que acompanha a mudança de pacote tem duas armadilhas, e as duas morderam

Arquivo que se move sai com os identificadores em inglês inteiros, e a troca é
textual porque o `gopls rename` não alcança componente `templ`. Duas coisas
quebraram, as duas em silêncio para o compilador:

- **A troca entrou em STRING.** `Convite` virou `Invite` dentro da frase que o
  jogador lê ("Invite inválido ou expirado"), e `Nome` virou `Name` num rótulo de
  formulário. É a família do "diálogo Int" da fatia 4, com o comentário protegido
  e a string não. **Pule comentário E string**, e depois varra as strings
  procurando as palavras que você acabou de introduzir.
- **A troca entrou em campo do KIT.** `ui.Field{Nome: …, Erros: …}` virou
  `ui.Field{Name: …, Errors: …}` — nomes do `web/ui`, que esta fatia não ia
  tocar. Esse o compilador pega, e é a regra do CLAUDE.md da raiz acontecendo: o
  nome que você CHAMA de fora segue o que está lá.

E o guarda de citação da ALE-285 pegou o terceiro caso no primeiro renome real
depois de existir: um comentário do `web/ui` citava o antigo `piloto_porta_view`,
que tinha acabado de virar `web/door/view.go`. Ele pegou o QUARTO logo em
seguida — esta seção, quando ela escreveu o nome morto por extenso.

### Os guardas funcionais da cena ficam no HOSPEDEIRO

O `web/door` tem dois testes: o de fronteira e o do redirecionamento aberto (que
é regra da cena e usa função não exportada). Todo o resto — treze casos que
montam um `api.Server` de verdade e dirigem o roteador de verdade — ficou em
`api/door_test.go`, que é o que a forja já tinha feito com o
`api/piloto_forja_test.go`.

Não é indecisão: um pacote de cena que hospedasse esses casos teria de importar o
`api`, que importa a cena de volta. **A cena sai; a bancada que a exercita
fica.** E as frases que os casos afirmam passam a ser escritas à mão, porque as
constantes da cena são inalcançáveis dali — o que é sorte, já que importar o
valor de quem está sendo testado faz o teste andar junto com o defeito.

## `web/finder` e `web/routes`: a cena SEM porta, e onde moram os endereços

O buscador do livro saiu na ALE-278, e ele é o caso oposto ao da porta: **ele não
declara `Deps` nenhuma.** A forja declara seis dependências, a porta nove, o
buscador zero — ele lê o livro embutido pelo `book`, pontua com o `search`,
desenha com o `web/ui` e linka pelo `web/routes`, quatro pacotes que já eram
folha. Não há banco, não há sessão, não há casca a montar.

Por isso o `Routes` dele recebe só o roteador, e não existe `Scene`. **Uma
interface vazia declarada por simetria seria cerimônia pura**: o que dá valor à
porta é ela DIZER o que a cena alcança do hospedeiro, e aqui a resposta é nada.

### `web/routes`: os endereços que uma cena cita de OUTRA

Ele nasceu porque o buscador monta o link do resultado com `/mestre/bestiario`,
que era constante do trilho do mestre — e depois de virar pacote ele não alcança
mais. A alternativa óbvia, escrever a string do lado de lá, é a que quebra em
silêncio, e a ALE-280 acabou de mudar endereços.

**O critério de entrada é estreito de propósito: só entra endereço que uma cena
cita de OUTRA.** Medido na entrada, das cinco constantes de rota do `api` só duas
eram citadas de fora da própria família — `/mestre/bestiario` e `/livro`. O
`/buscador` e o `/livro/ler` ficaram com o dono, porque trazê-los não compra nada
e transforma o pacote no lugar onde tudo cabe.

Ele também levou os três CONSTRUTORES de endereço do buscador, e aí a mudança
pagou sozinha: duas das três funções montavam `"/mestre/" + aba + …` à mão
enquanto a terceira usava a constante do bestiário — **três grafias do mesmo
prefixo em quatro linhas**, invisíveis porque as três moravam juntas.

E ele não é o mapa de rotas do app: quem registra rota é cada cena, no `Routes`
dela. Uma cena pode atender vinte rotas e não aparecer ali nenhuma vez.

> O guarda de fronteira dele permite a biblioteca PADRÃO e recusa só
> `t20engine/*`. A primeira versão recusava tudo, e isso empurraria o
> `url.QueryEscape` para cada chamador — que é como um endereço sai sem escapar
> em UM lugar e ninguém vê.

### O teste que era de duas camadas num arquivo só

O teste do buscador — o antigo `piloto_buscador` do `api` — misturava sete
casos da REGRA de ranqueamento
(funções puras: qual achado vem primeiro, o corte, o empate) com dois que
precisam do servidor montado. Na extração ele se dividiu pela camada — os sete
foram para `web/finder`, os dois ficaram em `api/finder_test.go`.

Vale a pena notar que **a mistura não incomodava enquanto tudo era um pacote
só**. Foi a fronteira que a tornou visível: os unitários não compilavam mais no
`api`, e os de HTTP não compilavam no `finder`. A divisão em pacotes está
cobrando isso de cada cena que sai, e a resposta é sempre a mesma — unitário onde
a regra mora, integração onde a composição acontece.

## `web/bookui`: a decisão que a fatia 4 adiou, cobrada pela fatia do mestre

O livro e os elos — desenhar um verbete, transformar um texto em citação
clicável, o selo "p289" que abre o leitor — saíram do `api` na ALE-278, 245
linhas em dois arquivos.

**Eles já eram puros.** Importavam `book`, `web/ui` e `web/routes` e mais nada,
então o movimento foi um `git mv`. O que custou não foi a extração: foi ela ter
sido ADIADA.

### A dívida tinha nome e endereço

A fatia 4 escreveu, ao criar o `web/ui`, que o livro e os elos ficavam para trás
*"porque o `trecho` que eles desenham nasce de uma consulta ao catálogo de
efeitos e de escolas de magia; levá-los faria o pacote de apresentação importar
catálogo, que é o contrário do que a divisão existe para conseguir"*.

A razão estava certa e a conclusão faltava uma opção. Não era escolher entre o
kit e o `api` — era o pacote do MEIO: um que sabe do livro e não sabe de HTTP.
Enquanto ele não existiu, os cinco símbolos ficaram no hospedeiro, e **o trilho
do mestre não podia sair**, porque lê todos os cinco. O verbete, pelo mesmo
motivo.

A lição não é sobre este pacote: **uma camada adiada não fica parada, ela vira o
bloqueio da próxima fatia.** E o preço não apareceu no dia em que foi adiada; ele
apareceu quando alguém foi extrair outra coisa e descobriu que não dava.

### O critério do `web/routes` funcionou sozinho, e é a prova dele

O `/livro/ler` e o `/verbete` estavam marcados como endereço INTERNO na fatia do
buscador — cada um era citado só pelos arquivos da própria família, e por isso
ficaram com o dono.

Nesta fatia os dois passaram a ser citados de fora, no MESMO instante em que o
`bookui` virou pacote, e entraram. Ninguém precisou decidir de novo: o critério
("só entra endereço que uma cena cita de OUTRA") reclassificou os dois sozinho.
É o que se espera de um critério escrito em vez de um julgamento repetido.

### O tipo do endereço foi junto, e vale saber por quê

O `enderecoDoLivro` — `Base`, `Abertura`, e o método que monta o endereço do
leitor — morava na cena que SERVE o PDF. Ele é a assinatura de todos os
componentes daqui (`PageSeal`, `Chunk`, `CrossRef` recebem um), então deixá-lo lá
faria o pacote de apresentação do livro importar a cena que serve o arquivo.

Os dois casos de teste que prendem a REGRA do endereço foram junto — livro não
configurado não produz link, e o termo entra escapado. O resto do
`piloto_livro_test.go` ficou no `api`, porque serve o PDF de verdade por HTTP:
outra camada, outra pergunta. **Quarta vez seguida que a fronteira separa um
arquivo de teste que misturava duas.**

### `elo` não vira `link`, e o glossário já dizia

A seção A proíbe `link` como identificador para `elo`, e a razão é que o elo anda
DENTRO do acervo enquanto o botão do livro SAI para o PDF. A grafia inglesa não
existia; foi escrita antes do código, como a regra manda: **`elo` → `crossref`**.

## `web/admin`: a MAIOR porta, e a armadilha que só o `.templ` tem

A administração saiu na ALE-278 com **treze** métodos na porta — contra nove da
porta de entrar, seis da forja e zero do buscador. O tamanho não é vício: esta
cena é um painel de CONTROLE sobre serviços do servidor. Backup, cunhar convite,
apagar conta e ler o tamanho do banco são coisas que o hospedeiro faz; a cena
oferece o botão e desenha o resultado.

O sinal de que a fronteira está no lugar é que nenhum dos treze desenha nada, e
nenhum handler daqui toca banco fora do `Queries`.

**Quatro assinaturas apertaram de uma vez**, pela regra da menor pergunta:
`backupDatabase` devolvia o caminho do arquivo, `deleteAccount` devolvia a
contagem de campanhas e um status HTTP, e `listBackups` devolvia a lista inteira
de `backupDTO` — um tipo do hospedeiro que teria feito a tela depender da forma
do JSON da API de backup. A cena descarta os três e pede `LastBackup() (nome,
tamanho, ok)`.

**E uma NÃO apertou, o que é a parte interessante.** As duas cunhagens devolvem a
LINHA do banco e não o token, porque o `hub.Deps` já pedia `MintAccountInvite`
com essa forma desde a extração dele. Encolher aqui obrigaria o `*Server` a ter
dois métodos com o mesmo nome e formas diferentes — que o compilador recusa — ou
um segundo nome para a mesma coisa, que é pior que a cena ler um campo. **Um
contrato que já existe ganha da regra**, e a linha diz isso.

### A armadilha do `.templ`: o texto da tela não tem aspas

O renome que acompanha a mudança de pacote já tinha mordido em STRING na porta.
Aqui ele mordeu de novo, num lugar que a proteção contra strings **não cobre**:
num `.templ`, o texto que a pessoa lê fica solto entre as tags.

`Convites` virou `Invites` e `Expira` virou `Expires` — e o resultado foi a tela
dizendo "**Invites abertos (1)**" e "**Expires em 7 dias**". Três frases, todas
visíveis, nenhuma delas dentro de aspas.

Quem pegou foi um teste de composição que afirmava o conteúdo do painel
remendado. Não foi sorte: é o que a seção "Testes" chama de integração, e é
exatamente o defeito que ela existe para pegar.

**A receita, depois de duas mordidas:** proteja comentário, proteja string, e
depois DIFERENCIE o arquivo contra o original comparando os nós de TEXTO —
`>…<` e o que vem entre eles. Um renome de pacote não deve mudar uma palavra do
que a pessoa lê, então qualquer nó de texto que mudou é defeito, e a lista é
curta o bastante para conferir a olho.

### Duas funções encontradas hospedadas por acidente

O compilador apontou as duas quando a cena saiu — é o ciclo funcionando como a
ALE-254 descreve, dizendo que a fronteira estava no lugar errado.

O `plural` morava na view da administração — o antigo `piloto_admin_view` — e
a CRÔNICA já o usava de lá; virou
`ui.Plural`. É a segunda vez nesta épica que uma função de apresentação aparece
hospedada numa cena por acidente de história — a primeira foi o `NoticeInternal`
da porta. **Vale esperar uma por cena daqui para frente**, e o sinal é sempre o
mesmo: o build quebra em um arquivo que a fatia não ia tocar.

## `web/master`: a MAIOR cena, com a MENOR porta

A Mesa do Mestre saiu na ALE-278 — treze arquivos, ~3.900 linhas, trinta rotas:
o trilho, os nove catálogos, o bestiário, os encontros e o improviso. E o
VERBETE, que foi junto.

**Ela tem DOIS métodos na porta**, contra treze da administração, nove da porta
de entrar e seis da forja. O contraste com a administração é o que ensina, e não
o número: aquela é um painel de CONTROLE sobre serviços do servidor, então tudo
que ela faz é do hospedeiro. Esta desenha o LIVRO, que é `go:embed`, igual para
todo mundo e sem uma linha de banco. Medido nos doze arquivos de produção, o
`*Server` era alcançado em exatamente dois lugares: o `WritePage` de cada
handler e o `s.livro.endereco`. **A porta é fina quando a cena não precisa do
servidor, não quando alguém foi disciplinado.**

### O verbete não é cena própria, e a medição é que disse

O `/verbete` tinha rota no hospedeiro porque a caixa que um elo abre pertence à
CASCA — ela aparece em qualquer cena. Isso continua verdade, e mesmo assim ele
foi para cá: o handler dele lê `groupForEntry`, `knownTab`, `CrossRefEntry` e
`SpellAugments`, que são o desenho do acervo. Ele é uma rota fina sobre a cena
dos catálogos, filtrada a uma entrada.

Isto corrige o que ficou escrito na fatia do `web/bookui`: **o verbete não estava
destravado.** O `bookui` levou os primitivos — `Chunk`, `CrossRef`, `PageSeal`,
`BookAddress` —, não o agrupamento por aba, e a nota daquela fatia leu o
destravamento do trilho como se valesse para os dois. Quem desfez foi medir o
que ele LÊ em vez de reler a nota.

### O bestiário é lido pela Mesa, e por isso ele exporta

Nove símbolos saem daqui para o `api`: a Mesa tem um painel de bestiário que é o
MESMO desenho da cena do mestre, parametrizado pelo endereço (`BestiaryView.Base`).
Isso é cena alcançando o miolo de outra, e é o preço aceito — a alternativa era
um segundo desenho da mesma criatura mantido em dois lugares. A direção continua
legal: o `api` importa `web/master`, nunca o contrário.

### `catalog.Resource` direto: o mesmo achado da forja, três fatias depois

O `improviso` lia `catalog.Resource("gm-tables")` e `("dungeon-design")` sem
passar pelo `book` — sem ciclo, sem erro, só a divisão vazando por baixo. É
letra por letra o que o guarda da forja pegou no primeiro dia com o `items.go`.
As sete formas foram para `book/gm_tables.go` pela regra que aquele achado
deixou: **o destino de uma função é a DEPENDÊNCIA dela.**

O guarda daqui nomeia esse caso na mensagem de erro, porque a lista de
permitidos tem seis leitores do livro e o caminho curto para o sétimo é chamar
o catálogo direto.

### Os arquivos, e o que a mudança de casa cobrou de arrumação

O `routes.go` chegou com 600 linhas e QUATRO famílias de handler que não se
chamam. Ele virou cinco: o registro das trinta rotas continua num lugar só — é
ele que o `api` chama, e espalhá-lo daria à cena quatro portas de entrada —, e
os handlers foram para `bestiary_routes.go`, `collection_routes.go`,
`encounters_routes.go` e `improv_routes.go`. Arquivo é unidade de
RESPONSABILIDADE e de conflito de merge, não de leitura.

O que NÃO coube: `bestiary.templ` (639) e `collection.templ` (609) seguem acima
do teto. Dividir componente `templ` é mais invasivo que mover função, e ficou
para quem tiver razão para abrir esses arquivos.

Um arquivo saiu inteiro de passagem: `piloto_catalogos_do_personagem.go` estava
com `package api` e um `import ()` e nada mais — vazio desde que o `book` levou
as 330 linhas dele (`4260797d`), três fatias antes.

### O renome mordeu num lugar novo: o `@componente(...)` PARECE nó de texto

A receita das duas fatias anteriores — proteja comentário, proteja string,
depois DIFERENCIE os nós de texto — foi seguida, e ainda assim escapou coisa. A
máscara que protege texto num `.templ` procura o que está entre `>` e `<` sem
`{`/`}` no meio, e três formas passam por esse funil sem serem texto:

- **`@componente(atual)` sozinho numa linha** entre duas tags. Vinte e sete
  chamadas ficaram para trás, e quem acusou foi o compilador — a declaração
  tinha sido renomeada, então todo uso sobrevivente virou símbolo indefinido.
- **Uma linha com chamada e strings**, como `@crBox(v.Base(), "nd-min", …)`: as
  strings viram marcas sem chave, e o que sobra passa a caber no funil.
- **Texto dentro de bloco de FILHOS**, `@ui.SectionLabel(…) { Dificuldade }`.
  Este é o inverso dos dois: é texto de verdade e a máscara NÃO o protegeu,
  porque ele é delimitado por chaves e não por tags. `Dificuldade` virou
  `Difficulty` na tela.

As três lições, em ordem de valor: **o compilador é rede COMPLETA para o
identificador** (declaração renomeada ⇒ uso sobrevivente não compila), e por
isso nenhuma das duas primeiras chegou perto de sair; **para o TEXTO não há
rede nenhuma** além do diff, então o diff é obrigatório; e **um diff de nós de
texto que usa a mesma máscara do renomeador herda o ponto cego dela** — o
primeiro que rodei disse "0 alterados" sobre regiões que ele também escondia.
O que achou o `Dificuldade` foi a definição ESTRITA: é prosa a linha sem `@`,
sem parênteses, sem `=` e sem chaves.

### Os comentários ficam falando dos nomes velhos, e nenhum guarda pega

O renome saiu do corpo do código e **não** dos comentários — de propósito, porque
comentário é o que o renomeador tem de proteger. A consequência é que toda
docstring passou a nomear uma função que não existe: `// handleImproviso desenha
a cena…` em cima de `handleImprov`. Foram **105** num pacote só.

Isso não era o guarda da ALE-285: ele prende citação de TESTE e de CAMINHO DE
ARQUIVO, e um nome de função não é nem um nem outro. **A rede existe desde a
ALE-286** — o `TestNoCitationNamesAMissingSymbol` —, e a sonda de duas linhas
abaixo foi o que a tornou possível.

O que achou foi uma sonda de duas linhas que vale repetir na próxima fatia:
**colete tudo que o repositório DECLARA, depois procure nos comentários os
tokens que parecem identificador e não estão na lista.** Ela é barata, e o
resultado veio com o passivo junto — dez citações mortas ANTERIORES a esta
fatia, apontando para nomes que outras extrações já tinham trocado
(`casaBusca`→`search.Matches`, `dobra`→`search.Fold`,
`NDDeGrupo`→`PartyChallengeLevel`, `requirePagina`→`requirePage`,
`rotuloDeSecao`→`ui.SectionLabel`, e a docstring do próprio `ConditionName` no
`book`, que ainda se chamava `nomeDaCondicao`).

Duas ressalvas para quem for repetir. A sonda só pode trocar automaticamente o
que **não pode ser prosa**: `filtra`, `aperta`, `empilha` e `sorteio` viraram
nomes ingleses no código e continuam sendo palavras portuguesas no comentário
ao lado — trocá-las corromperia a frase. O filtro é camelCase ou inicial
maiúscula. E o RUÍDO é grande: tag HTML (`fieldset`, `h4`), nome de pacote
(`book`, `api`), variável de ambiente (`LIVRO_PDF`) e componente da SPA morta
(`VirtualList`, `DialogHeader`) casam com o padrão e não são defeito — o último
grupo é a procedência que esta casa valoriza.

### O teste se dividiu pela quinta vez, e um caso mudou de camada de verdade

38 casos puros ficaram, 30 de composição voltaram para o `api`. Quatro estavam
no meio: liam o miolo da cena E subiam um `Server`. Eles ficaram aqui dirigindo
as rotas do PRÓPRIO pacote (`Routes` num `chi` com o dublê), porque a composição
que eles provam é a da cena — rota, handler, componente —, e não a do
hospedeiro. Ao hospedeiro cabe a outra pergunta, que é se a cena está montada e
atrás do login.

**O dublê da porta precisa RENDERIZAR.** O primeiro `WritePage` do dublê não
fazia nada, e onze subcasos reprovaram dizendo "a cena não desenha o próprio
nome" com a cena inteira: a resposta saía 200 com corpo vazio e todo
`strings.Contains` passou a medir a ausência da casca. Dublê que não escreve
nada é pior que dublê nenhum, porque ele responde 200.

### Dois casos derivavam o esperado do código sob teste, e a fronteira os expôs

`TestTheSearchInTheUrlHoldsOnAColdLoad` chamava `loadCollection` com os mesmos
critérios da página e afirmava que a página continha o número devolvido por ela.
`TestTheEncounterInTheUrlHoldsOnAColdLoad` fazia o mesmo com
`loadEncounters(...).Difficulty()`. Nos dois, um erro na conta sairia dos DOIS
lados e o guarda ficaria verde — é o que o CLAUDE.md da raiz proíbe com todas as
letras.

Nenhum dos dois foi escrito com má-fé, e é por isso que vale registrar COMO
apareceram: enquanto tudo era um pacote só, chamar a função ao lado era
gratuito. **A fronteira não consertou nada sozinha; ela tornou visível uma
escolha que não parecia escolha** — a mesma coisa que ela fez com os arquivos de
teste que misturavam camadas.

Os dois passaram a afirmar valor escrito à mão. O do encontro traz a conta do
livro no comentário: ogro ND 4, dois deles → 4 + 2·log2(2) = ND 6 (p282); 6
menos o nível 3 dá diferença 3, acima da faixa "Difícil" → **Mortal** (p281). O
primeiro palpite foi "Desafiador", que não é sequer um dos cinco rótulos que a
tabela produz — a prova de que escrever à mão só vale lendo a REGRA, e não
chutando.

## `web/characters`: a lista, e a PORTA que o teste do hospedeiro obrigou a abrir

A cena de `/personagens` saiu na ALE-278: três arquivos, ~750 linhas, TRÊS
métodos na porta. Ela lista os heróis de quem está olhando, e a forja — que
nasce um herói — continua irmã dela no mesmo endereço, montada ao lado no
roteador do `api`.

**A montagem de uma cena dentro da outra era organização, não dependência.** O
`CharacterRoutes` chamava `forge.Routes(r, forge.New(s))`, e isso parecia
acoplamento até ser medido: o `chi` não liga para quem registra o quê. A linha
subiu para o roteador e as duas cenas ficaram lado a lado, sem uma conhecer a
outra.

### Duas camadas saíram antes, e as duas pagam fatias adiante

- **A identidade visual** (`ui.NameHue`, `ui.NameGradient`, `ui.Monogram`) era
  lida por CINCO famílias do `api`. O arquivo original tinha quatro funções e só
  três foram: o `papelNaCampanha` decide entre "Mestrando", "Jogando" e "Mesa de
  X", que é regra de quem é o quê numa campanha — o kit não pode saber disso.
  **A linha divisória dentro de um arquivo é a mesma que entre arquivos.**
  E um nome MENTIA: `gradienteDaCampanha` é chamado com o nome do PERSONAGEM
  desde a ALE-239.
- **As habilidades de raça** viraram `book/race_traits.go`. O arquivo lia
  `catalog.Resource("race-defs")` DIRETO — o terceiro caso desta família, depois
  do `items.go` da forja e do improviso do trilho do mestre. Ele era lido por
  personagens E pela ficha, então não era de nenhuma das duas: era do livro.

### O que a porta NÃO pede, e por que o menor não é o mais estreito

O `sheetFromDTO` do hospedeiro esteve na lista e saiu depois de medido: ele é um
invólucro de UMA linha sobre `sheet.Compute(catalogs, dto)`, e a cena já tem o
`Catalogs()`. Pedir os dois é pedir a mesma coisa duas vezes.

É a regra da menor pergunta com uma nuance que as portas anteriores não tinham
mostrado: **o menor não é o mais estreito, é o que não se repete.** A porta da
administração aprendeu que um contrato que já existe ganha da regra; esta
aprendeu que uma dependência que já foi concedida também.

E o `HeroCardOf` deixou de ser método por isso: de tudo que a `Deps` oferece ele
usa só o motor, e como a FICHA reaproveita quatro campos do cartão, mantê-lo
método obrigaria o hospedeiro a montar uma `Scene` inteira para pedir um cartão.

### O teste do hospedeiro obrigou a cena a exportar a montagem, e vale saber por quê

Nove casos precisam de banco DE VERDADE: eles prendem o caminho BANCO → PALCO —
personagens gravados saem na lista, com a contagem, os vizinhos e a Defesa que a
ficha mostra. Este pacote não pode provar isso: não tem banco, e importar o
`db/testdb` junto com um `*api.Server` seria o ciclo que a divisão existe para
evitar.

Então `Load` e `SceneBody` são exportados, e o comentário de cada um diz que o
consumidor de hoje é uma BANCADA. **Isso não é abrir o pacote para o teste** — é
a fronteira ficando onde ela pode ficar: a cena diz como montar a si mesma, o
hospedeiro prova que o que está no banco chega até lá. Mesma direção do
`master.LoadBestiaryFrom`, que a Mesa lê de fora pelo mesmo motivo.

O que NÃO foi exportado, e é o contraste que importa: o `corpoDoBotao`, um
parser de nove linhas que a bancada usa. Ele foi COPIADO, pela regra que a fatia
da porta deixou escrita — importar do que está sendo testado faz o teste andar
junto com o defeito.

### Um caso afirmava duas camadas e se dividiu em duas

`TestALoneHeroGetsNoInventedNeighbor` prendia que o `peekAt` devolve nulo fora
do trilho **e** que o HTML do herói único não desenha um "Próximo". A segunda é
o que a pessoa vê e ficou no hospedeiro; a primeira é aritmética de índice e
virou `TestNoNeighborIsInventedOutsideTheRail` aqui — sem banco, sem servidor, e
com o controle que faltava (o índice VÁLIDO acha, senão um `peekAt` que
devolvesse nulo sempre passaria nas duas asserções).

### ~170 citações mortas herdadas, contadas aqui e fechadas na ALE-286

A sonda de citação de SÍMBOLO que a fatia do trilho do mestre inventou foi
rodada nos três pacotes que esta fatia tocou. As 22 desta fatia foram
consertadas. As outras são passivo de extrações ANTERIORES, e são muitas:
**~130 no `book`** (a extração dele renomeou tudo e as docstrings ficaram
falando de `catalogosDoLivro`, `filtraCriaturas`, `chaveDoNome`, `poderDoLivro`…)
e **~40 no `web/ui`** (`classesDoBotao`, `rotuloDeSecao`, `cascaNua`,
`molduraDePainel`…).

Nenhuma delas era pega por guarda quando isto foi escrito, e o parágrafo dizia
por quê: nome de função não é nome de teste nem caminho de arquivo. **Ele
cumpriu o que a linha final prometia** — contar transformou "tem umas dessas" em
trabalho que alguém pôde fechar, e a ALE-286 fechou: 466 medidas, 466
consertadas, e um guarda no ar.

O número aqui envelheceu para MENOS e para MAIS ao mesmo tempo, e vale dizer as
duas: eram ~170 quando a sonda rodou em três pacotes, e viraram 466 quando ela
rodou na árvore inteira depois de mais três fatias renomearem coisas. **Uma
medição parcial não é um número menor: é um número de outra pergunta.**

## `web/reader`: o arquivo que tinha DUAS responsabilidades

O leitor saiu na ALE-278, e é a primeira extração em que um arquivo se dividiu
porque fazia duas coisas — não porque era grande.

O `piloto_livro.go` lia `LIVRO_PDF` no boot, cunhava o dígito de cache, avisava
sobre linearização e SERVIA o arquivo com faixas; e desenhava a página que abre
esse arquivo numa página, com o termo destacado. **A divisão é a de sempre:
dependência.** O que ficou chama `os.Stat` e devolve um `http.Handler` sobre um
arquivo do disco do dono da mesa. O que saiu desenha uma página e não sabe onde
o arquivo está — só o endereço dele.

### A pergunta que o VALOR já responde

"Há livro configurado?" não entrou na porta. Sem `LIVRO_PDF` o hospedeiro não
monta endereço nenhum, então `BookAddress().Base == ""` diz isso.

É a regra da menor pergunta chegando ao limite dela, e vale ter as três nuances
lado a lado, porque foram descobertas em fatias seguidas:

- a administração: **um contrato que já existe** ganha da regra;
- personagens: o menor não é o mais estreito, é **o que não se repete**;
- o leitor: às vezes a menor pergunta é **nenhuma**.

### E o `templ` não alcança a porta

O `.templ` do leitor chamava `EstaticoDoPiloto` direto para o worker do pdf.js.
Um componente não tem acesso ao `s.deps`, então o endereço passou a viajar no
VIEW. É a decisão do `ui.Page.Asset` outra vez: **a casca recebe o que ela não
pode conhecer**, e vale para qualquer valor que um componente precise e só o
hospedeiro saiba.

### O guarda NÃO recusa a biblioteca padrão, e isso está escrito nele

Um `os.ReadFile` aqui passaria pelo guarda de fronteira. O que segura é a porta:
sem caminho de arquivo atravessando a fronteira, não há o que ler. A observação
está no próprio guarda, porque um leitor que confie nele para isso vai se
enganar.

## `web/campaigns`: a porta mais LARGA, e o SQL que a cena montava

A nona cena saiu na ALE-278: quatro telas com um endereço cada — a lista, a
campanha aberta, a folha em branco e a carta de entrar —, e **onze** métodos na
porta, contra treze da administração e dois do trilho do mestre.

O tamanho não é vício nem virtude: é o que a cena É. O trilho do mestre desenha
o livro embutido e não toca banco; esta é a tela onde uma campanha nasce, muda
de nome, ganha membro e é apagada. **Três das quatro telas ESCREVEM.**

### Duas coisas foram recusadas, e as duas já tinham precedente

**O `campaignListDTO` não atravessa.** A cena o consumia direto, e ele é a
resposta de `GET /campaigns` — tag `json:` em cada campo, nome do fio. É letra
por letra o que a administração recusou com o `backupDTO`. A cena declara o
`ListRow`, e o hospedeiro MAPEIA do `campaignList` que já existe: o outro erro
possível aqui era duplicar a consulta do lado da cena, que é a família de
defeito que esta épica mais encontrou.

**O SQL não atravessa.** A cena montava `setBuilder` + `execTouched` +
`"UPDATE campaigns"` à mão. **Cena que compõe SQL é cena com o banco dentro**, e
o remédio é a PERGUNTA: o `SaveText` existe porque quem sabe o nome da coluna,
que vazio é NULL e que a linha tem um `updatedAt` a tocar é o hospedeiro.

O `Queries` continua na porta, e a concessão é a mesma da forja e da
administração — o sinal de que ela está no lugar é nenhum handler tocar banco
fora dele.

### `IsAdmin` colidiu, e a colisão ensina o inverso do `MintAccountInvite`

O `*Server` já tinha `IsAdmin(email string) bool`, que a administração pede e
que olha a CONFIGURAÇÃO. A cena precisa de outra coisa: "quem está pedindo
AGORA é admin?". Dois métodos de mesmo nome e formas diferentes o compilador
recusa.

Onde o hub fez a administração ceder — mesmo contrato, mesma pergunta —, aqui o
certo foi um nome novo (`RequesterIsAdmin`), **porque a pergunta é outra**. A
regra completa fica sendo: um contrato que já existe ganha quando é a mesma
pergunta; quando só a cara é a mesma, forçar um nome só junta duas coisas
diferentes.

### O `JoinRefusal`, e a cena colapsando duas travas em uma

O hospedeiro tem sete sentinelas para as sete travas do `joinTable`; o
`JoinRefusal` da cena tem SEIS valores. A diferença é decisão da tela: "personagem
não existe" e "personagem é de outra pessoa" viram a mesma frase — "Escolha um
herói seu" — e distinguir diria a um estranho se um id existe.

É a porta de entrar acontecendo de novo, com a granularidade escolhida pelo lado
que escreve a frase.

### O guarda que reprovou a SI MESMO, uma fatia depois de eu escrever a regra

Escrevi neste guarda uma lista de recusa da biblioteca padrão com `os` e
`path/filepath`, por precaução. Ela reprovou o próprio arquivo do guarda, que
importa `os` para ler o diretório.

O guarda do `campaign`, escrito HORAS antes, tem a frase "lista de perigo
imaginado envelhece; lista de defeito acontecido, não" — e eu a violei na fatia
seguinte. A lista saiu, e a ausência dela ficou registrada no arquivo com o
motivo. Vale a pena notar o contraste: lá a recusa de `database/sql` era uma
tentação MEDIDA; aqui o `database/sql` é legítimo, porque esta cena grava.

### O SEGUNDO guarda que media o próprio diretório

`TestNoFlowContentInsideASectionLabel` varria `os.ReadDir(".")` no `api`. Quando
as campanhas mudaram de pacote sobraram **ZERO** usos de `@ui.SectionLabel` lá —
os oito arquivos que usam o componente estão todos em `web/*`.

E aqui está a diferença que vale guardar, porque ela julga a regra do
denominador: **o guarda do foco não tinha piso e teria passado verde medindo
metade; este tinha `visitados == 0` e falhou ALTO no instante em que deixou de
medir.** O controle não é enfeite — é o que transforma "não mediu" em vermelho.

Ele foi para o `convention/` caminhando a árvore, e ganhou o segundo piso (o de
arquivos LIDOS), porque a caminhada pode encolher sem zerar.

## `web/sheetui`: a MAIOR cena, e o SQL que ela montava em quatro lugares

A décima cena saiu na ALE-278: 36 arquivos de produção, sete abas, mais de trinta
mutações, e **dezoito** métodos na porta — contra onze das campanhas, treze da
administração e dois do trilho do mestre.

**O nome tem sufixo, e a medição é que decidiu.** `sheet` já é a FORMA do dado
(`CharacterDTO`, `Load`, `Compute`) e esta cena o lê 148 vezes em 20 arquivos:
com o mesmo nome, cada um desses vinte arquivos carregaria um apelido no import.
O plural que resolveu o `web/characters` e o `web/campaigns` não serve aqui —
a ficha é uma. O GLOSSARIO registra a linha.

### Cena que compõe SQL é cena com o banco dentro, quatro vezes

A porta das campanhas deixou a frase escrita e esta cena a provou em escala: ela
montava `setBuilder` + `"UPDATE character_items"` e `"UPDATE characters"` em
QUATRO lugares, e um deles abria a transação inteira — `BeginTx`, plano,
escritas, `Commit`. Viraram três perguntas:

- `SaveItemOverlays` — a melhoria e o material de um item. Atravessam a LISTA e o
  nome do material, não o JSON nem o `sql.NullString`.
- `SaveChoices` — as cinco colunas de escolha, com nulo querendo dizer "não toque
  nesta". A versão anterior mandava a string `"raceAttributeChoices"` de dentro da
  cena, que é SQL viajando com outra roupa.
- `ApplyPowerTempHp` — a reserva de PV temporários sob o vale-o-maior (p256).

**A terceira pagou sozinha**, e é o achado que vale repetir: a transação que a
cena montava era a MESMA do `applyPool` da rota JSON, escrita de novo. Duas
transações sobre a mesma regra divergem no dia em que uma das duas ganhar um
passo. As duas passaram pelo `applyPoolTx`, e o handler HTTP virou a casca que
traduz o resultado em resposta — e então a ALE-277 apagou o handler, porque
ninguém o chamava. Sobrou uma chamadora, e a extração continua sendo a certa: o
que a justifica não é o número de chamadores, é a regra ter UM lugar.

### A porta ENCOLHEU em dois lugares, e as razões são opostas

`ConsumeItem` devolvia `doseUsed`, que embrulha o corpo da resposta JSON com tag
`json:` em cada campo. A cena descartava o resultado inteiro, e a única recusa
que ela precisa — a porção diária — já chega como erro: a porta passou a devolver
só `error`. É o leitor outra vez, com a menor pergunta sendo NENHUMA.

`ApplyClassLevel` era o contrário: a porta estava certa e o chamador não. O
hospedeiro devolve quatro valores e um deles é o `storedVitals`, que é tipo DELE
— uma porta que devolvesse isso não seria porta.

E uma assinatura estava simplesmente ERRADA: o `SaveCustomItem` pedia `espacos
int64`, e a coluna `slots` é REAL porque a carga do livro conta de meio em meio
(p141). **Conserta-se a porta, não o chamador.**

### As regras que precisavam do LIVRO e da FICHA ao mesmo tempo

Duas não cabiam em lugar nenhum: quantas vagas de poder o nível abre
(`WithChoicesValid`, com os benefícios de origem, os caminhos e os deuses) e qual
círculo o personagem alcança (`HighestCastableCircle`). Elas leem o catálogo E o
personagem, o `book` não pode importar o `sheet` — ele é consultado por treze
famílias —, e as duas rodam nas DUAS portas, o endpoint JSON e os comandos da
ficha.

**Decisão do dono: o `sheet` passou a importar o `book`.** A lista de permitidos
do guarda de lá ganhou uma quarta entrada, e a prosa dele mudou junto — "a ficha
é forma de DADO" deixou de ser verdade quando as regras da ficha vieram morar
nela, o que já tinha começado na fatia anterior com o `equip.go` e o
`temp_hp.go`. O preço está escrito no guarda: toda cena que importa a ficha
alcança o livro de graça, e ele é pequeno só porque oito das dez já importam
`book` direto.

As outras três alternativas ficaram registradas na issue, e a que mais tenta é a
pior: `api` importando `web/sheetui` para a rota JSON ler regra de um pacote de
APRESENTAÇÃO.

### `catalog.Resource` direto, três vezes na mesma cena

Quinto, sexto e sétimo casos da família que o `items.go` da forja abriu.
`Resource("class-powers")` era lido em DOIS arquivos, com o mesmo `Unmarshal`
anônimo, e `Resource("activations")` num terceiro. Foram para o `book` —
`ClassPowerFlags`, `PowersThatTeachSpells` e `Activations` —, pela regra que
aquele achado deixou: **o destino de uma função é a DEPENDÊNCIA dela.**

O que NÃO foi junto é a metade que a cena decide: se o botão está ativo, que
crachá o limite desenha e a frase da recusa. Aquilo lê a tabela e a ficha, e a
voz é da tela.

E o que sobrou de `catalog` na cena está CONTADO no guarda, sem fingir que coube:
quatro chamadas (`LookupSpell`, `IsCondition`) e três assinaturas com
`catalog.Spell`/`catalog.Augment`. Elas não são a mesma coisa que o `Resource` —
são o acessor tipado, e o hospedeiro usa o mesmo. Unificá-lo com o `book.Spell`
mexe nos dois lados e é trabalho próprio.

### Cinco cópias do mesmo corpo, achadas por uma COLISÃO de nome

O renome de pacote deu o mesmo nome inglês a duas transcrições dos mesmos sete
valores e o compilador reclamou. Puxando o fio saíram cinco:

- **as sete categorias de proficiência**, em TRÊS lugares — a lista de chaves do
  `book`, um conjunto no `character_abilities.go` e a lista com rótulo e grupo da
  aba. Viraram `book.ProficiencyCategories`;
- **os seis atributos**, transcritos uma segunda vez ao lado de
  `engine.AttributeKeys`. Viraram `engine.IsAttributeKey`;
- **o leitor de blob de ids**, escrito TRÊS vezes (escolhas, condições ativas,
  proficiências). Virou `sheet.UnmarshalStrings`, ao lado do `MarshalStrings`
  que já era o caminho de ida;
- **o `*T` → `sql.Null*`**, que o `api` tinha e a cena precisava. Foi para o
  `plataforma`, ao lado do `NullToPtr` que faz o inverso;
- **o filtro dos itens equipados** que os dois tetos do livro contam. Virou
  `sheet.EquipLimitErrorOver` — o `EquipLimitError` sozinho não via QUAIS itens
  entram na conta, e esse pedaço é regra.

Nenhuma das cinco tinha guarda, e nenhuma daria erro: três transcrições do mesmo
dado compilam e ficam verdes até uma divergir. **A colisão foi sorte, e vale
dizer isso** — o que a produziu foi um renome em massa dando o mesmo nome a duas
delas.

### O teste se dividiu pela sétima vez, e o instrumento errou duas

25 casos puros ficaram, 126 blocos voltaram para o `api`. A divisão é a de
sempre — unitário onde a regra mora, integração onde a composição acontece —, e
os ajudantes que os dois lados usam foram COPIADOS num
`api/sheetui_helpers_test.go`, pela regra que a fatia da porta deixou escrita:
importar do que está sendo testado faz o teste andar junto com o defeito.

**O classificador automático errou duas vezes, e as duas valem para a próxima
divisão grande.** A primeira: ele classificou POR ARQUIVO, e os ajudantes de
teste são do PACOTE — um caso "puro" chamava um `combatant` declarado noutro
arquivo que tinha ido embora, e só o compilador acusou. O fecho tem de ser
global. A segunda: a semente de "isto precisa da bancada" listava `novoPiloto` e
`seedX`, e não `*Server` — casos que pegavam o servidor direto passaram por
puros.

E um recorte por marcador de texto levou junto QUATRO funções vizinhas que ele
não devia tocar. O que denunciou foi o build; o que consertou foi o `git show` do
commit anterior. **Corte de bloco em arquivo grande se confere com o diff de
declarações**, e não com a leitura do trecho recortado.

### Três nomes colidiram ao voltar para o hospedeiro

`combatant` já era um TIPO do `initiative_rules.go`, `sheetCombatant` já era um
ajudante do `test_helpers.go`, e o `aFichaDe` da bancada tinha sido renomeado
para `sheet` — que ESCONDE o pacote `sheet` no arquivo inteiro. O último é o mais
traiçoeiro: ele compila enquanto ninguém precisar do pacote naquele arquivo.

**Um símbolo de teste que volta para um pacote de 20 mil linhas encontra um
espaço de nomes que a cena não tinha**, e é onde o `grep` de menos de cinco
ocorrências vale mais.

## `web/table`: a ÚLTIMA cena, e o roteador que morava dentro dela

A Mesa saiu na ALE-278 e fechou a fila: **47 arquivos de produção, ~14.300
linhas, vinte rotas** e **trinta e um** métodos na porta — contra dezoito da
ficha, treze da administração e dois do trilho do mestre.

**Com ela o `api` ficou sem uma única cena.** Ele era 188 arquivos e 105 mil
linhas quando a épica começou; hoje são **50 arquivos de produção e 9.303
linhas**, e **nenhum `.templ`**. O que sobrou é a API JSON e a composição.

### A porta mais larga não é indisciplina: é o que a cena É

O contraste que ensina continua sendo com o trilho do mestre, que tem DOIS
métodos. Aquele desenha o livro embutido e não toca banco. Esta é a única cena
que **movimenta estado ao vivo** — abre e encerra cena, move peça, pinta
terreno, mede distância, vira turno e empurra tudo para quem está olhando, por
dois stores em memória, um hub de SSE e um barramento.

Os quatro stores atravessam INTEIROS (`Boards`, `Sessions`, `Presence`, `SSE`,
`Bus`), e é a mesma concessão do `Queries`: são tipos de OUTROS pacotes, o
vocabulário do domínio ao vivo, não o hospedeiro com outro nome. Embrulhá-los
método a método daria oitenta entradas e nenhuma fronteira a mais.

### O ROTEADOR DO APP morava no arquivo de rotas da Mesa

O `WebRouter` — o mux que monta porta, hub, campanhas, personagens, forja,
ficha, grimório, mestre, buscador, livro, leitor e administração — estava dentro
de `piloto_mesa_routes`. Não era desenho: a Mesa foi a primeira cena do
piloto, o mux nasceu no arquivo dela, e as onze cenas seguintes foram sendo
penduradas ali por dez fatias.

**Se ele tivesse ido junto, o `api` ficaria sem saber montar o app.** É a mesma
forma do guarda que media o próprio diretório: um símbolo que mora no arquivo
errado só aparece quando o arquivo se move.

E o corte teve uma metade que a compilação não pega: o grupo do `requirePage`
ficou de fora, a cena foi montada anônima, e **cinquenta casos ficaram
vermelhos de uma vez** dizendo coisas sem relação ("a cena não disse que não há
mapa"). Quem decide que a cena está atrás do login é o hospedeiro; a linha que
faz isso é um `r.Group` de três linhas, e ela não estava em lugar nenhum do
diff.

### A BANCADA do repositório inteiro também morava lá

`pilotoFixture`, `novoPiloto`, `.pede()`, `.posta()` e `.scene()` são lidos por
**49 arquivos de teste** do `api`, e viviam em `piloto_mesa_test`. Era a
fatia 2 desta épica acontecendo pela metade: o molde do BANCO virou pacote
(`db/testdb`) antes da primeira cena sair; esta bancada ficou onde estava porque
o tipo `api.Server` a prende ali — um pacote de bancada que o importasse seria
importado de volta pelos testes dele.

Ela virou `api/fixture_test.go` antes do `git mv`. O que a mudança conserta é o
nome: pelo prefixo, a bancada do repositório teria ido para `web/table`.

### O estado que era CAMPO do `*Server` e é da cena

A LENTE (quem está vendo como a mesa vê) e as ABAS ESCOLHIDAS (que tabuleiro
cada um está olhando) eram campos do `Server`. O argumento delas não mudou —
moram no servidor e não num sinal do navegador porque o stream não pergunta nada
a ninguém —, mas o DONO é a cena: a pergunta só existe nesta tela.

**E aí veio a única armadilha de desenho da fatia.** Com o estado na `Scene`, um
`table.New(deps)` por requisição dá um estado NOVO a cada pedido: metade da mesa
não veria a lente da outra metade. O `WebRouter` chamava `New` uma vez e estava
certo; os TESTES chamavam de novo, e oito casos de lente e de puxão ficaram
vermelhos. O servidor passou a guardar UMA instância (`s.tableScene`), como
guarda um store — o dono continua sendo a cena.

### TRÊS guardas mediam o diretório e quebraram, e é a terceira vez nesta épica

- **`TestEveryMarkerColorCanBePainted`** lia `piloto/piloto.src.css`, um caminho
  relativo ao `api`. Foi para o `convention/` lendo pela raiz. O irmão dele — o
  FALLBACK da cor — ficou na cena: aquilo é regra de uma função, isto é
  convenção do repositório.
- **`TestNoSceneCommandUsesTheDefaultTab`** varria `piloto_*.go` do próprio
  diretório e passou a achar ZERO. Foi para o `convention/` caminhando a árvore.
- **`TestAWireTagStartsLowercase`** ENUMERAVA quatro pacotes, e a Mesa saiu da
  lista: a contagem caiu abaixo do piso. Ele passou a caminhar a árvore, que é o
  que restaura a amostragem — *enumerar é remendo, e o pacote novo tem de nascer
  medido.*

**Os três falharam ALTO, e os três tinham piso.** É a diferença que a fatia das
campanhas já tinha registrado: o guarda do foco não tinha denominador e teria
passado verde medindo metade.

### O renome veio ANTES do movimento, e isso é método

536 identificadores saíram em inglês num commit próprio, com o pacote ainda
compilando — **o compilador é rede COMPLETA para identificador**, e ele só é
rede enquanto declaração e uso estão no mesmo lugar. O diff do movimento ficou
sendo estrutura pura; o do renome, nome puro.

O que o compilador NÃO pega ficou registrado na fatia do renome, e vale repetir
a parte nova: **a sonda de prosa do `.templ` tem ponto cego em linha com
atributo.** `<p class="…">O mestre está montando a cena.</p>` passou pela
definição estrita (linha sem `@`, sem parênteses, sem `=` e sem chaves), e
`cena` virou `scene` na tela. Quem denunciou foram dois testes de composição que
afirmam a frase. **O reparo certo é por NÓ DE TEXTO (`>…<`), não por linha.**

### O guarda nasceu vermelho, e a primeira sabotagem saiu INERTE

Sabotei com `database/sql` e o guarda passou — porque ele filtra por prefixo do
módulo e é cego para a biblioteca padrão, que é exatamente a tentação que a
prosa dele nomeia. É letra por letra o achado do guarda do `campaign`, e o
remédio é o mesmo: uma lista de RECUSA com **uma** entrada, a medida.

A segunda sabotagem também saiu inerte, por outro motivo e igualmente
instrutivo: `campaign.ValidateName` não existe, então o build quebrou e a
sabotagem nunca chegou. **Verde depois de sabotar só significa alguma coisa
quando a sabotagem CHEGOU** — e a terceira, com `campaign.Name`, reprovou o
guarda como devia.

## A mesa que não aceitava ninguém (ALE-287)

O `CreateCampaign` gerado pelo sqlc não escreve o `inviteToken`, então **toda
mesa aberta pela tela nascia com a coluna nula**. E o `joinTable` recusa quem
não é o dono já no `!c.Invitetoken.Valid`, antes de olhar o que a pessoa
digitou. As duas coisas juntas: a mesa não aceitava ninguém, e as únicas em que
alguém entrava eram as seis da `seed.sql`, com `seedtoken-0N` escrito à mão.

Não era funcionalidade faltando — era defeito entregue, e ele sobreviveu à
migração inteira da SPA para o Datastar.

### Por que a suíte passava por cima

A bancada semeia campanha com o token DADO (`seedCampanha(t, s, dono, nome,
convite)`), e todos os casos de entrar usavam essa porta. **O teste fornecia o
que a produção nunca fornecia.** É a família do "esperado calculado" com o
arranjo no lugar do valor: um dado de fixture que o código sob teste não sabe
produzir esconde exatamente o defeito de quem deveria produzi-lo.

O caso que fecha isso (`TestACampaignBornOnScreenLetsAPlayerIn`) chama a mesma
função que a cena chama, com os mesmos parâmetros e mais nada.

### A forma do conserto

Cunhar mora no `campaignRules.createCampaign` e não no `INSERT`, e é isso que
faz os DOIS caminhos passarem por ele — a cena de campanhas e a rota JSON que a
suíte de e2e usa como fixture. É `UPDATE` depois do `INSERT` e não coluna com
`DEFAULT` porque o token é `crypto/rand`, e SQLite não tem de onde tirar isso.

**Não há migração de dados, e a razão vale saber:** as mesas abertas antes desta
issue continuam sem link, e quem as conserta é o botão `Gerar link` da própria
tela. Um `UPDATE` varrendo o banco escreveria token para mesas que o dono talvez
nem use, e o botão custa uma linha e resolve na hora em que alguém precisa.

### Dois convites, duas VIDAS — e a nota virou parâmetro

O `ui.MintedInvite` é o mesmo widget para os dois, e por um tempo ele trazia a
frase do convite de CONTA colada: *"Cada convite serve para UMA conta."* O de
campanha vale enquanto o mestre não gerar outro. Escrever a frase errada embaixo
de um link reutilizável faria o mestre gerar um por jogador — e derrubar o dos
anteriores a cada vez.

> O que o componente já trazia de lição e continua valendo: **o link carrega o
> CAMINHO, e quem prefixa a origem é o navegador**. Com `r.Host` o link nascia
> apontando para a porta errada atrás de proxy, e link de convite existe para ser
> mandado a outra pessoa — host errado é link morto.

## A presença que nunca acendeu (ALE-287)

A Mesa desenha um anel por carta do elenco dizendo quem está com a aba aberta, e
ele ficava CINZA para sempre. Quem preenchia o registro era o handshake da rota
`/events` da SPA, apagada na ALE-277 por não ter consumidor: **ninguém em
produção chamava `PresenceRegistry.Join` desde a ALE-272.**

Cinza quer dizer "fora da mesa", então a tela dizia "todos fora" com cara de
medição — e o mestre agia sobre isso.

O registro passou a acontecer no fluxo próprio da Mesa
(`web/table/stream.go`), e ele é o lugar certo por três coisas que já tem e uma
rota nova teria de recriar: QUEM pede, EM QUE sessão, e o `r.Context()` que o
servidor cancela quando a aba fecha — a saída, que é a metade difícil. O papel
sai do `view.Mestre` (nil para o jogador), a mesma leitura que já decidiu o que
desenhar. Sem evento novo: o `writeTable` só manda bytes quando o HTML muda, e o
batimento de 1s leva a mudança às outras abas.

### O guarda do DESENHO existia e passava

`TestTheGmSeesWhoIsAtTheTableAndThePlayerDoesNot` já media o anel aceso, o
apagado, e que o jogador não recebe presença nenhuma — **chamando `Join` ele
mesmo**. Arranjo que a produção não sabia produzir: a terceira ocorrência da
mesma forma nesta issue, depois do token de convite e da coluna `role`.

> A regra que sai daí: **quando o arranjo de um teste usa uma porta que nenhum
> caminho de produção usa, o verde é sobre o arranjo.** Vale procurar por
> fixture que escreve coluna à mão, que chama método de store direto, ou que
> aceita um parâmetro cujo valor a produção nunca varia.

### Um teste cujo modo de falha era TRAVAR

O caso novo abre o `/stream` num `httptest.NewServer` de verdade. A primeira
versão criava o `context.WithCancel` e só chamava `fechar()` no fim — então
qualquer `t.Fatalf` no meio pulava o cancelamento, o stream ficava aberto, e o
`srv.Close()` do defer esperava por ele para sempre.

Descoberto sabotando o `Join`: o veredito veio como `panic: test timed out after
1m0s` apontando para a linha do `Fatalf`, e não como a frase que o teste
escreveu. Com `defer fechar()` logo depois do `WithCancel`, a mesma sabotagem
reprova em **0,01s** dizendo "o elenco presente = [], queria só o jogador 2".

É a família do `finally` que engole o erro de verdade (ALE-245), com outra
roupa: **limpeza que espera não pode falar mais alto que o defeito.**

## A coluna que não decidia nada (ALE-287)

`campaign_members.role` valia `'player'` em **toda linha que a produção
escreveu**: o único escritor fixava a string, e o `SetMemberRole` nunca teve
chamador. E a autorização **nunca a leu** — o `roleIn` decide por "é o dono da
campanha?", então escrever `'gm'` ali não mudaria nada do que a pessoa pode
fazer.

Ela era lida em quatro lugares, e os quatro estavam DESLIGADOS sem ninguém
saber:

| onde | o que devia fazer | o que fazia |
| --- | --- | --- |
| ordenação do elenco | o mestre primeiro | `a.Role == b.Role` sempre verdadeiro → ordem de entrada |
| `heroRow` | coroa ao lado do mestre | nunca desenhada |
| `tableRoster` | tirar o PC do mestre do grupo | nunca tirou ninguém |
| `listPlayerCombatants` | o mesmo, no popular-iniciativa | idem |

### Dois testes VERDES sobre comportamento que não existia

O `seedMember` da bancada recebia um papel e escrevia `"gm"` — **um estado que só
ela sabia produzir**. Dois casos passavam por causa disso: o
`TestTheGmComesFirstInTheCast`, que afirmava uma ordenação que a produção nunca
fez, e o `TestListMemberHelpers`, que esperava 2 combatentes onde a produção
sempre devolveu 3.

É a mesma família do convite desta mesma issue, e as duas moravam no mesmo
arquivo de fixture: **quando a bancada escreve o que a produção não escreve, o
verde é sobre a bancada.** O sinal de alerta é uma fixture com um parâmetro que
nenhum caminho de produção sabe variar.

### O que substituiu

A coluna saiu, e a coroa e a ordenação passaram a perguntar `ch.ownerId ==
campaigns.ownerId` — a **mesma verdade** que o `roleIn` usa para autorizar. Uma
regra, uma camada: a tela e a autorização não podem mais divergir sobre quem
mestra.

Os dois filtros de grupo (`tableRoster` e `listPlayerCombatants`) saíram **sem
virar a condição verdadeira**, e isso é deliberado: tornar real um filtro que
nunca filtrou MUDARIA o que a mesa mostra, e isso é decisão de produto, não de
quem apaga uma coluna morta.

### O `DROP COLUMN` passou pelo sqlc — e é a exceção da família

O guia avisa que `ALTER TABLE … ADD COLUMN` de um arquivo de migração NOVO não
entra no catálogo do sqlc (v1.31.1), e que trocar chave primária também não.
**O `DROP COLUMN` entra**: o `Role` sumiu do `models.go` e das consultas na
primeira geração. Medido nesta issue, no mesmo sqlc.

O que exigiu cuidado foi outra coisa: um `SELECT *` teria continuado a pedir a
coluna. Aqui não houve porque as três consultas que a tocavam de perto —
`GetMember` (`SELECT *`), `AddMember` e `SetMemberRole` — tinham **zero
chamadores** e saíram junto, pela regra da ALE-277.

### E o `seed.sql` foi editado à mão, porque o gerador estava quebrado

As nove linhas de `campaign_members` foram editadas no arquivo em vez de
regeradas: o `cmd/seed` dirigia a aplicação por HTTP e a ALE-277 tinha apagado
todas as rotas que ele usava. **O gerador voltou logo depois** — ver a seção
seguinte —, e a primeira coisa que ele fez foi confirmar a edição: o arquivo que
ele escreve é BYTE A BYTE o commitado.

## O gerador da seed, e o consumidor que a varredura não vê

O `cmd/seed` monta o conjunto de desenvolvimento e despeja `seed.sql`. Ele
dirigia os manipuladores HTTP em processo, e a ALE-277 apagou as **sete rotas**
que ele usava por não terem consumidor: `/auth/register`, `POST /characters`,
`GET /characters/{id}`, `/characters/{id}/vitals`, `/characters/{id}/spells`,
`…/spells/{id}/prepared` e `…/items/{id}/consume`. **O gerador parou de rodar e
nada acusou.**

### A forma que a varredura de órfãs não alcança

A varredura da ALE-277 procurava `s.handleX` — SÍMBOLO. O gerador chama por
CAMINHO EM STRING (`c.do(http.MethodPost, "/characters", …)`), e nenhum `grep`
por identificador chega perto disso. O e2e foi encontrado porque alguém foi
olhar o diretório `e2e/` de propósito; o `cmd/seed` não foi olhado.

**A lição é sobre a busca, não sobre o gerador:** quando se apaga uma ROTA, a
varredura tem de ser pelo ENDEREÇO também, e ela alcança lugares que a de
símbolo não alcança — `.ts` de teste, script de shell, `curl` em `.md`, e um
`cmd/` do próprio módulo.

### O conserto não podia ser "aponte para as cenas"

A resposta óbvia não serve: **a forja cria personagem de nível 1 com kit
inicial**, e o elenco da seed tem heróis de nível 8 a 10 com atributos, itens e
magias escolhidos. Escrever uma ficha inteira de uma vez era capacidade que só a
rota JSON tinha, e apagá-la levou junto o único caminho.

Então o gerador virou mais um CONSUMIDOR com porta declarada, como as onze
cenas: `casaDaSeed` mora no `cmd/seed` (quem escolhe o que atravessa é o
consumidor) e o `api.Seeder` a cumpre. Cada método do adaptador é um invólucro
sobre regra que já existe — nenhuma linha de regra mora nele, e é isso que
mantém de pé a promessa do cabeçalho do gerador: *os hashes, os vitais
computados e o leque normalizado vêm do mesmo código que o app roda.*

De passagem, três funções deixaram de pedir `*http.Request` e passaram a pedir
`context.Context` — `InsertCharacter`, `HealVitals` e `consumeItemForCharacter`
só liam o `r.Context()` dele. Exigir a requisição inteira obriga o chamador a
ter uma, e **foi exatamente o gerador que denunciou**, por ser o primeiro
chamador sem requisição.

### O guarda mede o programa INTEIRO terminando

`TestTheSeedGeneratorStillWritesTheCommittedFile` RODA o gerador num arquivo
temporário e compara com o `seed.sql` do repositório. É a única forma que pega
esta família: o defeito não era de compilação nem de asserção — era o programa
não terminar.

Ele custa ~1,8s e paga duas coisas de uma vez, porque comparar byte a byte
também prende que o gerador é **determinístico**: as datas são constantes e o
despejo normaliza carimbos de propósito, então um `time.Now()` que entre no
caminho reprova na primeira corrida.

E ele foi provado com a árvore de ONTEM em vez de sabotagem — `git archive` do
merge anterior, o guarda copiado por cima, `go test`. Reprovou com "o gerador
não rodou: exit status 1", que é o defeito real.

## O `*Server` deixou de ser porta (ALE-278, fatia 6)

Ele tinha **89 métodos exportados**, e todos existiam por um motivo só: cumprir
a UNIÃO das portas das onze cenas. Hoje tem **oito**, e eles são o que um
servidor deve expor — os dois roteadores, a sonda de saúde, os estáticos, o
agendador de backup e a espera do desligamento.

A medição que decidiu a divisão: das 76 assinaturas pedidas pelas portas,
**67 têm exatamente UMA cena pedindo**. Uma união que ninguém precisa como
união é objeto-deus com nome de servidor.

### A repartição, e o critério dela

**Um núcleo** (`sceneCore`, 7 métodos) com o que quase toda cena pede:
`WritePage` (11 das 11), `Queries` (7), `CurrentUserID` (6), `Catalogs` (4),
`BookAddress` (3), `Asset` e `CharacterList`. O corte é por CONTAGEM: duas
cenas pedindo a mesma coisa não sobem para o núcleo — o `MintAccountInvite` é
uma linha repetida no adaptador do hub e no da administração, e isso custa
menos que uma assinatura que nove cenas carregam sem usar.

**Quatro tipos de REGRA**, nomeados pelo que decidem e não por onde estavam:

| tipo | o que decide | carrega |
| --- | --- | --- |
| `accountRules` (13) | quem entra, quem se cadastra, quem troca a senha | cfg, db, queries |
| `campaignRules` (14) | de quem é esta mesa, quem pode entrar nela | db, queries |
| `sheetRules` (12) | o que muda numa ficha, e quem precisa saber | db, queries, catalogs, bus, sessions, sse |
| `tableRules` (32) | a iniciativa, o descanso, quem anda quanto, o que a mesa vê | quase tudo |

**Sete adaptadores de cena**, cada um "o núcleo mais o que aquela cena precisa":
`forgeHost` (núcleo + `db`), `hubHost` (+ cfg), `adminHost` (+ cfg + db),
`doorHost` (+ `accountRules`), `campaignsHost` (+ `campaignRules`), `sheetHost`
(+ `sheetRules`) e `tableHost` (+ `tableRules`). As outras quatro cenas —
grimório, mestre, leitor e personagens — recebem o núcleo puro.

> **O `tableRules` toca quase todo campo do `*Server`, e isso não é a divisão
> falhando.** A Mesa É a mesa ao vivo, e a mesa ao vivo é o que aqueles stores
> guardam. A diferença entre isto e receber o servidor não é o tamanho da lista,
> é o que ela NÃO tem: o livro, os trincos por personagem, a espera do
> desligamento e os métodos das outras dez cenas. Aqui está escrito de que a
> mesa depende; no `*Server` estava escrito "de tudo".

### A ordem, e por que ela não é negociável

Escrever o substituto, vê-lo verde, DEPOIS apagar. O núcleo entrou com uma
PONTE de sete linhas no `*Server` delegando para ele, e a ponte só saiu quando a
última cena tinha adaptador. Começar pelo apagar deixaria sete cenas sem
compilar — e um refactor que não compila no meio não tem como ser dividido em
commits revisáveis.

### Quatro armadilhas medidas, e três foram do compilador

1. **Recortar função de UMA LINHA procurando `}` na coluna zero come as
   vizinhas.** `func … { return s.queries }` não tem `}` sozinho, então o
   recorte engoliu quatro declarações abaixo. O `go build` denunciou; num
   arquivo de teste teria passado.
2. **Receptor colide com variável existente.** `c` para `campaignRules` esbarrou
   nos nove `c` que já eram `sqlcgen.Campaign`, e `rules` esbarrou numa variável
   LOCAL chamada `rules` dentro do próprio corpo que ia mudar de dono.
3. **`s.` dentro de uma string de formato.** A troca de `s.` por `h.` acertou o
   `%s.db` de um `fmt.Sprintf` e o nome do backup virou
   `t20-dev-%!h(string=…).db`. Quem pegou foi o `go vet`, não um teste — os
   quatro casos de backup usavam o nome devolvido e nenhum olhava para ele.
4. **A que o compilador NÃO pega, e é a que importa:** o adaptador COPIA o
   `*engine.Catalogs` quando é montado, e a cena da Mesa é montada uma vez no
   construtor. A bancada prima os catálogos DEPOIS do `NewServer`, então sete
   casos da Mesa passaram a estourar com nulo dentro do motor. Em produção o
   motor chega pelo construtor e nunca muda, mas a invariante é a mesma —
   **trocar o motor e reconstruir a cena são um gesto só**, e agora ele tem
   nome: `primeCatalogs`.

## A API JSON saiu, e o que ela levou junto (ALE-277)

Das **76 rotas** do `Router()` sobraram **sete**; dos **113 manipuladores**,
nove; dos **315 métodos** de `*Server`, 182. O `api` foi de 164 arquivos `.go` e
31 mil linhas para 150 e 25 mil. Nada disso é limpeza de estilo: **nenhum dos
apagados tinha chamador** — nem cena, nem spec de Playwright, nem `fetch` de
ilha — desde que a SPA foi apagada na ALE-272.

As sete que ficaram são as que alguma coisa de fato chama: `/health`, o CRUD de
campanha, a criação de sessão, a lista de personagens e o `PATCH` de condições.

### Por que isso não apareceu antes

Um handler sem chamador **não quebra compilação**. O Go reclama de import e de
variável local sem uso, e não de método sem uso — então 104 manipuladores
atravessaram onze fatias da ALE-278 sendo lidos, renomeados e movidos de arquivo
por gente que os tomava por código vivo. O mesmo vale para os 133 métodos de
`*Server` que só eles chamavam: cada um se defendia por um chamador que também
não tinha chamador.

**A busca que os acha é uma linha**, e ela é o que vale guardar deste parágrafo:

```
for m in $(grep -rho "func (s \*Server) handle[A-Za-z]*" api/ | sed 's/.*) //'); do
  n=$(grep -rn "s\.$m\b" --include=*.go . | grep -cv "func (s \*Server) $m")
  [ "$n" = 0 ] && echo "ORFAO $m"
done
```

Ela roda em segundos e teve de rodar TRÊS vezes: apagar os manipuladores órfãos
revelou 16 métodos órfãos, apagar esses 15 revelou mais três. Órfão é
transitivo, e uma passada só mede a primeira camada.

### O que o teste da rota apagada vira

Sessenta e dois casos ficaram vermelhos, e a regra que os separou não é "manter
ou apagar", é **de quem era a garantia**:

- **Regra** — o caso passa a chamar a REGRA (`s.createAccount`, `s.joinTable`,
  `s.castSpellForCharacter`), sem transporte nenhum. É a maioria, e é o teste
  ficando mais barato: some o roteador, some o JSON, some o código HTTP.
- **Contrato de rota** (o 409 do e-mail repetido, o 413 do corpo grande) — morre
  com a rota, e deixa LÁPIDE dizendo onde a garantia está agora. As lápides
  entram no `tombstones` do `convention/`, que é o que torna o ato explícito.
- **Regra que só aquele teste guardava** — muda de PACOTE, para o dono da regra:
  o bloco impossível de criatura foi para `creature/block_test.go`, a regra
  desconhecida para `campaign/rules_test.go`.

Uma armadilha específica: `newTestServer` + `sendRaw` dirigem o `Router()`, que
é a API. **Cena se testa pelo `WebRouter()`** — um caso repontado para um
endereço de cena pelo helper errado responde 404, e 404 se parece com "a rota
não existe" justamente quando ela existe.

### O que ficou sem gesto, e não foi apagado

Três capacidades perderam a porta e NÃO foram removidas, porque a coluna e a
consulta são o lugar onde elas voltam a morar quando alguma cena oferecer o
gesto. Elas estão anotadas para que a próxima pessoa não as descubra como
"funcionalidade quebrada":

- **Promover jogador a mestre.** O `PATCH /campaigns/{id}/members/{id}` era o
  único caminho, e nenhuma cena oferece o botão.
- **Rotacionar o convite da campanha.** Nada chama `SetInviteToken` fora da
  seed: a carta de convite RESOLVE um token, e ninguém cunha um.
- **O elenco presente na sala.** A Mesa lê `Presence().Roster`, e quem preenchia
  era o handshake da `/events`.

## `campaign`: a mesma regra recusando com DUAS frases

O que é um nome válido e o que é uma descrição válida saíram do `api` na
ALE-278, com a forma exata do `account` — e pelo mesmo defeito, que vale
registrar porque agora ele tem DOIS casos e um padrão.

A cena escrevia "O nome é obrigatório e cabe em 120 caracteres." e a rota JSON
respondia `err.Error()`, que era a frase inglesa herdada do NestJS. Duas frases
para uma regra é o que quebra quando alguém mexe no limite: uma das duas fica
para trás, e é sempre a que ninguém está olhando.

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
O `character_play_state.go` misturava a FORMA (dois structs sem dependência) com
o encanamento que a grava — os structs viajam dentro do `CharacterDTO`, então
foram; os handlers ficaram.

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

Quem preenche é o `piloto_render.go`, o ÚNICO lugar do projeto que monta uma
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
