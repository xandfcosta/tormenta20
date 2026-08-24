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
	spell, known := catalog.LookupSpell(catalogSpellID)
	if !known {
		plataforma.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Unknown spell %q", catalogSpellID))
		return
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not Load character")
		return
	}

	learned := findSpell(dto.Spells, catalogSpellID)
	if learned == nil {
		plataforma.WriteError(w, http.StatusNotFound, fmt.Sprintf("Spell %q not in character's spellbook", catalogSpellID))
		return
	}
	if requiresPreparation(dto.Classes, dto.ClassChoices) && !learned.Prepared {
		plataforma.WriteError(w, http.StatusForbidden, fmt.Sprintf("Spell %q must be prepared before casting", catalogSpellID))
		return
	}

	augmentPm, augErr := validateAugments(spell, body.Augments)
	if augErr != "" {
		plataforma.WriteError(w, http.StatusBadRequest, augErr)
		return
	}
	basePm := spellBasePmCost[spell.Circle]

	// One rule, one place (ALE-92): the engine owns the p224 ceiling and resolves
	// the item bonus the same way the sheet does. This handler used to carry its
	// own copy, and the two disagreed on which class counts AND on how item
	// bonuses stack — so the sheet offered a cap this gate then refused.
	ec, err := engineCharacterFrom(dto)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not read character")
		return
	}
	// p226: o modificador de custo entra AQUI, e não só na ficha — a redução era
	// calculada, exibida no mosaico "Custo PM" e ignorada na hora de cobrar
	// (ALE-110). O piso de 1 PM vem junto.
	totalPm := s.catalogs.SpellPmCostFor(ec, basePm, augmentPm, map[string]bool{})
	minPm := s.catalogs.SpellPmCostFor(ec, basePm, 0, map[string]bool{})

	limit := s.catalogs.SpellPmLimitFor(ec, spell.Classes)
	// A ressalva entre parênteses da p224: "(mas você sempre pode usar a
	// habilidade em seu custo mínimo)". O teto limita o gasto ADICIONAL — ele
	// nunca torna inconjurável uma magia que o personagem já possui, o que
	// acontecia com uma magia de círculo alto vinda de fora da classe.
	if spell.Circle > 0 && totalPm > limit && totalPm > minPm {
		plataforma.WriteFieldError(w, http.StatusBadRequest, fmt.Sprintf("PM cost %d exceeds per-spell limit %d", totalPm, limit), plataforma.FieldErrorMap{"augments": {fmt.Sprintf("Limite PM excedido (%d)", limit)}})
		return
	}
	if int64(totalPm) > dto.MpCurrent {
		plataforma.WriteFieldError(w, http.StatusBadRequest, fmt.Sprintf("Insufficient PM (need %d, have %d)", totalPm, dto.MpCurrent), plataforma.FieldErrorMap{"mpCurrent": {"Sem PM suficiente"}})
		return
	}

	if totalPm == 0 {
		plataforma.WriteJSON(w, http.StatusOK, castResult{MpCurrent: dto.MpCurrent, RemovedEffectIDs: []int64{}})
		return
	}
	mpCurrent := dto.MpCurrent - int64(totalPm)
	if err := s.queries.SetMpCurrent(r.Context(), sqlcgen.SetMpCurrentParams{MpCurrent: mpCurrent, UpdatedAt: plataforma.NowISO(), ID: row.ID}); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not cast spell")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, castResult{MpCurrent: mpCurrent, RemovedEffectIDs: []int64{}})
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

// validateAugments ports validateAugments: returns the total augment PM, or a
// message to 400 with.
func validateAugments(spell catalog.Spell, picks []augmentPick) (int, string) {
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
		total += a.PmCost * p.Stacks
	}
	return total, ""
}
