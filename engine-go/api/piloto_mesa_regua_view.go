package api

import (
	"fmt"
	"strconv"
	"strings"

	"t20engine/engine"
	"t20engine/tabuleiro"
)

// As expressões da RÉGUA e do GABARITO (ALE-269, superfície 8).
//
// Arquivo próprio e não mais um pedaço do `piloto_mesa_tabuleiro_view.go`, que
// já passa de 800 linhas: o que mora aqui é uma responsabilidade fechada — as
// duas ferramentas que MEDEM, e nenhuma delas muda a cena.
//
// A divisão do trabalho entre os dois lados é a mesma do resto do tabuleiro, e
// vale dizer onde ela cai: o NAVEGADOR guarda as pontas e traduz pixel em
// quadrado (é ele quem sabe o zoom); o SERVIDOR conta os quadrados, escolhe a
// faixa do livro, desenha a forma e diz quem ela pega. Nada do que a régua
// responde é recalculado na tela.

// FerramentaDaRegua e FerramentaDoGabarito são os valores do sinal `$ferramenta`
// quando o clique MEDE em vez de mover ou pintar.
//
// Constantes pela mesma razão da `MarkTool`: cada uma aparece em meia
// dúzia de expressões e `data-show`, e escrita à mão a sexta ocorrência é a que
// erra a letra e vira uma ferramenta que a tela liga e o mapa nunca escuta.
const (
	FerramentaDaRegua    = "regua"
	FerramentaDoGabarito = "gabarito"
)

// onIsBrush é o teste que separa PINTAR de medir, escrito a partir da
// lista de espécies e nunca à mão.
//
// Ele existe porque a régua e o gabarito quebraram a pergunta antiga: a camada de
// pintura mostrava-se com `$ferramenta != ” && $ferramenta != 'marcador'`, que
// era verdade para toda ferramenta que ainda não existia. Uma lista escrita à mão
// no `.templ` teria o mesmo defeito adiado — a espécie nova nasceria fora dela.
func onIsBrush() string {
	nomes := make([]string, 0, len(tabuleiro.TerrainKinds))
	for _, e := range tabuleiro.TerrainKinds {
		nomes = append(nomes, fmt.Sprintf("%q", string(e.ID)))
	}
	return fmt.Sprintf("[%s].includes($ferramenta)", strings.Join(nomes, ", "))
}

// AS FASES da régua. Elas eram 0/1/2 escritas à mão em nove lugares, e a
// terceira mudou de significado na ALE-203 — "fechada" deixou de ser "as duas
// pontas postas" e virou "CONGELADA", que é outra coisa.
const (
	reguaParada    = 0
	reguaMedindo   = 1
	reguaCongelada = 2
)

// clickedPointRuler é a máquina da POLILINHA (ALE-203, escolha do dono).
//
// Ela era de dois cliques: o primeiro punha a origem, o segundo fechava, o
// terceiro recomeçava. O dono usou e apontou o que faltava — "a régua não
// permite calcular distâncias com mais de uma parada" — e escolheu a gramática:
//
//	esquerdo        acrescenta uma parada (o primeiro começa a régua)
//	duplo esquerdo  CONGELA: para de seguir o ponteiro e fica desenhada
//	direito         APAGA, em qualquer estado
//
// Três gestos, três efeitos FIXOS: nenhum deles muda de significado conforme o
// estado, que era o custo das outras duas formas que estavam na mesa.
//
// A PARADA REPETIDA é recusada, e isso não é higiene: o duplo clique dispara um
// clique simples ANTES dele, na mesma casa. Sem esta linha a última parada
// nasceria duplicada e o `dblclick` congelaria uma régua com uma perna de zero
// quadrado pendurada. Com ela, o segundo clique do duplo não faz nada e o
// congelamento cai limpo.
//
// O ESC NÃO ENTRA, e é medido: o `cena.js` mapeia Escape para "voltar" e chama
// `stopPropagation` no documento — provado com controle, um `F2` chega a um
// listener cru na janela e o `Escape` não. Era ele que a nota do dono usava para
// apagar; quem apaga é o botão direito, que chega.
func clickedPointRuler(v boardView) string {
	return fmt.Sprintf(
		// SÓ O BOTÃO PRIMÁRIO acrescenta parada. O `click` do Chrome já é só do
		// primário, mas o guarda é barato e ele apareceu na bancada: a automação
		// do navegador dispara um `click` sintético junto com o clique direito, e
		// o sintético vem com `offsetX` ZERO — o que se via era a régua sendo
		// apagada e nascendo de novo na ORIGEM do plano, no mesmo gesto.
		"if (evt.button !== 0) return; const cx = %s, cy = %s; "+
			"if ($reguafase !== %d) { $reguapontos = [[cx, cy]]; $reguafase = %d } "+
			"else { const p = [...$reguapontos], u = p[p.length - 1]; "+
			"if (u[0] === cx && u[1] === cy) return; $reguapontos = [...p, [cx, cy]] } "+
			"$reguamirax = cx; $reguamiray = cy; %s",
		clicouEmX, clicouEmY, reguaMedindo, reguaMedindo, repatchRuler(v),
	)
}

