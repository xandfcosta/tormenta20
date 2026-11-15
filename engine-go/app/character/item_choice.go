package character

import (
	"context"
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// ChooseItemSchool grava a escola que o item pede a quem o tem: o tomo hermético dá +2 na CD "das magias
// arcanas dessa escola" (p160). Quem escolhe é o jogador, e trocar é gravar de novo (decisão do dono).
//
//	plays.ChooseItemSchool(ctx, item, "evocacao")
func (p Plays) ChooseItemSchool(ctx context.Context, item sqlcgen.GetItemRow, school string) error {
	if entry := itemEntryOf(item); entry == nil || !entry.AsksFor("school") {
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
	if entry := itemEntryOf(item); entry == nil || !entry.AsksFor("god") {
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

// ChooseImprovementSpell grava a magia que a melhoria pede: o Harmonizado diz "escolha uma magia. Seu custo
// diminui em –1 PM" (p165). A lista é o livro inteiro, e não só o grimório (decisão do dono): o livro não
// prende a escolha ao que a ficha sabe. Trocar é gravar de novo; o ritual de um dia e T$ 100 que a troca
// pede é com o mestre.
//
//	plays.ChooseImprovementSpell(ctx, item, "melhoria-harmonizado-esoterico", "bola-de-fogo")
func (p Plays) ChooseImprovementSpell(ctx context.Context, item sqlcgen.GetItemRow, improvement, spell string) error {
	entry := book.ItemByID(improvement)
	if entry == nil || !entry.AsksFor("spell") {
		return fmt.Errorf("%q não pede magia; quem pede é o Harmonizado (p165)", improvement)
	}
	if !slices.Contains(sheet.UnmarshalStrings(item.Improvements), improvement) {
		return fmt.Errorf("%s não tem a melhoria %s; aplique-a pelas Melhorias antes de escolher a magia",
			item.Name, entry.Name)
	}
	if _, found := book.SpellByID(spell); !found {
		return fmt.Errorf("%q não é uma magia do livro; esperava o id de uma delas, como \"bola-de-fogo\"", spell)
	}
	err := p.queries.SetImprovementChoice(ctx, sqlcgen.SetImprovementChoiceParams{
		Itemid: item.ID, Improvementid: improvement, Choice: spell,
	})
	if err != nil {
		return fmt.Errorf("gravar a magia da melhoria %s do item %d: %w", improvement, item.ID, err)
	}
	return nil
}

// ChooseImprovementAbility grava a habilidade que a arma Harmonizada corta: "Escolha uma habilidade ativada ao
// se fazer um ataque […] e que custe pontos de mana" (p165). Valem os poderes do livro que se pagam no golpe —
// a lista é a do livro, e não só o que a ficha tem, pela mesma razão da magia do Harmonizado — e o gasto de
// um encanto DESTA arma (decisão do dono), que é o único que sai com ela.
//
//	plays.ChooseImprovementAbility(ctx, item, "melhoria-harmonizada-arma", "class.paladino.golpe-divino-1d8")
func (p Plays) ChooseImprovementAbility(ctx context.Context, item sqlcgen.GetItemRow, improvement, power string) error {
	entry := book.ItemByID(improvement)
	if entry == nil || !entry.AsksFor("boost") {
		return fmt.Errorf("%q não pede habilidade; quem pede é a Harmonizada (p165)", improvement)
	}
	if !slices.Contains(sheet.UnmarshalStrings(item.Improvements), improvement) {
		return fmt.Errorf("%s não tem a melhoria %s; aplique-a pelas Melhorias antes de escolher a habilidade",
			item.Name, entry.Name)
	}
	paid, err := p.paidOnAttackWith(ctx, item, power)
	if err != nil {
		return err
	}
	if !paid {
		return fmt.Errorf("%q não é uma habilidade que se paga no ataque com %s; esperava um poder como %q, ou o "+
			"encanto desta arma que se paga no golpe", power, item.Name, "class.paladino.golpe-divino-1d8")
	}
	err = p.queries.SetImprovementChoice(ctx, sqlcgen.SetImprovementChoiceParams{
		Itemid: item.ID, Improvementid: improvement, Choice: power,
	})
	if err != nil {
		return fmt.Errorf("gravar a habilidade da melhoria %s do item %d: %w", improvement, item.ID, err)
	}
	return nil
}

// paidOnAttackWith diz se `power` se paga no golpe com o item: um poder de classe com impulso, ou um encanto
// do item com gasto (p335-336).
func (p Plays) paidOnAttackWith(ctx context.Context, item sqlcgen.GetItemRow, power string) (bool, error) {
	if found, ok := book.ClassPowers()[power]; ok {
		return found.Boost != nil, nil
	}
	enchants, err := p.queries.EnchantsOfItem(ctx, item.ID)
	if err != nil {
		return false, fmt.Errorf("ler os encantos do item %d: %w", item.ID, err)
	}
	entry := book.ItemByID(power)
	return slices.Contains(enchants, power) && entry != nil && entry.Boost != nil, nil
}

// ToggleItemMadeByOwner liga ou desliga a marca "feita por mim" do item: o app não sabe quem fabricou a arma, e
// o Ferreiro e o Cano Raiado valem na "arma feita por você mesmo" (p69). Quem diz é o jogador (decisão do dono).
//
//	plays.ToggleItemMadeByOwner(ctx, espada)
func (p Plays) ToggleItemMadeByOwner(ctx context.Context, item sqlcgen.GetItemRow) error {
	marked, err := p.queries.ItemsMadeByOwnerOfCharacter(ctx, item.Characterid)
	if err != nil {
		return fmt.Errorf("ler as marcas de feita por mim da ficha %d: %w", item.Characterid, err)
	}
	if slices.Contains(marked, item.ID) {
		if err := p.queries.UnmarkItemMadeByOwner(ctx, item.ID); err != nil {
			return fmt.Errorf("tirar a marca de feita por mim do item %d: %w", item.ID, err)
		}
		return nil
	}
	if err := p.queries.MarkItemMadeByOwner(ctx, item.ID); err != nil {
		return fmt.Errorf("marcar o item %d como feito por quem o tem: %w", item.ID, err)
	}
	return nil
}

// itemEntryOf é o verbete do livro de um item da mochila, ou nil no item próprio.
func itemEntryOf(item sqlcgen.GetItemRow) *book.Item {
	if !item.Catalogid.Valid {
		return nil
	}
	return book.ItemByID(item.Catalogid.String)
}
