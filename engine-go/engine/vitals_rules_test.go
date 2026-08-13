package engine

import "testing"

// PV/PM pools — livro p35 ("Subindo de Nível" e "Multiclasse").
//
// These pools were covered only by the parity oracles: byte-equal against a TS
// snapshot, which proves the two engines AGREE but never that either matches the
// book. Once the TS side is retired (ALE-104) the oracle regenerates from Go
// itself, so the rule needs to be pinned against the BOOK here, in the engine
// that runs it (ALE-105).
//
// The two rules with a worked example in the text:
//
//	"Some sua Constituição aos PV que ganha por nível (mas você sempre ganha
//	 pelo menos 1 PV ao subir de nível)."
//
//	"Quando você ganha o primeiro nível em uma nova classe, ganha os PV de um
//	 nível subsequente, não do primeiro. Zaled ganha 5 PV pelo primeiro nível de
//	 paladino, não 20."

func classes(pairs ...any) []ClassEntry {
	out := []ClassEntry{}
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, ClassEntry{ClassName: pairs[i].(string), Level: pairs[i+1].(int)})
	}
	return out
}

func TestPvPoolSingleClass(t *testing.T) {
	tests := []struct {
		nome  string
		class string
		level int
		con   int
		want  int
	}{
		// PV inicial só no 1º nível; os seguintes ganham pvPerLevel.
		{"Guerreiro L1 é só o PV inicial", "Guerreiro", 1, 0, 20},
		{"Guerreiro L5 = 20 + 4×5", "Guerreiro", 5, 0, 40},
		{"Guerreiro L20 = 20 + 19×5", "Guerreiro", 20, 0, 115},
		{"Bárbaro L20 = 24 + 19×6, a classe mais dura", "Bárbaro", 20, 0, 138},
		{"Arcanista L20 = 8 + 19×2, a mais frágil", "Arcanista", 20, 0, 46},

		// "Some sua Constituição aos PV que ganha por nível" — e ao inicial.
		{"Guerreiro L5 CON 2 = 20+2 + 4×(5+2)", "Guerreiro", 5, 2, 50},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			got := multiclassPvPool(classes(tt.class, tt.level), tt.con)
			if got != tt.want {
				t.Errorf("PV = %d, want %d", got, tt.want)
			}
		})
	}
}

// "mas você sempre ganha pelo menos 1 PV ao subir de nível" — o piso vale por
// nível GANHO. O 1º nível não é "subir de nível", então não tem piso: um
// Arcanista com CON −2 começa com 6 PV, e não com 8.
func TestPvPoolMinimumOnePerLevelGained(t *testing.T) {
	t.Run("CON negativa zera o ganho por nível, o piso devolve 1", func(t *testing.T) {
		// Arcanista ganha 2/nível; CON −2 zeraria. L5 = 8 − 2 + 4×1.
		if got := multiclassPvPool(classes("Arcanista", 5), -2); got != 10 {
			t.Errorf("PV = %d, want 10", got)
		}
	})

	t.Run("o piso vale mesmo com CON muito negativa", func(t *testing.T) {
		// Guerreiro ganha 5/nível; CON −5 zeraria. L3 = 20 − 5 + 2×1.
		if got := multiclassPvPool(classes("Guerreiro", 3), -5); got != 17 {
			t.Errorf("PV = %d, want 17", got)
		}
	})

	t.Run("o 1º nível NÃO tem piso — não é subir de nível", func(t *testing.T) {
		// 8 − 2 = 6. Se o piso valesse aqui, daria 8.
		if got := multiclassPvPool(classes("Arcanista", 1), -2); got != 6 {
			t.Errorf("PV = %d, want 6 (o piso não se aplica ao PV inicial)", got)
		}
	})
}

