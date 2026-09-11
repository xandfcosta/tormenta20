package table

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/board"
)

// A SELEÇÃO EM ÁREA de peças (ALE-203, item 10 do dono).
//
// "Não temos ferramenta de seleção em área." Com a ferramenta de MOVER na mão, o
// arrasto no vazio — que hoje não faz nada — marca as peças dentro do retângulo,
// e os verbos passam a valer para todas.
//
// # Marcar NÃO muta, mover MUTA
//
// São dois caminhos com naturezas diferentes, e por isso duas rotas:
//
//   - `marcar-area` responde SÓ SINAIS, como a régua: marcar não muda a cena de
//     ninguém, e uma marcação que remendasse o mapa trocaria a peça debaixo do
//     dedo de quem está arrastando.
//   - `grupo/mover` grava e publica, como qualquer movimento.
//
// # É gesto DE MESTRE
//
// Um jogador tem uma peça, e o que o grupo dispensa — a regra de deslocamento —
// é exatamente o que protege o turno dele. A razão inteira está no
// `board.MoveOGrupo`.

func (s Scene) PartyRoutes(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/tabuleiro"
	r.Post(base+"/marcar-area", s.handleMarcarArea)
	r.Post(base+"/grupo/mover", s.gmContinuousCommand(movePartyTable))
}

// handleMarcarArea devolve os ids das peças dentro do laço.
//
// QUEM DECIDE quais peças existem é o `BoardForRole`, e é por isso que a conta é
// do SERVIDOR e não uma varredura do DOM: a peça escondida não aparece na tela
// do jogador, mas aparece na do mestre, e a redação por papel tem de continuar
// com um dono só.
func (s Scene) handleMarcarArea(w http.ResponseWriter, r *http.Request) {
	papel, sessionID, tabuleiroID, ok := s.whoMeasuresTheTable(w, r)
	if !ok {
		return
	}
	if papel != "gm" {
		http.Error(w, "só o mestre marca um grupo", http.StatusForbidden)
		return
	}
	_, de, ate, err := pointsFromBody(r)
	if err != nil {
		http.Error(w, "os cantos do laço precisam ser dois pares de números", http.StatusBadRequest)
		return
	}
	b := s.deps.Boards().Get(r.Context(), sessionID, tabuleiroID)
	ids := board.TokensInRectangle(b, de, ate)
	writeSignals(w, r, map[string]any{
		markedTokensSignal: strings.Join(ids, ","),
	})
}

// movePartyTable desloca as peças marcadas pelo delta do arrasto.
//
// A LISTA E O DELTA vêm do mesmo CORPO (ALE-307). Aqui morava a divisão
// contrária — o delta no caminho, a lista nos sinais — com o argumento de que o
// caminho carrega o que o gesto acabou de decidir e o sinal, o estado que já
// estava lá. O argumento não se sustentava: `payload` carrega os dois, e o que a
// divisão custava era o endereço montado por concatenação na expressão.
func movePartyTable(st Scene, c commandCtx) (*board.BoardState, error) {
	corpo, err := partyDrag(c.R)
	if err != nil {
		return nil, err
	}
	if len(corpo.ids) == 0 {
		return nil, fmt.Errorf("não há peça marcada para mover")
	}
	return st.deps.Boards().MoveGroup(
		c.R.Context(), c.SessionID, c.TabuleiroID, corpo.ids, corpo.Delta.X, corpo.Delta.Y)
}

// partyDragBody é o arrasto do grupo: o DELTA e a lista, num corpo só.
//
// Os dois viajam juntos porque o corpo só pode ser lido UMA vez — o
// `ReadSignals` copia o `r.Body` inteiro num buffer, e a segunda chamada recebe
// vazio sem erro nenhum. Até a ALE-307 o delta vinha do caminho e a lista do
// corpo, e a divisão não era desenho: era o que sobrava de escrever o endereço
// com o delta concatenado dentro de uma expressão do Datastar.
//
// O `payload` do `@post` SUBSTITUI os sinais, então `marked_tokens` está
// listado ao lado do delta na expressão que posta — a mesma forma do colar.
type partyDragBody struct {
	Delta    struct{ X, Y int } `json:"delta"`
	Marcadas string             `json:"marked_tokens"`

	ids []string
}

// partyDrag lê o corpo do arrasto e reparte a lista de marcadas.
func partyDrag(r *http.Request) (partyDragBody, error) {
	var corpo partyDragBody
	if err := datastar.ReadSignals(r, &corpo); err != nil {
		return corpo, fmt.Errorf("as peças marcadas não vieram: %w", err)
	}
	for _, id := range strings.Split(corpo.Marcadas, ",") {
		if id != "" {
			corpo.ids = append(corpo.ids, id)
		}
	}
	if len(corpo.ids) > markedMax {
		return corpo, fmt.Errorf("o grupo tem %d peças e a mesa cabe %d", len(corpo.ids), markedMax)
	}
	return corpo, nil
}

// markedTokensSignal guarda os ids marcados, separados por vírgula.
//
// UMA string e não uma lista, ao contrário das paradas da régua, e a razão é o
// PROXY do Datastar: lista de sinal cria índice ao ser lida, e aqui a tela
// precisa perguntar "esta peça está marcada?" uma vez POR PEÇA. Com string, a
// pergunta é um `includes` sobre um valor só.
const markedTokensSignal = "marked_tokens"

// markedMax é o teto do grupo, e ele é o teto da MESA: 50 combatentes
// (`live`). Uma lista maior que isso não saiu de um laço sobre este tabuleiro.
const markedMax = 50

// markedTokens lê os ids do sinal.
func markedTokens(r *http.Request) ([]string, error) {
	var sinais struct {
		Marcadas string `json:"marked_tokens"`
	}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return nil, fmt.Errorf("as peças marcadas não vieram: %w", err)
	}
	var ids []string
	for _, id := range strings.Split(sinais.Marcadas, ",") {
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) > markedMax {
		return nil, fmt.Errorf("o grupo tem %d peças e a mesa cabe %d", len(ids), markedMax)
	}
	return ids, nil
}
