package table

import (
	"fmt"

	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O MOVIMENTO PROPOSTO, do ponto de vista de quem olha.
//
// Saiu do `board_view.go` porque muda por outra razão: aquele arquivo muda
// quando a FORMA do que se desenha muda — a peça ganha um campo, o marcador
// ganha uma cor —, e este muda quando a regra de DESLOCAMENTO muda (p105).
// Juntos davam 1240 linhas, e o teto do guia é 500 (ALE-360).
//
// A conta continua sendo do `domain/board`: aqui só se traduz o que ele decidiu
// para o que a tela desenha — a trilha em coordenadas de tela, a legenda das
// faixas, o saldo em metros. Nenhuma regra nasce neste arquivo.

// moveView é o movimento proposto, do ponto de vista de quem olha.
type moveView struct {
	TokenID string
	Rotulo  string
	// Trilha são as casas por onde a peça passa, já em Coordinate da tela.
	Trilha []boardSquare
	Custo  int
	// Orcamento -1 é "sem orçamento": o mestre move qualquer peça a qualquer
	// hora, e fora de combate cada um anda com a sua. Nesses casos não há
	// alcance para desenhar, porque não há teto que ele desenharia.
	Orcamento int
	Restante  int
	// NoActionLeft é "o turno de quem está na vez não paga este caminho". Ele é
	// SEPARADO do `Orcamento`, que fala de metros: a peça pode ter deslocamento
	// de sobra e o turno já ter acabado.
	NoActionLeft bool
	// Meu diz se quem olha decide sobre este movimento. O mestre decide por
	// qualquer um — é ele quem toca a mesa.
	Meu bool
	// Paradas são as casas onde a pessoa CLICOU, sem a primeira nem a última:
	// elas viram um pingo na trilha, e é ele que faz o "Desfazer parada" ter o
	// que desfazer aos olhos de quem clica. As duas pontas ficam de fora porque
	// já têm desenho próprio — a origem é o FANTASMA e o fim é a PEÇA.
	Paradas []boardSquare
	// PodeDesfazer é ter mais de UMA perna. Com uma só, desfazer é cancelar — e o
	// Cancelar está ali do lado, dizendo isso com a palavra certa.
	PodeDesfazer bool
	// Origem é a casa de onde a peça SAIU, e ela existe porque a peça deixou de
	// ficar lá: a peça é desenhada onde foi SOLTA, e quem marca o começo do
	// movimento é o fantasma nesta casa.
	Origem boardSquare
	// Fim é a casa onde a peça pousa, que é o fim do caminho. É dela que o
	// arrasto da próxima parada conta o deslocamento — a peça está lá.
	Fim boardSquare
	// Fio é o `d` da seta que liga o fantasma à peça, dobrando nas paradas. Vem
	// pronto do servidor porque o caminho é dele; ver
	// `move_drawing.go` para por que ela dobra na PARADA e não
	// em cada casa.
	Fio string
	// FioSegundo é o trecho que passa da ação de movimento e ainda cabe na ação
	// PADRÃO trocada por movimento (T20 p233): sai AZUL.
	//
	// FioAlem é o que passa das duas: sai VERMELHO, e não há terceira ação de
	// movimento no turno para pagá-lo.
	//
	// Os dois são vazios quando o caminho cabe, e também fora de combate — sem vez
	// não há ação padrão para trocar, e desenhar as faixas inventaria um teto.
	FioSegundo string
	FioAlem    string
	// Pernas são os rótulos em metros, um por trecho entre duas paradas. Eles
	// contam o CUSTO da perna e não a distância geométrica dela, para que o metro
	// do rótulo seja o mesmo metro que decide onde o `FioAlem` começa.
	Pernas []moveLeg
}

// moveBoard monta o movimento em curso, ou nil quando não há.
//
// O ALCANCE só é desenhado para quem PODE decidir: oferecer casas clicáveis a
// quem não vai poder confirmar é convidar para um beco.
func moveBoard(b *board.BoardState, st *live.SessionRuntimeState, m board.Mover) *moveView {
	if b == nil || b.Pending == nil {
		return nil
	}
	p := b.Pending
	peca := board.FindToken(b, p.TokenID)
	if peca == nil {
		return nil
	}
	v := &moveView{
		TokenID: p.TokenID, Rotulo: peca.Label, Custo: p.Cost, Orcamento: p.Budget,
		Meu: m.Role == "gm" || p.ByUserID == m.UserID,
		// O CAMINHO CABER NO DESLOCAMENTO E O TURNO NÃO PAGAR são perguntas
		// diferentes, e a segunda é a que faz a tela mentir quando falta: um
		// passo de 1,5m cabe em qualquer orçamento e não acontece se a padrão e
		// a de movimento já foram (p233).
		NoActionLeft: turnCannotPay(b, st, p.Cost, p.Budget),
	}
	for _, q := range p.Path {
		v.Trilha = append(v.Trilha, boardSquare{X: q.X, Y: q.Y})
	}
	// As duas PONTAS do caminho têm desenho de peça: o fantasma sai da origem e
	// a peça pousa no fim.
	if len(p.Path) > 0 {
		inicio, fim := p.Path[0], p.Path[len(p.Path)-1]
		v.Origem = boardSquare{X: inicio.X, Y: inicio.Y}
		v.Fim = boardSquare{X: fim.X, Y: fim.Y}
	}
	// A SETA, os RÓTULOS em metros e a divisão dourado/vermelho saem das MESMAS
	// dobras e do MESMO terreno: é o que faz o número escrito sobre a linha
	// explicar a cor dela em vez de contradizê-la.
	dobras := moveFolds(p)
	custos := legsCosts(dobras, moveTerrain(b))
	v.Fio, v.FioSegundo, v.FioAlem = moveWires(dobras, custos, p.Budget)
	v.Pernas = moveLegs(dobras, custos)
	// As paradas INTERMEDIÁRIAS: a última é onde a peça pousou e a primeira é de
	// onde ela saiu — as duas já são um disco na tela, e marcá-las de novo
	// contaria a mesma coisa duas vezes.
	if len(p.Stops) > 2 {
		v.PodeDesfazer = true
		for _, q := range p.Stops[1 : len(p.Stops)-1] {
			v.Paradas = append(v.Paradas, boardSquare{X: q.X, Y: q.Y})
		}
	}
	// O `Restante` é preenchido pelo chamador: ele sai da MESMA chamada que
	// desenha o alcance, e recalculá-lo aqui seria a segunda conta da regra.
	return v
}

// moveTerrain traduz o terreno difícil do tabuleiro para o motor.
//
// Existe porque o `moveTerrainOf` do pacote `board` é privado, e duplicar a
// TRADUÇÃO é barato — duplicar a REGRA não seria. Se um dia ela virar três
// linhas, ela sobe para lá.
func moveTerrain(b *board.BoardState) engine.MoveTerrain {
	if len(b.Difficult) == 0 {
		return engine.MoveTerrain{}
	}
	dificil := make(map[engine.Square]bool, len(b.Difficult))
	for _, q := range b.Difficult {
		dificil[q] = true
	}
	return engine.MoveTerrain{Difficult: dificil}
}

// reachAndTarget é a peça que quem olha pode COMEÇAR a mover agora, ou "".
//
// Uma só e não uma lista: a Mesa move uma peça por vez, e com um movimento em
// curso a resposta é vazia.
//
// **Quem responde é o `board.CanMove`, não esta função** — é o mesmo
// `assertMovable` que a ESCRITA usa. Perguntar de outro jeito na tela é como
// nasce um botão que existe e o servidor recusa. E o estado da sessão tem de ir
// junto: sem ele o `assertMovable` lê "fora de combate" e libera, e a tela
// ofereceria mover a peça do jogador fora da vez dele.
//
// O `Restante` sai daqui junto com as casas porque é a MESMA conta. Duas
// chamadas com os mesmos argumentos, cada uma jogando fora metade, é como este
// repositório já mostrou dois números diferentes para o mesmo combatente.
func reachAndTarget(b *board.BoardState, st *live.SessionRuntimeState, quem board.Mover, meus map[int64]bool) boardReach {
	if b == nil {
		return boardReach{}
	}
	var alvo, rotulo string
	// COM movimento em curso o alvo é a peça dele, e o alcance sai do fim do
	// caminho com o que sobrou. Sem, é a primeira peça que quem olha pode mover,
	// e o alcance sai de onde ela está com o orçamento inteiro.
	de := []engine.Square(nil)
	orcamento := 0
	if p := b.Pending; p != nil && (quem.Role == "gm" || p.ByUserID == quem.UserID) {
		if peca := board.FindToken(b, p.TokenID); peca != nil {
			alvo, rotulo, de, orcamento = p.TokenID, peca.Label, p.Path, p.Budget
		}
	}
	if alvo == "" && b.Pending == nil {
		for i := range b.Tokens {
			// A POSSE é por PEÇA e não por pessoa: o `Mover` carrega um booleano
			// só, e deixá-lo em falso aqui tira o alcance do jogador NA VEZ dele
			// — a tela diria que ele não pode mover a própria peça.
			//
			// Quem responde é o `meus`, montado contra o banco pelo `tableRoster`:
			// a ponte até a pessoa é o DONO do personagem.
			dela := quem
			if id := b.Tokens[i].CharacterID; id != nil {
				dela.OwnsCharacter = meus[*id]
			}
			podeMover, orcamentoDela := board.CanMoveWith(b, st, b.Tokens[i].ID, dela)
			if !podeMover {
				continue
			}
			alvo, rotulo = b.Tokens[i].ID, b.Tokens[i].Label
			de = []engine.Square{{X: b.Tokens[i].X, Y: b.Tokens[i].Y}}
			orcamento = orcamentoDela
			break
		}
	}
	if alvo == "" {
		return boardReach{}
	}
	dentro, segundo, restante := engine.ReachFromStops(de, orcamento, moveTerrain(b))
	return boardReach{
		Alvo: alvo, Rotulo: rotulo, Restante: restante,
		Dentro: screenSquares(dentro), Segundo: screenSquares(segundo),
	}
}

// boardReach junta o que sai de UMA pergunta: qual peça o clique move, e
// até onde ela vai com cada uma das duas ações de movimento (T20 p233).
//
// Struct e não seis valores de retorno porque as partes só fazem sentido juntas
// — o `Restante` é medido a partir do mesmo caminho que produz as faixas —, e
// porque a lista de retornos já tinha passado de quatro.
type boardReach struct {
	Alvo     string
	Rotulo   string
	Dentro   []boardSquare
	Segundo  []boardSquare
	Restante int
}

func screenSquares(casas []engine.Square) []boardSquare {
	out := make([]boardSquare, 0, len(casas))
	for _, q := range casas {
		out = append(out, boardSquare{X: q.X, Y: q.Y})
	}
	return out
}

// moveBalance diz o que SOBRA do deslocamento, ou "" quando o caminho passou.
//
// O `Restante` não serve para dizer o excesso: o `reachAndTarget` o trava em
// zero de propósito, porque ele alimenta o desenho das casas alcançáveis e
// alcance negativo não é lugar nenhum.
//
// @example moveBalance(&moveView{Custo: 4, Orcamento: 6, Restante: 2}) // "sobram 2"
func moveBalance(m *moveView) string {
	if m.Custo <= m.Orcamento {
		return fmt.Sprintf("sobram %d", m.Restante)
	}
	// PASSANDO DO DESLOCAMENTO não há saldo a dizer, e quem conta a história é a
	// LEGENDA logo abaixo. Um "além do deslocamento" mediria a mesma coisa que a
	// faixa acesa, e fica ambíguo com dois limiares — além de QUAL dos dois? O
	// metro por perna continua escrito sobre a seta, que é onde ele explica a cor.
	return ""
}

// spentActions nomeia o que o caminho CUSTA em ações do turno (T20 p233).
//
// Em AÇÕES e não em quadrados: "13 de 6" diz que estourou, não diz que estourar
// aqui é legítimo e custa o turno inteiro.
//
// A frase é a MESMA leitura das cores da seta, em palavras: quem não distingue o
// azul do vermelho no mapa lê aqui, e quem lê o mapa confirma aqui.
//
// @example spentActions(&moveView{Custo: 8, Orcamento: 6}) // "ação de movimento + ação principal"
func spentActions(m *moveView) string {
	return rangesThree[costRange(m)].Texto
}

// turnCannotPay diz se o turno de quem está na vez NÃO tem como pagar este
// caminho. Fora de uma cena que conta rodadas, ou movendo peça de quem não está
// na vez, não há o que pagar — e a resposta é não.
//
// Quem decide é o MOTOR, e não uma segunda conta aqui: as ações de movimento
// que o caminho pede são gastas uma a uma contra o que sobrou, então a troca da
// padrão (p233) vale aqui exatamente como vale na cobrança.
func turnCannotPay(b *board.BoardState, st *live.SessionRuntimeState, cost, orcamento int) bool {
	if orcamento <= 0 || !movedTokenIsOnTurn(st, b) || st.Scene == nil || !st.Scene.CountsRounds() {
		return false
	}
	left := engine.TurnBudget{Standard: st.Scene.StandardLeft, Movement: st.Scene.MovementLeft}
	for paid := 0; paid < cost; paid += orcamento {
		next, err := left.Spend(engine.ActionMovement)
		if err != nil {
			return true
		}
		left = next
	}
	return false
}

// moveRange é uma linha da LEGENDA das cores, no rodapé do movimento.
//
// Um mapa que ensina a regra pela cor só ensina se disser o que a cor quer dizer
// — senão ele pede que a mesa adivinhe, e adivinhar cor é pior que não ter cor.
type moveRange struct {
	Classe string
	Texto  string
	// Ativa é a faixa em que o caminho INTEIRO cai, e é a que fica acesa. As
	// outras continuam na tela, apagadas: quem nunca viu o azul não descobriria
	// que ele existe se só a faixa da vez aparecesse.
	Ativa bool
}

// rangesThree são as faixas na ordem em que se gastam, e a ÚNICA lista delas.
//
// O texto daqui é o mesmo que o `spentActions` devolve, de propósito: a legenda
// e a frase do rodapé são a mesma leitura, e duas listas divergiriam no dia em
// que alguém reescrevesse uma — com a tela dizendo "gasta a ação principal" ao
// lado de uma bolinha que diz outra coisa.
var rangesThree = []moveRange{
	{Classe: "board-band-fits", Texto: "ação de movimento"},
	{Classe: "board-band-second", Texto: "ação de movimento + ação principal"},
	{Classe: "board-band-beyond", Texto: "não cabe no turno"},
}

// costRange diz em qual das três faixas o caminho INTEIRO cai (T20 p233).
//
// É o índice em `rangesThree`, e é a mesma conta que parte a seta — o que muda é
// a granularidade: a seta corta perna a perna, e isto olha o total. Por isso a
// legenda acesa concorda com a cor da PONTA da seta, que é onde o caminho acaba.
func costRange(m *moveView) int {
	switch {
	case m.Custo <= m.Orcamento:
		return 0
	case m.Custo <= 2*m.Orcamento:
		return 1
	default:
		return 2
	}
}

// moveLegend monta as três linhas, com a da vez acesa.
func moveLegend(m *moveView) []moveRange {
	ativa := costRange(m)
	legenda := make([]moveRange, 0, len(rangesThree))
	for i, f := range rangesThree {
		f.Ativa = i == ativa
		legenda = append(legenda, f)
	}
	return legenda
}

// endWireFits é a ponta da seta, ou `none` quando ela não é dele.
//
// A seta tem UMA ponta e ela vai no FIM do caminho. Havendo faixa depois desta,
// o dourado termina no MEIO do plano — no ponto em que a ação de movimento
// acabou —, e uma ponta ali apontaria para o nada e pareceria um segundo
// destino.
func endWireFits(m *moveView) string {
	return endForEnd(m, m.FioSegundo == "" && m.FioAlem == "", "move")
}

// endWireSecond e endWireBeyond completam a regra: a ponta vai em quem
// TERMINA o caminho, e cada faixa a carrega na cor dela.
func endWireSecond(m *moveView) string {
	return endForEnd(m, m.FioAlem == "", "second")
}

func endWireBeyond(m *moveView) string {
	return endForEnd(m, true, "beyond")
}

// endForEnd devolve a ponta da cor pedida, ou `none`.
//
// `none` e não atributo ausente: `marker-end` é escrito pela mesma linha do
// `.templ` nos dois casos, e o templ não aceita `else if` numa lista de
// atributos — os DOIS ramos sairiam e o navegador guardaria o primeiro.
func endForEnd(m *moveView, eOFim bool, cor string) string {
	if !eOFim {
		return "none"
	}
	return "url(#board-tip-" + cor + ")"
}
