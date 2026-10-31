# Motor Go

Adapta o [CLAUDE.md da raiz](../CLAUDE.md) a este pacote. As regras da raiz
valem; o que está aqui estende ou sobrepõe.

`engine-go` é o app inteiro: a API HTTP na :3001, o motor de regras, e as CENAS
em `.templ` servidas com Datastar — mais a folha e as ilhas de JS delas, em
`serve/web/assets/src`, e o kit de apresentação em `serve/web/ui`. Um processo serve tudo,
direto na máquina, em HTTP e sem nada na frente — ver "O app fala HTTP",
abaixo.

## O mapa das pastas

Cinco grupos, e a pergunta que cada um responde:

```
engine-go/
├── cmd/          os binários: o servidor (`api`), a seed, o gerador do oráculo,
│                 os auditores do catálogo (`audit/`) e as ferramentas que
│                 gravam nele — `internal/` é o que elas dividem
├── domain/       O QUE O JOGO É — nada aqui sabe de HTTP
│   ├── engine/   as regras do livro, PURAS (só a stdlib)
│   ├── ecs/      o núcleo de entidade-componente-sistema do motor (só a stdlib)
│   ├── catalog/  o livro transcrito (`transcript/`, em Go) e quem o serve
│   ├── book/     o catálogo tipado
│   ├── sheet/  board/  live/   o domínio COM estado
│   └── campaign/ account/ creature/ search/ markdown/
├── app/          O QUE UM GESTO FAZ, do pedido à gravação
│   ├── session/  o ciclo da sessão, a trava de acesso e o STORE da fila
│   ├── boards/   o store dos tabuleiros abertos, com as abas e os lugares
│   ├── initiative/ quem entra na fila, e com que números
│   ├── combat/   o ataque e a manobra: quem pode, o que a regra decide, o
│   │             provisório na fila
│   ├── character/ o herói: nascer (`Births`) e jogar (`Plays`)
│   ├── campaign/ a mesa: quem a vê, quem a abre, quem senta nela
│   ├── accounts/ a conta: entrar, cadastrar-se com convite, trocar a senha
│   └── rest/     o que expira e o que recupera quando a cena ou o dia acaba
├── serve/        O QUE RESPONDE HTTP
│   ├── api/      a RAIZ DE COMPOSIÇÃO: monta o roteador e cumpre as portas
│   └── web/      as cenas, uma por pacote, e o que elas dividem (`ui`, `ds`,
│       │         `routes`)
│       └── assets/  o FRONT que não é `.templ`: a folha, as ilhas de JS,
│                    as fontes e o favicon — e o `go:embed` deles
├── infra/        O QUE NÃO É DOMÍNIO
│   ├── db/       migrações e as consultas do sqlc
│   ├── config/   o que o ambiente diz, lido no boot
│   ├── bookpdf/  a leitura do PDF do livro por coordenada, para os auditores
│   ├── httpio/   a borda HTTP: responder, ler corpo, comprimir
│   ├── wire/     a fronteira SEM transporte — é o que o domain alcança
│   ├── events/   o barramento tipado
│   └── secret/   o sorteio do token que alguém recebe por link
├── convention/   os guardas que não são de pacote nenhum
├── parity/       os oráculos de ficha
└── scripts/      os scripts de build: a folha, as ilhas de JS, os ícones e o
                  Datastar vendorizado
```

**A seta só aponta para BAIXO**: `serve` → `app` → `domain` + `infra`, e o
`infra` não conhece ninguém. Quem garante são os `boundary_test.go` de cada
pacote — e, entre os GRUPOS, o `TestNoLayerImportsUpwards`, que varre a árvore em
vez de uma lista, para o pacote que nascer amanhã já nascer medido. As listas de
permitidos são argumentadas linha a linha: acrescentar uma entrada é decisão, não
conveniência.

### O `app/`, e o que ele NÃO é

Ele não é "a pasta para onde mudo o que estava no `api`". O que entra aqui é o
gesto INTEIRO — quem pode, o que decide, o que grava —, e o sinal de que uma
coisa pertence a ele é ter as três. Um repasse de uma linha não vira caso de uso
por mudar de pasta.

**O `domain/board` tem ZERO toques de persistência**, e não por zelo de autor: o
`boundary_test.go` dele proíbe o import do `sqlcgen`. O store que carrega o
estado, chama a regra pura e grava mora em `app/boards`.

**O que sobra em `domain/` é a regra, e ela é PÚBLICA.** As mutações que os
stores chamam — `AddToken`, `AdvanceTurn`, `PatchEntryVitals` e as irmãs — são
a API dos pacotes puros, e é assim que se reconhece um: recebe estado, devolve
estado, não trava nada e não grava nada.

**As recusas daqui são TIPADAS** (`ErrNotFound`, `ErrForbidden`, `ErrRefused`) e
nunca um número de HTTP. Um caso de uso que devolvesse 403 não poderia ser
chamado de outro transporte — que é a única coisa que esta camada compra.

**E a TRANSAÇÃO é daqui, sempre.** Ela é o contorno de um gesto — ou as
campanhas mudam de dono e a conta some juntas, ou nada acontece —, e quem
desenha esse contorno está decidindo o que o gesto É. Quem mantém o `serve/`
sem `BeginTx` é o `TestNoPresentationLayerOpensATransaction`: linha de base
vazia, falha no primeiro que voltar.

> **O `character.Plays` é a exceção declarada, e a razão é a TELA.** As recusas
> dele não embrulham `app.ErrRefused`: o embrulho põe "recusado pela regra" no
> fim do `Error()`, e a cena da ficha mostra esse texto CRU ao jogador, porque o
> Datastar não desenha corpo de resposta 4xx e toda recusa dela volta como frase
> na cena. Os gestos da ficha têm UM transporte e nenhum chamador lê o tipo.
> Quando o segundo transporte chegar, o embrulho desce — uma linha por recusa. O
> que NÃO pode entrar é número de HTTP dentro da regra.

**Onde procurar:**

| a pergunta | a pasta |
|---|---|
| quanto de PV este herói tem | `domain/engine` |
| como a ficha é montada do banco | `domain/sheet` |
| o que a tela desenha | `serve/web/<cena>` |
| a folha, uma ilha de JS, uma fonte | `serve/web/assets` |
| uma peça do sistema Tempestade | `serve/web/ds` (e a folha viva em `/sistema`) |
| quem cumpre o que a cena pede | `serve/api` |
| onde o dado é gravado | `infra/db` |
| por que a suíte reprovou uma convenção | `convention` |

> Os TESTES moram ao lado do código, e isso é do Go: teste de caixa branca
> precisa do mesmo diretório para alcançar o que não é exportado.

**Cada cena é um pacote em `web/`**, e o `api` guarda a composição do roteador, as
portas que cada cena pede e as regras que não são de tela nenhuma. Ver "Como uma
CENA é construída".

**No `serve/api`, o nome do arquivo diz o ADAPTADOR que o possui**, e o assunto
vem depois do prefixo: `table_*` é do `tableRules`, `sheet_*` do `sheetRules`,
`campaign_*` e `account_*` dos outros dois. **Um arquivo, um dono.**

Quem cobra é o `TestEveryAdapterFileCarriesItsPrefix`: ele lê o RECEPTOR dos
métodos e falha com o nome do arquivo e o do dono. Duas coisas que ele NÃO
conta, e as duas são de propósito — o `*Server`, que é fiação e está em metade
do diretório, e os hosts de cena (`tableHost`, `hubHost` e os irmãos), que já têm a
convenção deles em `*_deps.go` e convivem com um adaptador no mesmo arquivo.

**As cenas atendem na RAIZ**, sem prefixo. Duas consequências que o `git grep`
não mostra:

- **O `alvoOriginal` lê `RequestURI` e não `URL.Path`**, porque o segundo
  descarta a QUERY.
- **Endereço interno não soma prefixo.** Um `"/" + rota` vira `//livro`, que não
  é uma barra a mais: `//algo` é URL relativa a PROTOCOLO, e o navegador a lê
  como o HOST `algo`. Quem denuncia é um teste de cena; o compilador não tem
  como.

## Ambientes

`LoadConfig` lê `.env.<APP_ENV>` deste diretório antes do env do processo —
**o processo vence o arquivo**, sempre. O `.env.development` é versionado
porque nada nele é segredo; o `.env.production` é do dono da mesa e não entra
no git, e o molde dele é o `.env.production.example`, que entra.

Configuração nova entra em `infra/config/config.go` **e** nos dois arquivos
versionados (`.env.development` e `.env.production.example`) — um default que só
existe no Go é um default que ninguém descobre. Se a variável puder derrubar
produção em silêncio (chave de assinatura, origem liberada), ela também entra em
`Config.Validate`, que roda antes de o servidor escutar.

## Regenerar oráculo é ato deliberado

Os JSONs em `parity/` são a rede de regressão da ficha inteira e o teste mais
valioso do repositório: cada um fixa, de um personagem, o PV, cada carta de
ataque com o breakdown e cada perícia com a composição, e acusa qualquer número
que mude sem ter sido pedido.

```
cd engine-go && go run ./cmd/genoracle
```

> **O diff de um oráculo é revisado contra o LIVRO, nunca aceito porque "o teste
> ficou verde".**

**Não há segunda implementação para conferir contra**: o oráculo é o Go
descrevendo o Go, e **um bug no motor vira a nova verdade em silêncio**. A
mitigação não é técnica, é de processo, e é a linha acima. Paridade prova
concordância; nunca prova correção.

## Regras vêm do livro, com página

Todo teste de regra cita a página do [Tormenta 20](/t20-book.pdf) e, quando
existe, o **exemplo trabalhado** do próprio livro — a Samira da p173, por
exemplo. Um exemplo trabalhado é melhor que uma asserção inventada: ele separa
leituras possíveis da mesma frase.

**Confira a citação antes de escrever.** O offset é `página do PDF = página do
livro + 6`. Citação "verificada" de cabeça já saiu errada mais de uma vez — ver
[APRENDIZADOS](../APRENDIZADOS.md).

## O sqlc trunca SQL por causa de comentário acentuado

Comentário em `infra/db/query.sql` é **ASCII**. O sqlc mede a query em bytes e conta o
comentário em runas, então cada letra acentuada acima de uma query corta um
caractere do SQL gerado — **em silêncio**, e o SQL truncado pode continuar
compilando. Vale também para o comentário das migrações, que o sqlc lê.

## O sqlc não enxerga `ALTER TABLE ADD COLUMN` de outro arquivo

No sqlc **v1.31.1**, uma coluna adicionada por `ALTER TABLE … ADD COLUMN` num
arquivo de migração NOVO não entra no catálogo — toda query que a cita falha com
`column "x" does not exist`, e o erro aponta a *query*, não a migração. O modelo
gerado ganha o campo mesmo assim: **ver o modelo não prova nada; só escrever a
consulta prova.** Dentro do MESMO arquivo (`CREATE TABLE` + `ALTER` juntos)
funciona. Nem a forma do comentário nem a seção `Down` mudam isso.

- **Coluna nova → tabela nova.** De quebra, "não tem" passa a ser dito pelo
  schema (a linha existe ou não existe) em vez de por convenção sobre um valor
  vazio.
- **Trocar a chave primária também não passa pelo gerador.** No SQLite é um
  rebuild com `RENAME TO`, e o `RENAME` some do catálogo do mesmo jeito. O
  caminho que funciona é criar a tabela NOVA com outro nome, copiar, e derrubar a
  velha.
- **`DROP COLUMN` funciona em minúsculas, e só assim.** Com a grafia que o
  `CREATE` usou (`hpMax`) ele falha com `column "hpMax" … does not exist`; com
  `hpmax` passa, e o gerador então reclama de toda query que ainda cita a coluna.

Derrubar tabela é seguro para o boot: o guarda de schema lê o `DROP` — ver "O
boot confere o SCHEMA".

## O que o servidor RENDERIZA sai comprimido na hora

Cena renderizada não existe antes da requisição, então a escolha real para ela é
gzip na hora ou nada — o `net/http` não comprime nada sozinho. E todo comando da
ficha responde redesenhando a cena INTEIRA, então cru é caro no telefone com
dados móveis.

**`gzip;q=0` é uma RECUSA**, e um `strings.Contains` a leria como aceitação. É
por isso que o `httpio.AcceptsEncoding` existe em vez de uma busca por
substring.

Quem faz é o `httpio.Gzip`, o primeiro `Use` do `WebRouter` — e não o `cmd`, para
o teste que bate no roteador de verdade ver o que o navegador recebe. Ele decide
pelo `Content-Type` que o handler escreveu, e pula o que já chega com
`Content-Encoding`: recomprimir produz bytes maiores gastando CPU.

**É o ÚNICO compressor, e só gzip.** O fluxo ao vivo da Mesa passa por ele
também: o SDK do Datastar sabe comprimir o fluxo sozinho, e em brotli, mas um
segundo compressor seria um segundo lugar onde o SSE pode ficar preso num
buffer. Quem prende é o `TestTheTableStreamCompresses`.

**A armadilha mora no SSE, e ela não deixa erro para trás.** A resposta de todo
comando do Datastar é `text/event-stream` — ela usa o envelope de SSE para
mandar UM remendo e fechar. Então "não comprimir SSE" pularia justamente o que
se quer comprimir; e comprimir SEM repassar o `Flush` prende o quadro no buffer
interno do `gzip.Writer`, e o fluxo AO VIVO da Mesa para de atualizar. Nada
falha, nada loga, e o sintoma — "o tempo real quebrou" — não aponta para um
middleware de compressão. Por isso o `Flush` esvazia o gzip ANTES de quem está
embaixo, e `TestTheLiveStreamCrossesTheGzip` mede um quadro chegando com a
conexão ainda aberta.

**E o envelope tem de decidir no `Flush` também.** O `datastar-go` escreve o
`Content-Type`, chama `Flush()` para MANDAR OS CABEÇALHOS, e só então escreve o
primeiro remendo. Um envelope que decide comprimir apenas no `Write` chega
tarde — os cabeçalhos já foram sem `Content-Encoding`, o corpo sai comprimido, e
o navegador lê gzip como texto. Nenhuma requisição falha e nenhum status muda:
os remendos só param de ser aplicados. O
`TestAFlushBeforeTheWriteAlreadyDecidesTheEnvelope` repete a ordem do Datastar
de propósito. A regra geral, para qualquer middleware que se escreva aqui:
*quem esvazia compromete os cabeçalhos* — e o teste tem de esvaziar na ordem da
biblioteca, não na que for conveniente.

O envelope também expõe `Unwrap() http.ResponseWriter`, que é o contrato do
`http.ResponseController` desde o Go 1.20 — sem ele, um `SetWriteDeadline` ou um
`Hijack` de qualquer camada acima responde `ErrNotSupported`, e quem chamou
conclui que o ambiente não suporta fluxo.

## O `synchronous` do SQLite é `NORMAL`

O banco abre com `synchronous=NORMAL`, e não com o `FULL` padrão: o `FULL` faz
`fsync` a cada commit, e num prato girante isso custa mais de cem milissegundos
POR TOQUE na ficha. Numa queda de energia perdem-se os commits ainda não
sincronizados, e o banco NÃO corrompe — é a garantia do WAL, e é o que separa
`NORMAL` de `OFF`. `OFF` é só do banco de teste, onde não há o que proteger.

## O app fala HTTP, sem nada na frente

