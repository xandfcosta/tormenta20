package engine

import "testing"

// Movimento no mapa de batalha — a regra vem do livro e o teste cita a página.
// O que se prova aqui é aritmética de mesa: o mestre e o jogador discordarem
// sobre "cabe ou não cabe" é a discussão que o app existe para encerrar.

func path(steps ...Square) []Square { return steps }

// p106: "seu deslocamento é 9 metros (6 quadrados no mapa)". É o exemplo
// trabalhado do próprio livro, e o caso mais comum da mesa inteira.
func TestSixOrthogonalStepsFitInNineMetres(t *testing.T) {
	cost := PathCost(
		path(Square{0, 0}, Square{1, 0}, Square{2, 0}, Square{3, 0}, Square{4, 0}, Square{5, 0}, Square{6, 0}),
		MoveTerrain{}, SquaresForDisplacement(9),
	)

	if cost.Squares != 6 || !cost.Legal {
		t.Errorf("seis passos retos com deslocamento 9m: %+v", cost)
	}
	if cost.StoppedAt != -1 {
		t.Errorf("caminho que coube marcou parada em %d", cost.StoppedAt)
	}
}

// p238: "mover-se na diagonal custa o dobro. Ou seja, andar 1,5m (1 quadrado)
// na diagonal conta como 3m (2 quadrados)". Esta é a asserção que pega quem
// portar por hábito a alternância 1-2-1 de outros d20 — que o livro NÃO tem.
func TestDiagonalStepCostsTwoSquaresAlways(t *testing.T) {
	one := PathCost(path(Square{0, 0}, Square{1, 1}), MoveTerrain{}, -1)
	if one.Squares != 2 {
		t.Errorf("uma diagonal custou %d quadrados, esperado 2", one.Squares)
	}

	// Três diagonais seguidas custam 6, não 4 nem 5: não há desconto na
	// segunda, que é exatamente o que a alternância faria.
	three := PathCost(path(Square{0, 0}, Square{1, 1}, Square{2, 2}, Square{3, 3}), MoveTerrain{}, 6)
	if three.Squares != 6 || !three.Legal {
		t.Errorf("três diagonais com deslocamento 9m: %+v", three)
	}

	four := PathCost(
		path(Square{0, 0}, Square{1, 1}, Square{2, 2}, Square{3, 3}, Square{4, 4}), MoveTerrain{}, 6,
	)
	if four.Legal {
		t.Error("a quarta diagonal (8 quadrados) coube num deslocamento de 6")
	}
	// Onde estourou importa: a tela pinta o trecho recusado em vez de recusar
	// o caminho inteiro.
	if four.StoppedAt != 4 {
		t.Errorf("estourou no passo %d, esperado o quarto", four.StoppedAt)
	}
}

// p238: terreno difícil "gasta 3m de deslocamento por quadrado, em vez de 1,5m".
func TestDifficultTerrainDoublesTheStep(t *testing.T) {
	mud := MoveTerrain{Difficult: map[Square]bool{{X: 1, Y: 0}: true}}

	cost := PathCost(path(Square{0, 0}, Square{1, 0}, Square{2, 0}), mud, -1)

	// Um passo na lama (2) + um no chão limpo (1).
	if cost.Squares != 3 {
		t.Errorf("dois passos, um deles em terreno difícil: %d quadrados, esperado 3", cost.Squares)
	}
	// O custo é de ENTRAR: sair da lama para o chão limpo custa o do chão limpo.
	leaving := PathCost(path(Square{1, 0}, Square{2, 0}), mud, -1)
	if leaving.Squares != 1 {
		t.Errorf("sair do terreno difícil custou %d, esperado 1", leaving.Squares)
	}
}

