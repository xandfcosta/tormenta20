package table

import "fmt"

// O GESTO CONTÍNUO do pincel e da borracha (ALE-203, itens 8 e 9 do dono).
//
// "Não é possível pintar terreno em vários quadrados segurando o botão direito e
// nem apagar segurando o botão esquerdo." Pintar uma parede de taverna casa a
// casa, com um clique por quadrado, é o tipo de trabalho que faz o mestre montar
// a cena antes da sessão e nunca durante ela.
//
// # A máquina, e por que ela precisa de um sinal de MEMÓRIA
//
// O `pointermove` chega dezenas de vezes por segundo e a maior parte deles cai
// no MESMO quadrado. Postar em todos seria mandar a mesma casa vinte vezes — e o
// que o `$ultimacasa` guarda é a última que já foi mandada, para o gesto só
// falar quando o dedo TROCA de casa. Sem ele o servidor recebe uma enxurrada
// idempotente e a mesa vê a cena piscar.
//
// # O botão DIREITO apaga
//
// É o que a issue pede desde o começo e o que todo editor de mapa faz. Ele custa
// o `contextmenu` preso: sem `preventDefault` o menu do navegador abre em cima do
// tabuleiro no primeiro quadrado e o arrasto morre ali.
//
// # O MODO é decidido no `pointerdown` e guardado
//
// E não relido a cada `pointermove` pelo `evt.buttons`. A razão é o gesto que
// começa com o esquerdo e ganha o direito no meio: relendo, o traço trocaria de
// mão sozinho; guardando, ele faz até o fim o que foi pedido no começo.

// brushSignal é o modo do gesto em curso: vazio (parado), `pintar` ou
// `apagar`. Um sinal só e o valor É o modo, como o `$ferramenta` — assim não
// existe o estado impossível "apagando e pintando".
const brushSignal = "pincelando"

// squareLastSignal é a casa que o gesto JÁ mandou, no formato do CAMINHO
// (`"x/y"`).
//
// String e não dois números porque ela tem DUAS perguntas a responder e as duas
// são sobre o par junto: "mudou de casa?" e "de onde vem o traço?". Dois sinais
// dariam duas chances de atualizar um e esquecer o outro.
//
// A BARRA e não a vírgula, e isso é conserto de um defeito MUDO: com `"x,y"` o
// caminho saía `terreno/dificil/12,5/ate/12/5`, o chi não casava a rota, o
// servidor devolvia 404, e o Datastar descartava a resposta sem escrever nada em
// lugar nenhum — nem no console. O sintoma era o pincel não pintar, sem uma linha
// de erro para seguir. Guardar já no formato de destino tira a conversão do
// caminho, e com ela o lugar onde o erro cabia.
const squareLastSignal = "ultimacasa"

const (
	pincelPinta = "pintar"
	pincelApaga = "apagar"
)

// brushSignals entram na semente da Mesa.
var brushSignals = fmt.Sprintf("%s: '', %s: ''", brushSignal, squareLastSignal)

// takesBrush começa o traço na casa clicada.
//
// `modoFixo` vazio quer dizer "o botão decide" — é a camada de PINTAR, onde o
// direito apaga. A camada da borracha passa `pincelApaga` porque lá os dois
// botões fazem a mesma coisa: a ferramenta já disse o que o gesto é.
//
// O `setPointerCapture` é o mesmo do arrasto da vista, e pelo mesmo motivo:
// soltar o botão fora do tabuleiro tem de terminar o traço. Sem ele o pincel
// fica preso e a próxima passada do mouse pinta sem ninguém ter apertado nada.
//
// Ele vai por ÚLTIMO, e isso é seguro e não estilo: ele LANÇA quando o ponteiro
// não está mais ativo (`NotFoundError: No active pointer with the given id`), e
// no meio da expressão essa exceção engoliria a pintura da primeira casa — o
// gesto começaria mudo. Por último, o pior que acontece é o traço perder a
// captura e terminar quando o dedo sai do elemento.
func takesBrush(v BoardView, modoFixo string) string {
	modo := fmt.Sprintf("evt.button === 2 ? %q : %q", pincelApaga, pincelPinta)
	if modoFixo != "" {
		modo = fmt.Sprintf("%q", modoFixo)
	}
	return fmt.Sprintf(
		"evt.preventDefault(); $%s = %s; $%s = ''; %s; "+
			"evt.currentTarget.setPointerCapture(evt.pointerId)",
		brushSignal, modo, squareLastSignal, brushActsOnSquare(v),
	)
}

