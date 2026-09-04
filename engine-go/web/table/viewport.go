package table

import "fmt"

// A JANELA sobre um plano INFINITO (ALE-203, decisão do dono).
//
// O tabuleiro deixou de ter moldura. O servidor manda o que EXISTE, em
// Coordinate absoluta do plano, e quem decide o que aparece é a janela — que
// mora no NAVEGADOR, ao lado do zoom, e nunca vai ao servidor.
//
// # O que a moldura custava, medido
//
// Ela CRESCIA. Pintar perto da borda expandia o retângulo e mexia em `X0`, então
// o mesmo ponto da tela virava outro quadrado entre dois cliques — medido na
// bancada, `X0` foi de -11 para -12 e tudo andou um quadrado. Era uma das duas
// causas do "apaguei e não apagou" que o dono relatou.
//
// E ela era uma caixa: fora dela não havia onde clicar, então pintar longe do
// grupo exigia primeiro que a moldura crescesse até lá.
//
// # Quem recorta é o NAVEGADOR (decisão do dono)
//
// O servidor manda tudo o que existe — o teto já é de 200 peças — e o navegador
// pinta só o que aparece. A alternativa (o cliente informar a janela e o servidor
// recortar) faria arrastar a vista virar conversa com a rede, e faria o stream
// depender de QUEM olha: cada pessoa receberia um recorte diferente do mesmo
// tabuleiro.
//
// O preço aceito é o HTML carregar o mapa inteiro, que é o que ele já fazia.

// A janela é medida em PIXELS e não em quadrados, e a razão é o arrasto: o dedo
// anda em pixels, e converter a cada passo daria um degrau visível quando o
// quadrado é grande. O zoom continua sendo o `$quadrado`, e os dois se compõem
// na hora de desenhar.
const (
	sinalDaVistaX = "vistax"
	sinalDaVistaY = "vistay"
	// sinalDoArrastoDaVista diz se o dedo está segurando o plano. Ele existe
	// para o CURSOR (a mão fechada) e não para a conta: o `followsView` já o lê
	// como guarda, e o `data-class` da camada o lê para trocar o desenho.
	sinalDoArrastoDaVista = "arrastandoavista"
)

// sceneStyle leva a janela e o zoom do sinal para o CSS.
//
// Vai na CENA — a caixa que não rola e que o remendo não substitui — e desce por
// herança até o plano e a grade. No plano ele seria apagado pelo primeiro quadro
// do SSE, que é a mesma razão pela qual as variáveis do arrasto moram no `#mesa`.
const sceneStyle = "`--quadrado: ${$quadrado}px; --vista-x: ${$vistax}px; --vista-y: ${$vistay}px`"

// oQuadradoClicado traduz o PONTO do clique em quadrado do plano.
//
// UM helper e não uma cópia por camada, e isto INVERTE a decisão anterior. O
// comentário que estava no `clickedPointMarking` dizia que extrair a conta
// "faria as duas rotas mudarem juntas no dia em que uma delas precisar do canto e
// não do centro" — e o dia em que TODAS mudaram juntas chegou primeiro: sem
// moldura, a conta ganhou a janela dentro dela, e havia cinco cópias para
// atualizar. Uma que ficasse para trás clicaria no quadrado errado sem erro
// nenhum.
//
// A conta é do CLIENTE porque ela é sobre PIXELS: o servidor não sabe o zoom nem
// onde cada pessoa está olhando. O que ela decide continua sendo do servidor — o
// caminho, o custo, se cabe.
//
// O `evt.offsetX` mede a partir do ALVO do evento, e aqui ele serve porque as
// camadas de clique são irmãs do plano e cobrem a janela inteira: a quina delas
// É a quina da tela. Quem escuta na CENA (a roda) não pode usá-lo — o alvo lá
// pode ser uma peça dentro do plano deslocado —, e por isso o zoom mede pelo
// retângulo do `currentTarget`.
const (
	clicouEmX = "Math.floor((evt.offsetX + $vistax) / $quadrado)"
	clicouEmY = "Math.floor((evt.offsetY + $vistay) / $quadrado)"
)

// A INTERSEÇÃO MAIS PERTO do clique, e não a casa.
//
// `round` no lugar de `floor`, e a diferença é a regra da esfera: ela "surge na
// interseção de quatro quadrados" (p225), e uma interseção é um CANTO. Com
// `floor` o clique cai sempre no canto de CIMA E À ESQUERDA da casa apontada —
// até meio quadrado longe de onde o dedo estava, sem nada na tela dizendo por
// quê.
//
// A convenção do canto é a do `engine.sphereSquares`: o canto `(X,Y)` é o ponto
// no alto-esquerda da casa `(X,Y)`, e a esfera se espalha simétrica em volta
// dele.
const (
	clicouNoCantoX = "Math.round((evt.offsetX + $vistax) / $quadrado)"
	clicouNoCantoY = "Math.round((evt.offsetY + $vistay) / $quadrado)"
)

// planPoint é o mesmo para um ponto qualquer da janela, e não só do clique —
// o arrasto da vista precisa dele para o zoom ancorado.
func planPoint(pixelX, pixelY string) (x, y string) {
	return fmt.Sprintf("((%s) + $vistax) / $quadrado", pixelX),
		fmt.Sprintf("((%s) + $vistay) / $quadrado", pixelY)
}

// ── ARRASTAR A VISTA ─────────────────────────────────────────────────────────
//
// A rolagem nativa saiu com a moldura: ela precisava de uma caixa com tamanho
// para ter até onde rolar, e num plano infinito não existe fim. O que entra é o
// arrasto, que é o gesto que a issue pedia desde o começo ("movimento livre") e
// que o dono cobrou.

// ViewTool é o valor do sinal quando o arrasto move a JANELA.
const ViewTool = "vista"