// O livro dobra a diagonal (p238) e dobra o terreno difícil (p238) em frases
// SEPARADAS, e nunca compõe as duas. A leitura desta casa é multiplicativa —
// decisão de mesa registrada, não texto do livro. Está fixado aqui para que
// mudar de ideia seja uma mudança visível, e não uma descoberta na mesa.
func TestDiagonalIntoDifficultTerrainCostsFour(t *testing.T) {
	brush := MoveTerrain{Difficult: map[Square]bool{{X: 1, Y: 1}: true}}

	cost := PathCost(path(Square{0, 0}, Square{1, 1}), brush, -1)

	if cost.Squares != 4 {
		t.Errorf("diagonal entrando em terreno difícil custou %d quadrados, esperado 4 (6m)", cost.Squares)
	}
}

// Caminho tem de ser quadrado a quadrado: um salto no meio significa que o
// cliente mandou lixo, e aceitar isso mediria uma distância que ninguém andou.
func TestPathMustBeContiguous(t *testing.T) {
	cost := PathCost(path(Square{0, 0}, Square{4, 0}), MoveTerrain{}, 6)

	if cost.Legal {
		t.Error("um salto de quatro quadrados passou como caminho")
	}
	// A mensagem carrega os valores ofendidos — quem recebe precisa saber o que
	// mandou de errado.
	if cost.Reason == "" {
		t.Error("caminho recusado sem dizer por quê")
	}
}

func TestSquaresForDisplacementTruncates(t *testing.T) {
	// p106: 9m = 6 quadrados. É a linha trabalhada do livro.
	if got := SquaresForDisplacement(9); got != 6 {
		t.Errorf("9m viraram %d quadrados, esperado 6", got)
	}
	// 10m não compra um sétimo quadrado: o sétimo custa 1,5m inteiros.
	if got := SquaresForDisplacement(10); got != 6 {
		t.Errorf("10m viraram %d quadrados, esperado 6", got)
	}
	// Sobrecarga tira 3m (p238): 9 − 3 = 6m = 4 quadrados.
	if got := SquaresForDisplacement(6); got != 4 {
		t.Errorf("6m viraram %d quadrados, esperado 4", got)
	}
	if got := SquaresForDisplacement(0); got != 0 {
		t.Errorf("quem não anda recebeu %d quadrados", got)
	}
}

// p107, Tab. 1-21. Fixa só o que é EXCEÇÃO à regra "ocupa um quadrado" — os três
// tamanhos que ocupam 1×1 são o default, e repeti-los seria transcrever tabela.
func TestFootprintFollowsTheSizeTable(t *testing.T) {
	cases := map[string]int{
		"Grande":   2,
		"Enorme":   3,
		"Colossal": 6,
		// As duas grafias do projeto convergem: o bestiário guarda "medio",
		// a ficha guarda "Médio", e uma peça de tamanho errado no tabuleiro
		// erraria alcance e ocupação de quadrado.
		"medio":  1,
		"Médio":  1,
		"grande": 2,
	}
	for size, want := range cases {
		if got := FootprintForSize(size); got != want {
			t.Errorf("%q ocupou %d quadrados de lado, esperado %d", size, got, want)
		}
	}
	// Tamanho desconhecido cai em 1: uma peça sem tamanho é Média, nunca some.
	if got := FootprintForSize("inventado"); got != 1 {
		t.Errorf("tamanho desconhecido virou %d", got)
	}
}

// O alcance é um LOSANGO, não um quadrado — e essa é a consequência VISÍVEL da
// diagonal dobrada (T20 p238): com 6 quadrados de deslocamento (9m, p106)
// dá para andar 6 em linha reta e só 3 na diagonal.
func TestReachIsADiamondBecauseOfTheDiagonal(t *testing.T) {
	reach, _, _ := ReachFromStops([]Square{{X: 0, Y: 0}}, 6, MoveTerrain{})

	inside := map[Square]bool{}
	for _, s := range reach {
		inside[s] = true
	}

	if !inside[Square{X: 6, Y: 0}] {
		t.Error("seis quadrados em linha reta não alcançaram: o deslocamento de 9m anda 6 (p106)")
	}
	if !inside[Square{X: 3, Y: 3}] {
		t.Error("três diagonais custam 6 e deveriam caber no orçamento de 6")
	}
	if inside[Square{X: 4, Y: 4}] {
		t.Error("quatro diagonais custam 8 e passaram por um orçamento de 6: a diagonal não está dobrando")
	}
	if inside[Square{X: 7, Y: 0}] {
		t.Error("sete quadrados em linha reta passaram por um orçamento de 6")
	}
	// A origem não é destino: acender a casa onde a peça já está seria oferecer
	// um movimento que não move.
	if inside[Square{X: 0, Y: 0}] {
		t.Error("a origem entrou na lista de casas alcançáveis")
	}
}

