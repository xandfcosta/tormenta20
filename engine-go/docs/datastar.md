# Datastar e os gestos

As armadilhas que não deixam erro, e por onde a coordenada de um gesto viaja. Faz parte do [guia do engine](../CLAUDE.md).

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

**No `data-on-signal-patch`, a ÚLTIMA instrução é expressão, nunca `if`**: medido
na ALE-608, `const r = …; if (r) { @post(…) }` ali é `SyntaxError: Unexpected
token 'if'` no console e o ouvinte nasce morto, enquanto o mesmo texto no
`data-init` do lado roda. Escreva o fim como `r && @post(…)` (ver o
`sendsFraming`).

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

**Com um diálogo aberto, a recusa aparece dentro dele**: a da página fica atrás do modal, e o gesto
parece não ter feito nada. Na ficha, todo diálogo usa o `sheetDialog`, e o
`TestEverySheetDialogShowsTheRefusal` reprova com o id do que usar o `ds.Dialog` direto. O Escolher é
a exceção: cada passo desenha a dele, acima do que rola.

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

- **A ficha SOLTA tem o fluxo dela** (`/personagens/{id}/ao-vivo`, o `sheetLive`): o mesmo carimbo, e
  o ouvido mora DENTRO da cena (`StandaloneRefetch`), porque é a cena que sabe a aba de agora.
- **UMA vez por mudança**, com a memória na CONEXÃO (`announceSheetChange`).
- **Quem diz "mudou" é um EVENTO**, não comparar o carimbo a cada tique (ver
  "O barramento de eventos" (infra.md)).
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
dela (ver o cabeçalho do `notes_page.templ`).

## Onde a coordenada de um gesto do tabuleiro viaja

**No CORPO, não no caminho**: `@post` com `{payload: …}`, calculado no clique —
não sinal, que viajaria em toda requisição. Um formato para o tabuleiro todo,
com chaves INGLESAS porque campo JSON é fronteira (ver "Idioma" no
[CLAUDE.md da raiz](../../CLAUDE.md)):

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

## Os nomes de sinal e de id

### Sinal: `snake_case`, inglês, uma grafia em todos os canais

A forma foi MEDIDA no navegador, e o mecanismo É a regra:

- **camelCase quebra em silêncio.** Nome de atributo é minusculado pelo parser de
  HTML: `data-bind:buscaCriatura` chega como `data-bind:buscacriatura` e o
  Datastar liga um sinal NOVO, deixando o declarado intocado. O fio leva os DOIS
  e o servidor lê o errado — sem erro em lugar nenhum.
- **kebab-case vira outra coisa.** O Datastar transforma `-[a-z]` em maiúscula
  por padrão, então `data-bind:creature-search` liga um sinal em camelCase: duas
  grafias para um conceito, que é a raiz do defeito acima.
- **`_` atravessa intacto** — não há caixa para perder, e o `-` é o único
  caractere que a transformação toca.

Prefixo por FAMÍLIA quando o sinal pertence a um grupo (`template_*`, `ruler_*`),
para o `grep` achar a família inteira.

Nenhum teste de Go pega isso — a minusculação acontece no NAVEGADOR, e o HTML
servido ainda tem a caixa que o autor escreveu. A garantia desceu para o TEXTO do
atributo: `TestNoDatastarAttributeKeyCarriesUppercase` varre a forma, e
`TestNoNewSignalBreaksTheNamingStandard` varre com linha de base VAZIA.

### Renomear um sinal são SETE canais, e nada liga um ao outro

Isto é checklist, não história: renomear é ato humano, nenhum guarda o cobre, e
**cinco dos sete são invisíveis para um `grep` de `$nome`**.

| # | canal | como aparece |
|---|---|---|
| 1 | expressão | `$creature_search` |
| 2 | chave de atributo | `data-bind:creature_search` |
| 3 | **valor** de atributo | `data-ref="delete_dialog"` |
| 4 | declaração | `data-signals="{…}"`, e as strings montadas em Go |
| 5 | tag JSON do servidor | `json:"creature_search"` |
| 6 | filtro de remendo | `data-on-signal-patch-filter="{include: /^sheet_version$/}"` |
| 7 | **argumento de string** | `toggleBadge(aba, key, …)`, que monta `"$" + key` |

Os canais 3 e 7 deixaram quatro diálogos sem abrir com a suíte de Go inteira
verde: **a expressão passa a ler `undefined`, que em JavaScript não é erro** — é
o gesto não fazer nada. O canal 4 tem a armadilha inversa: `undefined != ''` é
VERDADEIRO, então um `<p>` que devia nascer escondido nasce mostrado.

**Nem toda tag JSON é sinal** — leia o diff das tags antes de aceitar uma
varredura. E há um oitavo leitor que não é canal de escrita: a **constante**
(`const brushSignal = "pincelando"`, lida como `"$" + brushSignal`), que metade
das vezes mora num bloco `const (…)` com a palavra na linha de cima.

Quem fecha: `TestEverySignalDeclaredByValueHasAReader` (3),
`TestEverySignalTheTableDeclaresHasAReader` (4), e a catraca para a FORMA. **O 7
não tem guarda** — hoje é um sítio só, o `toggleBadge` do acervo do mestre, e
está nomeado aqui por isso.

### O id de elemento tem NOVE canais

`id="x"` · `getElementById('x')` · `aria-labelledby` · `aria-describedby` ·
`for=` · `popovertarget` · `commandfor` · `querySelector('#x')` · e
**`el.id === 'x'`**, que foi o que escapou: o renome passou por cima, o Enter
deixou de abrir o primeiro achado, e nada mais mudou na tela.

Quem cobra é o `TestEveryReferencedElementIdExists` — todo id APONTADO existe em
algum `.templ`, escrito como `id="x"` ou entregue a uma peça do kit que o
escreve (`ds.Dialog("x", …)`, ou o `sheetDialog` da ficha). Ele não cobra o contrário: um id pode existir só
para o CSS.
