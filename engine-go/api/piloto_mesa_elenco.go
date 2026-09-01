package api

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/go-chi/chi/v5"

	"t20engine/aovivo"
)

// O ELENCO DA CAMPANHA (ALE-269, superfície 6a) — o caminho.
//
// UM jogador por vez, que é o buraco que o "Adicionar grupo" não cobre: ele
// traz o grupo INTEIRO, e a cena em que só a Arwen desce na cripta não tinha
// gesto nenhum. Antes disto o mestre inventava um combatente com o nome dela à
// mão, e a linha nascia sem `characterId` — desligada da ficha, sem PV de
// verdade e fora do descanso.

func (s *Server) CastRoutes(r chi.Router) {
	r.Post("/mesa/{campaignId}/{sessionId}/elenco/{characterId}/na-fila",
		s.comandoDoMestre(poeOJogadorNaFila))
}

// poeOJogadorNaFila traz UM personagem do roster para o combate.
//
// Ele passa pelo MESMO `populateParty` do "Adicionar grupo", com uma lista de
// um: é ele que resolve o `characterId`, os vitais e a idempotência. Uma
// segunda forma de pôr PC na fila divergiria no dia em que a primeira mudasse —
// e a diferença apareceria como uma linha sem ficha, que é justamente o defeito
// que este gesto existe para não repetir.
func poeOJogadorNaFila(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	characterID, err := strconv.ParseInt(chi.URLParam(c.R, "characterId"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("personagem inválido: %q", chi.URLParam(c.R, "characterId"))
	}
	combatentes, err := st.listPlayerCombatants(c.R.Context(), c.CampaignID)
	if err != nil {
		return nil, errors.New("não deu para carregar o grupo desta campanha")
	}
	escolhido := aquele(combatentes, characterID)
	if escolhido == nil {
		// A TRAVA é esta, e ela é do servidor: o id vem do caminho, e o caminho
		// é digitável. Sem a conferência contra o roster, o mestre de uma mesa
		// poria na fila o personagem de OUTRA campanha.
		return nil, fmt.Errorf("o personagem %d não é jogador desta campanha", characterID)
	}
	estado, err := st.populateParty(c.SessionID, []combatant{*escolhido})
	if estado == nil {
		estado = st.sessions.GetState(c.SessionID)
	}
	return estado, err
}

// aquele acha o combatente do personagem pedido. Nil é a resposta para "não é
// jogador desta campanha", e quem chama decide o que fazer com isso.
func aquele(combatentes []combatant, characterID int64) *combatant {
	for i := range combatentes {
		if combatentes[i].characterID == characterID {
			return &combatentes[i]
		}
	}
	return nil
}

// ── as expressões que o Datastar executa ────────────────────────────────────

func abreAFichaDoElenco(m mesaMembro) string {
	return fmt.Sprintf("document.getElementById('elenco-%d').showModal()", m.CharacterID)
}

func fechaAFichaDoElenco(m mesaMembro) string {
	return fmt.Sprintf("document.getElementById('elenco-%d').close();", m.CharacterID)
}

// poeNaFila fecha ANTES de postar, e isso não é arrumação: a recusa do servidor
// acende no `$erroDoComando`, que é do RODAPÉ — um diálogo aberto por cima dela
// esconderia a única frase que explica o que houve.
func poeNaFila(v mesaView, m mesaMembro) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/elenco/%d/na-fila')",
		v.CampaignID, v.SessionID, m.CharacterID)
}