// Sem orçamento não há casas acesas: fora de combate a régua mede, mas não há
// limite para desenhar, e uma busca sem teto não termina num plano infinito.
func TestWithoutABudgetThereAreNoSquaresToLight(t *testing.T) {
	if got, _, _ := ReachFromStops([]Square{{X: 2, Y: 2}}, -1, MoveTerrain{}); len(got) != 0 {
		t.Errorf("orçamento negativo devolveu %d casas, esperava nenhuma", len(got))
	}
}

// Terreno difícil encolhe o alcance pela mesma conta do passo (p238): 3m por
// quadrado, ou seja, o dobro.
func TestDifficultTerrainShrinksTheReach(t *testing.T) {
	mud := MoveTerrain{Difficult: map[Square]bool{{X: 1, Y: 0}: true, {X: 2, Y: 0}: true}}

	reach, _, _ := ReachFromStops([]Square{{X: 0, Y: 0}}, 4, MoveTerrain{})
	inMud, _, _ := ReachFromStops([]Square{{X: 0, Y: 0}}, 4, mud)

	if len(inMud) >= len(reach) {
		t.Errorf("a lama não encolheu o alcance: %d contra %d", len(inMud), len(reach))
	}
	// (3,0) custa 5 por qualquer caminho: pela linha da lama, 1+2+2; pelo
	// contorno diagonal, 2 (diagonal) + 1 + 2 (diagonal). Os dois passam de 4.
	for _, s := range inMud {
		if s == (Square{X: 3, Y: 0}) {
			t.Error("(3,0) alcançado com 4 quadrados atravessando dois de terreno difícil")
		}
	}
}

// A CONTAGEM das dobras existe para a tela poder nomear a regra que produziu o
// número, e por isso ela sai do mesmo laço que cobrou o caminho: um texto
// escrito à mão no cliente divergiria do motor.
func TestPathCostCountsWhichRuleDoubledEachStep(t *testing.T) {
	swamp := MoveTerrain{Difficult: map[Square]bool{{X: 2, Y: 0}: true}}

	// Um reto limpo, um reto no brejo, uma diagonal limpa: 1 + 2 + 2 = 5.
	cost := PathCost(path(Square{0, 0}, Square{1, 0}, Square{2, 0}, Square{3, 1}), swamp, -1)

	if cost.Squares != 5 {
		t.Fatalf("custo = %d quadrados, esperado 5 (1 reto + 2 brejo + 2 diagonal)", cost.Squares)
	}
	if cost.Diagonals != 1 {
		t.Errorf("diagonais contadas = %d, esperado 1", cost.Diagonals)
	}
	if cost.Difficult != 1 {
		t.Errorf("passos em terreno difícil contados = %d, esperado 1", cost.Difficult)
	}
}

// Um passo pode dobrar pelas DUAS regras, e a leitura desta casa é
// multiplicativa: uma diagonal em terreno difícil custa 4 quadrados (6m). Os
// contadores registram as duas causas, senão a tela diria "terreno difícil"
// sobre um passo que também era diagonal — e a mesa aprenderia a regra errada.
func TestDiagonalInDifficultTerrainCountsBothRules(t *testing.T) {
	swamp := MoveTerrain{Difficult: map[Square]bool{{X: 1, Y: 1}: true}}

	cost := PathCost(path(Square{0, 0}, Square{1, 1}), swamp, -1)

	if cost.Squares != 4 {
		t.Fatalf("diagonal no brejo custou %d, esperado 4", cost.Squares)
	}
	if cost.Diagonals != 1 || cost.Difficult != 1 {
		t.Errorf("um passo, duas causas: diagonais=%d difícil=%d, esperado 1 e 1", cost.Diagonals, cost.Difficult)
	}
}

