package table

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-chi/chi/v5"

	"t20engine/app"
	"t20engine/app/character"
	"t20engine/app/combat"
	"t20engine/domain/book"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/serve/web/routes"
)

// O ROLAR DE NOVO NA FAIXA DO ATAQUE (p226, ALE-508): só quem rolou, enquanto
// o provisório espera o mestre, e vale o segundo dado. O d20 do golpe é rolado
// no servidor, e o de novo também — a faixa não tem o campo "Eu rolei".

// rerollsAttack rola de novo o ataque provisório com o gasto do caminho.
func rerollsAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	ctx := c.R.Context()
	state, err := st.deps.Sessions().State(ctx, c.SessionID)
	if err != nil {
		return nil, err
	}
	if state == nil || state.PendingAttack == nil {
		return nil, errors.New("não há ataque na mesa para rolar de novo")
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
	var object *combat.ObjectTarget
	if pa.TargetTokenID != "" {
		if object, err = st.objectTokenOf(c, pa.TargetTokenID); err != nil {
			return nil, err
		}
	}
	_, mine, _ := st.tableRoster(ctx, c.User, c.CampaignID)
	req := combat.RerollRequest(c.CampaignID, c.SessionID, pa, mine[*attacker.CharacterID], object)
	d20, err := rerollDie(c)
	if err != nil {
		return nil, err
	}
	err = st.plays.PayAttackReroll(ctx, row, meleeAttack(pa), pa.Replay.WeaponID, chi.URLParam(c.R, "gasto"),
		func(offer book.SkillSpendOffer) error {
			_, err := st.strike.Reroll(ctx, app.Caller{ID: c.User}, c.Role, req, d20,
				live.DieReroll{Name: offer.Name, PM: offer.PM}, offer.NaturalFrom)
			return err
		})
	if err != nil {
		return nil, err
	}
	return st.deps.Sessions().State(ctx, c.SessionID)
}

// meleeAttack diz se o teste foi de Luta: a manobra é sempre corpo a corpo
// (p234), e o provisório dela não leva o `Melee`.
func meleeAttack(pa live.PendingAttack) bool {
	return pa.Melee || pa.Maneuver != nil
}

// attackProposalFor é a faixa do ataque com os botões de rolar de novo de QUEM
// OLHA, quando é ele quem rolou.
func (s Scene) attackProposalFor(
	ctx context.Context, st *live.SessionRuntimeState, userID, campaignID, sessionID int64,
) *attackProposal {
	band := attackProposalOf(st, userID)
	if band == nil || !band.Mine {
		return band
	}
	attacker := entryByID(st, st.PendingAttack.AttackerEntryID)
	if attacker == nil || attacker.CharacterID == nil {
		return band
	}
	row, err := s.deps.Queries().GetCharacter(ctx, *attacker.CharacterID)
	if err != nil {
		return band
	}
	dto, err := sheet.Load(ctx, s.deps.Queries(), s.deps.Catalogs(), row)
	if err != nil {
		return band
	}
	pa := st.PendingAttack
	for _, offer := range character.AttackRerollOffersFor(dto, meleeAttack(*pa), pa.Replay.WeaponID) {
		band.Offers = append(band.Offers, rerollOffer{
			Label: fmt.Sprintf("Rolar de novo · %s · %d PM", offer.Name, offer.PM),
			Path:  fmt.Sprintf("%s/ataque/rolar-de-novo/%s", routes.Session(campaignID, sessionID), offer.Key),
		})
	}
	return band
}
