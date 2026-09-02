package api

import (
	"t20engine/search"
	"testing"
)

// A regra da busca (ALE-234), portada do `fuzzy-filter.ts` da SPA.
//
// Os casos acentuados são os que o comentário do arquivo original cita por
// nome: eles existem porque o domínio é pt-BR e ninguém digita "Anão" com til
// no meio de uma sessão.

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

// A tolerância a typo é o que o comentário do original chama de "a parte que de
// fato importava".
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
// Eu tinha escrito o contrário aqui, "uma letra exige prefixo", raciocinando
// que subsequência de uma letra devolveria a lista toda. O raciocínio estava
// certo e a conclusão errada: quem devolve a lista toda é o `Contains`, e ele é
// o comportamento ORIGINAL. Portar a minha versão "melhorada" teria mudado a
// busca em silêncio, e a única forma de saber foi rodar a biblioteca:
//
//	node -e "const {rankItem}=require('@tanstack/match-sorter-utils');
//	         console.log(rankItem('Sombras','a').passed)"  // true
//
// Os sete casos deste arquivo foram conferidos assim, um a um, e o port
// concorda com a biblioteca em todos — inclusive nos dois que RECUSAM
// ("nzcromante", "anaox"), que são os que provam que ele não ficou frouxo.
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

// O caso que me assustou na tela e que a biblioteca CONFIRMOU (ALE-234).
//
// Buscando "tauron" na cena, três das seis campanhas ficaram — e a terceira,
// "Segredos de Wynlla", não tem "tauron" em lugar nenhum. Achei que meu port
// tinha ficado frouxo. Não tinha: "t-a-u-r-o-n" É subsequência da sinopse dela
// ("in*t*riga *a*rcana ... *u*m nec*r*omante"), e o `match-sorter` casa
// exatamente igual:
//
//	rankItem(sinopseDeWynlla, 'tauron').passed  // true
//
// Fica FIXADO como está, e não "consertado". A busca difusa sobre sinopse longa
// é frouxa nos DOIS lados, e apertar só o lado novo faria as duas telas
// responderem coisas diferentes para a mesma busca — que é a divergência que
// esta migração inteira existe para evitar. Se um dia apertar, aperta nos dois.
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
