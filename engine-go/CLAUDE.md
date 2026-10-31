# Motor Go

Estende ou sobrepõe o [CLAUDE.md da raiz](../CLAUDE.md). `engine-go` é o app
inteiro, num processo só: API na :3001, motor de regras, CENAS em `.templ` com
Datastar, folha e ilhas de JS (`serve/web/assets/src`) e kit de apresentação
(`serve/web/ui`) — ver "O app fala HTTP".

## O mapa das pastas

```
engine-go/
├── cmd/        os binários: servidor (`api`), seed, gerador do oráculo, auditores
│               do catálogo (`audit/`) e quem grava nele; `internal/` é o comum
├── domain/     O QUE O JOGO É — nada aqui sabe de HTTP
│   ├── engine/   as regras do livro, PURAS (só a stdlib)
│   ├── ecs/      o núcleo de entidade-componente-sistema do motor (só a stdlib)
│   ├── catalog/  o livro transcrito (`transcript/`, em Go) e quem o serve
│   ├── book/     o catálogo tipado
│   ├── sheet/  board/  live/   o domínio COM estado
│   └── campaign/ account/ creature/ search/ markdown/
├── app/        O QUE UM GESTO FAZ, do pedido à gravação
│   ├── session/    o ciclo da sessão, a trava de acesso e o STORE da fila
│   ├── boards/     o store dos tabuleiros abertos, com as abas e os lugares
│   ├── initiative/ quem entra na fila, e com que números
│   ├── combat/     ataque e manobra: quem pode, o teste, a decisão e o dano
│   ├── character/  o herói: nascer (`Births`) e jogar (`Plays`)
│   ├── campaign/   a mesa: quem a vê, quem a abre, quem senta nela
│   ├── accounts/   a conta: entrar, cadastrar-se com convite, trocar a senha
│   └── rest/       o que expira e o que recupera quando a cena ou o dia acaba
├── serve/      O QUE RESPONDE HTTP
│   ├── api/      a RAIZ DE COMPOSIÇÃO: monta o roteador e cumpre as portas
│   └── web/      as cenas, uma por pacote, e o que dividem (`ui`, `ds`, `routes`)
│       └── assets/  o FRONT que não é `.templ` (folha, ilhas de JS, fontes,
│                    favicon) e o `go:embed` deles
├── infra/      O QUE NÃO É DOMÍNIO
│   ├── db/       migrações e as consultas do sqlc
│   ├── config/   o que o ambiente diz, lido no boot
│   ├── bookpdf/  a leitura do PDF do livro por coordenada, para os auditores
│   ├── httpio/   a borda HTTP: responder, ler corpo, comprimir
│   ├── wire/     a fronteira SEM transporte — é o que o domain alcança
│   ├── events/   o barramento tipado
│   └── secret/   o sorteio do token que alguém recebe por link
├── convention/ os guardas que não são de pacote nenhum
├── parity/     os oráculos de ficha
└── scripts/    o build da folha, das ilhas, dos ícones e do Datastar vendorizado
```

**A seta só aponta para BAIXO**: `serve` → `app` → `domain` + `infra`, e o
`infra` não conhece ninguém. Garantem os `boundary_test.go` de cada pacote e,
entre GRUPOS, o `TestNoLayerImportsUpwards`, que varre a árvore e não uma
lista. **Entrada nova numa lista de permitidos é decisão, argumentada na linha.**

### O `app/`, e o que ele NÃO é

**Entra aqui o gesto INTEIRO — quem pode, o que decide, o que grava.** Repasse
de uma linha não vira caso de uso por mudar de pasta.

- **O `domain/board` tem ZERO toques de persistência** (o `boundary_test.go`
  proíbe o `sqlcgen`); quem carrega, chama a regra e grava é o `app/boards`.
- **O que sobra em `domain/` é regra PÚBLICA**: as mutações dos stores
  (`AddToken`, `AdvanceTurn`, `PatchEntryVitals`…) recebem e devolvem estado,
  sem travar nem gravar.
- **As recusas são TIPADAS** (`ErrNotFound`, `ErrForbidden`, `ErrRefused`),
  nunca número de HTTP, que prenderia o caso de uso a um transporte.
- **A TRANSAÇÃO é daqui, sempre**: é o contorno do gesto (ou as campanhas mudam
  de dono e a conta some juntas, ou nada). O
  `TestNoPresentationLayerOpensATransaction` mantém o `serve/` sem `BeginTx`,
  com linha de base vazia.
- **Exceção declarada: o `character.Plays` não embrulha `app.ErrRefused`**,
  porque o embrulho põe "recusado pela regra" no `Error()` e a ficha mostra a
  recusa CRUA (o Datastar não desenha corpo de 4xx). Com um segundo transporte,
  o embrulho desce; número de HTTP na regra, nunca.

| onde procurar | a pasta |
|---|---|
| quanto de PV este herói tem | `domain/engine` |
| como a ficha é montada do banco | `domain/sheet` |
| o que a tela desenha | `serve/web/<cena>` |
| a folha, uma ilha de JS, uma fonte | `serve/web/assets` |
| uma peça do sistema Tempestade | `serve/web/ds` (e a folha viva em `/sistema`) |
| quem cumpre o que a cena pede | `serve/api` |
| onde o dado é gravado | `infra/db` |
| por que a suíte reprovou uma convenção | `convention` |

Os TESTES moram ao lado do código (caixa branca precisa do mesmo diretório).
**Cada cena é um pacote em `web/`**; o `api` guarda o roteador, as portas de
cada cena e as regras de tela nenhuma — ver "Como uma CENA é construída".

**No `serve/api`, o prefixo do arquivo diz o ADAPTADOR dono — um arquivo, um
dono**: `table_*` do `tableRules`, `sheet_*` do `sheetRules`, `campaign_*` e
`account_*` dos outros dois. O `TestEveryAdapterFileCarriesItsPrefix` lê o
RECEPTOR dos métodos e ignora, de propósito, o `*Server` (fiação) e os hosts de
cena (`tableHost`, `hubHost`…), que seguem a convenção de `*_deps.go` e podem
dividir arquivo com um adaptador.

**As cenas atendem na RAIZ**, e o `git grep` não mostra as consequências:

- **o `alvoOriginal` lê `RequestURI`**, porque o `URL.Path` descarta a QUERY;
- **endereço interno não soma prefixo**: `"/" + rota` vira `//livro`, e `//algo`
  é URL relativa a PROTOCOLO, lida como o HOST `algo`. Só teste de cena pega.

## Ambientes

`LoadConfig` lê `.env.<APP_ENV>` deste diretório, mas **o processo vence o
arquivo**. O `.env.development` é versionado; o `.env.production` não, e o molde
dele é o `.env.production.example`. **Configuração nova entra em
`infra/config/config.go` e nos dois versionados** — default só no Go ninguém
descobre — **e, se puder derrubar produção em silêncio** (chave de assinatura,
origem liberada), **em `Config.Validate`**, que roda antes de escutar.

## Regenerar oráculo é ato deliberado

Os JSONs em `parity/` fixam, por personagem, o PV, cada carta de ataque com o
breakdown e cada perícia com a composição — a rede de regressão da ficha:
```
cd engine-go && go run ./cmd/genoracle
```

**O diff de um oráculo é revisado contra o LIVRO, nunca aceito porque "o teste
ficou verde"**: o oráculo é o Go descrevendo o Go, e um bug no motor vira a nova
verdade em silêncio. Paridade prova concordância, nunca correção.

## Regras vêm do livro, com página

**Todo teste de regra cita a página do [Tormenta 20](/t20-book.pdf)** e, quando
existe, o **exemplo trabalhado** do livro (a Samira da p173), que separa
leituras possíveis da mesma frase. **Confira a citação antes de escrever**:
`página do PDF = página do livro + 6`. Citação "verificada" de cabeça já saiu
errada — ver [APRENDIZADOS](../APRENDIZADOS.md).

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
CENA. Ele passa pelo `requirePage`, que lê a sessão do COOKIE — sem token na
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

## A regra de EMPILHAMENTO mora onde o `targetKey` mora

O `TestNoEntityStacksWithItself` (`domain/engine/modifier_stacking_rules_test.go`)
prende a p226 — uma entidade não declara dois modificadores não-condicionais no
mesmo alvo e tipo — com o balde do `targetKey` real, lido no `Modifier` real
(passando pelo `UnmarshalJSON`). **Guarda que precisa de regra privada do motor
mora no motor**: `switch` espelhado noutro pacote diverge em silêncio.

## A MANOBRA é o primeiro teste OPOSTO do motor

O resto do combate mede um total contra um número parado; na manobra (p234) os
DOIS lados rolam, e o `ResolveManeuver` recebe dois `d20`. Duas regras da
página:

1. **a MARGEM viaja**, porque é regra — cinco ou mais dão efeito extra ao
   derrubar e ao desarmar, e o empurrar ganha 1,5m por cada 5;
2. **o defensor sempre rola LUTA**, mesmo com arma de disparo: a proibição de
   ataque à distância é de quem ATACA.

**A manobra divide o provisório com o golpe**: ela É ataque corpo a corpo, e
dois provisórios seriam duas verdades sobre a cena. `proposesAttack` e
`proposesManeuver` delegam ao mesmo `proposeStrike`: as conferências antes da
rolagem são as mesmas. **O BÔNUS conhece o LADO**: o `Desejo de Liberdade`
ajuda quem é agarrado, não quem agarra; o escopo está na chave do `targetKey`,
e um caso prende os dois lados.

## O GOLPE EM DOIS TEMPOS: o dano só existe depois da decisão do mestre

O crítico muda QUANTOS dados se rolam (p231), e quem diz se houve crítico é o
mestre ("O veredito é do mestre", no [CLAUDE.md da raiz](../CLAUDE.md)). Rolado
junto do d20, o dano sairia multiplicado pela SUGESTÃO do livro, e quem rola na
mão não saberia quantos dados pegar.

- **O teste**: o `JudgeAttack` devolve a rolagem, a Defesa e a sugestão
  (`Hit`, `Critical`), sem dano. O provisório leva a margem e o multiplicador da
  ARMA (`CritRange`, `CritMult`), que o botão "Crítico ×N" escreve.
- **A decisão** (`Ruling`: `miss`, `hit`, `critical`) é do mestre, UMA vez
  (`live.RuleAttack`). No acerto e no crítico ela já guarda o que rolar
  (`DamageToRoll`, do `engine.DamageToRoll`), na ordem em que os dados se leem:
  a arma primeiro, os extras depois.
- **O dano** é de quem atacou, ou do mestre por ele (`live.AttackAwaitingDamage`):
  o `RollAttackDamage` troca a sugestão pela decisão (`applyTheRuling`) e rola.
  Na mão, o `DiceRolledByHand` entrega os dados na mesma ordem, confere cada um
  contra as faces e o caso de uso recusa os que sobram.
- **O ✓ exige decisão e dano** (`AttackToCommit`); a manobra não rola dano, e
  o ✓ dela é a decisão.
