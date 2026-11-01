package sheetui

import (
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/search"

	"github.com/a-h/templ"
)

// A ABA MAGIAS DA FICHA (ALE-473). O dado é o do painel
// (`spellbookPanel`); daqui sai só o que a folha e a linha leem. Como nos
// Poderes, a linha não mostra o texto da regra: tocar no nome abre a folha da
// magia, e o Esquecer mora no pé dela, longe do Conjurar (decisão do dono).

// spellCasting é o topo da aba: a CD e o atributo, ditos uma vez. A CD é a do
// Combate (`spellDcTotal`, com o que os itens somam), e o atributo é o da
// classe que a dá — pelo caminho, no Arcanista. CD zero é quem não conjura.
type spellCasting struct {
	CD        int
	Attribute string
}

func spellCastingOf(computed engine.ComputedSheet) spellCasting {
	if !isCaster(computed) {
		return spellCasting{}
	}
	return spellCasting{CD: spellDcTotal(computed), Attribute: book.AttributeLabel(computed.BestSpellAttribute)}
}

// spellCircleGroup é um círculo do grimório, já filtrado pela busca.
type spellCircleGroup struct {
	Title  string
	Spells []learnedSpellRow
}

// spellCircleGroups agrupa o grimório por círculo, pela busca e pelo filtro.
// As linhas já chegam em ordem de círculo e nome (`learnedSpellRowsOf`), então
// o grupo fecha quando o círculo muda — e círculo sem magia não tem grupo.
func spellCircleGroups(v View) []spellCircleGroup {
	var groups []spellCircleGroup
	for _, spell := range v.Spells.Learned {
		if !grimoireShows(v, spell.Name, spell.Circle) {
			continue
		}
		title := spell.Circle + " círculo"
		if len(groups) == 0 || groups[len(groups)-1].Title != title {
			groups = append(groups, spellCircleGroup{Title: title})
		}
		last := &groups[len(groups)-1]
		last.Spells = append(last.Spells, spell)
	}
	return groups
}

// grantedSpellLine é a concedida com o índice que dá o `id` da folha: a
// concedida não tem id de catálogo na linha, e o índice é o da lista inteira
// para a folha não trocar de dono quando a busca esconde as de cima.
type grantedSpellLine struct {
	Index int
	Spell grantedSpellRow
}

func grantedSpellLines(v View) []grantedSpellLine {
	var lines []grantedSpellLine
	for i, spell := range v.Spells.Granted {
		if grimoireShows(v, spell.Name, spell.Circle) {
			lines = append(lines, grantedSpellLine{Index: i, Spell: spell})
		}
	}
	return lines
}

// grimoireShows diz se a magia passa pela busca e pelo filtro de círculo.
func grimoireShows(v View, name, circle string) bool {
	if v.GrimoireCircle != "" && circle != v.GrimoireCircle {
		return false
	}
	if strings.TrimSpace(v.GrimoireSearch) == "" {
		return true
	}
	return strings.Contains(search.Fold(name), search.Fold(v.GrimoireSearch))
}

// grimoireCircleOptions são as opções do filtro: "Todos" e os círculos que o
// grimório e as concedidas TÊM, em ordem — círculo vazio não vira opção.
func grimoireCircleOptions(v View) []filterOption {
	has := map[string]bool{}
	for _, spell := range v.Spells.Learned {
		has[spell.Circle] = true
	}
	for _, spell := range v.Spells.Granted {
		has[spell.Circle] = true
	}
	options := []filterOption{{Value: "", Label: "Todos", Active: v.GrimoireCircle == ""}}
	for circle := 1; circle <= 5; circle++ {
		if name := circleName(circle); has[name] {
			options = append(options, filterOption{Value: name, Label: name, Active: v.GrimoireCircle == name})
		}
	}
	return options
}

func spellSheetID(spell learnedSpellRow) string {
	return "spell-" + spell.Command
}

func grantedSheetID(index int) string {
	return "granted-spell-" + strconv.Itoa(index)
}

// spellLineMeta é a nota da linha: a execução e o alcance, como a mesa fala, e
// o aviso de quem prepara.
//
//	spellLineMeta(row, false) // "Padrão · médio", na Bola de Fogo
func spellLineMeta(spell learnedSpellRow, requiresPrep bool) string {
	meta := book.CastingName(spell.Execution) + " · " + strings.ToLower(book.RangeName(spell.Range))
	if requiresPrep && !spell.Prepared {
		meta += " · não preparada"
	}
	return meta
}

// spellSheetMeta é a linha de cima da folha: círculo, execução, alcance,
// duração e página.
//
//	spellSheetMeta(row) // "2º círculo · Padrão · médio · instantânea · p180", na Bola de Fogo
func spellSheetMeta(spell learnedSpellRow) string {
	parts := []string{spell.Circle + " círculo", spellLineMeta(spell, false)}
	if duration := book.DurationName(spell.Duration); duration != "" {
		parts = append(parts, duration)
	}
	if spell.Page > 0 {
		parts = append(parts, "p"+strconv.Itoa(spell.Page))
	}
	return strings.Join(parts, " · ")
}

// spellSheetFoot é o pé da folha: o Esquecer e o Conjurar com o custo.
func spellSheetFoot(v View, spell learnedSpellRow) templ.Component {
	return spellSheetFootBody(v, spell)
}

// castOpensSheet diz se o Conjurar da linha abre a folha: só quando há
// aprimoramento a escolher. Sem escolha, ele age direto, como o Usar do poder.
func castOpensSheet(spell learnedSpellRow) bool {
	for _, augment := range spell.Augments {
		if !augment.Locked {
			return true
		}
	}
	return false
}

// augmentsReset zera as seis pilhas: quem abre uma folha limpa, ou a escolha
// de uma magia reapareceria na próxima.
const augmentsReset = "$augment0 = 0; $augment1 = 0; $augment2 = 0; $augment3 = 0; $augment4 = 0; $augment5 = 0; "

// circleFilterOptions são os círculos do catálogo, para o Aprender: não há
// magia de círculo zero — o truque é aprimoramento (ver o GLOSSARY).
var circleFilterOptions = []filterOption{
	{Value: "", Label: "Todos"}, {Value: "1", Label: "1º"}, {Value: "2", Label: "2º"},
	{Value: "3", Label: "3º"}, {Value: "4", Label: "4º"}, {Value: "5", Label: "5º"},
}
