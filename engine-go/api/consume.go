package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"t20engine/plataforma"

	"t20engine/catalog"
	"t20engine/db"
	"t20engine/db/sqlcgen"
)

type consumeItemResult struct {
	ID       int64 `json:"id"`
	Quantity int64 `json:"quantity"`
	Removed  bool  `json:"removed"`
}

type consumeResult struct {
	Item      consumeItemResult `json:"item"`
	Effect    *EffectDTO        `json:"effect"`
	HpCurrent int64             `json:"hpCurrent"`
	MpCurrent int64             `json:"mpCurrent"`
}

// handleConsumeItem roll the instant
// gain (clamped to max), create the scene/day effect (if any), decrement/remove
// the item — all in one transaction. hpRolled/mpRolled override the dice.
func (s *Server) handleConsumeItem(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	itemID, ok := intParam(w, r, "itemId")
	if !ok {
		return
	}
	var body struct {
		HpRolled *int64 `json:"hpRolled"`
		MpRolled *int64 `json:"mpRolled"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not load character")
		return
	}
	item := findItemDTO(dto.Items, itemID)
	if item == nil {
		plataforma.WriteError(w, http.StatusNotFound, fmt.Sprintf("Item %d not found", itemID))
		return
	}
	if item.CatalogID == nil {
		plataforma.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Item %d is custom and has no consumable spec", itemID))
		return
	}
	cat, known := catalog.LookupItem(*item.CatalogID)
	if !known || cat.Consumable == nil {
		plataforma.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Item %q is not consumable", item.Name))
		return
	}
	if item.Quantity < 1 {
		plataforma.WriteError(w, http.StatusBadRequest, fmt.Sprintf("No remaining uses of %q", item.Name))
		return
	}
	spec := cat.Consumable
	if spec.OncePerDay {
		for _, e := range dto.ActiveEffects {
			if e.CatalogID == cat.ID {
				writeOncePerDay(w, cat.Name)
				return
			}
		}
	}

	hpGain, hasHp := rollGain(body.HpRolled, spec.Instant, true)
	mpGain, hasMp := rollGain(body.MpRolled, spec.Instant, false)

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not consume item")
		return
	}
	defer func() { _ = tx.Rollback() }()
	q := s.queries.WithTx(tx)
	now := plataforma.NowISO()

	var effect *EffectDTO
	if wantsEffectRow(spec) {
		eff, err := q.CreateActiveEffect(r.Context(), sqlcgen.CreateActiveEffectParams{
			Characterid: row.ID, Catalogid: cat.ID, Scope: spec.Scope, Modifiers: effectModifiers(spec.Modifiers), Createdat: now,
		})
		if db.IsUniqueViolation(err) {
			writeOncePerDay(w, cat.Name)
			return
		}
		if err != nil {
			plataforma.WriteError(w, http.StatusInternalServerError, "Could not apply effect")
			return
		}
		effect = &EffectDTO{ID: eff.ID, CatalogID: eff.Catalogid, Scope: eff.Scope, Modifiers: eff.Modifiers, CreatedAt: eff.Createdat}
	}

	removed := false
	newQty := item.Quantity - 1
	if item.Quantity > 1 {
		if err := q.SetItemQuantity(r.Context(), sqlcgen.SetItemQuantityParams{Quantity: newQty, ID: itemID}); err != nil {
			plataforma.WriteError(w, http.StatusInternalServerError, "Could not update item")
			return
		}
	} else {
		if err := q.DeleteItem(r.Context(), itemID); err != nil {
			plataforma.WriteError(w, http.StatusInternalServerError, "Could not remove item")
			return
		}
		removed, newQty = true, 0
	}

	hpCurrent, mpCurrent := row.Hpcurrent, row.Mpcurrent
	if hasHp || hasMp {
		if hasHp {
			hpCurrent = min(row.Hpmax, row.Hpcurrent+int64(hpGain))
		}
		if hasMp {
			mpCurrent = min(row.Mpmax, row.Mpcurrent+int64(mpGain))
		}
		if err := q.SetVitalsCurrent(r.Context(), sqlcgen.SetVitalsCurrentParams{HpCurrent: hpCurrent, MpCurrent: mpCurrent, UpdatedAt: now, ID: row.ID}); err != nil {
			plataforma.WriteError(w, http.StatusInternalServerError, "Could not apply vitals")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not consume item")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, consumeResult{
		Item: consumeItemResult{ID: itemID, Quantity: newQty, Removed: removed}, Effect: effect,
		HpCurrent: hpCurrent, MpCurrent: mpCurrent,
	})
}

func writeOncePerDay(w http.ResponseWriter, name string) {
	plataforma.WriteFieldError(w, http.StatusBadRequest, fmt.Sprintf("%q already active for the day", name), plataforma.FieldErrorMap{"catalogId": {"Apenas uma porção por dia"}})
}

func findItemDTO(items []ItemDTO, itemID int64) *ItemDTO {
	for i := range items {
		if items[i].ID == itemID {
			return &items[i]
		}
	}
	return nil
}

func rollGain(rolled *int64, instant *catalog.Instant, isHp bool) (int, bool) {
	if instant == nil {
		return 0, false
	}
	g := instant.Mp
	if isHp {
		g = instant.Hp
	}
	if g == nil {
		return 0, false
	}
	if rolled != nil {
		return int(*rolled), true
	}
	return rollAverage(g.Dice, g.Bonus), true
}

// wantsEffectRow decide se a dose deixa LINHA de efeito. A linha é duas coisas
// ao mesmo tempo: o que a ficha mostra em "Efeitos ativos" e o MARCADOR de que
// a porção do dia já foi consumida — é ela que o `oncePerDay` lá em cima
// procura, e é ela que o UNIQUE (characterId, catalogId, scope) protege.
//
// Exigir modificadores para criá-la matava a regra da porção diária inteira:
// os cinco pratos que o catálogo marca como `oncePerDay` (gorad quente,
// macarrão de Yuvalin, batata valkariana, prato do aventureiro, sopa de peixe)
// só curam, nenhum tem modificador — então nunca havia marcador para achar, e
// a mesa comia o mesmo prato a manhã inteira (ALE-186).
func wantsEffectRow(spec *catalog.Consumable) bool {
	return spec.Scope != "instant" && (spec.OncePerDay || hasModifiers(spec.Modifiers))
}

// effectModifiers normaliza o blob do catálogo. A coluna é NOT NULL e a ficha
// faz JSON.parse nela: o marcador sem modificador guarda "[]" e não "".
func effectModifiers(raw json.RawMessage) string {
	if !hasModifiers(raw) {
		return "[]"
	}
	return string(raw)
}

func hasModifiers(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s != "" && s != "null" && s != "[]"
}

var diceRe = regexp.MustCompile(`^(\d+)d(\d+)$`)

// rollAverage ports characters.helpers.ts rollAverage: NdF → floor(N*(F+1)/2),
// a plain integer as a flat bonus, "" / "0" → just the bonus.
func rollAverage(dice string, bonus int) int {
	t := strings.TrimSpace(dice)
	if t == "" || t == "0" {
		return bonus
	}
	if flat, err := strconv.Atoi(t); err == nil {
		return flat + bonus
	}
	m := diceRe.FindStringSubmatch(strings.ToLower(t))
	if m == nil {
		return bonus
	}
	n, _ := strconv.Atoi(m[1])
	f, _ := strconv.Atoi(m[2])
	return (n*(f+1))/2 + bonus
}
