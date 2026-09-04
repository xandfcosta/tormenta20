package api

import (
	"encoding/json"
	"net/http"
	"t20engine/plataforma"

	"t20engine/creature"
	"t20engine/db/sqlcgen"
)

// creatureDTO é o que sai pela API: identidade fora, bloco dentro. O nome mora
// numa coluna (é por onde se lista e se ordena) e o resto é o JSON do bloco —
// ter o nome nos dois lugares criaria duas verdades.
type creatureDTO struct {
	ID         int64          `json:"id"`
	CampaignID int64          `json:"campaignId"`
	Name       string         `json:"name"`
	Block      creature.Block `json:"block"`
	CreatedAt  string         `json:"createdAt"`
	UpdatedAt  string         `json:"updatedAt"`
}

type creatureInput struct {
	Name  string         `json:"name"`
	Block creature.Block `json:"block"`
}

func creatureToDTO(row sqlcgen.CampaignCreature) (creatureDTO, error) {
	var block creature.Block
	if err := json.Unmarshal([]byte(row.Block), &block); err != nil {
		return creatureDTO{}, err
	}
	creature.Normalize(&block)
	return creatureDTO{
		ID: row.ID, CampaignID: row.Campaignid, Name: row.Name, Block: block,
		CreatedAt: row.Createdat, UpdatedAt: row.Updatedat,
	}, nil
}

func decodeCreature(w http.ResponseWriter, r *http.Request) (creatureInput, bool) {
	var input creatureInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, "Body must be {name, block}")
		return input, false
	}
	creature.Normalize(&input.Block)
	if err := creature.Validate(input.Name, &input.Block); err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return input, false
	}
	return input, true
}

func writeCreature(w http.ResponseWriter, status int, row sqlcgen.CampaignCreature) {
	dto, err := creatureToDTO(row)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Stored creature block is unreadable")
		return
	}
	plataforma.WriteJSON(w, status, dto)
}
