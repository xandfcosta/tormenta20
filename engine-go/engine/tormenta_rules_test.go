package engine

import (
	"path/filepath"
	"testing"
)

// Poderes da Tormenta, p136: "Quando escolhe um poder da Tormenta, você perde 1 de
// Carisma. Para cada dois outros poderes da Tormenta, você perde mais 1 de Carisma."
//
// A regra vivia SÓ na cópia TS de `derived.ts`, que a produção não executa mais desde
// a ALE-104 — o teste de lá provava que o código morto estava certo. Aqui ela é
// afirmada onde o motor decide.

func TestCarismaLossFromPowers(t *testing.T) {
	// Um e dois poderes: a frase do livro não deixa margem. Com um poder, perde-se 1;
	// com dois, cada um tem apenas UM outro poder, e "para cada DOIS outros" ainda não
	// se paga — 1 + 1 = 2.
	for _, tc := range []struct{ powers, want int }{{0, 0}, {1, 1}, {2, 2}} {
		if got := carismaLossFromPowers(tc.powers); got != tc.want {
			t.Errorf("%d poder(es): perda=%d, livro p136 diz %d", tc.powers, got, tc.want)
		}
	}
}

// A partir do TERCEIRO poder a frase admite duas leituras, e o motor escolheu uma:
// ele congela o custo no momento da escolha (o 3º poder custa 2, total 4), em vez de
// recalcular o custo de todos com a contagem final (que daria 6). As duas leituras
// cabem em "para cada dois outros poderes você perde mais 1", e nada no livro decide.
//
// Este teste fixa a leitura ESCOLHIDA, não a verdade: se a mesa decidir a outra, ele é
// o lugar que muda — e a intenção é que quem mudar leia isto antes.
func TestCarismaLossEscalationIsSequential(t *testing.T) {
	want := []int{0, 1, 2, 4, 6, 9, 12}
	for powers, expected := range want {
		if got := carismaLossFromPowers(powers); got != expected {
			t.Errorf("%d poderes: perda=%d, queria %d (sequência 1,2,4,6,9,12)", powers, got, expected)
		}
	}
}

// A Deformidade do Lefou (p23) TROCA uma habilidade por um poder da Tormenta, e esse
// poder conta para a perda de Carisma como qualquer outro — inclusive quando o mesmo
// id também aparece na lista de poderes escolhidos, onde contar duas vezes cobraria
// Carisma a mais por um poder só.
func TestTormentaCarismaLossCountsDeformidadeOnce(t *testing.T) {
	cases := []struct {
		name    string
		powers  []string
		swapped string
		want    int
	}{
		{"só a troca da Deformidade", nil, "dentes-afiados", 1},
		{"poder escolhido + troca distinta", []string{"antenas"}, "dentes-afiados", 2},
		{"o MESMO poder nos dois lugares conta uma vez", []string{"dentes-afiados"}, "dentes-afiados", 1},
		{"poder que não é da Tormenta não cobra nada", []string{"ataque-poderoso"}, "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &CharacterInput{PowerIDs: tc.powers}
			if tc.swapped != "" {
				in.Deformidade = &Deformidade{TormentaPower: tc.swapped}
			}
			if got := tormentaCarismaLoss(in); got != tc.want {
				t.Errorf("perda=%d, queria %d", got, tc.want)
			}
		})
	}
}

// A outra metade da Deformidade (p23): +2 em cada perícia escolhida. Os nomes chegam
// acentuados da ficha e o índice de perícias é sem acento — "Percepção" tem de virar
// `percepcao`, senão o bônus some em silêncio para metade das perícias do livro.
func TestDeformidadeSkillIDs(t *testing.T) {
	ids := deformidadeSkillIDs(&Deformidade{Pericias: []string{"Furtividade", "Percepção"}})
	if len(ids) != 2 || ids[0] != "furtividade" || ids[1] != "percepcao" {
		t.Fatalf("ids=%v, queria [furtividade percepcao]", ids)
	}
	if got := deformidadeSkillIDs(&Deformidade{Pericias: []string{"Perícia Inventada"}}); len(got) != 0 {
		t.Errorf("perícia inexistente virou %v, queria nada", got)
	}
	if got := deformidadeSkillIDs(nil); got != nil {
		t.Errorf("sem deformidade devolveu %v, queria nil", got)
	}
}

// Qual raça CARREGA a deformidade: o Lefou pode entrar como raça secundária, e uma
// raça sem a habilidade não pode ganhar o bônus de tabela por ter a escolha gravada
// na ficha — era a última coisa que só a cópia TS cobria.
func TestRaceWithDeformidade(t *testing.T) {
	catalogs := primeFromDump(t, filepath.Clean(filepath.Join(mustWd(t), "..", "parity")))

	if got := catalogs.raceWithDeformidade("Lefou"); got != "Lefou" {
		t.Errorf("raça primária: got=%q, queria Lefou", got)
	}
	if got := catalogs.raceWithDeformidade("Minotauro", "Lefou"); got != "Lefou" {
		t.Errorf("Lefou como secundária: got=%q, queria Lefou", got)
	}
	if got := catalogs.raceWithDeformidade("Humano", "Minotauro"); got != "" {
		t.Errorf("raça sem a habilidade devolveu %q, queria vazio", got)
	}
}
