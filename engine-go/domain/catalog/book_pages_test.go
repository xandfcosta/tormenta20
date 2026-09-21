package catalog

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
)

// O guarda das PÁGINAS DO LIVRO.
//
// Ele não repete a conferência contra o texto da página — não tem o PDF, que
// vive fora do repositório. O que ele prende é o que sobrevive sem o livro na
// mão: FAIXA. Foi assim que o extrator foi pego devolvendo a p396 para uma
// condição — 396 é a primeira página do índice remissivo, onde todo nome
// aparece porque aquilo é uma lista de nomes, e a conferência passava.

// ultimaDeConteudo é a última página impressa antes do Índice Remissivo.
//
// Medido no PDF da casa (407 páginas do arquivo, abertura 6): o índice começa na
// impressa 396. Página maior que isto não é regra nenhuma — é o índice, a ficha
// em branco ou a contracapa.
const ultimaDeConteudo = 395

// Varre TODO catálogo embutido por AMOSTRAGEM: quem passar a ter `bookPage`
// amanhã nasce medido, sem entrada nova aqui.
func TestNoPageFallsOutsideTheContent(t *testing.T) {
	fileNames, err := fs.Glob(files, "data/*.json")
	if err != nil {
		t.Fatalf("listar catálogos: %v", err)
	}
	if len(fileNames) < 10 {
		t.Fatalf("só %d catálogos embutidos — o guarda mediria quase nada", len(fileNames))
	}

	withPage := 0
	for _, file := range fileNames {
		raw, err := files.ReadFile(file)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for _, entry := range entradasComNome(t, file, raw) {
			if entry.Page == 0 {
				continue
			}
			withPage++
			if entry.Page < 1 || entry.Page > ultimaDeConteudo {
				t.Errorf("%s → %q: p%d fora do conteúdo (1–%d) — p396+ é o índice remissivo",
					file, entry.Name, entry.Page, ultimaDeConteudo)
			}
		}
	}
	// O CONTROLE: sem ele, apagar o `bookPage` de todo mundo passaria verde.
	if withPage < 700 {
		t.Errorf("só %d entradas com página — eram 745 quando isto foi escrito", withPage)
	}
}

// As condições estão todas na mesma lista do apêndice, então "algumas sem
// página" é defeito, e não lacuna do livro.
func TestEveryConditionKnowsItsPage(t *testing.T) {
	raw, ok := Resource("conditions")
	if !ok {
		t.Fatal("catálogo de condições ausente")
	}
	var byID map[string]struct {
		Name     string `json:"name"`
		BookPage int    `json:"bookPage"`
	}
	if err := json.Unmarshal(raw, &byID); err != nil {
		t.Fatalf("condições: %v", err)
	}
	if len(byID) < 30 {
		t.Fatalf("só %d condições — o guarda mediria outra coisa", len(byID))
	}
	for _, c := range byID {
		if c.BookPage == 0 {
			t.Errorf("a condição %q ficou sem página do livro", c.Name)
		}
	}
}

type entradaComPagina struct {
	Name string
	Page int
}

// entradasComNome lê um catálogo nas DUAS formas em que eles existem — lista e
// mapa por id — e devolve só o que interessa aqui.
//
// Catálogo com outra forma (as tabelas do mestre, as ativações aninhadas) não é
// erro: ele simplesmente não tem entrada com página para medir.
func entradasComNome(t *testing.T, file string, raw []byte) []entradaComPagina {
	t.Helper()
	type crua struct {
		Name     string `json:"name"`
		ID       string `json:"id"`
		BookPage int    `json:"bookPage"`
	}
	converts := func(list []crua) []entradaComPagina {
		outside := make([]entradaComPagina, 0, len(list))
		for _, c := range list {
			name := c.Name
			if name == "" {
				name = c.ID
			}
			outside = append(outside, entradaComPagina{Name: name, Page: c.BookPage})
		}
		return outside
	}

	var items []crua
	if err := json.Unmarshal(raw, &items); err == nil {
		return converts(items)
	}
	var board map[string]crua
	if err := json.Unmarshal(raw, &board); err == nil {
		outside := make([]crua, 0, len(board))
		for _, v := range board {
			outside = append(outside, v)
		}
		return converts(outside)
	}
	return nil
}

