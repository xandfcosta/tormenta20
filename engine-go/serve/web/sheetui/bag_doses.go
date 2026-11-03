package sheetui

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// A INJEÇÃO ALQUÍMICA NA MOCHILA (p165). O gasto e a regra moram no `Plays.LoadDose`.

// itemSheetDoses é o que o diálogo do item diz da Injeção alquímica. Um item é uma das
// duas coisas: a arma que guarda doses, ou o item que se carrega nela.
type itemSheetDoses struct {
	// Held são as doses guardadas e Count a conta delas, na arma.
	Held  []overlayRow
	Count string
	// Loads diz que o item carrega, e LoadInto são as armas com espaço para ele agora.
	Loads    bool
	LoadInto []doseTarget
}

// CanLoad diz se o Usar da mochila tem o que oferecer. Sem arma com espaço, o item
// não tem Usar: para gastá-lo na interpretação, a pessoa o remove (decisão do dono).
func (d *itemSheetDoses) CanLoad() bool { return d != nil && len(d.LoadInto) > 0 }

// doseTarget é uma arma que pode receber o item.
type doseTarget struct {
	Command string
	Label   string
}

func dosesOnTheItemSheet(
	dto sheet.CharacterDTO, item sheet.ItemDTO, attributes map[string]engine.AttributeBreakdown,
) *itemSheetDoses {
	if sheet.HoldsDoses(item.Improvements) {
		doses := &itemSheetDoses{Count: fmt.Sprintf("%d de %d", len(item.Doses), sheet.MaxDoses)}
		for _, id := range item.Doses {
			name, effect := sheet.WrittenDose(id, dto.Level, attributes)
			doses.Held = append(doses.Held, overlayRow{Name: name, Effect: effect})
		}
		return doses
	}
	if item.CatalogID == nil || sheet.DoseEffectOf(*item.CatalogID) == nil {
		return nil
	}
	doses := &itemSheetDoses{Loads: true}
	for _, weapon := range dto.Items {
		if sheet.HoldsDoses(weapon.Improvements) && len(weapon.Doses) < sheet.MaxDoses {
			doses.LoadInto = append(doses.LoadInto, doseTarget{
				Command: strconv.FormatInt(weapon.ID, 10),
				Label:   fmt.Sprintf("%s · %d de %d", weapon.Name, len(weapon.Doses), sheet.MaxDoses),
			})
		}
	}
	return doses
}

// loadDose carrega o item da mochila na arma do endereço.
//
// RPG é jogo de turno: quem propôs um ataque não faz outra coisa até ele acabar
// (decisão do dono). Sem isso, a dose devolvida por um cancelamento acharia a arma
// recarregada, e ela ficaria com mais do que cabe.
func loadDose(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	attacking, err := s.deps.HasAttackInProgress(r.Context(), row.ID)
	if err != nil {
		return err
	}
	if attacking {
		return fmt.Errorf("%s tem um ataque na mesa: carregue depois de ele acabar", row.Name)
	}
	weapon, err := strconv.ParseInt(chi.URLParam(r, "arma"), 10, 64)
	if err != nil {
		return fmt.Errorf("a arma vem por id no caminho, e veio %q", chi.URLParam(r, "arma"))
	}
	return s.plays.LoadDose(r.Context(), row.ID, item.ID, weapon)
}

// A DOSE NA LINHA DE ATAQUE.

const (
	doseWeaponSignal = "dose_weapon"
	doseChoiceSignal = "dose_choice"
)

// doseChoice é uma dose guardada na arma.
type doseChoice struct {
	ID   string
	Name string
}

func doseChoicesOf(ids []string) []doseChoice {
	choices := make([]doseChoice, 0, len(ids))
	for _, id := range ids {
		choice := doseChoice{ID: id, Name: id}
		if item := book.ItemByID(id); item != nil {
			choice.Name = item.Name
		}
		choices = append(choices, choice)
	}
	return choices
}

// writtenDoses é a linha pequena da arma: "Doses: Ácido, Fogo alquímico".
func writtenDoses(doses []doseChoice) string {
	names := make([]string, 0, len(doses))
	for _, dose := range doses {
		names = append(names, dose.Name)
	}
	return "Doses: " + strings.Join(names, ", ")
}

// doseChoicesToAsk são as opções da pergunta, que só existe entre doses diferentes
// (decisão do dono): duas iguais não têm o que escolher.
func doseChoicesToAsk(doses []doseChoice) []doseChoice {
	distinct := make([]doseChoice, 0, len(doses))
	for _, dose := range doses {
		if !slices.Contains(distinct, dose) {
			distinct = append(distinct, dose)
		}
	}
	if len(distinct) < 2 {
		return nil
	}
	return distinct
}

// doseChosenExpr acende a opção escolhida. Sem escolha nesta arma vale a primeira,
// que é a mais antiga: é o que o servidor faz com o pedido sem dose.
func doseChosenExpr(weapon int, dose doseChoice, first bool) string {
	if first {
		return fmt.Sprintf("!($%s === %d && $%s && $%s !== %q)",
			doseWeaponSignal, weapon, doseChoiceSignal, doseChoiceSignal, dose.ID)
	}
	return fmt.Sprintf("$%s === %d && $%s === %q", doseWeaponSignal, weapon, doseChoiceSignal, dose.ID)
}

func doseChooseCommand(weapon int, dose doseChoice) string {
	return fmt.Sprintf("$%s = %d; $%s = %q", doseWeaponSignal, weapon, doseChoiceSignal, dose.ID)
}

// rollClearsTheDose zera a escolha depois do golpe, porque a dose pode ter saído.
func rollClearsTheDose() string { return "; $" + doseChoiceSignal + " = ''" }

// chosenDose lê a escolha, que só vale para a arma em que foi feita.
func chosenDose(sig Signals, weapon int) string {
	if sig.DoseWeapon == nil || int(*sig.DoseWeapon) != weapon {
		return ""
	}
	return sig.DoseChoice
}
