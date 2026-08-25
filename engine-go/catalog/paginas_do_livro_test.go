package catalog

import (
	"encoding/json"
	"io/fs"
	"testing"
)

// O guarda das PÁGINAS DO LIVRO (ALE-264).
//
// As 745 páginas que o `scripts/paginas-do-livro.py` derivou vieram do Índice
// Remissivo do próprio livro, e cada uma foi conferida contra o texto da página
// antes de entrar. Este teste não repete a conferência — ele não tem o PDF, que
// vive fora do repositório e é ignorado pelo git.
//
// O que ele prende é o que sobrevive sem o livro na mão: FAIXA. Foi assim que o
// script foi pego devolvendo a página 396 para uma condição — 396 é a primeira
// página do índice remissivo, onde todo nome aparece porque aquilo é uma lista
// de nomes. A conferência passava e o botão abriria o índice.

// ultimaDeConteudo é a última página impressa antes do Índice Remissivo.
//
// Medido no PDF da casa (407 páginas do arquivo, abertura 6): o índice começa na
// impressa 396. Página maior que isto não é regra nenhuma — é o índice, a ficha
// em branco ou a contracapa.
const ultimaDeConteudo = 395

// TestNenhumaPaginaAponta ParaOIndice varre TODO catálogo embutido por
// AMOSTRAGEM: quem passar a ter `bookPage` amanhã nasce medido, sem entrada
// nova aqui.
func TestNenhumaPaginaCaiForaDoConteudo(t *testing.T) {
	arquivos, err := fs.Glob(files, "data/*.json")
	if err != nil {
		t.Fatalf("listar catálogos: %v", err)
	}
	if len(arquivos) < 10 {
		t.Fatalf("só %d catálogos embutidos — o guarda mediria quase nada", len(arquivos))
	}

	comPagina := 0
	for _, arquivo := range arquivos {
		bruto, err := files.ReadFile(arquivo)
		if err != nil {
			t.Fatalf("%s: %v", arquivo, err)
		}
		for _, entrada := range entradasComNome(t, arquivo, bruto) {
			if entrada.Pagina == 0 {
				continue
			}
			comPagina++
			if entrada.Pagina < 1 || entrada.Pagina > ultimaDeConteudo {
				t.Errorf("%s → %q: p%d fora do conteúdo (1–%d) — p396+ é o índice remissivo",
					arquivo, entrada.Nome, entrada.Pagina, ultimaDeConteudo)
			}
		}
	}
	// O CONTROLE: sem ele, apagar o `bookPage` de todo mundo passaria verde.
	if comPagina < 700 {
		t.Errorf("só %d entradas com página — eram 745 quando isto foi escrito", comPagina)
	}
}

// TestTodaCondicaoSabeSuaPagina: as 35 estão na mesma lista do apêndice, então
// "algumas sem página" é defeito e não lacuna do livro.
func TestTodaCondicaoSabeSuaPagina(t *testing.T) {
	bruto, ok := Resource("conditions")
	if !ok {
		t.Fatal("catálogo de condições ausente")
	}
	var porID map[string]struct {
		Name     string `json:"name"`
		BookPage int    `json:"bookPage"`
	}
	if err := json.Unmarshal(bruto, &porID); err != nil {
		t.Fatalf("condições: %v", err)
	}
	if len(porID) < 30 {
		t.Fatalf("só %d condições — o guarda mediria outra coisa", len(porID))
	}
	for _, c := range porID {
		if c.BookPage == 0 {
			t.Errorf("a condição %q ficou sem página do livro", c.Name)
		}
	}
}

type entradaComPagina struct {
	Nome   string
	Pagina int
}

// entradasComNome lê um catálogo nas DUAS formas em que eles existem — lista e
// mapa por id — e devolve só o que interessa aqui.
//
// Catálogo com outra forma (as tabelas do mestre, as ativações aninhadas) não é
// erro: ele simplesmente não tem entrada com página para medir.
func entradasComNome(t *testing.T, arquivo string, bruto []byte) []entradaComPagina {
	t.Helper()
	type crua struct {
		Name     string `json:"name"`
		ID       string `json:"id"`
		BookPage int    `json:"bookPage"`
	}
	converte := func(lista []crua) []entradaComPagina {
		fora := make([]entradaComPagina, 0, len(lista))
		for _, c := range lista {
			nome := c.Name
			if nome == "" {
				nome = c.ID
			}
			fora = append(fora, entradaComPagina{Nome: nome, Pagina: c.BookPage})
		}
		return fora
	}

	var lista []crua
	if err := json.Unmarshal(bruto, &lista); err == nil {
		return converte(lista)
	}
	var mapa map[string]crua
	if err := json.Unmarshal(bruto, &mapa); err == nil {
		fora := make([]crua, 0, len(mapa))
		for _, v := range mapa {
			fora = append(fora, v)
		}
		return converte(fora)
	}
	return nil
}

// TestTodaClasseSabeSuaPagina: o catálogo de classes nasceu na ALE-264 com três
// campos, e a PÁGINA é o motivo dele existir — sem ela, ele não teria por quê.
func TestTodaClasseSabeSuaPagina(t *testing.T) {
	bruto, ok := Resource("classes")
	if !ok {
		t.Fatal("catálogo de classes ausente — a aba nasce vazia e nada estoura")
	}
	var classes []struct {
		Name     string `json:"name"`
		BookPage int    `json:"bookPage"`
	}
	if err := json.Unmarshal(bruto, &classes); err != nil {
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

// TestOsTresBlocosQueAbremUmaPaginaAdiante (ALE-264).
//
// A TRAP da tabela, e não a tabela inteira: estas três criaturas tinham no
// catálogo a página que as CITA, não a que abre o bloco delas. A p289 fala de
// "lobos-das-cavernas" no texto corrido, e a conferência por substring aprovava
// — o botão abria uma página antes, no mesmo capítulo, parecendo certo.
//
// Quem consertou foi a assinatura `<nome> nd <valor>`, que é como o livro
// imprime o começo de todo bloco. Fica preso aqui porque uma regeneração
// desatenta do catálogo os traria de volta em silêncio.
func TestOsTresBlocosQueAbremUmaPaginaAdiante(t *testing.T) {
	bruto, ok := Resource("bestiary")
	if !ok {
		t.Fatal("bestiário ausente")
	}
	var criaturas []struct {
		Name     string `json:"name"`
		BookPage int    `json:"bookPage"`
	}
	if err := json.Unmarshal(bruto, &criaturas); err != nil {
		t.Fatalf("bestiário: %v", err)
	}
	esperado := map[string]int{"Lobo": 290, "Troll": 308, "Trog": 291}
	visto := 0
	for _, c := range criaturas {
		pagina, cobrada := esperado[c.Name]
		if !cobrada {
			continue
		}
		visto++
		if c.BookPage != pagina {
			t.Errorf("%s: p%d — o bloco dele abre na p%d", c.Name, c.BookPage, pagina)
		}
	}
	if visto != len(esperado) {
		t.Errorf("só %d das %d criaturas cobradas existem no catálogo", visto, len(esperado))
	}
}