// "Quando você ganha o primeiro nível em uma nova classe, ganha os PV de um
// nível subsequente, não do primeiro."
func TestPvPoolMulticlassSeedsOnlyOnTheFirstClass(t *testing.T) {
	t.Run("Zaled (p35): o 1º nível de Paladino dá 5 PV, não 20", func(t *testing.T) {
		// Arcanista 3 = 8 + 2×2 = 12. O nível de Paladino soma 5 (pvPerLevel),
		// não os 20 do PV inicial da classe. Total 17.
		got := multiclassPvPool(classes("Arcanista", 3, "Paladino", 1), 0)
		if got != 17 {
			t.Errorf("PV = %d, want 17 — o Paladino entrou com o PV INICIAL?", got)
		}
	})

	// A prova de que a semente é a PRIMEIRA classe e não a maior nem a melhor:
	// as mesmas duas classes, nos mesmos níveis, dão totais diferentes conforme
	// a ordem. Um teste com uma ordem só passaria com a regra errada.
	t.Run("a ordem importa: Guerreiro→Arcanista dá 50, Arcanista→Guerreiro dá 41", func(t *testing.T) {
		guerreiroPrimeiro := multiclassPvPool(classes("Guerreiro", 5, "Arcanista", 5), 0)
		arcanistaPrimeiro := multiclassPvPool(classes("Arcanista", 5, "Guerreiro", 5), 0)

		if guerreiroPrimeiro != 50 {
			t.Errorf("Guerreiro 5 / Arcanista 5 = %d, want 50 (20 + 4×5 + 5×2)", guerreiroPrimeiro)
		}
		if arcanistaPrimeiro != 41 {
			t.Errorf("Arcanista 5 / Guerreiro 5 = %d, want 41 (8 + 4×2 + 5×5)", arcanistaPrimeiro)
		}
	})

	// O caso acima NÃO separa "primeira classe" de "classe de maior nível": nos
	// dois cenários a primeira também é a maior (ou empatam), então a regra
	// errada passaria verde. Aqui a primeira é a MENOR, e os totais divergem:
	// pela regra certa a semente é o Arcanista (8), pela errada seria o
	// Guerreiro (20). Descoberto sabotando a implementação de propósito.
	t.Run("a semente é a PRIMEIRA classe, não a de maior nível", func(t *testing.T) {
		// Arcanista 1 = 8. Guerreiro 5 níveis × 5 = 25. Total 33.
		// Se a semente fosse a maior: 20 + 4×5 + 1×2 = 42.
		got := multiclassPvPool(classes("Arcanista", 1, "Guerreiro", 5), 0)
		if got != 33 {
			t.Errorf("PV = %d, want 33 — semente na classe errada? (a maior daria 42)", got)
		}
	})
}

// "Pontos de Mana. Some os PM fornecidos por cada classe para determinar seu
// montante total." — sem semente, sem piso: soma pura.
func TestMpPoolSumsEveryClass(t *testing.T) {
	t.Run("classe única: mpPerLevel × nível", func(t *testing.T) {
		if got := multiclassMpPool(classes("Arcanista", 5)); got != 30 {
			t.Errorf("PM = %d, want 30 (6×5)", got)
		}
	})

	t.Run("Zaled: Arcanista 3 (6/nv) + Paladino 1 (3/nv)", func(t *testing.T) {
		if got := multiclassMpPool(classes("Arcanista", 3, "Paladino", 1)); got != 21 {
			t.Errorf("PM = %d, want 21", got)
		}
	})

	// O PM não tem a assimetria do PV: trocar a ordem não muda nada.
	t.Run("a ordem NÃO importa, ao contrário do PV", func(t *testing.T) {
		a := multiclassMpPool(classes("Arcanista", 5, "Guerreiro", 5))
		b := multiclassMpPool(classes("Guerreiro", 5, "Arcanista", 5))
		if a != b || a != 45 {
			t.Errorf("PM = %d e %d, want 45 nos dois (6×5 + 3×5)", a, b)
		}
	})
}

// Entradas que a ficha pode produzir antes de o personagem estar completo.
func TestVitalPoolsDegradeWithoutCrashing(t *testing.T) {
	t.Run("sem classe nenhuma, os poços são 0", func(t *testing.T) {
		if got := multiclassPvPool(nil, 3); got != 0 {
			t.Errorf("PV = %d, want 0", got)
		}
		if got := multiclassMpPool(nil); got != 0 {
			t.Errorf("PM = %d, want 0", got)
		}
	})

	t.Run("classe desconhecida como semente zera, não estoura", func(t *testing.T) {
		if got := multiclassPvPool(classes("Necromante", 5), 2); got != 0 {
			t.Errorf("PV = %d, want 0", got)
		}
	})

	// Uma classe desconhecida DEPOIS da semente é ignorada — o personagem não
	// perde os PV que já tinha por causa de um nome que o catálogo não conhece.
	t.Run("classe desconhecida depois da semente é ignorada", func(t *testing.T) {
		if got := multiclassPvPool(classes("Guerreiro", 5, "Necromante", 3), 0); got != 40 {
			t.Errorf("PV = %d, want 40 (só o Guerreiro conta)", got)
		}
	})
}
