# A infraestrutura

SQLite, sqlc, HTTP, compressão, tempo real, barramento e ambientes. Faz parte do [guia do engine](../CLAUDE.md).

## Ambientes

`LoadConfig` lê `.env.<APP_ENV>` deste diretório, mas **o processo vence o
arquivo**. O `.env.development` é versionado; o `.env.production` não, e o molde
dele é o `.env.production.example`. **Configuração nova entra em
`infra/config/config.go` e nos dois versionados** — default só no Go ninguém
descobre — **e, se puder derrubar produção em silêncio** (chave de assinatura,
origem liberada), **em `Config.Validate`**, que roda antes de escutar.

## O sqlc trunca SQL por causa de comentário acentuado

**Comentário em `infra/db/query.sql` e nas migrações é ASCII.** O sqlc mede a
query em bytes e o comentário em runas: cada acento corta, em silêncio, um
caractere do SQL gerado, e o truncado pode compilar.

## O sqlc não enxerga `ALTER TABLE ADD COLUMN` de outro arquivo

No sqlc **v1.31.1**, coluna de `ALTER TABLE … ADD COLUMN` num arquivo de
migração NOVO não entra no catálogo: a *query* que a cita falha com
`column "x" does not exist`, e o modelo gerado ganha o campo mesmo assim — **só
escrever a consulta prova.** No arquivo do `CREATE TABLE`, o `ALTER` funciona;
comentário e seção `Down` não mudam nada.

- **Coluna nova → tabela nova**, e "não tem" passa a ser dito pelo schema, não
  por convenção sobre valor vazio.
- **Trocar a chave primária também não passa**: o rebuild do SQLite usa
  `RENAME TO`, e o `RENAME` some do catálogo. Crie a tabela NOVA com outro
  nome, copie, derrube a velha.
- **`DROP COLUMN` só em minúsculas**: `hpMax` (a grafia do `CREATE`) falha com
  `column "hpMax" … does not exist`; `hpmax` passa, e o gerador acusa as queries
  que ainda citam a coluna.
- **Derrubar tabela é seguro para o boot**: o guarda de schema lê o `DROP`.

## O que o servidor RENDERIZA sai comprimido na hora

Cena renderizada só existe na requisição, o `net/http` não comprime sozinho, e
todo comando da ficha redesenha a cena INTEIRA — caro em dados móveis.
- **`gzip;q=0` é RECUSA**, que um `strings.Contains` leria como aceitação: daí
  o `httpio.AcceptsEncoding`.
- **Quem comprime é o `httpio.Gzip`, primeiro `Use` do `WebRouter`**, e não o
  `cmd`, para o teste do roteador ver o que o navegador recebe. Decide pelo
  `Content-Type` do handler e pula o que já tem `Content-Encoding`.
- **É o ÚNICO compressor, e só gzip**, também no fluxo ao vivo da Mesa: o
  brotli do SDK do Datastar seria um segundo lugar onde o SSE fica preso num
  buffer. Prende o `TestTheTableStreamCompresses`.

**A armadilha mora no SSE, e não deixa erro.** Todo comando do Datastar
responde `text/event-stream`, então pular SSE pularia tudo. **O `Flush` esvazia
o gzip ANTES de quem está embaixo** — senão o quadro fica no `gzip.Writer` e o
fluxo ao vivo para, sem log (`TestTheLiveStreamCrossesTheGzip`). **E o envelope
decide no `Flush` também**: o `datastar-go` chama `Flush()` para mandar os
cabeçalhos antes do primeiro remendo, e quem decide só no `Write` manda
cabeçalho sem `Content-Encoding` e corpo comprimido
(`TestAFlushBeforeTheWriteAlreadyDecidesTheEnvelope`). **Em todo middleware
daqui, quem esvazia compromete os cabeçalhos, e o teste esvazia na ordem da
biblioteca.** O envelope expõe `Unwrap() http.ResponseWriter`: sem ele,
`SetWriteDeadline` ou `Hijack` vindos de cima (via `http.ResponseController`)
respondem `ErrNotSupported`, e quem chamou conclui que não há fluxo.

## Todo 5xx deixa uma linha no log

**O `httpio.LogServerErrors` é o segundo `Use` do `WebRouter`**, dentro do gzip
para ler o corpo legível. Ele registra método, rota, status, milissegundos e o
começo do corpo de toda resposta 5xx, inclusive a do `Recoverer`. Ele repassa
`Flush` e `Unwrap` pela mesma razão do envelope do gzip.

**Quem responde 500 com frase genérica ANEXA a causa**: `httpio.WriteServerError`
na API JSON, `httpio.RecordCause(w, err)` antes do `ui.NoticeInternal` nas
cenas. Sem o anexo, a linha diz que falhou e não por quê. A causa vai para o
log e nunca para a resposta. Numa resposta 4xx ela é ignorada, então quem chama
um mapeador de status pode anexar sempre. Prende o
`TestACampaignThatFailsToOpenLogsTheCause`.

O tempo na linha é o que separa uma consulta quebrada de uma espera pela trava
de escrita: um 500 em 5000 ms é o `busy_timeout` inteiro (ver a seção da trava,
abaixo).

## O `synchronous` do SQLite é `NORMAL`

**O banco abre com `synchronous=NORMAL`**: o `FULL` padrão faz `fsync` a cada
commit, mais de cem milissegundos POR TOQUE num prato girante. Na queda de
energia perdem-se commits não sincronizados, mas o WAL garante que o banco NÃO
corrompe — o que separa `NORMAL` de `OFF`, que é só do banco de teste.

## O app fala HTTP, sem nada na frente

