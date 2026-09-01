package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"t20engine/plataforma"

	"github.com/go-chi/chi/v5"
	"t20engine/catalog"
	"t20engine/db/sqlcgen"
)

// spellBasePmCost is Tabela 4-1, "Custo de Magias" (livro p170).
var spellBasePmCost = map[int]int{0: 0, 1: 1, 2: 3, 3: 6, 4: 10, 5: 15}

// alwaysPrepareClasses mirrors ALWAYS_PREPARE_CLASSES.
var alwaysPrepareClasses = map[string]bool{"Clérigo": true, "Druida": true}

type augmentPick struct {
	AugmentIndex int `json:"augmentIndex"`
	Stacks       int `json:"stacks"`
}

type castResult struct {
	MpCurrent        int64   `json:"mpCurrent"`
	RemovedEffectIDs []int64 `json:"removedEffectIds"`
}

// handleCastSpell validate learned +
// prepared + augments, check the PM cost against the per-spell limit and current
// PM, then deduct. NOTE: the catalisador scene-discount is deferred (rare edge);
// removedEffectIds is therefore always empty.
func (s *Server) handleCastSpell(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	catalogSpellID := chi.URLParam(r, "catalogSpellId")
	var body struct {
		Augments []augmentPick `json:"augments"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if _, known := catalog.LookupSpell(catalogSpellID); !known {
		plataforma.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Unknown spell %q", catalogSpellID))
		return
	}
	dto, err := s.LoadCharacter(r.Context(), row)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not Load character")
		return
	}

	// A REGRA MORA NUM LUGAR SÓ. Este handler já teve cópia própria do teto da
	// p224 e ela discordava da ficha sobre qual classe conta (ALE-92); desde a
	// ALE-272 as duas pontas — este endpoint e o gesto da ficha em Datastar —
	// chamam a MESMA função.
	if err := s.castSpellForCharacter(r, dto, catalogSpellID, body.Augments); err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	atualizado, err := s.queries.GetCharacter(r.Context(), row.ID)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not read character")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, castResult{MpCurrent: atualizado.Mpcurrent, RemovedEffectIDs: []int64{}})
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
	r *http.Request, dto CharacterDTO, catalogSpellID string, augments []augmentPick,
) error {
	spell, known := catalog.LookupSpell(catalogSpellID)
	if !known {
		return fmt.Errorf("a magia %q não existe no livro", catalogSpellID)
	}
	learned := findSpell(dto.Spells, catalogSpellID)
	if learned == nil {
		return fmt.Errorf("%q não está no grimório desta ficha", catalogSpellID)
	}
	if requiresPreparation(dto.Classes, dto.ClassChoices) && !learned.Prepared {
		return fmt.Errorf("prepare a magia antes de conjurá-la")
	}
	augmentPm, augErr := validateAugments(spell, augments,
		highestCastableCircle(dto.Classes, spell.Circle))
	if augErr != "" {
		return fmt.Errorf("%s", augErr)
	}
	ec, err := engineCharacterFrom(dto)
	if err != nil {
		return err
	}
	basePm := spellBasePmCost[spell.Circle]
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

func findSpell(spells []SpellDTO, catalogSpellID string) *SpellDTO {
	for i := range spells {
		if spells[i].CatalogSpellID == catalogSpellID {
			return &spells[i]
		}
	}
	return nil
}

// requiresPreparation ports the same helper: Clérigo/Druida always prepare, and
// an Arcanista on the "mago" caminho does.
func requiresPreparation(classes []ClassDTO, classChoicesRaw string) bool {
	hasArcanista := false
	for _, c := range classes {
		if alwaysPrepareClasses[c.ClassName] {
			return true
		}
		if c.ClassName == "Arcanista" {
			hasArcanista = true
		}
	}
	if !hasArcanista {
		return false
	}
	var choices map[string]struct {
		Caminho string `json:"caminho"`
	}
	_ = json.Unmarshal([]byte(classChoicesRaw), &choices)
	return choices["Arcanista"].Caminho == "mago"
}

// validateAugments confere os aprimoramentos escolhidos e devolve o PM deles,
// ou a frase da recusa.
//
// O `castableCircle` fechou uma FRONTEIRA que estava aberta (ALE-272, fatia 6):
// 126 dos 486 aprimoramentos do catálogo exigem um círculo mínimo, e até aqui
// esse limite existia só na tela. A tabela que o decide vivia só no TypeScript,
// então o servidor nem tinha como perguntar — e um pedido montado à mão
// conjurava o que a regra não permite. Travar na UI é UX; a fronteira é aqui.
func validateAugments(spell catalog.Spell, picks []augmentPick, castableCircle int) (int, string) {
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
