package tabuleiro

import "t20engine/engine"

// O TRAÇO do pincel: as casas que o dedo ATRAVESSOU entre dois avisos do
// ponteiro (ALE-203).
//
// # Por que o servidor precisa saber disto
//
// O `pointermove` chega a cada quadro, e entre um quadro e o seguinte o dedo pode
// ter andado mais que uma casa. Medido na bancada: um arrasto de 300px em passos
// de ~73px pintou 11,6 · 13,6 · 15,7 · 16,8 · 18,9 — as colunas 12, 14 e 17
// ficaram VAZIAS no meio do traço. É pior num zoom pequeno, onde a casa tem 20px
// e qualquer movimento normal pula três.
//
// A saída é o cliente mandar de ONDE ATÉ ONDE, e não onde ele está: uma
// requisição por quadro do mesmo jeito, com o segmento inteiro pintado do outro
// lado. Interpolar no navegador daria a mesma conta escrita em JavaScript dentro
// de uma expressão do Datastar — e a conta é regra de tabuleiro, que é deste
// lado.

// CasasDoTraco são todas as casas que o segmento de `de` até `ate` toca, na
// ordem em que o dedo passou por elas.
//
// É a variante SUPERCOBERTURA do Bresenham: ela inclui as casas que a linha
// apenas ROÇA, e não só uma por coluna. A diferença importa num traço quase
// horizontal, onde o Bresenham clássico deixa buraco na hora de mudar de linha —
// e buraco num muro de taverna é passagem que o mestre não desenhou.
//
// Exemplo:
//
//	CasasDoTraco(engine.Square{X: 0, Y: 0}, engine.Square{X: 2, Y: 1})
//	// → (0,0) (1,0) (1,1) (2,1)
func CasasDoTraco(de, ate engine.Square) []engine.Square {
	casas := []engine.Square{de}
	x, y := de.X, de.Y
	dx, sx := passo(ate.X - x)
	dy, sy := passo(ate.Y - y)
	// `erro` é a distância acumulada da linha ideal ao centro da casa atual,
	// dobrada para caber em inteiros — é o Bresenham de sempre. O que muda é que
	// aqui os dois eixos avançam SEPARADOS quando o erro está no meio, e é isso
	// que faz a casa roçada entrar.
	erro := dx - dy
	for x != ate.X || y != ate.Y {
		dobro := 2 * erro
		if dobro > -dy {
			erro -= dy
			x += sx
		} else if dobro < dx {
			erro += dx
			y += sy
		}
		casas = append(casas, engine.Square{X: x, Y: y})
	}
	return casas
}

// passo devolve a distância absoluta e a direção (+1, -1 ou 0) de um eixo.
func passo(delta int) (distancia, sentido int) {
	if delta < 0 {
		return -delta, -1
	}
	if delta > 0 {
		return delta, 1
	}
	return 0, 0
}

// oTracoCabe é o teto de casas que um traço pode pintar de uma vez.
//
// Ele não protege a memória — um traço é uma lista de pares de inteiros. Ele
// protege contra o pedido POSSUÍDO: sem teto, um `POST .../0/0/ate/9999999/0`
// faria o servidor pintar dez milhões de casas e gravar o tabuleiro inteiro num
// JSON que a mesa depois teria de baixar.
//
// Cem é largo para o gesto real: num quadro de 16ms nenhum dedo atravessa cem
// casas. O traço mais longo medido na bancada teve 9.
const oTracoCabe = 100

// TracoValido recusa o segmento que não pode ter saído de um dedo.
//
// Recusa em vez de cortar, e a diferença é o silêncio: um traço cortado pinta
// uma parte e some com a outra, e quem pediu conclui que o pincel falha às vezes.
func TracoValido(de, ate engine.Square) bool {
	return max(abs(ate.X-de.X), abs(ate.Y-de.Y)) < oTracoCabe
}

// ── O RETÂNGULO (ALE-203, item 10 do dono) ───────────────────────────────────
//
// "Não temos ferramenta de seleção em área." Com o pincel na mão, o retângulo
// ENCHE: uma parede de taverna com dois cantos em vez de vinte passadas.
//
// Ele é irmão do traço e não uma máquina nova — as duas rotas de terreno passaram
// a receber DOIS CANTOS e a diferir só em quais casas o par nomeia. O que muda é
// a forma: o traço é a linha entre eles, o retângulo é tudo o que cabe dentro.

// CasasDoRetangulo são todas as casas entre os dois cantos, inclusive eles.
//
// A ORDEM dos cantos não importa: arrastar da direita para a esquerda ou de baixo
// para cima desenha o mesmo retângulo, porque é o que o dedo faz. Sem o `min`/`max`
// um arrasto "para trás" devolveria vazio, e a pessoa concluiria que a ferramenta
// falha às vezes — que é a pior forma de falhar.
//
// Exemplo:
//
//	CasasDoRetangulo(engine.Square{X: 2, Y: 1}, engine.Square{X: 0, Y: 0})
//	// → as 6 casas de (0,0) a (2,1)
func CasasDoRetangulo(de, ate engine.Square) []engine.Square {
	x0, x1 := min(de.X, ate.X), max(de.X, ate.X)
	y0, y1 := min(de.Y, ate.Y), max(de.Y, ate.Y)
	casas := make([]engine.Square, 0, (x1-x0+1)*(y1-y0+1))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			casas = append(casas, engine.Square{X: x, Y: y})
		}
	}
	return casas
}

// oRetanguloCabe é o teto de casas de um retângulo, e ele é maior que o do traço
// pela mesma razão que o traço tem um menor: o retângulo é UM gesto deliberado —
// dois cantos escolhidos —, enquanto o traço é um quadro de 16ms. Mil casas são
// 32×32, que é uma sala grande de masmorra; acima disso é pedido forjado.
const oRetanguloCabe = 1000

// RetanguloValido recusa a área que não pode ter saído de dois cantos escolhidos.
//
// Recusa em vez de cortar, como o traço: um retângulo cortado enche uma parte e
// some com a outra, e quem pediu conclui que a ferramenta falha às vezes.
func RetanguloValido(de, ate engine.Square) bool {
	largura, altura := abs(ate.X-de.X)+1, abs(ate.Y-de.Y)+1
	return largura*altura <= oRetanguloCabe
}
