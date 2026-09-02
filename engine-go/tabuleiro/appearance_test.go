package tabuleiro

import (
	"testing"
)

// ── a aparência da peça (ALE-179, portada na ALE-263) ────────────────────────
//
// A regra: a cor é da ESPÉCIE e o número é da INSTÂNCIA. Os casos são os que a
// suíte da SPA já nomeava, porque as bordas não mudaram com o transporte.

func TestEqualsComeOutEqualAndTheNumberStaysOutOfTheColor(t *testing.T) {
	um, tres := AppearanceOf("Zumbi 1"), AppearanceOf("Zumbi 3")

	if um.Matiz != tres.Matiz {
		t.Errorf("dois zumbis saíram em matizes diferentes: %d e %d", um.Matiz, tres.Matiz)
	}
	if um.Monograma != tres.Monograma {
		t.Errorf("monogramas diferentes: %q e %q", um.Monograma, tres.Monograma)
	}
	if um.Instancia != "1" || tres.Instancia != "3" {
		t.Errorf("selos: %q e %q", um.Instancia, tres.Instancia)
	}
}

// O monograma vem da ESPÉCIE. Ele comia as duas primeiras palavras, então
// "Zumbi Putrefato 2" virava "ZP" e o número — a única coisa que distingue as
// três peças na mesa — era justamente o que se perdia.
func TestTheMonogramComesFromTheKindAndTheNumberBecomesASeal(t *testing.T) {
	p := AppearanceOf("Zumbi Putrefato 2")
	if p.Monograma != "ZP" || p.Instancia != "2" {
		t.Errorf("ficou %q + %q, queria ZP + 2", p.Monograma, p.Instancia)
	}
}

// DUAS letras mesmo em nome de uma palavra: no tabuleiro um "O" solto tem
// metade da massa que a peça precisa para ser achada entre vinte vizinhas.
func TestWithoutANumberThereIsNoSealAndTheMonogramStillHasTwoLetters(t *testing.T) {
	p := AppearanceOf("Ogro")
	if p.Instancia != "" {
		t.Errorf("apareceu selo %q num nome sem número", p.Instancia)
	}
	if p.Monograma != "OG" {
		t.Errorf("monograma %q, queria OG", p.Monograma)
	}
}

// "Nv1" está no MEIO do nome e não é instância: separar por qualquer dígito
// transformaria "Recruta Nv1 Simples" em outra espécie.
func TestANumberInTheMiddleOfTheNameIsNotAnInstance(t *testing.T) {
	p := AppearanceOf("Recruta Nv1 Simples")
	if p.Instancia != "" {
		t.Errorf("o Nv1 virou selo %q", p.Instancia)
	}
	if p.Monograma != "RN" {
		t.Errorf("monograma %q", p.Monograma)
	}
}

func TestDifferentKindsStayDistinct(t *testing.T) {
	if AppearanceOf("Zumbi 1").Matiz == AppearanceOf("Goblin 1").Matiz {
		t.Error("zumbi e goblin saíram no mesmo matiz — a cor deixou de dizer algo")
	}
}

// O MATIZ É O MESMO DA SPA, e tem de continuar sendo: a peça do tabuleiro e o
// retrato do herói mostram a mesma criatura, e duas fórmulas dariam duas cores
// para ela em duas telas.
//
// Os números foram RODADOS no `hueFromName` da SPA com node e transcritos, e é
// isso que faz deste um guarda de contrato entre as duas linguagens. Derivá-los
// de um hash reescrito na asserção compararia a função consigo mesma — que foi
// a primeira versão deste teste, e ela passava sem provar nada.
//
// O "Ácido" está aqui de propósito: o JavaScript percorre PONTOS DE CÓDIGO
// (`for ch of name`), e um port que iterasse BYTES daria outro número em todo
// nome acentuado. É a única entrada da lista que pega esse erro.
//
// A doc do `hueFromName` diz `hueFromName('Thorvald') // => 214`, e ELA ESTÁ
// ERRADA: rodado, o valor é 186. Foi este teste que descobriu, porque nada no
// front afirma aquele exemplo. Avisada a sessão que cuida da SPA.
func TestTheHueIsTheSameAsTheHeroPortrait(t *testing.T) {
	casos := map[string]int{
		"Thorvald": 186,
		"Ogro":     197,
		"Zumbi":    9,
		"Arwen":    119,
		"Ácido":    218,
		"Goblin":   183,
	}
	for nome, quero := range casos {
		if got := hueOf(nome); got != quero {
			t.Errorf("matiz de %q = %d, quero %d (rodado no hueFromName da SPA)", nome, got, quero)
		}
	}
}
