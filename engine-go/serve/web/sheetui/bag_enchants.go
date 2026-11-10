package sheetui

import (
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// O ENCANTO NA FOLHA DO ITEM. Quem lê vê o que ele faz; quem ENCANTA é o mestre, porque encanto não se
// compra, se acha ou se fabrica (p333). O mestre já abre a ficha do jogador pelo "Ficha" da mesa e do Grupo,
// e é por isso que a folha do item é a porta dele, e não um painel próprio na mesa (decisão do dono): um
// diálogo só, alcançado de onde o mestre já chega.

// appliedEnchants são os encantos que o item carrega, na ordem do livro, com o que cada um faz.
//
// Ordenados pelo NOME e não pela ordem de gravação: o `EnchantsOfItem` devolve
// por id, e "encanto-ameacadora" antes de "encanto-formidavel" é a ordem do
// slug, não a que se lê.
func appliedEnchants(item sheet.ItemDTO) []overlayRow {
	rows := []overlayRow{}
	for _, id := range item.Enchants {
		entry := book.ItemByID(id)
		if entry == nil {
			continue
		}
		rows = append(rows, overlayRow{Name: enchantName(*entry, item.EnchantChoices[id]), Effect: enchantEffect(*entry)})
	}
	sort.SliceStable(rows, func(a, b int) bool { return rows[a].Name < rows[b].Name })
	return rows
}

// enchantName é o nome do encanto com a escolha dele: "Anticriatura · contra morto-vivo".
func enchantName(entry book.Item, chosen string) string {
	if chosen == "" {
		return entry.Name
	}
	return entry.Name + " · contra " + book.ChosenKindName(chosen)
}

// enchantBadgeName é o nome do encanto no crachá da Mochila, com a escolha: "Anticriatura contra morto-vivo".
// Sem o "·" da folha, porque no crachá o "·" já separa um encanto do outro.
func enchantBadgeName(entry book.Item, chosen string) string {
	if chosen == "" {
		return entry.Name
	}
	return entry.Name + " contra " + book.ChosenKindName(chosen)
}

// enchantEffect é o que o encanto faz: as notas do verbete e o gasto de PM dele. Sem os dois, "sem efeito
// mecânico", que antes era o que a folha dizia também do encanto cujo efeito inteiro é o gasto.
func enchantEffect(entry book.Item) string {
	parts := []string{}
	if notes := modifierNotes(entry); notes != "" {
		parts = append(parts, notes)
	}
	if spend := enchantSpendWritten(entry.Boost); spend != "" {
		parts = append(parts, spend)
	}
	if len(parts) == 0 {
		return "sem efeito mecânico"
	}
	return strings.Join(parts, "; ")
}

// enchantSpendWritten é o gasto do encanto como a mesa lê: "2 PM no golpe, uma vez por rodada: +4d8 de dano
// contra o tipo escolhido", ou o que ele dispara em vez do golpe (a bola de fogo da Flamejante, p335).
func enchantSpendWritten(boost *engine.ItemBoost) string {
	if boost == nil {
		return ""
	}
	price := strconv.Itoa(boost.PM) + " PM"
	if boost.HitPointCost != "" {
		price += " e " + boost.HitPointCost + " PV"
	}
	if in := boost.Instead; in != nil {
		return fmt.Sprintf("%s em vez do golpe, uma vez por rodada: %s de %s de %s, alcance %s, %s %s",
			price, in.Name, in.Damage, in.DamageType, in.Range, in.Save, in.OnSave)
	}
	said := []string{}
	for _, m := range boost.Modifiers {
		switch m.Target.K {
		case "announcedCondition":
			line := "no acerto, " + book.ConditionName(m.Target.Name)
			if m.Note != "" {
				line += " (" + m.Note + ")"
			}
			said = append(said, line)
		case "announcedEffect":
			said = append(said, "no acerto, "+m.Note)
		default:
			if m.Note != "" {
				said = append(said, m.Note)
			}
		}
	}
	return price + " no golpe, uma vez por rodada: " + strings.Join(said, ", ")
}

// enchantChoice é um encanto que cabe no item, no diálogo do mestre.
type enchantChoice struct {
	ID     string
	Name   string
	Effect string
	Weight int
	// Requires é o pré-requisito (id), e RequiresName o nome dele, para o "exige Formidável".
	Requires     string
	RequiresName string
	// Spend é o "2 PM no golpe" da linha, vazio no encanto sem gasto.
	Spend  string
	Active bool
	// Against são as opções da escolha que o encanto pede (o tipo da Anticriatura), nulo no resto.
	Against []book.ChosenKindOption
}

// offerEnchants dá a cada folha de item que aceita encanto o diálogo do mestre: a tabela da família dele
// (8-8 para a arma, 8-10 para a armadura e o escudo), com o que o item já carrega ligado. Só a Load do mestre
// chama isto; a folha do jogador fica sem o gesto.
func offerEnchants(sheets []itemSheet, dto sheet.CharacterDTO) {
	for i := range sheets {
		at := slices.IndexFunc(dto.Items, func(it sheet.ItemDTO) bool { return it.ID == sheets[i].ID })
		if at < 0 {
			continue
		}
		item := dto.Items[at]
		catalog := catalogItem(item)
		if catalog == nil {
			continue
		}
		for _, entry := range book.EnchantsThatFit(*catalog) {
			choice := enchantChoice{
				ID: entry.ID, Name: entry.Name, Effect: firstSentence(entry.Description), Weight: max(entry.CountsAs, 1),
				Requires: entry.Requires, Active: slices.Contains(item.Enchants, entry.ID),
				Against: book.ChosenKindOptions(entry.ID),
			}
			if required := book.ItemByID(entry.Requires); required != nil {
				choice.RequiresName = required.Name
			}
			if entry.Boost != nil {
				choice.Spend = strconv.Itoa(entry.Boost.PM) + " PM no golpe"
				if entry.Boost.Instead != nil {
					choice.Spend = strconv.Itoa(entry.Boost.PM) + " PM em vez do golpe"
				}
			}
			if len(choice.Against) > 0 && sheets[i].EnchantAgainst == "" {
				sheets[i].EnchantAgainst = item.EnchantChoices[entry.ID]
			}
			sheets[i].EnchantChoices = append(sheets[i].EnchantChoices, choice)
		}
		sheets[i].Enchantable = len(sheets[i].EnchantChoices) > 0
	}
}

// firstSentence é a primeira frase do verbete, que diz o que o encanto faz sem a regra inteira.
func firstSentence(description string) string {
	sentence, _, _ := strings.Cut(description, ". ")
	return strings.TrimSuffix(sentence, ".")
}

// enchantsTheItem grava os encantos do item, e é o gesto do MESTRE: o pedido de quem não mestra a mesa da
// ficha é recusado aqui, no servidor, e não só na tela que não oferece o botão. As regras do conjunto —
// família, teto, pré-requisito e a escolha da Anticriatura — são do `EnchantItem`.
//
// O tipo escolhido chega num sinal só (`item_enchant_against`), porque um encanto só pede escolha hoje; ele
// vai para cada encanto do pedido que a pede.
func enchantsTheItem(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	if !s.mastersTheCharacter(r.Context(), row, s.deps.CurrentUserID(r)) {
		return fmt.Errorf("quem encanta é o mestre da mesa: encanto não se compra, se acha ou se fabrica (p333)")
	}
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	choices := map[string]string{}
	for _, id := range signals.ItemEnchants {
		if book.EnchantChooses(id) && signals.ItemEnchantAgainst != "" {
			choices[id] = signals.ItemEnchantAgainst
		}
	}
	// Pelo `itemGesture`, como a melhoria: um encanto pode mudar o PV máximo, e tirá-lo com pouca vida derruba.
	withEnchants := func(dto *sheet.CharacterDTO) {
		for i := range dto.Items {
			if dto.Items[i].ID == item.ID {
				dto.Items[i].Enchants, dto.Items[i].EnchantChoices = signals.ItemEnchants, choices
			}
		}
	}
	return s.itemGesture(r, row, withEnchants, func() error {
		return s.plays.EnchantItem(r.Context(), item.ID, signals.ItemEnchants, choices)
	})
}

// enchantsOpens carrega o que o item JÁ tem antes de abrir o diálogo, pela razão das melhorias: quem troca de
// item limpa.
func enchantsOpens(s itemSheet) string {
	ids := []string{}
	for _, c := range s.EnchantChoices {
		if c.Active {
			ids = append(ids, "'"+c.ID+"'")
		}
	}
	return "$item_enchants = [" + strings.Join(ids, ", ") + "]; $item_enchant_against = '" + s.EnchantAgainst + "'"
}

// enchantChosen é a expressão que diz se o encanto está escolhido AGORA.
func enchantChosen(c enchantChoice) string { return "$item_enchants.includes('" + c.ID + "')" }

// thatTogglesEnchant liga ou desliga um encanto na lista do sinal, copiada antes, pela razão das melhorias.
func thatTogglesEnchant(c enchantChoice) string {
	return "const escolhidos = [...$item_enchants]; const onde = escolhidos.indexOf('" + c.ID + "'); " +
		"if (onde >= 0) { escolhidos.splice(onde, 1) } else { escolhidos.push('" + c.ID + "') }; " +
		"$item_enchants = escolhidos"
}

// enchantWeightExpr é o PESO do que está escolhido agora: os que "contam como dois" pesam 2 (p334).
func enchantWeightExpr(s itemSheet) string {
	heavy := []string{}
	for _, c := range s.EnchantChoices {
		if c.Weight > 1 {
			heavy = append(heavy, fmt.Sprintf("'%s': %d", c.ID, c.Weight))
		}
	}
	return "$item_enchants.reduce((n, id) => n + ({" + strings.Join(heavy, ", ") + "}[id] ?? 1), 0)"
}

// enchantCountExpr é a contagem do topo: "2 de 3 encantos".
func enchantCountExpr(s itemSheet) string {
	return enchantWeightExpr(s) + " + ' de " + strconv.Itoa(book.EnchantSlots) + " encantos'"
}

// enchantCountWritten é a mesma contagem, escrita no servidor para o primeiro quadro.
func enchantCountWritten(s itemSheet) string {
	return strconv.Itoa(chosenWeight(s)) + " de " + strconv.Itoa(book.EnchantSlots) + " encantos"
}

func chosenWeight(s itemSheet) int {
	weight := 0
	for _, c := range s.EnchantChoices {
		if c.Active {
			weight += c.Weight
		}
	}
	return weight
}

// enchantLockedExpr tranca o encanto que não cabe AGORA: o pré-requisito fora do conjunto, ou o peso que
// passaria do teto. O escolhido nunca tranca, para poder sair. Quem recusa de verdade é o servidor.
func enchantLockedExpr(s itemSheet, c enchantChoice) string {
	why := fmt.Sprintf("%s + %d > %d", enchantWeightExpr(s), c.Weight, book.EnchantSlots)
	if c.Requires != "" {
		why = "!$item_enchants.includes('" + c.Requires + "') || " + why
	}
	return "!" + enchantChosen(c) + " && (" + why + ")"
}

// enchantMetaExpr é a linha pequena sob o nome: o motivo da tranca ("exige Formidável · sem espaço"), ou o
// gasto do encanto quando nada o tranca.
func enchantMetaExpr(s itemSheet, c enchantChoice) string {
	reasons := []string{}
	if c.Requires != "" {
		reasons = append(reasons, fmt.Sprintf("!$item_enchants.includes('%s') ? 'exige %s' : ''", c.Requires, c.RequiresName))
	}
	reasons = append(reasons, fmt.Sprintf("%s + %d > %d ? 'sem espaço' : ''", enchantWeightExpr(s), c.Weight, book.EnchantSlots))
	return fmt.Sprintf("(%s ? '' : [%s].filter(Boolean).join(' · ')) || '%s'",
		enchantChosen(c), strings.Join(reasons, ", "), c.Spend)
}

// enchantMetaWritten é a mesma linha, no servidor, para o primeiro quadro.
func enchantMetaWritten(s itemSheet, c enchantChoice) string {
	if reasons := lockReasons(s, c); len(reasons) > 0 {
		return strings.Join(reasons, " · ")
	}
	return c.Spend
}

// enchantLockedNow diz se o encanto está trancado no primeiro quadro, pela mesma conta do `enchantLockedExpr`.
func enchantLockedNow(s itemSheet, c enchantChoice) bool { return len(lockReasons(s, c)) > 0 }

// lockReasons são os motivos de o encanto não caber AGORA, no conjunto que o item já tem: o pré-requisito
// fora dele e o peso que passaria do teto. O escolhido não tem motivo, para poder sair.
func lockReasons(s itemSheet, c enchantChoice) []string {
	if c.Active {
		return nil
	}
	reasons := []string{}
	if c.Requires != "" && !slices.ContainsFunc(s.EnchantChoices, func(o enchantChoice) bool { return o.ID == c.Requires && o.Active }) {
		reasons = append(reasons, "exige "+c.RequiresName)
	}
	if chosenWeight(s)+c.Weight > book.EnchantSlots {
		reasons = append(reasons, "sem espaço")
	}
	return reasons
}

func itemEnchantsID(command string) string { return "item-enchants-" + command }
