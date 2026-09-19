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
	dono := seedUser(t, s, "dono@t.com")

	casos := []struct {
		classe  string
		nivel   int64
		hpDano  int64
		mpGasto int64
	}{
		{"Guerreiro", 1, 0, 0},
		{"Guerreiro", 5, 12, 0},
		{"Arcanista", 8, 10, 5},
		{"Bardo", 5, 3, 2},
		{"Clérigo", 4, 0, 1},
	}
	for _, caso := range casos {
		t.Run(caso.classe, func(t *testing.T) {
			id := seedCharacterAtLevel(t, s, dono, "Herói", caso.classe, caso.nivel, caso.hpDano, caso.mpGasto)
			row, err := s.queries.GetCharacter(context.Background(), id)
			if err != nil {
				t.Fatalf("reler o personagem: %v", err)
			}
			poco := bookPools(t, s, caso.classe, caso.nivel)

			if row.Hpmax != poco.PvMax || row.Mpmax != poco.PmMax {
				t.Errorf("semeado com %d/%d de máximo, e o livro dá %d/%d",
					row.Hpmax, row.Mpmax, poco.PvMax, poco.PmMax)
			}
			if querido := poco.PvMax - caso.hpDano; row.Hpcurrent != querido {
				t.Errorf("PV atual = %d, e %d de dano sobre %d dá %d",
					row.Hpcurrent, caso.hpDano, poco.PvMax, querido)
			}
			if querido := poco.PmMax - caso.mpGasto; row.Mpcurrent != querido {
				t.Errorf("PM atual = %d, e %d gasto sobre %d dá %d",
					row.Mpcurrent, caso.mpGasto, poco.PmMax, querido)
			}

			// E a CLASSE existe, porque personagem sem classe é impossível nas
			// regras — e sem ela o poço do livro é zero.
			dto, err := sheet.Load(context.Background(), s.queries, s.catalogs, row)
			if err != nil {
				t.Fatalf("carregar o agregado: %v", err)
			}
			if len(dto.Classes) != 1 || dto.Classes[0].ClassName != caso.classe {
				t.Errorf("as classes do semeado são %+v, e o caso pediu %q", dto.Classes, caso.classe)
			}
		})
	}
}