// Caminho reto e limpo não inventa dobra nenhuma: o zero é o que faz a tela
// calar em vez de anunciar uma regra que não agiu.
func TestStraightCleanPathCountsNoDoubling(t *testing.T) {
	cost := PathCost(path(Square{0, 0}, Square{1, 0}, Square{2, 0}), MoveTerrain{}, 6)

	if cost.Diagonals != 0 || cost.Difficult != 0 {
		t.Errorf("caminho reto e limpo contou dobras: diagonais=%d difícil=%d", cost.Diagonals, cost.Difficult)
	}
}

// ── o caminho entre dois quadrados ───────────────────────────────────────────

func TestThePathStartsAtTheOriginAndEndsAtTheDestination(t *testing.T) {
	path := PathBetween(Square{X: 0, Y: 0}, Square{X: 3, Y: 1})

	if path[0] != (Square{X: 0, Y: 0}) {
		t.Errorf("o caminho começa em %+v", path[0])
	}
	if end := path[len(path)-1]; end != (Square{X: 3, Y: 1}) {
		t.Errorf("o caminho termina em %+v", end)
	}
	if cost := PathCost(path, MoveTerrain{}, -1); !cost.Legal {
		t.Errorf("o caminho desenhado é ilegal para o motor: %s", cost.Reason)
	}
}

// A DIAGONAL vem primeiro porque é o que o olho espera de quem corta caminho —
// e o custo é o mesmo em L, que é a razão de a escolha poder ser estética.
func TestTheDiagonalComesFirstAndCostsTheSameAsTheL(t *testing.T) {
	diagonalFirst := PathBetween(Square{X: 0, Y: 0}, Square{X: 3, Y: 1})
	emL := []Square{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 0}, {X: 3, Y: 1}}

	if segundo := diagonalFirst[1]; segundo != (Square{X: 1, Y: 1}) {
		t.Errorf("o segundo passo foi %+v, e a diagonal devia vir primeiro", segundo)
	}
	withDiagonal := PathCost(diagonalFirst, MoveTerrain{}, -1).Squares
	withL := PathCost(emL, MoveTerrain{}, -1).Squares
	if withDiagonal != withL {
		t.Errorf("diagonal custou %d e o L custou %d; com a diagonal valendo o dobro os dois têm de empatar (p238)", withDiagonal, withL)
	}
}

// A peça que não sai do lugar tem caminho de UM quadrado. Zero quadrados faria
// o `ProposeMove` recusar com "precisa de origem e destino", que é a mensagem
// errada para quem só soltou a peça onde ela estava.
func TestATokenThatDoesNotLeaveItsPlaceHasAOneSquarePath(t *testing.T) {
	path := PathBetween(Square{X: 2, Y: 2}, Square{X: 2, Y: 2})
	if len(path) != 1 || path[0] != (Square{X: 2, Y: 2}) {
		t.Errorf("o caminho parado ficou %+v", path)
	}
}

// COORDENADA NEGATIVA é lugar legítimo: o plano não tem bordas, e um sinal
// invertido faria o laço andar para longe do destino e nunca terminar.
func TestThePathWalksIntoNegativeCoordinates(t *testing.T) {
	path := PathBetween(Square{X: 1, Y: 1}, Square{X: -2, Y: -3})
	if end := path[len(path)-1]; end != (Square{X: -2, Y: -3}) {
		t.Errorf("o caminho terminou em %+v", end)
	}
	if len(path) != 5 {
		t.Errorf("o caminho tem %d quadrados; de (1,1) a (-2,-3) são 4 passos mais a origem", len(path))
	}
}

// ── o movimento por PARADAS ──────────────────────────────────────────────────

