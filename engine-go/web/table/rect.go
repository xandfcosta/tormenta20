package table

import "fmt"

// O RETÂNGULO no tabuleiro (ALE-203, item 10 do dono).
//
// "Não temos ferramenta de seleção em área." A escolha do dono foi que o
// retângulo faz DUAS coisas, e quem decide é a ferramenta na mão: com um pincel
// ou a borracha ele ENCHE de terreno; com "mover" ele MARCA peças.
//
// # Por que `Shift` no pincel e arrasto puro no mover
//
// O arrasto já está ocupado, e de formas diferentes:
//
//   - com o PINCEL, arrastar pinta à mão livre (é o traço da fatia 3);
//   - com a MÃO, arrastar move a janela;
//   - com MOVER, arrastar em cima de uma peça move a peça — mas arrastar no
//     VAZIO não faz nada.
//
// Então marcar peças cabe no arrasto vazio de graça, e o retângulo de terreno
// precisa de um gesto a mais. `Shift` é o modificador de "retângulo" em todo
// editor de desenho, e ele não colide com nada aqui.
//
// # O DESENHO do laço mora em sinais
//
// Como a régua e o gabarito: um retângulo remendado dentro da região do mapa
// seria apagado pelo quadro seguinte do stream, no meio do gesto. Ele é um nó só,
// posicionado por uma expressão — e por isso não custa nó por casa.

// Os sinais do laço. `retangulando` é o modo em curso e o valor É o modo, como
// o `$tool` e o `$pincelando`: vazio (parado), `terreno` ou `pecas`.
const (
	sinalDoRetangulo   = "rect_mode"
	sinalDoRetanguloDe = "rect_from" // "x/y" do canto onde o dedo desceu
)

const (
	retanguloDeTerreno = "terreno"
	retanguloDePecas   = "pecas"
)

// rectSignals entram na semente da Mesa.
//
// O canto de ORIGEM guarda-se como `"x/y"` pelo mesmo motivo do `$ultimacasa`: é
// o formato do CAMINHO, e converter na hora de montar a rota foi exatamente onde
// a vírgula produziu um 404 mudo.
var rectSignals = fmt.Sprintf(
	"%s: '', %s: '', rect_from_x: 0, rect_from_y: 0, rect_to_x: 0, rect_to_y: 0",
	sinalDoRetangulo, sinalDoRetanguloDe)

// takesRect abre o laço no canto em que o dedo desceu.
//
// Guarda o canto DUAS vezes — como texto de caminho e como par de números — e
// isso não é redundância: o texto vai para a rota no fim, e os números desenham o
// laço a cada quadro. Derivar um do outro na expressão custaria um `split` por
// movimento do ponteiro.
func takesRect(modo string) string {
	return fmt.Sprintf(
		"evt.preventDefault(); const cx = %s, cy = %s; "+
			"$%s = %q; $%s = cx + '/' + cy; "+
			"$rect_from_x = cx; $rect_from_y = cy; $rect_to_x = cx; $rect_to_y = cy; "+
			"evt.currentTarget.setPointerCapture(evt.pointerId)",
		clicouEmX, clicouEmY, sinalDoRetangulo, modo, sinalDoRetanguloDe,
	)
}

// followsRect leva o canto oposto atrás do dedo.
//
// NÃO fala com o servidor: o laço é geometria, e o resultado só é pedido quando o
// dedo solta. É a diferença para o pincel — lá cada casa cruzada é uma ação, aqui
// o gesto inteiro é UMA.
func followsRect(modo string) string {
	return fmt.Sprintf(
		"$%s === %q && ($rect_to_x = %s, $rect_to_y = %s)",
		sinalDoRetangulo, modo, clicouEmX, clicouEmY,
	)
}

// dropTerrainRect fecha o laço e manda encher.
//
// O `$tool` escolhe a rota: a borracha tem caminho sem espécie, que é o
// conserto que a fatia 1 fez e que não pode se perder aqui.
func dropTerrainRect(v BoardView) string {
	return fmt.Sprintf(
		"if ($%s !== %q) return; const de = $%s.split('/').map(Number); "+
			"const cantos = {from: {X: de[0], Y: de[1]}, to: {X: $rect_to_x, Y: $rect_to_y}}; "+
			"$%s = ''; "+
			"return $tool === %q "+
			"? @post('%s/terreno/limpar/retangulo', {payload: cantos}) "+
			": @post('%s/terreno/retangulo', {payload: {...cantos, kind: $tool}})",
		sinalDoRetangulo, retanguloDeTerreno, sinalDoRetanguloDe,
		sinalDoRetangulo,
		EraserTool,
		v.Base, v.Base,
	)
}

