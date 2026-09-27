package catalog_test

import (
	"encoding/json"
	"sort"
	"testing"

	"t20engine/domain/catalog"
)

// A PÁGINA DE UM PODER CAI DENTRO DO CAPÍTULO QUE O IMPRIME.
//
// Os poderes gerais e os da Tormenta moram na seção "Poderes", p124-137 — os dois
// extremos conferidos no PDF: a p124 abre com "Poderes gerais podem ser
// escolhidos por qualquer personagem" e a p138 já é o capítulo de Equipamento.
//
// O guarda existe porque a ALE-407 achou a `Proficiência` citando a p32. A p32
// não tem verbete de poder nenhum: ela traz a linha da FICHA — "Proficiências.
// Os tipos de armas e armaduras que você sabe usar" —, e o verbete do poder está
// na p129. Uma página plausível e errada não estoura em lugar nenhum: ela só
// manda quem for conferir a regra para o capítulo errado.
//
// Ele NÃO substitui o `scripts/audit-powers.py`, que compara a página exata do
// verbete. Este aqui é a metade que roda na CI, onde o PDF do livro não existe
// por ser gitignorado.
func TestEveryPowerOfTheChapterCitesAPageInsideIt(t *testing.T) {
	const abre, fecha = 124, 137
	medidos := 0
	for _, recurso := range []string{"general-powers", "tormenta-powers"} {
		raw, ok := catalog.Resource(recurso)
		if !ok {
			t.Fatalf("catálogo %q ausente", recurso)
		}
		for _, poder := range lerOsPoderes(t, recurso, raw) {
			medidos++
			if poder.BookPage < abre || poder.BookPage > fecha {
				t.Errorf("%s/%s cita a p%d, e a seção que imprime estes poderes é a "+
					"p%d-%d. Confira em que página o VERBETE está — a citação não "+
					"estoura, ela só manda quem for conferir a regra para o capítulo "+
					"errado", recurso, poder.Name, poder.BookPage, abre, fecha)
			}
		}
	}
	// O CONTROLE: a varredura achou os poderes. Zero reprovados sobre zero
	// verbetes lidos tem no terminal a mesma cara de um capítulo inteiro válido.
	if medidos != 90 {
		t.Errorf("varri %d poderes e são 90 (68 gerais + 22 da Tormenta). Se a conta "+
			"mudou de propósito, mude este número junto", medidos)
	}
}

type poderDoCapitulo struct {
	Name     string `json:"name"`
	BookPage int    `json:"bookPage"`
}

// lerOsPoderes desempacota os dois recursos, que NÃO têm a mesma forma: o
// `general-powers.json` é uma LISTA e o `tormenta-powers.json` é um MAPA por id.
// Tentar um só dos dois formatos devolve lista vazia sem erro, e o caso passa
// verde sobre metade do capítulo.
func lerOsPoderes(t *testing.T, recurso string, raw []byte) []poderDoCapitulo {
	t.Helper()
	var lista []poderDoCapitulo
	if err := json.Unmarshal(raw, &lista); err == nil {
		return lista
	}
	var mapa map[string]poderDoCapitulo
	if err := json.Unmarshal(raw, &mapa); err != nil {
		t.Fatalf("%s não é lista nem mapa: %v", recurso, err)
	}
	chaves := make([]string, 0, len(mapa))
	for chave := range mapa {
		chaves = append(chaves, chave)
	}
	sort.Strings(chaves)
	for _, chave := range chaves {
		lista = append(lista, mapa[chave])
	}
	return lista
}
