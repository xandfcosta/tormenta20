package api

import (
	"context"
	"testing"

	"t20engine/domain/sheet"
)

// O QUE A BANCADA SEMEIA É O QUE O LIVRO DÁ.
//
// # Por que este guarda existe
//
// O `seedCharacterAtLevel` escrevia os quatro vitais que o chamador passasse, e
// quarenta casos escolhiam os números à mão — um Arcanista de nível 8 com 30 de
// PV, quando o livro dá 42. Enquanto o PV máximo foi uma COLUNA isso passou
// despercebido: a bancada arranjava um personagem que a regra não produz, e
// media a tela desse personagem.
//
// O máximo está virando DERIVADO (ALE-355). Este guarda é o que impede a
// bancada de voltar a inventar: se alguém escrever um `hpMax` à mão de novo, o
// caso reprova aqui, e não em dezoito lugares espalhados quando a derivação
// entrar.
//
// # E o DANO é o que sobra
//
// A outra metade: o atual semeado tem de ser o poço menos o dano pedido. Sem
// ela, um ajudante que ignorasse o dano passaria — todo personagem nasceria
// cheio e os casos de "ferido" mediriam um herói intacto.
func TestTheBenchSeedsThePoolsTheBookGives(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t.com")

	cases := []struct {
		class    string
		level    int64
		hpDamage int64
		mpSpent  int64
	}{
		{"Guerreiro", 1, 0, 0},
		{"Guerreiro", 5, 12, 0},
		{"Arcanista", 8, 10, 5},
		{"Bardo", 5, 3, 2},
		{"Clérigo", 4, 0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.class, func(t *testing.T) {
			id := seedCharacterAtLevel(t, s, owner, "Herói", tc.class, tc.level, tc.hpDamage, tc.mpSpent)
			row, err := s.queries.GetCharacter(context.Background(), id)
			if err != nil {
				t.Fatalf("reler o personagem: %v", err)
			}
			pool := bookPools(t, s, tc.class, tc.level)

			seeded := poolsOf(t, s, id)
			if seeded.HpMax != pool.PvMax || seeded.MpMax != pool.PmMax {
				t.Errorf("semeado com %d/%d de máximo, e o livro dá %d/%d",
					seeded.HpMax, seeded.MpMax, pool.PvMax, pool.PmMax)
			}
			if wanted := pool.PvMax - tc.hpDamage; seeded.HpCurrent != wanted {
				t.Errorf("PV atual = %d, e %d de dano sobre %d dá %d",
					seeded.HpCurrent, tc.hpDamage, pool.PvMax, wanted)
			}
			if wanted := pool.PmMax - tc.mpSpent; seeded.MpCurrent != wanted {
				t.Errorf("PM atual = %d, e %d gasto sobre %d dá %d",
					seeded.MpCurrent, tc.mpSpent, pool.PmMax, wanted)
			}

			// E a CLASSE existe, porque personagem sem classe é impossível nas
			// regras — e sem ela o poço do livro é zero.
			dto, err := sheet.Load(context.Background(), s.queries, s.catalogs, row)
			if err != nil {
				t.Fatalf("carregar o agregado: %v", err)
			}
			if len(dto.Classes) != 1 || dto.Classes[0].ClassName != tc.class {
				t.Errorf("as classes do semeado são %+v, e o caso pediu %q", dto.Classes, tc.class)
			}
		})
	}
}
