package tabuleiro

import "testing"

// A regra da LETRA do marcador (ALE-195), agora no motor.
//
// Ela vivia só na SPA (`nextMarkerText`), onde o cliente escolhia a letra e
// mandava pronta — e "qual letra está livre?" é pergunta sobre o ESTADO DO
// TABULEIRO, que é o que este pacote guarda. Com ela aqui, as duas telas nomeiam
// igual.
//
// O que se prende é a ARMADILHA e não a tabela: que ela pula o que já foi usado
// (e não conta quantos existem) e o que acontece quando as 26 acabam.

func TestAProximaLetraPulaAsUsadasEmVezDeContar(t *testing.T) {
	// "A" livre porque o mapa está vazio — o CONTROLE de que a função responde
	// alguma coisa antes de eu afirmar o caso difícil.
	if got := ProximaLetraDeMarcador(nil); got != "A" {
		t.Fatalf("mapa vazio devolveu %q, esperado \"A\"", got)
	}

	// A ARMADILHA: dois marcadores no mapa, mas o "B" foi apagado. Contar daria
	// "C" e deixaria o "B" órfão para sempre; o certo é a primeira LIVRE.
	mapa := []BoardMarker{{Text: "A"}, {Text: "C"}}
	if got := ProximaLetraDeMarcador(mapa); got != "B" {
		t.Errorf("com A e C no mapa saiu %q, esperado \"B\" — a regra está contando em vez de procurar", got)
	}
}

func TestEsgotadasAsLetrasORotuloDesiste(t *testing.T) {
	var mapa []BoardMarker
	for letra := 'A'; letra <= 'Z'; letra++ {
		mapa = append(mapa, BoardMarker{Text: string(letra)})
	}
	// O CONTROLE: com 25 ainda há letra, senão "??" seria verdade sobre um laço
	// que nunca entra.
	if got := ProximaLetraDeMarcador(mapa[:25]); got != "Z" {
		t.Fatalf("com 25 usadas saiu %q, esperado \"Z\"", got)
	}
	if got := ProximaLetraDeMarcador(mapa); got != "??" {
		t.Errorf("com as 26 usadas saiu %q, esperado \"??\"", got)
	}
}

// TestOPredicadoDeCorSegueALista.
//
// Ele NÃO repete o `TestACorDoMarcadorEDeUmConjuntoFechado`, que é dono do
// caminho da gravação (e cobre o patch, que é a porta dos fundos). O que se
// prende aqui é o que a estrutura nova acrescenta: a lista e o predicado saíram
// do mesmo lugar e não podem divergir — divergir foi exatamente o defeito, só
// que entre o domínio e a tela.
func TestOPredicadoDeCorSegueALista(t *testing.T) {
	if len(CoresDeMarcador) == 0 {
		t.Fatal("a lista está vazia — não há o que medir")
	}
	for _, c := range CoresDeMarcador {
		if !CorDeMarcadorConhecida(c.ID) {
			t.Errorf("a cor %q está na lista e a função não a reconhece", c.ID)
		}
	}
	for _, torta := range []string{"gold", "carmesim", "", "red"} {
		if CorDeMarcadorConhecida(torta) {
			t.Errorf("a cor %q passou pelo conjunto fechado", torta)
		}
	}
	if !CorDeMarcadorConhecida(CorPadraoDeMarcador()) {
		t.Error("o PADRÃO não está na lista — quem manda cor torta cairia numa cor que não existe")
	}
}