// openIsLasso mostra o desenho do laço.
func openIsLasso() string {
	return fmt.Sprintf("$%s !== ''", sinalDoRetangulo)
}

// lassoStyle põe o retângulo na tela, em QUADRADOS do plano.
//
// `min` e `+1` porque o retângulo inclui as duas casas das pontas e o dedo pode
// arrastar para qualquer lado — a mesma regra que o `RectangleSquares` aplica do
// lado do servidor, e é de propósito que as duas existam: esta desenha o que
// aquela vai fazer, e uma promessa que não bate com o resultado é pior que não
// desenhar nada.
const lassoStyle = "`left: ${Math.min($rect_from_x, $rect_to_x) * $square}px; " +
	"top: ${Math.min($rect_from_y, $rect_to_y) * $square}px; " +
	"width: ${(Math.abs($rect_to_x - $rect_from_x) + 1) * $square}px; " +
	"height: ${(Math.abs($rect_to_y - $rect_from_y) + 1) * $square}px`"

// sinalDoCliqueEngolido diz ao `click` que o gesto anterior foi um ARRASTO.
//
// Ele existe porque o navegador dispara `click` depois de um `pointerdown` +
// `pointerup` no mesmo elemento, INCLUSIVE quando o dedo andou entre os dois. Sem
// ele, terminar um laço em cima da camada de repouso também MOVERIA a peça da
// vez para onde o laço terminou — o mestre marca um grupo e a peça do turno anda
// junto, sem ninguém ter pedido.
const sinalDoCliqueEngolido = "swallow_click"

// dropTokensRect fecha o laço e pergunta ao servidor quem ele pegou.
//
// Só PERGUNTA: marcar não muta a cena, e a resposta é do tamanho de um sinal.
//
// O laço que NÃO ANDOU (mesmo canto nas duas pontas) é um clique, não um laço, e
// segue o caminho do clique — é assim que a mesma camada serve aos dois gestos.
func dropTokensRect(v BoardView) string {
	return fmt.Sprintf(
		"if ($%s !== %q) return; const ate = $rect_to_x + '/' + $rect_to_y, de = $%s; "+
			"$%s = ''; if (de === ate) return; $%s = true; "+
			"const canto = de.split('/').map(Number); "+
			"return @post('%s/marcar-area', {payload: {from: {X: canto[0], Y: canto[1]}, "+
			"to: {X: $rect_to_x, Y: $rect_to_y}}})",
		sinalDoRetangulo, retanguloDePecas, sinalDoRetanguloDe,
		sinalDoRetangulo, sinalDoCliqueEngolido,
		v.Base,
	)
}

// markedIsToken é a pergunta que acende o anel, e ela é feita UMA VEZ POR
// PEÇA na cena.
//
// `,id,` com vírgulas nas pontas e não um `includes(id)` cru: sem elas, marcar a
// peça `abc` acenderia também a `abcd`. É a armadilha clássica de lista em
// string, e ela aparece exatamente no dia em que dois ids compartilham prefixo.
func markedIsToken(id string) string {
	return fmt.Sprintf("(',' + $%s + ',').includes(%q)", markedTokensSignal, ","+id+",")
}

// partyPhrase é o que a barra diz, com o plural certo.
//
// A frase INTEIRA numa expressão só, e não um número num `data-text` ao lado de
// um texto fixo: "1 peças marcadas" apareceu na tela na primeira medição, e a
// única forma de a palavra acompanhar o número é ela estar na mesma conta.
var partyPhrase = fmt.Sprintf(
	"(() => { const n = $%s.split(',').filter(Boolean).length; "+
		"return n + (n === 1 ? ' peça marcada' : ' peças marcadas') "+
		"+ ' · arraste qualquer uma para mover o grupo' })()",
	markedTokensSignal)

// hasMarkedParty mostra a barra do grupo.
var hasMarkedParty = fmt.Sprintf("$%s !== ''", markedTokensSignal)

// unmarkParty desmarca tudo.
var unmarkParty = fmt.Sprintf("$%s = ''", markedTokensSignal)

// ── ARRASTAR O GRUPO ─────────────────────────────────────────────────────────
//
// Com peças marcadas, arrastar QUALQUER UMA delas move todas pelo mesmo delta.
// É o gesto que todo editor faz e o motivo inteiro de marcar: chegou uma horda
// de seis zumbis e reposicioná-los hoje custa seis arrastos.

