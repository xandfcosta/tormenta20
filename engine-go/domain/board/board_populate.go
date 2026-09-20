package board

import "t20engine/domain/live"

// QUEM ENTRA no tabuleiro quando o combate começa, e em que quadrado cada um
// nasce.

// PopulateBoard traz para o tabuleiro cada linha ESCOLHIDA da iniciativa que
// ainda não tem peça, com os PERSONAGENS de um lado e o resto do outro.
// Idempotente de propósito, como o `populateParty` do rastreador: clicar duas
// vezes não duplica ninguém.
//
// Em dois lados e não numa fileira única no meio do mapa: este é o estado em
// que o mestre encontra o tabuleiro no segundo em que o combate começa, com a
// mesa esperando, e nascendo em dois lados ele AJUSTA em vez de DISTRIBUIR — a
// informação de lado já existia na linha.
//
// Quem não foi escolhido não nasce — nem escondido: o assassino que o mestre
// montou para aparecer no terceiro turno não deve estar no mapa, e peça que não
// existe não vaza por bug de redação.
func PopulateBoard(b *BoardState, st *live.SessionRuntimeState, newID func() string, chosen EntrySelection) int {
	placed := 0
	for _, entry := range st.Initiative {
		if !chosen.wants(entry.ID) || hasTokenForEntry(b, entry.ID) {
			continue
		}
		token := BoardToken{
			Label: entry.Label, Kind: entry.Type, Footprint: 1,
			EntryID: strPtr(entry.ID), CharacterID: entry.CharacterID,
		}
		spot := clusterSpot(b, entry.Type == "character")
		token.X, token.Y = spot.x, spot.y
		if err := AddToken(b, token, newID); err != nil {
			break
		}
		placed++
	}
	return placed
}

// Os dois lados onde uma cena de combate começa, em quadrados.
//
// A distância entre as bordas é de 6 quadrados — 9m, o alcance curto do livro
// (T20 p224). É perto o bastante para a briga começar sem ninguém andar meia
// tela, e longe o bastante para o primeiro turno ainda ter escolha: aproximar,
// atirar ou conjurar.
const (
	partySideX  = -5 // personagens: colunas -5, -4, -3
	enemySideX  = 3  // o resto: colunas 3, 4, 5
	clusterCols = 3
)

// TopChromeRows são as fileiras do plano que o CROMO do topo cobre quando a
// cena abre, e ninguém nasce nelas.
//
// A janela nasce em (0,0) — o quadrado (0,0) do plano fica na quina de cima da
// tela, ver `table.viewportSignals` — e o painel de verbos flutua a 0.5rem do
// topo dela com 50px de altura: 8px de recuo mais 44px do botão mais alto
// (`min-h-11`) e a borda. Ele termina em y=58px, que no zoom padrão de 44px
// (`table.DefaultSquare`) são as fileiras 0 e 1. Peça nascida ali fica sob o
// painel, e todo gesto nela — menu da peça, pintar terreno, largar marcador,
// arrastar — vira gesto do painel.
//
// O número é PIXEL DO NAVEGADOR escrito no servidor, e ele não pode ser
// importado de `web/table` porque aquele pacote importa este. Quem o mantém
// honesto é o e2e `o submenu de duplicar só entra no caminho do teclado quando
// é aberto`, que abre o menu da peça a 390px de largura: painel mais alto ou
// zoom padrão menor põem a peça de volta debaixo dele e o caso fica vermelho.
const TopChromeRows = 2

// clusterSpot devolve o primeiro quadrado livre do lado pedido, preenchendo em
// blocos de três colunas que crescem para baixo — a partir da primeira fileira
// que o cromo do topo não cobre.
//
// Continua havendo um lugar COMBINADO onde a peça nova aparece, que é o que um
// plano infinito exige — só que agora são dois, um por lado. E continua
// respeitando quem já está no tabuleiro: o mestre pode ter posicionado alguém
// ali antes de trazer o resto.
//
// O recuo é das DUAS colunas de uma vez, e é por isso que ele não mexe na
// distância entre os lados: os seis quadrados do alcance curto (p224) são
// horizontais.
func clusterSpot(b *BoardState, isParty bool) boardSpot {
	baseX := enemySideX
	if isParty {
		baseX = partySideX
	}
	for i := 0; ; i++ {
		spot := boardSpot{x: baseX + i%clusterCols, y: TopChromeRows + i/clusterCols}
		if !occupied(b, spot.x, spot.y) {
			return spot
		}
	}
}

func hasTokenForEntry(b *BoardState, entryID string) bool {
	for _, t := range b.Tokens {
		if t.EntryID != nil && *t.EntryID == entryID {
			return true
		}
	}
	return false
}
