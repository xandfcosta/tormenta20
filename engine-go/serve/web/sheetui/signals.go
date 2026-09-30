package sheetui

import (
	"net/http"
	"t20engine/domain/sheet"

	"github.com/starfederation/datastar-go/datastar"
)

// OS SINAIS DA FICHA, lidos UMA VEZ por requisição.
//
// UMA struct e UMA leitura porque o `datastar.ReadSignals` CONSOME O CORPO num
// `POST`: duas chamadas na mesma requisição deixam a segunda sem nada — e sem
// erro, porque corpo vazio é JSON ausente e não JSON inválido. Quem rodasse por
// último receberia vazio, e a lista voltaria sem filtro como se a pessoa
// tivesse apagado a busca.
//
// A biblioteca ainda exige a ORDEM: `ReadSignals` ANTES do `NewSSE`, senão ela
// devolve "are you sure you created the SSE ***AFTER*** the ReadSignals?". Num
// `GET` os sinais vêm na consulta e a ordem não morde — o que faria o defeito
// nascer no dia em que um gesto virasse `POST`.

// Signals é o que o cliente manda junto de qualquer gesto da ficha.
//
// Os campos são PONTEIROS para separar "não veio" de "veio vazio". Apagar a
// busca é gesto legítimo, e tratá-lo como ausência ressuscitaria o termo
// anterior — é a mesma decisão do `finderTerm`.
type Signals struct {
	Search *string `json:"search"`
	// NewExpertise e NovoAtributo são os dois campos do diálogo de ofício novo.
	NewExpertise *string `json:"new_expertise"`
	NewAttribute *string `json:"new_attribute"`
	// Status é a CHAVE do condicional que o gesto quer alternar. Ela vem por
	// sinal e não pelo caminho porque é um encadeado com `::` e texto livre do
	// catálogo dentro — um `PathEscape` daquilo funciona e é ilegível no log.
	Status *string `json:"conditional"`
	// Aprimoramentos são as pilhas escolhidas no diálogo de conjurar, uma por
	// índice. Seis porque é o máximo do catálogo (Conjurar Monstro).
	Aug0 *int `json:"augment0"`
	Aug1 *int `json:"augment1"`
	Aug2 *int `json:"augment2"`
	Aug3 *int `json:"augment3"`
	Aug4 *int `json:"augment4"`
	Aug5 *int `json:"augment5"`
	// Os filtros do catálogo de magias.
	SpellSearch string `json:"spell_search"`
	SpellCircle string `json:"spell_circle"`
	SpellSchool string `json:"spell_school"`
	// Os filtros da Mochila: a busca da grade e o chip de categoria.
	ItemSearch   string `json:"item_search"`
	ItemCategory string `json:"item_category"`
	// O diálogo do dinheiro: o modo (receber, gastar, corrigir) e o valor.
	TibarMode  string   `json:"tibar_mode"`
	TibarValue *float64 `json:"tibar_value"`
	// Os filtros do diálogo de adicionar do catálogo.
	CatalogSearch   string `json:"catalog_search"`
	CatalogCategory string `json:"catalog_category"`
	// Os campos de um item: quantidade, nome e espaços. `ItemQtd` serve ao
	// catálogo e à edição; os outros dois só ao item custom.
	ItemQtd   *int64   `json:"item_qty"`
	ItemName  *string  `json:"item_name"`
	ItemSlots *float64 `json:"item_slots"`
	// O que a MESA rolou ao usar um consumível. A ficha não rola por ninguém.
	ItemHPRoll *int64 `json:"item_roll_hp"`
	ItemMPRoll *int64 `json:"item_roll_mp"`
	// As melhorias escolhidas no diálogo, e o material. Lista e não par de
	// ids: são até quatro melhorias no mesmo item.
	ItemImprovements []string `json:"item_improvements"`
	ItemMaterial     string   `json:"item_material"`
	// Os degraus escolhidos ao entrar numa postura que escala com o nível, e a
	// busca da lista de poderes.
	PowerSteps  *int64 `json:"stance_degrees"`
	PowerSearch string `json:"power_search"`
	// Os atributos que a raça distribui, escolhidos no diálogo.
	RaceAttributes []string `json:"race_attributes"`
}

// augments traduz os seis sinais no que a validação espera.
//
// Zero e nulo saem da lista: "não escolhi" não é "escolhi zero pilhas", e um
// `stacks: 0` é recusado pelo servidor de propósito.
func (s Signals) augments() []sheet.AugmentPick {
	picks := []sheet.AugmentPick{}
	for i, value := range []*int{s.Aug0, s.Aug1, s.Aug2, s.Aug3, s.Aug4, s.Aug5} {
		if value == nil || *value <= 0 {
			continue
		}
		picks = append(picks, sheet.AugmentPick{AugmentIndex: i, Stacks: *value})
	}
	return picks
}

// sheetSignals lê o que o cliente mandou, caindo na URL quando não há sinal.
//
// A queda para a query serve a quem abre o endereço à mão — e serve à bancada,
// que precisa poder pedir uma aba filtrada sem montar um corpo de Datastar.
func sheetSignals(r *http.Request) Signals {
	signals := Signals{}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		signals = Signals{}
	}
	if signals.Search == nil {
		fromURL := r.URL.Query().Get("busca")
		signals.Search = &fromURL
	}
	// Os FILTROS caem para a query pela mesma razão da busca, e com uma a mais:
	// eles são o estado que faz sentido num endereço guardado — "a mochila,
	// filtrada por armas" é um lugar. O sinal do cliente vence quando existe.
	fillsURL(r, "poderbusca", &signals.PowerSearch)
	fillsURL(r, "itembusca", &signals.ItemSearch)
	fillsURL(r, "itemcategoria", &signals.ItemCategory)
	fillsURL(r, "magiabusca", &signals.SpellSearch)
	fillsURL(r, "magiacirculo", &signals.SpellCircle)
	fillsURL(r, "magiaescola", &signals.SpellSchool)
	return signals
}

// fillsURL põe o valor da query no campo quando o sinal veio vazio.
func fillsURL(r *http.Request, key string, field *string) {
	if *field == "" {
		*field = r.URL.Query().Get(key)
	}
}

// term é o termo já resolvido, para quem só quer o texto.
func (s Signals) term() string {
	if s.Search == nil {
		return ""
	}
	return *s.Search
}