O binário serve HTTP puro e não sabe servir TLS, e nada fica na frente dele: sem
contêiner e sem proxy. Houve um nginx com HTTPS e um compose, e eles saíram por
decisão do dono — a infra custava mais do que comprava.

`COOKIE_SECURE` fica DESLIGADO: em HTTP, o navegador descarta um cookie
`Secure` e o login não conclui, sem erro visível. O preço de não ter HTTPS é o
app não INSTALAR no telefone — ver o README.

## O boot confere o SCHEMA, não o `goose_db_version`

A migração pode CONSTAR aplicada sem a tabela existir, e aí o goose diz "no
migrations to run" e o servidor sobe gravando no vazio.

Por isso o `db.Open` roda `assertSchema` DEPOIS de migrar e **recusa subir**
nomeando as tabelas que faltam. A lista de esperadas é lida das próprias
migrações embutidas, nunca escrita à mão: lista à mão envelhece em silêncio.

Ela lê os DOIS verbos da seção `Up`, `CREATE TABLE` e `DROP TABLE`, na ordem dos
arquivos. Só com o primeiro, o schema nunca podia PERDER uma tabela, e uma
derrubada legítima viraria "falta a tabela" num banco certo.

Consequência a saber: um banco alterado por fora (um `goose down` parcial, um
backup anterior restaurado) **não sobe**. É deliberado — gravar no vazio em
silêncio é pior.

## Transação pega a trava no BEGIN (`_txlock=immediate`)

A conexão abre com `_txlock=immediate`, e não com o `DEFERRED` padrão.

O motivo: travas de unicidade decididas no CÓDIGO (pergunta ao banco, depois
escreve) não sobrevivem a dois pedidos simultâneos, porque as duas perguntas
acontecem antes de qualquer escrita. Quem torna o resultado **correto** é
refazer a checagem DENTRO da transação; o `immediate` é o que o torna
**honesto** — sem ele um dos perdedores recebe 500 (o SQLite recusa a escrita
sobre snapshot mudado, o que está certo, mas chega ao jogador como erro do
servidor), e com ele o perdedor espera, relê e recebe a recusa que descreve o
que houve.

Custo: os escritores serializam entre si. As transações do app são de escrita e
a leitura fora de transação continua livre (WAL), então numa mesa doméstica isso
é de graça; o `busy_timeout(5000)` cobre a espera.

## Timeouts, encerramento e backup

O servidor tem `ReadHeaderTimeout` (5s) e `IdleTimeout` (120s), e **não tem
`WriteTimeout` de propósito**: ele mataria o fluxo SSE, que é conexão longa por
natureza, e um download grande numa rede ruim. É o timeout que parece
obrigatório e é justamente o errado aqui. Nenhum teste prende a ausência dele —
quem a defende é o comentário em `cmd/api/main.go`, e é lá que se confere antes
de "consertar".

Um sinal encerra com ordem (`signal.NotifyContext` + `Shutdown`, janela de 10s):
defer não roda quando o processo morre por sinal, e um `VACUUM INTO` ou uma
gravação no meio morreria pela metade.

Corpo de requisição tem teto de 1 MB no `httpio.DecodeJSON`, com **413 próprio**: dizer
"JSON inválido" para um JSON válido manda procurar defeito de sintaxe onde o
problema é tamanho.

O backup automático (`BACKUP_EVERY`, `BACKUP_KEEP`) usa o mesmo `VACUUM INTO` do
manual e poda os mais antigos. Zero em qualquer um dos dois desliga. A poda só
alcança o que a listagem reconhece como backup — arquivo estranho na pasta não é
candidato.

## O tempo real é SSE: a mutação sobe por requisição, o estado desce pelo fluxo

**Bidirecional não é requisito, é hábito.** O que sobe é sempre MUTAÇÃO, e
mutação é uma requisição; o que desce é um `text/event-stream` que fica aberto.
O argumento vale para o próximo canal.

A Mesa tem fluxo PRÓPRIO (`/campanhas/{campanha}/sessoes/{sessao}/fluxo`, em
`serve/web/table/stream.go`): ele assina o `events.Bus`, e os comandos dela são
rotas da CENA. Ele entra pelo mesmo `requirePage` das cenas, que lê a sessão do
COOKIE, então o fluxo não precisa de token na query string. Queda do
cliente se detecta pelo `r.Context()`, que é cancelado quando ele vai embora;
nenhuma batida é para isso.

> **O `SSEHub` (`domain/live`) só EMITE.** Em produção ninguém chama
> `SSEHub.Add` — só testes —, e emitir para zero ouvinte não estoura nada. O
> laço de leitura dele (`live.StreamFrames`, com a batida `: ping` que é
> comentário SSE para atravessar intermediário que fecha conexão ociosa) também
> só é exercitado por teste.

O `Emit` do hub **nunca bloqueia**: fila cheia descarta o quadro daquele leitor.
Quem perdeu reconecta e busca o estado por HTTP, que é o mesmo caminho da
primeira carga — nenhum estado vive só no fio.

**Campo que precisa ser preenchido por outro arquivo para o recurso existir é
recurso que nasce desligado** — e o `go test ./...` não acusa: quem acusa é o
e2e de dois clientes.

## O barramento de eventos: o que acontece na mesa é TIPADO

O `events.Bus` entrega, dentro do processo, o que aconteceu numa mesa: O QUÊ
mudou, e não só QUE algo mudou.

**Não é event sourcing, e a diferença é a decisão inteira.** O banco continua
sendo o estado. As regras do livro são conta e não fluxo — empilhamento de
modificador, PV/PM, círculo de magia são função pura de `sheet → números`, com o
oráculo como rede —, então o `engine/` não participa disto e reconstruir ficha a
partir de eventos compraria versionamento de evento e snapshot sem nada em troca.

Três coisas que este desenho pede, e uma que ele proíbe:

- **Nomear o evento é obrigação de COMPILAÇÃO.** O `apply` dos dois stores recebe
  o evento por parâmetro, então não dá para mutar sem dizer o que aconteceu — o
  que um comentário prometendo que ninguém escapa não consegue fazer.
- **Publicar acontece FORA da trava.** O barramento é folha e poderia ser chamado
  de dentro, mas quem acorda agora sabe o que houve e pode ler o estado na hora;
  publicar sob a trava faria esse leitor esperar pelo escritor no instante em que
  foi acordado para ler. Onde o método usa `defer Unlock`, o corpo vira
  `…Locked` e um invólucro fino publica.
- **A fila cheia DESCARTA e CONTA.** Evento tipado não colapsa como um "mudou"
  colapsa, então a fila tem dezesseis lugares (`queueSize`). O contrato de quem escuta: o evento
  é a notícia, a verdade está no store. E o descarte não é invisível
  (`Subscription.Dropped`), pela mesma razão que o medidor de contraste devolve o
  denominador.
- **`events/` não importa NADA do projeto.** Ele está na lista de permitidos do
  `domain/live` e do `domain/board`, cujos guardas de fronteira avisam que
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

## A regra de EMPILHAMENTO mora onde o `targetKey` mora

O `TestNoEntityStacksWithItself` (`domain/engine/modifier_stacking_rules_test.go`)
prende a invariante da p226 — uma entidade não declara dois modificadores
não-condicionais no mesmo alvo e tipo —, e o balde sai do `targetKey` de verdade,
lido no `Modifier` de verdade (o que faz o caso atravessar o `UnmarshalJSON`
junto). **Guarda que precisa de uma regra privada do motor mora no motor**: um
`switch` espelhado noutro pacote diverge em silêncio.

## A MANOBRA é o primeiro teste OPOSTO do motor

Todo o resto do combate mede um total contra um número parado — a Defesa, a CD
de uma magia. A manobra (p234) manda os DOIS lados rolarem, e por isso o
`ResolveManeuver` recebe dois `d20` por parâmetro em vez de um.

Duas regras da página, e nenhuma é opcional:

1. **oposto**, com a MARGEM viajando porque ela é regra — cinco ou mais dão
   efeito extra ao derrubar e ao desarmar, e o empurrar ganha 1,5m por cada 5;
2. **o defensor sempre rola LUTA**, mesmo empunhando arma de disparo. A
   proibição de ataque à distância é de quem ATACA, e a página diz as duas
   coisas na mesma frase.

O motor não decide quem venceu: quem decide o desfecho é o mestre — ver a
seção abaixo.

**A manobra divide o provisório com o golpe**, e não tem um irmão: ela É um
ataque corpo a corpo, a divisa de propor e confirmar é a mesma, e dois
provisórios simultâneos seriam duas verdades sobre a mesma cena. O
`proposesAttack` e o `proposesManeuver` delegam ao mesmo `proposeStrike` pela
mesma razão — as conferências antes da rolagem são as mesmas.

E o BÔNUS conhece o LADO: o `Desejo de Liberdade` ajuda quem está sendo
agarrado, não quem agarra. O escopo está na chave do `targetKey`, e o caso que
prende os dois lados é o que impede a volta do defeito.

## A MANOBRA ANUNCIA a condição; ela não a aplica

Duas das cinco manobras deixam condição — *"você deixa o alvo CAÍDO"* e *"uma
criatura AGARRADA fica desprevenida e imóvel"* (p234). As outras três não são
esquecimento: o livro lhes dá efeito de ITEM ou de MOVIMENTO, e escrever uma
condição ali seria inventar regra.

**O motor devolve o embate e nunca o vencedor:** os dois totais, a `Margin` e o
`ConditionOnAWin`, a condição que o livro RESERVA à vitória. Quem decide se a
manobra passou é o mestre — ver "O sistema INFORMA; o mestre DECIDE" no
[CLAUDE.md da raiz](../CLAUDE.md).

**A condição viaja mesmo quando quem tentou rolou menos.** Ela é o que o LIVRO
prevê, não o que aconteceu; um campo que sumisse na diferença negativa obrigaria
a faixa a adivinhar o vencedor — o veredicto voltando pela porta dos fundos.
Quem marca condição é o `toggleCondition` da fila, gesto do mestre.

Dois guardas prendem a AUSÊNCIA do veredicto, um por metade:

- o `TestTheManeuverNeverDeclaresAWinner` varre os campos do `ManeuverOutcome`
  por reflexão — uma asserção sobre um campo de vencedor deixaria de compilar
  quando ele saísse, e caso que não compila é caso que alguém apaga;
- o `TestNoManeuverTextDeclaresAWinner` varre o TEXTO de todas as faixas
  possíveis procurando palavra de desfecho — a metade que o guarda de forma não
  alcança, porque uma palavra reintroduzida compila e chega à mesa. O crachá diz
  a manobra no INFINITIVO.

> **O id não vai para a tela.** Quem traduz o id da condição é o
> `book.ConditionName`, e a regra é a do `GLOSSARY.md`: id em inglês no código,
> texto em português na tela.

## O guarda de ÍCONE varre as CENAS, e o `switch` do gerado não tem `default`

Um nome que ninguém gerou rende um `<svg>` VAZIO: sem erro de compilação, sem
aviso, um buraco do tamanho do ícone. O `TestEveryRequestedIconExistsInTheGeneratedFile`
varre `serve/` inteiro, casa as duas grafias (`@icone("X")` e `@ui.Icon("X")`) e
tem DENOMINADOR. **Ícone novo entra na lista do `scripts/gen-icons-templ.mjs`, e
o gerador roda — nunca à mão no `icons.templ` gerado.**

## Todo ALVO de modificador é declarado, e o RÓTULO não conta como leitor

O `targetKey` nomeia os alvos que um modificador pode mirar, e cada um está em
uma de duas listas do `domain/engine/modifier_target_rules_test.go`: os que o
motor APLICA, com o arquivo que os lê, e os declarados INERTES
(`alvosDeclaradosInertes`), com o motivo. As duas juntas têm de cobrir
exatamente o `switch` — alvo novo sem entrada em nenhuma reprova pelo nome, e
alvo que sai do `switch` sem sair da lista também. A lista de inertes **só pode
encolher**.

**O rótulo do `serve/web/sheetui/effects_labels.go` NÃO conta como leitor**, e
essa é a linha que faz o guarda valer: o defeito que ele caça é o modificador que
aparece na aba Efeitos e não entra em conta nenhuma — o rótulo é exatamente a
metade que já funciona, e tratá-lo como prova faria o guarda aprovar o que veio
medir.

Três famílias de inércia, e elas pedem consertos diferentes:

- **o mecanismo ESQUECEU** — havia uma conta e ela não olhava o modificador. O
  conserto é uma linha, e o teste nasce vermelho;
- **o mecanismo NÃO EXISTE** — não há onde somar, e o conserto é construir o
  gesto, ou declarar por escrito que ele não será construído (é o `tempMp` hoje:
  o app não modela PM temporário, e a entrada diz por quê);
- **o alvo não devia existir** — só aparece lendo o livro: o alvo era outro, já
  existente, com condição, e o conserto é reescrever a entrada do catálogo.

**Casar nome não é casar conceito.** Achar o nome de um alvo num arquivo do
motor não prova que ele é lido: ali ele pode ser outra coisa com a mesma grafia.

## O motor é ECS, e a ENTIDADE muda de fase para fase

**REGRA NOVA NASCE EM ECS. É o default, e o critério é APRENDER** (decisão do
dono). Diante de uma regra do livro a modelar, a pergunta é *"quem é a
ENTIDADE, que COMPONENTES ela carrega, e que SISTEMA os lê?"* — e não "que campo
esta struct ganha".

**Quem escolhe ECS paga o guarda de denominador junto**: um sistema que não roda
entrega ZERO com cara de conta (ver a seção seguinte). O guarda é o irmão do
`TestEveryDerivationSystemWritesItsComponent`, com a lista à mão.

Quando a regra não tem entidade — um formatador, uma consulta, uma tabela
fechada que só se lê —, ela não é ECS. ECS para o que COMPÕE; função para o que
calcula.

O pipeline da ficha é coleta → resolução → derivação → decomposição, e o ataque
é um mundo à parte. A entidade muda porque a pergunta muda:

| fase | entidade | pergunta |
|---|---|---|
| **coleta** (`collect_ecs.go`) | a FONTE | quem concede alguma coisa a este personagem? |
| **resolução** (`resolve_ecs.go`) | o TERMO | quanto vale `defense`, e por quê? |
| **derivação** (`derive_ecs.go`) | a FICHA | qual é o número final? |
| **ataque** (`attack_ecs.go`) | a PARCELA DE DANO | de que se soma este dano, e o que ficou de fora? |
| **situação especial** (`special_situations_ecs.go`) | a LINHA da Tabela 5-3 | o que a cena em volta faz com este ataque? |
| **redação do tabuleiro** (`board_ecs.go`) | a PEÇA | quem olha daqui enxerga isto? |
| **ocupação do plano** (`board_occupancy_ecs.go`) | a PEÇA | que quadrado está tomado, e quem já tem peça na mesa? |
| **terreno sob a peça** (`situations_ecs.go`) | a PEÇA | que terreno ela está pisando, e o que ele faz com o ataque? |
| decomposição (`breakdowns.go`) | — | como se explica o número na tela? |

### A derivação, e o que ela CUSTOU

O grafo é raso: uma aresta só — a carga alimenta o deslocamento e as perícias.
Todo o resto lê `(personagem, efeitos)`.

Sem sistemas, o compilador era o escalonador. Com eles, **um sistema que não
roda deixa o componente ausente, e quem o lê recebe ZERO** — sem erro, sem log,
com a ficha mostrando 0 como se fosse a conta. Dois guardas pagam isso:

