package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// Os guardas dos CATÁLOGOS (ALE-258).

// TestTodoValorDoLivroTemRotulo é o guarda GENÉRICO que a fatia do bestiário me
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
func TestTodoValorDoLivroTemRotulo(t *testing.T) {
	a := catalogosDoLivro()
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

	cobra("execução", execucoes, nomeDaExecucao)
	cobra("alcance", alcances, nomeDoAlcance)
	cobra("categoria", categorias, nomeDaCategoria)
}

// TestABuscaExigeTODOSOsTermos: "luz cur" só casa com o que carrega as duas
// coisas. É a regra que separa esta busca da das outras cenas.
func TestABuscaExigeTODOSOsTermos(t *testing.T) {
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

// TestOAcentoNaoSeparaNaBusca: ninguém digita til no meio da sessão.
func TestOAcentoNaoSeparaNaBusca(t *testing.T) {
	campos := []string{"Ilusão Lacerante", "Cria uma imagem que fere."}
	for _, busca := range []string{"ilusao", "Ilusão", "ILUSAO", "imagem"} {
		if !casaTodosOsTermos(campos, busca) {
			t.Errorf("%q não casou com %v", busca, campos)
		}
	}
}

// TestBuscarVarreOsQUATROCatalogos, e não só a aba aberta.
//
// É a ALE-22: a versão em React filtrava só a aba ativa, então "bola de fogo"
// digitado na aba Condições dizia "nada encontrado" com a magia existindo. A
// aba é para NAVEGAR sem termo; com termo, o assunto é o acervo inteiro.
func TestBuscarVarreOsQuatroCatalogos(t *testing.T) {
	v := carregaCatalogos("fogo", "condicoes")
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

// TestSemBuscaMostraSoAAbaAberta: sem termo a cena é um catálogo por vez.
func TestSemBuscaMostraSoAAbaAberta(t *testing.T) {
	a := catalogosDoLivro()
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
			v := carregaCatalogos("", caso.aba)
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

// TestAbaInventadaCaiNaPrimeira: o `?aba=` é endereço e alguém o digita errado
// — cair em tela vazia leria como catálogo quebrado.
func TestAbaInventadaCaiNaPrimeira(t *testing.T) {
	v := carregaCatalogos("", "grimorios-proibidos")
	if v.Aba != "condicoes" {
		t.Errorf("aba %q, quero cair em condicoes", v.Aba)
	}
	if v.Achados == 0 {
		t.Error("aba inventada devolveu tela vazia")
	}
}

// TestOsPoderesVemDosTRESCatalogos achatados, com a fonte preservada.
//
// O achatamento é o ponto da ferramenta — o livro espalha poder por três
// lugares e o mestre quer uma lista só —, e o que ele não pode perder é DE ONDE
// veio. Sem a fonte, "Ataque Poderoso" não diz se é poder de classe ou geral.
func TestOsPoderesVemDosTresCatalogos(t *testing.T) {
	fontes := map[string]int{}
	for _, p := range catalogosDoLivro().Poderes {
		switch {
		case strings.HasPrefix(p.ID, "general."):
			fontes["geral"]++
		case strings.HasPrefix(p.ID, "granted."):
			fontes["concedido"]++
		default:
			fontes["classe"]++
		}
		if p.Fonte == "" {
			t.Fatalf("o poder %q não diz de onde veio", p.Name)
		}
	}
	for _, esperada := range []string{"classe", "geral", "concedido"} {
		if fontes[esperada] == 0 {
			t.Errorf("nenhum poder de %q — um dos três catálogos não entrou", esperada)
		}
	}
	fmt.Sprint(fontes)
}

// ── a cena pelo fio ──────────────────────────────────────────────────────────

// TestOsCatalogosAbremNaPrimeiraAba e desenham o catálogo inteiro.
func TestOsCatalogosAbremNaPrimeiraAba(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/catalogos", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	corpo := rec.Body.String()
	a := catalogosDoLivro()
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

// TestOAcervoInteiroSaiNaAbaDePoderes: a decisão do dono foi mandar TUDO, sem
// teto nem paginação. Se alguém puser um `[:60]` aqui um dia, este guarda cai.
func TestOAcervoInteiroSaiNaAbaDePoderes(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/catalogos?aba=poderes", "")
	poderes := catalogosDoLivro().Poderes
	if !strings.Contains(rec.Body.String(), fmt.Sprintf("%d entradas", len(poderes))) {
		t.Fatalf("a contagem não é a dos %d poderes", len(poderes))
	}
	// O ÚLTIMO da lista, e não o primeiro: um teto cortaria pelo fim.
	ultimo := poderes[len(poderes)-1]
	if !strings.Contains(rec.Body.String(), ultimo.Name) {
		t.Errorf("o último poder (%q) não saiu — a lista foi cortada", ultimo.Name)
	}
}

// TestABuscaNaURLValeNaCargaFria: `?busca=` é endereço, e um link colado no chat
// da mesa tem de abrir já filtrado.
func TestABuscaNaURLValeNaCargaFria(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/catalogos?busca=fogo", "")
	esperados := carregaCatalogos("fogo", "").Achados
	if esperados == 0 {
		t.Fatal("buscar fogo não acha nada: o dado mudou e o teste perdeu o sentido")
	}
	if !strings.Contains(rec.Body.String(), fmt.Sprintf("%d achados", esperados)) {
		t.Errorf("a página não abriu filtrada; queria %d achados", esperados)
	}
}

// TestBuscarEsconde AFileiraDeAbas — com termo digitado o assunto é o acervo
// inteiro, e deixar a fileira acesa diria que a busca é de uma aba só (ALE-22).
func TestBuscarEscondeAFileiraDeAbas(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	semBusca := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/catalogos", "").Body.String()
	comBusca := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/catalogos?busca=fogo", "").Body.String()

	if !strings.Contains(semBusca, `aria-label="Catálogos"`) {
		t.Error("sem busca a fileira de abas sumiu")
	}
	if strings.Contains(comBusca, `aria-label="Catálogos"`) {
		t.Error("com busca a fileira de abas continuou na tela")
	}
}
