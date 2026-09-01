package api

import (
	"encoding/json"
	"sync"

	"t20engine/catalog"
	"t20engine/engine"
)

// O QUE O HERÓI LEVA AO NASCER (ALE-272, fatia 9).
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
func birthItems(folha forgeAnswers, kit engine.StartingKit) []startingItemBody {
	escolhidos := append([]string{}, kit.BaseItems...)
	escolhidos = append(escolhidos, folha.SimpleWeapon, folha.MartialWeapon, folha.Armor)
	if folha.Shield {
		escolhidos = append(escolhidos, kit.Shield)
	}

	itens := make([]startingItemBody, 0, len(escolhidos)+2)
	for _, id := range escolhidos {
		if linha := catalogRow(id); linha != nil {
			itens = append(itens, *linha)
		}
	}
	for _, concessao := range originGrants(folha.Origin) {
		if concessao.Kind == engine.OriginItemFixed {
			itens = append(itens, originRow(concessao.Name))
		}
	}
	return itens
}

// catalogRow é um item do livro virando linha da mochila. Id vazio ou
// desconhecido devolve nil: o kit tem peças opcionais, e conferir se elas EXISTEM
// é trabalho de `forgeRefusals`, não deste montador.
func catalogRow(id string) *startingItemBody {
	item := itemDoLivroPorID(id)
	if item == nil {
		return nil
	}
	quantidade := int64(1)
	return &startingItemBody{
		CatalogID: &item.ID, Name: &item.Name, Quantity: &quantidade, Slots: &item.Slots,
	}
}

// originRow é uma concessão fixa virando linha.
//
// Ela procura o item no catálogo pelo NOME, porque a origem cita o item por
// escrito e não por id — "Símbolo sagrado" é uma entrada de `items.json`. Quando
// não casa (a origem cita coisas que o catálogo não vende, como "Traje de
// sacerdote"), a linha nasce sem catálogo, ocupando um espaço.
func originRow(nome string) startingItemBody {
	if item := bookItemByName(nome); item != nil {
		quantidade := int64(1)
		return startingItemBody{
			CatalogID: &item.ID, Name: &item.Name, Quantity: &quantidade, Slots: &item.Slots,
		}
	}
	quantidade, espacos := int64(1), 1.0
	return startingItemBody{Name: &nome, Quantity: &quantidade, Slots: &espacos}
}

// birthPurse rola os T$ 4d6 de p140 e soma o dinheiro que a origem
// conceder ("T$ 2d6 (último salário)", do Artesão).
func birthPurse(origem string) (float64, error) {
	total, err := engine.RollStartingMoney()
	if err != nil {
		return 0, err
	}
	for _, concessao := range originGrants(origem) {
		if concessao.Kind != engine.OriginItemMoney {
			continue
		}
		extra, err := engine.RollDiceNotation(concessao.Dice)
		if err != nil {
			return 0, err
		}
		total += extra
	}
	return float64(total), nil
}

// originGrants são as linhas "Itens" da origem, já classificadas.
func originGrants(origem string) []engine.OriginItemGrant {
	frases := originItemsByName()[origem]
	concessoes := make([]engine.OriginItemGrant, 0, len(frases))
	for _, frase := range frases {
		concessoes = append(concessoes, engine.ParseOriginItem(frase))
	}
	return concessoes
}

// originItemsByName indexa a linha "Itens" de cada origem POR NOME.
//
// Lê `origens` e não `origins`: são dois arquivos sobre as mesmas 35 origens,
// com campos diferentes — `origins` tem os benefícios (que a aba de Poderes já
// usa, em `origensDoLivro`) e `origens` tem os itens e as perícias. Ler o
// arquivo errado devolve mapa vazio em silêncio, que é a razão de o guarda de
// varredura afirmar o denominador.
var (
	originItemsOnce  sync.Once
	originItemsIndex map[string][]string
)

func originItemsByName() map[string][]string {
	originItemsOnce.Do(func() {
		originItemsIndex = map[string][]string{}
		bruto, ok := catalog.Resource("origens")
		if !ok {
			return
		}
		var porID map[string]struct {
			Name          string   `json:"name"`
			ItensIniciais []string `json:"itensIniciais"`
		}
		if err := json.Unmarshal(bruto, &porID); err != nil {
			return
		}
		for _, origem := range porID {
			originItemsIndex[origem.Name] = origem.ItensIniciais
		}
	})
	return originItemsIndex
}