// pointerFollowsRuler é a PERNA VIVA: a linha que sai da última parada e vai
// até o dedo, sempre (pedido do dono).
//
// Ela só fala com o servidor quando o ponteiro TROCA DE CASA, pelo mesmo motivo
// do pincel: a medida muda por quadrado e não por pixel, e postar a cada quadro
// seria mandar a mesma pergunta sessenta vezes por segundo.
//
// O DESENHO da perna, esse acompanha o pixel: ele sai dos sinais e não do
// servidor, porque é geometria e não regra.
func rulerFollowsPointer(v boardView) string {
	return fmt.Sprintf(
		"if ($reguafase !== %d) return; const cx = %s, cy = %s; "+
			"if (cx === $reguamirax && cy === $reguamiray) return; "+
			"$reguamirax = cx; $reguamiray = cy; %s",
		reguaMedindo, clicouEmX, clicouEmY, repatchRuler(v),
	)
}

// freezeRuler fecha a medida: ela para de seguir o ponteiro e fica na tela.
//
// É o `dblclick`, que é a convenção de "terminar polilinha" de todo editor de
// desenho. O `preventDefault` existe porque um duplo clique também SELECIONA
// texto, e um tabuleiro com meia tela selecionada em azul é o que a mesa vê.
var freezeRuler = fmt.Sprintf("evt.preventDefault(); $reguafase = %d", reguaCongelada)

// saveRuler apaga as paradas, o desenho E a leitura.
//
// As três juntas porque são a mesma coisa: a frase de uma medida cujo desenho
// sumiu é a resposta a uma pergunta que ninguém consegue mais ver.
var saveRuler = fmt.Sprintf("$reguafase = %d; $reguapontos = []; $reguarotulos = []; $reguatexto = %q",
	reguaParada, emptyRulerHint)

// repatchRuler pede os rótulos das pernas e a frase do total.
//
// UMA expressão para os dois gestos que mudam a régua — acrescentar parada e
// mover o ponteiro —, porque os dois fazem a mesma pergunta. Duas cópias dariam
// dois lugares para esquecer de mandar, e o sintoma seria a régua que desenha
// certo e mede errado.
func repatchRuler(v boardView) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/regua')", v.CampaignID, v.SessionID)
}

// clickedPointTemplate põe o gabarito, e aponta quando a forma pede.
//
// O primeiro clique põe a origem; o segundo, quando a forma aponta, diz para
// onde. Depois disso o clique seguinte RECOMEÇA — a mesa posiciona a mesma bola
// de fogo três vezes antes de decidir, e um botão de limpar entre cada tentativa
// seria um clique a mais em cada uma.
//
// QUEM APONTA é dito pelo `$gabaritoaponta`, escrito pelo botão da forma com o
// valor que o `pointsTemplate` do servidor deu. Perguntar aqui `$gabarito ===
// 'cone' || $gabarito === 'linha'` seria a segunda cópia da regra da p225, livre
// para divergir da primeira.
func clickedPointTemplate(v boardView) string {
	return fmt.Sprintf(
		// SÓ O BOTÃO PRIMÁRIO põe gabarito, pela mesma razão medida na régua: o
		// clique sintético que acompanha o botão direito vem com `offsetX` zero e
		// poria a área na origem do plano no mesmo gesto que a apagou.
		//
		// A ORIGEM pousa onde o LIVRO manda, e é o `$gabaritonaintersecao` quem
		// diz qual: a esfera na interseção de quatro quadrados, o resto na casa
		// (p225). Perguntar aqui `$gabarito === 'esfera'` seria a segunda cópia da
		// regra, livre para divergir da primeira — a mesma razão do
		// `$gabaritoaponta`.
		//
		// A MIRA continua sendo CASA em qualquer forma: ela é para onde o cone
		// aponta, e direção se escolhe apontando para um quadrado.
		"if (evt.button !== 0) return; "+
			"const cx = $gabaritonaintersecao ? %s : %s, cy = $gabaritonaintersecao ? %s : %s; "+
			"if ($gabaritofase === 1 && $gabaritoaponta) { $gabaritomirax = %s; $gabaritomiray = %s; $gabaritofase = 2 } "+
			"else { $gabaritox = cx; $gabaritoy = cy; $gabaritomirax = %s; $gabaritomiray = %s; $gabaritofase = 1 } "+
			"%s",
		clicouNoCantoX, clicouEmX, clicouNoCantoY, clicouEmY,
		clicouEmX, clicouEmY,
		clicouEmX, clicouEmY,
		repatchTemplate(v),
	)
}

