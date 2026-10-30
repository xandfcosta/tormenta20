package sheetui

import (
	"strconv"
	"strings"

	"t20engine/domain/engine"
)

// O QUE DÁ PARA FAZER, agrupado pelo que a vez CUSTA (T20 p233).
//
//	"No seu turno, você pode fazer uma ação padrão e uma ação de movimento, em
//	 qualquer ordem… Você também pode abrir mão das duas para fazer uma ação
//	 completa. Você também pode executar qualquer quantidade de ações livres e
//	 reações." (p233)
//
// ESTE PAINEL NÃO CALCULA NADA. Ele REAGRUPA o que o Combate, os Poderes e as
// Magias já computaram — e isso é a decisão de desenho, não economia: o ataque
// da aba Combate e o ataque daqui têm de ser o MESMO número, e dois caminhos de
// conta divergem no dia em que um deles passar um condicional diferente. É a
// mesma razão que faz o `Load` chamar o motor uma vez e repartir.
//
// POR QUE AGRUPAR PELO CUSTO e não por categoria (armas, poderes, magias): a
// pergunta do jogador na vez dele não é "que tipo de coisa eu tenho?", é "o que
// eu ainda posso fazer?". Agrupado por categoria, descobrir que já se gastou a
// padrão exige ler as três listas.

// actionsPanel é a superfície Ações pronta para desenhar.
type actionsPanel struct {
	Groups []actionGroup
	// Passives é a CONTAGEM das passivas, nunca as linhas delas: não há gesto a
	// fazer com uma passiva, e uma linha por passiva encheria a tela. Que ela
	// esteja nos números da ficha é OUTRA pergunta, e a resposta é "nem sempre"
	// — quem diz quais entram é o `cmd/audit/coverage` (ALE-482).
	Passives int
}

// actionGroup é um custo do livro com o que cabe nele. Grupo sem linha não é
// desenhado: um cabeçalho "Ação completa" sobre o nada é ruído com cara de
// seção.
type actionGroup struct {
	Title string
	// OnTurn marca o que GASTA a vez — padrão, movimento, completa —, e é o que
	// apaga fora da vez na sessão paralela (decisão do dono). A livre e a
	// reação não: a reação é justamente o que se faz fora da vez (p233).
	OnTurn bool
	// Note é o que o título não diz — "abre mão das duas", "não gastam a vez".
	Note string
	Rows []actionRow
	// Chips são os interruptores que mudam ALGUM número deste grupo.
	//
	// DO GRUPO E NÃO DA LINHA, e isto foi decidido OLHANDO a tela: por linha, a
	// Fúria de um bárbaro aparecia três vezes — no Atacar, na Manobra e na
	// Investida —, e com ela ligada eram três botões "Encerrar Fúria" idênticos
	// empilhados. No grupo ela aparece uma vez, logo ACIMA dos números que ela
	// muda, que é a adjacência que o desenho queria; e ela ainda aparece no
	// grupo da ação completa, porque lá ela muda outro número.
	Chips []actionChip
}

