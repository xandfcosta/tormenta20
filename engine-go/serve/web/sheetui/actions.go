package sheetui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"t20engine/domain/book"
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
	// Passives é a CONTAGEM das passivas, nunca as linhas delas: elas já estão
	// dentro dos números da ficha — a Defesa já traz a Esquiva Sobrenatural — e
	// uma linha por passiva encheria a tela de coisas sobre as quais não há o que
	// fazer. São 238 das 411 ativações do catálogo.
	Passives int
}

// actionGroup é um custo do livro com o que cabe nele. Grupo sem linha não é
// desenhado: um cabeçalho "Ação completa" sobre o nada é ruído com cara de
// seção.
type actionGroup struct {
	Title string
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
}

// actionChip é um interruptor sobre o número.
//
// Ele é o gesto que JÁ EXISTE — a postura dos Poderes, o situacional dos
// Efeitos — desenhado onde o número está. Essa é a metade da dor que a lista
// sozinha não cura: ver `+8` não lembra ninguém de que a Fúria existe, e ligá-la
// pela aba Poderes exige saber que ela existe, que é o que a pessoa esqueceu.
type actionChip struct {
	Label string
	// Cost é o preço de LIGAR. Um interruptor que cobra PM sem dizer quanto faz
	// a pessoa descobrir o custo depois de pagar.
	Cost string
	On   bool
	Can  bool
	Why  string
	// Note é o que o degrau compra ("+1 no bônus de Fúria"), vindo do catálogo.
	Note string
	// Command é o endereço do gesto, e ele é o MESMO das abas Poderes e Efeitos:
	// um segundo caminho de escrita seria uma segunda regra de validação.
	Command string
	// Moves são os números que este chip muda — ver `actionRow.Moves`.
	Moves []string
}

// actionsPanelFrom monta a superfície a partir dos painéis JÁ computados.
//
// A ORDEM DOS GRUPOS é a do livro (p233): padrão, movimento, completa, e o que
// não gasta a vez por último. O custo negociado fecha a lista porque ele não é
// um custo — é a mesa decidindo um.
func actionsPanelFrom(
	id int64, computed engine.ComputedSheet,
	combat panelCombat, powers powersPanel, spells spellbookPanel, effects effectsPanel,
) actionsPanel {
	melee := expertiseOrZero(computed, "Luta", "strength").Total + computed.AttackAll.Total
	byCost := map[engine.ActionCost][]actionRow{
		engine.ActionStandard: standardRows(combat, melee, computed),
		engine.ActionMovement: {movementRow(computed)},
		engine.ActionFull:     fullRows(melee, computed),
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
				actionGroup{Title: group.title, Note: group.note, Rows: rows})
		}
	}
	panel.Passives = len(powers.Passives)
	hangChips(&panel, chipsOffered(id, powers, effects.Situational))
	return panel
}

// hangChips pendura cada interruptor nos GRUPOS que ele muda.
func hangChips(panel *actionsPanel, chips []actionChip) {
	for g := range panel.Groups {
		group := &panel.Groups[g]
		numbers := []string{}
		for _, row := range group.Rows {
			numbers = append(numbers, row.Moves...)
		}
		for _, chip := range chips {
			if sharesANumber(numbers, chip.Moves) {
				group.Chips = append(group.Chips, chip)
			}
		}
	}
}

func sharesANumber(left, right []string) bool {
	for _, l := range left {
		if slices.Contains(right, l) {
			return true
		}
	}
	return false
}

// chipsOffered junta as duas famílias de interruptor: a POSTURA, que custa PM e
// dura, e o SITUACIONAL, que é opt-in e não cobra nada.
//
// Elas moram em abas diferentes porque são coisas diferentes de ADMINISTRAR;
// aqui são a mesma coisa de USAR — "um interruptor que muda este número" —, e
// por isso viram uma lista só.
func chipsOffered(id int64, powers powersPanel, situational []situationalRow) []actionChip {
	chips := []actionChip{}
	for _, power := range powers.Actions {
		if power.Stance != nil {
			chips = append(chips, stanceChips(id, power)...)
		}
	}
	for _, row := range situational {
		chips = append(chips, actionChip{
			Label: row.Label, On: row.Active, Can: true,
			Moves: numbersMovedBy(row.Targets),
			Command: fmt.Sprintf("$status = %q; @post('/personagens/%d/efeitos/situacao?embutida=1')",
				row.Key, id),
		})
	}
	return chips
}

