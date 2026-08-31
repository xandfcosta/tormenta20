package api

import (
	"net/http"

	"github.com/starfederation/datastar-go/datastar"
)

// OS SINAIS DA FICHA, lidos UMA VEZ por requisição (ALE-272, fatia 4).
//
// # Por que uma struct só, e por que uma leitura só
//
// `datastar.ReadSignals` CONSOME O CORPO num `POST`. Duas chamadas na mesma
// requisição deixam a segunda sem nada — e sem erro, porque um corpo vazio é um
// JSON ausente e não um JSON inválido. O gesto de criar um ofício precisa do
// nome digitado, e o redesenho precisa do termo de busca; escritos como duas
// leituras, o segundo a rodar receberia vazio e a lista voltaria sem filtro,
// como se a pessoa tivesse apagado a busca.
//
// A biblioteca ainda exige a ordem: `ReadSignals` ANTES do `NewSSE`, senão ela
// devolve "are you sure you created the SSE ***AFTER*** the ReadSignals?". Num
// `GET` os sinais vêm na consulta e a ordem não morde, que é justamente o que
// faria o defeito nascer no dia em que um gesto virasse `POST`.
//
// # As chaves são MINÚSCULAS, e não é estilo
//
// Chave de atributo é minusculada pelo HTML: um `data-bind:novaPericia` vira
// `data-bind:novapericia` e liga um sinal NOVO, deixando o que o servidor lê
// sempre vazio. Só o VALOR de um atributo preserva a caixa, que é por que o
// `$fichaAberta` do bestiário pode ser camelCase — ele só aparece dentro de
// expressões.

// fichaSignals é o que o cliente manda junto de qualquer gesto da ficha.
//
// Os campos são PONTEIROS para separar "não veio" de "veio vazio". Apagar a
// busca é gesto legítimo, e tratá-lo como ausência ressuscitaria o termo
// anterior — é a mesma decisão do `termoDoBuscador`.
type fichaSignals struct {
	Busca *string `json:"busca"`
	// NovaPericia e NovoAtributo são os dois campos do diálogo de ofício novo.
	NovaPericia  *string `json:"novapericia"`
	NovoAtributo *string `json:"novoatributo"`
	// Situacao é a CHAVE do condicional que o gesto quer alternar. Ela vem por
	// sinal e não pelo caminho porque é um encadeado com `::` e texto livre do
	// catálogo dentro — um `PathEscape` daquilo funciona e é ilegível no log.
	Situacao *string `json:"situacao"`
	// Aprimoramentos são as pilhas escolhidas no diálogo de conjurar, uma por
	// índice: `aug0`..`aug5`. Seis porque é o máximo do catálogo (Conjurar
	// Monstro), e nomes minúsculos pela regra de sempre.
	Aug0 *int `json:"aug0"`
	Aug1 *int `json:"aug1"`
	Aug2 *int `json:"aug2"`
	Aug3 *int `json:"aug3"`
	Aug4 *int `json:"aug4"`
	Aug5 *int `json:"aug5"`
	// Os filtros do catálogo de magias. Minúsculos como todos os outros.
	MagiaBusca   string `json:"magiabusca"`
	MagiaCirculo string `json:"magiacirculo"`
	MagiaEscola  string `json:"magiaescola"`
	// Os filtros da Mochila: a busca da grade e o chip de categoria.
	ItemBusca     string `json:"itembusca"`
	ItemCategoria string `json:"itemcategoria"`
	// O diálogo do dinheiro: o modo (receber, gastar, corrigir) e o valor.
	TibarModo  string   `json:"tibarmodo"`
	TibarValor *float64 `json:"tibarvalor"`
}

// osAprimoramentos traduz os seis sinais no que a validação espera.
//
// Zero e nulo saem da lista: "não escolhi" não é "escolhi zero pilhas", e um
// `stacks: 0` é recusado pelo servidor de propósito.
func (s fichaSignals) osAprimoramentos() []augmentPick {
	picks := []augmentPick{}
	for i, valor := range []*int{s.Aug0, s.Aug1, s.Aug2, s.Aug3, s.Aug4, s.Aug5} {
		if valor == nil || *valor <= 0 {
			continue
		}
		picks = append(picks, augmentPick{AugmentIndex: i, Stacks: *valor})
	}
	return picks
}

// osSinaisDaFicha lê o que o cliente mandou, caindo na URL quando não há sinal.
//
// A queda para a query serve a quem abre o endereço à mão — e serve à bancada,
// que precisa poder pedir uma aba filtrada sem montar um corpo de Datastar.
func osSinaisDaFicha(r *http.Request) fichaSignals {
	sinais := fichaSignals{}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		sinais = fichaSignals{}
	}
	if sinais.Busca == nil {
		daURL := r.URL.Query().Get("busca")
		sinais.Busca = &daURL
	}
	// Os FILTROS caem para a query pela mesma razão da busca, e com uma a mais:
	// eles são o estado que faz sentido num endereço guardado — "a mochila,
	// filtrada por armas" é um lugar. O sinal do cliente vence quando existe.
	preencheDaURL(r, "itembusca", &sinais.ItemBusca)
	preencheDaURL(r, "itemcategoria", &sinais.ItemCategoria)
	preencheDaURL(r, "magiabusca", &sinais.MagiaBusca)
	preencheDaURL(r, "magiacirculo", &sinais.MagiaCirculo)
	preencheDaURL(r, "magiaescola", &sinais.MagiaEscola)
	return sinais
}

// preencheDaURL põe o valor da query no campo quando o sinal veio vazio.
func preencheDaURL(r *http.Request, chave string, campo *string) {
	if *campo == "" {
		*campo = r.URL.Query().Get(chave)
	}
}

// aBusca é o termo já resolvido, para quem só quer o texto.
func (s fichaSignals) aBusca() string {
	if s.Busca == nil {
		return ""
	}
	return *s.Busca
}
