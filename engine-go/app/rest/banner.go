package rest

import (
	"context"
	"fmt"
	"slices"

	"t20engine/app"
	"t20engine/domain/book"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
)

// GiveBanner entrega o Estandarte (p54) do cavaleiro a UM aliado que o mestre diz que
// vê o estandarte: PM temporários iguais ao Carisma do cavaleiro (mínimo 1), até o fim
// da cena.
//
// O MESTRE DECIDE quem vê, e por isso é um gesto dele, aliado por aliado; o cavaleiro
// já recebeu sozinho no início da cena. A recusa é da regra: fora de cena, o poder do
// cavaleiro que não o tem, o aliado que não é do grupo, e a segunda entrega na mesma
// cena, que reescreveria a mesma reserva sem dar nada.
//
//	party.GiveBanner(ctx, gm, campaignID, sessionID, 47, 8) // o Porta-Estandarte dá ao bardo
func (p Party) GiveBanner(
	ctx context.Context, who app.Caller, campaignID, sessionID, knightID, allyID int64,
) (*live.SessionRuntimeState, error) {
	if _, err := p.access.GM(ctx, who, campaignID, sessionID); err != nil {
		return nil, err
	}
	state, err := p.sessions.State(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !state.InScene() {
		return nil, fmt.Errorf("o Estandarte vale a partir do início da cena, e não há cena aberta: %w", app.ErrRefused)
	}
	members, err := p.memberCharacterIDs(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if knightID == allyID || !slices.Contains(members, knightID) || !slices.Contains(members, allyID) {
		return nil, fmt.Errorf("o Estandarte vai do cavaleiro a um ALIADO do grupo (personagens %d e %d): %w",
			knightID, allyID, app.ErrRefused)
	}
	spec := book.ActivationOf(book.BannerPowerID, "")
	if spec == nil || spec.Grant == nil {
		return nil, fmt.Errorf("o Estandarte não está no catálogo")
	}
	knightRow, err := p.queries.GetCharacter(ctx, knightID)
	if err != nil {
		return nil, fmt.Errorf("ler o cavaleiro %d: %w", knightID, err)
	}
	knight, err := sheet.Load(ctx, p.queries, p.scopes.catalogs, knightRow)
	if err != nil {
		return nil, fmt.Errorf("montar a ficha de %s: %w", knightRow.Name, err)
	}
	if !sheet.OwnsActivation(knight, *spec) {
		return nil, fmt.Errorf("%s não tem o Estandarte: %w", knightRow.Name, app.ErrRefused)
	}
	allyRow, err := p.queries.GetCharacter(ctx, allyID)
	if err != nil {
		return nil, fmt.Errorf("ler o aliado %d: %w", allyID, err)
	}
	ally, err := sheet.Load(ctx, p.queries, p.scopes.catalogs, allyRow)
	if err != nil {
		return nil, fmt.Errorf("montar a ficha de %s: %w", allyRow.Name, err)
	}
	if sheet.ReceivedFrom(ally, book.BannerPowerID) {
		return nil, fmt.Errorf("%s já recebeu o Estandarte nesta cena: %w", allyRow.Name, app.ErrRefused)
	}
	howMuch, ok := p.plays.GrantAmount(ctx, knightRow, *spec.Grant)
	if !ok {
		return nil, fmt.Errorf("não deu para medir o Carisma de %s", knightRow.Name)
	}
	if _, err := p.plays.ApplyTempPool(ctx, allyID, "power", book.BannerPowerID, *spec.Grant, howMuch); err != nil {
		return nil, err
	}
	p.sheetTouched(allyID)
	return state, nil
}
