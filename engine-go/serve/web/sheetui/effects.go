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
	if s.deps.Catalogs() == nil {
		return effectsPanelFor(dto, nil, nil)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return effectsPanelFor(dto, nil, nil)
	}
	oferecidos := engine.ComputeItemEffects(s.deps.Catalogs().ActiveItemsFor(ec)).Conditional
	return effectsPanelFor(dto, oferecidos, s.deps.Catalogs().ComputeEquippedFlags(ec.Items))
}

// effectsPanelFor monta a aba inteira.
func effectsPanelFor(dto sheet.CharacterDTO, offered []engine.ConditionalEffect, flags []engine.EquippedFlag) effectsPanel {
	ativos := sheet.ToStringSet(dto.Conditionals)
	panel := effectsPanel{
		Conditions:       conditionRowsOf(dto),
		ConditionOptions: conditionOptionsFor(dto),
		Stances:          stanceRowsOf(dto),
		Applied:          appliedEffectRowsOf(dto),
		BuffOptions:      buffOptions(),
	}
	panel.Situational, _ = situationalRowsOf(offered, ativos)
	panel.AlwaysOn = alwaysOnRowsOf(flags)
	panel.Count = len(panel.Conditions) + len(panel.Applied) + len(panel.Stances) + activeCountOf(panel.Situational)
	return panel
}

func activeCountOf(linhas []situationalRow) int {
	n := 0
	for _, l := range linhas {
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
	porID := map[string]book.Condition{}
	for _, c := range book.Catalogs().Conditions {
		porID[c.ID] = c
	}
	linhas := []conditionRow{}
	for _, id := range sheet.UnmarshalStrings(dto.ActiveConditions) {
		c, conhecida := porID[id]
		if !conhecida {
			continue
		}
		linhas = append(linhas, conditionRow{
			ID: c.ID, Name: c.Name, Effect: c.Description, Page: c.BookPage, Command: c.ID,
		})
	}
	sort.SliceStable(linhas, func(a, b int) bool { return linhas[a].Name < linhas[b].Name })
	return linhas
}

func conditionOptionsFor(dto sheet.CharacterDTO) []pickerOption {
	ligadas := sheet.ToStringSet(sheet.UnmarshalStrings(dto.ActiveConditions))
	opcoes := []pickerOption{}
	for _, c := range book.Catalogs().Conditions {
		if ligadas[c.ID] {
			continue
		}
		opcoes = append(opcoes, pickerOption{ID: c.ID, Label: c.Name, Detail: c.Description, Command: c.ID})
	}
	return opcoes
}

// stanceRowsOf são as posturas em curso.
func stanceRowsOf(dto sheet.CharacterDTO) []stanceRow {
	doLivro := book.StancesFromCatalog()
	linhas := []stanceRow{}
	for _, s := range dto.Stances {
		nome := s.Flag
		if posture, conhecida := doLivro[s.Flag]; conhecida {
			nome = posture.Name
		}
		linha := stanceRow{Flag: s.Flag, Name: nome, Command: s.Flag}
		if s.PmPaid > 0 {
			linha.Paid = strconv.FormatInt(s.PmPaid, 10) + " PM"
		}
		if s.Steps > 0 {
			linha.Steps = "+" + strconv.FormatInt(s.Steps, 10)
		}
		linhas = append(linhas, linha)
	}
	return linhas
}

// appliedEffectRowsOf são os consumíveis e as magias de bônus em curso.
func appliedEffectRowsOf(dto sheet.CharacterDTO) []appliedEffectRow {
	linhas := []appliedEffectRow{}
	for _, e := range dto.ActiveEffects {
		linhas = append(linhas, appliedEffectRow{
			ID:        e.ID,
			Name:      effectDisplayName(e.CatalogID),
			Scope:     scopeLabel(e.Scope),
			Modifiers: modifierRowsOf(e.Modifiers),
			Command:   strconv.FormatInt(e.ID, 10),
		})
	}
	return linhas
}

// modifierRowsOf traduz o blob de modificadores em linhas legíveis.
func modifierRowsOf(bruto string) []breakdownRow {
	var mods []engine.Modifier
	if err := json.Unmarshal([]byte(bruto), &mods); err != nil {
		return nil
	}
	linhas := make([]breakdownRow, 0, len(mods))
	for _, m := range mods {
		linhas = append(linhas, breakdownRow{
			Label: targetLabel(m.Target), Value: book.WithSign(m.Amount), Note: m.Note,
		})
	}
	return linhas
}

// buffOptions são as magias com efeito aplicável.
func buffOptions() []pickerOption {
	opcoes := []pickerOption{}
	for _, m := range book.Catalogs().Spells {
		spell, conhecida := catalog.LookupSpell(m.ID)
		if !conhecida || spell.Buff == nil {
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
		opcoes = append(opcoes, pickerOption{
			ID:      m.ID,
			Label:   m.Name,
			Detail:  circleLabel(m.Circle) + " · " + scopeLabel(scope),
			Command: m.ID,
		})
	}
	return opcoes
}

// situationalRowsOf agrupa os condicionais que o motor oferece.
//
// # Quem compartilha FLAG vira UM interruptor
//
// Um item caseiro com três modificadores é uma coisa só na mesa; como três
// linhas, a pessoa deixaria metade do efeito ligado. As POSTURAS ficam de fora:
// o interruptor delas mora nos Poderes, porque entrar custa PM.
func situationalRowsOf(offered []engine.ConditionalEffect, ativos map[string]bool) ([]situationalRow, []alwaysOnRow) {
	posturas := book.StancesFromCatalog()
	porFlag := map[string][]engine.ConditionalEffect{}
	ordem := []string{}
	soltos := []engine.ConditionalEffect{}
	for _, c := range offered {
		if c.Flag == "" {
			soltos = append(soltos, c)
			continue
		}
		if _, ehPostura := posturas[c.Flag]; ehPostura {
			continue
		}
		if _, visto := porFlag[c.Flag]; !visto {
			ordem = append(ordem, c.Flag)
		}
		porFlag[c.Flag] = append(porFlag[c.Flag], c)
	}

	linhas := []situationalRow{}
	for _, c := range soltos {
		id := engine.ConditionalID(c)
		linhas = append(linhas, situationalRow{
			Key: id, Label: conditionalLabel(c), Source: c.Source, Active: ativos[id],
			Modifiers: []breakdownRow{{Label: targetLabel(c.Target), Value: book.WithSign(c.Amount)}},
			Command:   id,
		})
	}
	for _, flag := range ordem {
		grupo := porFlag[flag]
		linha := situationalRow{
			Key: engine.ConditionalID(grupo[0]), Label: conditionalLabel(grupo[0]),
			Source: grupo[0].Source, Folded: len(grupo) > 1,
			Active:  ativos[engine.ConditionalID(grupo[0])],
			Command: engine.ConditionalID(grupo[0]),
		}
		for _, c := range grupo {
			linha.Modifiers = append(linha.Modifiers, breakdownRow{
				Label: targetLabel(c.Target), Value: book.WithSign(c.Amount),
			})
		}
		linhas = append(linhas, linha)
	}
	return linhas, nil
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
	linhas := []alwaysOnRow{}
	for _, f := range flags {
		rotulo, conhecida := itemFlagLabel[f.Flag]
		if !conhecida {
			rotulo = f.Flag
		}
		linhas = append(linhas, alwaysOnRow{Label: rotulo, Source: f.Source})
	}
	return linhas
}
