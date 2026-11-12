package sheetui

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"t20engine/app/character"
	"t20engine/domain/book"
	"t20engine/domain/catalog"
	"t20engine/domain/engine"
	"t20engine/domain/search"
	"t20engine/domain/sheet"
)

// A aba MAGIAS como dado.
//
// O grimório do personagem: só o que ele APRENDEU, cada magia com preparar,
// esquecer e conjurar. "Aprender" abre o catálogo inteiro do Capítulo 4 com
// filtros de círculo, escola e busca.
//
// # As CONCEDIDAS aparecem mesmo para quem não conjura
//
// Um poder pode ensinar uma magia (Totem Espiritual, p42), e um bárbaro com
// Totem tem de ver a magia dele. Elas não estão no grimório — não se aprendem
// nem se esquecem —, então são um bloco próprio, acima.
//
// # O SERVIDOR é a autoridade do custo, e a tela só antecipa
//
// Não há pré-validação de PM no cliente: o `@post` responde em milissegundos e a
// recusa vem com a frase certa. O que a tela mostra é uma PRÉVIA do total,
// somada por expressão do Datastar sobre números que o servidor já mandou — e
// ela nunca decide nada.

// spellbookPanel é a aba Magias pronta para desenhar.
type spellbookPanel struct {
	Learned []learnedSpellRow
	Granted []grantedSpellRow
	// Catalog são as magias que o personagem AINDA não sabe, já filtradas.
	Catalog []catalogSpellRow
	// IsCaster diz se há classe conjuradora. Sem ela não há o que aprender — mas
	// as concedidas continuam aparecendo.
	IsCaster bool
	// RequiresPrep é Clérigo, Druida e o Arcanista do caminho `mago`: para eles
	// a magia precisa estar PREPARADA para ser conjurada.
	RequiresPrep bool
	// CastableCircle é o maior círculo que este personagem alcança, e é o que
	// TRANCA os aprimoramentos de círculo alto.
	CastableCircle int
	PmCurrent      int64
	// Ritual diz que a próxima magia é ritual, e o diálogo escreve a conta dele (p131).
	Ritual bool
	// ShapeSilence é a frase da forma que não conjura (p62), vazia fora dela ou com a Magia Natural: o
	// Conjurar fica desabilitado, e o servidor recusa de todo jeito (`character.Plays.Cast`).
	ShapeSilence string
	Search       string
	Circle       string
	School       string
	Schools      []filterOption
}

type filterOption struct {
	Value  string
	Label  string
	Active bool
}

type learnedSpellRow struct {
	ID       string
	Name     string
	Circle   string
	School   string
	BasePm   int
	Prepared bool
	// CD é a Classe de Dificuldade dos testes contra esta magia, pelo
	// atributo-chave da classe que a concede.
	CD        string
	Augments  []augmentRow
	Execution string
	Range     string
	Duration  string
	// ExecutionWritten e RangeWritten são o texto do livro. O `Execution` é o custo de ação.
	ExecutionWritten string
	RangeWritten     string
	DurationWritten  string
	// Targeting é o Alvo, a Área ou o Efeito, nas palavras do livro.
	Targeting string
	// Resistance é o que se testa para resistir, vazio na magia sem resistência.
	Resistance string
	Effect     string
	Page       int
	Command    string
	// CostCut avisa que este custo vai sair cortado, e por quem — ver
	// `costCutNote`. Vazio quando nada corta.
	CostCut string
	// SpellCut é quem corta o custo SÓ DESTA magia, além do que a caixa "Custo PM" do Combate já mostra:
	// o medalhão de prata nas de alcance pessoal (p160).
	SpellCut string
	// FocusCut diz que o Foco em Magia tira 1 PM desta magia — ver
	// `spellFocusNote`. Vazio quando ela não foi escolhida.
	FocusCut string
}

// grantedSpellRow é a magia que um PODER ensinou.
type grantedSpellRow struct {
	Name   string
	Circle string
	Source string
	Effect string
	Page   int
}

type catalogSpellRow struct {
	ID      string
	Name    string
	Circle  string
	School  string
	Effect  string
	Page    int
	Command string
}

