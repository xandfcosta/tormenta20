package engine

import (
	"math"
	"testing"
)

// A conta do encontro contra o LIVRO (ALE-259).
//
// Os casos vêm dos exemplos escritos no próprio texto da regra (p282) e das
// armadilhas que a SPA já pagou. Não é uma transcrição de tabela: é o
// comportamento nas bordas, que é o que quebra.

func TestThePartyChallengeLevelFollowsTheBook(t *testing.T) {
	casos := []struct {
		nome  string
		nd    float64
		qtd   int
		quero float64
	}{
		// Abaixo de ND 1 a regra é MULTIPLICAR, e os dois exemplos do texto:
		{"quatro de ND 1/4 dão ND 1", 0.25, 4, 1},
		{"dois de ND 1/2 dão ND 1", 0.5, 2, 1},
		// De ND 1 para cima é +2 A CADA DOBRA, e os dois exemplos do texto:
		{"dois de ND 1 dão ND 3", 1, 2, 3},
		{"quatro de ND 5 dão ND 9", 5, 4, 9},
		// Uma criatura só é ela mesma, em qualquer faixa.
		{"uma de ND 1/4", 0.25, 1, 0.25},
		{"uma de ND 7", 7, 1, 7},
		// Dobra não-inteira cai ENTRE os degraus, e é extensão do log2 — o
		// livro cala sobre grupos de 3.
		{"três de ND 1 caem entre 1x e 2x", 1, 3, 1 + 2*math.Log2(3)},
		// Quantidade zero ou negativa não é encontro.
		{"zero criaturas", 3, 0, 0},
		{"quantidade negativa", 3, -2, 0},
	}
	for _, c := range casos {
		if got := NDDeGrupo(c.nd, c.qtd); math.Abs(got-c.quero) > 1e-9 {
			t.Errorf("%s: NDDeGrupo(%v, %d) = %v, quero %v", c.nome, c.nd, c.qtd, got, c.quero)
		}
	}
}

// TestAFractionalDifferenceDoesNotFallIntoHard é a ALE-25, e é a razão de a
// diferença ser arredondada ANTES de escolher a faixa.
//
// Uma criatura de ND 1/4 contra um grupo de nível 1 dá diferença −0,75, que
// escapa do `<= -1` e do `== 0` e caía em "Difícil" — o oposto da verdade.
// Arredondada, ela vira −1 e cai em "Fácil", que é a leitura certa.
//
// Eu escrevi este teste esperando "Médio" e ele falhou. O errado era a minha
// expectativa, não a regra: `Round(-0,75)` é −1 tanto em Go quanto em JS, e
// uma criatura de ND 1/4 contra um grupo inteiro de nível 1 é fácil mesmo. Fica
// registrado porque o que o teste protege é a FAIXA CALMA, não um rótulo
// específico — se alguém trocar o arredondamento, o sintoma volta a ser
// "Difícil" e é isso que tem de acusar.
func TestAFractionalDifferenceDoesNotFallIntoHard(t *testing.T) {
	d := DificuldadeDoEncontro(NDDeGrupo(0.25, 1) - 1)
	if d.Rotulo != "Fácil" {
		t.Errorf("uma criatura de ND 1/4 contra grupo de nível 1 deu %q, quero Fácil", d.Rotulo)
	}
	if d.Tom != "calmo" {
		t.Errorf("e pintou de %q — a ALE-25 é exatamente isto: o combate mais fácil "+
			"possível anunciado como ameaça", d.Tom)
	}
}

func TestTheDifficultyBands(t *testing.T) {
	casos := map[float64]string{
		-5:   "Trivial",
		-3:   "Trivial",
		-2:   "Fácil",
		-1:   "Fácil",
		-0.4: "Médio",
		0:    "Médio",
		0.4:  "Médio",
		1:    "Difícil",
		2:    "Difícil",
		3:    "Mortal",
		9:    "Mortal",
	}
	for diferenca, quero := range casos {
		if got := DificuldadeDoEncontro(diferenca).Rotulo; got != quero {
			t.Errorf("diferença %v deu %q, quero %q", diferenca, got, quero)
		}
	}
}

// TestAnIrrelevantChallengeIsWorthNoXp: cinco degraus abaixo do nível do grupo e o
// combate não ensina nada (p326).
func TestAnIrrelevantChallengeIsWorthNoXp(t *testing.T) {
	if xp := XPDoEncontro(2, 7, 4, Vitoria); xp != 0 {
		t.Errorf("ND 2 contra grupo de nível 7 deu %d XP, quero 0", xp)
	}
	// A borda: exatamente cinco abaixo ainda é irrelevante; um pouco acima já
	// vale. Prender as duas é o que impede o `<=` de virar `<`.
	if xp := XPDoEncontro(2, 7, 4, Vitoria); xp != 0 {
		t.Errorf("a borda de cinco degraus deu %d, quero 0", xp)
	}
	if xp := XPDoEncontro(2.5, 7, 4, Vitoria); xp == 0 {
		t.Error("ND 2,5 contra nível 7 é menos de cinco degraus e devia valer XP")
	}
}

func TestXpIsSplitAndDependsOnTheOutcome(t *testing.T) {
	// ND 4, grupo de 4 no nível 3: base 4.000, dividida por quatro.
	if xp := XPDoEncontro(4, 3, 4, Vitoria); xp != 1000 {
		t.Errorf("vitória deu %d, quero 1000", xp)
	}
	if xp := XPDoEncontro(4, 3, 4, Empate); xp != 500 {
		t.Errorf("empate deu %d, quero metade", xp)
	}
	if xp := XPDoEncontro(4, 3, 4, Derrota); xp != 250 {
		t.Errorf("derrota deu %d, quero um quarto", xp)
	}
	// Grupo vazio não recebe: dividir por zero seria +Inf virando um número na
	// tela.
	if xp := XPDoEncontro(4, 3, 0, Vitoria); xp != 0 {
		t.Errorf("grupo de zero deu %d, quero 0", xp)
	}
	// Desfecho desconhecido cai em vitória em vez de zerar o XP: o dado vem da
	// URL, e um typo não pode apagar a recompensa da mesa.
	if xp := XPDoEncontro(4, 3, 4, DesfechoDoEncontro("qualquer-coisa")); xp != 1000 {
		t.Errorf("desfecho desconhecido deu %d, quero cair em vitória", xp)
	}
}