// pointerFollowsTemplate leva a MIRA atrás do dedo enquanto a forma aponta.
//
// É o irmão do `rulerFollowsPointer` e tem as mesmas duas guardas: só fala com
// o servidor quando o ponteiro TROCA DE CASA — a área muda por quadrado e não
// por pixel —, e só enquanto a fase é a de mirar. Posto, o desenho fica e o
// ponteiro passeia.
//
// SÓ AS FORMAS QUE APONTAM: a esfera vai para todos os lados (p225), e arrastar
// o ponteiro sobre uma esfera posta não tem o que mudar.
func templateFollowsPointer(v boardView) string {
	return fmt.Sprintf(
		"if ($gabaritofase !== 1 || !$gabaritoaponta) return; const cx = %s, cy = %s; "+
			"if (cx === $gabaritomirax && cy === $gabaritomiray) return; "+
			"$gabaritomirax = cx; $gabaritomiray = cy; %s",
		clicouEmX, clicouEmY, repatchTemplate(v),
	)
}

// repatchTemplate pede o desenho e a lista de quem ele pega.
//
// UMA expressão para os três gestos que mudam o gabarito — pôr, trocar de forma,
// mudar o tamanho —, porque os três fazem a mesma pergunta ao servidor. Três
// cópias dariam três lugares para esquecer de mandar um dos parâmetros, e o
// sintoma seria um gabarito que ignora o número que a pessoa acabou de digitar.
func repatchTemplate(v boardView) string {
	return fmt.Sprintf(
		"@post('/mesa/%d/%d/tabuleiro/gabarito/' + $gabarito + '/' + $gabaritotamanho"+
			" + '/' + $gabaritox + '/' + $gabaritoy + '/' + $gabaritomirax + '/' + $gabaritomiray)",
		v.CampaignID, v.SessionID,
	)
}

// pickShape troca a forma e LARGA o que estava posto.
//
// Trocar sem largar foi medido na SPA e desenha errado: o primeiro clique depois
// da troca cai na regra do segundo e APONTA a forma nova a partir da origem da
// antiga — escolher "Cone" com uma esfera na tela desenhava um cone apontado para
// o lado de onde se clicou.
//
// O `$gabaritoaponta` sai daqui com o valor do SERVIDOR: é o botão que sabe qual
// forma ele liga, e é o `pointsTemplate` que sabe quais formas apontam.
func pickShape(forma engine.AreaKind) string {
	return fmt.Sprintf("$gabarito = %q; $gabaritoaponta = %t; $gabaritonaintersecao = %t; %s",
		string(forma), pointsTemplate(forma), shapeStartsAtIntersection(forma), saveTemplate)
}

// emptyTemplateHint é o que a barra diz enquanto não há gabarito posto, e ela
// vem do SERVIDOR — é a mesma frase que o `takesTemplateWho` devolve para uma
// área sem casa nenhuma. Um literal aqui seria a segunda cópia, livre para
// divergir da primeira no dia em que alguém melhorar uma das duas.
var emptyTemplateHint = takesTemplateWho(nil, nil)

// saveTemplate apaga o desenho E a lista, pelo mesmo motivo da régua.
var saveTemplate = fmt.Sprintf(
	"$gabaritofase = 0; $gabaritopath = ''; $gabaritotexto = %q", emptyTemplateHint)

// shapeMeasure é a palavra que nomeia o número que a pessoa digita: a esfera
// tem raio, o cone tem alcance, a linha tem comprimento e o quadrado tem lado
// (p225).
//
// O rótulo muda com a forma porque "tamanho" não responde a pergunta de nenhuma
// das quatro — o conjurador está lendo "raio 6m" na ficha, e é esse número que
// ele quer digitar.
func shapeMeasure(k engine.AreaKind) string {
	switch k {
	case engine.AreaSphere:
		return "raio"
	case engine.AreaCone:
		return "alcance"
	case engine.AreaLine:
		return "comprimento"
	default:
		return "lado"
	}
}