- **O erro ENCERRA o golpe**: cobra a ação padrão (`session.Store.RuleAttack`)
  e deixa o que o ✓ deixaria — PM do impulso, guarda da investida, gatilhos
  cumulativos (`afterTheStrike`). O provisório fica com a decisão, para o
  jogador ler "o mestre disse: errou", até a vez mudar (`dropTheSettledMiss`)
  ou o próximo ataque o substituir.
- **Rolar de novo vale antes da decisão** (p226): o `Reroll` recusa depois dela.
- **Os campos da mão são sinais NUMERADOS** (`damage_die_1`…), montados em Go
  (`handDamageSignal`) e declarados vazios a cada cartão
  (`handSignalsDeclaration`), para o dado de um ataque não vazar para o
  seguinte. Um campo vazio recusa o gesto: misturar o dado da mesa com o do
  servidor seria pior que pedir o que falta.

- **No desktop o cartão FLUTUA** e se arrasta pela cabeça (`lib/float-card.ts`,
  `data-float` e `data-float-handle`); no celular fica no fluxo e mostra SÓ A
  CABEÇA, e o corpo é o diálogo `attack-sheet` que ela abre. É o mesmo nó nas
  duas larguras: no desktop o diálogo é desenhado dentro do cartão
  (`display: contents`), para o conteúdo não existir duas vezes. A posição vive
  no `style`, e o cartão declara `data-preserve-attr="style"`: sem isso o
  remendo do servidor o devolve ao lugar padrão no primeiro toque do mestre. O
  `attack-card-float.spec.ts` prova com um remendo que muda o PRÓPRIO cartão —
  passar a vez não o remenda, e "ficou no lugar" mediria o nada.

A ordem e quem faz cada passo estão presas no `TestTheDamageIsRolledOnlyAfterTheGMRules`
e no `TestTheGMsMissEndsTheAttackWithoutDamage`; a regra de dano sobre a
decisão, no `attack_ruling_rules_test.go`. Os casos de regra do livro decidem
como ele sugere (`suggestedAttack`), porque é assim que os exemplos se escrevem.

## A MANOBRA ANUNCIA a condição e o desfecho ao mestre; ela não os aplica

Duas das cinco manobras deixam condição — *"você deixa o alvo CAÍDO"* e *"uma
criatura AGARRADA fica desprevenida e imóvel"* (p234). As outras três têm efeito
de ITEM ou de MOVIMENTO, e condição ali seria inventar regra.

**O resultado guardado não traz vencedor nem aplica nada:** o `ManeuverOutcome`
tem os dois totais, a `Margin` e o `ConditionOnAWin`, a condição que o livro
RESERVA à vitória. Quem marca condição é o mestre, pelo `toggleCondition` da
fila.

**A LEITURA do embate é só para o mestre** ("O veredito é do mestre, e só ele o
vê", no [CLAUDE.md da raiz](../CLAUDE.md)). O `engine.ReadManeuver` aplica o
desempate da p234 — maior total; no empate, maior bônus; bônus iguais, outro
teste —, e a faixa do mestre diz sucesso, falha ou empate, e o efeito extra da
margem de 5. A faixa do jogador passa pelo `forPlayer`, que tira o veredito, a
Defesa e o lado de quem resiste.

- O `TestTheManeuverNeverDeclaresAWinner` varre os campos do `ManeuverOutcome`
  por reflexão: o vencedor não é guardado nem aplicado.
- O `TestNoManeuverTextTheTablePlayerSeesDeclaresAWinner` varre o TEXTO da
  faixa do JOGADOR atrás de palavra de desfecho, que compila.
- O `TestOnlyTheGMSeesTheAttackVerdict` desenha a mesma mesa para os dois
  papéis, que é o predicado.
- **O id não vai para a tela**: o `book.ConditionName` o traduz (regra do
  `GLOSSARY.md`: id em inglês no código, texto em português na tela).

## O guarda de ÍCONE varre as CENAS, e o `switch` do gerado não tem `default`

Nome que ninguém gerou rende um `<svg>` VAZIO, sem erro. O
`TestEveryRequestedIconExistsInTheGeneratedFile` varre `serve/` inteiro, casa
`@icone("X")` e `@ui.Icon("X")` e tem DENOMINADOR. **Ícone novo entra na lista
do `scripts/gen-icons-templ.mjs` e o gerador roda — nunca à mão no `icons.templ`
gerado.**

## Todo ALVO de modificador é declarado, e o RÓTULO não conta como leitor

Cada alvo do `targetKey` está numa de duas listas do
`domain/engine/modifier_target_rules_test.go`: os que o motor APLICA, com o
arquivo que os lê, e os INERTES (`alvosDeclaradosInertes`), com o motivo. As
duas cobrem exatamente o `switch`, nos dois sentidos, e reprovam pelo nome.
**A lista de inertes só pode encolher.** **O rótulo do
`serve/web/sheetui/effects_labels.go` NÃO conta como leitor**: o defeito caçado
é o modificador que aparece na aba Efeitos sem entrar em conta nenhuma.

Três famílias de inércia, três consertos:

- **o mecanismo ESQUECEU** (havia conta e ela não olhava o modificador): uma
  linha, e o teste nasce vermelho;
- **o mecanismo NÃO EXISTE**: construir o gesto, ou declarar por escrito que
  não será construído, como o `tempMp` (o app não modela PM temporário);
- **o alvo não devia existir** (só lendo o livro): era outro alvo, já existente,
  com condição; reescreva a entrada do catálogo.

**Casar nome não é casar conceito**: o nome de um alvo num arquivo do motor pode
ser outra coisa com a mesma grafia.

## O motor é ECS, e a ENTIDADE muda de fase para fase

**REGRA NOVA NASCE EM ECS. É o default, e o critério é APRENDER** (decisão do
dono): pergunte *"quem é a ENTIDADE, que COMPONENTES ela carrega, e que SISTEMA
os lê?"*, e não "que campo esta struct ganha". Regra sem entidade (formatador,
consulta, tabela fechada que só se lê) não é ECS: ECS para o que COMPÕE, função
para o que calcula. **Quem escolhe ECS paga o guarda de denominador junto**, um
irmão do `TestEveryDerivationSystemWritesItsComponent` com a lista à mão.

O pipeline da ficha é coleta → resolução → derivação → decomposição; o ataque é
um mundo à parte. A entidade muda porque a pergunta muda:

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

O grafo tem uma aresta só (a carga alimenta o deslocamento e as perícias); o
resto lê `(personagem, efeitos)`. **Um sistema que não roda deixa o componente
ausente, e quem o lê recebe ZERO**, sem erro nem log. Pagam isso o
`TestEveryDerivationSystemWritesItsComponent` (lista à mão, uma conferência por
sistema, porque o `ecs` não diz quantos componentes uma entidade tem) e o
`TestTheLoadIsDerivedBeforeWhatReadsIt`, que prende a aresta com o personagem
SOBRECARREGADO: sem isso, a carga zerada e a de verdade dão o mesmo deslocamento.

### Os componentes são agrupados por PERGUNTA

Um componente junta o que se lê junto: `derivedMagic` carrega limite de PM, CD e
custo. Um campo por componente seria a struct de volta, com cerimônia.

### O que a resolução em sistemas comprou

O termo que não cumpre condição, perde o empilhamento ou vira fator fica no
mundo com a tag `Suppressed{Why}`, para a ficha poder responder *"por que 12?"*
com "−2 de X, não aplicado por não empilhar". A tela ainda não o lê.

### O ataque em sistemas, e a única ordem que a rede NÃO pegava

- **O `d20` entra por parâmetro e os dados por função** (`JudgeAttack`, `RollAttackDamage`), para
  cada caso do livro ser um teste com o número escrito à mão. O `attack.go` só
  tem o vocabulário da fronteira e a montagem do mundo.
- Os casos do livro prendem quase toda a ordem; que cada sistema escreveu o
  componente dele prende o `TestEveryAttackSystemWritesItsComponent`, com um
  CENÁRIO por sistema, porque alguns só agem numa metade do dado.
- **Suprimir é write-once, e a PRIMEIRA razão é a que fica**: um sistema que
  passe depois não troca o motivo que a mesa lê pelo dele.
- **A notação de dano só é lida no tempo do dano**: julgar o teste não a lê, e o
  `RollAttackDamage` e o `DamageToRoll` falham num `"1d"` do catálogo sempre.

### O tabuleiro entrou, e foi ele quem devolveu o `Despawn`

- **Peça e marcador são a mesma espécie de entidade, e o que o mestre escondeu
  SOME INTEIRO**: `Despawn` apaga a entidade e todo componente dela; "presente
  porém anônima" entregaria a emboscada igual. O `Despawn` pela metade é pego
  pelos casos de redação do `domain/board`.
- **A ENTIDADE NÃO TROCA entre os gestos do tabuleiro**, ao contrário da ficha:
  muda o COMPONENTE (`body` para quem pergunta quadrado, `boundTo` para quem
  pergunta fila), e os dois pipelines compartilham um construtor de mundo só.
- **A ORDEM da redação é regra**: é a de DESENHO (quem vem depois fica por cima)
  e decide qual peça a tela oferece para mover. Asserção sobre CONTEÚDO
  sobrevive a um embaralhamento; prendem o `TestTheRedactionKeepsTheDrawingOrder`
  e o `TestTheRedactionKeepsTheMarkerOrder`.
- **O CORPO da peça é um componente** (`body`), dono de "ocupa
  `footprint × footprint` quadrados a partir da âncora". O footprint zerado vira
  1 ali porque a peça do BANCO não passa pelo `AddToken`
  (`TestATokenSavedWithoutASizeStillTakesItsSquare`).
- **Monte o mundo UMA vez por gesto**: o `PopulateBoard` põe a peça recém-criada
  no mundo antes de procurar o quadrado da seguinte, ou as duas acham o mesmo
  lugar livre.

### A FRONTEIRA do tabuleiro abriu para o núcleo, e o preço foi um guarda

O `domain/ecs` está na lista de permitidos do
`TestTheBoardDoesNotKnowTheCharacterSheet` pela razão do `infra/events`: é FOLHA
e não sabe o que é Tormenta. A lista avisa: *acrescentar import a ela transforma
a porta em enfeite*. A folha tem guarda, `TestTheCoreImportsNothingOfTheProject`
(exceção: o `world_test.go`, do pacote externo `ecs_test`), que varre os testes
de propósito: os componentes de mentira do núcleo se chamam `label` e `weight`
porque *"se uma palavra do livro aparecer neste pacote, o desenho vazou"*.

### A ORDEM dos sistemas é a regra, em toda fase

O oráculo não cobre ordem (nenhuma fixture tem todas as fontes juntas). Guarda
por fase: **coleta**, `TestEveryCollectorLandsInTheDeclaredOrder`; **resolução**,
`TestAFactorWhoseConditionFailsDoesNotApply` (colher o fator antes de julgar a
condição, preso pela CONSEQUÊNCIA, que sobrevive a uma reescrita dos sistemas);
**derivação**, `TestTheLoadIsDerivedBeforeWhatReadsIt`; **redação do
tabuleiro**, os dois guardas de ordem de desenho.

