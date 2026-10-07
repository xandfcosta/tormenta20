package sheetui

import (
	"fmt"
	"slices"
	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// O INTERRUPTOR DA SUPERFÍCIE AÇÕES.
//
// Ele saiu do `actions.go` quando o arquivo passou do teto, e a divisão tem
// nome: ali mora O QUE DÁ PARA FAZER, aqui mora O QUE MUDA O NÚMERO DISSO. As
// duas mudam por razões diferentes — uma ação nova do livro mexe lá; uma
// família nova de modificador (a segunda postura, o situacional que sai de
// outro lugar) mexe aqui.

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
	// Moves são os números que este chip muda — ver `actionRow.Moves`. Vazio na
	// POSTURA, que pendura por outro critério (ver `Cost`).
	Moves []string
	// GroupCost é o custo de ASSUMIR, e é por ele que a postura acha o grupo.
	// Vazio no situacional, que não custa nada e pendura pelo número.
	GroupCost engine.ActionCost
}

// hangChips pendura cada interruptor nos GRUPOS que ele muda.
// O NÚMERO VEM PRIMEIRO; O CUSTO É O RECUO.
//
// A tese dos chips é a adjacência: *"o interruptor fica em cima do número que
// ele muda"*. Ela só responde quando o número ESTÁ NESTA TELA — a Fúria mexe
// em ataque e dano, que são linhas daqui, e o chip dela fica sobre elas.
//
// As Posturas de Combate mostraram o outro caso: a Muralha mexe em Defesa e
// Reflexos, que a superfície não lista, e a Torre e o Foco de Batalha não
// mexem em número nenhum. Penduradas só pelo número, as três sumiriam — e com
// elas o gesto de assumi-las.
//
// O recuo é o que a superfície já organiza: o que a vez CUSTA. Assumir uma
// postura custa movimento, e o chip aparece onde a pessoa procura o que fazer
// com o movimento dela.
func hangChips(panel *actionsPanel, chips []actionChip) {
	byNumber := map[string]bool{}
	for g := range panel.Groups {
		numbers := []string{}
		for _, row := range panel.Groups[g].Rows {
			numbers = append(numbers, row.Moves...)
		}
		for _, chip := range chips {
			if sharesANumber(numbers, chip.Moves) {
				byNumber[chip.Label] = true
			}
		}
	}
	for g := range panel.Groups {
		group := &panel.Groups[g]
		for _, chip := range chips {
			if chip.GroupCost == "" || byNumber[chip.Label] {
				continue
			}
			if slices.Contains(groupOrderOf(group.Title), chip.GroupCost) {
				group.Chips = append(group.Chips, chip)
			}
		}
	}
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
			Label: "Encerrar " + power.Name, On: true, Can: true,
			Moves:     numbersMovedBy(book.StanceTargets(stance.Flag)),
			GroupCost: stanceGroupCost(stance.Flag),
			Command: fmt.Sprintf("@post('/personagens/%d/efeitos/postura/%s?embutida=1')",
				id, stance.Flag),
		}}
	}
	return []actionChip{{
		Label: power.Name,
		Cost:  strconv.Itoa(stance.BasePm) + " PM",
		Can:   power.Can, Why: power.Why,
		Moves:     numbersMovedBy(book.StanceTargets(stance.Flag)),
		GroupCost: stanceGroupCost(stance.Flag),
		Command: fmt.Sprintf("$stance_degrees = 0; @post('/personagens/%d/poderes/postura/%s/entra?embutida=1')",
			id, stance.Flag),
	}}
}

// A POSTURA PENDURA PELO QUE ELA CUSTA, e não pelo que ela muda.
//
// Aqui morava uma lista fixa de ataque e dano, e o comentário dela prometia:
// *"quando a segunda postura chegar, é aqui que ela entra"*. Chegaram SEIS —
// as Posturas de Combate do cavaleiro (p54) —, e elas mostraram que a lista
// não tinha conserto por esse caminho: a Muralha mexe em Defesa e Reflexos, e
// a Torre e o Foco de Batalha não mexem em número NENHUM. Lendo os alvos de
// verdade, as duas últimas não teriam grupo e sumiriam da tela — o jogador
// perderia o gesto de assumi-las.
//
// A resposta é a pergunta que a superfície já organiza: ela agrupa pelo que a
// vez CUSTA, e assumir uma postura custa — livre na Fúria, padrão na
// Inspiração, movimento nas Posturas de Combate. O chip aparece onde a pessoa
// procura o que fazer com aquela ação, que é onde ela decide.
func stanceGroupCost(flag string) engine.ActionCost {
	spec, found := book.StancesFromCatalog()[flag]
	if !found || spec.Action == "" {
		return engine.ActionFree
	}
	return engine.ActionCost(spec.Action)
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

// groupOrderOf são os custos que caem naquele grupo da tela.
//
// Ela lê a MESMA tabela que desenha os grupos, para não haver uma segunda
// opinião sobre onde o movimento cai.
func groupOrderOf(title string) []engine.ActionCost {
	for _, g := range groupOrder {
		if g.title == title {
			return g.costs
		}
	}
	return nil
}