HTTP puro, sem TLS, contêiner nem proxy — decisão do dono: nginx com HTTPS e
compose custavam mais do que compravam. **`COOKIE_SECURE` fica DESLIGADO**: em
HTTP o navegador descarta cookie `Secure` e o login não conclui, sem erro. O
preço é o app não INSTALAR no telefone — ver o README.

## O boot confere o SCHEMA, não o `goose_db_version`

A migração pode CONSTAR aplicada sem a tabela existir, e o servidor subiria
gravando no vazio. **O `db.Open` roda `assertSchema` DEPOIS de migrar e recusa
subir**, nomeando as tabelas que faltam. As esperadas saem das migrações
embutidas, nunca da mão, lendo `CREATE TABLE` e `DROP TABLE` da seção `Up` na
ordem dos arquivos (sem o `DROP`, uma derrubada legítima viraria "falta a
tabela"). Banco alterado por fora (`goose down` parcial, backup antigo
restaurado) **não sobe**, de propósito.

## Transação pega a trava no BEGIN (`_txlock=immediate`)

**A conexão abre com `_txlock=immediate`**, e não `DEFERRED`. Unicidade
decidida no CÓDIGO (pergunta, depois escreve) perde para dois pedidos
simultâneos. **Refazer a checagem DENTRO da transação** torna o resultado
correto; o `immediate` o torna honesto: sem ele o perdedor leva 500 (o SQLite
recusa escrita sobre snapshot mudado), com ele espera, relê e recebe a recusa
certa. Custo: os escritores serializam, a leitura segue livre (WAL), e o
`busy_timeout(5000)` cobre a espera.

## Timeouts, encerramento e backup

- **Nenhum `WriteTimeout`, de propósito** (há `ReadHeaderTimeout` de 5s e
  `IdleTimeout` de 120s): ele mataria o fluxo SSE e download grande em rede
  ruim. Nenhum teste prende a ausência; confira o comentário em
  `cmd/api/main.go` antes de "consertar".
- **Sinal encerra com ordem** (`signal.NotifyContext` + `Shutdown`, 10s): defer
  não roda em morte por sinal, e um `VACUUM INTO` morreria pela metade.
- **Os fluxos ao vivo ABORTAM ao encerrar**, o da mesa e o da ficha solta (`CloseLiveStreams`, `panic(http.ErrAbortHandler)`):
  sem isso, uma conexão aberta prende o `Shutdown` até o prazo. E não pode ser `return`:
  fechar limpo é resposta completa para o Datastar, que não reconecta, e a mesa fica parada
  sem erro nenhum. Medido no navegador.
- **Corpo de requisição tem teto de 1 MB** no `httpio.DecodeJSON`, com 413
  próprio, e não "JSON inválido".
- O backup automático (`BACKUP_EVERY`, `BACKUP_KEEP`) usa o `VACUUM INTO` do
  manual e poda os mais antigos; zero desliga. A poda só alcança o que a
  listagem reconhece como backup.

## O tempo real é SSE: a mutação sobe por requisição, o estado desce pelo fluxo

**Bidirecional não é requisito, é hábito**: o que sobe é MUTAÇÃO, uma
requisição; o que desce é um `text/event-stream` aberto. Vale para o próximo
canal. A Mesa tem fluxo PRÓPRIO (`/campanhas/{campanha}/sessoes/{sessao}/fluxo`, em
`serve/web/table/stream.go`) que assina o `events.Bus`; os comandos são rotas da
CENA. A ficha solta tem o dela (`/personagens/{id}/ao-vivo`, em `serve/web/sheetui/live.go`), que só
avisa a versão nova da ficha. Ele passa pelo `requirePage`, que lê a sessão do COOKIE — sem token na
query string. A queda do cliente vem do `r.Context()` cancelado, não de batida.

> **O `SSEHub` (`domain/live`) só EMITE**: `SSEHub.Add` e o laço de leitura
> (`live.StreamFrames`, com a batida `: ping` para atravessar intermediário que
> fecha conexão ociosa) só rodam em teste.

O `Emit` do hub **nunca bloqueia**: fila cheia descarta o quadro daquele
leitor, que reconecta e busca o estado por HTTP. **Nenhum estado vive só no
fio. Campo que outro arquivo precisa preencher para o recurso existir é recurso
que nasce desligado**: o `go test ./...` não acusa, só o e2e de dois clientes.

## O barramento de eventos: o que acontece na mesa é TIPADO

O `events.Bus` entrega, dentro do processo, O QUÊ mudou numa mesa, não só QUE
mudou. **Não é event sourcing: o banco é o estado.** As regras são conta
(`sheet → números`, com o oráculo como rede); o `engine/` fica fora.

- **Nomear o evento é obrigação de COMPILAÇÃO**: o `apply` dos dois stores
  recebe o evento por parâmetro.
- **Publicar acontece FORA da trava**, senão quem acorda para ler espera o
  escritor. Com `defer Unlock`, o corpo vira `…Locked` e um invólucro publica.
- **A fila cheia DESCARTA e CONTA** (`queueSize`, `Subscription.Dropped`):
  evento tipado não colapsa como um "mudou". O evento é a notícia; a verdade
  está no store.
- **`events/` não importa NADA do projeto** (`TestVocabularyImportsNothing`):
  está nos permitidos do `domain/live` e do `domain/board`, e importar a ficha
  ali daria a ficha aos dois com o guarda de lá verde.

A PORTA não serve aqui: ela só casa com tipos do consumidor, um vocabulário por
contexto. O que legitima o compartilhamento é ser *shared kernel* — pequeno,
sem dependências, de todos porque não é de ninguém.