- `TestEveryDerivationSystemWritesItsComponent` — cada sistema escreveu o
  componente dele. A lista é à mão porque o `ecs` não diz quantos componentes
  uma entidade tem; o DENOMINADOR exige tantas conferências quantos sistemas.
- `TestTheLoadIsDerivedBeforeWhatReadsIt` — a única aresta, prendida pela
  consequência. O personagem é SOBRECARREGADO de propósito: sem isso, a carga
  zerada e a de verdade dão o mesmo deslocamento.

### Os componentes são agrupados por PERGUNTA

Um componente junta o que se lê junto: `derivedMagic` carrega limite de PM, CD e
custo porque quem lê um lê os outros. Um campo por componente seria a struct de
volta, com cerimônia.

### O que a resolução em sistemas comprou

O termo que não cumpre condição, que perde o empilhamento ou que vira fator não
sai do mundo: fica com a tag `Suppressed{Why}`. É o que vai deixar a ficha
responder *"por que 12?"* com "−2 de X, não aplicado por não empilhar" — a tela
ainda não o lê.

### O ataque em sistemas, e a única ordem que a rede NÃO pegava

- **O `d20` entra por parâmetro e os dados por função** (`ResolveAttack`). É isso
  que mantém cada caso do livro um teste com o número escrito à mão. A regra
  mora em `attack_ecs.go`; o `attack.go` tem o vocabulário da fronteira e a
  montagem do mundo.
- A ordem entre os sistemas é presa, quase toda, pelos casos do livro. O que
  eles não prendem é cada sistema ter escrito o componente dele:
  `TestEveryAttackSystemWritesItsComponent`, que traz o CENÁRIO junto com cada
  sistema, porque alguns só agem numa metade do dado.
- **Suprimir é write-once, e a PRIMEIRA razão é a que fica.** Mais de um sistema
  alcança a mesma parcela, então a ordem decide qual razão a tela mostra: o
  bônus de crítico de um ataque que ERROU sai dizendo que o ataque errou, e não
  que "não foi um acerto crítico".
- **A notação de dano é lida antes de se saber se o ataque acertou**, porque a
  parcela precisa existir para ser suprimida. Um `"1d"` no catálogo falha sempre,
  e não só nos ataques que acertam.

### O tabuleiro entrou, e foi ele quem devolveu o `Despawn`

- **Peça e marcador são a mesma espécie de entidade**, e o que o mestre escondeu
  SOME INTEIRO: `Despawn` mata a entidade e apaga todo componente dela. Uma coisa
  "presente porém anônima" entregaria a emboscada do mesmo jeito. O `Despawn`
  pela metade é pego pelos casos de redação do `domain/board`.
- **A ENTIDADE NÃO TROCA entre os gestos do tabuleiro**, ao contrário da ficha: a
  peça responde tanto "você é vista?" quanto "que quadrado você toma". O que muda
  é o COMPONENTE — o `body` só existe para quem pergunta quadrado, o `boundTo`
  só para quem pergunta fila —, e por isso os dois pipelines compartilham um
  construtor de mundo só.
- **A ORDEM da redação é regra**: é a ordem de DESENHO (quem vem depois fica por
  cima), e é ela que decide qual peça a tela oferece para mover. Asserção sobre
  CONTEÚDO sobrevive a um embaralhamento; quem prende são o
  `TestTheRedactionKeepsTheDrawingOrder` e o `TestTheRedactionKeepsTheMarkerOrder`.
- **O CORPO da peça é um componente** (`body`): "ocupa `footprint × footprint`
  quadrados a partir da âncora" mora num lugar só. O footprint zerado vira 1 ali,
  porque a peça que chega do BANCO não passa pelo `AddToken` —
  `TestATokenSavedWithoutASizeStillTakesItsSquare` monta a fatia direto, como o
  banco a entrega.
- **Montar o mundo UMA vez por gesto é o ponto.** O `PopulateBoard` põe a peça
  recém-criada no mundo antes de procurar o quadrado da seguinte — sem isso, as
  duas acham o mesmo lugar livre.

### A FRONTEIRA do tabuleiro abriu para o núcleo, e o preço foi um guarda

O `domain/ecs` está na lista de permitidos do
`TestTheBoardDoesNotKnowTheCharacterSheet` pela mesma razão do `infra/events`:
ele é FOLHA e não sabe o que é Tormenta, então depender dele não deixa a
dependência vazar. A lista avisa: *acrescentar import a ela transforma a porta
em enfeite*.

A folha tem guarda: `TestTheCoreImportsNothingOfTheProject`. A única exceção é o
`world_test.go`, do pacote externo `ecs_test`, que importa o `ecs` por
construção. Os testes continuam varridos de propósito: os componentes de
mentira do núcleo se chamam `label` e `weight` porque *"se uma palavra do livro
aparecer neste pacote, o desenho vazou"*.

### A ORDEM dos sistemas é a regra, em toda fase

O oráculo não cobre ordem: nenhuma fixture tem todas as fontes ao mesmo tempo.
Cada fase tem o guarda dela:

- **coleta**: `TestEveryCollectorLandsInTheDeclaredOrder`.
- **resolução**: `TestAFactorWhoseConditionFailsDoesNotApply` — colher o fator
  antes de julgar a condição. Ele prende pela CONSEQUÊNCIA, e não pela
  sequência, e por isso sobrevive a uma reescrita dos sistemas.
- **derivação**: `TestTheLoadIsDerivedBeforeWhatReadsIt` (acima).
- **redação do tabuleiro**: os dois guardas de ordem de desenho (acima).

**Acrescentar uma fonte NÃO faz o guarda da coleta reprovar.** Um guarda de
ORDEM só mede os pares que o DADO dele põe na mesma ficha, e por isso o
`everyCollectorCharacter` cresce junto com o `collectionSystems`. O controle é
sabotar a ordem depois de mexer: trocar duas linhas tem de sair nomeando as
duas espécies fora de lugar.

### O PODER é uma coisa; a CONCESSÃO dele é outra

O `transcript.ClassPowers` tem duas espécies de linha:

- o **verbete**, sem `className`, com a regra — nome, descrição, página, bloco
  de escolha;
- a **concessão**, com `className`, o nível ou a escolha que destrava, e um
  `powerUid` apontando para o verbete.

A regra é escrita uma vez, no verbete; a concessão aponta. Copiá-la para cada
classe que concede o poder é o caminho para duas regras com o mesmo nome.

**Quem resolve são DOIS pontos, um por leitor.** O `domain/book` resolve para a
tela, em `classPowersResolved`. O MOTOR resolve na carga do catálogo, em
`resolveClassPowerPointers`: a concessão sem nome ou sem modificador próprio
herda os do verbete. Os dois não compartilham código porque isso exigiria uma
aresta entre `domain/engine` e `domain/catalog`, que não se conhecem.
`TestTheAttributeIncreaseLandsOnTheChosenAttribute` prende que o ponteiro chega
à ficha.

**Antes de mudar a forma de um recurso, ache os leitores pelo nome dele**
(`git grep '"class-powers"'`), e não pela memória.

### O verbete tem DOIS identificadores, e eles têm papéis opostos

- **`id`** é o SLUG: minúsculo, sem acento, `[a-z0-9-]`, único dentro do
  recurso — `esquiva`, `bencao-do-mana`, `heroi-campones`. É o PÚBLICO: o que a
  URL carrega no padrão `/<recurso>/<id>`, o que o `grep` acha, o que a fixture
  lê. Ele muda quando o nome muda.
- **`uid`** é opaco e estável: `pwr_k3m9x2ft4q`. É a IDENTIDADE: o que o banco
  guarda e o que um catálogo usa para apontar para outro. Ele **nunca** muda.

O `uid` é opaco de propósito: um identificador derivável do nome volta a mudar
no dia em que alguém "corrigir" a derivação.

Cunhar é `go run ./cmd/mintuids --aplicar` (de `engine-go/`; sem a flag, só
relata), e ela só CRIA: verbete que já tem uid nunca é tocado. O escopo — quais
arquivos e quais caminhos têm verbete — é declarado uma vez só, no
`cmd/mintuids`, e **não se repete em Go**. Os guardas de
`domain/catalog/uid_rules_test.go` (`TestEveryCatalogEntryCarriesAUid`,
`TestNoUidNamesTwoDifferentThings`, `TestNoUidEverChanges`) conferem o que a
cunhagem produziu contra a linha de base `testdata/uid_baseline.txt`, sem saber o
escopo.

Verbete NOVO nasce com uid e entra na base quando alguém a regerar. O que
reprova é um uid EXISTENTE mudar: o banco perde a referência e a ficha deixa de
achar o verbete, sem erro nenhum.

#### Endereço não carrega RÓTULO, e a coincidência esconde isso

- O `TestEveryCatalogIdIsASlug` varre todo objeto com `id`, sem linha de base, e
  recusa caixa alta, acento e espaço. O ponto é namespacing e passa
  (`class.barbaro.impeto`): cada pedaço entre pontos é um slug por si.
- **A prova de que um renome de chave preservou o comportamento é o diff do
  oráculo não ter NENHUMA linha de número.**
- **Escolha guardada como JSON dentro de uma string também é ponteiro** — o
  `"race":"Lefou"` do `secondaryRaceChoices` não se parece com um, e nenhuma
  varredura de campo o alcança. Renomeou chave? Procure ali também.

### O bônus CIRCUNSTANCIAL já tem vocabulário — não invente um

Antes de acrescentar forma nova ao `Modifier`, veja se o caso não é um destes:
`context` com nota é o interruptor que a PESSOA liga, `against` com traço é
"contra o quê", e `terrain` é o tipo de terreno. **Cada `against` é um
interruptor**; modificadores que a pessoa liga JUNTOS dividem uma flag
(`flagOn`), como a Marca da Presa e os acréscimos dela. Quase sempre o que falta
é o dado, não o motor.

Escolhas cujo erro é invisível num teste que só confere números:

- **Alvo que já É o escopo dispensa condição.** "+2 em testes de ataque para
  derrubar" (p125) é `maneuver:derrubar`, sem `condition`: pedir opt-in para uma
  circunstância que a pessoa já escolheu ao usar a manobra faria o bônus valer
  de menos. O inverso — circunstância sem interruptor — faz valer de mais.
- **"Resistência a X +N" é bônus em TESTE**, não redução de dano: "uma criatura
  com resistência a magia +2 recebe +2 em testes" (p226). O alvo é `resistance`.
- **A manobra tem DOIS lados, e o alvo diz qual.** Ela é um teste OPOSTO
  (p234). O `maneuver:<nome>` sem escopo é a rolagem de quem ATACA (o +2 do
  `Derrubar Aprimorado`); com `scope: "defense"` é a de quem resiste (o +5 do
  `Desejo de Liberdade`, p89). O escopo entra na CHAVE do efeito, ou os dois
  somam no mesmo balde.

  **Saber QUE TESTE a pessoa rola não basta para escolher o alvo**; pergunte
  **"que OUTROS números saem desse mesmo teste?"**. A manobra é um teste de
  Luta, e o ATAQUE também: um bônus defensivo em `expertise:Luta` sobe o ataque
  da arma empunhada. Uma perícia que alimenta combate não aceita bônus de recorte
  fino; quem precisa de recorte precisa de alvo próprio.
- **Número impresso SEM alvo não vira modificador**, e o motivo fica escrito: o
  poder que NEGA uma penalidade que o motor não aplica (o "Água no Feijão" nega
  um –5 de fabricação; um +5 valeria sempre para um –5 que vale raramente), ou o
  poder cujo TESTE o livro não define (o "+5 no teste" de extrair recursos do
  "Vendedor de Carcaças"). O guarda é o
  `TestEveryBonusAnOriginUniquePowerPrintsBecomesAModifier`: lista de
  PERMITIDOS, cada linha dizendo POR QUE não há alvo.

### O DANO TEM TRÊS PARCELAS, e só uma multiplica no crítico

A p231 dá o critério:

> "Multiplica os DADOS de dano do ataque (incluindo quaisquer aumentos por
> passos) pelo multiplicador da arma. Bônus numéricos de dano, **assim como
> dados extras**, não são multiplicados."

| parcela | onde mora | multiplica? |
|---|---|---|
| os dados da ARMA | `WeaponCard.Damage` ("1d8") | **sim** |
| os dados EXTRAS de um encanto | `WeaponCard.ExtraDamage`, com o TIPO | não |
| o bônus numérico | `WeaponCard.DamageBonus` | não |
| o que só vale no crítico | `WeaponCard.CriticalBonus` | não, e fora do crítico nem existe |

O tipo viaja com a parcela extra porque é ele que dá à resistência a fogo onde
agir.

**Estas duas formas do `Modifier` não são parcela da pilha**, e o
`harvestWeaponRiders` as tira do mundo ANTES do `deferConditionals`:

```
{"target": {"k": "damage", "damageType": "fogo"}, "dice": "1d6"}
{"target": {"k": "damage"}, "amount": 10, "condition": {"c": "onCritical"}}
```

O `onCritical` é condição que **não se liga**: chegando ao `deferConditionals`,
viraria interruptor na aba Efeitos, e crítico não é escolha de ninguém. E o dado
extra virando parcela somaria como `amount` ZERO — um modificador que existe e
não faz nada. Os dois casos estão presos no `weapon_riders_rules_test.go`.

**O que denuncia o erro aritmético é a CONTAGEM de dados, não o total.** Num
crítico x2 com arma flamejante o certo é 2d8 + 1d6; com o extra multiplicado
seriam 2d8 + 2d6, e com dados presos os dois totais são plausíveis. Por isso o
dublê do `attack_rules_test.go` estoura quando a regra pede um dado a mais do
que o caso preparou.

### A escala por PATAMAR, e o `default` que engole o `per` desconhecido

`scale.per: patamar` multiplica pelo ORDINAL do patamar — 1 no iniciante, 2 no
veterano, 3 no campeão, 4 no lenda —, e o "Coração Heroico" (p91) cabe num
modificador só, `maxPm 3`, dando 3/6/9/12. O `levelStep` não serve: os
limiares (5, 11, 17) não são um passo fixo.

**O `switch` do `evalModifierScale` tem `attribute` no `default`**: um `per`
desconhecido não estoura — procura `attrTotals[""]`, acha zero, e o modificador
some da ficha **sem erro nenhum**. Quem torna a diferença visível é o
`TestNoCatalogScaleIsOneTheEngineCannotRead`, que varre o catálogo contra a
lista de escalas conhecidas.

Para achar um poder no PDF, case por `bookpdf.Key` e procure **dentro da página
que o catálogo declara**: o nome vem em VERSALETE (o `pdftotext` o devolve em
caixa baixa), e o texto do capítulo tem a TABELA de pré-requisitos antes dos
verbetes.

### A SOMA DE ATRIBUTO: o `scale` fora dos vitais

"Você soma seu X em Y" é um modificador com `scale: {per: attribute, attribute}`
num alvo qualquer — Defesa, dano, perícia —, e o `cap: classLevel` o limita ao
nível NA CLASSE que concede o poder (p226). O teto só entra onde o verbete
escreve "limitado pelo seu nível": a p226 diz que ALGUMAS habilidades são
limitadas, e a Presença Paralisante e a Mente Criminosa não são.

- **Ela se resolve numa SEGUNDA passada** (`sheetEffects`), porque o número é o
  atributo EFETIVO, com os itens. Por isso uma soma não pode mirar um atributo
  (`TestNoAttributeSumTargetsAnAttribute`), e toda decomposição da ficha lê o
  `sheetEffects`, e não o `ComputeItemEffects` direto.
