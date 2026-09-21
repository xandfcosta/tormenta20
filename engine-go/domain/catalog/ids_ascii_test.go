package catalog

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode"
)

// NENHUM ID DE CATÁLOGO CARREGA ACENTO.
//
// A grafia irregular é o que faz toda cópia, toda URL e todo teste escrito de
// memória errarem NAQUELE id e em nenhum outro: entre 35 condições, 34
// normalizadas e um `enfeitiçado` com cedilha produziram uma lista de 34 escrita
// à mão ao lado das 35 do catálogo, e a condição que faltava dava 400 para o
// jogador e para o mestre.
//
// O que ele NÃO cobra: o `name` e a `description` são texto que uma pessoa LÊ, e
// lá o acento é
// obrigatório: "Enfeitiçado" continua com cedilha na tela. A regra é do
// IDENTIFICADOR, que é o que viaja em URL, em JSON gravado e em código.
//
// Ele varre todos os recursos do catálogo, e não só as condições: o próximo
// arquivo entra coberto sem ninguém acrescentar uma linha.
func TestNoCatalogIDIsAccented(t *testing.T) {
	var seen, names, withAccent int
	for _, name := range Resources() {
		raw, ok := Resource(name)
		if !ok {
			t.Fatalf("o recurso %q está no índice e não abre: o guarda não pode confiar no resto", name)
		}
		for _, id := range idsDe(t, name, raw) {
			// SÓ O QUE É SLUG. Metade dos catálogos é chaveada pelo NOME de
			// exibição — `origins` tem "Acólito", `gods` tem "Allihanna" —, e ali o
			// acento é o texto do livro, não grafia de identificador: sem a linha
			// abaixo, trinta nomes reprovam de uma vez.
			//
			// A linha entre os dois é mecânica e não uma lista: slug é minúsculo e
			// sem espaço. `enfeitiçado` é slug e por isso entra; "Acólito" tem
			// maiúscula e por isso é nome.
			if id != strings.ToLower(id) || strings.ContainsAny(id, " ") {
				names++
				continue
			}
			seen++
			if outside := naoASCII(id); outside != "" {
				withAccent++
				t.Errorf("%s: o id %q tem %q, e id de catálogo é ASCII.\n"+
					"    Ele viaja em URL, em JSON gravado e em código escrito de memória — a grafia irregular\n"+
					"    é o que faz errar NELE e em nenhum outro (ALE-122). O `name` continua acentuado.",
					name, id, outside)
			}
		}
	}
	// O DENOMINADOR: sem ele, um extrator que parou de achar ids e um catálogo
	// inteiramente ASCII dizem a mesma coisa.
	if seen < 150 {
		t.Fatalf("o guarda olhou só %d ids em %d recursos: o extrator parou de casar com a forma dos arquivos",
			seen, len(Resources()))
	}
	t.Logf("%d slugs olhados em %d recursos (%d chaves de NOME puladas), %d com acento",
		seen, len(Resources()), names, withAccent)
}

// idsDe tira os ids de um recurso, cobrindo as DUAS formas que o catálogo usa:
// lista de objetos com `id`, e mapa cujo id é a chave.
func idsDe(t *testing.T, name string, raw []byte) []string {
	t.Helper()
	var list []struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &list) == nil {
		ids := make([]string, 0, len(list))
		for _, e := range list {
			if e.ID != "" {
				ids = append(ids, e.ID)
			}
		}
		return ids
	}
	var board map[string]struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &board) == nil {
		ids := make([]string, 0, len(board))
		for key, e := range board {
			ids = append(ids, key)
			if e.ID != "" && e.ID != key {
				ids = append(ids, e.ID)
			}
		}
		return ids
	}
	// Recurso que não é nenhuma das duas formas não tem id a conferir — o
	// `options.json` é lista de nomes, e nome é texto que se lê.
	return nil
}

func naoASCII(id string) string {
	var outside []string
	for _, r := range id {
		if r > unicode.MaxASCII {
			outside = append(outside, fmt.Sprintf("%c", r))
		}
	}
	return strings.Join(outside, "")
}