// partyTakes começa o arrasto, e SÓ se a peça estiver marcada.
//
// Sem a guarda, arrastar uma peça qualquer com um grupo marcado em outro canto
// do mapa moveria o grupo distante — o gesto agiria sobre o que a pessoa não
// está olhando, que é a pior classe de surpresa num tabuleiro.
func partyTakes(id string) string {
	return fmt.Sprintf("if (!(%s)) return; %s", markedIsToken(id), startsTheDrag(dragsTheParty))
}

// dropParty converte o deslocamento em QUADRADOS e move todas.
//
// O arredondamento é o mesmo do arrasto de uma peça só (para o quadrado mais
// próximo, e não para baixo), porque o gesto é o mesmo gesto — o que muda é
// quantas peças ele leva.
func dropParty(v BoardView) string {
	return fmt.Sprintf(
		"if ($dragging === '%s') { "+
			"const dx = Math.round($drag_x / $square), dy = Math.round($drag_y / $square); "+
			"$dragging = ''; $drag_x = 0; $drag_y = 0; "+
			"if (dx || dy) @post('%s/grupo/mover', {payload: {delta: {X: dx, Y: dy}, marked_tokens: $marked_tokens}}) }",
		dragsTheParty, v.Base,
	)
}

// tokenStyling junta as duas marcas que a peça pode vestir.
//
// UM `data-class` só porque atributo repetido não existe: o navegador guarda o
// primeiro e descarta o segundo, e a marca do grupo nasceria morta — é a mesma
// armadilha do `data-on:keydown__window` duplicado que a fatia 2 registrou.
func tokenStyling(id string, movesItself bool) string {
	marcada := fmt.Sprintf("'board-token-marked': %s", markedIsToken(id))
	if !movesItself {
		return "{" + marcada + "}"
	}
	// O ID e não o literal `'peca'` (ALE-299): com o literal, a única peça que
	// vestia a classe era a `ArrastaAPeca`, então no rascunho pegar o Beta fazia
	// o ALFA correr atrás do dedo. A classe segue quem o gesto marcou.
	return fmt.Sprintf("{'board-dragging': $dragging === '%s', %s}", id, marcada)
}

// brushGesture decide entre TRAÇO e RETÂNGULO no `pointerdown`.
//
// A decisão é no `pointerdown` e vale para o gesto inteiro, como o modo do
// pincel: soltar o `Shift` no meio do arrasto não pode trocar o que o gesto está
// fazendo — o dedo já está a caminho de um canto.
//
// `if/else` e NÃO um ternário, e isto é conserto de um defeito MUDO. Os dois
// ramos são SEQUÊNCIAS DE COMANDOS (`preventDefault(); $sinal = …;
// setPointerCapture(…)`), e sequência de comandos entre parênteses é erro de
// SINTAXE em JavaScript. O Datastar engoliu o erro de parse e o `pointerdown`
// inteiro virou nada — não só o retângulo: o pincel à mão livre, que funcionava,
// parou junto. Medido: o `pointerdown` chegava ao elemento e nenhuma requisição
// saía, sem uma linha no console.
func brushGesture(v BoardView, modoFixo string) string {
	return fmt.Sprintf("if (evt.shiftKey) { %s } else { %s }",
		takesRect(retanguloDeTerreno), takesBrush(v, modoFixo))
}

// restLayerName diz os dois gestos que ela aceita.
//
// O nome acessível é onde o gesto de ARRASTO fica descoberto: ele não tem ícone
// nem botão, e quem navega por teclado não tem outro lugar para achá-lo.
func restLayerName(v BoardView) string {
	if v.AlvoDoMovimento == "" {
		return "Marcar peças — arraste um retângulo em volta delas"
	}
	if !v.Mestre {
		return "Mover " + v.RotuloDoAlvo + " — escolha a casa"
	}
	return "Mover " + v.RotuloDoAlvo + " — escolha a casa, ou arraste para marcar um grupo"
}

// clickRest é o clique da camada, com o ENGOLE na frente.
//
// Sem alvo de movimento não há o que o clique faça, e a expressão fica só com o
// engole — que continua precisando existir, porque o `click` vem do mesmo jeito
// depois de um laço.
func clickRest(v BoardView) string {
	engole := fmt.Sprintf("if ($%s) { $%s = false; return }", sinalDoCliqueEngolido, sinalDoCliqueEngolido)
	if v.AlvoDoMovimento == "" {
		return engole
	}
	return engole + "; " + clickedPointStop(v)
}
