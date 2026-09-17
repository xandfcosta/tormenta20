package master

import (
	"strings"
	"t20engine/domain/book"
	"t20engine/serve/web/bookui"
	"testing"
)

// Os guardas dos CATÁLOGOS.

// AMOSTRAGEM e não enumeração: UM guarda percorre o DADO e cobra rótulo para
// cada valor distinto que encontra, então ele cresce sozinho quando o livro ganha
// uma escola de magia nova. Um guarda por tabela deixa passar o valor que
// ninguém lembrou de exemplificar — foi assim que 27 criaturas passaram a mostrar
// o dado cru com sete guardas no ar, todos usando o MESMO valor de exemplo.
//
// Provado VERMELHO removendo a linha "catalyst" do `book.categoryLabel`.
func TestEveryBookValueHasALabel(t *testing.T) {
	a := book.Catalogs()
	if len(a.Magias) == 0 || len(a.Itens) == 0 {
		t.Fatal("catálogo vazio: não há o que medir, e verde aqui não valeria nada")
	}

	cobra := func(campo string, valores map[string]int, rotulo func(string) string) {
		for v, quantos := range valores {
			if v == "" {
				continue
			}
			if rotulo(v) == v {
				t.Errorf("%s %q (%d entradas) sai na tela como o dado cru", campo, v, quantos)
			}
		}
	}

	execucoes, alcances := map[string]int{}, map[string]int{}
	for _, m := range a.Magias {
		execucoes[m.Execution]++
		alcances[m.Range]++
	}
	categorias := map[string]int{}
	for _, i := range a.Itens {
		categorias[i.Category]++
	}

	cobra("execução", execucoes, book.CastingName)
	cobra("alcance", alcances, book.RangeName)
	cobra("categoria", categorias, book.CategoryName)
}

// "luz cur" só casa com o que carrega as duas coisas. É a regra que separa esta
// busca da das outras cenas.
func TestTheSearchRequiresEveryTerm(t *testing.T) {
	campos := []string{"Curar Ferimentos", "Restaura pontos de vida ao toque."}
	casos := []struct {
		busca string
		casa  bool
		por   string
	}{
		{"curar", true, "termo único que existe"},
		{"curar toque", true, "dois termos, um no nome e outro na descrição"},
		{"curar dragão", false, "o segundo termo não existe em campo nenhum"},
		{"", true, "não digitar não é filtrar"},
		{"   ", true, "só espaço também não é filtrar"},
		{"CURAR", true, "caixa não separa"},
		{"ferimento", true, "prefixo casa por substring"},
		// A diferença para o `search.Matches`: LÁ isto casaria por subsequência
		// (c-u-r-r nas letras de "Curar"), e uma consulta de regra que devolve
		// quase-acerto no meio da sessão parece defeito.
		{"crr", false, "subsequência NÃO casa: esta busca não é tolerante a typo"},
	}
	for _, c := range casos {
		if got := matchesAllTerms(campos, c.busca); got != c.casa {
			t.Errorf("casaTodosOsTermos(%q) = %v, quero %v — %s", c.busca, got, c.casa, c.por)
		}
	}
}

// Ninguém digita til no meio da sessão.
func TestAnAccentDoesNotSplitTheSearch(t *testing.T) {
	campos := []string{"Ilusão Lacerante", "Cria uma imagem que fere."}
	for _, busca := range []string{"ilusao", "Ilusão", "ILUSAO", "imagem"} {
		if !matchesAllTerms(campos, busca) {
			t.Errorf("%q não casou com %v", busca, campos)
		}
	}
}

// Buscar varre os catálogos, e não só a aba aberta: filtrar só a aba ativa faz
// "bola de fogo" digitado em Condições dizer "nada encontrado" com a magia
// existindo. A aba é para NAVEGAR sem termo; com termo, o assunto é o acervo.
func TestSearchingSweepsTheFourCatalogs(t *testing.T) {
	v := loadCollection(collectionCriteria{Term: "fogo", Aba: "condicoes"}, bookui.BookAddress{})
	if !v.Searching() {
		t.Fatal("a cena não se considerou em busca")
	}
	achouMagia := false
	for _, g := range v.Grupos {
		if g.Rotulo == "Magias" && len(g.Magias) > 0 {
			achouMagia = true
		}
		if g.Count() == 0 {
			t.Errorf("o grupo %q veio vazio — cabeçalho sobre nada é ruído", g.Rotulo)
		}
	}
	if !achouMagia {
		t.Error("buscar 'fogo' com a aba em Condições não achou magia nenhuma (ALE-22)")
	}
}

// Sem termo a cena é um catálogo por vez.
func TestWithoutASearchOnlyTheOpenTabShows(t *testing.T) {
	a := book.Catalogs()
	for _, caso := range []struct {
		aba     string
		rotulo  string
		quantas int
	}{
		{"condicoes", "Condições", len(a.Condicoes)},
		{"magias", "Magias", len(a.Magias)},
		{"poderes", "Poderes", len(a.Poderes)},
		{"itens", "Itens", len(a.Itens)},
	} {
		t.Run(caso.aba, func(t *testing.T) {
			v := loadCollection(collectionCriteria{Aba: caso.aba}, bookui.BookAddress{})
			if len(v.Grupos) != 1 {
				t.Fatalf("%d grupos, quero 1", len(v.Grupos))
			}
			if v.Grupos[0].Rotulo != caso.rotulo {
				t.Errorf("grupo %q, quero %q", v.Grupos[0].Rotulo, caso.rotulo)
			}
			if v.Achados != caso.quantas {
				t.Errorf("%d entradas, quero as %d do catálogo inteiro", v.Achados, caso.quantas)
			}
		})
	}
}