// spellbookPanelOf monta a aba.
func (s Scene) spellbookPanelOf(dto sheet.CharacterDTO, search, circle, school string) spellbookPanel {
	panel := spellbookPanel{
		IsCaster:       len(casterClassesOf(dto)) > 0,
		RequiresPrep:   sheet.RequiresPreparation(dto.Classes, dto.ClassChoices),
		CastableCircle: sheet.HighestCastableCircle(dto.Classes, 0),
		PmCurrent:      dto.MpCurrent,
		Ritual:         character.CastsAsRitual(dto),
		Search:         search,
		Circle:         circle,
		School:         school,
		Schools:        schoolOptions(school),
		Granted:        grantedSpellRowsOf(dto),
	}
	if ec, err := sheet.EngineCharacterFrom(dto); err == nil && !engine.CastsInTheShape(ec) {
		panel.ShapeSilence = character.ShapeSilence
	}
	panel.Learned = learnedSpellRowsOf(dto, panel.CastableCircle)
	// Quem não conjura não aprende, e por isso não paga o catálogo: a lista
	// inteira do Capítulo 4 viajaria em toda cena da ficha de um guerreiro.
	if panel.IsCaster {
		panel.Catalog = catalogSpellRowsOf(dto, search, circle, school)
	}
	return panel
}

// casterClassesOf são as classes do personagem que conjuram.
func casterClassesOf(dto sheet.CharacterDTO) []string {
	names := []string{}
	for _, c := range dto.Classes {
		if sheet.IsCasterClass(c.ClassName) {
			names = append(names, c.ClassName)
		}
	}
	return names
}

// learnedSpellRowsOf é o grimório, por círculo e depois por nome.
//
// A FICHA É CALCULADA UMA VEZ para a lista inteira, e cada linha pergunta a ela o que vale para a magia
// (`ForSpell`): o grimório de um conjurador de nível alto tem dezenas de magias.
func learnedSpellRowsOf(dto sheet.CharacterDTO, castable int) []learnedSpellRow {
	computed, computedOK := spellSheetOf(dto)
	rows := []learnedSpellRow{}
	for _, learned := range dto.Spells {
		spell, known := catalog.LookupSpell(learned.CatalogSpellID)
		if !known {
			continue
		}
		fromBook := spellOfBook(learned.CatalogSpellID)
		tradition := sheet.SpellTradition(dto.Classes, sheet.CastingClasses(dto, dto.Classes, fromBook.School, fromBook.Classes))
		at := engine.SpellAt{Tradition: tradition, School: spell.School, Range: spell.Range, Spell: learned.CatalogSpellID}
		rows = append(rows, learnedSpellRow{
			ID: learned.CatalogSpellID, Name: fromBook.Name,
			Circle: circleName(fromBook.Circle), School: schoolName(fromBook.School),
			BasePm: sheet.SpellBasePmCost[fromBook.Circle], Prepared: learned.Prepared,
			CD:        spellCdOf(dto, computed, computedOK, spell, at),
			Augments:  augmentRowsOf(dto, learned.CatalogSpellID, spell, castable),
			Execution: fromBook.Execution, Range: fromBook.Range, Duration: fromBook.Duration,
			ExecutionWritten: book.SpellExecutionWritten(fromBook), RangeWritten: book.SpellRangeWritten(fromBook),
			DurationWritten: book.SpellDurationWritten(fromBook),
			Targeting:       fromBook.Targeting,
			Resistance:      book.SpellResistanceWritten(fromBook),
			Effect:          fromBook.BaseEffect, Page: fromBook.BookPage,
			Command:  learned.CatalogSpellID,
			CostCut:  costCutNote(computed, computedOK, tradition),
			FocusCut: spellFocusNote(dto, learned.CatalogSpellID),
			SpellCut: onlyHereCutNote(computed, computedOK, at),
		})
	}
	sort.SliceStable(rows, func(a, b int) bool {
		if rows[a].Circle != rows[b].Circle {
			return rows[a].Circle < rows[b].Circle
		}
		return rows[a].Name < rows[b].Name
	})
	return rows
}

// spellCdOf é a CD dos testes contra a magia, pelo atributo-chave da classe,
// com os bônus de CD de item e de poder (a Flauta Mística, p158).
//
// Ela vem do MOTOR, pelo mapa por atributo que a caixa "CD Magia" do Combate já
// usa: uma segunda conta aqui daria dois números para a mesma pergunta. O bônus
// é o que alcança a magia (`ForSpell`), com a tradição da classe que lança: a varinha arcana
// sobe a CD das arcanas e não a das divinas (p160), e o tomo hermético só a da escola dele.
func spellCdOf(dto sheet.CharacterDTO, computed engine.ComputedSheet, ok bool, spell catalog.Spell, at engine.SpellAt) string {
	if !ok {
		return "—"
	}
	best := 0
	for _, class := range dto.Classes {
		prog, casts := book.SpellProgressions()[class.ClassName]
		if !casts || !sheet.CastsTheSpell(dto, class.ClassName, spell.School, spell.Classes) {
			continue
		}
		byThisClass := at
		byThisClass.Tradition = prog.List
		if cd := computed.SpellCdByAttribute[prog.Attribute] + computed.ForSpell("spellDC", byThisClass).Total; cd > best {
			best = cd
		}
	}
	if best == 0 {
		return "—"
	}
	return strconv.Itoa(best)
}

