package api

import "testing"

// A regra da busca (ALE-234), portada do `fuzzy-filter.ts` da SPA.
//
// Os casos acentuados são os que o comentário do arquivo original cita por
// nome: eles existem porque o domínio é pt-BR e ninguém digita "Anão" com til
// no meio de uma sessão.

func TestBuscaIgnoraAcento(t *testing.T) {
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
		if got := casaBusca([]string{c.campo}, c.busca); got != c.quer {
			t.Errorf("casaBusca(%q, %q) = %v, queria %v", c.campo, c.busca, got, c.quer)
		}
	}
}

// A tolerância a typo é o que o comentário do original chama de "a parte que de
// fato importava".
func TestBuscaToleraLetraFaltando(t *testing.T) {
	if !casaBusca([]string{"Necromante"}, "ncromante") {
		t.Error("uma letra pulada derrubou a busca — é o typo que se comete digitando rápido")
	}
	if !casaBusca([]string{"Sombras de Valkaria"}, "valkria") {
		t.Error("typo no meio da palavra derrubou a busca")
	}
}

// E ela é frouxa numa direção SÓ. Aceitar troca e transposição faria uma lista
// de seis campanhas devolver a lista inteira quase sempre, e aí o filtro deixa
// de filtrar.
func TestBuscaNaoAceitaLetraTrocadaNemSobrando(t *testing.T) {
	if casaBusca([]string{"Necromante"}, "nzcromante") {
		t.Error("letra TROCADA passou: a busca ficou frouxa demais para filtrar")
	}
	if casaBusca([]string{"Anão"}, "anaox") {
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
func TestBuscaDeUmaLetraCasaEmQualquerPosicao(t *testing.T) {
	for _, campo := range []string{"Anão", "Sombras"} {
		if !casaBusca([]string{campo}, "a") {
			t.Errorf("%q não casou com \"a\" — o match-sorter casa", campo)
		}
	}
	if casaBusca([]string{"Sombras"}, "z") {
		t.Error(`"z" casou com "Sombras"`)
	}
}

func TestBuscaVaziaNaoFiltraNada(t *testing.T) {
	for _, busca := range []string{"", "   "} {
		if !casaBusca([]string{"qualquer coisa"}, busca) {
			t.Errorf("busca %q filtrou — não digitar não é filtrar", busca)
		}
	}
}

// Vários campos: casa se QUALQUER um casar. Na cena das campanhas são o nome e
// a sinopse.
func TestBuscaOlhaTodosOsCampos(t *testing.T) {
	campos := []string{"Sombras de Valkaria", "Uma campanha sobre a Tormenta"}
	if !casaBusca(campos, "tormenta") {
		t.Error("não achou pela sinopse")
	}
	if casaBusca(campos, "dragão") {
		t.Error("casou com o que não está em campo nenhum")
	}
}