// A EMENDA não repete o quadrado da parada: ele é o fim de um segmento e o
// começo do outro, e repeti-lo poria no meio do caminho um passo que não anda —
// e o `PathCost` mede passo a passo.
func TestTheStopsJoinWithoutRepeatingTheSquare(t *testing.T) {
	path := PathThroughStops([]Square{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}})

	want := []Square{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 1}, {X: 2, Y: 2}}
	if len(path) != len(want) {
		t.Fatalf("o caminho tem %d quadrados: %+v", len(path), path)
	}
	for i := range want {
		if path[i] != want[i] {
			t.Errorf("passo %d é %+v, queria %+v", i, path[i], want[i])
		}
	}
	if cost := PathCost(path, MoveTerrain{}, -1); !cost.Legal {
		t.Errorf("o caminho emendado é ilegal para o motor: %s", cost.Reason)
	}
}

// Uma parada IGUAL à anterior não acrescenta nada: quem soltou a peça onde ela
// já estava não gastou movimento.
func TestARepeatedStopAddsNoStep(t *testing.T) {
	path := PathThroughStops([]Square{{X: 1, Y: 1}, {X: 1, Y: 1}, {X: 2, Y: 1}})
	if len(path) != 2 {
		t.Errorf("o caminho ficou %+v; a parada repetida virou passo", path)
	}
}

// A ROTA É ESCOLHA DE QUEM JOGA, e ela custa o que custar — é por isso que as
// paradas existem: sem elas o cliente desenha a reta e pronto, e o contorno que
// alguém faria para não passar ao lado de um inimigo não tem como ser expresso.
func TestGoingAroundCostsMoreAndThatIsThePoint(t *testing.T) {
	straight := PathThroughStops([]Square{{X: 0, Y: 0}, {X: 4, Y: 0}})
	goingAround := PathThroughStops([]Square{{X: 0, Y: 0}, {X: 2, Y: 2}, {X: 4, Y: 0}})

	straightCost := PathCost(straight, MoveTerrain{}, -1).Squares
	aroundCost := PathCost(goingAround, MoveTerrain{}, -1).Squares
	if aroundCost <= straightCost {
		t.Errorf("o contorno custou %d e a reta %d; o desvio tem de custar mais", aroundCost, straightCost)
	}
}

// E COM TERRENO DIFÍCIL o desvio pode custar MENOS que a reta.
func TestInDifficultTerrainTheDetourCanCostLessThanTheStraightLine(t *testing.T) {
	// Lama exatamente sobre a reta de (0,0) a (4,0).
	mud := MoveTerrain{Difficult: map[Square]bool{{X: 1, Y: 0}: true, {X: 2, Y: 0}: true, {X: 3, Y: 0}: true}}

	// O desvio sobe UMA fileira e anda por fora, em vez de cortar em diagonal:
	// quatro diagonais custam 2 cada e sairiam mais caras que atravessar a lama —
	// o exemplo não mediria evitar nada.
	straight := PathThroughStops([]Square{{X: 0, Y: 0}, {X: 4, Y: 0}})
	detouring := PathThroughStops([]Square{{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 3, Y: 1}, {X: 4, Y: 0}})

	straightCost := PathCost(straight, mud, -1).Squares
	detourCost := PathCost(detouring, mud, -1).Squares
	if detourCost >= straightCost {
		t.Errorf("atravessar a lama custou %d e contorná-la custou %d; era para contornar sair mais barato", straightCost, detourCost)
	}
	// O CONTROLE: sem lama a desigualdade se INVERTE, e é isso que prova que o
	// que a mudou foi o terreno e não a forma do caminho.
	if PathCost(detouring, MoveTerrain{}, -1).Squares <= PathCost(straight, MoveTerrain{}, -1).Squares {
		t.Error("sem lama o desvio devia custar mais que a reta")
	}
}