- **O atributo negativo SUBTRAI** (decisão do dono): o livro não tem regra
  própria, e o uso dele é somar o valor como é — "10 + sua Destreza" (p106).
- **A trava que nomeia duas coisas usa `flags`**: o `flagOff` cala se QUALQUER
  uma estiver posta ("de armadura pesada ou na condição imóvel", p47), e o
  `flagRequired` pede todas. A condição Imóvel (e quem a contém) levanta a flag
  `imovel` para isso.
- **O dano por TIPO de arma** cai nos escopos `melee-light-or-agile` e `ranged`
  (arremesso e disparo, p142), lidos pela carta da arma junto do `DamageOwn`. O
  ATAQUE por escopo de arma (o `weapon:<id>` do Foco em Arma) é o `AttackOwn`, e
  a decomposição da tela o soma.

### A devoção é fonte da coleta, e não só de PV e PM

O poder concedido de quem é devoto é uma FONTE da coleta
(`godPowerActiveItem`), e chega a qualquer alvo. O `vitalGrantMods` continua
somando PV e PM, e a fonte não duplica esse número.

### Os acumuladores moram numa entidade SINGLETON

As flags e os fatores são do mundo inteiro, e ficam numa entidade só, com dois
componentes (`worldFlags`, `worldFactors`) — o idioma de "recurso" do ECS. Sem
ele, um sistema escreveria numa variável capturada e deixaria de ser função do
mundo, que é o contrato do `ecs.System`.

## O deslocamento é contado em QUADRADOS, e metro é coisa de tela

