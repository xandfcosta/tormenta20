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
// Lê `origins-source` e não `origins`: são dois arquivos sobre as mesmas 35
// origens, com campos diferentes — `origins` tem os BENEFÍCIOS derivados (que a
// aba de Poderes já usa, em `Origins`) e `origins-source` é a transcrição do
// LIVRO, com os itens, as perícias e a página. Ler o arquivo errado devolve mapa
// vazio em silêncio, que é a razão de o guarda de varredura afirmar o
// denominador.
//
// Os dois se chamavam `origins` e `origens` até a ALE-301, e o par era a
// armadilha que este parágrafo existe para desarmar: um acento decidindo qual
// dos dois catálogos você leu. O sufixo `-source` diz QUAL é a fonte, que é a
// pergunta que o leitor tinha — decisão do dono.
var (
	originItemsOnce  sync.Once
	originItemsIndex map[string][]string
)

func OriginItemsByName() map[string][]string {
	originItemsOnce.Do(func() {
		originItemsIndex = map[string][]string{}
		bruto, ok := catalog.Resource("origins-source")
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
