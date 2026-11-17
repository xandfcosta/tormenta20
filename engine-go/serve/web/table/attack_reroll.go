package table

import (
	"errors"
	"fmt"

	"github.com/go-chi/chi/v5"

	"t20engine/app"
	"t20engine/app/character"
	"t20engine/domain/live"
	"t20engine/serve/web/routes"
)

// O ROLAR DE NOVO NA FAIXA DO ATAQUE (p226, ALE-508): só quem rolou, enquanto
// o provisório espera o mestre, e vale o segundo dado. O d20 do golpe é rolado
// no servidor, e o de novo também — a faixa não tem o campo "Eu rolei", e por
// isso este gesto não lê o `reroll_d20`: o sinal é do teste de perícia, e fica na
// página depois que o campo dele some.

// rerollsAttack rola de novo o ataque provisório com o gasto do caminho.
func rerollsAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	ctx := c.R.Context()
	state, err := st.deps.Sessions().State(ctx, c.SessionID)
	if err != nil {
		return nil, err
	}
	if state == nil || state.PendingAttack == nil {
		return nil, errors.New("não há ataque na mesa")
	}
	pa := *state.PendingAttack
	attacker := entryByID(state, pa.AttackerEntryID)
	if attacker == nil || attacker.CharacterID == nil {
		return nil, errors.New("quem atacou não tem ficha de onde pagar o rolar de novo")
	}
	row, err := st.deps.Queries().GetCharacter(ctx, *attacker.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("a ficha de %s não existe mais", attacker.Label)
	}
	offer, err := st.plays.AttackRerollOffer(ctx, row, meleeAttack(pa), pa.WeaponID, chi.URLParam(c.R, "gasto"))
	// O GASTO grava na ficha (a sorte, o PM), e a ficha aberta só se redesenha no evento.
	st.deps.CharacterChanged(row.ID)
	if err != nil {
		return nil, err
	}
	if _, err := st.strike.Reroll(ctx, app.Caller{ID: c.User}, c.SessionID, nil,
		live.DieReroll{Name: offer.Name, PM: offer.PM}, offer.NaturalFrom); err != nil {
		return nil, err
	}
	return st.deps.Sessions().State(ctx, c.SessionID)
}

// meleeAttack diz se o teste foi de Luta: a manobra é sempre corpo a corpo
// (p234), e o provisório dela não leva o `Melee`.
func meleeAttack(pa live.PendingAttack) bool {
	return pa.Melee || pa.Maneuver != nil
}

// attackProposalFor é a faixa do ataque de QUEM OLHA: inteira para o mestre, e
// recortada para o jogador (ver `forPlayer`), com os botões de rolar de novo
// quando é ele quem rolou.
func (s Scene) attackProposalFor(
	st *live.SessionRuntimeState, userID int64, role string, campaignID, sessionID int64, sheets *tableSheets,
) *attackProposal {
	band := attackProposalOf(st, userID)
	if band != nil && role != "gm" {
		band = band.forPlayer(*st.PendingAttack)
	}
	if band == nil || !band.Mine {
		return band
	}
	attacker := entryByID(st, st.PendingAttack.AttackerEntryID)
	if attacker == nil || attacker.CharacterID == nil {
		return band
	}
	dto, ok := sheets.of(*attacker.CharacterID)
	if !ok {
		return band
	}
	pa := st.PendingAttack
	band.Offers = rerollOffersAt(character.AttackRerollOffersFor(dto, meleeAttack(*pa), pa.WeaponID),
		routes.Session(campaignID, sessionID)+"/ataque/rolar-de-novo/")
	return band
}
