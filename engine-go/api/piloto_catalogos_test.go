package api

import (
	"fmt"
	"net/http"
	"strings"
	"t20engine/book"
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
// Provado VERMELHO removendo a linha "catalyst" do `rotuloDaCategoria`.
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
		// A diferença para o `casaBusca`: LÁ isto casaria por subsequência
		// (c-u-r-r nas letras de "Curar"), e uma consulta de regra que devolve
		// quase-acerto no meio da sessão parece defeito.
		{"crr", false, "subsequência NÃO casa: esta busca não é tolerante a typo"},
	}
	for _, c := range casos {
		if got := casaTodosOsTermos(campos, c.busca); got != c.casa {
			t.Errorf("casaTodosOsTermos(%q) = %v, quero %v — %s", c.busca, got, c.casa, c.por)
		}
	}
}

// TestAnAccentDoesNotSplitTheSearch: ninguém digita til no meio da sessão.
func TestAnAccentDoesNotSplitTheSearch(t *testing.T) {
	campos := []string{"Ilusão Lacerante", "Cria uma imagem que fere."}
	for _, busca := range []string{"ilusao", "Ilusão", "ILUSAO", "imagem"} {
		if !casaTodosOsTermos(campos, busca) {
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
	v := carregaCatalogos(criteriosDoAcervo{Busca: "fogo", Aba: "condicoes"}, enderecoDoLivro{})
	if !v.Buscando() {
		t.Fatal("a cena não se considerou em busca")
	}
	achouMagia := false
	for _, g := range v.Grupos {
		if g.Rotulo == "Magias" && len(g.Magias) > 0 {
			achouMagia = true
		}
		if g.Quantos() == 0 {
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
			v := carregaCatalogos(criteriosDoAcervo{Aba: caso.aba}, enderecoDoLivro{})
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
	v := carregaCatalogos(criteriosDoAcervo{Busca: "", Aba: "grimorios-proibidos"}, enderecoDoLivro{})
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

// TestACatalogSceneDrawsTheWholeCatalog.
//
// O endereço mudou na ALE-264: cada catálogo virou uma parada do trilho e ganhou
// cena própria (`/mestre/condicoes`). Este guarda passou a pedir a cena
// direto — quem cobra o endereço VELHO é o `TestTheOldCollectionAddressRedirects`.
func TestACatalogSceneDrawsTheWholeCatalog(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	corpo := rec.Body.String()
	a := book.Catalogs()
	if !strings.Contains(corpo, fmt.Sprintf("%d entradas", len(a.Condicoes))) {
		t.Errorf("a contagem não é a das %d condições", len(a.Condicoes))
	}
	// Uma condição de verdade, e o texto dela: sem isso o teste passaria com a
	// cena desenhando só o cabeçalho.
	if !strings.Contains(corpo, "Abalado") {
		t.Error("a condição Abalado não está na página")
	}
	if strings.Contains(corpo, "Bola de Fogo") {
		t.Error("a aba Condições desenhou magia — a aba não é um catálogo só")
	}
}

// TestTheWholeCollectionComesOutInThePowersTab: a decisão do dono foi mandar TUDO, sem
// teto nem paginação. Se alguém puser um `[:60]` aqui um dia, este guarda cai.
func TestTheWholeCollectionComesOutInThePowersTab(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/mestre/poderes", "")
	poderes := book.Catalogs().Poderes
	if !strings.Contains(rec.Body.String(), fmt.Sprintf("%d entradas", len(poderes))) {
		t.Fatalf("a contagem não é a dos %d poderes", len(poderes))
	}
	// O ÚLTIMO da lista, e não o primeiro: um teto cortaria pelo fim.
	ultimo := poderes[len(poderes)-1]
	if !strings.Contains(rec.Body.String(), ultimo.Name) {
		t.Errorf("o último poder (%q) não saiu — a lista foi cortada", ultimo.Name)
	}
}

// TestTheSearchInTheUrlHoldsOnAColdLoad: `?busca=` é endereço, e um link colado no chat
// da mesa tem de abrir já filtrado.
func TestTheSearchInTheUrlHoldsOnAColdLoad(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes?busca=fogo", "")
	esperados := carregaCatalogos(criteriosDoAcervo{Busca: "fogo", Aba: ""}, enderecoDoLivro{}).Achados
	if esperados == 0 {
		t.Fatal("buscar fogo não acha nada: o dado mudou e o teste perdeu o sentido")
	}
	if !strings.Contains(rec.Body.String(), fmt.Sprintf("%d achados", esperados)) {
		t.Errorf("a página não abriu filtrada; queria %d achados", esperados)
	}
}

// TestSearchingSweepsTheEightCatalogsFromAnyScene.
//
// Este guarda cobrava a FILEIRA DE ABAS, que sumiu na ALE-264 — cada catálogo
// virou uma parada do trilho, e ter as duas coisas seria o mesmo estado
// desenhado em dois lugares. O que ele protegia CONTINUA valendo e é o que ele
// cobra agora: com termo digitado a busca varre os OITO catálogos, não só o da
// cena. É a ALE-22 — "bola de fogo" digitado em Condições dizia "nada
// encontrado" com a magia existindo.
func TestSearchingSweepsTheEightCatalogsFromAnyScene(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	// Da cena das CONDIÇÕES, buscando uma MAGIA.
	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes?busca=bola+de+fogo", "").Body.String()
	if !strings.Contains(corpo, "Bola de Fogo") {
		t.Error("a busca da cena de condições não achou a magia — voltou a filtrar só a aba")
	}
	// O CONTROLE: sem termo, a cena mostra só o catálogo dela.
	so := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes", "").Body.String()
	if strings.Contains(so, "Bola de Fogo") {
		t.Error("sem busca a cena das condições trouxe magia")
	}
}

// TestTheOldCollectionAddressRedirects: `?aba=` foi o único endereço por duas
// fatias desta issue, e pode estar colado no chat de alguma mesa.
func TestTheOldCollectionAddressRedirects(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/mestre/catalogos?aba=magias&busca=fogo", "")
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("o endereço velho respondeu %d", rec.Code)
	}
	destino := rec.Header().Get("Location")
	if !strings.HasPrefix(destino, "/mestre/magias") {
		t.Errorf("levou para %q", destino)
	}
	// A CONSULTA sobrevive: um redirecionamento que perde a busca devolve a
	// pessoa a uma tela que não é a que ela pediu.
	if !strings.Contains(destino, "busca=fogo") {
		t.Errorf("o redirecionamento perdeu a busca: %q", destino)
	}
}

// TestEveryCollectionTabOffersTheBook (ALE-264).
//
// AMOSTRAGEM e não enumeração: o teste percorre `abasDoAcervo`, então a aba que
// entrar amanhã já nasce medida. Foi a lição da ALE-252 — guarda que nomeia cada
// caso deixa o próximo nascer sem medição, em silêncio.
//
// O CONTROLE é a segunda metade: sem livro configurado, a mesma cena não pode
// trazer link nenhum. Sem ele, "achei `#page=`" seria verdade sobre um endereço
// que a cena escreve de qualquer jeito.
func TestEveryCollectionTabOffersTheBook(t *testing.T) {
	s := servidorComLivro(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")

	for _, aba := range abasDoAcervo {
		corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/"+aba.ID, "").Body.String()
		if !strings.Contains(corpo, "/livro/ler?p=") {
			t.Errorf("a aba %q não oferece o livro em nenhuma entrada", aba.Rotulo)
		}
		if !strings.Contains(corpo, "Abrir o livro na página") {
			t.Errorf("a aba %q tem o endereço mas não o título que diz o que ele faz", aba.Rotulo)
		}
	}

	semLivro := newTestServer(t)
	outro := seedUser(t, semLivro, "mestre@t20.local")
	sem := pedeNoMestre(t, semLivro, outro, "GET", "/mestre/condicoes", "").Body.String()
	if strings.Contains(sem, "/livro/ler") {
		t.Error("sem LIVRO_PDF a cena linkou um livro que não é servido")
	}
	// E a página continua ESCRITA: o mestre com o livro de papel usa o número.
	if !strings.Contains(sem, "p394") {
		t.Error("sem livro a página impressa sumiu do cartão — ela não depende do PDF")
	}
}
