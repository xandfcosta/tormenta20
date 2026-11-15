package sheetui

import (
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/search"
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
	if entry := catalogItem(item); entry == nil || !entry.AsksFor("school") {
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
	if entry := catalogItem(item); entry == nil || !entry.AsksFor("god") {
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

// A MAGIA QUE A MELHORIA PEDE: o Harmonizado diz "escolha uma magia. Seu custo diminui em –1 PM" (p165). A
// escolha é da MELHORIA, e não do item, porque um tomo hermético harmonizado pede a escola e a magia. A lista
// é o livro inteiro, por busca (decisão do dono): com a busca vazia, a folha mostra só a escolhida, e a busca
// a lista também, marcada, para "bola" não dizer "nenhuma" com a Bola de Fogo escolhida.

// improvementSpell é a escolha na folha: a melhoria que pede, a magia escolhida, e o que a busca achou.
type improvementSpell struct {
	Improvement        string
	Chosen, ChosenName string
	Search             string
	Found              []book.Spell
}

// spellOnTheItemSheet é a escolha da melhoria aplicada que pede magia, ou nil. `query` é a busca da folha.
func spellOnTheItemSheet(item sheet.ItemDTO, query string) *improvementSpell {
	applied := savedImprovements(item.Improvements)
	at := slices.IndexFunc(applied, func(id string) bool {
		entry := book.ItemByID(id)
		return entry != nil && entry.AsksFor("spell")
	})
	if at < 0 {
		return nil
	}
	out := &improvementSpell{Improvement: applied[at], Chosen: item.ImprovementChoices[applied[at]], Search: query}
	if out.Chosen != "" {
		out.ChosenName = book.SpellName(out.Chosen)
	}
	if strings.TrimSpace(query) == "" {
		return out
	}
	for _, spell := range book.Catalogs().Spells {
		if strings.Contains(search.Fold(spell.Name), search.Fold(query)) {
			out.Found = append(out.Found, spell)
		}
	}
	return out
}

// chosenCutRow troca, na linha da melhoria aplicada, o "custo da magia escolhida" (ou da habilidade) pelo
// `effect` que diz o nome dela.
func chosenCutRow(rows []overlayRow, improvement string, effect string) {
	entry := book.ItemByID(improvement)
	if entry == nil {
		return
	}
	for i := range rows {
		if rows[i].Name == entry.Name {
			rows[i].Effect = effect
		}
	}
}

// improvementSpellPath é o endereço que escolhe `spell` para a melhoria da folha.
func improvementSpellPath(s itemSheet, spell string) string {
	return "/itens/" + s.Command + "/melhorias/" + s.Spell.Improvement + "/magia/" + spell
}

// choosesTheImprovementSpell grava a magia da melhoria.
func choosesTheImprovementSpell(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	return s.plays.ChooseImprovementSpell(r.Context(), item, chi.URLParam(r, "melhoria"), chi.URLParam(r, "magia"))
}

// A HABILIDADE QUE A ARMA HARMONIZADA PEDE: "escolha uma habilidade ativada ao se fazer um ataque […] e que
// custe pontos de mana" (p165). São os poderes do livro que se pagam no golpe — a lista é o livro e não o que a
// ficha tem, como a magia do Harmonizado — e o gasto dos encantos DESTA arma (decisão do dono), que só sai com
// ela. São poucos, e a folha os mostra todos, sem busca.

// improvementAbility é a escolha na folha: a melhoria que pede, a habilidade escolhida e as que se oferecem.
type improvementAbility struct {
	Improvement        string
	Chosen, ChosenName string
	Options            []abilityOption
}

// abilityOption é uma habilidade que a Harmonizada pode cortar: o id do poder ou do encanto, e o rótulo.
type abilityOption struct {
	ID, Label, Name string
}

// abilityOnTheItemSheet é a escolha da melhoria aplicada que pede habilidade, ou nil.
func abilityOnTheItemSheet(item sheet.ItemDTO) *improvementAbility {
	applied := savedImprovements(item.Improvements)
	at := slices.IndexFunc(applied, func(id string) bool {
		entry := book.ItemByID(id)
		return entry != nil && entry.AsksFor("boost")
	})
	if at < 0 {
		return nil
	}
	out := &improvementAbility{Improvement: applied[at], Chosen: item.ImprovementChoices[applied[at]]}
	for _, power := range book.PaidOnAttackPowers() {
		out.Options = append(out.Options, abilityOption{ID: power.ID, Name: power.Name, Label: power.Name + " (" + power.ClassName + ")"})
	}
	for _, enchant := range item.Enchants {
		if entry := book.ItemByID(enchant); entry != nil && entry.Boost != nil {
			out.Options = append(out.Options, abilityOption{ID: enchant, Name: entry.Name, Label: entry.Name + " (encanto desta arma)"})
		}
	}
	for _, option := range out.Options {
		if option.ID == out.Chosen {
			out.ChosenName = option.Name
		}
	}
	return out
}

// improvementAbilityPath é o endereço que escolhe `power` para a melhoria da folha.
func improvementAbilityPath(s itemSheet, power string) string {
	return "/itens/" + s.Command + "/melhorias/" + s.Ability.Improvement + "/habilidade/" + power
}

// choosesTheImprovementAbility grava a habilidade da melhoria.
func choosesTheImprovementAbility(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	return s.plays.ChooseImprovementAbility(r.Context(), item, chi.URLParam(r, "melhoria"), chi.URLParam(r, "habilidade"))
}

// A MARCA "FEITA POR MIM": o Ferreiro e o Cano Raiado valem na "arma feita por você mesmo" (p69), e o app não sabe
// quem fabricou o item (decisão do dono: a marca é do jogador). Ela só aparece na arma de quem tem algo que a
// lê — o motor diz, pelo escopo da marca —, porque para os outros seria um interruptor que não muda nada.

// madeByOwnerOnTheItemSheet diz se a folha oferece a marca e se ela está ligada; nil quando não oferece.
func madeByOwnerOnTheItemSheet(dto sheet.CharacterDTO, item sheet.ItemDTO) *bool {
	if entry := catalogItem(item); entry == nil || entry.Weapon == nil || !readsTheMadeByOwnerMark(dto) {
		return nil
	}
	made := item.MadeByOwner
	return &made
}

// readsTheMadeByOwnerMark diz se algum efeito da ficha soma no escopo da marca.
func readsTheMadeByOwnerMark(dto sheet.CharacterDTO) bool {
	if dto.Ruleset == nil {
		return false
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return false
	}
	effects := dto.Ruleset.EffectsOf(ec, sheet.ToStringSet(dto.Conditionals))
	for _, target := range []engine.ModifierTarget{
		{K: "damageStep", Scope: engine.MadeByOwnerMeleeScope},
		{K: "critRange", Scope: engine.MadeByOwnerFiredScope},
	} {
		if engine.StatFor(effects, target).Total != 0 {
			return true
		}
	}
	return false
}

// madeByOwnerPath é o endereço que liga e desliga a marca.
func madeByOwnerPath(s itemSheet) string { return "/itens/" + s.Command + "/feita-por-mim" }

// togglesMadeByOwner liga ou desliga a marca do item.
func togglesMadeByOwner(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	return s.plays.ToggleItemMadeByOwner(r.Context(), item)
}