// stanceChips é UM chip por postura: entrar, ou encerrar o que está em curso.
//
// UM E NÃO UM POR DEGRAU, e a razão foi MEDIDA no navegador, não deduzida. O
// bárbaro de nível 6 entrou em Fúria pagando o degrau ZERO — `steps=0,
// pmPaid=2` no banco —, e o ataque subiu de +8 para +11. O bônus veio do +3, que
// o catálogo concede por NÍVEL (`class.barbaro.furia-3`, `grantedAtLevel: 6`) e
// o motor resolve por `bonusType: morale`. Os degraus pagos NÃO entram na conta.
//
// Então um chip "Fúria 3 PM" cobraria 1 PM a mais pelo mesmo número: a tela
// oferecendo uma escolha que não muda nada, que é pior do que não oferecer. O
// contador de degraus continua na aba Poderes, onde ele já estava e onde este
// slice não encosta — a divergência é do MOTOR, e consertá-la é outra fatia.
func stanceChips(id int64, power powerRow) []actionChip {
	stance := power.Stance
	if stance.Active {
		return []actionChip{{
			Label: "Encerrar " + power.Name, On: true, Can: true, Moves: stanceMoves(),
			Command: fmt.Sprintf("@post('/personagens/%d/efeitos/postura/%s?embutida=1')",
				id, stance.Flag),
		}}
	}
	return []actionChip{{
		Label: power.Name,
		Cost:  strconv.Itoa(stance.BasePm) + " PM",
		Can:   power.Can, Why: power.Why, Moves: stanceMoves(),
		Command: fmt.Sprintf("$stance_degrees = 0; @post('/personagens/%d/poderes/postura/%s/entra?embutida=1')",
			id, stance.Flag),
	}}
}

// stanceMoves são os números que uma postura muda.
//
// HOJE É UMA LISTA FIXA — ataque e dano —, e o comentário diz por quê: os
// modificadores de uma postura entram por `condition: {c: "flagOn"}` e NÃO
// passam pelo `Conditional`, que é de onde o situacional tira os alvos dele.
// Ler os alvos de verdade exigiria o catálogo de poderes aqui dentro, e o que
// isso compraria é a postura que mexe em OUTRA coisa — a Inspiração do bardo
// soma em perícia. Quando a segunda postura chegar, é aqui que ela entra, e o
// caso que a cobra nasce com ela.
func stanceMoves() []string {
	return []string{"attack", "damage"}
}

// numbersMovedBy traduz alvos de modificador no vocabulário das linhas.
//
// Ele colapsa o ESCOPO de propósito: `{k: attack, scope: all}` e
// `{k: attack, scope: this}` mudam o mesmo número da mesma linha, e distinguir
// os dois aqui daria dois vocabulários para uma pergunta só.
func numbersMovedBy(targets []engine.ModifierTarget) []string {
	out := []string{}
	for _, t := range targets {
		if t.K == "expertise" {
			out = append(out, "expertise:"+t.Name)
			continue
		}
		out = append(out, t.K)
	}
	return out
}

// groupOrder são os grupos na ordem da tela, e QUAIS custos caem em cada um.
//
// LIVRE E REAÇÃO dividem um grupo, e a razão é de largura e não de conceito: as
// duas são "não gasta a vez", e dois cabeçalhos para duas listas de uma linha
// custariam a 390px mais do que entregam. A distinção que o GLOSSARY.md defende
// — a livre é escolha consciente no seu turno, a reação vale fora dele — não se
// perde: ela vai escrita na linha, no `Cost`.
var groupOrder = []struct {
	title string
	note  string
	costs []engine.ActionCost
}{
	{"Ação padrão", "", []engine.ActionCost{engine.ActionStandard}},
	{"Ação de movimento", "", []engine.ActionCost{engine.ActionMovement}},
	{"Ação completa", "abre mão das duas", []engine.ActionCost{engine.ActionFull}},
	{"Livre e reação", "não gastam a vez",
		[]engine.ActionCost{engine.ActionFree, engine.ActionReaction}},
	{"Custo que a mesa decide", "o livro não fixa um",
		[]engine.ActionCost{engine.ActionVaries}},
}

