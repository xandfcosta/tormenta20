package catalog

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode"
)

// NENHUM ID DE CATÁLOGO CARREGA ACENTO (ALE-152).
//
// # Por que isto não é preciosismo
//
// Dos 35 ids das condições, 34 nasceram normalizados — `caido` e não "caído",
// `desprevenido`, `esmorecido` — e UM não: `enfeitiçado`, com cedilha. É a
// grafia irregular que faz toda cópia, toda URL e todo teste escrito de memória
// errarem NAQUELE id e em nenhum outro.
//
// E ele já quebrou: a API tinha uma lista de 34 ids escrita à mão ao lado das 35
// do catálogo, e a que faltava era justo essa — aplicá-la dava 400 para o
// jogador e para o mestre (ALE-122). O teste que a issue escreveu para reproduzir
// também errou, escrevendo `enfeiticado`.
//
// # O que ele NÃO cobra
//
// O `name` e a `description` são texto que uma pessoa LÊ, e lá o acento é
// obrigatório: "Enfeitiçado" continua com cedilha na tela. A regra é do
// IDENTIFICADOR, que é o que viaja em URL, em JSON gravado e em código.
//
// Ele varre todos os recursos do catálogo, e não só as condições: o próximo
// arquivo entra coberto sem ninguém acrescentar uma linha.
func TestNoCatalogIDIsAccented(t *testing.T) {
	var vistos, nomes, comAcento int
	for _, nome := range Resources() {
		bruto, ok := Resource(nome)
		if !ok {
			t.Fatalf("o recurso %q está no índice e não abre: o guarda não pode confiar no resto", nome)
		}
		for _, id := range idsDe(t, nome, bruto) {
			// SÓ O QUE É SLUG. Metade dos catálogos é chaveada pelo NOME de
			// exibição — `origins` tem "Acólito", `gods` tem "Allihanna" —, e ali o
			// acento é o texto do livro, não grafia de identificador. A primeira
			// versão deste guarda reprovou 30 desses de uma vez.
			//
			// A linha entre os dois é mecânica e não uma lista: slug é minúsculo e
			// sem espaço. `enfeitiçado` é slug e por isso entra; "Acólito" tem
			// maiúscula e por isso é nome.
			if id != strings.ToLower(id) || strings.ContainsAny(id, " ") {
				nomes++
				continue
			}
			vistos++
			if fora := naoASCII(id); fora != "" {
				comAcento++
				t.Errorf("%s: o id %q tem %q, e id de catálogo é ASCII.\n"+
					"    Ele viaja em URL, em JSON gravado e em código escrito de memória — a grafia irregular\n"+
					"    é o que faz errar NELE e em nenhum outro (ALE-122). O `name` continua acentuado.",
					nome, id, fora)
			}
		}
	}
	// O DENOMINADOR: sem ele, um extrator que parou de achar ids e um catálogo
	// inteiramente ASCII dizem a mesma coisa.
	if vistos < 150 {
		t.Fatalf("o guarda olhou só %d ids em %d recursos: o extrator parou de casar com a forma dos arquivos",
			vistos, len(Resources()))
	}
	t.Logf("%d slugs olhados em %d recursos (%d chaves de NOME puladas), %d com acento",
		vistos, len(Resources()), nomes, comAcento)
}

// idsDe tira os ids de um recurso, cobrindo as DUAS formas que o catálogo usa:
// lista de objetos com `id`, e mapa cujo id é a chave.
func idsDe(t *testing.T, nome string, bruto []byte) []string {
	t.Helper()
	var lista []struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(bruto, &lista) == nil {
		ids := make([]string, 0, len(lista))
		for _, e := range lista {
			if e.ID != "" {
				ids = append(ids, e.ID)
			}
		}
		return ids
	}
	var mapa map[string]struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(bruto, &mapa) == nil {
		ids := make([]string, 0, len(mapa))
		for chave, e := range mapa {
			ids = append(ids, chave)
			if e.ID != "" && e.ID != chave {
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
	var fora []string
	for _, r := range id {
		if r > unicode.MaxASCII {
			fora = append(fora, fmt.Sprintf("%c", r))
		}
	}
	return strings.Join(fora, "")
}
