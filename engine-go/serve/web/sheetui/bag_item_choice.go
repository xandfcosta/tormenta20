package sheetui

import (
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/book"
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

// spellCutRow troca, na linha da melhoria aplicada, o "custo da magia escolhida" pelo nome dela.
func spellCutRow(rows []overlayRow, improvement string, chosen string) {
	entry := book.ItemByID(improvement)
	if entry == nil || chosen == "" {
		return
	}
	for i := range rows {
		if rows[i].Name == entry.Name {
			rows[i].Effect = "−1 PM no custo de " + book.SpellName(chosen)
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