// pegaAVista guarda de onde o dedo saiu. Em PIXELS da janela, não do plano: o
// que se mede é o deslocamento do dedo, e ele não depende do zoom.
const pegaAVista = "$arrastandoavista = true; $vistainix = evt.clientX + $vistax; " +
	"$vistainiy = evt.clientY + $vistay; evt.currentTarget.setPointerCapture(evt.pointerId)"

// followsView move a janela junto com o dedo.
//
// A janela anda ao CONTRÁRIO do dedo: arrastar para a direita traz o conteúdo da
// esquerda, que é como todo mapa se comporta. Por isso o sinal é
// `início - atual` e não o inverso.
const followsView = "$arrastandoavista && ($vistax = $vistainix - evt.clientX, " +
	"$vistay = $vistainiy - evt.clientY)"

const dropView = "$arrastandoavista = false"

// viewportMoveWheel: a roda ROLA o plano, e `Ctrl+roda` amplia.
//
// É a mesma divisão de antes, quando havia rolagem nativa — a roda percorria o
// mapa e o `Ctrl` ampliava (convenção de mapa). O que muda é que agora a roda
// escreve na janela em vez de o navegador rolar uma caixa, e por isso ela precisa
// do `preventDefault`: sem ele, a página inteira rolaria atrás do tabuleiro.
var wheelMovesViewport = fmt.Sprintf(
	"evt.preventDefault(); if (evt.ctrlKey) { const janela = evt.currentTarget.getBoundingClientRect(); %s } "+
		"else { $%s += evt.deltaX; $%s += evt.deltaY }",
	zoomAnchored("evt.deltaY < 0 ? "+step(ZoomStep)+" : "+step(-ZoomStep),
		"evt.clientX - janela.left", "evt.clientY - janela.top"),
	sinalDaVistaX, sinalDaVistaY)

// centerViewport põe um quadrado do plano no meio da tela.
//
// Ela substitui o `scrollTo` do centralizar: sem caixa que rola, o que se move é
// a janela. E ela é PURA aritmética no cliente — nada vai ao servidor, como todo
// o resto do enquadramento.
func centerViewport(x, y int) string {
	return fmt.Sprintf(
		"const cena = document.getElementById(%q); "+
			"$vistax = (%d + 0.5) * $quadrado - cena.clientWidth / 2; "+
			"$vistay = (%d + 0.5) * $quadrado - cena.clientHeight / 2",
		sceneId, x, y)
}

// sceneId é como o centralizar acha a janela para medir.
//
// A CENA e não o palco: o palco deixou de existir como caixa que rola, e quem
// tem o tamanho da janela agora é a caixa que ancora os overlays.
const sceneId = "tabuleiro-cena"

// viewportFollowsFocus devolve o que a ROLAGEM NATIVA fazia de graça, e sem ele
// esta fatia teria embutido uma regressão de teclado.
//
// MEDIDO, e vermelho antes de escrito: com a peça em (-2039,-1268) e a janela em
// (92,97,1756×807), um `focus()` na peça deixava tudo exatamente onde estava —
// `dentro: false`, foco na peça. O navegador TENTA trazer o elemento focado para
// a vista, mas ele rola um ANCESTRAL ROLÁVEL, e não existe mais nenhum: a cena
// recorta com `overflow: hidden` e a página não rola. Quem navega por teclado
// podia focar uma peça que nunca ia conseguir ver.
//
// A conta é a do "rolar o mínimo": só o quanto o alvo transborda de cada lado. O
// alvo INTEIRAMENTE dentro não move nada, e é por isso que o clique numa camada
// (que cobre a janela) e nos botões do trilho (que ficam dentro dela) não
// empurram a vista — os dois `Math.max` dão zero.
//
// `alvo.width &&` pula o que não tem caixa: um alvo de tamanho zero daria uma
// conta sobre um retângulo em lugar nenhum.
const viewportFollowsFocus = "const janela = evt.currentTarget.getBoundingClientRect(), " +
	"alvo = evt.target.getBoundingClientRect(); alvo.width && (" +
	"$vistax += Math.max(0, alvo.right - janela.right) - Math.max(0, janela.left - alvo.left), " +
	"$vistay += Math.max(0, alvo.bottom - janela.bottom) - Math.max(0, janela.top - alvo.top))"

// AS SETAS NÃO PERCORREM O PLANO, e isto é medido e não escolhido — é a mesma
// história do Escape que o trilho registra.
//
// Elas já têm dono: o `cena.js` mapeia as quatro para "mover o foco" na gramática
// espacial do teclado e chama `preventDefault` + `stopPropagation` no
// `document`. O evento **nunca chega à janela**, que é onde o `__window` escuta.
// Provado com controle no mesmo teclado: um `F2` chega a um `addEventListener`
// cru na janela, um `ArrowRight` não chega — e o `-` do zoom chega, que é o
// controle positivo do próprio canal.
//
// Então quem percorre o plano pelo teclado é o FOCO, com o `viewportFollowsFocus`
// acima: a gramática move o foco entre as peças, e a janela vai atrás. Escrever
// um ramo de seta aqui seria uma promessa que a tela não cumpre.

// viewportSignals são o estado do enquadramento no navegador.
//
// A janela nasce em ZERO, e isso quer dizer "o quadrado (0,0) do plano está na
// quina de cima da tela". É um lugar arbitrário num plano sem bordas — e é por
// isso que a cena CENTRALIZA nas peças ao abrir, em vez de deixar o mestre
// procurar o próprio grupo.
var viewportSignals = fmt.Sprintf("%s: 0, %s: 0, %s: false, vistainix: 0, vistainiy: 0",
	sinalDaVistaX, sinalDaVistaY, sinalDoArrastoDaVista)
