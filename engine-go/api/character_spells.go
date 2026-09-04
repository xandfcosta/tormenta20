package api

import (
	"t20engine/db/sqlcgen"
)

// spellRowDTO is the CharacterSpell row Prisma returns from the spell mutations
// (full row incl. characterId; prepared as a bool from the INTEGER column).
type spellRowDTO struct {
	ID             int64  `json:"id"`
	CharacterID    int64  `json:"characterId"`
	CatalogSpellID string `json:"catalogSpellId"`
	Prepared       bool   `json:"prepared"`
	LearnedAt      string `json:"learnedAt"`
}

func spellRowFrom(s sqlcgen.CharacterSpell) spellRowDTO {
	return spellRowDTO{
		ID: s.ID, CharacterID: s.Characterid, CatalogSpellID: s.Catalogspellid,
		Prepared: s.Prepared != 0, LearnedAt: s.Learnedat,
	}
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
