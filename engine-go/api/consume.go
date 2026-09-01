package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"t20engine/plataforma"

	"t20engine/catalog"
	"t20engine/db"
	"t20engine/db/sqlcgen"
	"t20engine/sheet"
)

type consumeItemResult struct {
	ID       int64 `json:"id"`
	Quantity int64 `json:"quantity"`
	Removed  bool  `json:"removed"`
}

type consumeResult struct {
	Item      consumeItemResult `json:"item"`
	Effect    *sheet.EffectDTO  `json:"effect"`
	HpCurrent int64             `json:"hpCurrent"`
	MpCurrent int64             `json:"mpCurrent"`
}

// errPorcaoDoDia é a recusa da porção diária, que a API JSON responde com um
// erro de CAMPO próprio — por isso ela é reconhecível em vez de virar texto.
var errPorcaoDoDia = errors.New("apenas uma porção por dia")

// handleConsumeItem roll the instant
// gain (clamped to max), create the scene/day effect (if any), decrement/Remove
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
	resultado, err := s.consumeItemForCharacter(r, row, itemID, body.HpRolled, body.MpRolled)
	if errors.Is(err, errPorcaoDoDia) {
		writeOncePerDay(w, resultado.Nome)
		return
	}
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, resultado.consumeResult)
}

// consumeItemForCharacter é a dose INTEIRA, sem HTTP: a rolagem imediata presa
// no máximo, a linha de efeito de cena ou dia, e a baixa do item — tudo numa
// transação.
//
// Ela nasceu extraída na fatia 7 da ALE-272, quando a Mochila em Datastar
// passou a usar item: reescrever a regra lá daria DUAS respostas para "posso
// beber esta poção?", e elas divergiriam no dia em que uma mudasse. É a mesma
// razão do `castSpellForCharacter` da fatia 6.
func (s *Server) consumeItemForCharacter(
	r *http.Request, row sqlcgen.Character, itemID int64, hpRolled, mpRolled *int64,
) (doseUsada, error) {
	dto, err := s.LoadCharacter(r.Context(), row)
	if err != nil {
		return doseUsada{}, err
	}
	item := findItemDTO(dto.Items, itemID)
	if item == nil {
		return doseUsada{}, fmt.Errorf("o item %d não está nesta ficha", itemID)
	}
	if item.CatalogID == nil {
		return doseUsada{}, fmt.Errorf("%q é um item custom e não tem o que usar", item.Name)
	}
	cat, known := catalog.LookupItem(*item.CatalogID)
	if !known || cat.Consumable == nil {
		return doseUsada{}, fmt.Errorf("%q não é um consumível", item.Name)
	}
	if item.Quantity < 1 {
		return doseUsada{}, fmt.Errorf("não sobrou nenhum uso de %q", item.Name)
	}
	spec := cat.Consumable
	if spec.OncePerDay {
		for _, e := range dto.ActiveEffects {
			if e.CatalogID == cat.ID {
				return doseUsada{Nome: cat.Name}, errPorcaoDoDia
			}
		}
	}

	hpGain, hasHp := rollGain(hpRolled, spec.Instant, true)
	mpGain, hasMp := rollGain(mpRolled, spec.Instant, false)

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		return doseUsada{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.queries.WithTx(tx)
	now := plataforma.NowISO()

	var effect *sheet.EffectDTO
	if wantsEffectRow(spec) {
		eff, err := q.CreateActiveEffect(r.Context(), sqlcgen.CreateActiveEffectParams{
			Characterid: row.ID, Catalogid: cat.ID, Scope: spec.Scope, Modifiers: effectModifiers(spec.Modifiers), Createdat: now,
		})
		if db.IsUniqueViolation(err) {
			return doseUsada{Nome: cat.Name}, errPorcaoDoDia
		}
		if err != nil {
			return doseUsada{}, err
		}
		effect = &sheet.EffectDTO{ID: eff.ID, CatalogID: eff.Catalogid, Scope: eff.Scope, Modifiers: eff.Modifiers, CreatedAt: eff.Createdat}
	}

	removed := false
	newQty := item.Quantity - 1
	if item.Quantity > 1 {
		if err := q.SetItemQuantity(r.Context(), sqlcgen.SetItemQuantityParams{Quantity: newQty, ID: itemID}); err != nil {
			return doseUsada{}, err
		}
	} else {
		if err := q.DeleteItem(r.Context(), itemID); err != nil {
			return doseUsada{}, err
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
			return doseUsada{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return doseUsada{}, err
	}
	return doseUsada{
		Nome: cat.Name,
		consumeResult: consumeResult{
			Item: consumeItemResult{ID: itemID, Quantity: newQty, Removed: removed}, Effect: effect,
			HpCurrent: hpCurrent, MpCurrent: mpCurrent,
		},
	}, nil
}

// doseUsada é o resultado da dose mais o NOME do item, que a recusa da porção
// diária precisa e o corpo da resposta JSON não carrega.
type doseUsada struct {
	consumeResult
	Nome string
}

func writeOncePerDay(w http.ResponseWriter, name string) {
	plataforma.WriteFieldError(w, http.StatusBadRequest, fmt.Sprintf("%q already active for the day", name), plataforma.FieldErrorMap{"catalogId": {"Apenas uma porção por dia"}})
}

func findItemDTO(items []sheet.ItemDTO, itemID int64) *sheet.ItemDTO {
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
