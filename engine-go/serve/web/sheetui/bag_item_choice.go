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

// A ESCOLHA QUE O ITEM PEDE, e são duas. A ESCOLA: o tomo hermético dá +2 na CD "das magias arcanas dessa escola" (p160). Quem
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

// O DEUS QUE O SÍMBOLO PEDE: o +1 em resistência é do "símbolo sagrado de um deus do qual é devoto" (p157).
// O deus da ficha vem primeiro, sozinho, e os outros ficam atrás de "Outro deus" (decisão do dono).

// itemGod é a escolha na folha: o deus do símbolo, o deus da ficha, e os outros deuses.
type itemGod struct {
	Chosen, ChosenName string
	// Own é o deus de quem a ficha é devota, vazio sem devoção. Ele não entra em `Others`.
	Own, OwnName string
	Others       []book.God
}

// Devout diz se o +1 do símbolo vale: o deus dele é o da ficha.
func (g itemGod) Devout() bool { return g.Chosen != "" && g.Chosen == g.Own }

// godOnTheItemSheet é a escolha do item que pede deus, ou nil.
func godOnTheItemSheet(item sheet.ItemDTO, dto sheet.CharacterDTO) *itemGod {
	entry := catalogItem(item)
	if entry == nil || !slices.ContainsFunc(entry.Modifiers, func(m engine.Modifier) bool { return m.FromChoice == "god" }) {
		return nil
	}
	out := &itemGod{Chosen: item.Choice}
	if dto.God != nil {
		out.Own = *dto.God
	}
	for _, god := range book.CatalogGods() {
		if god.ID == item.Choice {
			out.ChosenName = god.Name
		}
		if god.ID == out.Own {
			out.OwnName = god.Name
			continue
		}
		out.Others = append(out.Others, god)
	}
	return out
}

// choosesTheItemGod grava o deus do símbolo.
func choosesTheItemGod(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	return s.plays.ChooseItemGod(r.Context(), item, chi.URLParam(r, "deus"))
}