// bookShapes é a ordem em que a mesa as usa (p225), e ela é a mesma da SPA.
var bookShapes = []engine.AreaKind{
	engine.AreaSphere, engine.AreaCone, engine.AreaLine, engine.AreaSquare,
}

// shapeLabel é o nome na tela, com maiúscula.
//
// Escrito por extenso e não derivado do id com um `ToUpper` na primeira letra: o
// id é minúsculo e sem acento porque é identificador, e o dia em que uma forma
// tiver acento no nome a derivação erraria em silêncio.
func shapeLabel(k engine.AreaKind) string {
	switch k {
	case engine.AreaSphere:
		return "Esfera"
	case engine.AreaCone:
		return "Cone"
	case engine.AreaLine:
		return "Linha"
	default:
		return "Quadrado"
	}
}

// AS EXPRESSÕES DO DESENHO da polilinha. Todas saem dos SINAIS e nenhuma do
// servidor, e a divisa é a de sempre nesta casa: GEOMETRIA é do navegador,
// REGRA é do motor. O caminho, os pontos e a posição dos rótulos são geometria;
// o texto de cada rótulo é a regra, e vem de `polylineReading`.
//
// Tudo em unidades de QUADRADO, porque o `<g>` já converte com o `scale` da
// janela — desenhar em pixel aqui obrigaria cada expressão a conhecer o zoom.

// rulerPath é o `d` da polilinha, do centro de uma parada ao da próxima.
//
// A MIRA entra no fim enquanto a régua mede, e é ela a "linha conectando a
// última parada até o mouse" que o dono pediu. Congelada, o desenho para nas
// paradas e o ponteiro passeia sem mexer nele.
var rulerPath = fmt.Sprintf(
	"(() => { const p = [...$reguapontos]; "+
		"if ($reguafase === %d) p.push([$reguamirax, $reguamiray]); "+
		"return p.map((c, i) => (i ? 'L' : 'M') + (c[0] + 0.5) + ' ' + (c[1] + 0.5)).join(' ') })()",
	reguaMedindo)

// ── LER UMA LISTA DE SINAL SEM CRIAR SINAL NENHUM ────────────────────────────
//
// `$reguapontos[0]` NÃO é "o primeiro item da lista": o Datastar lê `$nome...`
// como CAMINHO DE SINAL e REGISTRA o que não existe. Medido no navegador, com a
// reserva de doze rótulos no ar, o sinal virou
//
//	[[9,3], "", "", "", "", "", "", "", "", "", "", "", "", [17,7], [17,13], …]
//
// — doze strings vazias entre a primeira parada e as seguintes, uma por nó da
// reserva que leu um índice. O desenho saía com pingos na origem do plano e o
// servidor media zero, sem erro em lugar nenhum. É a mesma família da armadilha
// que a memória desta casa já registra para `data-bind` em caminho inexistente.
//
// A saída é COPIAR a lista para fora do sinal antes de indexá-la. Guardar o
// sinal numa constante não basta — medido: a constante continua sendo o proxy, e
// `lista[12]` cria `reguapontos.12` do mesmo jeito. O espalhamento `[...]` anda
// pelo ITERADOR, que só visita os índices que existem, e devolve um vetor comum:
// depois dele, ler um índice ausente é `undefined` e mais nada.
//
// Daí este helper: toda leitura de lista passa por aqui, e nenhuma expressão
// escreve `$lista[` no meio do código.
func list(sinal, corpo string) string {
	return fmt.Sprintf("(() => { const lista = [...$%s]; return %s })()", sinal, corpo)
}

// rulerStop é o centro da i-ésima parada, para o pingo que a marca.
func rulerStop(i int, eixo int) string {
	return list("reguapontos", fmt.Sprintf("(lista[%d]?.[%d] ?? 0) + 0.5", i, eixo))
}

// existsDot esconde o pingo da reserva que ainda não tem parada.
func existsDot(i int) string {
	return list("reguapontos", fmt.Sprintf("lista.length > %d", i))
}

