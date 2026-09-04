package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/tabuleiro"
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
// `tabuleiro.MoveOGrupo`.

func (s *Server) PartyRoutes(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/tabuleiro"
	r.Post(base+"/marcar-area/{x}/{y}/{x2}/{y2}", s.handleMarcarArea)
	r.Post(base+"/grupo/mover/{dx}/{dy}", s.gmContinuousCommand(movePartyTable))
}

// handleMarcarArea devolve os ids das peças dentro do laço.
//
// QUEM DECIDE quais peças existem é o `BoardForRole`, e é por isso que a conta é
// do SERVIDOR e não uma varredura do DOM: a peça escondida não aparece na tela
// do jogador, mas aparece na do mestre, e a redação por papel tem de continuar
// com um dono só.
func (s *Server) handleMarcarArea(w http.ResponseWriter, r *http.Request) {
	papel, sessionID, tabuleiroID, ok := s.whoMeasuresTheTable(w, r)
	if !ok {
		return
	}
	if papel != "gm" {
		http.Error(w, "só o mestre marca um grupo", http.StatusForbidden)
		return
	}
	de, err1 := quadradoDaURL(r)
	ate, err2 := urlSquareSecond(r)
	if err1 != nil || err2 != nil {
		http.Error(w, "os cantos do laço precisam ser dois pares de números", http.StatusBadRequest)
		return
	}
	b := s.boards.Get(r.Context(), sessionID, tabuleiroID)
	ids := tabuleiro.TokensInRectangle(b, de, ate)
	writeSignals(w, r, map[string]any{
		markedTokensSignal: strings.Join(ids, ","),
	})
}

// movePartyTable desloca as peças marcadas pelo delta do arrasto.
//
// A LISTA vem dos SINAIS e o DELTA vem do caminho, e a divisão é a mesma do
// resto: o caminho carrega o que o gesto ACABOU de decidir (quantos quadrados o
// dedo andou), e o sinal carrega o estado que já estava lá (quem foi marcado).
func movePartyTable(st *Server, c commandCtx) (*tabuleiro.BoardState, error) {
	dx, errX := intDoCaminho(chi.URLParam(c.R, "dx"))
	dy, errY := intDoCaminho(chi.URLParam(c.R, "dy"))
	if errX != nil || errY != nil {
		return nil, fmt.Errorf("o deslocamento (%q,%q) não é um par de números",
			chi.URLParam(c.R, "dx"), chi.URLParam(c.R, "dy"))
	}
	ids, err := markedTokens(c.R)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("não há peça marcada para mover")
	}
	return st.boards.MoveGroup(c.R.Context(), c.SessionID, c.TabuleiroID, ids, dx, dy)
}

// markedTokensSignal guarda os ids marcados, separados por vírgula.
//
// UMA string e não uma lista, ao contrário das paradas da régua, e a razão é o
// PROXY do Datastar: lista de sinal cria índice ao ser lida, e aqui a tela
// precisa perguntar "esta peça está marcada?" uma vez POR PEÇA. Com string, a
// pergunta é um `includes` sobre um valor só.
const markedTokensSignal = "pecasmarcadas"

// markedMax é o teto do grupo, e ele é o teto da MESA: 50 combatentes
// (`aovivo`). Uma lista maior que isso não saiu de um laço sobre este tabuleiro.
const markedMax = 50

// markedTokens lê os ids do sinal.
func markedTokens(r *http.Request) ([]string, error) {
	var sinais struct {
		Marcadas string `json:"pecasmarcadas"`
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