// actionRow é uma coisa que dá para fazer.
//
// Os quatro campos de texto já vêm ESCRITOS porque quem os escreve é o painel de
// origem: o `Attack` de um cartão de arma já é "+8", e reformatá-lo aqui seria a
// segunda grafia de um número só.
type actionRow struct {
	Name string
	// Detail é a linha pequena sob o nome — "1d8+5 · 20/×3 · Luta".
	Detail string
	// Value é o número que se rola, quando existe. Vazio na linha que não rola
	// nada (a postura que se liga, o poder de efeito fixo).
	Value string
	// Cost é o preço EM PM e o limite, nunca a ação: a ação já é o grupo, e
	// repeti-la em toda linha gastaria a largura que a 390px é cara.
	Cost string
	// Can e Why são a decisão de usar AGORA, como na aba Poderes. A linha
	// recusada aparece apagada em vez de sumir: saber que a Fúria existe e que
	// falta PM é informação, e sumir seria o app escondendo a ficha de alguém.
	Can bool
	Why string
	// Live marca o que está EM CURSO — a postura ligada, a passiva de gatilho com
	// o gatilho no ar.
	Live bool
	// Moves são os NÚMEROS que esta linha mostra, no vocabulário dos alvos de
	// modificador: `attack`, `damage`, `expertise:Atletismo`. É por eles que o
	// chip sabe em que linha pousar — e é por isso que eles existem mesmo numa
	// linha sem chip nenhum.
	Moves []string
	// Rolls são os gestos de ROLAR que esta linha oferece. Vazio na linha que
	// não rola nada — Mover, um poder de efeito fixo, a Investida.
	Rolls []actionRoll
	// Weapon e Boosts são da linha de ATAQUE: o índice da arma e os impulsos de
	// PM que este golpe pode levar (ver `attack_boost.go`).
	Weapon int
	Boosts []attackBoost
}

// actionRoll é um gesto de rolar que sai de uma linha.
//
// É uma LISTA e não um campo porque a MANOBRA tem cinco (p234) e o ataque tem
// um. Um campo só obrigaria a manobra a virar cinco linhas, e a lista de ações
// padrão dobraria de tamanho para repetir o mesmo detalhe cinco vezes.
type actionRoll struct {
	// Label é o verbo no botão: "Rolar" quando a linha já se nomeia, e o nome da
	// manobra quando não — cinco botões escritos "Rolar" não dizem qual é qual.
	Label string
	// Path é o endereço RELATIVO à ficha. Quem monta o `@post` é o `sheetPost`,
	// que carrega a aba e a marca de embutida que a URL da página não leva.
	Path string
	// NeedsTarget diz que o gesto não acontece sem ALVO escolhido, e o botão
	// nasce desabilitado até haver um. O servidor recusa de todo jeito; isto é
	// para o botão não existir só para levar recusa.
	NeedsTarget bool
	// Command, quando escrito, substitui o `@post` do `Path`: é o toque que
	// ABRE o contador de degraus do poder. Before vem antes do `@post` — o
	// poder que vende degrau zera o contador antes de usar.
	Command string
	Before  string
	// Opens marca o gesto que abre uma escolha, para o `aria-haspopup`.
	Opens string
	// Dialog é o diálogo do kit que o gesto abre na sessão paralela, pelo
	// `commandfor` — o `Opens` é a sobreposição de hoje.
	Dialog string
}

// rollCommand é a expressão do toque num gesto da superfície.
func rollCommand(v View, roll actionRoll, after string) string {
	if roll.Command != "" {
		return roll.Command
	}
	return roll.Before + sheetPost(v, roll.Path) + after
}

// actionsPanelFrom monta a superfície a partir dos painéis JÁ computados.
//
// A ORDEM DOS GRUPOS é a do livro (p233): padrão, movimento, completa, e o que
// não gasta a vez por último. O custo negociado fecha a lista porque ele não é
// um custo — é a mesa decidindo um.
func actionsPanelFrom(
	id int64, computed engine.ComputedSheet,
	combat panelCombat, powers powersPanel, spells spellbookPanel, effects effectsPanel,
	boosts [][]attackBoost,
) actionsPanel {
	melee := expertiseOrZero(computed, "Luta", "strength").Total + computed.AttackAll.Total
	byCost := map[engine.ActionCost][]actionRow{
		engine.ActionStandard: standardRows(combat, melee, computed, boosts),
		engine.ActionMovement: {movementRow(computed)},
		engine.ActionFull:     fullRows(combat, computed, ownsPower(powers, engine.RecklessChargePowerID)),
	}
	panel := actionsPanel{}
	for _, row := range powers.Actions {
		cost := engine.ActionCost(row.Action)
		byCost[cost] = append(byCost[cost], rowFromPower(row))
	}
	for _, spell := range spells.Learned {
		cost := engine.ActionCost(spell.Execution)
		byCost[cost] = append(byCost[cost], rowFromSpell(spell))
	}
	for _, group := range groupOrder {
		var rows []actionRow
		for _, cost := range group.costs {
			rows = append(rows, byCost[cost]...)
		}
		if len(rows) > 0 {
			panel.Groups = append(panel.Groups,
				actionGroup{Title: group.title, Note: group.note, OnTurn: group.onTurn, Rows: rows})
		}
	}
	panel.Passives = len(powers.Passives)
	hangChips(&panel, chipsOffered(powers, effects.Situational))
	dropRowsTheChipAlreadySays(&panel)
	return panel
}

