package api

import (
	"strings"
	"testing"
)

// O guarda dos FILTROS de cada catálogo (ALE-264).
//
// O que se protege é a regra de COMBINAÇÃO — OU dentro de um filtro, E entre
// eles — e o fato de que cada catálogo oferece os seus. Um filtro que some da
// cena não estoura nada: a lista continua desenhando, só que inteira.

// TestCadaCatalogoOfereceOsFiltrosDele: AMOSTRAGEM sobre as abas, para o
// catálogo que entrar amanhã nascer medido.
func TestCadaCatalogoOfereceOsFiltrosDele(t *testing.T) {
	esperado := map[string][]string{
		"condicoes": {"efeito"},
		"magias":    {"circulo", "escola", "classe"},
		"poderes":   {"fonte"},
		"itens":     {"familia"},
		"deuses":    {"energia"},
		"racas":     {"linhagem"},
		// Classes (14) e Efeitos (18) não têm filtro, e é decisão: numa lista
		// que cabe na tela, filtro é cromo que não poupa rolagem.
		"classes": nil,
		"efeitos": nil,
	}
	for _, aba := range abasDoAcervo {
		chaves := esperado[aba.ID]
		filtros := filtrosDaAba(aba.ID)
		if len(filtros) != len(chaves) {
			t.Errorf("a aba %q tem %d filtros, esperado %d", aba.ID, len(filtros), len(chaves))
			continue
		}
		for i, f := range filtros {
			if f.Chave != chaves[i] {
				t.Errorf("a aba %q: filtro %d é %q, esperado %q", aba.ID, i, f.Chave, chaves[i])
			}
			// Filtro sem opção é uma linha de rótulo e mais nada.
			if len(f.Opcoes) == 0 {
				t.Errorf("o filtro %q da aba %q nasceu sem opções", f.Chave, aba.ID)
			}
		}
	}
}

// TestOFiltroSomaDentroEMultiplicaEntre: a regra de combinação.
//
// Medido na tela: 198 magias, 39 no 3º círculo, 6 no 3º círculo E da escola de
// evocação. Os números ficam presos porque são o que separa "filtrou" de
// "filtrou do jeito certo" — um E virando OU daria 39 + as evocações todas, que
// também é uma lista plausível.
func TestOFiltroSomaDentroEMultiplicaEntre(t *testing.T) {
	todas := len(catalogosDoLivro().Magias)
	if todas != 198 {
		t.Fatalf("%d magias no catálogo — os números abaixo perderam o sentido", todas)
	}

	terceiro := quantasMagias(map[string][]string{"circulo": {"3"}})
	if terceiro != 39 {
		t.Errorf("%d magias de 3º círculo, esperado 39", terceiro)
	}
	// OU dentro do mesmo filtro: 3º ou 4º é mais que só 3º.
	doisCirculos := quantasMagias(map[string][]string{"circulo": {"3", "4"}})
	if doisCirculos <= terceiro {
		t.Errorf("3º OU 4º deu %d, e só 3º dá %d — o OU dentro do filtro sumiu", doisCirculos, terceiro)
	}
	// E entre filtros: 3º E evocação é menos que só 3º.
	comEscola := quantasMagias(map[string][]string{"circulo": {"3"}, "escola": {"evocacao"}})
	if comEscola != 6 {
		t.Errorf("3º círculo E evocação deu %d, esperado 6", comEscola)
	}
}

func quantasMagias(filtros map[string][]string) int {
	v := carregaCatalogos(criteriosDoAcervo{Aba: "magias", Filtros: filtros}, enderecoDoLivro{})
	return v.Achados
}

// TestOFiltroEUmEndereco: `?circulo=3` abre a cena já filtrada.
func TestOFiltroEUmEndereco(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/magias?circulo=3&escola=evocacao", "").Body.String()
	if !strings.Contains(corpo, "6 entradas") {
		t.Error("a cena não abriu filtrada pelo endereço")
	}
	// E os crachás nascem ACESOS: sem isso a lista está filtrada e a tela não
	// diz por quê — o pior estado possível.
	if !strings.Contains(corpo, `aria-pressed="true"`) {
		t.Error("nenhum crachá aceso numa cena que abriu filtrada")
	}
}

// TestOFiltroDeOutraAbaNaoEntra: `?circulo=3` nas condições não filtra nada.
//
// Aceitá-lo faria a cena carregar um estado que ela não sabe desenhar — filtro
// aplicado sem crachá para desligá-lo, e a pessoa presa numa lista curta sem
// entender.
func TestOFiltroDeOutraAbaNaoEntra(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/condicoes?circulo=3", "").Body.String()
	if !strings.Contains(corpo, "35 entradas") {
		t.Error("um filtro de outra aba mexeu na lista das condições")
	}
}

// TestABuscaEsconde OsFiltros — com termo digitado a cena responde outra
// pergunta, e um crachá de círculo sobre uma lista com itens e condições diria
// que ele filtra o que não filtra.
func TestABuscaEscondeOsFiltros(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	semBusca := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/magias", "").Body.String()
	comBusca := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/magias?busca=fogo", "").Body.String()

	if !strings.Contains(semBusca, "Filtrar por Círculo") {
		t.Error("a cena das magias não desenha os filtros")
	}
	if strings.Contains(comBusca, "Filtrar por Círculo") {
		t.Error("os filtros continuaram na tela durante a busca")
	}
}

// TestAsRacasExoticasDizemQueSaoExoticas.
//
// PROVADO VERMELHO: o código procurava `exotica` e o dado guarda `extra`, então
// o `else` devolvia "Comum" para as DEZESSETE — inclusive as nove exóticas. Só
// apareceu ao medir o dado para montar o filtro; na tela era um rótulo plausível
// em todo cartão.
func TestAsRacasExoticasDizemQueSaoExoticas(t *testing.T) {
	racas, _, _ := catalogosDoPersonagem()
	contagem := map[string]int{}
	for _, r := range racas {
		contagem[nomeDoTier(r.Tier)]++
	}
	if contagem["Exótica"] != 9 || contagem["Comum"] != 8 {
		t.Errorf("%d exóticas e %d comuns — o livro tem 9 e 8", contagem["Exótica"], contagem["Comum"])
	}
}
