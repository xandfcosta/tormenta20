package forge

import (
	"fmt"
	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/sheet"

	"t20engine/domain/engine"
)

// O QUE O HERÓI LEVA AO NASCER.
//
// Duas procedências, as duas de p140: o kit ("Personagens de 1º nível começam
// com os itens fornecidos pela sua origem e os itens a seguir") e a linha
// "Itens" da origem ("Você começa com todos os itens descritos na linha 'Itens'
// de sua origem sem pagar por eles", p85).
//
// NADA NASCE EQUIPADO, e isso é decisão e não esquecimento. Empunhar tem regra —
// no máximo duas mãos ocupadas (p141) —, e o kit de um guerreiro tem arma
// simples, arma marcial e escudo: equipar os três no nascimento gravaria à
// revelia um estado que a Mochila recusaria se alguém pedisse. Quem empunha é o
// jogador, na Mochila, que é onde a regra das mãos mora.

// birthItems monta as linhas de item do kit escolhido mais as
// concessões FIXAS da origem.
//
// As concessões de ESCOLHA da origem ficam de fora de propósito — "Estojo de
// disfarces OU gazua" não é item, é uma pergunta, e gravá-la como nome poria na
// mochila uma linha que ocupa carga e não existe no livro. A folha diz quais
// são, e a Mochila é onde elas viram item.
func birthItems(stylesheet forgeAnswers, kit engine.StartingKit) []sheet.StartingItem {
	chosen := append([]string{}, kit.BaseItems...)
	chosen = append(chosen, stylesheet.SimpleWeapon, stylesheet.MartialWeapon, stylesheet.Armor)
	if stylesheet.Shield {
		chosen = append(chosen, kit.Shield)
	}

	items := make([]sheet.StartingItem, 0, len(chosen)+2)
	for _, id := range chosen {
		if row := catalogRow(id); row != nil {
			items = append(items, *row)
		}
	}
	for _, grant := range originGrants(stylesheet.Origin) {
		if grant.Kind == engine.OriginItemFixed {
			items = append(items, originRow(grant.Name))
		}
	}
	return items
}

// catalogRow é um item do livro virando linha da mochila. Id vazio ou
// desconhecido devolve nil: o kit tem peças opcionais, e conferir se elas EXISTEM
// é trabalho de `forgeRefusals`, não deste montador.
func catalogRow(id string) *sheet.StartingItem {
	item := book.ItemByID(id)
	if item == nil {
		return nil
	}
	amount := int64(1)
	return &sheet.StartingItem{
		CatalogID: &item.ID, Name: &item.Name, Quantity: &amount, Slots: &item.Slots,
	}
}

// originRow é uma concessão fixa virando linha.
//
// Ela procura o item no catálogo pelo NOME, porque a origem cita o item por
// escrito e não por id — "Símbolo sagrado" é uma entrada de `items.json`. Quando
// não casa (a origem cita coisas que o catálogo não vende, como "Traje de
// sacerdote"), a linha nasce sem catálogo, ocupando um espaço.
func originRow(name string) sheet.StartingItem {
	if item := book.ItemByName(name); item != nil {
		amount := int64(1)
		return sheet.StartingItem{
			CatalogID: &item.ID, Name: &item.Name, Quantity: &amount, Slots: &item.Slots,
		}
	}
	quantity, spaces := int64(1), 1.0
	return sheet.StartingItem{Name: &name, Quantity: &quantity, Slots: &spaces}
}

// birthPurse é a bolsa de p140 para este nível, somada ao dinheiro que a origem
// conceder ("T$ 2d6 (último salário)", do Artesão). O 1º nível rola 4d6; acima
// dele, o valor é o da Tabela 3-1, sem dado.
func birthPurse(origin string, level int64) (float64, error) {
	var total int
	var err error
	if level > 1 {
		total, err = engine.StartingMoneyForLevel(int(level))
	} else {
		total, err = engine.RollStartingMoney()
	}
	if err != nil {
		return 0, err
	}
	for _, grant := range originGrants(origin) {
		if grant.Kind != engine.OriginItemMoney {
			continue
		}
		extra, err := engine.RollDiceNotation(grant.Dice)
		if err != nil {
			return 0, err
		}
		total += extra
	}
	return float64(total), nil
}

// originGrants são as linhas "Itens" da origem, já classificadas.
func originGrants(origin string) []engine.OriginItemGrant {
	sentences := book.OriginItemsByName()[origin]
	grants := make([]engine.OriginItemGrant, 0, len(sentences))
	for _, sentence := range sentences {
		grants = append(grants, engine.ParseOriginItem(sentence))
	}
	return grants
}

// purseWritten diz na folha a bolsa que o herói vai ter neste nível (p140).
//
//	purseWritten(4) // "T$ 1.000 pelo nível 4"
func purseWritten(level int64) string {
	tibar, err := engine.StartingMoneyForLevel(int(level))
	if err != nil {
		return "T$ " + engine.StartingMoneyDice + " rolados na hora"
	}
	return fmt.Sprintf("T$ %s pelo nível %d", thousandsWritten(tibar), level)
}

// thousandsWritten separa os milhares como a tabela do livro escreve: 260.000.
func thousandsWritten(n int) string {
	digits := strconv.Itoa(n)
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "." + digits[i:]
	}
	return digits
}
