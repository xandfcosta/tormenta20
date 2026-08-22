package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"t20engine/db/sqlcgen"
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
	ID        int64      `json:"id"`
	OwnerID   int64      `json:"ownerId"`
	Name      string     `json:"name"`
	Level     int64      `json:"level"`
	HpCurrent int64      `json:"hpCurrent"`
	HpMax     int64      `json:"hpMax"`
	MpCurrent int64      `json:"mpCurrent"`
	MpMax     int64      `json:"mpMax"`
	Classes   []ClassDTO `json:"classes"`
}

func memberScalars(m sqlcgen.CampaignMember) memberDTO {
	return memberDTO{ID: m.ID, CampaignID: m.Campaignid, CharacterID: m.Characterid, Role: m.Role, AddedAt: m.Addedat}
}

func (s *Server) classDTOs(r *http.Request, characterID int64) []ClassDTO {
	classes, _ := s.queries.ListClassesByCharacter(r.Context(), characterID)
	out := make([]ClassDTO, 0, len(classes))
	for _, cl := range classes {
		out = append(out, ClassDTO{ClassName: cl.Classname, Level: cl.Level})
	}
	return out
}

// campaignAccess enforces owner-or-member read access (resolveAccess), writing
// the 404/403 and returning false.
func (s *Server) campaignAccess(w http.ResponseWriter, r *http.Request, campaignID int64) bool {
	if _, status, err := s.resolveRole(r.Context(), currentUser(r), campaignID); err != nil {
		writeError(w, status, err.Error())
		return false
	}
	return true
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
	sheet, err := s.computeSheet(ctx, row)
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

// handleListMembers ports members.list: any player in the campaign sees the
// roster with live vitals.
func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	if !s.campaignAccess(w, r, cid) {
		return
	}
	rows, err := s.queries.ListMembers(r.Context(), cid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not list members")
		return
	}
	out := make([]memberDTO, 0, len(rows))
	for _, m := range rows {
		// ownerId lets a player match their own campaign character even though
		// it's a campaign snapshot excluded from their /characters roster (ALE-33).
		owner, _ := s.queries.GetCharacterOwner(r.Context(), m.Characterid)
		out = append(out, memberDTO{
			ID: m.ID, CampaignID: m.Campaignid, CharacterID: m.Characterid, Role: m.Role, AddedAt: m.Addedat,
			Character: &memberCharacterDTO{
				ID: m.Characterid, OwnerID: owner, Name: m.Charname, Level: m.Charlevel,
				HpCurrent: m.Charhpcurrent, HpMax: m.Charhpmax, MpCurrent: m.Charmpcurrent, MpMax: m.Charmpmax,
				Classes: s.classDTOs(r, m.Characterid),
			},
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAddMember ports members.add: caller must own the character; owner joins
// freely, others need a valid invite token; one player-PC per user per campaign.
func (s *Server) handleAddMember(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	var body struct {
		CharacterID *int64  `json:"characterId"`
		Role        *string `json:"role"`
		InviteToken *string `json:"inviteToken"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	user := currentUser(r)

	c, err := s.queries.GetCampaign(r.Context(), cid)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Campaign %d not found", cid))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load campaign")
		return
	}
	if c.Ownerid != user.ID {
		token := derefStr(body.InviteToken, "")
		if !c.Invitetoken.Valid || token == "" || token != c.Invitetoken.String {
			writeError(w, http.StatusForbidden, fmt.Sprintf("A valid invite token is required to join campaign %d", cid))
			return
		}
	}
	if body.CharacterID == nil {
		writeValidationError(w, FieldErrorMap{"characterId": {"characterId must be an integer number"}})
		return
	}
	owner, err := s.queries.GetCharacterOwner(r.Context(), *body.CharacterID)
	if errors.Is(err, sql.ErrNoRows) {
		writeFieldError(w, http.StatusBadRequest, fmt.Sprintf("Character %d not found", *body.CharacterID), FieldErrorMap{"characterId": {"Character does not exist"}})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load character")
		return
	}
	if owner != user.ID {
		writeError(w, http.StatusForbidden, fmt.Sprintf("Cannot add a character you don't own (character %d)", *body.CharacterID))
		return
	}
	role := derefStr(body.Role, "player")
	if !campaignMemberRoles[role] {
		writeValidationError(w, FieldErrorMap{"role": {"role must be one of: player, gm"}})
		return
	}
	// As duas travas abaixo falhavam ABERTAS (ALE-156): o erro era descartado
	// com `_`, e um erro de banco virava `false`, que significa "pode entrar".
	// Eram as únicas checagens do repositório que um erro ABRIA — as outras
	// engolem o erro e NEGAM (mentem o status, não a decisão). Checagem de
	// autorização ou de unicidade nunca descarta erro: na dúvida, nega.
	if role == "player" {
		hasPc, err := s.queries.HasPlayerPc(r.Context(), sqlcgen.HasPlayerPcParams{Campaignid: cid, Ownerid: user.ID})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not check existing characters")
			return
		}
		if hasPc {
			writeFieldError(w, http.StatusConflict, fmt.Sprintf("You already have a character in campaign %d", cid), FieldErrorMap{"characterId": {"Você já tem um personagem nesta campanha"}})
			return
		}
	}
	// Snapshot model (ALE-33): the roster character is a template; a mesa holds
	// its own copy. Dedupe on "this template already snapshotted here" rather
	// than membership of the source (the source is never a member — the copy is).
	hasCopy, err := s.campaignHasCopyOf(r.Context(), *body.CharacterID, cid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not check campaign roster")
		return
	}
	if hasCopy {
		writeFieldError(w, http.StatusConflict, fmt.Sprintf("Character %d already in campaign %d", *body.CharacterID, cid), FieldErrorMap{"characterId": {"Already a member"}})
		return
	}
	m, err := s.joinCampaign(r.Context(), *body.CharacterID, cid, user.ID, role)
	if errors.Is(err, errAlreadyInCampaign) {
		// Perdeu a corrida para um pedido simultâneo: o desfecho é o mesmo 409
		// que a checagem de fora daria, e não um 500 que culparia o servidor.
		writeFieldError(w, http.StatusConflict, fmt.Sprintf("Character %d already in campaign %d", *body.CharacterID, cid), FieldErrorMap{"characterId": {"Already a member"}})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not add member")
		return
	}
	writeJSON(w, http.StatusCreated, memberScalars(m))
}

// handleUpdateMemberRole ports updateRole (owner-only).
func (s *Server) handleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	mid, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, ok := s.ownedCampaign(w, r, cid); !ok {
		return
	}
	if !campaignMemberRoles[body.Role] {
		writeValidationError(w, FieldErrorMap{"role": {"role must be one of: player, gm"}})
		return
	}
	m, err := s.queries.GetMember(r.Context(), mid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && m.Campaignid != cid) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Member %d not found", mid))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load member")
		return
	}
	updated, err := s.queries.SetMemberRole(r.Context(), sqlcgen.SetMemberRoleParams{Role: body.Role, ID: mid})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update member")
		return
	}
	writeJSON(w, http.StatusOK, memberScalars(updated))
}

// handleRemoveMember ports members.remove: GM or the character's owner may remove.
func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	mid, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	owners, err := s.queries.GetMemberOwners(r.Context(), mid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && owners.Campaignid != cid) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Member %d not found", mid))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load member")
		return
	}
	uid := currentUser(r).ID
	if owners.Campaignowner != uid && owners.Characterowner != uid {
		writeError(w, http.StatusForbidden, "You are neither the GM of this campaign nor the character's owner")
		return
	}
	if err := s.queries.DeleteMember(r.Context(), mid); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not remove member")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": mid})
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
func (s *Server) joinCampaign(ctx context.Context, sourceID, campaignID, ownerID int64, role string) (sqlcgen.CampaignMember, error) {
	tx, err := s.db.BeginTx(ctx, nil)
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
	if err := assertCanJoin(ctx, s.queries.WithTx(tx), tx, sourceID, campaignID, ownerID, role); err != nil {
		return sqlcgen.CampaignMember{}, err
	}

	copyID, err := cloneCharacterTx(ctx, tx, sourceID, campaignID)
	if err != nil {
		return sqlcgen.CampaignMember{}, err
	}
	member, err := s.queries.WithTx(tx).CreateMember(ctx, sqlcgen.CreateMemberParams{
		Campaignid: campaignID, Characterid: copyID, Role: role, Addedat: nowISO(),
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
