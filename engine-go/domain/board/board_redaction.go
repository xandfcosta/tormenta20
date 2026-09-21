package board

// O TABULEIRO COMO UM PAPEL O VÊ — a cortina, a peça escondida, o marcador da
// armadilha.
//
// Arquivo próprio porque isto é FRONTEIRA DE SEGURANÇA e não uma vista a mais:
// o que estas duas funções deixam passar chega à tela de quem não devia ver. No
// meio das mutações do estado elas liam como mais uma função sobre
// `BoardState`, e o nome do arquivo é o que diz a quem chega que mexer aqui é
// mexer no que vaza (ALE-361).

// BoardForRole é o tabuleiro como UM papel pode vê-lo. Papel desconhecido cai em
// jogador: errar para o lado que MOSTRA seria vazar por omissão, a mesma regra
// do `live.StateForRole`.
func BoardForRole(role string, b *BoardState) *BoardState {
	if b == nil || role == "gm" {
		return b
	}
	// A CORTINA vem ANTES da redação de peça: com ela fechada, a mesa
	// não recebe o mapa nenhum — nem as peças visíveis, nem o terreno, nem o
	// nome do lugar. Redigir peça por peça deixaria passar tudo o que não está
	// marcado como escondido, que é justamente a cena que o mestre está
	// montando.
	//
	// O que a mesa recebe é um tabuleiro VAZIO com a cortina ligada, e não
	// `nil`: `nil` significa "não há tabuleiro" e a cortina é outra coisa —
	// o jogador precisa saber que vem cena, sem ver qual (decisão do dono).
	if b.Curtained {
		// `Tokens` vai VAZIO e não nulo: fatia nil vira `null` no JSON, e o
		// cliente indexa `tokens.length` no cabeçalho — a cortina derrubaria a
		// tela da mesa em vez de escondê-la. O contrato do fio é "lista vazia é
		// uma lista", e quem o garante é aqui, não cada leitor.
		//
		// O `ID` ATRAVESSA a cortina, e é o único campo que atravessa: sem ele a
		// aba do jogador não teria como se chamar nem como ser clicada, e a
		// decisão do dono foi que ela APARECE mostrando a cortina — sumindo da
		// barra, a aba trocaria debaixo do dedo de quem estava olhando. O id é um
		// UUID e não conta nada sobre a cena. Quem esconde o NOME é esta linha, e
		// é o nome que diria "Cripta do Rei" para quem não devia saber.
		return &BoardState{ID: b.ID, Version: b.Version, Curtained: true, Tokens: []BoardToken{}}
	}
	return redactBoardForPlayers(b)
}

// redactBoardForPlayers apaga da cópia do jogador as peças que o mestre
// escondeu. Some a peça INTEIRA — e essa é a assimetria deliberada em relação ao
// `hpHidden` da iniciativa, onde a linha fica sem os números: aqui a existência
// da peça é a informação, e uma peça "presente porém anônima" entregaria a
// emboscada do mesmo jeito.
func redactBoardForPlayers(b *BoardState) *BoardState {
	out := *b
	out.Tokens = make([]BoardToken, 0, len(b.Tokens))
	hidden := map[string]bool{}
	for _, t := range b.Tokens {
		if t.Hidden {
			hidden[t.ID] = true
			continue
		}
		out.Tokens = append(out.Tokens, t)
	}
	// O marcador escondido some INTEIRO, como a peça: o mestre marca a armadilha
	// antes da mesa chegar nela, e um marcador "presente porém anônimo" diria à
	// mesa exatamente onde não pisar.
	out.Markers = make([]BoardMarker, 0, len(b.Markers))
	for _, marker := range b.Markers {
		if marker.Hidden {
			continue
		}
		out.Markers = append(out.Markers, marker)
	}
	// O provisório de uma peça escondida entregaria a emboscada por outro
	// caminho: um caminho desenhado saindo do nada é a peça sem o círculo.
	if out.Pending != nil && hidden[out.Pending.TokenID] {
		out.Pending = nil
	}
	return &out
}