// A PÁGINA é o motivo de o catálogo de classes existir — sem ela, ele não teria
// por quê.
func TestEveryClassKnowsItsPage(t *testing.T) {
	raw, ok := Resource("classes")
	if !ok {
		t.Fatal("catálogo de classes ausente — a aba nasce vazia e nada estoura")
	}
	var classes []struct {
		Name     string `json:"name"`
		BookPage int    `json:"bookPage"`
	}
	if err := json.Unmarshal(raw, &classes); err != nil {
		t.Fatalf("classes: %v", err)
	}
	if len(classes) != 14 {
		t.Errorf("%d classes — o livro tem 14", len(classes))
	}
	for _, c := range classes {
		if c.BookPage == 0 {
			t.Errorf("a classe %q ficou sem página do livro", c.Name)
		}
	}
}

// A ARMADILHA da tabela, e não a tabela inteira: estas três criaturas caem com
// facilidade na página que as CITA em vez da que abre o bloco delas — a p289
// fala de "lobos-das-cavernas" no texto corrido, e uma conferência por substring
// aprova, com o botão abrindo uma página antes, no mesmo capítulo.
//
// Quem separa é a assinatura `<nome> nd <valor>`, que é como o livro imprime o
// começo de todo bloco. Uma regeneração desatenta do catálogo traz as três de
// volta em silêncio.
func TestTheThreeBlocksThatOpenOnePageLater(t *testing.T) {
	raw, ok := Resource("bestiary")
	if !ok {
		t.Fatal("bestiário ausente")
	}
	var creatures []struct {
		Name     string `json:"name"`
		BookPage int    `json:"bookPage"`
	}
	if err := json.Unmarshal(raw, &creatures); err != nil {
		t.Fatalf("bestiário: %v", err)
	}
	want := map[string]int{"Lobo": 290, "Troll": 308, "Trog": 291}
	seen := 0
	for _, c := range creatures {
		page, charged := want[c.Name]
		if !charged {
			continue
		}
		seen++
		if c.BookPage != page {
			t.Errorf("%s: p%d — o bloco dele abre na p%d", c.Name, c.BookPage, page)
		}
	}
	if seen != len(want) {
		t.Errorf("só %d das %d criaturas cobradas existem no catálogo", seen, len(want))
	}
}

// Os tipos de efeito e as escolas de magia são EXTRAÍDOS do PDF pelo
// `scripts/book-pages.py`, e as duas formas de sujeira que o extrator já deixou
// passar estão aqui — as duas vistas na tela, nenhuma detectada por ele:
//
//   - o HÍFEN de quebra de linha, que virou "impede convoca- ções";
//   - a MOBÍLIA da página colada no último verbete: uma citação decorativa na
//     p172 ("Uma magia será tão poderosa quanto seu conjurador") e a seção
//     inteira de "Habilidades Gerais" na p228, que deu 1.800 caracteres.
//
// Este teste não tem o livro — ele mora fora do repositório. O que ele mede é a
// FORMA do que foi extraído, que é o que sobrevive sem o PDF na mão.
func TestNoExtractedEntryCarriesPageDirt(t *testing.T) {
	for _, resource := range []string{"effect-types", "spell-schools"} {
		raw, ok := Resource(resource)
		if !ok {
			t.Errorf("catálogo %q ausente", resource)
			continue
		}
		var entries []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(raw, &entries); err != nil {
			t.Errorf("%s: %v", resource, err)
			continue
		}
		if len(entries) < 8 {
			t.Errorf("%s tem só %d verbetes", resource, len(entries))
		}
		for _, v := range entries {
			if strings.Contains(v.Description, "- ") {
				t.Errorf("%s → %q: hífen de quebra solto no meio do texto", resource, v.Name)
			}
			// Uma definição do livro é uma ou duas frases. Passando disto, o
			// extrator comeu a seção vizinha.
			if len(v.Description) > 700 {
				t.Errorf("%s → %q: %d caracteres, a página vizinha entrou junto",
					resource, v.Name, len(v.Description))
			}
			if !strings.HasSuffix(strings.TrimSpace(v.Description), ".") {
				t.Errorf("%s → %q: a definição não termina em ponto — foi cortada no meio",
					resource, v.Name)
			}
		}
	}
}