// existsLabel esconde o rótulo da perna que ainda não existe.
//
// A RESERVA é o preço de o Datastar não ter laço: os rótulos são nós FIXOS no
// HTML, um por perna possível, e cada um se mostra conforme a lista de textos
// que o servidor devolveu. É verboso e é honesto — a alternativa seria remendar
// HTML dentro da região do mapa, e o quadro seguinte do stream apagaria a régua
// de quem estava medindo.
func existsLabel(i int) string {
	// VAZIO também esconde, e não é a mesma pergunta que "existe": a perna de
	// zero quadrado devolve texto vazio de propósito (ver `metersLeg`), e um
	// `<text>` sem conteúdo continuaria ocupando o nó com o halo do contorno.
	return list("reguarotulos", fmt.Sprintf("(lista[%d] ?? '') !== ''", i))
}

// legLabel é o texto que o servidor mediu para a perna `i`.
func legLabel(i int) string {
	return list("reguarotulos", fmt.Sprintf("lista[%d] ?? ''", i))
}

// legMid é onde o rótulo pousa: o meio do segmento entre a parada `i` e a
// seguinte — que pode ser a MIRA, quando a perna é a viva.
func legMid(i int, eixo int) string {
	return list("reguapontos", fmt.Sprintf(
		"(() => { const a = lista[%d], b = lista[%d] ?? "+
			"($reguafase === %d ? [$reguamirax, $reguamiray] : a); "+
			"return a && b ? ((a[%d] + b[%d]) / 2) + 0.5 : 0 })()",
		i, i+1, reguaMedindo, eixo, eixo))
}

// stopsReserve é a contagem que o `.templ` percorre para desenhar os nós
// fixos. Sai do MESMO teto que o servidor recusa — escritos em dois lugares,
// uma perna nasceria medida e sem rótulo.
func stopsReserve() []int {
	reserva := make([]int, stopsMax)
	for i := range reserva {
		reserva[i] = i
	}
	return reserva
}

// viewportDrawing põe o SVG a falar a língua do tabuleiro: a JANELA e o ZOOM.
//
// Ele era um `viewBox` do tamanho da moldura mais um `transform` que descontava
// a quina dela, e a moldura saiu na ALE-203. A primeira tentativa foi só um
// `scale($quadrado)` com o SVG dentro do plano deslocado — e ela NÃO DESENHAVA
// NADA. Medido: o `<path>` tinha caixa certa (176×176 no lugar certo), `fill`
// certo, `display: block`, e a tela ficava vazia; dar tamanho ao `<svg>` na mão
// fazia a esfera aparecer na hora.
//
// A causa é que o `<svg>` MAIS EXTERNO recorta pelo VIEWPORT dele, e `overflow:
// visible` não levanta esse recorte — dentro de um plano de tamanho zero, o
// viewport era 0×0 e tudo caía fora. É um recorte que não acusa: nada no DOM
// diz "isto está cortado".
//
// Então o SVG passou a ser filho da CENA e a cobrir a JANELA — que é um recorte
// que a gente QUER —, e os dois números que todo o resto usa entram aqui no
// `transform`: a janela desloca, o zoom escala. Nessa ordem, porque a janela é
// medida em PIXELS e o `scale` viria depois multiplicá-la.
const viewportDrawing = "`translate(${-$vistax}, ${-$vistay}) scale(${$quadrado})`"

// metersSize converte o número digitado para a unidade da FICHA.
//
// É o único cálculo desta superfície que fica do lado do navegador, e a razão é
// a digitação: ele acompanha a tecla, e uma ida ao servidor por tecla trocaria um
// número que muda a cada instante por uma conversa. O FATOR vem do
// `engine.SquareMetres` — escrever `1.5` no `.templ` seria mais uma cópia da
// p236 solta no repositório.
func metersSize() string {
	return fmt.Sprintf("($gabaritotamanho * %g).toFixed(1).replace('.', ',') + 'm'", engine.SquareMetres)
}

// templateBiggest é o teto da caixa de digitação: o alcance longo do livro
// (p224), que é o maior gabarito que cabe numa mesa. É o MESMO número que o
// `templateSize` trava do lado do servidor, e da mesma constante — a caixa
// é cortesia, e a trava é de lá.
func templateBiggest() string { return strconv.Itoa(engine.LongRangeSquares) }

// number escreve um inteiro do Go dentro de uma expressão do navegador.
func number(n int) string { return strconv.Itoa(n) }

// rulerSignals são a polilinha no navegador.
//
// `reguapontos` é uma LISTA de pares e não quatro números soltos, e foi a
// polilinha que forçou a troca: com número variável de paradas, quatro sinais
// nomeados viravam oito, depois vinte e quatro.
var rulerSignals = fmt.Sprintf(
	"reguapontos: [], reguamirax: 0, reguamiray: 0, reguafase: %d, reguarotulos: [], reguatexto: %q",
	reguaParada, emptyRulerHint)
