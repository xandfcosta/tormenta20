package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"t20engine/catalog"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

// spellBasePmCost mirrors t20-data SPELL_BASE_PM_COST (PDF p171).
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

// handleCastSpell ports CharactersSpellsService.castSpell: validate learned +
// prepared + augments, check the PM cost against the per-spell limit and current
// PM, then deduct. NOTE: the catalisador scene-discount is deferred (rare edge);
// removedEffectIds is therefore always empty.
func (s *Server) handleCastSpell(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	catalogSpellID := chi.URLParam(r, "catalogSpellId")
	var body struct {
		Augments []augmentPick `json:"augments"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	row, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	spell, known := catalog.LookupSpell(catalogSpellID)
	if !known {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Unknown spell %q", catalogSpellID))
		return
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load character")
		return
	}

	learned := findSpell(dto.Spells, catalogSpellID)
	if learned == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Spell %q not in character's spellbook", catalogSpellID))
		return
	}
	if requiresPreparation(dto.Classes, dto.ClassChoices) && !learned.Prepared {
		writeError(w, http.StatusForbidden, fmt.Sprintf("Spell %q must be prepared before casting", catalogSpellID))
		return
	}

	augmentPm, augErr := validateAugments(spell, body.Augments)
	if augErr != "" {
		writeError(w, http.StatusBadRequest, augErr)
		return
	}
	totalPm := 0
	if spell.Circle != 0 {
		totalPm = max(0, spellBasePmCost[spell.Circle]+augmentPm)
	}

	limit := spellPmLimitFor(int(dto.Level), dto.Classes, spell.Classes, pmLimitFromItems(s.catalogs, dto.Items))
	if spell.Circle > 0 && totalPm > limit {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"statusCode":  http.StatusBadRequest,
			"error":       "Bad Request",
			"message":     fmt.Sprintf("PM cost %d exceeds per-spell limit %d", totalPm, limit),
			"fieldErrors": FieldErrorMap{"augments": {fmt.Sprintf("Limite PM excedido (%d)", limit)}},
		})
		return
	}
	if int64(totalPm) > dto.MpCurrent {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"statusCode":  http.StatusBadRequest,
			"error":       "Bad Request",
			"message":     fmt.Sprintf("Insufficient PM (need %d, have %d)", totalPm, dto.MpCurrent),
			"fieldErrors": FieldErrorMap{"mpCurrent": {"Sem PM suficiente"}},
		})
		return
	}

	if totalPm == 0 {
		writeJSON(w, http.StatusOK, castResult{MpCurrent: dto.MpCurrent, RemovedEffectIDs: []int64{}})
		return
	}
	mpCurrent := dto.MpCurrent - int64(totalPm)
	if err := s.queries.SetMpCurrent(r.Context(), sqlcgen.SetMpCurrentParams{MpCurrent: mpCurrent, UpdatedAt: nowISO(), ID: id}); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not cast spell")
		return
	}
	writeJSON(w, http.StatusOK, castResult{MpCurrent: mpCurrent, RemovedEffectIDs: []int64{}})
}

// spellPmLimitFor is the per-use PM ceiling (p224): the level in the CLASS that
// provides the ability, or the character level when no class does — a spell
// granted by a race, origin or power. Item `pmLimit` bonuses add on top.
//
// It used to be `level/2`, from a Nest comment citing "p171 — ½ nível" that
// misread the book: p171 defers to p224 and its own example spends the FULL
// level ("um arcanista de 11º nível pode gastar até 11 PM"). The halving made
// the server refuse casts the sheet correctly offered.
func spellPmLimitFor(characterLevel int, classes []ClassDTO, spellClasses []string, itemBonus int) int {
	onList := map[string]bool{}
	for _, name := range spellClasses {
		onList[name] = true
	}
	best := 0
	for _, c := range classes {
		if onList[c.ClassName] && int(c.Level) > best {
			best = int(c.Level)
		}
	}
	if best == 0 {
		best = characterLevel
	}
	return max(1, best) + itemBonus
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

// pmLimitFromItems ports pmLimitFromItems: sum of `pmLimit` modifiers on equipped
// catalog items (base + improvements + material).
func pmLimitFromItems(cats *engine.Catalogs, items []ItemDTO) int {
	if cats == nil {
		return 0
	}
	total := 0
	for _, it := range items {
		if it.Equipped == nil || it.CatalogID == nil {
			continue
		}
		var mods []engine.Modifier
		if base := cats.Item(*it.CatalogID); base != nil {
			mods = append(mods, base.Modifiers...)
		}
		var imps []string
		_ = json.Unmarshal([]byte(it.Improvements), &imps)
		for _, impID := range imps {
			if imp := cats.Item(impID); imp != nil {
				mods = append(mods, imp.Modifiers...)
			}
		}
		if it.Material != nil {
			if mat := cats.Item(*it.Material); mat != nil {
				mods = append(mods, mat.Modifiers...)
			}
		}
		for _, m := range mods {
			if m.Target.K == "pmLimit" {
				total += m.Amount
			}
		}
	}
	return total
}
