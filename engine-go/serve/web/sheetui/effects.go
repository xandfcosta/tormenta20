package sheetui

import (
	"encoding/json"
	"sort"
	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/catalog"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// A aba EFEITOS como dado.
//
// É tudo que está mexendo nos números do personagem AGORA, em quatro blocos que
// diferem por QUEM é dono do estado:
//
//  1. CONDIÇÕES do livro (p394-395) — coluna `activeConditions`. Elas MOVEM os
//     números; uma condição que fosse só crachá não seria uma condição.
//  2. POSTURAS em curso, com o que cada uma custou. O interruptor de LIGAR mora
//     nos Poderes, onde o PM é cobrado; aqui elas só se leem e se encerram.
//  3. EFEITOS ATIVOS — consumível usado e magia de bônus aplicada, com escopo de
//     cena ou dia.
//  4. SITUAÇÃO — o opt-in de contexto (terreno, tipo de alvo, item caseiro).
//
// As posturas saem do CATÁLOGO e não de uma tabela escrita aqui: elas estão no
// `activations.json` como `"kind": "stance"`, e a FLAG de cada uma sai do poder
// de mesmo id, lendo o `condition.flag` dos modificadores dele.

// effectsPanel é a aba Efeitos pronta para desenhar.
type effectsPanel struct {
	Conditions []conditionRow
	// ConditionOptions são as que ainda cabem — o catálogo menos as ativas.
	ConditionOptions []pickerOption
	Stances          []stanceRow
	Applied          []appliedEffectRow
	// BuffOptions são as magias que têm efeito aplicável, para o diálogo.
	BuffOptions []pickerOption
	Situational []situationalRow
	// AlwaysOn são as flags de item equipado que não têm interruptor — a
	// "Fadiga ao dormir" da armadura pesada. Elas aparecem em modo de leitura
	// para o bloco não contradizer o aviso que a mesma flag produz no cabeçalho.
	AlwaysOn []alwaysOnRow
	// Count é o que o crachá da aba mostra, e ele conta COISAS e não modificadores
	// do motor: a Fúria entra como oito modificadores e é UMA coisa na mesa.
	Count int
}

type conditionRow struct {
	ID      string
	Name    string
	Effect  string
	Page    int
	Command string
}

type pickerOption struct {
	ID    string
	Label string
	// Detail é a segunda linha do item — o efeito da condição, o círculo da
	// magia. Sem ela o jogador escolhe por nome decorado.
	Detail  string
	Command string
}

type stanceRow struct {
	Flag string
	Name string
	// Paid é "2 PM" — o que foi cobrado para entrar. Ele fica visível porque
	// encerrar não devolve nada, e a pessoa merece saber o que gastou.
	Paid    string
	Steps   string
	Command string
}

type appliedEffectRow struct {
	ID     int64
	Name   string
	Scope  string
	Detail string
	// Modifiers são as linhas de "o que isto faz", que é o que separa um efeito
	// aplicado de um nome numa lista.
	Modifiers []breakdownRow
	Command   string
}

type situationalRow struct {
	// Key é o que o comando manda: a chave do CONDICIONAL, ou a flag quando o
	// grupo tem uma.
	Key    string
	Label  string
	Source string
	Active bool
	// Folded diz que a linha liga MAIS DE UM modificador de uma vez. Um item
	// caseiro com três modificadores é UM interruptor, senão a pessoa deixa
	// metade do efeito ligado.
	Folded    bool
	Modifiers []breakdownRow
	Command   string
}

type alwaysOnRow struct {
	Label  string
	Source string
}

// ── as posturas, lidas do catálogo ───────────────────────────────────────────

// book.Stance é a postura de cada flag: `furia` → Fúria, 2 PM, p40.
// ── a montagem ───────────────────────────────────────────────────────────────

// effectsPanelOf computa a aba Efeitos de um personagem.
//
// Ela pergunta ao motor QUAIS condicionais existem — a lista de opt-ins que
// aquela ficha oferece, dado o que ela veste e sabe — e não o que está ligado:
// o que está ligado é o `dto.Conditionals`, que é do jogador. Sem catálogo
// primado a aba mostra o que não depende do motor (condições, posturas e
// efeitos aplicados) e perde a Situação, que é derivada.
func (s Scene) effectsPanelOf(dto sheet.CharacterDTO) effectsPanel {
	if dto.Ruleset == nil {
		return effectsPanelFor(dto, nil, nil)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return effectsPanelFor(dto, nil, nil)
	}
	offered := engine.ComputeItemEffects(dto.Ruleset.ActiveItemsFor(ec)).Conditional
	return effectsPanelFor(dto, offered, dto.Ruleset.ComputeEquippedFlags(ec.Items))
}

// effectsPanelFor monta a aba inteira.
func effectsPanelFor(dto sheet.CharacterDTO, offered []engine.ConditionalEffect, flags []engine.EquippedFlag) effectsPanel {
	active := sheet.ToStringSet(dto.Conditionals)
	panel := effectsPanel{
		Conditions:       conditionRowsOf(dto),
		ConditionOptions: conditionOptionsFor(dto),
		Stances:          stanceRowsOf(dto),
		Applied:          appliedEffectRowsOf(dto),
		BuffOptions:      buffOptions(),
	}
	panel.Situational, _ = situationalRowsOf(offered, active)
	panel.AlwaysOn = alwaysOnRowsOf(flags)
	panel.Count = len(panel.Conditions) + len(panel.Applied) + len(panel.Stances) + activeCountOf(panel.Situational)
	return panel
}

func activeCountOf(rows []situationalRow) int {
	n := 0
	for _, l := range rows {
		if l.Active {
			n++
		}
	}
	return n
}

// conditionRowsOf são as condições do livro que estão ligadas.
//
// Id desconhecido é DESCARTADO: o catálogo é a autoridade sobre o que é uma
// condição, e um blob velho não pode injetar uma condição fantasma na ficha.
func conditionRowsOf(dto sheet.CharacterDTO) []conditionRow {
	byID := map[string]book.Condition{}
	for _, c := range book.Catalogs().Conditions {
		byID[c.ID] = c
	}
	rows := []conditionRow{}
	for _, id := range sheet.UnmarshalStrings(dto.ActiveConditions) {
		c, known := byID[id]
		if !known {
			continue
		}
		rows = append(rows, conditionRow{
			ID: c.ID, Name: c.Name, Effect: c.Description, Page: c.BookPage, Command: c.ID,
		})
	}
	sort.SliceStable(rows, func(a, b int) bool { return rows[a].Name < rows[b].Name })
	return rows
}

func conditionOptionsFor(dto sheet.CharacterDTO) []pickerOption {
	on := sheet.ToStringSet(sheet.UnmarshalStrings(dto.ActiveConditions))
	options := []pickerOption{}
	for _, c := range book.Catalogs().Conditions {
		if on[c.ID] {
			continue
		}
		options = append(options, pickerOption{ID: c.ID, Label: c.Name, Detail: c.Description, Command: c.ID})
	}
	return options
}

// stanceRowsOf são as posturas em curso.
func stanceRowsOf(dto sheet.CharacterDTO) []stanceRow {
	fromBook := book.StancesFromCatalog()
	rows := []stanceRow{}
	for _, s := range dto.Stances {
		name := s.Flag
		if posture, known := fromBook[s.Flag]; known {
			name = posture.Name
		}
		row := stanceRow{Flag: s.Flag, Name: name, Command: s.Flag}
		if s.PmPaid > 0 {
			row.Paid = strconv.FormatInt(s.PmPaid, 10) + " PM"
		}
		if s.Steps > 0 {
			row.Steps = "+" + strconv.FormatInt(s.Steps, 10)
		}
		rows = append(rows, row)
	}
	return rows
}

// appliedEffectRowsOf são os consumíveis e as magias de bônus em curso.
func appliedEffectRowsOf(dto sheet.CharacterDTO) []appliedEffectRow {
	rows := []appliedEffectRow{}
	for _, e := range dto.ActiveEffects {
		rows = append(rows, appliedEffectRow{
			ID:        e.ID,
			Name:      effectDisplayName(e.CatalogID),
			Scope:     scopeLabel(e.Scope),
			Modifiers: modifierRowsOf(e.Modifiers),
			Command:   strconv.FormatInt(e.ID, 10),
		})
	}
	return rows
}

// modifierRowsOf traduz o blob de modificadores em linhas legíveis.
func modifierRowsOf(raw string) []breakdownRow {
	var mods []engine.Modifier
	if err := json.Unmarshal([]byte(raw), &mods); err != nil {
		return nil
	}
	rows := make([]breakdownRow, 0, len(mods))
	for _, m := range mods {
		rows = append(rows, breakdownRow{
			Label: targetLabel(m.Target), Value: book.WithSign(m.Amount), Note: m.Note,
		})
	}
	return rows
}

// buffOptions são as magias com efeito aplicável.
func buffOptions() []pickerOption {
	options := []pickerOption{}
	for _, m := range book.Catalogs().Spells {
		spell, known := catalog.LookupSpell(m.ID)
		if !known || spell.Buff == nil {
			continue
		}
		// A DURAÇÃO DA MAGIA MANDA, aqui como na gravação. Ler o
		// `defaultScope` cru mostrava a SEGUNDA transcrição — a que divergia em
		// oito magias, e que o Escudo da Fé deixou de ter (ALE-365).
		scope, err := engine.EffectScope(spell.Duration, spell.DurationNote, spell.Buff.DefaultScope)
		if err != nil {
			// Magia cujo efeito não sabe quando acaba não é aplicável: some da
			// lista em vez de virar uma opção que falha ao clicar. Que ela não
			// exista é o que o `TestEveryBuffLastsAsLongAsItsSpell` cobra.
			continue
		}
		options = append(options, pickerOption{
			ID:      m.ID,
			Label:   m.Name,
			Detail:  circleLabel(m.Circle) + " · " + scopeLabel(scope),
			Command: m.ID,
		})
	}
	return options
}

// situationalRowsOf agrupa os condicionais que o motor oferece.
//
// # Quem compartilha FLAG vira UM interruptor
//
// Um item caseiro com três modificadores é uma coisa só na mesa; como três
// linhas, a pessoa deixaria metade do efeito ligado. As POSTURAS ficam de fora:
// o interruptor delas mora nos Poderes, porque entrar custa PM.
func situationalRowsOf(offered []engine.ConditionalEffect, active map[string]bool) ([]situationalRow, []alwaysOnRow) {
	stances := book.StancesFromCatalog()
	byFlag := map[string][]engine.ConditionalEffect{}
	order := []string{}
	loose := []engine.ConditionalEffect{}
	for _, c := range offered {
		if c.Flag == "" {
			loose = append(loose, c)
			continue
		}
		if _, isStance := stances[c.Flag]; isStance {
			continue
		}
		if _, seen := byFlag[c.Flag]; !seen {
			order = append(order, c.Flag)
		}
		byFlag[c.Flag] = append(byFlag[c.Flag], c)
	}

	rows := []situationalRow{}
	for _, c := range loose {
		id := engine.ConditionalID(c)
		rows = append(rows, situationalRow{
			Key: id, Label: conditionalLabel(c), Source: c.Source, Active: active[id],
			Modifiers: []breakdownRow{{Label: targetLabel(c.Target), Value: book.WithSign(c.Amount)}},
			Command:   id,
		})
	}
	for _, flag := range order {
		group := byFlag[flag]
		row := situationalRow{
			Key: engine.ConditionalID(group[0]), Label: conditionalLabel(group[0]),
			Source: group[0].Source, Folded: len(group) > 1,
			Active:  active[engine.ConditionalID(group[0])],
			Command: engine.ConditionalID(group[0]),
		}
		for _, c := range group {
			row.Modifiers = append(row.Modifiers, breakdownRow{
				Label: targetLabel(c.Target), Value: book.WithSign(c.Amount),
			})
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// alwaysOnRowsOf são as flags de item equipado que NÃO têm interruptor.
//
// A "Fadiga ao dormir" da armadura pesada é o caso: ela está ligada porque a
// armadura está vestida, e não porque alguém a escolheu. Mostrá-las em modo de
// leitura é o que impede este bloco de contradizer o aviso que a MESMA flag
// produz no cabeçalho da ficha — sem elas, a pessoa lê "nenhum efeito" embaixo
// de um aviso que existe por causa de um.
//
// Quem as CALCULA é o motor (`ComputeEquippedFlags`), e não uma varredura
// escrita aqui: ele resolve as condições de uso (vestido, empunhado) e sabe
// quais modificadores do item contam.
func alwaysOnRowsOf(flags []engine.EquippedFlag) []alwaysOnRow {
	rows := []alwaysOnRow{}
	for _, f := range flags {
		label, known := itemFlagLabel[f.Flag]
		if !known {
			label = f.Flag
		}
		rows = append(rows, alwaysOnRow{Label: label, Source: f.Source})
	}
	return rows
}