**Acrescentar uma fonte NÃO faz o guarda da coleta reprovar**: ele só mede os
pares que o DADO dele põe na mesma ficha, então o `everyCollectorCharacter`
cresce junto com o `collectionSystems`. Controle: trocar duas linhas da ordem
tem de reprovar nomeando as duas espécies fora de lugar.

### O PODER é uma coisa; a CONCESSÃO dele é outra

O `transcript.ClassPowers` tem o **verbete** (sem `className`, com a regra: nome,
descrição, página, bloco de escolha) e a **concessão** (com `className`, o nível
ou a escolha que destrava, e um `powerUid` apontando para o verbete). **A regra
é escrita uma vez, no verbete**; copiá-la para cada classe que concede o poder
dá duas regras com o mesmo nome.

**Quem resolve são DOIS pontos, um por leitor**: o `domain/book`, para a tela, em
`classPowersResolved`; o MOTOR, na carga do catálogo, em
`resolveClassPowerPointers` (a concessão sem nome ou sem modificador próprio
herda os do verbete). Não compartilham código porque isso exigiria uma aresta
entre `domain/engine` e `domain/catalog`.
`TestTheAttributeIncreaseLandsOnTheChosenAttribute` prende que o ponteiro chega
à ficha. **Antes de mudar a forma de um recurso, ache os leitores pelo nome
dele** (`git grep '"class-powers"'`), e não pela memória.

### O verbete tem DOIS identificadores, e eles têm papéis opostos

- **`id`** é o SLUG, o PÚBLICO: `[a-z0-9-]`, único no recurso (`esquiva`,
  `bencao-do-mana`). A URL (`/<recurso>/<id>`), o `grep` e a fixture o usam.
  Muda quando o nome muda.
- **`uid`** é a IDENTIDADE (`pwr_k3m9x2ft4q`): o banco o guarda, e um catálogo
  aponta para outro por ele. **Nunca** muda, e é opaco porque um id derivável do
  nome volta a mudar quando alguém "corrigir" a derivação.

Cunhar é `go run ./cmd/mintuids --aplicar` (de `engine-go/`; sem a flag, só
relata), e ela só CRIA. O escopo (quais arquivos e caminhos têm verbete) mora só
no `cmd/mintuids` e **não se repete em Go**: os guardas de
`domain/catalog/uid_rules_test.go` (`TestEveryCatalogEntryCarriesAUid`,
`TestNoUidNamesTwoDifferentThings`, `TestNoUidEverChanges`) conferem o resultado
contra `testdata/uid_baseline.txt` sem saber o escopo. **Verbete NOVO entra na
base quando alguém a regerar; o que reprova é um uid EXISTENTE mudar** — o banco
perde a referência e a ficha deixa de achar o verbete, sem erro.

#### Endereço não carrega RÓTULO, e a coincidência esconde isso

- O `TestEveryCatalogIdIsASlug` varre todo objeto com `id`, sem linha de base; o
  ponto é namespacing e passa (`class.barbaro.impeto`).
- **Um renome de chave preservou o comportamento quando o diff do oráculo não
  tem NENHUMA linha de número.**
- **Escolha guardada como JSON dentro de uma string também é ponteiro** (o
  `"race":"Lefou"` do `secondaryRaceChoices`), e varredura de campo não a
  alcança: renomeou chave, procure ali também.

### O bônus CIRCUNSTANCIAL já tem vocabulário — não invente um

Antes de acrescentar forma ao `Modifier`, veja se não é `context` com nota (o
interruptor que a PESSOA liga), `against` com traço ("contra o quê") ou
`terrain` (o tipo de terreno). **Cada `against` é um interruptor**; o que se
liga JUNTO divide uma flag (`flagOn`), como a Marca da Presa e os acréscimos
dela. Quase sempre falta o dado, não o motor. Erros que um teste só de números
não vê:

- **Alvo que já É o escopo dispensa condição**: "+2 em testes de ataque para
  derrubar" (p125) é `maneuver:derrubar`, sem `condition`. Opt-in para uma
  circunstância já escolhida faz valer de menos; circunstância sem interruptor
  faz valer de mais.
- **"Resistência a X +N" é bônus em TESTE (`resistance`), não redução de dano**
  (p226).
- **A manobra é teste OPOSTO (p234), e o alvo diz o lado**: `maneuver:<nome>` é
  quem ATACA (o +2 do `Derrubar Aprimorado`); com `scope: "defense"`, quem
  resiste (o +5 do `Desejo de Liberdade`, p89). O escopo entra na CHAVE do
  efeito, ou os dois somam no mesmo balde.
- **Para escolher o alvo, pergunte "que OUTROS números saem desse mesmo
  teste?"**, não só que teste se rola: a manobra é teste de Luta, e um bônus
  defensivo em `expertise:Luta` sobe o ataque da arma. Recorte fino pede alvo
  próprio.
- **Número impresso SEM alvo não vira modificador**, e o motivo fica escrito:
  penalidade que o motor não aplica (o "Água no Feijão": um +5 valeria sempre
  por um –5 de fabricação raro) ou TESTE que o livro não define (o "Vendedor de
  Carcaças"). Guarda: `TestEveryBonusAnOriginUniquePowerPrintsBecomesAModifier`,
  lista de PERMITIDOS com o POR QUÊ de cada linha.

### O DANO TEM TRÊS PARCELAS, e só uma multiplica no crítico

A p231: *"Multiplica os DADOS de dano do ataque (incluindo quaisquer aumentos
por passos) pelo multiplicador da arma. Bônus numéricos de dano, **assim como
dados extras**, não são multiplicados."*

| parcela | onde mora | multiplica? |
|---|---|---|
| os dados da ARMA | `WeaponCard.Damage` ("1d8") | **sim** |
| os dados EXTRAS de um encanto | `WeaponCard.ExtraDamage`, com o TIPO (é ele que diz à resistência a fogo onde agir) | não |
| o bônus numérico | `WeaponCard.DamageBonus` | não |
| o que só vale no crítico | `WeaponCard.CriticalBonus` | não, e fora do crítico nem existe |

**Estas duas formas do `Modifier` não são parcela da pilha**: o
`harvestWeaponRiders` as tira do mundo ANTES do `deferConditionals`, senão o
`onCritical` (condição que **não se liga**) viraria interruptor na aba Efeitos e
o dado extra somaria como `amount` ZERO (`weapon_riders_rules_test.go`):

```
{"target": {"k": "damage", "damageType": "fogo"}, "dice": "1d6"}
{"target": {"k": "damage"}, "amount": 10, "condition": {"c": "onCritical"}}
```

**O que denuncia o erro aritmético é a CONTAGEM de dados, não o total**: crítico
x2 com arma flamejante é 2d8 + 1d6, e 2d8 + 2d6 também dá total plausível. Por
isso o dublê do `attack_rules_test.go` estoura quando a regra pede um dado a
mais do que o caso preparou.

### A escala por PATAMAR, e o `default` que engole o `per` desconhecido

`scale.per: patamar` multiplica pelo ORDINAL do patamar (1 iniciante, 2
veterano, 3 campeão, 4 lenda): o "Coração Heroico" (p91) é um modificador só,
`maxPm 3`, dando 3/6/9/12. O `levelStep` não serve: os limiares (5, 11, 17) não
são passo fixo. **O `switch` do `evalModifierScale` tem `attribute` no
`default`**: um `per` desconhecido lê `attrTotals[""]`, acha zero, e o
modificador some sem erro. Guarda: `TestNoCatalogScaleIsOneTheEngineCannotRead`.

Para achar um poder no PDF, case por `bookpdf.Key` **dentro da página que o
catálogo declara**: o nome vem em VERSALETE (caixa baixa no `pdftotext`), e o
capítulo tem a TABELA de pré-requisitos antes dos verbetes.

### A SOMA DE ATRIBUTO: o `scale` fora dos vitais

"Você soma seu X em Y" é `scale: {per: attribute, attribute}` num alvo qualquer;
o `cap: classLevel` o limita ao nível NA CLASSE que concede o poder (p226), só
onde o verbete escreve "limitado pelo seu nível" (a Presença Paralisante e a
Mente Criminosa não são).

- **Ela se resolve numa SEGUNDA passada** (`sheetEffects`), porque o número é o
  atributo EFETIVO, com os itens. Por isso uma soma não mira atributo
  (`TestNoAttributeSumTargetsAnAttribute`), e toda decomposição da ficha lê o
  `sheetEffects`, não o `ComputeItemEffects`.
- **O atributo negativo SUBTRAI** (decisão do dono): o livro não tem regra
  própria e usa o valor como é — "10 + sua Destreza" (p106).
- **A trava que nomeia duas coisas usa `flags`**: o `flagOff` cala se QUALQUER
  uma estiver posta ("de armadura pesada ou na condição imóvel", p47); o
  `flagRequired` pede todas. A condição Imóvel (e quem a contém) levanta a flag
  `imovel`.
- **O dano por TIPO de arma** cai nos escopos `melee-light-or-agile` e `ranged`
  (arremesso e disparo, p142), lidos pela carta da arma junto do `DamageOwn`. O
  ATAQUE por escopo de arma (o `weapon:<id>` do Foco em Arma) é o `AttackOwn`,
  que a decomposição da tela soma.

### A devoção é fonte da coleta, e não só de PV e PM

O poder concedido do devoto é FONTE da coleta (`godPowerActiveItem`) e chega a
qualquer alvo; o `vitalGrantMods` continua somando PV e PM, sem duplicar.

### Os acumuladores moram numa entidade SINGLETON

Flags e fatores são do mundo inteiro e ficam numa entidade só (`worldFlags`,
`worldFactors`), o idioma de "recurso" do ECS. Sem ela, um sistema escreveria
numa variável capturada e deixaria de ser função do mundo, que é o contrato do
`ecs.System`.

## O deslocamento é contado em QUADRADOS, e metro é coisa de tela

Todo deslocamento do livro é múltiplo de 1,5m: em metros, arredondaria na
fronteira do JSON e inventaria bônus; em quadrados, fecha exato. É a unidade do
tabuleiro (*"metro é coisa de tela"*, `board_movement.go`, p236), e a metade do
Lento (p395, *"arredonde para baixo para o primeiro incremento de 1,5m"*) vira
divisão inteira.

### A conversão é SIMÉTRICA, e isso não é enfeite

`amountInEngineUnits` na leitura, `amountInBookUnits` na escrita
(`Modifier.MarshalJSON`); sem a volta, reler o que o motor escreveu converte
duas vezes. Quem pega a assimetria é o `roundTrip` dos casos de paridade.
**O FIO fala a unidade do LIVRO**: o oráculo mostra −3 e +1,5 como a página
imprime, e o diff se revisa contra o livro.

### O modificador que MULTIPLICA

O `engine.Ratio` é um fator racional em inteiros (`{1,2}` metade, `{0,1}` zero)
— o Lento reduz o deslocamento à metade (p395), o Imóvel a 0m (p394), a Alta
Arcana corta o custo em PM pela metade —, num campo PRÓPRIO do `Modifier`: quem
tem fator não é parcela da pilha, o `Amount` fica zero e o valor vai do
`byTarget` para o `ItemEffects.Factors`. Racional porque o motor é em inteiros,
e a divisão inteira em quadrados é o arredondamento que o Lento pede.