// groupOrder são os grupos na ordem da tela, e QUAIS custos caem em cada um.
//
// LIVRE E REAÇÃO dividem um grupo, e a razão é de largura e não de conceito: as
// duas são "não gasta a vez", e dois cabeçalhos para duas listas de uma linha
// custariam a 390px mais do que entregam. A distinção que o GLOSSARY.md defende
// — a livre é escolha consciente no seu turno, a reação vale fora dele — não se
// perde: ela vai escrita na linha, no `Cost`.
var groupOrder = []struct {
	title  string
	note   string
	onTurn bool
	costs  []engine.ActionCost
}{
	{"Ação padrão", "", true, []engine.ActionCost{engine.ActionStandard}},
	{"Ação de movimento", "", true, []engine.ActionCost{engine.ActionMovement}},
	{"Ação completa", "abre mão das duas", true, []engine.ActionCost{engine.ActionFull}},
	{"Livre e reação", "não gastam a vez", false,
		[]engine.ActionCost{engine.ActionFree, engine.ActionReaction}},
	{"Custo que a mesa decide", "o livro não fixa um", false,
		[]engine.ActionCost{engine.ActionVaries}},
}

// rowFromPower traduz uma linha da aba Poderes.
//
// O `Cost` perde a palavra da AÇÃO e fica só com o preço: "MOVIMENTO · 1 PM"
// vira "1 PM", porque a ação virou o grupo. O corte é feito no separador que o
// `writtenCost` escreve, e a postura — que nunca escreveu ação nenhuma — passa
// inteira.
func rowFromPower(power powerRow) actionRow {
	// SEM A DESCRIÇÃO DO PODER, e isto foi medido na tela: o `Detail` da aba
	// Poderes é o verbete inteiro do catálogo, e a Tatuagem Mística sozinha
	// ocupava três linhas — mais que as três ações padrão acima dela somadas. Com
	// dez poderes a lista deixa de ser uma lista.
	//
	// A pergunta desta superfície é "o que dá para fazer e quanto custa"; o texto
	// da regra é da aba Poderes, que existe e não some. É a mesma divisa que faz
	// a linha da arma mostrar o dano e não o verbete da espada.
	row := actionRow{
		Name: power.Name,
		Cost: priceWithoutTheAction(power.Cost),
		Can:  power.Can, Why: power.Why,
	}
	if power.Limit != "" {
		row.Cost = joinWithDot(row.Cost, power.Limit)
	}
	if power.Stance != nil {
		row.Live = power.Stance.Active
	}
	// O INSTANTÂNEO QUE VENDE DEGRAU ganha o gesto aqui (a Marca da Presa,
	// p50, ALE-487), e com o que está dando quando em curso. Os outros
	// instantâneos continuam sem botão nesta superfície — usar é da aba Poderes.
	if power.Degrees != nil {
		row.Live = power.InEffect != ""
		if row.Live {
			row.Detail = power.InEffect + " em curso"
		}
		roll := actionRoll{Label: "Usar", Path: "/poderes/usa/" + power.Command, Before: useResets}
		if power.Degrees.opensDialog() {
			roll.Command, roll.Opens = useOpensCommand(power.Command), useDialogID(power.Command)
			roll.Dialog = actionsDialogID(useStepsDialogID(power))
		}
		row.Rolls = []actionRoll{roll}
	}
	return row
}

