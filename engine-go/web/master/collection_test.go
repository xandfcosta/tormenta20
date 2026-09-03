package master

import (
	"strings"
	"t20engine/book"
	"t20engine/web/bookui"
	"testing"
)

// Os guardas dos CATÁLOGOS (ALE-258).

// TestEveryBookValueHasALabel é o guarda GENÉRICO que a fatia do bestiário me
// ensinou a escrever, e é a resposta certa ao defeito de lá.
//
// Naquela fatia um `sed` trocou a chave de um mapa de rótulos e 27 criaturas
// passaram a mostrar o dado cru; sete guardas não pegaram porque todos usavam o
// MESMO valor de exemplo. O remendo de lá foi um guarda por tabela — enumeração,
// que é o regime que o CLAUDE.md chama de remendo.
//
// Aqui a forma é outra: UM guarda percorre o DADO e cobra rótulo para cada valor
// distinto que ele encontra. Ele cresce sozinho quando o livro ganha uma escola
// de magia nova, e não depende de alguém lembrar de acrescentar um caso.
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

// TestTheSearchRequiresEveryTerm: "luz cur" só casa com o que carrega as duas
// coisas. É a regra que separa esta busca da das outras cenas.
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

// TestAnAccentDoesNotSplitTheSearch: ninguém digita til no meio da sessão.
func TestAnAccentDoesNotSplitTheSearch(t *testing.T) {
	campos := []string{"Ilusão Lacerante", "Cria uma imagem que fere."}
	for _, busca := range []string{"ilusao", "Ilusão", "ILUSAO", "imagem"} {
		if !matchesAllTerms(campos, busca) {
			t.Errorf("%q não casou com %v", busca, campos)
		}
	}
}

// TestSearchingSweepsTheFourCatalogs, e não só a aba aberta.
//
// É a ALE-22: a versão em React filtrava só a aba ativa, então "bola de fogo"
// digitado na aba Condições dizia "nada encontrado" com a magia existindo. A
// aba é para NAVEGAR sem termo; com termo, o assunto é o acervo inteiro.
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

// TestWithoutASearchOnlyTheOpenTabShows: sem termo a cena é um catálogo por vez.
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

// TestAnInventedTabFallsBackToTheFirst: o `?aba=` é endereço e alguém o digita errado
// — cair em tela vazia leria como catálogo quebrado.
func TestAnInventedTabFallsBackToTheFirst(t *testing.T) {
	v := loadCollection(collectionCriteria{Term: "", Aba: "grimorios-proibidos"}, bookui.BookAddress{})
	if v.Aba != "condicoes" {
		t.Errorf("aba %q, quero cair em condicoes", v.Aba)
	}
	if v.Achados == 0 {
		t.Error("aba inventada devolveu tela vazia")
	}
}

// TestPowersComeFromTheThreeCatalogs achatados, com a fonte preservada.
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
	// Os DIVINOS são 72 e eram 36 (ALE-264): o acervo lia o `granted-powers`,
	// que é metade dos nomes, porque um comentário afirmava que os poderes
	// divinos não têm texto de regra. Eles têm — os 80 do `divine-powers` vêm
	// com descrição completa, e 80 viram 72 ao juntar por nome os que vários
	// deuses concedem ("Coragem Total" aparece quatro vezes).
	//
	// O número está preso porque a lacuna era INVISÍVEL: o cartão do deus
	// mostrava os poderes como texto e ninguém via que metade não virava elo.
	if fontes["divino"] != 72 {
		t.Errorf("%d poderes divinos no acervo — eram 72 quando isto foi escrito", fontes["divino"])
	}
}

// ── a cena pelo fio ──────────────────────────────────────────────────────────

// TestTheSearchInTheUrlHoldsOnAColdLoad: `?busca=` é endereço, e um link colado no chat
// da mesa tem de abrir já filtrado.
func TestTheSearchInTheUrlHoldsOnAColdLoad(t *testing.T) {
	corpo := pedeNaCena(t, "/mestre/condicoes?busca=fogo").Body.String()

	// A prova de que a página abriu FILTRADA é o que ela MOSTRA, e não uma
	// contagem colhida da mesma função que a desenhou.
	//
	// A versão anterior fazia isso: chamava `loadCollection` com os mesmos
	// critérios e afirmava que a página continha o número devolvido por ela. Um
	// erro na busca sairia dos DOIS lados e o guarda ficaria verde. É o que o
	// CLAUDE.md chama de derivar o esperado do código sob teste, e quem o expôs
	// foi a fronteira desta fatia, ao tirar a função do alcance do `api`
	// (ALE-278).
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

// TestEveryCollectionTabOffersTheBook (ALE-264).
//
// AMOSTRAGEM e não enumeração: o teste percorre `collectionTabs`, então a aba que
// entrar amanhã já nasce medida. Foi a lição da ALE-252 — guarda que nomeia cada
// caso deixa o próximo nascer sem medição, em silêncio.
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
