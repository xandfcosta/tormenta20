package sheetui

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"t20engine/app/character"
	"t20engine/domain/book"
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
	// Memorization é "2 de 3 memorizadas" com a regra do estudo, só para quem tem o mago (p37), e vazia
	// para os outros: o clérigo e o druida conjuram o que conhecem.
	Memorization string
	PmCurrent    int64
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
	ID        string
	Name      string
	Circle    string
	School    string
	BasePm    int
	Memorized bool
	// NeedsMemorizing é a magia que a ficha só lança memorizada (`SpellOnTheSheet.NeedsMemorizing`): só ela
	// tem o interruptor.
	NeedsMemorizing bool
	// MemorizeLock é a recusa de memorizá-la agora (`Caster.MemorizeRefusal`), vazia quando se pode ou quando
	// ela já está memorizada: no limite, o interruptor tranca e a folha escreve por quê — o `title` sozinho
	// não chega ao celular.
	MemorizeLock string
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
	// o medalhão de prata nas de alcance pessoal (p160), o Foco em Magia na escolhida (p131).
	SpellCut string
	// AboveTheCircle é a magia gravada acima do círculo que a ficha alcança, que fica no grimório; a linha a
	// marca. OffTheList é o aviso de `SpellOnTheSheet.OffTheList`.
	AboveTheCircle string
	// CastLock é a recusa de conjurar sem aprimoramento (`SpellOnTheSheet.CastLock`): o Conjurar das abas
	// Magias e Ações tranca com ela, a mesma frase que o servidor diria. LockedBySustained diz que a razão é
	// outra magia sustentada de pé (p227), para a linha a marcar.
	CastLock          string
	LockedBySustained bool
	// Rolls é o dado que a magia rola ao ser conjurada, uma linha por parcela (ver GLOSSARY, **dado da magia**),
	// com o campo do total rolado na mão.
	Rolls      []spellRollLine
	OffTheList string
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
	// AboveTheCircle tranca o Aprender, e aparece em vez de sumir, como o aprimoramento de círculo alto:
	// quem ainda não alcança merece saber que a magia existe. OffTheList só avisa.
	AboveTheCircle string
	OffTheList     string
}

// spellbookPanelOf monta a aba.
func (s Scene) spellbookPanelOf(dto sheet.CharacterDTO, search, circle, school string) spellbookPanel {
	panel := spellbookPanel{
		IsCaster:     len(casterClassesOf(dto)) > 0,
		Memorization: memorizationOf(dto),
		PmCurrent:    dto.MpCurrent,
		Ritual:       sheet.CasterOf(dto).CastsAsRitual(),
		Search:       search,
		Circle:       circle,
		School:       school,
		Schools:      schoolOptions(school),
		Granted:      grantedSpellRowsOf(dto),
	}
	if ec, err := sheet.EngineCharacterFrom(dto); err == nil && !engine.CastsInTheShape(ec) {
		panel.ShapeSilence = character.ShapeSilence
	}
	panel.Learned = learnedSpellRowsOf(dto)
	// Quem não conjura não aprende, e por isso não paga o catálogo: a lista
	// inteira do Capítulo 4 viajaria em toda cena da ficha de um guerreiro.
	if panel.IsCaster {
		panel.Catalog = catalogSpellRowsOf(dto, search, circle, school)
	}
	return panel
}

// memorizationOf é "2 de 3 memorizadas" com a regra do estudo (p37), vazia sem mago na ficha. A hora de
// estudo uma vez por dia não se confere: quando o dia passou é o mestre quem diz.
func memorizationOf(dto sheet.CharacterDTO) string {
	caster := sheet.CasterOf(dto)
	if !caster.Memorizes() {
		return ""
	}
	memorized, limit := caster.Memorization()
	return strconv.Itoa(memorized) + " de " + strconv.Itoa(limit) +
		" memorizadas · trocar pede uma hora de estudo, uma vez por dia (p37)"
}

// memorizeLockOf é a recusa de memorizar a magia agora, vazia para a que não se memoriza e para a já memorizada,
// que sempre se desmemoriza.
func memorizeLockOf(caster sheet.Caster, onSheet sheet.SpellOnTheSheet, learned sheet.SpellDTO) string {
	if !onSheet.NeedsMemorizing || learned.Prepared {
		return ""
	}
	return caster.MemorizeRefusal(learned.CatalogSpellID)
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
func learnedSpellRowsOf(dto sheet.CharacterDTO) []learnedSpellRow {
	computed, computedOK := spellSheetOf(dto)
	caster := sheet.CasterOf(dto)
	rows := []learnedSpellRow{}
	for _, learned := range dto.Spells {
		onSheet := caster.Spell(learned.CatalogSpellID)
		if !onSheet.Known {
			continue
		}
		spell, at := onSheet.Spell, onSheet.At()
		rows = append(rows, learnedSpellRow{
			ID: learned.CatalogSpellID, Name: spell.Name,
			Circle: circleName(spell.Circle), School: schoolName(spell.School),
			BasePm: sheet.SpellBasePmCost[spell.Circle], Memorized: learned.Prepared,
			NeedsMemorizing: onSheet.NeedsMemorizing,
			MemorizeLock:    memorizeLockOf(caster, onSheet, learned),
			CD:              spellCdOf(computed, computedOK, onSheet),
			Augments:        augmentRowsOf(spell, onSheet),
			Execution:       spell.Execution, Range: spell.Range, Duration: spell.Duration,
			ExecutionWritten: book.SpellExecutionWritten(spell), RangeWritten: book.SpellRangeWritten(spell),
			DurationWritten: book.SpellDurationWritten(spell),
			Targeting:       spell.Targeting,
			Resistance:      book.SpellResistanceWritten(spell),
			Effect:          spell.BaseEffect, Page: spell.BookPage,
			Command:           learned.CatalogSpellID,
			CostCut:           costCutNote(computed, computedOK, onSheet.Tradition),
			SpellCut:          onlyHereCutNote(computed, computedOK, at),
			AboveTheCircle:    onSheet.AboveTheCircle,
			CastLock:          onSheet.CastLock,
			LockedBySustained: onSheet.LockedBySustained(),
			Rolls:             spellRollLinesOf(spell),
			OffTheList:        onSheet.OffTheList,
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

// spellCdOf é a CD da magia escrita na linha, ou "—" sem ela (`SpellOnTheSheet.DC`).
func spellCdOf(computed engine.ComputedSheet, ok bool, spell sheet.SpellOnTheSheet) string {
	if !ok {
		return "—"
	}
	if dc, has := spell.DC(computed); has {
		return strconv.Itoa(dc)
	}
	return "—"
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
	caster := sheet.CasterOf(dto)
	rows := []catalogSpellRow{}
	for _, m := range book.Catalogs().Spells {
		if known[m.ID] || !passesFilter(m, search, circle, school) {
			continue
		}
		onSheet := caster.Spell(m.ID)
		rows = append(rows, catalogSpellRow{
			ID: m.ID, Name: m.Name, Circle: circleName(m.Circle), School: schoolName(m.School),
			Effect: m.BaseEffect, Page: m.BookPage, Command: m.ID,
			AboveTheCircle: onSheet.AboveTheCircle, OffTheList: onSheet.OffTheList,
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