// rowFromSpell traduz uma magia aprendida.
//
// O CÍRCULO entra no detalhe porque ele é o que a mesa pergunta primeiro, e o PM
// base é o preço — o aprimoramento sobe esse número e mora no diálogo da magia,
// que esta superfície não substitui.
func rowFromSpell(spell learnedSpellRow) actionRow {
	row := actionRow{
		Name:   "Conjurar · " + spell.Name,
		Detail: spell.Circle + " · " + spell.School,
		Cost:   strconv.Itoa(spell.BasePm) + " PM",
		Can:    true,
	}
	if spell.CD != "" {
		row.Detail = joinWithDot(row.Detail, "CD "+spell.CD)
	}
	return row
}

// priceWithoutTheAction tira o "PADRÃO · " do começo de "PADRÃO · 1 PM".
//
// Pelo separador e não por uma lista das sete palavras de ação: a lista teria de
// ser mantida em dia com o `writtenActions`, e duas listas divergem. O que sobra
// quando não há separador é o texto inteiro, que é o caso da postura.
func priceWithoutTheAction(cost string) string {
	if _, price, found := strings.Cut(cost, " · "); found {
		return price
	}
	return cost
}

func joinWithDot(left, right string) string {
	if left == "" {
		return right
	}
	return left + " · " + right
}

// rollWaitsForATarget é a expressão que DESABILITA o botão enquanto não há
// alvo escolhido.
//
// O SINAL É DA MESA e é lido aqui, e os dois lados disso são de propósito: quem
// desenha a lista de alvos é a Mesa, porque quem muda a lista é a FILA; quem
// desenha o botão é a ficha, porque o botão mora colado no número. O sinal é o
// único canal que atravessa os dois remendos — o do stream e o do gateway da
// ficha — sem ser trocado por nenhum deles.
//
// "false" e não a ausência do atributo: `data-attr` trata valor booleano como
// ATRIBUTO BOOLEANO, então o falso o REMOVE, que é exatamente o certo aqui.
func rollWaitsForATarget(roll actionRoll) string {
	if !roll.NeedsTarget {
		return "false"
	}
	return "$turn_target === ''"
}

// rollTitle explica o botão apagado. Um gesto desabilitado e sem motivo manda
// procurar o que está errado na ficha, e o que falta está na barra acima.
func rollTitle(roll actionRoll) string {
	if !roll.NeedsTarget {
		return ""
	}
	return "Escolha o alvo da vez na barra acima"
}

// dropRowsTheChipAlreadySays tira a linha que repete um chip do MESMO grupo.
//
// A linha de uma postura traz nome e preço; o chip traz nome, preço, ESTADO e
// o gesto. Quando os dois caem no mesmo grupo, a linha é a metade muda da
// mesma coisa — e com as Posturas de Combate (p54) isso virou quatro chips e
// quatro linhas idênticas logo abaixo deles.
//
// NO MESMO GRUPO, e é essa a precisão que custou uma volta: cortar toda linha
// de postura esvaziou o grupo "Livre e reação" do bárbaro, cuja única linha
// era a Fúria — e o chip dela mora no grupo do ATAQUE, porque é lá que está o
// número que ela muda. O que o jogador perdia não era a duplicata: era saber
// que ele tem uma ação livre.
func dropRowsTheChipAlreadySays(panel *actionsPanel) {
	for g := range panel.Groups {
		group := &panel.Groups[g]
		said := map[string]bool{}
		for _, chip := range group.Chips {
			said[chip.Label] = true
			said[strings.TrimPrefix(chip.Label, "Encerrar ")] = true
		}
		kept := make([]actionRow, 0, len(group.Rows))
		for _, row := range group.Rows {
			if said[row.Name] {
				continue
			}
			kept = append(kept, row)
		}
		group.Rows = kept
	}
}
