package api

import (
	"t20engine/domain/search"
	"testing"
)

// A regra da busca, portada do `match-sorter`.
//
// Os casos acentuados existem porque o domínio é pt-BR e ninguém digita "Anão"
// com til no meio de uma sessão.

func TestSearchIgnoresAccents(t *testing.T) {
	casos := []struct {
		campo, busca string
		quer         bool
	}{
		{"Anão", "anao", true},
		{"anao", "Anão", true},
		{"Necromante", "necromante", true},
		{"Sombras de Valkaria", "VALKARIA", true},
		{"Anão", "elfo", false},
	}
	for _, c := range casos {
		if got := search.Matches([]string{c.campo}, c.busca); got != c.quer {
			t.Errorf("casaBusca(%q, %q) = %v, queria %v", c.campo, c.busca, got, c.quer)
		}
	}
}

// A tolerância a typo é a metade que de fato importa.
func TestSearchToleratesAMissingLetter(t *testing.T) {
	if !search.Matches([]string{"Necromante"}, "ncromante") {
		t.Error("uma letra pulada derrubou a busca — é o typo que se comete digitando rápido")
	}
	if !search.Matches([]string{"Sombras de Valkaria"}, "valkria") {
		t.Error("typo no meio da palavra derrubou a busca")
	}
}

// E ela é frouxa numa direção SÓ. Aceitar troca e transposição faria uma lista
// de seis campanhas devolver a lista inteira quase sempre, e aí o filtro deixa
// de filtrar.
func TestSearchAcceptsNeitherASwappedNorAnExtraLetter(t *testing.T) {
	if search.Matches([]string{"Necromante"}, "nzcromante") {
		t.Error("letra TROCADA passou: a busca ficou frouxa demais para filtrar")
	}
	if search.Matches([]string{"Anão"}, "anaox") {
		t.Error("letra SOBRANDO passou")
	}
}

// Uma letra casa por SUBSTRING, em qualquer posição — e isso não é palpite: é o
// que o `match-sorter` faz, medido rodando a biblioteca de verdade.
//
// "Uma letra exige prefixo" é a conclusão intuitiva e ERRADA: parece que
// subsequência de uma letra devolveria a lista toda, mas quem devolve a lista
// toda é o `Contains`, e ele é o comportamento original. Todo caso deste
// arquivo foi conferido contra a biblioteca, um a um, e é assim que se confere
// o próximo:
//
//	node -e "const {rankItem}=require('@tanstack/match-sorter-utils');
//	         console.log(rankItem('Sombras','a').passed)"  // true
//
// Os dois que RECUSAM ("nzcromante", "anaox") são os que provam que o port não
// ficou frouxo.
func TestASingleLetterSearchMatchesAtAnyPosition(t *testing.T) {
	for _, campo := range []string{"Anão", "Sombras"} {
		if !search.Matches([]string{campo}, "a") {
			t.Errorf("%q não casou com \"a\" — o match-sorter casa", campo)
		}
	}
	if search.Matches([]string{"Sombras"}, "z") {
		t.Error(`"z" casou com "Sombras"`)
	}
}

func TestAnEmptySearchFiltersNothing(t *testing.T) {
	for _, busca := range []string{"", "   "} {
		if !search.Matches([]string{"qualquer coisa"}, busca) {
			t.Errorf("busca %q filtrou — não digitar não é filtrar", busca)
		}
	}
}

// Vários campos: casa se QUALQUER um casar. Na cena das campanhas são o nome e
// a sinopse.
func TestSearchLooksAtEveryField(t *testing.T) {
	campos := []string{"Sombras de Valkaria", "Uma campanha sobre a Tormenta"}
	if !search.Matches(campos, "tormenta") {
		t.Error("não achou pela sinopse")
	}
	if search.Matches(campos, "dragão") {
		t.Error("casou com o que não está em campo nenhum")
	}
}

// O caso que assusta na tela e que a biblioteca CONFIRMA: buscar "tauron" traz
// "Segredos de Wynlla", que não tem "tauron" em lugar nenhum — mas
// "t-a-u-r-o-n" É subsequência da sinopse ("in*t*riga *a*rcana ... *u*m
// nec*r*omante"), e o `match-sorter` casa igual:
//
//	rankItem(sinopseDeWynlla, 'tauron').passed  // true
//
// Fica FIXADO como está, e não "consertado": a busca difusa sobre sinopse longa
// é frouxa por construção, e apertá-la é decisão de produto, não conserto.
func TestSearchStaysLooseOverALongSynopsisAsItAlwaysDid(t *testing.T) {
	wynlla := "Campanha de intriga arcana na Academia Arcana de Wynlla — segredos proibidos e um necromante à espreita."

	if !search.Matches([]string{"Segredos de Wynlla", wynlla}, "tauron") {
		t.Error("o port ficou MAIS restrito que o match-sorter — as duas telas passariam a discordar")
	}
	// O nome curto sozinho não casa: é a sinopse que abre a porta.
	if search.Matches([]string{"Segredos de Wynlla"}, "tauron") {
		t.Error(`"tauron" casou com o NOME "Segredos de Wynlla"`)
	}
	// E o caso legítimo continua legítimo.
	if !search.Matches([]string{"A Queda de Tauron"}, "tauron") {
		t.Error("a campanha que tem Tauron no nome não casou")
	}
	// Palavras coladas: o match-sorter aceita, e o port também.
	if !search.Matches([]string{"A Queda de Tauron"}, "quedatauron") {
		t.Error("palavras coladas não casaram — o match-sorter casa")
	}
}
