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
	// A MANOBRA divide os verbos de confirmar e cancelar com o golpe, e só a
	// proposta é dela: uma manobra É um ataque corpo a corpo (p234), e o
	// provisório é o mesmo.
	r.Post(sessionPattern+"/iniciativa/{entryId}/manobra/{kind}", s.tableStateCommand(proposesManeuver))
	r.Post(sessionPattern+"/ataque/confirmar", s.gmCommand(confirmsAttack))
	r.Post(sessionPattern+"/ataque/cancelar", s.tableStateCommand(cancelsAttack))
}

// proposesAttack rola o ataque de quem está na vez contra a linha do caminho.
func proposesAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.proposeStrike(c, "")
}

// proposeStrike é o que o golpe e a manobra têm em comum, que é tudo menos a
// regra: a vez, a posse conferida contra o BANCO, e o provisório.
//
// A manobra vazia é o golpe. Um ramo aqui, e não dois caminhos: as sete
// conferências antes da rolagem são as mesmas, e duplicá-las faria a próxima
// correção acertar uma e esquecer a outra.
func (st Scene) proposeStrike(c commandCtx, maneuver string) (*live.SessionRuntimeState, error) {
	state, err := st.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	if state == nil || state.TurnIndex < 0 || state.TurnIndex >= len(state.Initiative) {
		return nil, fmt.Errorf("fora de combate não há vez, e é a vez que diz quem ataca")
	}
	onTurn := state.Initiative[state.TurnIndex]
	// A POSSE é resolvida CONTRA O BANCO e nunca contra o cliente — é o mesmo
	// caminho que o `Mover.OwnsCharacter` do tabuleiro usa.
	_, mine, _ := st.tableRoster(c.R.Context(), c.User, c.CampaignID)
	if _, err := st.strike.Propose(c.R.Context(), app.Caller{ID: c.User}, c.Role, combat.Request{
		CampaignID:      c.CampaignID,
		SessionID:       c.SessionID,
		AttackerEntryID: onTurn.ID,
		TargetEntryID:   chi.URLParam(c.R, "entryId"),
		OwnsAttacker:    onTurn.CharacterID != nil && mine[*onTurn.CharacterID],
		Maneuver:        maneuver,
	}); err != nil {
		return nil, err
	}
	return st.deps.Sessions().State(c.R.Context(), c.SessionID)
}

// proposesManeuver rola a manobra de quem está na vez contra a linha do caminho.
//
// Ele divide com o golpe a conferência inteira — a vez, a posse, a ação padrão —
// e muda só a REGRA no meio, que é o ramo do `Strike`. A MANOBRA vem no caminho
// e não no corpo porque o gesto é de menu: o mestre escolhe "Derrubar" num item,
// e um corpo JSON exigiria uma ilha de JS onde há um `@post`.
func proposesManeuver(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.proposeStrike(c, chi.URLParam(c.R, "kind"))
}

func confirmsAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.deps.Sessions().CommitAttack(c.R.Context(), c.SessionID, live.Attacker{UserID: c.User, Role: c.Role})
}

func cancelsAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.deps.Sessions().CancelAttack(c.R.Context(), c.SessionID, live.Attacker{UserID: c.User, Role: c.Role})
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

// maneuverOnTarget é o `@post` de uma manobra do menu da peça, com a manobra no
// CAMINHO — o gesto é de menu, e um corpo JSON pediria uma ilha de JS onde há um
// `@post`.
//
// Ele fecha as DUAS camadas, como o `copyCommand`: escolher a manobra fecha o
// submenu e o menu da peça, e o popover não se fecha sozinho quando o clique é
// num botão dentro dele.
func maneuverOnTarget(v BoardView, tokenID, entryID, kind string) string {
	return closesTheManeuverMenu(tokenID) +
		fmt.Sprintf("@post('%s/iniciativa/%s/manobra/%s')",
			routes.Session(v.CampaignID, v.SessionID), entryID, kind) +
		"; " + closeMenuToken
}