// standardRows são as ações padrão que toda ficha de combate tem.
//
// O ATAQUE VEM PRIMEIRO, e o livro diz por quê: *"fazer um ataque ou lançar uma
// magia são as ações padrão mais comuns"* (p233). A 390px a primeira linha é a
// única que se lê sem rolar.
func standardRows(combat panelCombat, melee int, computed engine.ComputedSheet) []actionRow {
	rows := make([]actionRow, 0, len(combat.Weapons)+2)
	for _, weapon := range combat.Weapons {
		rows = append(rows, actionRow{
			Name:   "Atacar · " + weapon.Name,
			Detail: weapon.Damage + " · " + weapon.Crit + " · " + weapon.Skill,
			Value:  weapon.Attack,
			Can:    true,
			Moves:  []string{"attack", "damage"},
		})
	}
	// A MANOBRA SÓ EXISTE COM ARMA CORPO A CORPO NA MÃO: *"não é possível fazer
	// manobras de combate com ataques à distância"* (p234). Oferecê-la ao
	// arqueiro seria desenhar um gesto que a regra recusa.
	if hasMeleeWeapon(combat.Weapons) {
		rows = append(rows, actionRow{
			Name:   "Manobra",
			Detail: "agarrar, derrubar, desarmar, empurrar, quebrar · p234",
			Value:  book.WithSign(melee),
			Can:    true,
			Moves:  []string{"attack"},
		})
	}
	// FINTAR é teste de Enganação contra os Reflexos do alvo (p234) — é a única
	// ação padrão do livro que não sai de Luta nem de Pontaria.
	rows = append(rows, actionRow{
		Name:   "Fintar",
		Detail: "Enganação contra os Reflexos do alvo · p234",
		Value:  book.WithSign(expertiseOrZero(computed, "Enganação", "charisma").Total),
		Can:    true,
		Moves:  []string{"expertise:Enganação"},
	})
	return rows
}

// movementRow é o deslocamento, em METROS.
//
// O motor o guarda em QUADRADOS de 1,5m porque é assim que o tabuleiro conta, e
// a tela diz metros porque é assim que a mesa fala — a conversão mora aqui, de um
// lado só.
func movementRow(computed engine.ComputedSheet) actionRow {
	meters := float64(computed.Displacement.Total) * 1.5
	written := strconv.FormatFloat(meters, 'f', -1, 64)
	return actionRow{
		Name:   "Mover",
		Detail: "percorrer o seu deslocamento · p233",
		Value:  strings.Replace(written, ".", ",", 1) + "m",
		Can:    true,
	}
}

// fullRows são as duas completas que saem de número que a ficha já tem.
//
// O GOLPE DE MISERICÓRDIA fica de fora apesar de ser a terceira da p235: ele não
// tem teste nem número — é crítico automático mais uma chance de morte que o
// mestre rola —, e a decisão do dono é que linha sem número nem botão não entra.
func fullRows(melee int, computed engine.ComputedSheet) []actionRow {
	return []actionRow{{
		Name:   "Investida",
		Detail: "em linha reta · +2 no ataque, −2 na Defesa até a sua vez · p235",
		Value:  book.WithSign(melee + investidaBonus),
		Can:    true,
		Moves:  []string{"attack"},
	}, {
		Name:   "Corrida",
		Detail: "Atletismo · p235",
		Value:  book.WithSign(expertiseOrZero(computed, "Atletismo", "strength").Total),
		Can:    true,
		Moves:  []string{"expertise:Atletismo"},
	}}
}

// investidaBonus é o +2 que a investida dá no ataque (p235). Ele é constante do
// livro e não entra no motor: a investida ainda não é um gesto que o app resolva,
// e um modificador no `ComputeSheet` sem ninguém para ligá-lo seria regra sem
// produtor.
const investidaBonus = 2

// hasMeleeWeapon pergunta pela PERÍCIA do cartão, que é "Luta" ou "Pontaria".
//
// Pela perícia e não por uma lista de nomes de arma: quem decide se a arma é
// corpo a corpo é o catálogo, e o cartão já carrega a resposta dele.
func hasMeleeWeapon(weapons []weaponTile) bool {
	for _, weapon := range weapons {
		if weapon.Skill == "Luta" {
			return true
		}
	}
	return false
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

// chipTitle é o que o chip diz ao passar o mouse: o que o degrau compra, ou a
// razão de ele estar apagado. Um chip desligado e sem explicação manda procurar.
func chipTitle(chip actionChip) string {
	if !chip.Can && chip.Why != "" {
		return chip.Why
	}
	return chip.Note
}

// ariaBool escreve o `aria-pressed` de um interruptor. Ele é "true"/"false" e
// não a ausência do atributo: um botão que ALTERNA tem estado, e quem lê por
// leitor de tela precisa dele nos dois sentidos.
func ariaBool(on bool) string {
	if on {
		return "true"
	}
	return "false"
}
