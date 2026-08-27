package api

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
// o `$ferramenta` e o `$pincelando`: vazio (parado), `terreno` ou `pecas`.
const (
	sinalDoRetangulo   = "retangulando"
	sinalDoRetanguloDe = "retangulode" // "x/y" do canto onde o dedo desceu
)

const (
	retanguloDeTerreno = "terreno"
	retanguloDePecas   = "pecas"
)

// osSinaisDoRetangulo entram na semente da Mesa.
//
// O canto de ORIGEM guarda-se como `"x/y"` pelo mesmo motivo do `$ultimacasa`: é
// o formato do CAMINHO, e converter na hora de montar a rota foi exatamente onde
// a vírgula produziu um 404 mudo.
var osSinaisDoRetangulo = fmt.Sprintf(
	"%s: '', %s: '', retangulodex: 0, retangulodey: 0, retanguloatex: 0, retanguloatey: 0",
	sinalDoRetangulo, sinalDoRetanguloDe)

// oRetanguloPega abre o laço no canto em que o dedo desceu.
//
// Guarda o canto DUAS vezes — como texto de caminho e como par de números — e
// isso não é redundância: o texto vai para a rota no fim, e os números desenham o
// laço a cada quadro. Derivar um do outro na expressão custaria um `split` por
// movimento do ponteiro.
func oRetanguloPega(modo string) string {
	return fmt.Sprintf(
		"evt.preventDefault(); const cx = %s, cy = %s; "+
			"$%s = %q; $%s = cx + '/' + cy; "+
			"$retangulodex = cx; $retangulodey = cy; $retanguloatex = cx; $retanguloatey = cy; "+
			"evt.currentTarget.setPointerCapture(evt.pointerId)",
		clicouEmX, clicouEmY, sinalDoRetangulo, modo, sinalDoRetanguloDe,
	)
}

// oRetanguloSegue leva o canto oposto atrás do dedo.
//
// NÃO fala com o servidor: o laço é geometria, e o resultado só é pedido quando o
// dedo solta. É a diferença para o pincel — lá cada casa cruzada é uma ação, aqui
// o gesto inteiro é UMA.
func oRetanguloSegue(modo string) string {
	return fmt.Sprintf(
		"$%s === %q && ($retanguloatex = %s, $retanguloatey = %s)",
		sinalDoRetangulo, modo, clicouEmX, clicouEmY,
	)
}

// oRetanguloDeTerrenoSolta fecha o laço e manda encher.
//
// O `$ferramenta` escolhe a rota: a borracha tem caminho sem espécie, que é o
// conserto que a fatia 1 fez e que não pode se perder aqui.
func oRetanguloDeTerrenoSolta(v tabuleiroView) string {
	return fmt.Sprintf(
		"if ($%s !== %q) return; const ate = $retanguloatex + '/' + $retanguloatey, de = $%s; "+
			"$%s = ''; "+
			"return $ferramenta === %q "+
			"? @post('/piloto/mesa/%d/%d/tabuleiro/terreno/limpar/retangulo/' + de + '/' + ate) "+
			": @post('/piloto/mesa/%d/%d/tabuleiro/terreno/' + $ferramenta + '/retangulo/' + de + '/' + ate)",
		sinalDoRetangulo, retanguloDeTerreno, sinalDoRetanguloDe,
		sinalDoRetangulo,
		FerramentaDaBorracha,
		v.CampaignID, v.SessionID,
		v.CampaignID, v.SessionID,
	)
}

// oLacoEstaAberto mostra o desenho do laço.
func oLacoEstaAberto() string {
	return fmt.Sprintf("$%s !== ''", sinalDoRetangulo)
}

// oEstiloDoLaco põe o retângulo na tela, em QUADRADOS do plano.
//
// `min` e `+1` porque o retângulo inclui as duas casas das pontas e o dedo pode
// arrastar para qualquer lado — a mesma regra que o `CasasDoRetangulo` aplica do
// lado do servidor, e é de propósito que as duas existam: esta desenha o que
// aquela vai fazer, e uma promessa que não bate com o resultado é pior que não
// desenhar nada.
const oEstiloDoLaco = "`left: ${Math.min($retangulodex, $retanguloatex) * $quadrado}px; " +
	"top: ${Math.min($retangulodey, $retanguloatey) * $quadrado}px; " +
	"width: ${(Math.abs($retanguloatex - $retangulodex) + 1) * $quadrado}px; " +
	"height: ${(Math.abs($retanguloatey - $retangulodey) + 1) * $quadrado}px`"

// oGestoDoPincel decide entre TRAÇO e RETÂNGULO no `pointerdown`.
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
func oGestoDoPincel(v tabuleiroView, modoFixo string) string {
	return fmt.Sprintf("if (evt.shiftKey) { %s } else { %s }",
		oRetanguloPega(retanguloDeTerreno), oPincelPega(v, modoFixo))
}
