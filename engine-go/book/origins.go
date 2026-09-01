package book

import (
	"encoding/json"
	"sync"

	"t20engine/catalog"
)

// O ÍNDICE DE ITENS DE ORIGEM, por nome de origem.
//
// Ele morava na forja e veio para cá na ALE-278, apontado pelo guarda de
// fronteira da cena na primeira execução dele: a forja importava `catalog`
// direto, contornando a camada tipada que existe para isso.

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

func OriginItemsByName() map[string][]string {
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
