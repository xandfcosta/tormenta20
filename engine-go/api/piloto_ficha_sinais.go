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
	return sinais
}

// aBusca é o termo já resolvido, para quem só quer o texto.
func (s fichaSignals) aBusca() string {
	if s.Busca == nil {
		return ""
	}
	return *s.Busca
}
