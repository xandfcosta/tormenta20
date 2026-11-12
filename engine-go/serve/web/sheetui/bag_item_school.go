package sheetui

import (
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// A ESCOLA QUE O ITEM PEDE: o tomo hermético dá +2 na CD "das magias arcanas dessa escola" (p160). Quem
// escolhe é o jogador, na folha do tomo, e pode trocar (decisão do dono).

// itemSchool é a escolha na folha: a escola atual, vazia antes da primeira, e as oito do livro.
type itemSchool struct {
	Chosen  string
	Options []book.SpellSchool
}

// schoolOnTheItemSheet é a escolha do item que a pede, ou nil.
func schoolOnTheItemSheet(item sheet.ItemDTO) *itemSchool {
	entry := catalogItem(item)
	if entry == nil || !slices.ContainsFunc(entry.Modifiers, func(m engine.Modifier) bool { return m.FromChoice == "school" }) {
		return nil
	}
	return &itemSchool{Chosen: item.Choice, Options: book.SpellSchools()}
}

// choosesTheItemSchool grava a escola do item.
func choosesTheItemSchool(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	return s.plays.ChooseItemSchool(r.Context(), item, chi.URLParam(r, "escola"))
}
