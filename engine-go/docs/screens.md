# As telas

`templ`, como uma cena é construída, o kit e o sistema de desenho. Faz parte do [guia do engine](../CLAUDE.md).

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
  da primeira`, e o da reconexão, `live-stream-reconnects.spec.ts` — e a leitura
  do diff das FONTES, pelo `.js.map` das duas tags: a reconexão mudou na 1.0.5
  com a suíte inteira verde.
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
- **A CLASSE DE ESCOPO não é tinta**: `scene-ds` é condição dos tokens, e sem ela
  o app sai sem cor. Cobre o `TestEveryScopeClassExistsInTheStylesheet`, que lê só
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

## `web/ui`: o kit de apresentação, e o que ele NÃO pode saber

Rótulo de seção, caixa rolável, ícone e CASCA moram em `serve/web/ui`, e é isso que
permite cada cena ser um pacote. As peças de tela são do Tempestade, em `serve/web/ds`. **O kit não
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

## O sistema de desenho

Toda cena veste o **Tempestade**, e a casca põe `.scene-ds` nela. Vocabulário na seção H
do [GLOSSARY](../../GLOSSARY.md).

- **Os tokens `--ds-*` só existem dentro de `.scene-ds`** (`serve/web/assets/src/ds.css`).
- **Cada cor é UM token com os dois modos** (`light-dark()`), resolvido pelo
  `color-scheme` do elemento que o USA; o modo escolhido é atributo no
  `<html>`, não segundo bloco de tokens.
- **A tela não escolhe cor**: monta-se com as peças de `serve/web/ds` e escreve
  só leiaute (`grid`, `gap`, `px-4`; a escala do Tailwind já é a do sistema,
  base 4). Peça nova entra no kit e na `/sistema` ANTES de entrar numa tela.
- **Botão de copiar passa por `ui.CopyThen`**, nunca por `navigator.clipboard`: a API
  só existe em HTTPS e `localhost`, e o app é servido em HTTP na rede da mesa. O e2e
  roda em `localhost`, onde ela existe, e por isso não vê a chamada direta quebrar.
- **Pisos do Tempestade: 12px de texto e 44×44 de alvo no toque**, medidos pelo
  e2e (`support/ds-floors.ts`, no laço do `appearance.spec.ts`). O elo citado no meio de uma frase é a
  exceção `Inline` da 2.5.5, e não conta; alvo sozinho numa linha conta, mesmo sendo elo.

## O anel de foco é UMA receita, e o repouso dela existe para não animar

A regra de foco mora no `ds.css` e **não é layerada**: ganha de
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
  INSTRUMENTO MENTE COM CARA DE RESULTADO" em [docs/testing.md](../../docs/testing.md).
- **A opção de rádio é a única exceção, e é do RÓTULO.** A peça `ds-pick` (a
  forja e o "entrar na mesa") esconde o `<input>` e realça o `<label>` pela
  folha (`.ds-pick:has(:focus-visible)` no `ds.css`), que a regra global
  (`:is(a, button, input, …)`) não alcança. Uma `label:has(:focus-visible)`
  global desenharia DOIS anéis em todo rótulo que envolve campo VISÍVEL.
- **Guarda que aproxima um seletor mede outro seletor**: a isenção do anel
  COPIA o da folha (`[data-nav-region] :is(a, button, [data-nav-item])`), não
  "tudo dentro de `[data-nav-region]`".

## O guarda de ÍCONE varre as CENAS, e o `switch` do gerado não tem `default`

Nome que ninguém gerou rende um `<svg>` VAZIO, sem erro. O
`TestEveryRequestedIconExistsInTheGeneratedFile` varre `serve/` inteiro, casa
`@icone("X")` e `@ui.Icon("X")` e tem DENOMINADOR. **Ícone novo entra na lista
do `scripts/gen-icons-templ.mjs` e o gerador roda — nunca à mão no `icons.templ`
gerado.**