// O `?aba=` é endereço e alguém o digita errado — cair em tela vazia leria como
// catálogo quebrado.
func TestAnInventedTabFallsBackToTheFirst(t *testing.T) {
	v := loadCollection(collectionCriteria{Term: "", Aba: "grimorios-proibidos"}, bookui.BookAddress{})
	if v.Aba != "condicoes" {
		t.Errorf("aba %q, quero cair em condicoes", v.Aba)
	}
	if v.Achados == 0 {
		t.Error("aba inventada devolveu tela vazia")
	}
}

// Os poderes vêm dos três catálogos achatados, com a fonte preservada.
//
// O achatamento é o ponto da ferramenta — o livro espalha poder por três
// lugares e o mestre quer uma lista só —, e o que ele não pode perder é DE ONDE
// veio. Sem a fonte, "Ataque Poderoso" não diz se é poder de classe ou geral.
func TestPowersComeFromTheThreeCatalogs(t *testing.T) {
	fontes := map[string]int{}
	for _, p := range book.Catalogs().Poderes {
		switch {
		case strings.HasPrefix(p.ID, "general."):
			fontes["geral"]++
		case strings.HasPrefix(p.ID, "divino."):
			fontes["divino"]++
		default:
			fontes["classe"]++
		}
		if p.Fonte == "" {
			t.Fatalf("o poder %q não diz de onde veio", p.Name)
		}
	}
	for _, esperada := range []string{"classe", "geral", "divino"} {
		if fontes[esperada] == 0 {
			t.Errorf("nenhum poder de %q — um dos três catálogos não entrou", esperada)
		}
	}
	// O número dos DIVINOS está preso porque a lacuna é INVISÍVEL: o cartão do
	// deus mostra os poderes como texto, e ler a lista errada (o `granted-powers`,
	// que é metade dos nomes) deixa metade sem virar elo sem ninguém ver. São 72
	// porque os 80 do `divine-powers` juntam por nome os que vários deuses
	// concedem — "Coragem Total" aparece quatro vezes.
	if fontes["divino"] != 72 {
		t.Errorf("%d poderes divinos no acervo — eram 72 quando isto foi escrito", fontes["divino"])
	}
}

// ── a cena pelo fio ──────────────────────────────────────────────────────────

// `?busca=` é endereço, e um link colado no chat da mesa tem de abrir já
// filtrado.
func TestTheSearchInTheUrlHoldsOnAColdLoad(t *testing.T) {
	corpo := pedeNaCena(t, "/mestre/condicoes?busca=fogo").Body.String()

	// A prova de que a página abriu FILTRADA é o que ela MOSTRA, e não uma
	// contagem colhida do `loadCollection` — a mesma função que a desenhou. Um
	// erro na busca sairia dos DOIS lados e o guarda ficaria verde.
	if !strings.Contains(corpo, "Bola de Fogo") {
		t.Error("buscar fogo na cena das condições não trouxe a magia — a busca não varreu os oito")
	}
	// O CONTROLE pelo outro lado: uma condição que NÃO casa com o termo não
	// pode estar na página. Sem ele, "achou a magia" também passaria numa cena
	// que ignorou o filtro e desenhou tudo.
	if strings.Contains(corpo, "Abalado") {
		t.Error("a página trouxe uma condição que não casa com o termo: ela não abriu filtrada")
	}
}

// AMOSTRAGEM e não enumeração: o teste percorre `collectionTabs`, então a aba que
// entrar amanhã já nasce medida.
//
// O CONTROLE é a segunda metade: sem livro configurado, a mesma cena não pode
// trazer link nenhum. Sem ele, "achei `#page=`" seria verdade sobre um endereço
// que a cena escreve de qualquer jeito.
func TestEveryCollectionTabOffersTheBook(t *testing.T) {
	for _, aba := range collectionTabs {
		corpo := pedeNaCenaComLivro(t, "/mestre/"+aba.ID).Body.String()
		if !strings.Contains(corpo, "/livro/ler?p=") {
			t.Errorf("a aba %q não oferece o livro em nenhuma entrada", aba.Rotulo)
		}
		if !strings.Contains(corpo, "Abrir o livro na página") {
			t.Errorf("a aba %q tem o endereço mas não o título que diz o que ele faz", aba.Rotulo)
		}
	}

	// E o CONTROLE pelo outro lado da porta: sem `LIVRO_PDF` não há link.
	sem := pedeNaCena(t, "/mestre/condicoes").Body.String()
	if strings.Contains(sem, "/livro/ler") {
		t.Error("sem LIVRO_PDF a cena linkou um livro que não é servido")
	}
	// E a página continua ESCRITA: o mestre com o livro de papel usa o número.
	if !strings.Contains(sem, "p394") {
		t.Error("sem livro a página impressa sumiu do cartão — ela não depende do PDF")
	}
}
