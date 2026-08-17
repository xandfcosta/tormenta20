package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"t20engine/db/sqlcgen"
)

// creatureDTO é o que sai pela API: identidade fora, bloco dentro. O nome mora
// numa coluna (é por onde se lista e se ordena) e o resto é o JSON do bloco —
// ter o nome nos dois lugares criaria duas verdades.
type creatureDTO struct {
	ID         int64         `json:"id"`
	CampaignID int64         `json:"campaignId"`
	Name       string        `json:"name"`
	Block      CreatureBlock `json:"block"`
	CreatedAt  string        `json:"createdAt"`
	UpdatedAt  string        `json:"updatedAt"`
}

type creatureInput struct {
	Name  string        `json:"name"`
	Block CreatureBlock `json:"block"`
}

func creatureToDTO(row sqlcgen.CampaignCreature) (creatureDTO, error) {
	var block CreatureBlock
	if err := json.Unmarshal([]byte(row.Block), &block); err != nil {
		return creatureDTO{}, err
	}
	normalizeCreature(&block)
	return creatureDTO{
		ID: row.ID, CampaignID: row.Campaignid, Name: row.Name, Block: block,
		CreatedAt: row.Createdat, UpdatedAt: row.Updatedat,
	}, nil
}

// requireGM é a porta desta família inteira: o bloco de criatura é informação
// do MESTRE. O jogador continua vendo o que via — nome e barra de PV pela
// iniciativa, com a regra de PV oculto (ALE-137). Aqui a recusa é do servidor,
// não da tela: esconder o botão é UX, o limite é aqui.
func (s *Server) requireGM(w http.ResponseWriter, r *http.Request, campaignID int64) bool {
	role, status, err := s.resolveRole(r.Context(), currentUser(r), campaignID)
	if err != nil {
		writeError(w, status, err.Error())
		return false
	}
	if role != "gm" {
		writeError(w, http.StatusForbidden, "Only the GM may read or write campaign creatures")
		return false
	}
	return true
}

func (s *Server) handleListCreatures(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok || !s.requireGM(w, r, cid) {
		return
	}
	rows, err := s.queries.ListCampaignCreatures(r.Context(), cid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not list creatures")
		return
	}
	out := make([]creatureDTO, 0, len(rows))
	for _, row := range rows {
		dto, err := creatureToDTO(row)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Stored creature block is unreadable")
			return
		}
		out = append(out, dto)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateCreature(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok || !s.requireGM(w, r, cid) {
		return
	}
	input, ok := decodeCreature(w, r)
	if !ok {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	blob, err := json.Marshal(input.Block)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not store creature block")
		return
	}
	row, err := s.queries.CreateCampaignCreature(r.Context(), sqlcgen.CreateCampaignCreatureParams{
		Campaignid: cid, Name: strings.TrimSpace(input.Name), Block: string(blob),
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create creature")
		return
	}
	writeCreature(w, http.StatusCreated, row)
}

func (s *Server) handleUpdateCreature(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok || !s.requireGM(w, r, cid) {
		return
	}
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	// A criatura tem de ser DESTA campanha: sem esta conferência, o mestre de
	// uma mesa reescreveria o vilão de outra sabendo só o id.
	if !s.creatureBelongsTo(w, r, id, cid) {
		return
	}
	input, ok := decodeCreature(w, r)
	if !ok {
		return
	}
	blob, err := json.Marshal(input.Block)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not store creature block")
		return
	}
	row, err := s.queries.UpdateCampaignCreature(r.Context(), sqlcgen.UpdateCampaignCreatureParams{
		ID: id, Name: strings.TrimSpace(input.Name), Block: string(blob),
		Updatedat: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update creature")
		return
	}
	writeCreature(w, http.StatusOK, row)
}

func (s *Server) handleDeleteCreature(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok || !s.requireGM(w, r, cid) {
		return
	}
	id, ok := intParam(w, r, "id")
	if !ok || !s.creatureBelongsTo(w, r, id, cid) {
		return
	}
	if err := s.queries.DeleteCampaignCreature(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not delete creature")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) creatureBelongsTo(w http.ResponseWriter, r *http.Request, id, campaignID int64) bool {
	row, err := s.queries.GetCampaignCreature(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && row.Campaignid != campaignID) {
		writeError(w, http.StatusNotFound, "Creature not found in this campaign")
		return false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load creature")
		return false
	}
	return true
}

func decodeCreature(w http.ResponseWriter, r *http.Request) (creatureInput, bool) {
	var input creatureInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Body must be {name, block}")
		return input, false
	}
	normalizeCreature(&input.Block)
	if err := validateCreature(input.Name, &input.Block); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return input, false
	}
	return input, true
}

func writeCreature(w http.ResponseWriter, status int, row sqlcgen.CampaignCreature) {
	dto, err := creatureToDTO(row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Stored creature block is unreadable")
		return
	}
	writeJSON(w, status, dto)
}
