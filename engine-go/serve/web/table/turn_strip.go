package table

import "t20engine/domain/live"

// A FAIXA DE QUEM VEM DEPOIS: a vez de agora e as duas seguintes, na ordem da
// mesa, dando a volta.
//
// Ler a lista não responde "chamo quem depois?" no ÚLTIMO da rodada — ali o
// próximo está no TOPO, e é justamente o turno em que a pergunta mais importa.
//
// Ela não substitui o "Próximo: Arwen" do botão, e a repetição é deliberada: a
// faixa é onde se LÊ a ordem, o botão é o que o CLIQUE vai fazer.

// turnAhead é um lugar na faixa.
type turnAhead struct {
	Label string
	// Mine é o meu personagem, e é o que faz a faixa responder "quanto falta para
	// mim?" — a pergunta do JOGADOR, que é quem mais precisa dela.
	Mine bool
	// Now é a vez em curso. Sempre a primeira, e escrito mesmo assim: o
	// desenho não deve depender da posição no laço para saber o que destacar.
	Now bool
	// WrapsRound marca o lugar onde a rodada seguinte começa — o ponto em que a
	// lista deu a volta. Sem ele a faixa mentiria por omissão: "Zumbi 2 › Ogro"
	// parece a mesma rodada, e não é.
	WrapsRound bool
	// NewRound é o número da rodada que começa nesta posição, na faixa sem fim
	// — que dá a volta mais de uma vez e precisa dizer QUAL rodada começa. Zero
	// onde nenhuma começa.
	NewRound int
	// SaysYou troca o nome pela palavra, e ela é uma decisão da faixa INTEIRA e
	// não desta linha: a palavra só desambigua enquanto for UMA. Com dois
	// personagens meus na faixa, "você › Fulano › ⟲ você" deixa de responder
	// quanto falta para mim.
	SaysYou bool
}

// turnStripOf traduz a janela circular para a tela.
//
// A REGRA continua no `live`: quem escolhe os três e dá a volta é o
// `UpcomingTurns`. O que se decide aqui são dois fatos de APRESENTAÇÃO: qual
// dos três é de quem está olhando, e onde a rodada vira.
//
// A VOLTA é recalculada do índice em vez de vir do `UpcomingTurns`, que devolve
// entradas e não posições — a regra é *quem* vem; isto é *onde desenhar o
// símbolo*.
//
// A fila que chega aqui é a que o `StateForRole` já redigiu: um segundo caminho
// até os nomes seria um segundo lugar por onde vazar o que a mesa não vê.
func turnStripOf(st *live.SessionRuntimeState, mine map[int64]bool) []turnAhead {
	window := live.UpcomingTurns(st.Initiative, st.TurnIndex, turnsAhead)
	strip := make([]turnAhead, 0, len(window))
	for step, entry := range window {
		strip = append(strip, turnAhead{
			Label:      entry.Label,
			Mine:       entry.CharacterID != nil && mine[*entry.CharacterID],
			Now:        step == 0,
			WrapsRound: st.TurnIndex+step == len(st.Initiative),
		})
	}
	// "VOCÊ" é decisão da faixa inteira, então ela é tomada com ela pronta: a
	// palavra só desambigua enquanto for UMA. Ver o `SaysYou`.
	if meusNaFaixa(strip) == 1 {
		for i := range strip {
			strip[i].SaysYou = strip[i].Mine
		}
	}
	return strip
}

func meusNaFaixa(strip []turnAhead) int {
	howMany := 0
	for _, p := range strip {
		if p.Mine {
			howMany++
		}
	}
	return howMany
}

// endlessStripOf é a faixa SEM FIM: a fila inteira a partir de quem joga
// agora, e mais `endlessTail` da rodada seguinte, com cada volta marcada pela
// rodada que começa. Ela anda uma casa a cada turno porque é refeita a cada
// tique do stream a partir da vez.
//
// "VOCÊ" vale em toda volta quando o jogador tem UM personagem na fila — é o
// mesmo, repetido; com dois, o nome volta, pela razão do `SaysYou`.
func endlessStripOf(st *live.SessionRuntimeState, mine map[int64]bool) []turnAhead {
	size := len(st.Initiative)
	if st.TurnIndex < 0 || size == 0 {
		return nil
	}
	strip := make([]turnAhead, 0, size+endlessTail)
	minePresent := map[int64]bool{}
	for step := 0; step < size+endlessTail; step++ {
		position := st.TurnIndex + step
		entry := st.Initiative[position%size]
		ahead := turnAhead{Label: entry.Label, Now: step == 0}
		if entry.CharacterID != nil && mine[*entry.CharacterID] {
			ahead.Mine = true
			minePresent[*entry.CharacterID] = true
		}
		if step > 0 && position%size == 0 {
			ahead.WrapsRound = true
			ahead.NewRound = st.Round + position/size
		}
		strip = append(strip, ahead)
	}
	if len(minePresent) == 1 {
		for i := range strip {
			strip[i].SaysYou = strip[i].Mine
		}
	}
	return strip
}

// endlessTail é quanto da rodada seguinte a faixa sem fim mostra depois da
// fila inteira: o bastante para a volta aparecer, mesmo no último da rodada.
const endlessTail = 3

// turnsAhead é o tamanho da faixa: a vez e as duas seguintes.
//
// TRÊS é escolhido e não medido — o Go não sabe quanto cabe na caixa. Três
// nomes cabem no cabeçalho a 390px sem truncar a ponto de não identificar
// ninguém. Quem quer a ordem inteira tem o trilho e a gaveta; crescer a faixa
// até competir com eles é como ela deixa de caber.
const turnsAhead = 3
