package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"t20engine/infra/db/sqlcgen"
)

// O QUE O RASTREADOR PRECISA SABER SOBRE UM PERSONAGEM — a MESA lendo vitais e
// bônus para a fila de iniciativa. Quem admite um herói na campanha é o
// `campaign_members.go`, que não tem a mesma razão para mudar.

// initiativeBonus é o total da perícia Iniciativa do personagem (½ nível +
// atributo + treino + itens), lido da ficha COMPUTADA pelo motor.
//
// A soma da rolagem é trivial, mas o BÔNUS é regra do livro: computá-lo na tela
// seria uma segunda implementação livre para divergir do motor. Aqui é a mesma
// `ComputeSheetV2` que a ficha inteira usa.
//
// @example bonus, err := s.initiativeBonus(ctx, 7) // 8, para o Arcanista Nv9
func (tr tableRules) initiativeBonus(ctx context.Context, characterID int64) (int64, error) {
	if tr.catalogs == nil {
		return 0, errors.New("Rules catalog not loaded")
	}
	row, err := tr.queries.GetCharacter(ctx, characterID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("Character %d not found", characterID)
	}
	if err != nil {
		return 0, errors.New("Could not load character")
	}
	sheet, err := tr.sheet.ComputeSheet(ctx, row)
	if err != nil {
		return 0, errors.New("Could not compute sheet")
	}
	for _, ex := range sheet.Expertises {
		if ex.Name == initiativeExpertise {
			return int64(ex.Total), nil
		}
	}
	// Ficha sem a perícia na lista é ficha sem classe (o motor não computa
	// perícia nenhuma). Zero é a resposta honesta: o d20 sozinho vale, e recusar
	// deixaria o jogador sem conseguir entrar na fila por causa de uma ficha
	// incompleta — o que o mestre resolve na hora arrastando a ordem.
	return 0, nil
}

// initiativeExpertise é o nome da perícia no catálogo. Escrito UMA vez porque a
// string literal em dois lugares é como um typo sobrevive: o `for` acima não
// acharia nada e devolveria zero em silêncio.
const initiativeExpertise = "Iniciativa"

// combatant é o retrato de um personagem que interessa ao rastreador — nome mais
// vitais vivos — para uma linha de iniciativa.
type combatant struct {
	characterID int64
	name        string
	hpCurrent   int64
	hpMax       int64
	mpCurrent   int64
	mpMax       int64
}

// resolveCombatant resolve os números de um personagem para uma linha de
// iniciativa, cobrando as regras da campanha: o personagem tem de ser membro
// dela, e quem pede tem de ser o dono do personagem ou o mestre. A ORDEM da
// checagem é personagem → campanha → filiação → autorização.
func (tr tableRules) resolveCombatant(ctx context.Context, callerID, campaignID, characterID int64) (combatant, int, error) {
	ch, err := tr.queries.GetCharacter(ctx, characterID)
	if errors.Is(err, sql.ErrNoRows) {
		return combatant{}, http.StatusNotFound, fmt.Errorf("Character %d not found", characterID)
	}
	if err != nil {
		return combatant{}, http.StatusInternalServerError, errors.New("Could not load character")
	}
	camp, err := tr.queries.GetCampaign(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return combatant{}, http.StatusNotFound, fmt.Errorf("Campaign %d not found", campaignID)
	}
	if err != nil {
		return combatant{}, http.StatusInternalServerError, errors.New("Could not load campaign")
	}
	isMember, err := tr.queries.IsCharacterMember(ctx, sqlcgen.IsCharacterMemberParams{Campaignid: campaignID, Characterid: characterID})
	if err != nil {
		return combatant{}, http.StatusInternalServerError, errors.New("Could not check membership")
	}
	if !isMember {
		return combatant{}, http.StatusBadRequest, fmt.Errorf("Character %d is not a member of campaign %d", characterID, campaignID)
	}
	if callerID != ch.Ownerid && callerID != camp.Ownerid {
		return combatant{}, http.StatusForbidden, fmt.Errorf(
			"Caller %d is neither the GM of campaign %d nor the owner of character %d", callerID, campaignID, characterID)
	}
	return combatant{
		characterID: characterID, name: ch.Name,
		hpCurrent: ch.Hpcurrent, hpMax: ch.Hpmax, mpCurrent: ch.Mpcurrent, mpMax: ch.Mpmax,
	}, http.StatusOK, nil
}

// listPlayerCombatants devolve todo personagem de jogador da campanha com os
// vitais vivos — é o "pôr o grupo na fila" de um clique do mestre.
func (tr tableRules) listPlayerCombatants(ctx context.Context, campaignID int64) ([]combatant, error) {
	rows, err := tr.queries.ListMembers(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	out := []combatant{}
	// SEM filtro de papel: o mestre não tem personagem próprio, e os NPCs dele não
	// são membros da campanha. Ver a nota mais longa no `tableRoster`.
	for _, m := range rows {
		out = append(out, combatant{
			characterID: m.Characterid, name: m.Charname,
			hpCurrent: m.Charhpcurrent, hpMax: m.Charhpmax, mpCurrent: m.Charmpcurrent, mpMax: m.Charmpmax,
		})
	}
	return out, nil
}

// listMemberCharacterIds devolve o id do personagem de cada membro — o conjunto
// que um descanso de sessão inteira percorre.
func (tr tableRules) listMemberCharacterIds(ctx context.Context, campaignID int64) ([]int64, error) {
	rows, err := tr.queries.ListMembers(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, m := range rows {
		ids = append(ids, m.Characterid)
	}
	return ids, nil
}