// spellOfBook acha a magia no acervo já ordenado.
func spellOfBook(id string) book.Spell {
	if spell, found := book.SpellByID(id); found {
		return spell
	}
	return book.Spell{ID: id, Name: id}
}

// catalogSpellRowsOf são as magias que ainda dá para aprender, filtradas.
func catalogSpellRowsOf(dto sheet.CharacterDTO, search, circle, school string) []catalogSpellRow {
	known := map[string]bool{}
	for _, s := range dto.Spells {
		known[s.CatalogSpellID] = true
	}
	rows := []catalogSpellRow{}
	for _, m := range book.Catalogs().Spells {
		if known[m.ID] || !passesFilter(m, search, circle, school) {
			continue
		}
		rows = append(rows, catalogSpellRow{
			ID: m.ID, Name: m.Name, Circle: circleName(m.Circle), School: schoolName(m.School),
			Effect: m.BaseEffect, Page: m.BookPage, Command: m.ID,
		})
	}
	return rows
}

func passesFilter(m book.Spell, query, circle, school string) bool {
	if circle != "" && strconv.Itoa(m.Circle) != circle {
		return false
	}
	if school != "" && m.School != school {
		return false
	}
	if strings.TrimSpace(query) == "" {
		return true
	}
	return strings.Contains(search.Fold(m.Name), search.Fold(query))
}

// grantedSpellRowsOf são as magias que um PODER ensinou.
//
// Elas não moram no grimório: não se aprendem nem se esquecem, e some com o
// poder. Aparecem mesmo para quem não tem classe conjuradora — um bárbaro com
// Totem Espiritual (p42) tem de ver a magia dele.
func grantedSpellRowsOf(dto sheet.CharacterDTO) []grantedSpellRow {
	choices := map[string][]string{}
	if err := json.Unmarshal([]byte(dto.PowerChoices), &choices); err != nil {
		return nil
	}
	byName := map[string]book.Spell{}
	for _, m := range book.Catalogs().Spells {
		byName[m.Name] = m
	}
	rows := []grantedSpellRow{}
	for _, power := range book.PowersThatTeachSpells() {
		picks, chose := choices[power.ID]
		if !chose {
			continue
		}
		for _, pick := range picks {
			spell, exists := byName[power.Options[pick]]
			if !exists {
				continue
			}
			rows = append(rows, grantedSpellRow{
				Name: spell.Name, Circle: circleName(spell.Circle),
				Source: power.Name, Effect: spell.BaseEffect, Page: spell.BookPage,
			})
		}
	}
	return rows
}

// ── o que a TELA precisa escrever ────────────────────────────────────────────

// aprendidasEscrito é "3 aprendidas", com o singular certo.
func aprendidasEscrito(n int) string {
	if n == 1 {
		return "1 aprendida"
	}
	return strconv.Itoa(n) + " aprendidas"
}

// circleName é "Truque" ou "3º", como a mesa fala.
func circleName(circle int) string {
	if circle == 0 {
		return "Truque"
	}
	return strconv.Itoa(circle) + "º"
}

var schoolNames = map[string]string{
	"abjuracao": "Abjuração", "adivinhacao": "Adivinhação", "convocacao": "Convocação",
	"encantamento": "Encantamento", "evocacao": "Evocação", "ilusao": "Ilusão",
	"necromancia": "Necromancia", "transmutacao": "Transmutação",
}

func schoolName(id string) string {
	if name, found := schoolNames[id]; found {
		return name
	}
	return id
}

func schoolOptions(active string) []filterOption {
	ids := make([]string, 0, len(schoolNames))
	for id := range schoolNames {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	options := []filterOption{{Value: "", Label: "Todas as escolas", Active: active == ""}}
	for _, id := range ids {
		options = append(options, filterOption{Value: id, Label: schoolNames[id], Active: active == id})
	}
	return options
}

// spellSheetOf é a ficha com o opt-in do jogador, sem as cartas de arma que o `sheetForPanels` monta junto:
// a CD da magia não as lê.
func spellSheetOf(dto sheet.CharacterDTO) (engine.ComputedSheet, bool) {
	if dto.Ruleset == nil {
		return engine.ComputedSheet{}, false
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return engine.ComputedSheet{}, false
	}
	return dto.Ruleset.ComputeSheet(ec, sheet.ToStringSet(dto.Conditionals)), true
}
