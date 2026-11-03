# Motor Go

Estende ou sobrepõe o [CLAUDE.md da raiz](../CLAUDE.md). `engine-go` é o app
inteiro, num processo só: API na :3001, motor de regras, CENAS em `.templ` com
Datastar, folha e ilhas de JS (`serve/web/assets/src`) e kit de apresentação
(`serve/web/ui`).

## Antes de mexer, leia o guia do assunto

Este arquivo é o índice. **As regras moram em `docs/`, e nada as carrega sozinho:
abra o guia ANTES de escrever.** A maior parte do que está neles não deixa erro para
trás, então não dá para descobrir depois.

| vai mexer em | leia | o que costuma morder |
|---|---|---|
| regra do jogo, ficha, `domain/engine`, `domain/board`, `domain/live`, `app/combat` | [engine.md](docs/engine.md) | regra nova nasce em ECS, ordem dos sistemas, parcelas do dano, metro e quadrado |
| catálogo, `domain/catalog`, `domain/book`, `cmd/audit`, `cmd/seed` | [catalog.md](docs/catalog.md) | `transcriptfmt`, emendas da mesa, validação de escolha, rodar o auditor |
| `.templ`, cena, `serve/web`, CSS | [screens.md](docs/screens.md) | `templ generate`, classe nova, ícone, porta da cena |
| `data-*`, `@post`, sinal, remendo, gesto do tabuleiro | [datastar.md](docs/datastar.md) | recusa em 4xx some, `ReadSignals` antes do `NewSSE`, o morph, coordenada no corpo |
| migração, `query.sql`, `infra/`, `cmd/api`, fluxo ao vivo | [infra.md](docs/infra.md) | comentário de SQL é ASCII, coluna nova vira tabela, SSE e gzip |
| teste, `convention/`, apagar código | [tests.md](docs/tests.md) | bancada, guarda novo, a regra que só o teste apagado guardava |

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
cada cena e as regras de tela nenhuma — ver "Como uma CENA é construída", em [screens.md](docs/screens.md).

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