Todo deslocamento que o livro imprime é múltiplo de 1,5m. Guardado em metros, ele
arredondaria na fronteira do JSON e inventaria bônus; em quadrados, a conta fecha
exata. É a mesma unidade do tabuleiro — *"a conta é feita em QUADRADOS inteiros,
nunca em metros… metro é coisa de tela"* (`board_movement.go`, p236) —, e a
metade do Lento (p395: *"arredonde para baixo para o primeiro incremento de
1,5m"*) vira divisão inteira, sem regra de arredondamento escrita à mão.

### A conversão é SIMÉTRICA, e isso não é enfeite

`amountInEngineUnits` na leitura, `amountInBookUnits` na escrita
(`Modifier.MarshalJSON`). Sem a volta, ler de novo o que o motor escreveu
converte duas vezes. Quem pega a assimetria é o `roundTrip` dos casos de
paridade, que serializa a saída do motor e a relê.

**O FIO fala a unidade do LIVRO**: o oráculo mostra −3 e +1,5 como a página
imprime, e quem revisa o diff contra o livro compara os mesmos números.

### O modificador que MULTIPLICA

Quase toda regra do livro SOMA; algumas multiplicam — o Lento reduz o
deslocamento à metade (p395), o Imóvel o reduz a 0m (p394), a Alta Arcana corta
o custo em PM pela metade.

O `engine.Ratio` é um fator racional em inteiros (`{1,2}` é metade, `{0,1}` é
zero), num campo PRÓPRIO do `Modifier`. Quem tem fator não é parcela da pilha: o
`Amount` fica em zero e o valor sai do `byTarget` para o `ItemEffects.Factors`.
Racional e não decimal porque o motor é modelado em inteiros, e a divisão
inteira de um total em quadrados é exatamente o arredondamento que o Lento pede.

**São DUAS FAMÍLIAS, e elas se distinguem pelo ALVO.**

- O **fator de ALVO** (`{k: displacement}`, `{k: pmCost, scope: arcana}`) age
  DEPOIS da soma, e sobre a BASE junto: a Alta Arcana escreve "reduzido à metade
  (APÓS aplicar aprimoramentos e outros modificadores)". O Lento não corta o
  bônus de item de ninguém — corta o quanto a pessoa anda.
- O **fator de FONTE** (`{k: source, name: <id do poder>}`) age na PARCELA
  daquela fonte, ANTES do empilhamento. A Fúria Titânica (p42) dobra "o bônus
  que você recebe quando usa Fúria", e o exemplo trabalhado é quem separa as
  leituras: 5 PM gastos dão +10, e não o dobro do ataque inteiro. Com o degrau
  pago, o balde `morale` tem a base e o degrau, os dois entram dobrados, e o
  não-empilhamento escolhe o maior.

**O fator de fonte tem DOIS sítios de aplicação.** O
`scaleWhatTheSourceFactorHits` cuida do caminho incondicional, no ECS; o
`ApplyActiveConditionals` multiplica a parcela DEFERIDA, que não é multiplicada
no ECS — dobrada nos dois sairia dobrada duas vezes. O bônus de Fúria é
condicional, então passa pelo segundo.

**Dois fatores no mesmo alvo: vale o MAIS SEVERO, e eles não compõem** — com ou
sem condição. O livro proíbe o acúmulo nas duas famílias: "aplique apenas o mais
severo" (p394) para condição, *"reduções no custo de PM não são cumulativas"*
(p226) para custo. Lento mais Imóvel dá 0m.

**O Foco em Magia é a exceção ESCRITA**: −1 PM na magia escolhida, *"cumulativo
com outras reduções de custo"* (p131). Ele não é modificador `pmCost` — entraria
na disputa da p226 —, e o `SpellPmCostFor` o soma por fora, antes do fator e do
piso.

**O fator pode ter ESCOPO.** A Alta Arcana corta "suas magias arcanas" (p39), e
não toda habilidade que custa PM: a chave é `pmCost:arcana`, e o clérigo do
multiclasse paga cheio pela divina dele — mesma divisa do `displacement:armor`.
Quem responde de que lista a magia vem é a `sheet.SpellTradition`, porque **a
tradição é da CLASSE e não da magia**: a Luz é arcana na mão do arcanista e
divina na do clérigo.

> **A ISENÇÃO não é uma terceira forma do fator — ela é FLAG.** O anão "não tem o
> deslocamento reduzido por uso de armadura ou excesso de carga" (p20), e quem
> responde são o `DisplacementIgnoresArmor` e o `DisplacementIgnoresLoad`, lidos
> pelo `displacementBreakdown`: a isenção pula a chave `displacement:armor` e a
> penalidade de carga, e não toca em fator nenhum — uma magia de lentidão
> continua valendo para o anão. **São duas flags porque o livro as vende
> separadas**: o Fanático (p128) compra só a da armadura. Isenção de terreno
> difícil (Acrobático, p129) não é da ficha: é do `PathCost` do tabuleiro.
>
> Confira na PÁGINA, e não na descrição do catálogo, antes de concluir que um
> poder pede isenção nova: o corte pode ser da CIRCUNSTÂNCIA, que o app não
> modela.

### Onde a unidade aparece, e onde ela não aparece

| lugar | unidade |
|---|---|
| catálogo (os literais de `domain/catalog/transcript`) | metros |
| `Modifier.Amount` em memória | quadrados |
| `Modifier` no fio (JSON) | metros |
| `ComputedSheet.Displacement` (`json:"displacementSquares"`) | quadrados — e o nome do campo diz |
| `ItemEffects.ByTarget["displacement"]` | quadrados, e o nome do alvo é a única pista |
| a tela (crachá da mochila, régua da Mesa) | metros |

**Trocar a unidade de um campo do fio troca o NOME dele** (`displacementSquares`):
trocar calado faria todo leitor antigo ler 6 onde lia 9, sem erro em lugar
nenhum.

## Catálogos

**As cenas leem o catálogo EMBUTIDO direto** — ele não viaja por HTTP, e não há
rota que o sirva.

Quem o lê lê **uma vez e guarda**, e não por requisição: o livro é compilado no
binário e não muda enquanto ele for o mesmo. É o que o `race_traits.go` faz com
`sync.Once`.

**O LIVRO é código Go**, em `domain/catalog/transcript`: cada verbete é um
literal tipado, e é ali que uma regra transcrita se edita. A cena, o motor e os
testes leem o mesmo valor, e o `catalog.Resource` o serializa para os leitores
que decodificam JSON (o `transcript.ResourceJSON`). É Go para o compilador
cobrar a chave escrita errado.

- **A forma dos arquivos de dados é CANÔNICA**, e quem a escreve é o emissor
  (`transcript.WriteSource`). O `TestTheTranscriptIsInCanonicalForm` reprova o
  arquivo que saiu dela; depois de editar à mão, `go run ./cmd/transcriptfmt`.
  É isso que deixa as ferramentas que gravam no catálogo (`cmd/mintuids`,
  `cmd/bookpages`, `cmd/audit/bestiary --aplicar`) mudarem o VALOR e
  reescreverem o arquivo, sem uma cirurgia de AST por ferramenta.
- **Comentário dentro de um literal não sobrevive à reescrita**, e isso é a
  regra: o porquê de um verbete mora no `modeled`/`unmodeled` dele, que a ficha
  lê.
- **Os tipos (`*_types.go`) e o registro (`layout.go`) se escrevem à mão**; o
  emissor não os toca. Recurso novo é uma linha no `layout.go`.
- **`Modifier.Amount`, `Item.Price` e `Item.Slots` não distinguem ausente de
  zero.**

**O que uma CAMPANHA acrescenta ao livro não mora ali, e nem podia**: o livro é
compilado no binário, e a mesa autora a dela em tempo de execução. A `emenda de
catálogo` vive em `campaign_items`, ACRESCENTA modificador a um verbete
existente e nunca o substitui — ver a linha dela no
[GLOSSARY.md](../GLOSSARY.md).

**O mundo é um TIPO, e é o compilador que cobra.** `engine.Catalogs` é o LIVRO
e só responde consulta de verbete; quem computa uma ficha precisa de um
`engine.Ruleset`, que é o livro sob as emendas de UMA mesa. Não há como esquecer
de dizer em que mundo a conta acontece: `book.ComputeSheet(…)` não compila.

Fora de campanha se escreve `engine.BookRuleset(book)`, e a linha é explícita de
propósito — o molde do elenco, o oráculo e os fixtures estão TOMANDO essa
decisão, e ela aparece em vez de ser o que acontece quando ninguém disse nada.

As emendas não viajam no `engine.Character`: no personagem, campanha-inteira e
personagem-específico chegariam ao motor já misturados, e a procedência — que é
o que a ficha usa para dizer de onde veio cada termo — morreria no caminho.

Quem resolve o mundo é o carregamento da ficha (`sheet.RulesetFor`), UMA vez, e
ele viaja no agregado (`CharacterDTO.Ruleset`, `json:"-"`) até quem computa. O
lote da mesa (`sheet.RulesetsFor`) faz uma consulta por campanha DISTINTA, e não
uma por ficha.

### As três espécies de emenda, e por que não são uma lista só

| espécie | tabela | o que muda | escopo |
|---|---|---|---|
| **verbete** | `campaign_items` | o que uma COISA é | a campanha inteira |
| **concessão** | `campaign_grants` | quem TEM as coisas | seletor |
| **silêncio** | `campaign_silences` | o que esta mesa NÃO aplica | seletor |

Elas não viram uma tabela de "regras" com um `kind`: respondem a perguntas
diferentes e cada uma tem a forma dela. O que as une é a campanha, não o
formato. A de verbete não tem escopo de propósito: um item que significa uma
coisa para você e outra para o vizinho de mesa não é um mundo, são dois.

O **seletor** (`engine.Selector`) tem uma espécie — `everyone`, a mesa inteira,
ou `character`, uma ficha — e o id é o do CLONE que vive na campanha. Espécie
desconhecida não alcança ninguém, e o id ZERO nunca casa: o `Character{}` dos
fixtures tem id zero, e casá-lo faria a emenda de uma mesa aparecer no oráculo.

### O endereço de um termo NÃO carrega o valor

`engine.TermID` é `fonte::alvo::escala::condição`, mais a categoria e a peça
(`From`) só quando existem. Valor, tipo de bônus e prosa (`note`, `label`) ficam
de fora: corrigir um número ou uma frase no livro trocaria o endereço, e o que
estava preso a ele evaporaria sem aviso. A escala e a condição são os
discriminadores porque separam os pares que existem — o anão tem `maxPv +2` e
`maxPv +1 por nível`.

Quem garante "é único", e não só "é único hoje", é o
`TestEveryCatalogTermHasAUniqueAddress`. Duas sutilezas dele:

- **A raça é UMA fonte**: o `raceActiveItems` junta os modificadores de todas as
  habilidades num `ActiveItem` com o `race.ID`.
- **O mesmo benefício de origem é oferecido por várias origens**, e o
  `getOriginBenefit` lê o primeiro que casa o id: medir cada cópia acusaria a
  fonte colidindo consigo mesma.

### Um endereço só, e o opt-in do jogador usa o mesmo

O opt-in do jogador (`character_conditionals`) e o silêncio do mestre gravam o
mesmo `TermID`. **Mudar a forma do endereço é migração**: linha gravada na forma
velha não casa com nada, e a ficha mostraria o situacional desligado para sempre.
É por isso que a categoria e a peça só entram no endereço quando existem.

### JOGADOR NÃO MUDA REGRA — isso é do mestre (decisão do dono)

É a linha que separa as duas famílias de escrita, e vale para toda tela que
vier. O jogador alterna o OPT-IN de um condicional que a ficha dele já oferece;
quem AUTORA regra — emenda de verbete, concessão, silêncio — é o mestre, na
campanha dele.

**Quem cobra é o servidor; a tela esconder não conta.** Um POST montado à mão
com um interruptor que a tela nunca ofereceu tem de ser recusado pelo caso de
uso. As posturas ficam fora da lista de situacionais, porque entrar nelas custa
PM.

O conjunto aceito é uma lista de PERMITIDOS (`character.SituationalGroupsOf`), e
ela tem UM dono: a cena desenha a partir dela e o caso de uso recusa o que não
estiver nela. Duas implementações divergem, e a divergência aparece como a tela
escondendo o que o servidor aceita.

**`character_conditionals` FICA, e não é o mesmo que `campaign_silences`**,
ainda que as duas guardem endereços de termo. O opt-in nasce DESLIGADO e é do
dono da ficha; o silêncio nasce LIGADO e é de quem manda na mesa. Juntá-las numa
tabela só juntaria duas permissões de escrita que não podem ser a mesma.

### O interruptor de GRUPO tem endereço próprio

Os condicionais que dividem uma flag viram UM interruptor na tela — "um item
caseiro com três modificadores é uma coisa só na mesa". A chave do grupo é o
`engine.FlagGroupID`, e não o endereço de um membro: assim "metade ligada" não é
representável. Vale para a POSTURA e para as flags fora dela — o Folião, o
terreno do Explorador e a `marca-da-presa`, que junta o dado da Marca e os
acréscimos dela num interruptor só.

### O catálogo chega ao processo por dois caminhos

O `catalog.Resource` serve o `transcript`, que é Go compilado no binário: existe
sempre que o binário existe. O `s.catalogs` é o mesmo livro indexado pelo motor
(`book.EngineCatalogs`), mas é um CAMPO que a bancada troca — há fixture que
prima `{"items":[]}` de propósito. **Validação de regra lê o embutido**; o
`s.catalogs` fica para o motor.

### Regra que só a TELA sabe é fronteira aberta

O sintoma é sempre o mesmo: a tela tranca e o servidor não. A progressão de
círculo (`Spellcasting` em `transcript.Classes`), a compatibilidade entre
melhoria/material e o item (`aceitaMelhoria`, no comando da ficha) e o limite de
texto da campanha (`domain/campaign/rules.go`) moram no servidor. O filtro que a
tela aplica é conveniência sobre a mesma regra, nunca a regra.

### A validação de ESCOLHA é do servidor, e é estrita

As regras de escolha de poder — quantas vagas o nível abre (uma por nível a
partir do 2º, p33), quantos benefícios a origem dá, quais caminhos e quais deuses
cada classe aceita — são conferidas na gravação, não só no diálogo.

- **`sheet.WithChoicesValid`** roda nos comandos de escolha da ficha
  (`saveTheChoices`) e é **estrita**: a escrita tem de deixar a ficha VÁLIDA, não
  só "não piorar". O projeto ainda não foi usado numa mesa real, então não há
  ficha antiga para proteger (decisão do dono).
- **`powerChoices`**: só poder que a ficha tem, só opção que o livro oferece, mais
  de uma só no poder repetível. O repetível ocupa uma vaga por ESCOLHA
  (`sheet.UsedPowerSlots`), não uma por poder.
- **Poder GERAL que pede escolha** (Foco em Perícia, Foco em Arma) entra pela
  mesma máquina via `book.PowerWithChoice`; a escolha dele que veio da origem não
  ocupa vaga (`sheet.FreePicks`).
- **Escolha contada por ATRIBUTO** (`countAttribute`, as perícias do
  Especialista, p73) não ocupa vaga e depende da ficha — perícia treinada, no
  máximo o atributo sem temporários (`Ruleset.PermanentAttribute`), mínimo 1. Por
  isso todo gesto que mexe no N ou no treino (treino, ofício, guardar/vestir/
  remover item, distribuição da raça, atributo na forja) aplica a mudança numa
  cópia e chama `sheet.ChangeKeepsChosenExpertises` antes de gravar.
- **Escolha contada por NÍVEL** (`countLevels`, os terrenos do Explorador, p50)
  repete opção sem gastar vaga, e descer o nível confere
  `sheet.LevelCountedChoicesFit` no nível novo.

### Schema pega FORMA, não número errado

O que protege dado transcrito é **validação de schema**
(`domain/catalog/rules_tables_test.go`), não um `expect` por campo: o risco é
typo, não regressão. Ela cobre o que quebra tela — perícia que não existe, faixa
de rolagem com buraco, termo de devoto apontando para raça inexistente.

**Ela não cobre número errado**: um inteiro no lugar de outro inteiro é schema
válido. Contra isso só serve conferir contra o livro, com o auditor:

```
cd engine-go
go run ./cmd/audit/bestiary            # relatório
go run ./cmd/audit/bestiary --aplicar  # escreve as correções
```

### Os auditores do livro

Cada auditor é um comando em `cmd/audit/<nome>`, e o livro (gitignorado) se acha
sozinho — `T20_BOOK_PDF`, um `t20-book.pdf` subindo do diretório, ou o do
checkout principal quando se está numa worktree. A leitura do PDF mora em
`infra/bookpdf`, compartilhada por todos; ela usa `\s` e `\w` Unicode
(`IsPySpace`), porque o livro tem espaço não separável.

Rodar é **ato deliberado**, como o `genoracle`: a ferramenta PROPÕE lendo o PDF,
e é o diff revisado contra o livro que decide.

**Mexeu num catálogo? rode o auditor dele.** São um por seção do sumário, e o
que muda entre eles é a ÂNCORA — o par de campos que diz "este pedaço do livro é
este verbete":

| auditor | catálogo | âncora |
|---|---|---|
| `cmd/audit/bestiary` | `transcript.Bestiary` | Defesa + Pontos de Vida |
| `cmd/audit/spells` | `transcript.Spells` | nome + escola, para os aprimoramentos; para a cobertura, os links em negrito do índice (p174-177) **e** o título do verbete (`Tormenta20` a 24pt) |
| `cmd/audit/classes` | `transcript.Classes`, `transcript.ClassPowers` | nome na prosa, ou nível na tabela |
| `cmd/audit/equipment` | `transcript.Items` | nome + preço na tabela |
| `cmd/audit/races` | `transcript.Races` | linha de atributos, ligada por semântica |
| `cmd/audit/expertises` | `transcript.Expertises` | Tabela 2-1 **e** a linha de estado do verbete |
| `cmd/audit/gods` | `transcript.Gods` | Tabela 1-20 **e** os campos rotulados do verbete |
| `cmd/audit/conditions` | `conditionModifierTable` (Go!) | a HERANÇA que a primeira frase do verbete declara |
| `cmd/audit/origins` | `transcript.OriginSources`, `transcript.Origins` | Tabela 1-19 **e** o verbete, que é o título com `Benefícios.` abaixo |
| `cmd/audit/powers` | `transcript.GeneralPowers`, `transcript.TormentaPowers` | a tabela de pré-requisitos **e** a linha `Pré-requisito:` do verbete |
| `cmd/audit/grantedpowers` | `transcript.GrantedPowers`, `transcript.DivinePowers` | a coluna "Devoto de X" da Tabela da p127 **e** o rótulo de deus do verbete |
| `cmd/audit/enchants` | `transcript.Items` (`weapon-enchant`, `armor-enchant`) | as Tabelas 8-8 e 8-10 (`d%` + encanto + efeito) **e** o verbete `Nome. Regra` |
| `cmd/audit/materials` | `transcript.Items` (`material`) | a Tabela 3-9, uma MATRIZ de preço, **e** o verbete com as metades `Arma.` / `Armadura e Escudo.` / `Esotérico.` |
| `cmd/audit/objects` | `objectExamplesOfTheBook` + as duas escadas (Go!) | a Tabela 5-4, lida por COORDENADA — a p239 tem duas tabelas lado a lado |
| `cmd/audit/specificitems` | `transcript.Items` (os de `composedOf`) | as Tabelas 8-9 e 8-11 (`d%` + nome + preço), o verbete `Nome. Regra` **e o ITÁLICO dele**, que é a composição do item |
| `cmd/audit/potions` | `transcript.Items` (os de `consumable.spell`) | a Tabela 8-12 (`d%` + a MAGIA + preço), resolvida contra o catálogo de magias — a poção não tem verbete |
| `cmd/audit/accessories` | `transcript.Items` (`apparel` com faixa de `d%`) | as Tabelas 8-13/14/15, com UMA lista alfabética de verbetes para as três |
| `cmd/audit/artifacts` | `transcript.Items` (`artifact`), `transcript.GMTables` (`chaosDeck`) | NENHUMA tabela — a p346 tira o artefato da geração aleatória. O denominador vem do TIPO: o cabeçalho de seção e o corpo têm corpos de letra diferentes, e nas p346-349 não há outra seção |

**Dois não leem o livro.** O `cmd/audit/coverage` confere os verbetes de raça,
classe e origem contra o MOTOR — quanto de cada um vira número, gesto ou nota do
mestre. O balde é DECLARADO no verbete (`modeled`/`unmodeled`), a nota cita a
issue da fatia que a resolve, e `--destino` agrupa por ela. Verbete sem efeito no
motor e sem `unmodeled` reprova. Não tem guarda em Go, por decisão do dono. O
`cmd/audit/pointers` lista os arquivos citados em comentários de teste que não
existem mais; a saída são candidatos, não veredito.

### O que vale para todo auditor

- **A âncora é o trabalho; o resto é regex.** Ela só aceita um bloco contíguo e
  completo cujos campos-âncora já batam com o catálogo. O que não se liga a um
  verbete REPROVA com nome em vez de ser adivinhado.
- **Ler por coordenada, nunca por `-layout`.** O `pdftotext -layout` junta
  colunas VIZINHAS na mesma linha. Quando nem a linha serve — o `<line>` junta
  título, estado e o título da coluna vizinha na mesma altura —, leia por
  PALAVRA (`Book.Words`).
- **Toda tabela de rolagem dá denominador de graça: as faixas `d%` têm de
  ladrilhar o intervalo.** Uma faixa é uma decomposição, e a soma se afirma antes
  das parcelas (`bookpdf.RangesThatDontClose`).
- **Duas âncoras se conferem uma contra a outra ANTES de qualquer uma encostar
  no catálogo.** Quando o livro imprime a mesma verdade em dois lugares, uma
  leitura errada vira discordância nomeada em vez de tabela plausível.
- **O que o motor NÃO aplica sai com denominador, não como lista de proibidos.**
  Todo encanto ou tem modificador, ou diz por que não — nunca os dois, porque um
  motivo ao lado de um modificador é um motivo que envelheceu. Quem cobra sem o
  PDF é o `TestEveryEnchantEitherAppliesOrSaysWhyNot`. Nos materiais, `modeled` e
  `unmodeled` são PERMITIDOS por METADE do verbete, e a união tem de dar
  exatamente as partes dele.
- **Medidor de prosa diz onde a prosa acaba**, e o relatório imprime a cobertura
  mínima: sem recortar por coluna, palavras de outro assunto inflam a medida.
- **A DIREÇÃO da medida é do TERRENO.** Quando o catálogo escreve uma forma
  EQUIVALENTE à do livro, a pergunta é "a regra do livro está aqui?". Quando ele
  RESUME, é a inversa: "tudo o que o catálogo afirma está na página?". Vale para
  números, perícias nomeadas e cobertura de palavras (`bookpdf.Coverage`). O
  veredito sobre a frase continua humano, e por isso as duas saem lado a lado.
- **Filtro de lixo é um só**: o `WithoutJunk` do `bookpdf` (marca d'água, número
  de página, cabeçalho de capítulo).

### A FRONTEIRA JSON do `Modifier`

O `Modifier` tem `MarshalJSON` e `UnmarshalJSON` por UMA razão: o `amount` do
catálogo está na unidade do LIVRO, e o motor guarda deslocamento em quadrados.

- **Na leitura o tipo é EMBUTIDO** — um alias local do próprio `Modifier`, que
  mata o método e evita a recursão —, e só o `amount` é redeclarado, porque o
  campo menos profundo vence o embutido. Campo novo atravessa sem ninguém
  lembrar.
- **Na escrita os campos ficam à mão**, e tem de ser assim: o oráculo compara
  byte a byte, e embutir jogaria o `amount` para o fim da linha. Quem cobra essa
  metade é o `TestEveryModifierFieldSurvivesTheCatalogJSON`, que varre o TIPO por
  reflexão em vez de repetir a lista num terceiro lugar.

Teste que monta o `Modifier` em Go não atravessa essa fronteira, e não a prova.

### A LEITURA de tabela e de verbete mora no `bookpdf`

As tabelas `d%` + nome + preço do capítulo de Recompensas são lidas por vários
auditores, e a leitura mora no `bookpdf`: `Book.LootRows`,
`RangesThatDontClose`, `Book.EntriesByName`. As armadilhas que elas dividem —
a linha se monta pelo Y, a última linha pode vir colapsada num `<line>` só, a
célula do `d%` é alinhada à direita (o x é faixa), e a margem muda de página par
para ímpar — estão lá e não em cada auditor.

**Leitura que um segundo auditor precisaria copiar vira função do `bookpdf`.**
Um refactor de leitura se prova com o relatório dos auditores BYTE A BYTE igual
antes e depois: um número que muda é indistinguível de uma correção, e o diff é
o que separa os dois.

### O ITÁLICO é dado, e ler estilo pede outro leitor

O `-bbox` do `pdftotext`, que é o que o resto do `bookpdf` usa, **não marca
estilo nenhum**. Quem marca é o `pdftohtml -xml`, e ele responde noutra régua. O
`Book.Italics` converte pela razão entre as alturas que os dois declaram para a
MESMA página e devolve na coordenada do resto do módulo; quem misturar as duas
leituras sem ele compara alturas de réguas diferentes.

### E o buraco INVERSO: o valor existe, mas não se confere que ele APONTA

O schema pega FORMA e não VALOR. No `cmd/seed/seed-data.json` os valores são
strings bem formadas, e o que importa é se elas acham alguma coisa no catálogo:
raça, origem, classe, deus, poder concedido, tamanho, magia e item passam por
lookup (`cmd/seed/catalog_refs.go`).

**Seed que mente é caro porque não quebra**: um id de arma errado produz um
personagem sem a arma, com a ficha abrindo normal. E o e2e roda contra a seed,
então ele passa a medir o ambiente em vez do app.

O que vale para qualquer validador de referência:

- **A mensagem sugere o VIZINHO**, porque digitação erra por pouco. O teto de
  distância — um terço do comprimento — separa sugestão de chute: sugestão errada
  é pior que nenhuma, porque quem lê a segue.
- **Ele junta TUDO antes de falhar.** Parar no primeiro erro faz quem escreveu
  cinco ids errados rodar o gerador cinco vezes.
- **Ele afirma o próprio DENOMINADOR.** O catálogo é carregado uma vez
  (`sync.Once`) e engole erro de carga: com a carga quebrada toda lista vem
  vazia, e um validador ingênuo acusaria TODOS os nomes do arquivo. Lista vazia
  é falha de carga, e a mensagem diz isso.
- **Referência escondida em CHAVE de objeto não se parece com referência** — a
  chave do `classChoices` é um nome de classe. O
  `TestEveryCreateFieldOfTheSeedIsClassified` obriga a classificar campo por
  campo, e guarda que força a varredura acha o que a leitura não acha.

## O fixture do app prima o catálogo DE VERDADE

**Catálogo vazio no fixture é validação desligada em silêncio.** Quando uma
checagem responde "não sei" para o que não está no catálogo — e quase todas
respondem, porque recusar o desconhecido travaria fichas antigas —, primar vazio
é escolher que ela nunca rode, com o guarda verde afirmando o contrário do que
mede.

Por isso o `newTestServer` prima o mesmo catálogo embutido que o servidor carrega
(`book.EngineCatalogs`). O fixture que prima `{"items":[]}` é exceção declarada
(ver "O catálogo chega ao processo por dois caminhos").

## A bancada dos testes: um molde migrado, copiado por teste

O `newTestServer` abre um SQLite de VERDADE por teste — é o que faz o `serve/api`
provar composição em vez de mock. O caro não é o banco, é MIGRAR: o `db.Open`
roda as migrações todas, e cada `fsync` custa o que o disco do `TMPDIR` cobra.

**O molde** (`infra/db/testdb`): o `TestMain` migra UM banco e cada teste o
copia (`testdb.Fresh`). Cada pacote que usa declara uma linha:

```go
func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }
```

- **`PRAGMA synchronous=OFF` fica no helper de teste (`newTestServer`) e NUNCA
  no `db.Open`.** Banco que morre no fim do caso não tem o que proteger; em
  produção essa linha é perda de dados do mestre.
- **A cópia FALHA ALTO** — o `Close` volta o erro, o `Sync` força os bytes, e o
  TAMANHO é comparado com o do molde. Uma cópia parcial se comporta como um banco
  NÃO MIGRADO: o sintoma é `no such table`, e quem o vir vai caçar migração.
- **O molde é PACOTE e não um `_test.go`.** Arquivo de teste não exporta nada, e
  a cena que virasse pacote próprio acharia a bancada inalcançável e escreveria a
  dela — que é como um fixture nasce com catálogo vazio.

O que NÃO é compartilhável é a montagem do servidor: ela precisa do tipo
`api.Server`, e um pacote de bancada que o importasse seria importado de volta
pelos testes dele. O molde é a parte cara; o fixture é de cada pacote.

## Testes

As faixas, o vermelho antes de confiar e o que não merece teste estão na seção
"Testes" do [CLAUDE.md da raiz](../CLAUDE.md). O que é deste pacote:

- `go test ./...` — sem flag, sem setup.
- **Teste de regra vive junto da regra e cita a página** — e, quando o livro tem,
  o exemplo trabalhado (`TestBolaDeFogoWorkedExample`). Ele não substitui o
  ORÁCULO, nem o oráculo a ele: o oráculo prende a ficha inteira e acusa qualquer
  número que mude sem ter sido pedido; o teste de regra explica POR QUE aquele
  número é aquele.
- **Provar o vermelho em Go é barato**: inverta o operador, rode o caso, confirme
  o vermelho, reverta. Para um guarda novo, a árvore de ONTEM é o caso vermelho
  real — `git archive` do commit anterior, o guarda copiado por cima, `go test` —,
  e não há como ela sair inerte como uma sabotagem.

## `convention`: os guardas que não são de pacote nenhum

`convention/` não tem código de produção. Ele existe porque duas frases do
CLAUDE.md da raiz se contradizem quando a regra é sobre TODOS os pacotes: "regra
mecanizável vira guarda" e "o guarda mora no pacote que a possui". Pôr uma regra
do repositório no `api` seria escolher um dono arbitrário.

**Quantos são se pergunta ao código**:

```
grep -h '^func Test' convention/*_test.go | wc -l
```

**Um guarda nasce aqui quando alguma destas três vale:** a regra é do
REPOSITÓRIO e não de um pacote; ele varreria `.` e mediria por acaso o diretório
em que mora; ou ele é da família das CITAÇÕES (arquivo, teste e símbolo citados
que não existem mais).

- **O terreno é o DIRETÓRIO, nunca um padrão de nome.** Guarda que varre `.` mede
  o diretório em que por acaso mora, e glob de nome deixa de casar quando um
  arquivo muda de nome — nos dois casos o guarda segue VERDE medindo menos.
- **Guarda que proíbe a cópia porque existe o original tem de falhar quando o
  original sumir**, senão cobra uma regra cuja razão morreu. O
  `TestNoHandwrittenFocusRing` lê o `index.css` e afirma que a regra global
  continua lá.
- **A LÁPIDE é declarada, não apagada.** `// Aqui morava o TestX, que prendia…`
  diz por que uma garantia SAIU, que é o que o `git log` esconde de quem lê o
  arquivo; os guardas de citação pedem que ela entre em `tombstones`,
  `arquivosAusentesDePROPOSITO` ou `simbolosAusentesDePROPOSITO`.

**Idioma não é guarda deste pacote, nem de nenhum** — ver "Isto NÃO tem guarda,
e é decisão do dono" no [CLAUDE.md da raiz](../CLAUDE.md). Nome PRÓPRIO do livro
num identificador não é prosa em português: `TestBolaDeFogoWorkedExample` é o
nome da magia.

## templ — as armadilhas que já custaram tempo

As cenas são `.templ` compiladas para `.go` por `go tool templ generate`. O
`.templ` e o `_templ.go` andam juntos, e a CI recusa o par desencontrado.

- **`else if` NÃO existe numa lista de atributos, e o templ não reclama.** Ele
  escreve ` else` como TEXTO e abre um `if` INDEPENDENTE: os DOIS ramos saem, o
  navegador guarda o PRIMEIRO atributo repetido e descarta o resto em silêncio.
  **A escolha entre dois atributos volta para o Go**, onde `else if` é exclusão
  por construção.
- **Comentário NÃO vive na lista de atributos de um elemento.** `// ...` entre
  dois atributos derruba o parser, e a mensagem aponta OUTRA linha: solto dá
  `malformed open element`; dentro de um `for` ou `if` dá `for:`/`if: expected
  nodes, but none were found`, apontando a linha do BLOCO. Se um `for`/`if` que
  não mudou começou a reclamar de "expected nodes", procure o comentário entre
  atributos. O comentário vai ACIMA do elemento. **E o `go build` fica VERDE por
  cima do erro**, porque o `_templ.go` antigo continua no lugar: leia a saída do
  `templ generate`, não a do `go build`.
- **`@componente()` tem de COMEÇAR a linha.** No meio de um texto ele vira texto
  literal, e a página mostra `@tecla("⏎")` escrito na tela.
- **Valor CONSTANTE de atributo sai literal; só o DINÂMICO é escapado.** Uma aspa
  simples num literal fica aspa simples; vinda de variável vira `&#39;`. Importa
  quando se afirma HTML em teste.
- **NUNCA passe formatador em arquivo GERADO.** `goimports -w` num glob alcança
  os `_templ.go` e reorganiza os imports que o `templ generate` emite soltos: o
  conteúdo está em dia e a FORMA diverge, então regenerar e compilar não denuncia
  nada, e a CI reprova em "Fail if the generated templates were stale". Só o
  `git diff` DEPOIS de regenerar pega — de dentro de `engine-go/`:

  ```
  go tool templ generate && git diff --quiet -- '*_templ.go'
  ```

  Ele entra no roteiro de toda fatia que move arquivo de pacote, que é quando o
  `goimports` é preciso. `_templ.go` só se corrige regerando.
- **Regenerar não basta: o servidor precisa REINICIAR.** O `go run ./cmd/api`
  compila uma vez; depois do `templ generate` o processo continua servindo o HTML
  antigo, e o que se medir nele é a página velha.
- **Classe nova exige `scripts/build-css.sh`.** Classe que não passou pelo
  scanner não existe na folha, e o elemento aparece sem estilo em vez de dar
  erro. A detecção automática do Tailwind v4 varre a árvore por conta própria
  (respeitando o `.gitignore`), e os `@source` positivos do
  `serve/web/assets/app.src.css` não a limitam. **O que ela alcança entra na
  folha — prosa de `.md`, docstring e identificador de Go incluídos**; o que o
  `app.src.css` exclui com `@source not` (a saída `static/`, `infra/`, `cmd/`,
  `domain/catalog/transcript/`) não entra. Daí:
  - **compile por ÚLTIMO**: **código → prosa → `build-css.sh` → `git add`**. Vale
    também quando a mudança APAGA classe: o artefato guarda o estado de antes, e
    a CI reprova em "Fail if the stylesheet was stale";
  - **rode o `build-css.sh` mesmo quando a mudança não é de tela** — um
    identificador que coincide com o nome de uma utilidade, numa pasta que o
    scanner alcança, acrescenta a utilidade à folha;
  - **o comentário que explica esse risco não escreve o token** em minúsculas:
    nomeie o mecanismo, não a utilidade.
- **O `datastar.js` é VENDORIZADO, e quem o sobe é `scripts/vendor-datastar.sh`.**
  Ele não passa pelo Vite — o `serve/web/ui/layout.templ` o referencia com um
  `<script>` à mão —, então não há dependência no `package.json` para subir. O
  script fixa a versão numa linha, baixa da tag e reprova se o cabeçalho do
  arquivo disser outra coisa. Ele fica FORA do `prebuild`: subir a versão é ato
  deliberado, porque o reconciliador é o que mantém as cenas vivas. Depois de
  subir, o Playwright INTEIRO — o guarda que responde pelo morph é `ferir duas
  vezes seguidas não apaga a piscada da primeira`.
- **Regra da casa que precisa GANHAR de um utilitário mora em `@layer utilities`,
  não em `components`.** No Tailwind v4 a CAMADA decide antes da especificidade:
  `utilities` vence `components` mesmo contra um seletor de duas classes. Regra
  descartada pode parecer regra aplicada com cálculo errado. E tirar o
  utilitário do elemento troca o defeito de lado: quem tem de mudar é a camada.
- **`inset-0` põe `left: 0`, e `left` ganha de `right`.** Com os dois lados
  definidos e a largura fixa, o navegador resolve pelo `left`. Toda regra que
  ancora à direita sobre uma classe base com `inset` escreve `left: auto` junto —
  e o comentário fica, porque a linha parece redundante.
- **TOKEN inventado não vira regra, nem com o script rodado.** O Tailwind não
  emite regra para o que não está na paleta, e o elemento fica com a cor HERDADA.
  A paleta mora no `@theme` do `serve/web/assets/src/index.css`; confira lá antes
  de inventar um nome. O `TestEveryHouseTintExistsInTheStylesheet` cobra cada
  token — da paleta da casa e dos papéis semânticos (`destructive`, `primary`,
  `muted`, …, listados em `asPaletasDaCasa`) — contra a folha compilada. Ele
  varre o DIRETÓRIO, não um padrão de nome, e tem piso de denominador que SOBE
  quando a varredura cresce: um piso que o conjunto antigo satisfazia não
  denunciaria a volta dele.
- **A CLASSE DE ESCOPO não é uma tinta, e o guarda de tinta não a vê.**
  `scene-grimorio` é a condição para os tokens existirem — o `@custom-variant
  dark` do `index.css` é `&:is(.dark *, .scene-grimorio, …)`; sem ela o app
  inteiro sai sem cor. O `TestEveryScopeClassExistsInTheStylesheet` cobre, e lê só
  o que está dentro de `class=` e de `templ.KV("…")` — valores de `data-slot`
  como `scene-title` não são classe, e a casca escolhe o escopo (`scene-ds`) por
  condição.

  > **Renome em massa:** `\bnome\b` alcança muito mais que identificador Go — o
  > hífen é fronteira de palavra, então a classe composta, a ROTA e o nome de
  > arquivo vão junto. Renome de símbolo Go se faz com `gopls rename`, que é
  > semântico; troca textual fica só para o que o `gopls` não alcança — o
  > componente `templ` —, e aí o alvo tem de ser um nome COMPOSTO e único.

## Como uma CENA é construída

**O formato é uma PORTA declarada pela cena.** A interface `Deps` mora em
`serve/web/<cena>/deps.go`, e o `*api.Server` a cumpre. Quem escolhe o que
atravessa a fronteira é o CONSUMIDOR, não o objeto que tem tudo. O `api` monta
com uma linha por cena (`serve/api/web_router.go`) —
`campaigns.Routes(r, campaigns.New(s.campaignsHost(), …))` —, e é nessa linha
que o compilador cobra quando a porta deixa de ser cumprida.

**O CASO DE USO não entra pela porta: ele entra por PARÂMETRO.** Uma porta
existe para desviar de um ciclo — a cena precisa do `api`, que importa a cena —,
e o `app/` está ABAIXO das duas: não há ciclo, então não há interface a
declarar. A assinatura do `New` diz o que a cena faz:

```go
table.New(s.tableHost(), s.sessionLifecycle(), s.tableMemory, s.restParty(), s.initiativeQueue(), …)
sheetui.New(s.sheetHost(), s.characterPlays())
campaigns.New(s.campaignsHost(), s.sessionAccess(), s.campaignDirectory(), …)
door.New(s.doorHost(), s.accountGate(), s.accountResets())
```

Uma entrada que vira caso de uso SAI da `Deps` — ela não ganha um adaptador
novo, e a porta encolhe.

**Estado que a CENA GUARDA e outro alcança nasce no HOSPEDEIRO, não dentro do
`New`.** A Mesa guarda mapas por `(sessão, pessoa)` — a lente e a aba de cada
um —, e quando a sessão é APAGADA quem os esvazia é o ciclo da sessão, montado
ANTES da cena: ele não alcança o que ela criou por dentro. Montado no hospedeiro
(`s.tableMemory`), o mesmo objeto vai para os dois. O sinal de que é este caso:
"o fim de X tem de alcançar isto".

**Recusa que vira sentinela do `app/` deixa de precisar de um tipo para
viajar.** A cena a lê direto com `errors.Is` (`campaign.ErrNoSuchCampaign`), sem
um enum do meio que o hospedeiro traduzisse.

Cada `serve/web/*` tem um `boundary_test.go` com uma lista de imports PERMITIDOS,
que recusa o hospedeiro. Ele não existe para pegar o ciclo — esse o compilador
já pega —, existe para pegar o atalho: um import DIRETO de `domain/catalog`
contornando a camada tipada do `book`, que não é ciclo, não é erro, e é a divisão
vazando por baixo.

### As regras da porta

- **Ela pede o MENOR tipo que resolve.** `CurrentUserID(r) int64`, e não o
  usuário inteiro: uma porta que devolve tipo do hospedeiro não é porta, é o
  hospedeiro com outro nome.
- **Ela pede a PERGUNTA, não o objeto** — `RequesterIsAdmin(r) bool`, em vez de
  entregar o que permitiria decidir.
- **Quem escolhe a FRASE é a cena, sempre.** Quem CLASSIFICA depende de onde o
  sentinela mora: enquanto ele for valor do `api`, a cena não o alcança e o
  hospedeiro classifica; quando a regra desce para o `app/`, a cena lê o
  sentinela com `errors.Is` e o classificador some junto com o tipo que ele
  devolvia. O texto da tela continua sendo da cena.
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
**o menor não é o mais estreito, é o que não se repete**: uma cópia curta
declarada no comentário dela custa menos que um método a mais na porta.

**A cena sai; a bancada que a exercita fica.** Mover a cena não move os testes de
integração dela — eles batem no roteador de verdade, que continua no `api`.

## Código órfão, bancada mentirosa, e raciocínio envelhecido

Três famílias que se confundem porque as três produzem VERDE sobre nada.

### Órfão é dívida OU capacidade esperando gesto

**O que separa as duas não é a contagem de chamadores — é ler o que a coisa
FAZ.** Uma camada inteira no ar, com teste e sem rota chegando nela, é
funcionalidade perdida, não lixo.

E a outra metade: **a capacidade esperando gesto pode NÃO servir ao gesto quando
ele chegar.** Guardar código por "vai ser útil" é apostar no desenho que você
ainda não fez.

### A bancada que escreve o que a produção não escreve

**Código que ninguém alcança e teste que arranja o próprio estado são a mesma
doença**: os dois parecem cobertura.

- **Arranjo que usa uma porta que nenhum caminho de produção usa**: o verde é
  sobre o arranjo.
- **Bancada que escreve uma coluna que a produção nunca escreve**: o verde é
  sobre a bancada.

O controle é o mesmo nos dois casos: **monte o estado pelo caminho de
produção**, e não pela porta mais curta que chega ao mesmo dado.

### O RACIOCÍNIO guardado envelhece como o código guardado

Docstring envelhece como `.md`, e ninguém relê a de função que não vai tocar —
ver "Comentários" no [CLAUDE.md da raiz](../CLAUDE.md); a sonda que caça isso
precisa do controle positivo de "O INSTRUMENTO MENTE COM CARA DE RESULTADO".

### Guarda de leiaute precisa do ESTADO, não só da cena

Visitar a cena não é visitar o estado dela — ver "Um guarda só mede o que ele
VISITA" no [CLAUDE.md da raiz](../CLAUDE.md).

> **Esconder texto por CSS o tira da árvore de acessibilidade**: um rótulo
> escondido por `display:none` não é lido por ninguém — nem pelo leitor de tela,
> nem pelo `getByLabel`.

## Apagar código morto: as armadilhas do RECORTE

Quantos métodos exportados o `*Server` ainda tem se pergunta ao código:

```
grep -c "^func (s \*Server) [A-Z]" serve/api/*.go
```

**O compilador pega três das quatro armadilhas**, e a quarta é a que importa:

- Recortar função de UMA LINHA procurando `}` na coluna zero come as vizinhas.
- Receptor colide com variável existente no destino.
- `s.` dentro de uma string de formato vira recorte errado.
- **E a que ele NÃO pega: a regra que só aquele teste guardava.** Método sem
  chamador não quebra compilação, e o teste que o exercitava some junto com ele.
  Antes de apagar, pergunte o que o teste afirmava — não se alguém chama.

**Cena se testa pelo `WebRouter()`**, não pelo `Router()` da API JSON. Guarda
que varre rotas morre com o terreno dele; confira se a invariante mudou de casa
(a escrita de ficha por estranho, por exemplo, é do
`TestNoSheetWriteAcceptsAStranger`).

## Os pacotes que saíram do `api`

`campaign`, `account`, `search`, `book`, `sheet` e `creature` são pacotes do
`domain/`, e `events` da `infra/`. Três regras para a próxima extração:

**1. Função pura copiada por causa de fronteira é lugar de defeito
silencioso.**

> **Quando copiar, copie o CORPO do original — nunca uma reescrita de memória.**
> E prefira o pacote: ele apaga a cópia E a razão de haver duas.

**2. O guarda de fronteira que mais importa é o do pacote MAIS IMPORTADO.** No
dia em que o `book` alcançar o `api`, toda cena que o importa alcança HTTP de
graça — **com o guarda de cada uma VERDE, porque cada guarda só olha os imports
dele.** Vale igual para `sheet`, `creature` e `events`.

**3. A regra de PRODUTO não desce para a infraestrutura.** "A senha tem ao menos
8 caracteres" é regra, e por isso o `account` mora no `domain/`. Uma regra com
consumidor de TELA e de API tem a frase morando COM ela, em pt-BR e uma só: duas
frases para uma regra é o que quebra quando alguém mexe no limite.

**O critério de mover é a DEPENDÊNCIA, medida e não suposta:** "isto carrega
HTTP?" se responde nos imports do arquivo, não no assunto dele.

## `web/ui`: o kit de apresentação, e o que ele NÃO pode saber

O botão, o campo, a moldura, o rótulo de seção, a caixa rolável, o ícone e a
CASCA moram em `serve/web/ui`, e é isso que permite cada cena ser um pacote.

**A linha divisória não é tamanho, é DEPENDÊNCIA: o kit não importa catálogo.**
O que desenha a partir do catálogo — o livro, o verbete, o buscador — mora em
pacote próprio (`serve/web/bookui`, `serve/web/finder`).

**A casca RECEBE o que não pode conhecer**, como campo de `ui.Page`:

- `Asset func(string) string` — o endereço versionado dos estáticos, que são
  `go:embed` de outro pacote;
- `Overlays []templ.Component` — o livro, o verbete e o buscador, que leem
  catálogo. A casca só reserva o lugar;
- `ColorMode` — o claro ou escuro que a pessoa escolheu, lido do cookie
  `color_mode`. A casca o escreve no `<html data-mode>`, e não sabe de cookie.

Quem preenche é o `WritePage` do `serve/api/scene_core.go`, o ÚNICO lugar que
monta uma página. Não ponha esses campos em cada `ui.Page{…}`.

**Renomear palavra COMUM** (`campo`, `tamanho`, `layout`) pede um renomeador que
pule comentário e string, com o compilador como rede para o resto. O que o
compilador não pega é o identificador hifenizado (`data-nav-layout`): esse se
acha com `grep` depois.

### O `extra` do botão NÃO ganha da base, e quem decide é a FOLHA

`ButtonClasses(v, t, extra)` concatena, e **concatenar não resolve conflito**: o
`Join` não é `tailwind-merge`. Duas classes do mesmo eixo na mesma tag não se
somam — vence a que a folha compilada declara por ÚLTIMO. Um `extra` cuja
utilidade venha ANTES da base perde, e perde calado.

**Nenhum eixo mora na BASE se um TAMANHO pode querer outro valor.** A fonte e o
`gap` moram no mapa de tamanhos, onde não há duas classes para disputar; na base
fica o que nenhum tamanho contradiz — forma, foco, transição, desabilitado. E
um hover que precisa de outra cor vira VARIANTE (`VariantGhostDanger`), não
`extra`.

Para conferir um conflito: `grep -bo '\.classe'` na folha compilada, dos dois
lados. A ordem do `class=` não decide nada — o navegador a ignora.

## Dois sistemas de desenho, até a última tela migrar

A transição é POR ESCOPO: a cena migrada veste o sistema **Tempestade** inteiro,
e a que não migrou continua no **grimório**. Quem escolhe é a cena, com
`ui.Page.System`; o zero do tipo é o grimório, para nenhuma tela mudar sem
alguém pedir. O que cada palavra quer dizer está na seção H do
[GLOSSARY](../GLOSSARY.md).

- **Os tokens do Tempestade são `--ds-*` e só existem dentro de `.scene-ds`**
  (`serve/web/assets/src/ds.css`). Nada é declarado na raiz: um token com o nome
  de um do grimório repintaria as telas que ainda não migraram.
- **Cada cor é UM token com os dois modos** (`light-dark()`), e o modo é o
  `color-scheme` do elemento que USA o token. Por isso a folha viva desenha as
  duas metades lado a lado, e o modo escolhido é um atributo no `<html>`, não um
  segundo bloco de tokens.
- **A tela não escolhe cor.** Ela se monta com as peças de `serve/web/ds`, que
  escrevem a classe e a variante; a tela escreve leiaute (`grid`, `gap`,
  `px-4`), e a escala de espaço do Tailwind já é a do sistema (base 4). Peça
  nova entra no kit e na `/sistema` ANTES de entrar numa tela.
- **Os pisos do Tempestade são outros**: 12px de texto e 44×44 de alvo no toque.
  Quem mede é o e2e (`support/ds-floors.ts`, no laço do `appearance.spec.ts`),
  só na cena com `.scene-ds`; o caso anota qual sistema a cena veste, e não
  existe lista de cenas migradas para alguém esquecer. O guarda da Cinzel mede só
  a outra metade.
- **Medidor que mire um escopo pelo nome mira os DOIS** até o grimório sair. Um
  que mira só `.scene-grimorio`, só o `index.css` ou só o `class="…"` passa
  VERDE sobre a cena nova sem medi-la.
- **Tela grande migra em PARALELO, no mesmo endereço.** A versão antiga fica
  intacta até a última parte migrar; parte ainda não migrada desenha um vazio
  que leva à antiga — nunca o grimório dentro da cena Tempestade. Quando a
  última parte migra, a antiga é APAGADA, e os testes que liam o desenho dela são
  portados antes.
- **O fim da migração é apagar**, não converter: o `.scene-grimorio`, os
  tokens `grimorio-*`, a Cinzel, a Liberation, a JetBrains Mono e a `/grimorio`
  saem de uma vez, depois da última tela.

## O anel de foco é UMA receita, e o repouso dela existe para não animar

A regra de foco de cada sistema mora na folha dele — `index.css` para
`.scene-grimorio`, `ds.css` para `.scene-ds` — e **não é layerada**, então ganha
de todo utilitário do Tailwind, que é layerado. `focus-visible:outline-*`
escrito à mão numa `class=` não faz efeito nenhum dentro de uma cena, e quem
impede a próxima cópia é o `TestNoHandwrittenFocusRing`.

O que MUDA o pixel é o REPOUSO. O `@layer base` traz um `* { outline-color }`
do shadcn, e o `transition-colors`/`transition-all` do Tailwind v4 **incluem
`outline-*`**. Nada daquilo pinta em repouso (`outline-style: none`), mas é o
ponto de PARTIDA da transição: o anel passa a ser ALCANÇADO em 150ms em vez de
desenhado, e quem tabula na autorrepetição do teclado nunca o vê inteiro.

**O conserto é escrever o repouso igual ao destino**, no mesmo `*`: sem
diferença, não há o que interpolar. A duplicação da largura e do afastamento com
a regra de foco é deliberada, e quem cobra que os dois lados não se separem é o
`e2e/tests/support/focus.ts` (`expectOneFocusRing`).

- **Para medir, pergunte ao navegador se há transição** (`getAnimations()`) —
  ver "O INSTRUMENTO MENTE COM CARA DE RESULTADO" no
  [CLAUDE.md da raiz](../CLAUDE.md).
- **A carta de rádio é a única exceção legítima, e ela é do RÓTULO.** A forja e
  o "entrar na mesa" escondem o `<input>` com `sr-only` e desenham o realce no
  `<label>` (`has-[:focus-visible]:outline-*` na forja,
  `.ds-hero-pick:has(:focus-visible)` no `ds.css`). A regra global casa
  `:is(a, button, input, …)` e não alcança; uma `label:has(:focus-visible)` no
  lugar dela desenharia DOIS anéis concêntricos em todo rótulo que envolve um
  campo VISÍVEL.
- **Guarda que aproxima um seletor mede outro seletor.** A isenção do anel tem
  de COPIAR o seletor da folha (`[data-nav-region] :is(a, button,
  [data-nav-item])`), e não "tudo dentro de `[data-nav-region]`".

## As notas numa janela própria, e o pacto que as torna únicas

`GET /campanhas/{campanha}/sessoes/{sessao}/notas` desenha o MESMO painel das
notas, sem o mapa em volta.

**Ela é CENA, e não painel mudado de lugar**: o nó adotado por outro documento
perde o Datastar (ver "O nó que muda de DOCUMENTO", abaixo). O corpo é reusado
(`notesBody`, `modesRange`, `autosaveState`); o que a cena NÃO tem — alternador
de flutuar, divisa de largura, ✕ — está decidido no cabeçalho do
`notes_window.templ`.

**As duas não podem estar abertas ao mesmo tempo, e a razão é o BANCO.** As duas
escrevem `sessions.notes` com autosave de 1,2s, e as notas não são região do
stream — são de um leitor só. Com as duas no ar, quem salva por último apaga o
parágrafo do outro, com as DUAS faixas dizendo "Salvo".

O pacto é `localStorage` mais o evento `storage`, não um canal de difusão: o
evento chega a toda janela da origem sem ninguém segurar um objeto vivo. A chave
guarda o ID DA SESSÃO, porque a janela da sessão 4 não tem o que dizer sobre a
7.

- **Quem fecha a coluna é o ANÚNCIO da janela, não o clique.** Com o pop-up
  bloqueado, nada foi tomado e a coluna fica. O bloqueio é dito em PALAVRAS —
  `window.open` devolvendo nulo é a única pista.
- **`pagehide` e não `beforeunload`**: aquele pede confirmação em alguns
  caminhos, e `unload` é pulado quando a página vai para o cache de ida e volta.
- **A chave pode ficar PRESA** se a janela morrer sem `pagehide`. O conserto é o
  próprio gesto: o botão chama `window.open` com o mesmo NOME, que acha a janela
  viva ou abre outra.

**O `window.open` é endereço que a cena escreve**, e o
`TestEveryAddressAPostWritesExistsInTheRouter` o confere como GET — porque é
navegação. Perguntar ao chi por um POST no mesmo caminho responderia "existe"
pela rota de SALVAR, e o guarda ficaria verde sobre uma página que não existe.

## Onde a coordenada de um gesto do tabuleiro viaja

**No CORPO, e não no caminho.** O `@post` do Datastar aceita `{payload: …}`,
calculado no instante do clique — não vira sinal, que é estado compartilhado e
viajaria em toda requisição. O formato é um só para o tabuleiro inteiro:

```json
{"from":{"x":2,"y":2},"to":{"x":8,"y":5}}        // o traço e o retângulo
{"kind":"dificil","erase":true,"from":…,"to":…}  // com a espécie do pincel
{"shape":"esfera","size":"6","from":…,"to":…}    // o gabarito: origem e mira
{"from":{"x":7,"y":3}}                           // um lugar só: parada, marcador
{"delta":{"x":3,"y":-2},"marked_tokens":"a,b"}   // o arrasto do grupo marcado
```

As chaves são INGLESAS porque campo JSON é fronteira (ver "Idioma" no
[CLAUDE.md da raiz](../CLAUDE.md)). Um tipo só — o `strokeBody` — para os
gestos de pontos: um formato por gesto é como nasce a terceira grafia do mesmo
par de números.

### O que o corpo quebrado produz, e as DUAS famílias de recusa

O corpo quebrado de verdade é `{"from":{"X":undefined}}`: um sinal indefinido na
expressão do `payload` manda `undefined`, que não é JSON. O servidor não pode
decidir sozinho que o gesto foi na origem.

**As rotas que leem pontos têm DOIS contratos de recusa:**

- as que respondem **só sinais** (`marcar-area`, `gabarito`, `regua`) recusam em
  **400**, porque não há cena para redesenhar;
- as que são **comando** (terreno, retângulo, peça avulsa, grupo) recusam em
  **200** com a frase no `$command_error` — porque o Datastar DESCARTA o remendo
  de resposta não-2xx.

O que as duas têm em comum é o que importa à mesa: **a frase CHEGA** — e é a
frase que o `TestEveryGestureThatReadsPointsRefusesABrokenBody` prende, não o
status.

> **Corpo VAZIO não é recusa, e isso é desenho.** `{}` decodifica para (0,0) em
> silêncio; prender isso exigiria ponteiro em todo campo de coordenada.
>
> Para quem escreve teste: **(0,0) é o VALOR-ZERO, então caso ancorado na origem
> não detecta um `from` que parou de ser lido.** Todo caso sai da origem de
> propósito, e vários afirmam que (0,0) NÃO foi tocado.

### Quando o corpo JÁ É o formulário, o payload lista os sinais

O `payload` SUBSTITUI os sinais no corpo — mas pode CARREGÁ-los, nomeando-os:

```js
@post('…/pecas/nova', {payload: {from: {x: cx, y: cy},
  new_token_name: $new_token_name, new_token_size: $new_token_size}})
```

**O corpo não se lê duas vezes**: o `ReadSignals` copia `r.Body` inteiro, e um
segundo leitor pega vazio. É um struct só, lido uma vez.

**O preço é uma grafia a mais** de cada sinal, que um `grep` de `$nome` não
distingue de leitura qualquer. Quem paga é o
`TestEveryPayloadKeyMatchesTheSignalItReads`: chave que é ela mesma nome de
sinal tem de carregar aquele sinal (`new_token_size: $new_token_look` reprova;
`kind: $tool` passa, porque `kind` não é sinal), e toda chave tem de achar uma
tag `json:"chave"` do outro lado. Chave renomeada de um lado só chega e cai no
chão: 200 com o valor-zero.

**E a CAIXA da chave conta.** O `encoding/json` casa campo sem diferenciar caixa
**só quando não há correspondência exata** — então `{X: cx}` contra `json:"x"`
funciona por ACIDENTE, até o struct ganhar um campo que case exatamente. A
grafia do fio é minúscula (`x`, `y`), a mesma gravada em `campaign_places` e
`open_boards`; o guarda exige a grafia exata, e **a tag GANHA do nome do
campo**.

> **Aqui o navegador NÃO é testemunha**: com a caixa errada, o app FUNCIONA e o
> Playwright passaria junto. **Defeito que só existe como risco latente não tem
> testemunha em tempo de execução**, e quem o prende é guarda de TEXTO.

### DESLOCAMENTO é coordenada, e o guarda não sabia disso

**O que faz de um número "coordenada" não é ele ser absoluto — é ele vir do
PONTEIRO.** O deslocamento do arrasto (`delta`) viaja no corpo como qualquer
ponto, porque no caminho ele obrigaria a concatenar o endereço.

#### E o guarda foi REFEITO, porque ele era duas coisas estreitas

O `TestNoRouteCarriesACoordinateInThePath` pergunta ao ROTEADOR, não ao
código-fonte: o `chi.Walk` devolve o padrão já montado — o `base :=` juntado, o
pai do `r.Route` concatenado com o filho, o registro em duas linhas lido como
um —, e não há linha para ler errado.

E ele é lista de PERMITIDOS: todo parâmetro de caminho da árvore está em
`serve/api/testdata/route_params.txt`, e **parâmetro novo REPROVA até alguém
escrever a linha** — que é o ato de declarar que ele não vem do ponteiro. O
`TestNoAllowedRouteParamIsStale` cobra o lado inverso: linha que nenhuma rota
usa mais reprova, porque permissão pendurada aprova em silêncio o próximo nome
igual.

### O PASSO da ficha FICA no caminho, e a razão não é a que estava escrita

`/personagens/{id}/vitais/{qual}/{passo}`, `/nivel/{classe}/{passo}` e
`/poderes/classe/{classe}/{escolha}/{valor}` continuam com o dado no caminho, de
propósito.

A razão NÃO é "quatro botões disputariam um sinal": o `payload` é calculado por
chamada, e cada botão escreve o seu literal. **O obstáculo é o REDESENHO.** Toda
mutação da ficha passa pelo `sheetCommand`, que lê os sinais UMA vez e os
entrega ao `s.Load(…)` que redesenha a cena — buscas, filtros, aumentos,
construtor de item. Como o `payload` SUBSTITUI os sinais, um `{step: -1}`
chegaria com o resto zerado, e a tela voltaria sem o filtro que a pessoa estava
usando.

**A divisão que vale não é "caminho para o que o gesto decidiu, corpo para o
estado"**: coordenada vem do ponteiro e obrigaria a concatenar o endereço; passo
é um literal que o `templ` escreve na renderização, num endereço constante por
botão.

## Datastar: armadilhas que não deixam erro para trás

Nenhuma delas escreve no console (no máximo uma linha que ninguém olha), e o
sintoma de cada uma aponta para o lugar errado.

### `data-show` esconde TARDE: o nó pinta antes de o Datastar chegar

O `data-show` só é avaliado quando o runtime carrega. Até lá o nó pinta com as
classes que tem — um diálogo com `fixed inset-0 bg-black/60` cobre a janela e
depois some.

**O conserto é `style="display:none"` estático ao lado do `data-show`**: lido na
análise do HTML, antes de qualquer script, e trocado por vazio quando o
Datastar mostra. Não confundir com o `data-attr:style` da armadilha seguinte,
que reescreve o atributo a cada avaliação.

O `TestNoDataShowNodeIsBornVisible` varre todo `.templ`: sobreposição e
conteúdo escondido por sinal puro (`data-show="$x"` sem negação nem operador
quer dizer "escondido até ficar verdadeiro"). Ele erra para o lado seguro —
aparecer um quadro depois é melhor que pintar e sumir.

### `ReadSignals` vem ANTES do `NewSSE`, e a ordem errada passa VERDE no teste

O `NewSSE` ASSUME a resposta e fecha o corpo do pedido; um `ReadSignals` depois
dele chega sem sinal nenhum.

**O `httptest.NewRequest` não reproduz esse ciclo de vida**, então o teste de
handler passa VERDE e o defeito só aparece no navegador. Num `GET` não morde (o
corpo é vazio dos dois jeitos): o defeito nasce no dia em que alguém troca o
gesto para `POST`.

E **o corpo do pedido se lê UMA vez** — um segundo leitor pega vazio, sem erro.

### `data-show` + `data-attr:style` no MESMO nó CONGELA a aba

O `data-show` escreve `el.style.display`; o `data-attr:style` reescreve o
atributo `style` INTEIRO, apagando o `display` — que o outro põe de novo. O
renderizador entra em laço, e a aba para de responder a TUDO, sem console e sem
conseguir navegar para fora.

**Quem ESCONDE é um nó, quem POSICIONA é outro.** O
`TestNoNodeHasDataShowAndDataAttrStyleTogether` recusa a combinação no HTML
servido da mesa.

### O sinal é um PROXY: ler um índice que não existe o CRIA

`$lista[0]` não é "o primeiro item": o Datastar registra o caminho e o cria
vazio. `$ruler_points[i]` sobre índices reservados encheu o sinal de strings
vazias entre os pontos de verdade.

**Guardar o sinal numa constante NÃO resolve** (a constante continua sendo o
proxy). O que resolve é COPIAR: `const p = [...$ruler_points]`. O
`TestNoExpressionIndexesTheListSignal` afirma a regra pelo que PODE vir depois
de `$lista` — `=` ou `]` —, e não por uma forma errada conhecida.

### Sequência de comandos NÃO cabe num ternário

`evt.shiftKey ? (stmt; stmt) : (stmt; stmt)` é erro de SINTAXE. O Datastar
engole o erro e o handler inteiro vira nada — o gesto que já funcionava para
junto. Use `if (…) { … } else { … }`.

E `setPointerCapture` vai por ÚLTIMO na expressão: ele LANÇA quando o ponteiro
não está mais ativo, e no meio engole o resto do gesto.

### O servidor escrevendo num SINAL pelo stream: uma vez, nunca a cada quadro

O stream da mesa remenda HTML, e escreve num SINAL (`PatchSignals`) onde a
decisão é do navegador: o "mostrar à mesa" leva quem foi puxado para a
superfície do tabuleiro (`surface`), e a ficha mudada avisa por `sheet_version`
(abaixo).

**A armadilha é a cadência.** O estado do puxão vive até a pessoa escolher
outra aba, então a leitura ingênua manda o sinal em TODO quadro — e quem tenta
voltar para a Mesa é devolvido ao mapa um segundo depois, para sempre.

A memória do que já foi empurrado é da CONEXÃO (uma variável do laço do
stream), não do servidor: duas abas da mesma pessoa merecem o empurrão cada uma.
**Remendo de HTML é idempotente, remendo de sinal não é.**

### O `@post` que redesenha a CENA precisa carregar o estado que está na URL

O comando responde com a cena inteira, e o handler descobre o que desenhar
lendo a própria requisição. O que está na URL da PÁGINA (`?tab=` na ficha,
`?aba=` e `?entrada=` no acervo do mestre) **não vai junto**: o `@post` manda o
endereço escrito nele, e mais nada. Na ficha, mexer no PV com a Mochila aberta
devolvia a cena na PRIMEIRA aba, sem nada falhar.

**O servidor escreve o `?` no comando**, porque é ele quem sabe o estado ao
renderizar o botão: uma função só monta todo `@post` da ficha (`sheetPost`), e o
`TestNoSheetCommandLosesTheTab` lê o HTML de cada aba e falha no comando que
sair sem ele. Sinal do cliente também resolveria, e some no F5 — que é o que o
endereço existe para sobreviver.

### Dois pedidos de UM gesto: quem CHEGA por último manda

O clique do mouse **também foca**. Um nó que pede ao servidor no foco e no
clique manda dois pedidos por um gesto, e se os dois remendam a mesma cena eles
disputam os sinais — a ordem de CHEGADA não é a de saída. Atravessa a rede,
então some na bancada rápida e aparece no CI.

**O remédio é não mandar o segundo pedido:** pedido disparado por FOCO é
afordância de TECLADO, e pede `:focus-visible`
(`el.matches(':focus-visible') && (…)`). O clique dá `false`; o Tab e o foco
programático do driver de setas dão `true`.

O `TestNoFocusAsksTheServerWithoutAKeyboardGuard` CAMINHA a árvore inteira de
`.templ` (enumerar cena deixaria a próxima sem medição) e tem piso de arquivos
VISITADOS: mudar o guarda de pacote não pode encolher o que ele varre sem
reprovar.

### Resposta que não é 2xx: o remendo é DESCARTADO e a recusa some

O Datastar não aplica o remendo de resposta de erro. `http.Error(w, msg, 400)` —
o certo numa API JSON — deixa a tela EXATAMENTE como estava, e a única marca é
uma linha no console do navegador.

**Numa cena servida, a recusa é CONTEÚDO.** Ela volta 200 com a cena
redesenhada mais a frase num `role="alert"` (`sheetCommand` +
`sheetui.View.Refusal`). Como o status deixou de distinguir "gravou" de
"recusou", **os testes afirmam a FRASE** (`sceneRefusal`). A API JSON continua
com os status dela; quem desenha página responde página.

**O caso EXTREMO é o 404: um endereço que não existe.** O handler nunca roda, e
o gesto simplesmente não acontece. Endereço montado em metades (`"initiative/"+
route` no helper, o verbo no chamador) é invisível ao `grep`: **o endereço só
existe RESOLVIDO**.

Quem cobra é o `TestEveryAddressAPostWritesExistsInTheRouter`, que RENDERIZA:
tira todo `@post`/`@get`/`window.open` do HTML das cenas que visita
(`scenesThatWriteAddresses`, cujo cabeçalho declara a enumeração como o limite
dele) e pergunta ao chi com `Match` se a rota existe — `Match` e não um pedido,
porque um 404 do HANDLER ("personagem não existe") pareceria rota faltando.

Para quem escrever o próximo guarda de cena:

- **Ele precisa do ESTADO vivo.** O `‹` da mesa nasce `disabled` quando
  `CanAdvance` é falso (`st.CountsRounds() && len(st.Initiative) > 0`), e numa
  bancada recém-montada o endereço morto não está no HTML.
- **Expressão que o extrator não souber resolver REPROVA**, em vez de sair da
  conta em silêncio.

### `contentType: 'form'` valida o formulário ANTES de mandar

`@post(url, {contentType: 'form'})` manda o `<form>` mais próximo em vez dos
sinais: os controles já são o estado, e uma cópia num `data-signals` só cria a
pergunta de qual vale. A forja é assim — o `@post` que redesenha e o `submit`
que forja leem o MESMO formulário, pelo mesmo `r.ParseForm()`.

**A armadilha é o `if (!form.noValidate && !form.checkValidity())`** que o
cliente roda antes: com um campo `required` em branco ele chama
`reportValidity()` e **não manda o pedido**.

O conserto não é `novalidate`, que desligaria a validação do `submit` também: é
**o campo não ser `required` no HTML** e a recusa ser do servidor, que já é a
autoridade. **Num formulário que também é remendado, validação nativa de campo
briga com o redesenho.**

### `data-on-signal-patch` SEM filtro escuta o remendo dos OUTROS sinais

A ficha dentro da sessão não é região do stream (recomputá-la a cada tique seria
o gasto mais caro da página): o servidor manda um carimbo de uma linha
(`sheet_version`) e o CLIENTE repede a ficha, pelo `data-on-signal-patch`.

**Ele dispara em QUALQUER remendo de sinal**, e a cena tem outro (`surface`, o
puxão). Sem filtro, um puxão do mestre repediria a ficha. O filtro mora num
atributo IRMÃO:

```
data-on-signal-patch-filter="{include: /^sheet_version$/}"
data-on-signal-patch="… @get('/personagens/13?tab=' + $sheet_tab + '&embutida=1')"
```

- **O servidor manda UMA vez por mudança**, nunca por quadro: a memória do que
  já foi avisado é da CONEXÃO (`announceSheetChange`).
- **Quem diz "mudou" é um EVENTO**, não a comparação de um carimbo a cada tique:
  o stream só lê o carimbo quando o evento diz que a ficha mexeu (ver "O
  barramento de eventos").
- **O interesse é por PERSONAGEM, não por sessão.** A mesma ficha pode estar em
  duas mesas, e pendurar isso no `session.Store` obrigaria quem PUBLICA a saber
  em quais. Quem não tem ficha na mesa não pede o interesse.
- **Quem publica é o GATEWAY** (`characterChanged`), não cada comando: a linha
  esquecida numa mutação seria uma ficha que não atualiza só naquele gesto.
- **O servidor NÃO sabe a aba**: o stream abriu antes de qualquer clique. Quem a
  guarda é um sinal que o clique na aba escreve (`$sheet_tab`), e o repedido a
  concatena — senão volta a aba padrão.

### O morph REMOVE o nó que o seu JS pendurou na linha

**O reconciliador remove todo filho que não veio no HTML do servidor.** Um véu,
um brilho, um espaçador criado pelo JS dentro da subárvore remendada some — no
morph do próprio gesto (o `MutationObserver` roda como MICROTAREFA, no meio do
morph) ou no do remendo SEGUINTE, de qualquer gesto. Esperar um
`requestAnimationFrame` só ganha do primeiro. O `data-ignore-morph` do bundle
não socorre: exige o atributo nos DOIS lados, e o do servidor nunca o tem.

O caro é que a animação foi PEDIDA: um guarda que conte `el.animate` fica verde,
e a sonda que olhe o DOM no instante do `animate()` também.

**O nó mora FORA da subárvore que o remendo alcança** — `position:fixed` sobre o
retângulo do alvo, filho do `<body>` (`lib/turn-juice.ts`). O preço é a caixa
não seguir o alvo se ele andar durante a animação.

Animação que mexe no PRÓPRIO nó (escala, sombra, opacidade) não precisa disso:
o morph reusa o nó. E **o morph não toca atributo que já bate**, então um
observador de `aria-current` (ou de qualquer atributo de estado) só acorda quando
o valor MUDA — não escreva guarda contra repetição.

### O nó que muda de DOCUMENTO perde o Datastar, e continua desenhado

Uma janela própria (Document Picture-in-Picture, um `<iframe>`) pede o nó vivo
emprestado, e `appendChild` o leva inteiro — classes, texto, `data-*`. **O que
não vai é a reatividade**: o runtime mora no documento de origem, e o nó
adotado sai da árvore que ele observa. Sem exceção, sem console, tela perfeita.

**A janela recebe um DOCUMENTO**, com o runtime dela: superfície que vai para
uma janela própria é CENA com endereço. Ver "As notas numa janela própria".

## O evento de ponteiro SINTÉTICO destrói o que ele mede

`element.dispatchEvent(new PointerEvent(...))` com um `pointerId` inventado faz
o `setPointerCapture` lançar `NotFoundError`, e a exceção leva junto as escritas
de sinal do handler. O gesto parece MORTO quando está inteiro.

**Gesto de ponteiro se mede com entrada REAL.** O Playwright (`page.mouse.down`,
`page.keyboard.down('Shift')`) tem pointer de verdade; a automação do Chrome MCP
NÃO aplica `modifiers` ao arrasto (`shiftKey` chega `false`), então para
modificador o caminho é o Playwright.

## Remendo em nó COMPARTILHADO exige limpeza no gesto que troca de item

Uma cena em Datastar desenha **todos** os itens e alterna qual aparece com
`data-show`. Isso é seguro enquanto o conteúdo de cada item vier inteiro do
servidor — e deixa de ser no instante em que algo é escrito, DEPOIS da
renderização, num nó que os itens dividem. O diálogo de redefinição de senha é
UM só para todas as linhas de jogador; sem limpar, o link de um aparece sob o
nome do outro.

**Quem LIMPA é o gesto que TROCA de item, não o gesto que gera.** Quem gera não
sabe que vai haver um próximo; quem troca sabe que houve um anterior.

```
data-on:click="$target_id = el.dataset.id; $target_name = el.dataset.nome;
               $copied = '';
               document.getElementById('reset-link').innerHTML = '';
               $reset_dialog.showModal()"
```

A família é **estado de um item sobrevivendo à troca por outro**, e o
`data-show` a multiplica. A cada cena nova: existe nó compartilhado que recebe
escrita depois da renderização? Se existe, quem troca de item tem de apagá-lo.
