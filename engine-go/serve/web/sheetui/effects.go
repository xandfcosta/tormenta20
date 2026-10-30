package sheetui

import (
	"encoding/json"
	"sort"
	"strconv"
	"t20engine/app/character"

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
	// Circle é o círculo da magia ("2º"), para o filtro; a condição não tem.
	Circle string
}

type stanceRow struct {
	Flag string
	Name string
	// Paid é "2 PM" — o que foi cobrado para entrar. Ele fica visível porque
	// encerrar não devolve nada, e a pessoa merece saber o que gastou.
	Paid string
	// Bonus é "+3" — o que a postura está DANDO, com os degraus pagos somados à
	// base. Vazio na postura que não move número nenhum.
	Bonus   string
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
	// Dice é o dado que o interruptor liga ("+1d12"), quando liga um — o chip
	// das Ações o escreve ao lado do nome.
	Dice    string
	Command string
	// Targets são os alvos CRUS dos modificadores que esta linha liga, e eles
	// existem para quem ROTEIA: a superfície de Ações põe o interruptor sobre o
	// número que ele muda, e o `Modifiers` acima já é texto de tela.
	Targets []engine.ModifierTarget
	// Degrees é a escolha de DEGRAU, quando o poder vende uma (p53). Nula no
	// situacional comum, que é a maioria.
	Degrees []degreeOption
	// DegreesOf é o NOME DO PODER que vende o degrau, e ele existe para o
	// rótulo acessível do seletor. O `Label` da linha é a nota da CONDIÇÃO —
	// "contra o alvo do Duelo — 2 PM, até o fim da cena" —, e "Degraus de
	// contra o alvo do Duelo…" é uma frase que ninguém lê. Visto na tela.
	DegreesOf string
}

// degreeOption é um degrau que o nível autoriza comprar, com o que ele custa e
// quanto ele dá.
//
// O PREÇO aparece porque o app NÃO cobra o PM deste gesto: ele informa, e quem
// paga é a mesa — a mesma divisa de todo situacional, e a razão de o Baluarte
// não ter virado um botão de "Usar" (decisão do dono, ALE-454).
type degreeOption struct {
	Steps int
	// Label é o bônus TOTAL com este degrau — "+6".
	Label string
	// Pm é o preço inteiro, base mais degraus — "3 PM".
	Pm string
	// Chosen marca o degrau que está pago agora.
	Chosen bool
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
	panel.Situational, _ = situationalRowsOf(offered, active, dto)
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
		// O BÔNUS EM CURSO, e não a contagem de degraus. Os dois saem como "+N" e
		// são números DIFERENTES — "+1" de um degrau pago é uma Fúria +3 —, então
		// as duas grafias na mesma tela eram um conceito com duas caras: aqui
		// "Fúria +1", no contador de entrar "+3", e no ataque o +3 somado.
		if bonus := book.StanceBonusInEffect(s.Flag, int(s.Steps)); bonus > 0 && book.StanceScales(s.Flag) {
			row.Bonus = book.WithSign(bonus)
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
			Circle:  circleName(m.Circle),
		})
	}
	return options
}

