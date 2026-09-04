package api

import (
	"fmt"
	"net/http"
	"t20engine/plataforma"

	"t20engine/catalog"
	"t20engine/db/sqlcgen"
	"t20engine/sheet"
)

// O `AugmentPick` mora no `sheet` desde a ALE-278: a cena o lê dos sinais e
// este arquivo o consome ao cobrar o PM.

type castResult struct {
	MpCurrent        int64   `json:"mpCurrent"`
	RemovedEffectIDs []int64 `json:"removedEffectIds"`
}

// castSpellForCharacter é a conjuração INTEIRA, sem HTTP.
//
// Ela nasceu extraída na ALE-272 (fatia 6), quando a ficha em Datastar passou a
// conjurar: escrever as recusas de novo lá daria DUAS regras para a mesma
// pergunta, e elas divergiriam no dia em que uma mudasse. É a mesma razão da
// ALE-110, que registrou o custo sendo exibido num lugar e ignorado no outro.
//
// Devolve o PM que sobrou e uma frase de recusa quando a regra barra — a frase é
// para um humano ler numa tela, e não um `FieldErrorMap` para um cliente.
func (s *Server) castSpellForCharacter(
	r *http.Request, dto sheet.CharacterDTO, catalogSpellID string, augments []sheet.AugmentPick,
) error {
	spell, known := catalog.LookupSpell(catalogSpellID)
	if !known {
		return fmt.Errorf("a magia %q não existe no livro", catalogSpellID)
	}
	learned := findSpell(dto.Spells, catalogSpellID)
	if learned == nil {
		return fmt.Errorf("%q não está no grimório desta ficha", catalogSpellID)
	}
	if sheet.RequiresPreparation(dto.Classes, dto.ClassChoices) && !learned.Prepared {
		return fmt.Errorf("prepare a magia antes de conjurá-la")
	}
	augmentPm, augErr := validateAugments(spell, augments,
		sheet.HighestCastableCircle(dto.Classes, spell.Circle))
	if augErr != "" {
		return fmt.Errorf("%s", augErr)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return err
	}
	basePm := sheet.SpellBasePmCost[spell.Circle]
	totalPm := s.catalogs.SpellPmCostFor(ec, basePm, augmentPm, map[string]bool{})
	minPm := s.catalogs.SpellPmCostFor(ec, basePm, 0, map[string]bool{})
	limit := s.catalogs.SpellPmLimitFor(ec, spell.Classes)
	if spell.Circle > 0 && totalPm > limit && totalPm > minPm {
		return fmt.Errorf("o custo de %d PM passa do limite de %d por magia", totalPm, limit)
	}
	if int64(totalPm) > dto.MpCurrent {
		return fmt.Errorf("faltam PM: a magia custa %d e restam %d", totalPm, dto.MpCurrent)
	}
	if totalPm == 0 {
		return nil
	}
	return s.queries.SetMpCurrent(r.Context(), sqlcgen.SetMpCurrentParams{
		MpCurrent: dto.MpCurrent - int64(totalPm), UpdatedAt: plataforma.NowISO(), ID: dto.ID,
	})
}

func findSpell(spells []sheet.SpellDTO, catalogSpellID string) *sheet.SpellDTO {
	for i := range spells {
		if spells[i].CatalogSpellID == catalogSpellID {
			return &spells[i]
		}
	}
	return nil
}

// validateAugments confere os aprimoramentos escolhidos e devolve o PM deles,
// ou a frase da recusa.
//
// O `castableCircle` fechou uma FRONTEIRA que estava aberta (ALE-272, fatia 6):
// 126 dos 486 aprimoramentos do catálogo exigem um círculo mínimo, e até aqui
// esse limite existia só na tela. A tabela que o decide vivia só no TypeScript,
// então o servidor nem tinha como perguntar — e um pedido montado à mão
// conjurava o que a regra não permite. Travar na UI é UX; a fronteira é aqui.
func validateAugments(spell catalog.Spell, picks []sheet.AugmentPick, castableCircle int) (int, string) {
	if len(picks) == 0 {
		return 0, ""
	}
	if spell.Circle == 0 {
		return 0, "Truques cannot receive aprimoramentos"
	}
	seen := map[int]bool{}
	total := 0
	for _, p := range picks {
		if p.AugmentIndex < 0 || p.AugmentIndex >= len(spell.Augments) {
			return 0, fmt.Sprintf("Invalid augmentIndex %d", p.AugmentIndex)
		}
		if seen[p.AugmentIndex] {
			return 0, fmt.Sprintf("Duplicate augmentIndex %d — combine stacks in one pick", p.AugmentIndex)
		}
		seen[p.AugmentIndex] = true
		if p.Stacks < 1 {
			return 0, fmt.Sprintf("stacks must be an integer ≥ 1 (got %d)", p.Stacks)
		}
		a := spell.Augments[p.AugmentIndex]
		if a.Kind == "muda" && p.Stacks > 1 {
			return 0, fmt.Sprintf("'muda' augment cannot stack (index %d)", p.AugmentIndex)
		}
		if a.RequiresCircle != nil && *a.RequiresCircle > castableCircle {
			return 0, fmt.Sprintf(
				"aprimoramento %d exige o %dº círculo e este personagem alcança o %dº",
				p.AugmentIndex, *a.RequiresCircle, castableCircle)
		}
		total += a.PmCost * p.Stacks
	}
	return total, ""
}
