package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"
	"t20engine/sheet"
)

var campaignMemberRoles = map[string]bool{"player": true, "gm": true}

type memberDTO struct {
	ID          int64               `json:"id"`
	CampaignID  int64               `json:"campaignId"`
	CharacterID int64               `json:"characterId"`
	Role        string              `json:"role"`
	AddedAt     string              `json:"addedAt"`
	Character   *memberCharacterDTO `json:"character,omitempty"`
}

type memberCharacterDTO struct {
	ID        int64            `json:"id"`
	OwnerID   int64            `json:"ownerId"`
	Name      string           `json:"name"`
	Level     int64            `json:"level"`
	HpCurrent int64            `json:"hpCurrent"`
	HpMax     int64            `json:"hpMax"`
	MpCurrent int64            `json:"mpCurrent"`
	MpMax     int64            `json:"mpMax"`
	Classes   []sheet.ClassDTO `json:"classes"`
}

func memberScalars(m sqlcgen.CampaignMember) memberDTO {
	return memberDTO{ID: m.ID, CampaignID: m.Campaignid, CharacterID: m.Characterid, Role: m.Role, AddedAt: m.Addedat}
}

// initiativeBonus é o total da perícia Iniciativa do personagem (½ nível +
// atributo + treino + itens), lido da ficha COMPUTADA pelo motor (ALE-213).
//
// Existe porque o total da rolagem passou a ser somado no servidor: a soma é
// trivial, mas o BÔNUS é regra do livro, e deixá-lo na tela seria uma segunda
// implementação livre para divergir do motor — o que a ALE-104 apagou. Aqui não
// há segunda conta: é a mesma `ComputeSheetV2` que a ficha inteira usa.
//
// Vizinha do `resolveCombatant` porque é o mesmo assunto — o que o rastreador
// precisa saber sobre um personagem — e transport-agnostic pela mesma razão.
//
// @example bonus, err := s.initiativeBonus(ctx, 7) // 8, para o Arcanista Nv9
func (s *Server) initiativeBonus(ctx context.Context, characterID int64) (int64, error) {
	if s.catalogs == nil {
		return 0, errors.New("Rules catalog not loaded")
	}
	row, err := s.queries.GetCharacter(ctx, characterID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("Character %d not found", characterID)
	}
	if err != nil {
		return 0, errors.New("Could not load character")
	}
	sheet, err := s.ComputeSheet(ctx, row)
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

// combatant is a character's tracker-relevant snapshot (name + live vitals) for an
// initiative entry. Transport-agnostic — the WS gateway maps it into an InitiativeEntry.
type combatant struct {
	characterID int64
	name        string
	hpCurrent   int64
	hpMax       int64
	mpCurrent   int64
	mpMax       int64
}

// resolveCombatant resolves a character's tracker stats for an initiative entry, enforcing
// the campaign rules: the character must be a member of the campaign, and the caller must
// be either the character's owner or the campaign GM (owner). Transport-agnostic (the WS
// gateway maps status→WsException). — same
// check order (character → campaign → membership → authorization).
func (s *Server) resolveCombatant(ctx context.Context, callerID, campaignID, characterID int64) (combatant, int, error) {
	ch, err := s.queries.GetCharacter(ctx, characterID)
	if errors.Is(err, sql.ErrNoRows) {
		return combatant{}, http.StatusNotFound, fmt.Errorf("Character %d not found", characterID)
	}
	if err != nil {
		return combatant{}, http.StatusInternalServerError, errors.New("Could not load character")
	}
	camp, err := s.queries.GetCampaign(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return combatant{}, http.StatusNotFound, fmt.Errorf("Campaign %d not found", campaignID)
	}
	if err != nil {
		return combatant{}, http.StatusInternalServerError, errors.New("Could not load campaign")
	}
	isMember, err := s.queries.IsCharacterMember(ctx, sqlcgen.IsCharacterMemberParams{Campaignid: campaignID, Characterid: characterID})
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

// listPlayerCombatants returns every player character in the campaign with live vitals —
// the GM's one-shot "populate tracker".
func (s *Server) listPlayerCombatants(ctx context.Context, campaignID int64) ([]combatant, error) {
	rows, err := s.queries.ListMembers(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	out := []combatant{}
	for _, m := range rows {
		if m.Role != "player" {
			continue
		}
		out = append(out, combatant{
			characterID: m.Characterid, name: m.Charname,
			hpCurrent: m.Charhpcurrent, hpMax: m.Charhpmax, mpCurrent: m.Charmpcurrent, mpMax: m.Charmpmax,
		})
	}
	return out, nil
}

// listMemberCharacterIds returns the character id of every member (any role) — the set a
// session-wide rest iterates over.
func (s *Server) listMemberCharacterIds(ctx context.Context, campaignID int64) ([]int64, error) {
	rows, err := s.queries.ListMembers(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, m := range rows {
		ids = append(ids, m.Characterid)
	}
	return ids, nil
}

// joinCampaign clona o personagem para a mesa e cria o membro NA MESMA
// transação (ALE-156).
//
// Eram duas antes, e a que falhava era a segunda: um `CreateMember` com erro
// deixava a cópia órfã no banco — e cópia órfã é pior que nada, porque o
// `campaignHasCopyOf` passa a responder "já está na mesa" e o herói fica
// impedido de entrar para sempre, sem membro nenhum que se possa remover para
// desfazer.
//
// A cópia existe porque a ficha da mesa é um instantâneo (ALE-33): editar
// durante a sessão não pode vazar para as outras campanhas.
func (rules campaignRules) joinCampaign(ctx context.Context, sourceID, campaignID, ownerID int64, role string) (sqlcgen.CampaignMember, error) {
	tx, err := rules.db.BeginTx(ctx, nil)
	if err != nil {
		return sqlcgen.CampaignMember{}, err
	}
	defer func() { _ = tx.Rollback() }()

	// A checagem é REFEITA aqui dentro, e é isto que fecha a corrida.
	//
	// A de fora existe para a mensagem amigável e para o caminho rápido; ela
	// roda sem transação, então dois pedidos simultâneos passam os dois por
	// ela. Com o `_txlock=immediate` (ALE-156), a transação já nasce com a
	// trava de escrita, então o segundo pedido ESPERA o primeiro terminar — e
	// esta releitura enxerga o que ele gravou.
	//
	// É a mesma forma do commit de movimento no tabuleiro, que reconfere a vez:
	// entre decidir e escrever, a mesa pode ter mudado.
	if err := assertCanJoin(ctx, rules.queries.WithTx(tx), tx, sourceID, campaignID, ownerID, role); err != nil {
		return sqlcgen.CampaignMember{}, err
	}

	copyID, err := cloneCharacterTx(ctx, tx, sourceID, campaignID)
	if err != nil {
		return sqlcgen.CampaignMember{}, err
	}
	member, err := rules.queries.WithTx(tx).CreateMember(ctx, sqlcgen.CreateMemberParams{
		Campaignid: campaignID, Characterid: copyID, Role: role, Addedat: plataforma.NowISO(),
	})
	if err != nil {
		return sqlcgen.CampaignMember{}, err
	}
	return member, tx.Commit()
}

// errAlreadyInCampaign: a releitura dentro da transação achou o que a checagem
// de fora não tinha achado — alguém ganhou a corrida.
var errAlreadyInCampaign = errors.New("personagem já está na campanha")

// assertCanJoin repete, DENTRO da transação, as duas travas que o handler já
// tentou por fora. Repetição de propósito: a de fora é pela mensagem, esta é
// pela verdade (ALE-156).
func assertCanJoin(ctx context.Context, q *sqlcgen.Queries, tx *sql.Tx, sourceID, campaignID, ownerID int64, role string) error {
	if role == "player" {
		hasPc, err := q.HasPlayerPc(ctx, sqlcgen.HasPlayerPcParams{Campaignid: campaignID, Ownerid: ownerID})
		if err != nil {
			return err
		}
		if hasPc {
			return errAlreadyInCampaign
		}
	}
	var hasCopy bool
	err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM characters WHERE sourceCharacterId = ? AND campaignId = ?)`,
		sourceID, campaignID).Scan(&hasCopy)
	if err != nil {
		return err
	}
	if hasCopy {
		return errAlreadyInCampaign
	}
	return nil
}