**São DUAS FAMÍLIAS, distinguidas pelo ALVO:**

- **Fator de ALVO** (`{k: displacement}`, `{k: pmCost, scope: arcana}`): age
  DEPOIS da soma, sobre a BASE junto ("APÓS aplicar aprimoramentos e outros
  modificadores", Alta Arcana).
- **Fator de FONTE** (`{k: source, name: <id do poder>}`): age na PARCELA da
  fonte, ANTES do empilhamento. A Fúria Titânica (p42) dobra "o bônus que você
  recebe quando usa Fúria": 5 PM gastos dão +10, não o ataque inteiro dobrado;
  com o degrau pago, base e degrau do balde `morale` dobram e o não-empilhamento
  escolhe o maior.

**O fator de fonte tem DOIS sítios de aplicação**: o
`scaleWhatTheSourceFactorHits` no caminho incondicional, no ECS, e o
`ApplyActiveConditionals` na parcela DEFERIDA, que o ECS não multiplica —
dobrar nos dois dobraria duas vezes. A Fúria é condicional: vai pelo segundo.

**Dois fatores no mesmo alvo: vale o MAIS SEVERO, e eles não compõem**, com ou
sem condição — "aplique apenas o mais severo" (p394), *"reduções no custo de PM
não são cumulativas"* (p226); Lento mais Imóvel dá 0m. **O Foco em Magia é a
exceção ESCRITA**: −1 PM na magia escolhida, *"cumulativo com outras reduções de
custo"* (p131). Não é modificador `pmCost` (entraria na disputa da p226): o
`SpellPmCostFor` o soma por fora, antes do fator e do piso.

**O fator pode ter ESCOPO**: a Alta Arcana corta "suas magias arcanas" (p39),
pela chave `pmCost:arcana` (mesma divisa do `displacement:armor`), e o clérigo
do multiclasse paga cheio a divina. Quem diz a lista é a `sheet.SpellTradition`,
porque **a tradição é da CLASSE, não da magia**: a Luz é arcana no arcanista e
divina no clérigo.

> **A ISENÇÃO não é terceira forma do fator — é FLAG.** O anão "não tem o
> deslocamento reduzido por uso de armadura ou excesso de carga" (p20): o
> `displacementBreakdown` lê `DisplacementIgnoresArmor` e
> `DisplacementIgnoresLoad`, e pula a chave `displacement:armor` e a penalidade
> de carga sem tocar em fator (a magia de lentidão vale para o anão). **São duas
> flags porque o livro as vende separadas** (o Fanático, p128, compra só a da
> armadura). Isenção de terreno difícil (Acrobático, p129) é do `PathCost` do
> tabuleiro. **Confira na PÁGINA, e não na descrição do catálogo, antes de
> concluir que um poder pede isenção nova**: o corte pode ser da CIRCUNSTÂNCIA,
> que o app não modela.

### Onde a unidade aparece, e onde ela não aparece

| lugar | unidade |
|---|---|
| catálogo (os literais de `domain/catalog/transcript`) | metros |
| `Modifier.Amount` em memória | quadrados |
| `Modifier` no fio (JSON) | metros |
| `ComputedSheet.Displacement` (`json:"displacementSquares"`) | quadrados — e o nome do campo diz |
| `ItemEffects.ByTarget["displacement"]` | quadrados, e o nome do alvo é a única pista |
| a tela (crachá da mochila, régua da Mesa) | metros |

**Trocar a unidade de um campo do fio troca o NOME dele**
(`displacementSquares`): calado, todo leitor antigo leria 6 onde lia 9, sem erro.

## Catálogos

**As cenas leem o catálogo EMBUTIDO direto** (não há rota que o sirva), **uma
vez, e guardam** (`sync.Once`, como no `race_traits.go`): o livro é compilado no
binário.

**O LIVRO é código Go**, em `domain/catalog/transcript`: cada verbete é um
literal tipado, e é ali que regra transcrita se edita — para o compilador cobrar
a chave errada. Cena, motor e testes leem o mesmo valor; o `catalog.Resource` o
serializa para quem lê JSON (`transcript.ResourceJSON`).

- **A forma dos dados é CANÔNICA** (`transcript.WriteSource`), cobrada pelo
  `TestTheTranscriptIsInCanonicalForm`; depois de editar à mão, `go run
  ./cmd/transcriptfmt`. É o que deixa `cmd/mintuids`, `cmd/bookpages` e
  `cmd/audit/bestiary --aplicar` reescreverem o arquivo sem cirurgia de AST.
- **Comentário dentro de literal não sobrevive à reescrita**: o porquê do
  verbete mora no `modeled`/`unmodeled`, que a ficha lê.
- **Tipos (`*_types.go`) e registro (`layout.go`) se escrevem à mão**; recurso
  novo é uma linha no `layout.go`.
- **`Modifier.Amount`, `Item.Price` e `Item.Slots` não distinguem ausente de
  zero.**

**O que a CAMPANHA acrescenta não mora no livro**, que é compilado, enquanto a
mesa autora em tempo de execução: a `emenda de catálogo` vive em
`campaign_items` e ACRESCENTA modificador, nunca substitui — ver o
[GLOSSARY.md](../GLOSSARY.md).

**O mundo é um TIPO.** `engine.Catalogs` é o LIVRO e só consulta verbete;
computar ficha pede `engine.Ruleset`, o livro sob as emendas de UMA mesa —
`book.ComputeSheet(…)` não compila. Fora de campanha,
`engine.BookRuleset(book)`, explícito porque elenco, oráculo e fixtures estão
TOMANDO essa decisão. As emendas não viajam no `engine.Character`, onde campanha
e personagem chegariam misturados e a procedência morreria: `sheet.RulesetFor`
resolve UMA vez e o resultado viaja em `CharacterDTO.Ruleset` (`json:"-"`); o
lote da mesa (`sheet.RulesetsFor`) consulta uma vez por campanha DISTINTA.

### As três espécies de emenda, e por que não são uma lista só

| espécie | tabela | o que muda | escopo |
|---|---|---|---|
| **verbete** | `campaign_items` | o que uma COISA é | a campanha inteira |
| **concessão** | `campaign_grants` | quem TEM as coisas | seletor |
| **silêncio** | `campaign_silences` | o que esta mesa NÃO aplica | seletor |

Respondem a perguntas diferentes; uma tabela com `kind` as forçaria a uma forma
só. A de verbete não tem escopo: item que significa uma coisa para você e outra
para o vizinho são dois mundos. O **seletor** (`engine.Selector`) é `everyone`
ou `character` (id do CLONE na campanha); espécie desconhecida não alcança
ninguém, e **o id ZERO nunca casa** — senão o `Character{}` dos fixtures poria a
emenda de uma mesa no oráculo.

### O endereço de um termo NÃO carrega o valor, e o opt-in do jogador usa o mesmo

`engine.TermID` é `fonte::alvo::escala::condição`, mais categoria e peça
(`From`) só quando existem. Sem valor, tipo de bônus ou prosa (`note`, `label`):
corrigir o livro trocaria o endereço e soltaria o que estava preso a ele. Escala
e condição separam pares reais (anão: `maxPv +2` e `maxPv +1 por nível`). O
`TestEveryCatalogTermHasAUniqueAddress` prende "é único", não "é único hoje";
nele, a raça é UMA fonte (`raceActiveItems` junta tudo num `ActiveItem` com o
`race.ID`), e o benefício de origem repetido em várias origens conta uma vez
(`getOriginBenefit` lê o primeiro que casa).

O opt-in (`character_conditionals`) e o silêncio gravam o mesmo `TermID`.
**Mudar a forma do endereço é migração**: linha velha não casa, e o situacional
fica desligado para sempre — por isso categoria e peça só entram quando existem.

### JOGADOR NÃO MUDA REGRA — isso é do mestre (decisão do dono)

Vale para toda tela que vier. O jogador alterna o OPT-IN de um condicional que a
ficha já oferece; emenda, concessão e silêncio são do mestre. **Quem cobra é o
servidor**: POST à mão com interruptor que a tela não oferece é recusado pelo
caso de uso. A lista de PERMITIDOS tem UM dono (`character.SituationalGroupsOf`)
— a cena desenha dela, o caso de uso recusa fora dela —, e as posturas ficam de
fora porque custam PM.

**`character_conditionals` não é `campaign_silences`**: o opt-in nasce DESLIGADO
e é do dono da ficha; o silêncio nasce LIGADO e é do mestre. Uma tabela só
juntaria duas permissões de escrita.

### O interruptor de GRUPO tem endereço próprio

Condicionais que dividem uma flag viram UM interruptor, com chave
`engine.FlagGroupID` — não o endereço de um membro, para "metade ligada" não ser
representável. Vale para a POSTURA, o Folião, o terreno do Explorador e a
`marca-da-presa`.

### O catálogo chega ao processo por dois caminhos

O `catalog.Resource` serve o `transcript` compilado, que sempre existe. O
`s.catalogs` (`book.EngineCatalogs`) é um CAMPO que a bancada troca — há fixture
com `{"items":[]}` de propósito. **Validação de regra lê o embutido**; o
`s.catalogs` é do motor.

### Regra que só a TELA sabe é fronteira aberta

Sintoma: a tela tranca e o servidor não. Progressão de círculo (`Spellcasting`
em `transcript.Classes`), melhoria/material × item (`aceitaMelhoria`, no comando
da ficha) e limite de texto da campanha (`domain/campaign/rules.go`) moram no
servidor; o filtro da tela é só conveniência.

### A validação de ESCOLHA é do servidor, e é estrita

Vagas de poder (uma por nível a partir do 2º, p33), benefícios de origem,
caminhos e deuses por classe se conferem na gravação, não só no diálogo.

- **`sheet.WithChoicesValid`** roda em `saveTheChoices` e é **estrita**: a
  escrita deixa a ficha VÁLIDA, não só "não piora". Não há ficha de mesa real a
  proteger (decisão do dono).
- **`powerChoices`**: só poder que a ficha tem, opção que o livro oferece, mais
  de uma só no repetível, que ocupa uma vaga por ESCOLHA
  (`sheet.UsedPowerSlots`).
- **Poder GERAL com escolha** (Foco em Perícia, Foco em Arma) passa por
  `book.PowerWithChoice`; a escolha vinda da origem é grátis
  (`sheet.FreePicks`).
- **Contada por ATRIBUTO** (`countAttribute`, Especialista, p73): sem vaga,
  perícia treinada, até o atributo sem temporários
  (`Ruleset.PermanentAttribute`), mínimo 1. Gesto que mexe no N ou no treino
  (treino, ofício, guardar/vestir/remover item, raça, forja) aplica numa cópia e
  chama `sheet.ChangeKeepsChosenExpertises` antes de gravar.
- **Contada por NÍVEL** (`countLevels`, terrenos do Explorador, p50): repete sem
  vaga; descer de nível confere `sheet.LevelCountedChoicesFit`.

### Schema pega FORMA, não número errado

Dado transcrito se protege por **schema**
(`domain/catalog/rules_tables_test.go`), não por `expect` por campo: pega
perícia inexistente, faixa com buraco, devoto de raça inexistente. **Número
errado é schema válido** — contra ele, o auditor:

```
cd engine-go
go run ./cmd/audit/bestiary            # relatório
go run ./cmd/audit/bestiary --aplicar  # escreve as correções
```

### Os auditores do livro

`cmd/audit/<nome>`; o livro (gitignorado) se acha por `T20_BOOK_PDF`, por um
`t20-book.pdf` subindo do diretório ou o do checkout principal numa worktree. A
leitura mora em `infra/bookpdf`, com `\s`/`\w` Unicode (`IsPySpace`) por causa
do espaço não separável. Rodar é **ato deliberado**, como o `genoracle`: a
ferramenta PROPÕE e o diff revisado contra o livro decide. **Mexeu num catálogo?
rode o auditor dele.** É um por seção do sumário, e o que os distingue é a
ÂNCORA — os campos que ligam um pedaço do livro a um verbete:

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

**Dois não leem o livro.** O `cmd/audit/coverage` confere raça, classe e origem
contra o MOTOR — quanto vira número, gesto ou nota do mestre. O balde é
DECLARADO (`modeled`/`unmodeled`), a nota cita a issue da fatia que a resolve,
`--destino` agrupa por ela, e verbete sem efeito nem `unmodeled` reprova (sem
guarda em Go, por decisão do dono). O `cmd/audit/pointers` lista arquivos
citados em comentário de teste que sumiram — candidatos, não veredito.

### O que vale para todo auditor

- **A âncora é o trabalho; o resto é regex.** Só bloco contíguo e completo cujas
  âncoras batam com o catálogo; o que não se liga REPROVA com nome.
- **Coordenada, nunca `pdftotext -layout`**, que junta colunas vizinhas. Quando
  o `<line>` junta título, estado e coluna vizinha, leia por PALAVRA
  (`Book.Words`).
- **As faixas `d%` ladrilham o intervalo**: a soma se afirma antes das parcelas
  (`bookpdf.RangesThatDontClose`).
- **Duas âncoras se conferem entre si ANTES do catálogo**: leitura errada vira
  discordância nomeada, não tabela plausível.
- **O que o motor NÃO aplica sai com denominador, não como lista de proibidos.**
  Todo encanto tem modificador ou diz por que não, nunca os dois (motivo ao lado
  de modificador envelheceu) — cobra o
  `TestEveryEnchantEitherAppliesOrSaysWhyNot`. Nos materiais,
  `modeled`/`unmodeled` são PERMITIDOS por METADE do verbete, e a união dá
  exatamente as partes dele.
- **Medidor de prosa recorta onde a prosa acaba** e imprime a cobertura mínima;
  sem o recorte, outro assunto infla a medida.
- **A DIREÇÃO da medida é do TERRENO**: catálogo EQUIVALENTE pergunta "a regra
  do livro está aqui?"; catálogo que RESUME, "tudo o que ele afirma está na
  página?" — para números, perícias e palavras (`bookpdf.Coverage`). O veredito
  é humano, e as duas saem lado a lado.
- **Filtro de lixo é um só**: `WithoutJunk` (marca d'água, número de página,
  cabeçalho de capítulo).

### A FRONTEIRA JSON do `Modifier`

O `amount` do catálogo está na unidade do LIVRO e o motor guarda deslocamento em
quadrados — é a razão de `MarshalJSON`/`UnmarshalJSON`. **Na leitura o tipo é
EMBUTIDO** (alias local, sem método e sem recursão) e só o `amount` se
redeclara, porque o campo menos profundo vence o embutido: campo novo atravessa
sozinho. **Na escrita os campos ficam à mão**, porque o oráculo compara byte a
byte e embutir jogaria o `amount` para o fim; cobra o
`TestEveryModifierFieldSurvivesTheCatalogJSON`, por reflexão sobre o TIPO. Teste
que monta o `Modifier` em Go não atravessa essa fronteira.

### A LEITURA de tabela e de verbete mora no `bookpdf`

`Book.LootRows`, `RangesThatDontClose` e `Book.EntriesByName` guardam as
armadilhas comuns: linha montada pelo Y, última linha colapsada num `<line>`,
`d%` alinhado à direita (o x é faixa), margem que muda entre página par e ímpar.
**Leitura que um segundo auditor copiaria vira função do `bookpdf`**, e o
refactor se prova com o relatório BYTE A BYTE igual — número que muda é
indistinguível de correção.

### O ITÁLICO é dado, e ler estilo pede outro leitor

O `-bbox` do `pdftotext` **não marca estilo**; o `pdftohtml -xml` marca, noutra
régua. O `Book.Italics` converte pela razão entre as alturas da MESMA página e
devolve na coordenada do resto do módulo; sem ele, comparam-se réguas
diferentes.

### E o buraco INVERSO: o valor existe, mas não se confere que ele APONTA

No `cmd/seed/seed-data.json`, raça, origem, classe, deus, poder concedido,
tamanho, magia e item passam por lookup (`cmd/seed/catalog_refs.go`). **Seed que
mente não quebra**: id de arma errado dá ficha sem a arma, abrindo normal — e o
e2e passa a medir o ambiente. Todo validador de referência:

- **sugere o VIZINHO**, com teto de um terço do comprimento: sugestão errada é
  seguida;
- **junta TUDO antes de falhar**, para cinco ids errados não pedirem cinco
  rodadas;
- **afirma o próprio DENOMINADOR**: o catálogo carrega em `sync.Once` e engole
  erro: com a carga quebrada toda lista vem vazia e todo nome seria acusado.
  Lista vazia é falha de carga, e a mensagem diz isso;
- **trata CHAVE de objeto como referência** (a chave do `classChoices` é nome de
  classe) — o `TestEveryCreateFieldOfTheSeedIsClassified` obriga a classificar
  campo por campo.

## O fixture do app prima o catálogo DE VERDADE

**Catálogo vazio no fixture é validação desligada em silêncio**: quase toda
checagem diz "não sei" ao desconhecido (recusá-lo travaria fichas antigas), e o
guarda fica verde sem rodar. O `newTestServer` prima o embutido
(`book.EngineCatalogs`); `{"items":[]}` é exceção declarada (ver "O catálogo
chega ao processo por dois caminhos").

## A bancada dos testes: um molde migrado, copiado por teste

O `newTestServer` abre um SQLite de VERDADE por teste, para o `serve/api` provar
composição. O caro é MIGRAR (cada `fsync` do `db.Open` custa o disco do
`TMPDIR`), então o `TestMain` migra UM molde (`infra/db/testdb`) e cada teste o
copia (`testdb.Fresh`). Cada pacote declara:

```go
func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }
```

- **`PRAGMA synchronous=OFF` fica no `newTestServer`, NUNCA no `db.Open`**:
  banco que morre no fim do caso não tem o que proteger; em produção é perda de
  dados do mestre.
- **A cópia FALHA ALTO** (`Close` com erro, `Sync`, TAMANHO comparado): cópia
  parcial parece banco não migrado (`no such table`) e manda caçar migração.
- **O molde é PACOTE, não `_test.go`**, que não exporta: senão a cena que
  virasse pacote escreveria a própria bancada, e o catálogo vazio nasce assim.

A montagem do servidor NÃO se compartilha: precisa do `api.Server`, e a bancada
que o importasse seria importada de volta. O molde é a parte cara; o fixture é
de cada pacote.

## Testes

As faixas, o vermelho antes de confiar e o que não merece teste estão em
"Testes" do [CLAUDE.md da raiz](../CLAUDE.md). Daqui:

- `go test ./...` — sem flag, sem setup.
- **Teste de regra vive junto da regra e cita a página** (e o exemplo
  trabalhado: `TestBolaDeFogoWorkedExample`). Ele e o ORÁCULO não se substituem:
  o oráculo acusa qualquer número mudado na ficha inteira; o teste explica POR
  QUE o número é aquele.
- **Vermelho em Go**: inverta o operador, rode, confirme o vermelho, reverta.
  Para guarda novo, a árvore de ONTEM é o vermelho real — `git archive` do
  commit anterior, o guarda por cima, `go test` —, e não sai inerte como
  sabotagem.

## `convention`: os guardas que não são de pacote nenhum

`convention/` não tem código de produção. Existe porque "regra mecanizável vira
guarda" e "o guarda mora no pacote que a possui" se contradizem quando a regra é
de TODOS os pacotes, e pôr a regra no `api` seria dono arbitrário. Quantos são:

```
grep -h '^func Test' convention/*_test.go | wc -l
```

**Nasce aqui o guarda** cuja regra é do REPOSITÓRIO, que mediria por acaso o
diretório em que mora se varresse `.`, ou que é da família das CITAÇÕES
(arquivo, teste e símbolo que não existem mais).

- **O terreno é o DIRETÓRIO, nunca padrão de nome**: glob deixa de casar no
  renome, e o guarda segue VERDE medindo menos.
- **Guarda que proíbe a cópia falha quando o original some** — o
  `TestNoHandwrittenFocusRing` afirma que a regra global segue no `index.css`.
- **A LÁPIDE é declarada, não apagada**: `// Aqui morava o TestX, que prendia…`
  diz o que o `git log` esconde; os guardas de citação pedem que entre em
  `tombstones`, `arquivosAusentesDePROPOSITO` ou `simbolosAusentesDePROPOSITO`.

**Idioma não é guarda de pacote nenhum** — ver "Isto NÃO tem guarda, e é decisão
do dono" na [raiz](../CLAUDE.md). Nome PRÓPRIO do livro não é prosa:
`TestBolaDeFogoWorkedExample` é o nome da magia.

## templ — as armadilhas que já custaram tempo

`.templ` compila por `go tool templ generate`; a CI recusa `.templ` e
`_templ.go` desencontrados.

- **`else if` NÃO existe numa lista de atributos, e o templ não reclama**: ele
  escreve ` else` como TEXTO e abre um `if` independente, os DOIS ramos saem e o
  navegador fica com o PRIMEIRO atributo. **A escolha volta para o Go.**
- **Comentário (`// ...`) NÃO vive na lista de atributos**, e o erro aponta
  OUTRA linha: solto, `malformed open element`; dentro de `for`/`if`, `for:`/`if: expected
  nodes, but none were found` na linha do BLOCO — `for`/`if` intocado reclamando
  de "expected nodes" é comentário entre atributos. Ponha-o ACIMA do elemento.
  **E o `go build` fica VERDE** com o `_templ.go` antigo: leia a saída do
  `templ generate`.
- **`@componente()` tem de COMEÇAR a linha**; no meio de texto, a tela mostra
  `@tecla("⏎")`.
- **Atributo CONSTANTE sai literal; só o DINÂMICO é escapado** (`'` vira
  `&#39;`) — importa ao afirmar HTML em teste.
- **NUNCA passe formatador em arquivo GERADO.** O `goimports -w` reordena os
  imports soltos do `_templ.go`: regenerar e compilar não denunciam, e a CI
  reprova em "Fail if the generated templates were stale". Em toda fatia que
  move arquivo de pacote, de dentro de `engine-go/`:

  ```
  go tool templ generate && git diff --quiet -- '*_templ.go'
  ```

  `_templ.go` só se corrige regerando.
- **Regenerar não basta: REINICIE o servidor** — o `go run ./cmd/api` segue
  servindo o HTML antigo.
- **Classe nova exige `scripts/build-css.sh`**, senão sai sem estilo e sem erro.
  O Tailwind v4 varre a árvore sozinho (respeita `.gitignore`; os `@source` do
  `serve/web/assets/app.src.css` não o limitam): **entra na folha prosa de
  `.md`, docstring e identificador Go**; só `@source not` exclui (`static/`,
  `infra/`, `cmd/`, `domain/catalog/transcript/`). Daí:
  - **compile por ÚLTIMO**: **código → prosa → `build-css.sh` → `git add`** —
    também ao APAGAR classe, senão "Fail if the stylesheet was stale";
  - **rode-o mesmo fora de tela**: identificador igual a uma utilidade a
    acrescenta;
  - **o comentário sobre esse risco não escreve o token** em minúsculas.
- **O `datastar.js` é VENDORIZADO e sobe por `scripts/vendor-datastar.sh`**, não
  pelo Vite (o `serve/web/ui/layout.templ` o referencia num `<script>` à mão), e
  não há dependência no `package.json`. O script fixa a versão, baixa da tag e
  reprova se o cabeçalho disser outra; fica FORA do `prebuild` — subir é ato
  deliberado, porque o reconciliador mantém as cenas vivas. Depois, o Playwright
  INTEIRO — o guarda do morph é `ferir duas vezes seguidas não apaga a piscada
  da primeira`.
- **Regra da casa que precisa GANHAR de utilitário mora em `@layer utilities`**,
  não em `components`: no Tailwind v4 a CAMADA decide antes da especificidade, e
  a regra descartada parece cálculo errado. Muda-se a camada, não o elemento.
- **`inset-0` põe `left: 0`, e `left` ganha de `right`**: regra que ancora à
  direita sobre base com `inset` escreve `left: auto`, com o comentário dizendo
  por quê.
- **TOKEN inventado não vira regra, nem com o script rodado**: o elemento herda
  a cor. Confira o `@theme` do `serve/web/assets/src/index.css` antes de
  inventar um nome; o `TestEveryHouseTintExistsInTheStylesheet` cobra cada token
  (casa e papéis em `asPaletasDaCasa`: `destructive`, `primary`, `muted`, …)
  contra a folha, varrendo o DIRETÓRIO com piso de denominador que SOBE com a
  varredura — piso antigo não denunciaria a volta do conjunto antigo.
- **A CLASSE DE ESCOPO não é tinta**: `scene-grimorio` é condição dos tokens (o
  `@custom-variant dark` é `&:is(.dark *, .scene-grimorio, …)`) — sem ela o app
  sai sem cor. Cobre o `TestEveryScopeClassExistsInTheStylesheet`, que lê só
  `class=` e `templ.KV("…")` — `data-slot` como `scene-title` não é classe, e a
  casca escolhe `scene-ds` por condição.

  > **Renome em massa:** `\bnome\b` pega classe composta, ROTA e nome de arquivo
  > (o hífen é fronteira). Símbolo Go, `gopls rename`, que é semântico; troca
  > textual só para o componente `templ`, com alvo COMPOSTO e único.

## Como uma CENA é construída

**O formato é uma PORTA declarada pela cena**: a `Deps` em
`serve/web/<cena>/deps.go`, cumprida pelo `*api.Server` — o CONSUMIDOR escolhe o
que atravessa. O `serve/api/web_router.go` monta uma linha por cena
(`campaigns.Routes(r, campaigns.New(s.campaignsHost(), …))`), onde o compilador
cobra a porta.

**O CASO DE USO entra por PARÂMETRO**: a porta só desvia do ciclo cena ↔ `api`,
e o `app/` está abaixo dos dois. O `New` diz o que a cena faz:

```go
table.New(s.tableHost(), s.sessionLifecycle(), s.tableMemory, s.restParty(), s.initiativeQueue(), …)
sheetui.New(s.sheetHost(), s.characterPlays())
campaigns.New(s.campaignsHost(), s.sessionAccess(), s.campaignDirectory(), …)
door.New(s.doorHost(), s.accountGate(), s.accountResets())
```

Entrada que vira caso de uso SAI da `Deps`, sem adaptador.

**Estado que a cena guarda e outro alcança nasce no HOSPEDEIRO**, não no `New`:
os mapas `(sessão, pessoa)` da Mesa são esvaziados pelo ciclo da sessão, montado
ANTES dela — daí `s.tableMemory`. O sinal: "o fim de X tem de alcançar isto".
**Sentinela do `app/` se lê com `errors.Is`** (`campaign.ErrNoSuchCampaign`),
sem enum traduzido pelo hospedeiro.

O `boundary_test.go` de cada `serve/web/*` PERMITE imports e recusa o hospedeiro
— não pelo ciclo, que o compilador pega, mas pelo atalho: import direto de
`domain/catalog` contornando o `book`, que não é erro e é a divisão vazando por
baixo.

### As regras da porta

- **O MENOR tipo**: `CurrentUserID(r) int64`; porta que devolve tipo do
  hospedeiro é o hospedeiro com outro nome.
- **A PERGUNTA, não o objeto**: `RequesterIsAdmin(r) bool`.
- **A FRASE é sempre da cena.** Quem CLASSIFICA é o hospedeiro enquanto o
  sentinela for do `api`; descido ao `app/`, a cena usa `errors.Is` e o
  classificador some.
- **Porta larga é sintoma de cena fazendo trabalho de outra** — não se pergunta
  "dá para cortar um método?".
- **Conserta-se a porta, não o chamador**, quando o teste do hospedeiro obriga a
  abrir método.
- **Contrato de terceiro existente ganha da regra.**
- **O dublê da porta RENDERIZA**; struct sem desenho não prova nada da tela.

### Onde um pedaço de código mora

**O destino de uma função é a DEPENDÊNCIA dela**, não o assunto nem o chamador:
`bcrypt` não vai para cena, `sql.Null*` vai para o `dbvalue`, regra que lê o
livro vai para o `book`. **Dentro do arquivo a linha é a mesma**, e **o menor é
o que não se repete**: cópia curta declarada no comentário custa menos que
método a mais na porta. **A cena sai; a bancada fica** — os testes batem no
roteador, que segue no `api`.

## Código órfão, bancada mentirosa, e raciocínio envelhecido

As três produzem VERDE sobre nada.

### Órfão é dívida OU capacidade esperando gesto

**O que separa as duas é ler o que a coisa FAZ, não contar chamadores.** Uma
camada inteira com teste e sem rota chegando nela é funcionalidade perdida, não
lixo. E **a capacidade esperando gesto pode NÃO servir ao gesto quando ele
chegar**: guardar código por "vai ser útil" aposta num desenho ainda não feito.

### A bancada que escreve o que a produção não escreve

Código que ninguém alcança e teste que arranja o próprio estado parecem
cobertura. Arranjo por uma porta que nenhum caminho de produção usa, ou bancada
que escreve uma coluna que a produção nunca escreve, deixa o verde sobre o
arranjo. **Monte o estado pelo caminho de produção**, não pela porta mais curta.

### O RACIOCÍNIO guardado envelhece como o código guardado

Docstring envelhece como `.md`, e ninguém relê a de função que não vai tocar —
ver "Comentários" no [CLAUDE.md da raiz](../CLAUDE.md); a sonda que caça isso
precisa do controle positivo de "O INSTRUMENTO MENTE COM CARA DE RESULTADO".

### Guarda de leiaute precisa do ESTADO, não só da cena

Ver "Um guarda só mede o que ele VISITA" no [CLAUDE.md da raiz](../CLAUDE.md).
**Texto escondido por CSS sai da árvore de acessibilidade**: um rótulo com
`display:none` não é lido nem pelo leitor de tela nem pelo `getByLabel`.

## Apagar código morto: as armadilhas do RECORTE

Quantos métodos exportados o `*Server` ainda tem se pergunta ao código:

```
grep -c "^func (s \*Server) [A-Z]" serve/api/*.go
```

O compilador pega três armadilhas: recortar função de UMA LINHA procurando `}`
na coluna zero come as vizinhas; o receptor colide com variável do destino;
`s.` dentro de string de formato vira recorte errado. **A quarta ele não pega:
a regra que só aquele teste guardava** — o método sem chamador some com o teste
que o exercitava. Antes de apagar, pergunte o que o teste afirmava, não se
alguém chama.

**Cena se testa pelo `WebRouter()`**, não pelo `Router()` da API JSON. Guarda
que varre rotas morre com o terreno; confira se a invariante mudou de casa (a
escrita de ficha por estranho é do `TestNoSheetWriteAcceptsAStranger`).

## Os pacotes que saíram do `api`

`campaign`, `account`, `search`, `book`, `sheet` e `creature` são do `domain/`;
`events`, da `infra/`. Para a próxima extração:

1. **Função pura copiada por causa de fronteira é lugar de defeito silencioso.**
   Copie o CORPO do original, nunca uma reescrita de memória — e prefira o
   pacote, que apaga a cópia e a razão de haver duas.
2. **O guarda de fronteira que mais importa é o do pacote MAIS IMPORTADO.** Se
   o `book` alcançar o `api`, toda cena que o importa alcança HTTP com o guarda
   de cada uma VERDE, porque cada um só olha os próprios imports. Vale para
   `sheet`, `creature` e `events`.
3. **A regra de PRODUTO não desce para a infraestrutura** — por isso o
   `account` ("a senha tem ao menos 8 caracteres") mora no `domain/`. Regra com
   consumidor de TELA e de API tem UMA frase, em pt-BR, morando com ela: duas
   frases quebram quando alguém mexe no limite.

**O critério de mover é a DEPENDÊNCIA, medida:** "isto carrega HTTP?" se
responde nos imports do arquivo, não no assunto dele.

## `web/ui`: o kit de apresentação, e o que ele NÃO pode saber

Botão, campo, moldura, rótulo de seção, caixa rolável, ícone e CASCA moram em
`serve/web/ui`, e é isso que permite cada cena ser um pacote. **O kit não
importa catálogo**: o livro, o verbete e o buscador moram em pacote próprio
(`serve/web/bookui`, `serve/web/finder`). **A casca RECEBE o que não pode
conhecer**, como campo de `ui.Page`:

- `Asset func(string) string` — endereço versionado dos estáticos (`go:embed`
  de outro pacote);
- `Overlays []templ.Component` — livro, verbete e buscador; a casca só reserva
  o lugar;
- `ColorMode` — claro ou escuro, lido do cookie `color_mode`; a casca o escreve
  no `<html data-mode>` e não sabe de cookie.

Quem preenche é o `WritePage` do `serve/api/scene_core.go`, o ÚNICO lugar que
monta página. Não ponha esses campos em cada `ui.Page{…}`.

**Renomear palavra COMUM** (`campo`, `tamanho`, `layout`) pede renomeador que
pule comentário e string, com o compilador de rede; o identificador hifenizado
(`data-nav-layout`) ele não pega — ache-o com `grep` depois.

### O `extra` do botão NÃO ganha da base, e quem decide é a FOLHA

`ButtonClasses(v, t, extra)` concatena, e o `Join` não é `tailwind-merge`: de
duas classes do mesmo eixo, vence a que a folha compilada declara por ÚLTIMO, e
um `extra` declarado ANTES da base perde calado. **Nenhum eixo mora na BASE se
um TAMANHO pode querer outro valor**: fonte e `gap` ficam no mapa de tamanhos;
a base tem forma, foco, transição, desabilitado. Hover de outra cor é VARIANTE
(`VariantGhostDanger`), não `extra`. Para conferir: `grep -bo '\.classe'` na
folha compilada, dos dois lados — a ordem do `class=` o navegador ignora.

## Dois sistemas de desenho, até a última tela migrar

A transição é POR ESCOPO: a cena migrada veste o **Tempestade** inteiro, a
outra fica no **grimório**. A cena escolhe com `ui.Page.System`, e o zero do
tipo é o grimório, para nenhuma tela mudar sem pedido. Vocabulário na seção H
do [GLOSSARY](../GLOSSARY.md).

- **Os tokens `--ds-*` só existem dentro de `.scene-ds`**
  (`serve/web/assets/src/ds.css`): na raiz, um com nome de token do grimório
  repintaria as telas não migradas.
- **Cada cor é UM token com os dois modos** (`light-dark()`), resolvido pelo
  `color-scheme` do elemento que o USA; o modo escolhido é atributo no
  `<html>`, não segundo bloco de tokens.
- **A tela não escolhe cor**: monta-se com as peças de `serve/web/ds` e escreve
  só leiaute (`grid`, `gap`, `px-4`; a escala do Tailwind já é a do sistema,
  base 4). Peça nova entra no kit e na `/sistema` ANTES de entrar numa tela.
- **Pisos do Tempestade: 12px de texto e 44×44 de alvo no toque**, medidos pelo
  e2e (`support/ds-floors.ts`, no laço do `appearance.spec.ts`) só na cena com
  `.scene-ds` — o caso anota o sistema, sem lista de cenas migradas. O guarda
  da Cinzel mede a outra metade.
- **Medidor que mire um escopo pelo nome mira os DOIS** até o grimório sair: só
  `.scene-grimorio`, só o `index.css` ou só o `class="…"` passa VERDE sem medir
  a cena nova.
- **Tela grande migra em PARALELO, no mesmo endereço.** Parte não migrada
  desenha um vazio que leva à antiga — nunca o grimório dentro da cena
  Tempestade. Migrada a última, a antiga é APAGADA, com os testes portados
  antes.
- **O fim da migração é apagar**, não converter: `.scene-grimorio`, tokens
  `grimorio-*`, Cinzel, Liberation, JetBrains Mono e `/grimorio` saem juntos.

## O anel de foco é UMA receita, e o repouso dela existe para não animar

A regra de foco mora na folha de cada sistema (`index.css` para
`.scene-grimorio`, `ds.css` para `.scene-ds`) e **não é layerada**: ganha de
todo utilitário do Tailwind, então `focus-visible:outline-*` numa `class=` não
faz efeito. O `TestNoHandwrittenFocusRing` impede a cópia.

O que muda o pixel é o REPOUSO: o `* { outline-color }` do shadcn no
`@layer base` é o ponto de PARTIDA do `transition-colors`/`transition-all` do
Tailwind v4, que **incluem `outline-*`** — em repouso nada pinta
(`outline-style: none`), mas a transição parte dali. O anel é ALCANÇADO em
150ms em vez de desenhado, e quem tabula na autorrepetição nunca o vê inteiro.
**O conserto é escrever o repouso igual ao destino**, no mesmo `*`. A
duplicação é deliberada, e o `e2e/tests/support/focus.ts`
(`expectOneFocusRing`) cobra que não se separe.

- **Para medir, pergunte se há transição** (`getAnimations()`) — ver "O
  INSTRUMENTO MENTE COM CARA DE RESULTADO" no [CLAUDE.md da raiz](../CLAUDE.md).
- **A opção de rádio é a única exceção, e é do RÓTULO.** A peça `ds-pick` (a
  forja e o "entrar na mesa") esconde o `<input>` e realça o `<label>` pela
  folha (`.ds-pick:has(:focus-visible)` no `ds.css`), que a regra global
  (`:is(a, button, input, …)`) não alcança. Uma `label:has(:focus-visible)`
  global desenharia DOIS anéis em todo rótulo que envolve campo VISÍVEL.
- **Guarda que aproxima um seletor mede outro seletor**: a isenção do anel
  COPIA o da folha (`[data-nav-region] :is(a, button, [data-nav-item])`), não
  "tudo dentro de `[data-nav-region]`".

## As notas numa janela própria, e o pacto que as torna únicas

`GET /campanhas/{campanha}/sessoes/{sessao}/notas` desenha o MESMO painel, sem
o mapa. **É CENA**, porque nó adotado por outro documento perde o Datastar (ver
"O nó que muda de DOCUMENTO"). Reusa `notesBody`, `modesRange` e
`autosaveState`; o que ela NÃO tem está decidido no cabeçalho do
`notes_window.templ`.

**As duas não podem estar abertas ao mesmo tempo**: ambas gravam
`sessions.notes` com autosave de 1,2s, e quem salva por último apaga o parágrafo
do outro, com as DUAS faixas dizendo "Salvo". O pacto é `localStorage` mais o
evento `storage` (chega a toda janela da origem sem objeto vivo), com chave
pelo ID DA SESSÃO.

- **Quem fecha a coluna é o ANÚNCIO da janela, não o clique**: com o pop-up
  bloqueado, a coluna fica, e o bloqueio é dito em PALAVRAS — `window.open`
  devolvendo nulo é a única pista.
- **`pagehide`, não `beforeunload`** (que pede confirmação em alguns caminhos);
  `unload` é pulado no cache de ida e volta.
- **A chave pode ficar PRESA** se a janela morrer sem `pagehide`; o botão chama
  `window.open` com o mesmo NOME, que acha a janela viva ou abre outra.

O `TestEveryAddressAPostWritesExistsInTheRouter` confere o `window.open` como
GET: perguntar por POST no mesmo caminho acharia a rota de SALVAR.

## Onde a coordenada de um gesto do tabuleiro viaja

**No CORPO, não no caminho**: `@post` com `{payload: …}`, calculado no clique —
não sinal, que viajaria em toda requisição. Um formato para o tabuleiro todo,
com chaves INGLESAS porque campo JSON é fronteira (ver "Idioma" no
[CLAUDE.md da raiz](../CLAUDE.md)):

```json
{"from":{"x":2,"y":2},"to":{"x":8,"y":5}}        // o traço e o retângulo
{"kind":"dificil","erase":true,"from":…,"to":…}  // com a espécie do pincel
{"shape":"esfera","size":"6","from":…,"to":…}    // o gabarito: origem e mira
{"from":{"x":7,"y":3}}                           // um lugar só: parada, marcador
{"delta":{"x":3,"y":-2},"marked_tokens":"a,b"}   // o arrasto do grupo marcado
```

Um tipo só, o `strokeBody`, para os gestos de pontos: um formato por gesto é
como nasce a terceira grafia do mesmo par de números.

### O que o corpo quebrado produz, e as DUAS famílias de recusa

O corpo quebrado é `{"from":{"X":undefined}}` — sinal indefinido no `payload`
manda `undefined`, que não é JSON —, e o servidor não pode presumir a origem.
As rotas de **só sinais** (`marcar-area`, `gabarito`, `regua`) recusam em
**400**; as de **comando** (terreno, retângulo, peça avulsa, grupo) em **200**
com a frase no `$command_error`, porque o Datastar DESCARTA remendo não-2xx. O
`TestEveryGestureThatReadsPointsRefusesABrokenBody` prende **a frase**, não o
status.

> **Corpo VAZIO não é recusa, por desenho**: `{}` vira (0,0), e prender isso
> exigiria ponteiro em todo campo. **Caso ancorado na origem não detecta um
> `from` que parou de ser lido** — todo caso sai da origem, e vários afirmam que
> (0,0) NÃO foi tocado.

### Quando o corpo JÁ É o formulário, o payload lista os sinais

O `payload` SUBSTITUI os sinais no corpo, mas pode carregá-los nomeando-os:

```js
@post('…/pecas/nova', {payload: {from: {x: cx, y: cy},
  new_token_name: $new_token_name, new_token_size: $new_token_size}})
```

**É um struct só, lido uma vez**: o `ReadSignals` copia `r.Body` inteiro, e um
segundo leitor pega vazio. A grafia a mais de cada sinal, que um `grep` de
`$nome` não distingue, é cobrada pelo
`TestEveryPayloadKeyMatchesTheSignalItReads`: chave que é nome de sinal carrega
aquele sinal (`new_token_size: $new_token_look` reprova; `kind: $tool` passa,
`kind` não é sinal) e toda chave acha uma tag `json:"chave"` — renomeada de um
lado só, ela cai no chão com 200 e valor-zero.

**A CAIXA conta**: o `encoding/json` ignora caixa **só sem correspondência
exata**, então `{X: cx}` contra `json:"x"` funciona por ACIDENTE. O fio é
minúsculo (`x`, `y`, como em `campaign_places` e `open_boards`), o guarda exige
a grafia exata, e **a tag GANHA do nome do campo**. Com a caixa errada o app
funciona e o Playwright passa: **risco latente não tem testemunha em execução, e
quem o prende é guarda de TEXTO.**

### DESLOCAMENTO é coordenada, e o guarda não sabia disso

**Coordenada é o número que vem do PONTEIRO, absoluto ou não**: o `delta` do
arrasto viaja no corpo, porque no caminho obrigaria a concatenar o endereço.

O `TestNoRouteCarriesACoordinateInThePath` pergunta ao ROTEADOR (`chi.Walk`),
que devolve o padrão montado — `base :=`, `r.Route` pai e filho, registro em
duas linhas. É lista de PERMITIDOS (`serve/api/testdata/route_params.txt`):
**parâmetro novo REPROVA até alguém escrever a linha** que declara que ele não
vem do ponteiro. O `TestNoAllowedRouteParamIsStale` reprova a linha que nenhuma
rota usa, porque permissão pendurada aprova o próximo nome igual.

### O PASSO da ficha FICA no caminho, e a razão não é a que estava escrita

`/personagens/{id}/vitais/{qual}/{passo}`, `/nivel/{classe}/{passo}` e
`/poderes/classe/{classe}/{escolha}/{valor}` ficam no caminho de propósito. A
razão NÃO é "botões disputariam um sinal" (cada um escreve o seu literal no
`payload`). **É o REDESENHO**: o `sheetCommand` lê os sinais UMA vez e os
entrega ao `s.Load(…)` que redesenha a cena (buscas, filtros, aumentos,
construtor de item); um `payload` `{step: -1}` os substituiria, e a tela
voltaria sem o filtro em uso. Coordenada vem do ponteiro; passo é literal que o
`templ` escreve num endereço constante por botão.

## Datastar: armadilhas que não deixam erro para trás

Nenhuma escreve no console (no máximo uma linha que ninguém olha), e o sintoma
de cada uma aponta para o lugar errado.

### `data-show` esconde TARDE: o nó pinta antes de o Datastar chegar

O `data-show` só é avaliado quando o runtime carrega; até lá um diálogo
`fixed inset-0 bg-black/60` cobre a janela e some. **Ponha
`style="display:none"` estático ao lado do `data-show`**: lido na análise do
HTML, antes de script, e trocado por vazio quando o Datastar mostra (não é o
`data-attr:style`, que reescreve a cada avaliação). O
`TestNoDataShowNodeIsBornVisible` varre todo `.templ` — sobreposição e
`data-show="$x"` de sinal puro, sem negação nem operador — e erra para o lado
seguro.

### `ReadSignals` vem ANTES do `NewSSE`, e a ordem errada passa VERDE no teste

O `NewSSE` ASSUME a resposta e fecha o corpo do pedido: `ReadSignals` depois
dele chega sem sinal. **O `httptest.NewRequest` não reproduz esse ciclo**, então
o teste de handler passa e só o navegador quebra; num `GET` não morde, e nasce
quando o gesto vira `POST`. **O corpo se lê UMA vez** — o segundo leitor pega
vazio, sem erro.

### `data-show` + `data-attr:style` no MESMO nó CONGELA a aba

O `data-show` escreve `el.style.display`; o `data-attr:style` reescreve o
`style` INTEIRO, apagando-o, e o outro o põe de novo: laço no renderizador, aba
morta, sem console e sem navegar para fora. **Quem ESCONDE é um nó, quem
POSICIONA é outro** — o `TestNoNodeHasDataShowAndDataAttrStyleTogether` recusa
a combinação no HTML servido da mesa.

### O sinal é um PROXY: ler um índice que não existe o CRIA

`$lista[0]` registra o caminho e o cria vazio — `$ruler_points[i]` sobre
índices reservados enche o sinal de strings vazias. **Constante NÃO resolve**
(continua proxy); **COPIAR resolve**: `const p = [...$ruler_points]`. O
`TestNoExpressionIndexesTheListSignal` cobra pelo que PODE vir depois de
`$lista` (`=` ou `]`), não por uma forma errada conhecida.

### Sequência de comandos NÃO cabe num ternário

`evt.shiftKey ? (stmt; stmt) : (stmt; stmt)` é erro de SINTAXE, engolido: o
handler inteiro vira nada, inclusive o que já funcionava. Use
`if (…) { … } else { … }`. E **`setPointerCapture` vai por ÚLTIMO**: ele LANÇA
com o ponteiro inativo, e no meio engole o resto do gesto.

### O servidor escrevendo num SINAL pelo stream: uma vez, nunca a cada quadro

Onde a decisão é do navegador, o stream da mesa escreve num SINAL
(`PatchSignals`): `surface` (o "mostrar à mesa") e `sheet_version` (abaixo). O
puxão vive até a pessoa trocar de aba, e mandá-lo em TODO quadro devolve ao
mapa, para sempre, quem tenta voltar para a Mesa. **Remendo de HTML é
idempotente, remendo de sinal não é**: a memória do que já foi empurrado é da
CONEXÃO (uma variável do laço do stream) — duas abas da mesma pessoa merecem um
empurrão cada.

### O `@post` que redesenha a CENA precisa carregar o estado que está na URL

O handler descobre o que desenhar lendo a própria requisição, e a URL da PÁGINA
(`?tab=` na ficha, `?aba=` e `?entrada=` no acervo do mestre) **não vai junto**:
a cena volta na PRIMEIRA aba sem nada falhar. **O servidor escreve o `?` no
comando**, porque sabe o estado ao renderizar: o `sheetPost` monta todo `@post`
da ficha, e o `TestNoSheetCommandLosesTheTab` falha no comando de qualquer aba
que sair sem ele. Sinal do cliente resolveria e some no F5.

### Dois pedidos de UM gesto: quem CHEGA por último manda

**O clique também foca.** Nó que pede ao servidor no foco e no clique manda dois
pedidos que disputam os sinais da mesma cena, na ordem de CHEGADA — some na
bancada rápida e aparece no CI. **Pedido por FOCO é afordância de TECLADO** e
pede `el.matches(':focus-visible') && (…)`: o clique dá `false`; o Tab e o foco
programático do driver de setas, `true`. O
`TestNoFocusAsksTheServerWithoutAKeyboardGuard` CAMINHA a árvore inteira de
`.templ` (enumerar cena deixaria a próxima de fora) com piso de arquivos
VISITADOS, para mudar de pacote não encolher a varredura.

### Resposta que não é 2xx: o remendo é DESCARTADO e a recusa some

O Datastar não aplica remendo de resposta de erro: `http.Error(w, msg, 400)`
deixa a tela como estava, com uma linha no console. **Numa cena, a recusa é
CONTEÚDO**: 200 com a cena redesenhada e a frase num `role="alert"`
(`sheetCommand` + `sheetui.View.Refusal`), e **os testes afirmam a FRASE**
(`sceneRefusal`). A API JSON segue com os status dela.

**O EXTREMO é o 404**: o handler nunca roda. Endereço montado em metades
(`"initiative/"+route` no helper, o verbo no chamador) é invisível ao `grep` —
**o endereço só existe RESOLVIDO**. O
`TestEveryAddressAPostWritesExistsInTheRouter` RENDERIZA as cenas de
`scenesThatWriteAddresses` (cujo cabeçalho declara a enumeração como limite),
extrai todo `@post`/`@get`/`window.open` e pergunta ao chi com `Match` — não
com pedido, porque um 404 do HANDLER ("personagem não existe") pareceria rota
faltando. Para o próximo guarda de cena:

- **Ele precisa do ESTADO vivo**: o `‹` da mesa nasce `disabled` quando
  `CanAdvance` é falso (`st.CountsRounds() && len(st.Initiative) > 0`), e numa
  bancada recém-montada o endereço morto não está no HTML.
- **Expressão que o extrator não resolve REPROVA**, em vez de sair da conta.

### `contentType: 'form'` valida o formulário ANTES de mandar

`@post(url, {contentType: 'form'})` manda o `<form>` mais próximo, não os
sinais: os controles já são o estado, e uma cópia num `data-signals` só cria a
pergunta de qual vale (na forja, redesenho e `submit` leem o
MESMO formulário pelo `r.ParseForm()`). O cliente roda antes
`if (!form.noValidate && !form.checkValidity())`, e um `required` em branco
chama `reportValidity()` e **não manda o pedido**. `novalidate` desligaria
também o `submit`: **o campo não é `required` no HTML, e a recusa é do
servidor.** Num formulário remendado, validação nativa briga com o redesenho.

### `data-on-signal-patch` SEM filtro escuta o remendo dos OUTROS sinais

A ficha na sessão não é região do stream (recomputá-la a cada tique seria o
gasto mais caro da página): o servidor manda o carimbo `sheet_version` e o
CLIENTE repede a ficha. **O `data-on-signal-patch` dispara em QUALQUER remendo
de sinal** — um puxão (`surface`) repediria a ficha —, e o filtro mora num
atributo IRMÃO:

```
data-on-signal-patch-filter="{include: /^sheet_version$/}"
data-on-signal-patch="… @get('/personagens/13?tab=' + $sheet_tab + '&embutida=1')"
```

- **UMA vez por mudança**, com a memória na CONEXÃO (`announceSheetChange`).
- **Quem diz "mudou" é um EVENTO**, não comparar o carimbo a cada tique (ver
  "O barramento de eventos").
- **O interesse é por PERSONAGEM, não por sessão**: a ficha pode estar em duas
  mesas, e o `session.Store` obrigaria quem PUBLICA a saber em quais. Quem não
  tem ficha na mesa não pede interesse.
- **Quem publica é o GATEWAY** (`characterChanged`), não cada comando: a linha
  esquecida seria uma ficha que não atualiza só naquele gesto.
- **O servidor NÃO sabe a aba** (o stream abriu antes do clique): o clique
  escreve `$sheet_tab`, e o repedido a concatena.

### O morph REMOVE o nó que o seu JS pendurou na linha

**O reconciliador remove todo filho que não veio do servidor.** Véu, brilho ou
espaçador do JS dentro da subárvore remendada some no morph do próprio gesto (o
`MutationObserver` roda como MICROTAREFA, no meio dele) ou no do remendo
SEGUINTE; `requestAnimationFrame` só ganha do primeiro, e o `data-ignore-morph`
exige o atributo nos DOIS lados. A animação foi PEDIDA, então guarda que conte
`el.animate` e sonda no instante do `animate()` ficam verdes.

**O nó mora FORA da subárvore remendada**: `position:fixed` sobre o retângulo
do alvo, filho do `<body>` (`lib/turn-juice.ts`), ao preço de não seguir um
alvo que ande. Animação no PRÓPRIO nó (escala, sombra, opacidade) dispensa
isso: o morph reusa o nó. E **o morph não toca atributo que já bate**: um
observador de `aria-current` só acorda quando o valor MUDA — não escreva guarda
contra repetição.

### O nó que muda de DOCUMENTO perde o Datastar, e continua desenhado

Uma janela própria (Document Picture-in-Picture, `<iframe>`) que pega o nó vivo
com `appendChild` leva classes, texto e `data-*`, **mas não a reatividade**: o
runtime observa o documento de origem. Sem exceção, sem console, tela perfeita.
**Superfície que vai para janela própria é CENA com endereço**, com o runtime
dela (ver "As notas numa janela própria").

## O evento de ponteiro SINTÉTICO destrói o que ele mede

`element.dispatchEvent(new PointerEvent(...))` com `pointerId` inventado faz o
`setPointerCapture` lançar `NotFoundError`, que leva junto as escritas de sinal
do handler: o gesto parece MORTO estando inteiro. **Gesto de ponteiro se mede
com entrada REAL** — o Playwright (`page.mouse.down`,
`page.keyboard.down('Shift')`). O Chrome MCP NÃO aplica `modifiers` ao arrasto
(`shiftKey` chega `false`).

## Remendo em nó COMPARTILHADO exige limpeza no gesto que troca de item

A cena desenha **todos** os itens e alterna com `data-show` — seguro enquanto
cada item vem inteiro do servidor. Escrita DEPOIS da renderização num nó que os
itens dividem vaza: o diálogo de redefinição de senha é UM para todas as linhas
de jogador, e o link de um apareceria sob o nome do outro. **Quem LIMPA é o
gesto que TROCA de item**, que sabe que houve um anterior; quem gera não sabe
que haverá um próximo.

```
data-on:click="$target_id = el.dataset.id; $target_name = el.dataset.nome;
               $copied = '';
               document.getElementById('reset-link').innerHTML = '';
               $reset_dialog.showModal()"
```

A cada cena nova: existe nó compartilhado que recebe escrita depois da
renderização? Se existe, quem troca de item tem de apagá-lo.