// situationalRowsOf desenha uma linha por interruptor.
//
// A PARTIÇÃO — quem compartilha flag vira um interruptor só, e as posturas
// ficam de fora — é do `character.SituationalGroupsOf`, e não daqui: o caso de
// uso RECUSA a chave que não estiver nela, e esta tela desenha exatamente o que
// ele aceita. Duas implementações divergiriam, e a divergência apareceria como
// a tela escondendo o que o servidor deixa passar (ALE-387).
func situationalRowsOf(
	offered []engine.ConditionalEffect, active map[string]bool, dto sheet.CharacterDTO,
) ([]situationalRow, []alwaysOnRow) {
	rows := []situationalRow{}
	for _, g := range character.SituationalGroupsOf(offered) {
		head := g.Members[0]
		row := situationalRow{
			Key: g.Key, Label: conditionalLabel(head), Source: head.Source,
			Folded:  len(g.Members) > 1,
			Active:  active[g.Key],
			Command: g.Key,
		}
		for _, c := range g.Members {
			// O DADO é o valor do condicional que rola — a Marca da Presa, o
			// encanto contra construtos —, e o `Amount` dele é zero: escrito
			// pelo número, a linha dizia "+0".
			value := book.WithSign(c.Amount)
			if c.Dice != "" {
				value = "+" + c.Dice
				row.Dice = value
			}
			row.Modifiers = append(row.Modifiers, breakdownRow{
				Label: targetLabel(c.Target), Value: value,
			})
			row.Targets = append(row.Targets, c.Target)
		}
		row.Degrees, row.DegreesOf = degreeOptionsFor(g.Key, head, dto)
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

// degreeOptionsFor são os degraus que o NÍVEL autoriza comprar neste
// situacional, ou nada quando o poder dele não vende degrau.
//
// O ZERO entra na lista, e ele é o bônus de base: o cavaleiro que gastou só o
// PM da entrada tem +2, e uma lista que começasse no primeiro degrau não teria
// como dizer isso. O teto sai do nível NA CLASSE, como o da postura.
//
// O JÁ PAGO é lido do efeito ativo com a chave do poder — o mesmo lugar onde o
// gesto o grava. Ler o número de volta em vez de guardá-lo numa segunda coluna
// mantém uma verdade só.
func degreeOptionsFor(
	key string, head engine.ConditionalEffect, dto sheet.CharacterDTO,
) ([]degreeOption, string) {
	offer, sold := offerBehind(key, head)
	if !sold {
		return nil, ""
	}
	base := baseAmountOf(offer)
	max := offer.Degrees.MaxDegrees(character.ClassPowerLevel(dto, offer.PowerID))
	if max <= 0 {
		return nil, ""
	}
	chosen := degreesPaidFor(dto, offer, base)
	options := make([]degreeOption, 0, max+1)
	for steps := 0; steps <= max; steps++ {
		options = append(options, degreeOption{
			Steps:  steps,
			Label:  book.WithSign(offer.Degrees.Amount(base, steps)),
			Pm:     strconv.Itoa(offer.Degrees.Cost(steps)) + " PM",
			Chosen: steps == chosen,
		})
	}
	return options, offer.PowerName
}

// offerBehind acha a oferta de degrau atrás da chave do grupo ou da flag.
func offerBehind(key string, head engine.ConditionalEffect) (book.DegreeOffer, bool) {
	offers := book.DegreesByFlag()
	if offer, found := offers[head.Flag]; found && head.Flag != "" {
		return offer, true
	}
	for flag, offer := range offers {
		if engine.FlagGroupID(flag) == key {
			return offer, true
		}
	}
	return book.DegreeOffer{}, false
}

// baseAmountOf é o bônus que o verbete concede sem degrau nenhum.
func baseAmountOf(offer book.DegreeOffer) int {
	for _, m := range offer.Base {
		return m.Amount
	}
	return 0
}

// degreesPaidFor lê do efeito ativo quantos degraus estão pagos.
//
// Zero quando não há efeito: o situacional ligado sem degrau vale a base, e é
// esse o estado de quem gastou só o PM de entrada.
func degreesPaidFor(dto sheet.CharacterDTO, offer book.DegreeOffer, base int) int {
	step := offer.Degrees.StepAmount
	if step <= 0 {
		return 0
	}
	for _, effect := range dto.ActiveEffects {
		if effect.CatalogID != offer.PowerID {
			continue
		}
		var mods []engine.Modifier
		if json.Unmarshal([]byte(effect.Modifiers), &mods) != nil || len(mods) == 0 {
			return 0
		}
		return max(0, (mods[0].Amount-base)/step)
	}
	return 0
}