// O FEEDBACK: quanto sobra e até onde dá para ir a partir da última parada.
//
// Sem ele a pessoa empilha paradas, o total passa do deslocamento, e ela não
// sabe o que desfazer para corrigir.
func TestTheReachShrinksAtEachStopAndHitsZeroAtTheEnd(t *testing.T) {
	const budget = 6

	reach, segundo, remaining := ReachFromStops([]Square{{X: 0, Y: 0}}, budget, MoveTerrain{})
	if remaining != budget {
		t.Errorf("sem andar, sobravam %d de %d", remaining, budget)
	}
	if len(reach) == 0 {
		t.Fatal("com orçamento inteiro não havia para onde ir")
	}

	after, _, remainingAfter := ReachFromStops([]Square{{X: 0, Y: 0}, {X: 2, Y: 0}}, budget, MoveTerrain{})
	if remainingAfter != budget-2 {
		t.Errorf("depois de andar 2, sobravam %d", remainingAfter)
	}
	if len(after) >= len(reach) {
		t.Errorf("o alcance não encolheu: %d antes, %d depois", len(reach), len(after))
	}

	// Gastou tudo: o alcance é VAZIO, e a tela diz "acabou" em vez de oferecer
	// casas que o servidor recusaria.
	end, stillSecond, noSpare := ReachFromStops([]Square{{X: 0, Y: 0}, {X: 6, Y: 0}}, budget, MoveTerrain{})
	if noSpare != 0 {
		t.Errorf("depois de gastar tudo, sobravam %d", noSpare)
	}
	if len(end) != 0 {
		t.Errorf("com orçamento zerado o alcance ofereceu %d casas", len(end))
	}
	// E A SEGUNDA FAIXA CONTINUA OFERECENDO: gastar a ação de movimento inteira
	// não encerra o turno — sobra a ação padrão para trocar por outra (p233), e é
	// disso que o azul fala. Sem esta linha, "o ouro secou" seria lido como
	// "acabou", que é a leitura antiga e agora está errada.
	if len(stillSecond) == 0 {
		t.Error("com a ação de movimento gasta, o segundo movimento não ofereceu casa nenhuma")
	}
	// E o CONTROLE do começo: sem andar nada, as duas faixas existem e a segunda
	// é a de fora — ela alcança o que a primeira não alcança.
	if len(segundo) == 0 {
		t.Error("sem andar, a segunda faixa nasceu vazia")
	}
	for _, q := range segundo {
		if RangeSquares(Square{}, q) <= 0 {
			t.Fatalf("a segunda faixa trouxe a própria casa da peça: %+v", q)
		}
	}
}

// AS DUAS FAIXAS SÃO DISJUNTAS e cobrem tudo até 2× o deslocamento (T20 p233).
//
// É o que faz o mapa não pintar a mesma casa de duas cores, e é a garantia que a
// leitura pedida pelo dono depende: "até onde vou com uma ação" e "até onde vou
// gastando as duas" só são duas perguntas se as respostas não se sobrepõem.
func TestTheTwoReachBandsDoNotOverlap(t *testing.T) {
	const budget = 4

	inside, segundo, _ := ReachFromStops([]Square{{}}, budget, MoveTerrain{})

	visited := map[Square]bool{}
	for _, q := range inside {
		visited[q] = true
	}
	for _, q := range segundo {
		if visited[q] {
			t.Errorf("a casa %+v foi pintada nas duas faixas", q)
		}
	}
	// OS NÚMEROS SÃO ESCRITOS À MÃO: derivá-los de uma segunda chamada à
	// implementação é o "esperado calculado" que o CLAUDE.md proíbe — um erro na
	// conta sairia dos DOIS lados e o guarda ficaria verde.
	//
	// A conta vem da REGRA. Com a diagonal custando o dobro (T20 p238), um passo
	// diagonal (2) vale dois ortogonais (1+1), então o custo até (dx,dy) é
	// |dx|+|dy| — o alcance é um losango de Manhattan. As casas com |x|+|y| ≤ B
	// são 2B²+2B+1, e tirando a origem (que não é destino) sobram 2B²+2B.
	//
	//	B=4 → 2·16+8  =  40 casas com UMA ação de movimento
	//	B=8 → 2·64+16 = 144 casas com as DUAS (p233)
	const oneAction, twoActions = 40, 144
	if len(inside) != oneAction {
		t.Errorf("a faixa de dentro tem %d casas, e o losango de raio %d tem %d",
			len(inside), budget, oneAction)
	}
	if soma := len(inside) + len(segundo); soma != twoActions {
		t.Errorf("as duas faixas somam %d casas, e o losango de raio %d tem %d",
			soma, 2*budget, twoActions)
	}
}
