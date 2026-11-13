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

// redactBoardForPlayers apaga da cópia do jogador o que o mestre escondeu.
//
// Some a coisa INTEIRA, e essa é a assimetria deliberada em relação ao
// `hpHidden` da iniciativa, onde a linha fica sem os números: aqui a EXISTÊNCIA
// é a informação. Uma peça "presente porém anônima" entregaria a emboscada do
// mesmo jeito.
//
// A ORDEM que sobra é a de DESENHO (quem vem depois fica por cima), e ela
// decide qual peça a tela oferece para mover: o filtro a preserva, e o
// `TestTheRedactionKeepsTheDrawingOrder` e o `TestTheRedactionKeepsTheMarkerOrder`
// a prendem.
func redactBoardForPlayers(b *BoardState) *BoardState {
	out := *b
	out.Tokens = make([]BoardToken, 0, len(b.Tokens))
	for _, token := range b.Tokens {
		if !token.Hidden {
			out.Tokens = append(out.Tokens, token)
		}
	}
	out.Markers = make([]BoardMarker, 0, len(b.Markers))
	for _, marker := range b.Markers {
		if !marker.Hidden {
			out.Markers = append(out.Markers, marker)
		}
	}

	// O PROVISÓRIO DE QUEM NÃO SOBROU morre junto: um caminho desenhado saindo
	// do nada é a peça sem o círculo, e entregaria a emboscada por outro
	// caminho.
	//
	// A pergunta é "sobrou?" e não "estava escondida?", e as duas não são a
	// mesma: a segunda deixaria passar um provisório apontando para peça que
	// nem existe. Hoje não há como chegar nesse estado — o `RemoveToken` limpa
	// o provisório junto —, e perguntar pelo sobrevivente é a forma que
	// continua certa se um dia houver.
	if out.Pending != nil && !hasTokenWithID(out.Tokens, out.Pending.TokenID) {
		out.Pending = nil
	}
	return &out
}

func hasTokenWithID(tokens []BoardToken, tokenID string) bool {
	for i := range tokens {
		if tokens[i].ID == tokenID {
			return true
		}
	}
	return false
}
