package table

import (
	"fmt"

	"github.com/go-chi/chi/v5"

	"t20engine/app"
	"t20engine/app/combat"
	"t20engine/domain/live"
	"t20engine/serve/web/routes"
)

// O GESTO DE ATACAR, e a divisa dele é a do movimento: propor é de quem joga,
// confirmar é de quem mestra.
//
// O ALVO vem no caminho e QUEM ATACA é a vez. É o que o gesto de menu permite
// dizer — o menu abre sobre o alvo, e não há segundo clique onde caberia
// escolher o atacante —, e é também o que o livro diz: combate acontece em
// turnos, e quem age é quem está na vez (p231).
func (s Scene) AttackRoutes(r chi.Router) {
	r.Post(sessionPattern+"/iniciativa/{entryId}/atacar", s.tableStateCommand(proposesAttack))
	r.Post(sessionPattern+"/ataque/confirmar", s.gmCommand(confirmsAttack))
	r.Post(sessionPattern+"/ataque/cancelar", s.tableStateCommand(cancelsAttack))
}

// proposesAttack rola o ataque de quem está na vez contra a linha do caminho.
func proposesAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	state := st.deps.Sessions().GetState(c.SessionID)
	if state == nil || state.TurnIndex < 0 || state.TurnIndex >= len(state.Initiative) {
		return nil, fmt.Errorf("fora de combate não há vez, e é a vez que diz quem ataca")
	}
	onTurn := state.Initiative[state.TurnIndex]
	// A POSSE é resolvida CONTRA O BANCO e nunca contra o cliente — é o mesmo
	// caminho que o `Mover.OwnsCharacter` do tabuleiro usa.
	_, mine, _ := st.tableRoster(c.R.Context(), c.User, c.CampaignID)
	if _, err := st.strike.Propose(c.R.Context(), app.Caller{ID: c.User}, c.Role, combat.Request{
		SessionID:       c.SessionID,
		AttackerEntryID: onTurn.ID,
		TargetEntryID:   chi.URLParam(c.R, "entryId"),
		OwnsAttacker:    onTurn.CharacterID != nil && mine[*onTurn.CharacterID],
	}); err != nil {
		return nil, err
	}
	return st.deps.Sessions().GetState(c.SessionID), nil
}

func confirmsAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.deps.Sessions().CommitAttack(c.SessionID, live.Attacker{UserID: c.User, Role: c.Role})
}

func cancelsAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.deps.Sessions().CancelAttack(c.SessionID, live.Attacker{UserID: c.User, Role: c.Role})
}

// attackCommand escreve o `@post` dos dois verbos da faixa.
func attackCommand(v View, verb string) string {
	return fmt.Sprintf("@post('%s/ataque/%s')", routes.Session(v.CampaignID, v.SessionID), verb)
}

// attackOnTarget é o `@post` do menu da peça: o ALVO no caminho, e quem ataca é
// a vez.
func attackOnTarget(v BoardView, entryID string) string {
	return fmt.Sprintf("@post('%s/iniciativa/%s/atacar')",
		routes.Session(v.CampaignID, v.SessionID), entryID)
}
