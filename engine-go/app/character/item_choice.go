package character

import (
	"context"
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
)

// ChooseItemSchool grava a escola que o item pede a quem o tem: o tomo hermético dá +2 na CD "das magias
// arcanas dessa escola" (p160). Quem escolhe é o jogador, e trocar é gravar de novo (decisão do dono).
//
//	plays.ChooseItemSchool(ctx, item, "evocacao")
func (p Plays) ChooseItemSchool(ctx context.Context, item sqlcgen.GetItemRow, school string) error {
	var entry *book.Item
	if item.Catalogid.Valid {
		entry = book.ItemByID(item.Catalogid.String)
	}
	if entry == nil || !slices.ContainsFunc(entry.Modifiers, func(m engine.Modifier) bool { return m.FromChoice == "school" }) {
		return fmt.Errorf("%s não pede escola; quem pede é o tomo hermético (p160)", item.Name)
	}
	if !slices.ContainsFunc(book.SpellSchools(), func(s book.SpellSchool) bool { return s.ID == school }) {
		return fmt.Errorf("%q não é uma escola de magia; esperava uma das oito escolas, como \"evocacao\"", school)
	}
	if err := p.queries.SetItemChoice(ctx, sqlcgen.SetItemChoiceParams{Itemid: item.ID, Choice: school}); err != nil {
		return fmt.Errorf("gravar a escola do item %d: %w", item.ID, err)
	}
	return nil
}

// ChooseItemGod grava o deus do símbolo sagrado: o +1 em resistência é do "símbolo sagrado de um deus do qual
// é devoto" (p157), e o −1 PM do Símbolo abençoado é de quem "for devoto do deus" (p345). Quem escolhe é o
// jogador, e trocar é gravar de novo, como a escola do tomo.
//
//	plays.ChooseItemGod(ctx, item, "wynna")
func (p Plays) ChooseItemGod(ctx context.Context, item sqlcgen.GetItemRow, god string) error {
	var entry *book.Item
	if item.Catalogid.Valid {
		entry = book.ItemByID(item.Catalogid.String)
	}
	if entry == nil || !slices.ContainsFunc(entry.Modifiers, func(m engine.Modifier) bool { return m.FromChoice == "god" }) {
		return fmt.Errorf("%s não pede deus; quem pede é o símbolo sagrado (p157) e o abençoado (p345)", item.Name)
	}
	if !slices.ContainsFunc(book.CatalogGods(), func(g book.God) bool { return g.ID == god }) {
		return fmt.Errorf("%q não é um deus do Panteão; esperava o id de um dos vinte, como \"wynna\"", god)
	}
	if err := p.queries.SetItemChoice(ctx, sqlcgen.SetItemChoiceParams{Itemid: item.ID, Choice: god}); err != nil {
		return fmt.Errorf("gravar o deus do item %d: %w", item.ID, err)
	}
	return nil
}
