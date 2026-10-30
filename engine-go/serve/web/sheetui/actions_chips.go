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
	// Path é o caminho do gesto, e ele é o MESMO das abas Poderes e Efeitos:
	// um segundo caminho de escrita seria uma segunda regra de validação. O
	// ENDEREÇO sai do `chipCommand`, com as marcas da ficha em que o chip é
	// desenhado.
	Path string
	// Before é o que roda antes do gesto: o sinal que ele lê.
	Before string
	// Detail é a sobreposição de hoje que o chip que pergunta abre, e Dialog é
	// o diálogo do kit que a mesma pergunta abre na sessão do jogador.
	Detail, Dialog string
	// Asks marca o chip que ABRE uma escolha em vez de fazer o gesto — hoje só a
	// postura que escala com o nível. O botão ganha `aria-haspopup`, porque quem
	// lê por leitor de tela precisa saber que o toque vai pedir mais uma coisa.
	Asks bool
	// Bonus é o que a postura EM CURSO está dando ("+3"), e ele ocupa o lugar do
	// `Cost` — um e outro nunca são escritos ao mesmo tempo: antes de subir o que
	// importa é o preço, depois de subir é o número.
	Bonus string
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
func chipsOffered(powers powersPanel, situational []situationalRow) []actionChip {
	chips := []actionChip{}
	for _, power := range powers.Actions {
		if power.Stance != nil {
			chips = append(chips, stanceChips(power)...)
		}
	}
	for _, row := range situational {
		chips = append(chips, actionChip{
			Label: row.Label, On: row.Active, Can: true, Bonus: row.Dice,
			Moves:  numbersMovedBy(row.Targets),
			Before: fmt.Sprintf("$status = %q; ", row.Key), Path: "/efeitos/situacao",
		})
	}
	return chips
}

// stanceChips é UM chip por postura: assumir a que está parada, ou encerrar a
// que está em curso.
//
// UM E NÃO UM POR DEGRAU. O motivo era uma DIVERGÊNCIA — os degraus eram
// cobrados e não entravam na conta, então um chip "Fúria 3 PM" cobraria 1 PM a
// mais pelo mesmo número —, e ela foi consertada (ALE-423). Hoje o motivo é de
// tela: a Fúria aparece em DOIS grupos (ataque e ação completa), então um chip
// por degrau daria oito botões de um poder só a 390px, e o grupo deixaria de se
// ler. O `TestTheStanceChipRidesTheNumberItChanges` conta essa unicidade.
//
// O CHIP QUE TEM DEGRAU PARA ESCOLHER NÃO ENTRA: ele ABRE o contador, que é o
// mesmo da aba Poderes (ALE-427). Entrar pelo mínimo sem perguntar era o app
// escolhendo pelo jogador, e a escolha não tem volta — o degrau se paga ao
// INVOCAR (p41, p44), encerrar não devolve PM, e reentrar mais alto cobra a
// entrada inteira de novo. Medido: 5 PM por um +3 que custa 3.
//
// Sem degrau a escolher, segue de um toque — são as seis Posturas de Combate
// (p54) e todo bárbaro abaixo do 6º. Perguntar ali seria um toque a mais numa
// pergunta de uma resposta só.
func stanceChips(power powerRow) []actionChip {
	stance := power.Stance
	if stance.Active {
		return []actionChip{{
			Label: "Encerrar " + power.Name, On: true, Can: true,
			// O BÔNUS EM CURSO vai no mesmo lugar onde o preço estava antes de a
			// postura subir, e ele responde o que o verbo sozinho não responde:
			// encerrar desfaz um PM que não volta, e quem decide precisa saber o
			// que vai perder. Era preciso ir à aba Efeitos para descobrir.
			Bonus:     bonusInEffect(stance),
			Moves:     numbersMovedBy(book.StanceTargets(stance.Flag)),
			GroupCost: stanceGroupCost(stance.Flag),
			Path:      "/efeitos/postura/" + stance.Flag,
		}}
	}
	chip := actionChip{
		Label: power.Name,
		Cost:  strconv.Itoa(stance.BasePm) + " PM",
		Can:   power.Can, Why: power.Why,
		Moves:     numbersMovedBy(book.StanceTargets(stance.Flag)),
		GroupCost: stanceGroupCost(stance.Flag),
		Before:    "$stance_degrees = 0; ", Path: "/poderes/postura/" + stance.Flag + "/entra",
	}
	if stance.MaxSteps > 0 {
		// O `+` é a mesma grafia do crachá da linha ("POSTURA · 2+ PM"), e aqui
		// ele tem a segunda função de avisar que o toque vai PERGUNTAR: um chip
		// que anuncia 2 PM e abre um diálogo que cobra 3 é um preço que muda
		// depois do toque.
		chip.Cost = strconv.Itoa(stance.BasePm) + "+ PM"
		chip.Path, chip.Detail, chip.Dialog = "", "postura-"+stance.Flag, actionsDialogID(stanceDialogID(power))
		chip.Asks = true
	}
	return []actionChip{chip}
}

// chipCommand escreve o gesto do chip na ficha em que ele é desenhado: o
// caminho vira endereço pelo `sheetPost`, com a aba e as marcas dela. O chip
// que pergunta abre a sobreposição de hoje; na sessão do jogador, quem abre o
// diálogo do kit é o `commandfor` (`Dialog`).
//
//	chipCommand(v, chip) // "$stance_degrees = 0; @post('/personagens/7/poderes/postura/furia/entra?tab=…&embutida=1')"
func chipCommand(v View, chip actionChip) string {
	if chip.Asks {
		return chip.Before + "$detail = '" + chip.Detail + "'"
	}
	return chip.Before + sheetPost(v, chip.Path)
}

// actionsDialogID é o id de um diálogo de degrau desenhado na superfície
// Ações. Ele é OUTRO que o da aba Poderes porque os dois convivem na página
// da sessão, e o `commandfor` acharia o primeiro — escondido na Ficha.
func actionsDialogID(id string) string {
	return "actions-" + id
}

// bonusInEffect é o "+3" de uma postura em curso.
//
// VAZIO em duas situações, e a segunda foi decidida OLHANDO a tela: quando a
// postura não tem um número que a descreva (a Torre Inabalável, p55, está em pé
// ou não está) e quando o número não foi ESCOLHIDO. O "+1" constante da Muralha
// Intransponível ficava ao lado de um rótulo de 38 caracteres, competindo com o
// nome por uma informação que a aba Combate já dá na Defesa.
func bonusInEffect(stance *stanceState) string {
	if stance.Bonus <= 0 || !book.StanceScales(stance.Flag) {
		return ""
	}
	return book.WithSign(stance.Bonus)
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