// followsBrush continua o traço, e só quando o dedo TROCA de casa.
func followsBrush(v BoardView) string {
	return fmt.Sprintf("$%s !== '' && (%s)", brushSignal, brushActsOnSquare(v))
}

// wideBrush encerra o traço. Quem apaga a memória é o `pointerdown` do gesto
// SEGUINTE, e por uma razão que só apareceu com o traço: se o `pointerup`
// limpasse, o `$ultimacasa` ficaria vazio entre os gestos — e ficaria vazio
// também depois de um `pointercancel` no meio do traço, com o gesto ainda em
// curso. Limpar na abertura é o único instante em que "não há traço anterior" é
// verdade.
var wideBrush = fmt.Sprintf("$%s = ''", brushSignal)

// brushActsOnSquare é o corpo compartilhado: traduz o ponto, sai se a casa é a
// mesma de antes, e manda o TRAÇO daquela até esta.
//
// O SEGMENTO e não o ponto, e isso é conserto de um defeito medido: entre dois
// avisos do ponteiro o dedo anda mais de uma casa, e mandar só onde ele ESTÁ
// deixava buraco. Na bancada, um arrasto pintou 11,6 · 13,6 · 15,7 · 16,8 · 18,9
// — as colunas 12, 14 e 17 vazias. Quem preenche é o `tabuleiro.CasasDoTraco`,
// porque a conta é regra de tabuleiro e não de tela.
//
// O `pointerdown` manda a casa CONTRA ELA MESMA (um traço de uma casa), e é por
// isso que ele limpa o `$ultimacasa` antes: senão o primeiro traço do gesto sairia
// da última casa do gesto ANTERIOR, e clicar em dois cantos opostos do mapa
// pintaria a diagonal inteira entre eles.
//
// UMA função para os dois gestos porque `pointerdown` e `pointermove` fazem a
// mesma coisa — a diferença é só quem decide o modo. Duas cópias dariam duas
// rotas para atualizar no dia em que o caminho mudar, e a esquecida seria a do
// arrasto, que é a que ninguém testa clicando.
func brushActsOnSquare(v BoardView) string {
	return fmt.Sprintf(
		"(() => { const cx = %s, cy = %s, casa = cx + '/' + cy; "+
			"if (casa === $%s) return; "+
			"const de = $%s === '' ? casa : $%s; $%s = casa; "+
			"const traco = de + '/ate/' + casa; "+
			"return $%s === %q "+
			"? @post('/mesa/%d/%d/tabuleiro/terreno/limpar/' + traco) "+
			": @post('/mesa/%d/%d/tabuleiro/terreno/' + $ferramenta + '/' + traco) })()",
		clicouEmX, clicouEmY,
		squareLastSignal,
		squareLastSignal, squareLastSignal, squareLastSignal,
		brushSignal, pincelApaga,
		v.CampaignID, v.SessionID,
		v.CampaignID, v.SessionID,
	)
}

// takesEraser é o traço da BORRACHA: os dois botões apagam.
//
// Ela usa a rota sem espécie (`terreno/limpar`), então o `$ferramenta` não entra
// na conta — que é exatamente o conserto do defeito que o dono relatou como "a
// borracha não funciona".
func takesEraser(v BoardView) string { return takesBrush(v, pincelApaga) }

// prendeOMenuDoNavegador é o `contextmenu` das camadas que usam o botão direito.
//
// Sem ele o menu do navegador abre no primeiro quadrado e o traço morre ali —
// com o botão ainda apertado, que é o pior dos dois mundos: o gesto parece ter
// travado.
const prendeOMenuDoNavegador = "evt.preventDefault()"
