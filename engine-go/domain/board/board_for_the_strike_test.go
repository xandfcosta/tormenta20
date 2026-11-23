package board

import "testing"

// O TABULEIRO DO GOLPE é o primeiro, na ordem das abas, que tem as duas peças; sem ele, o primeiro que tem o alvo
// (`BoardForTheStrike`).
func TestTheStrikeBoardIsTheFirstWithBothPiecesOrElseTheTargets(t *testing.T) {
	piece := func(entry string) BoardToken { return BoardToken{ID: "peca-" + entry, EntryID: &entry, Footprint: 1} }
	tavern := &BoardState{ID: "taverna"}
	crypt := &BoardState{ID: "cripta"}
	target := AttackTargetOnTheBoard{EntryID: "alvo"}
	for _, c := range []struct {
		why           string
		tavern, crypt []BoardToken
		want          string
	}{
		{"as duas na cripta", nil, []BoardToken{piece("atk"), piece("alvo")}, "cripta"},
		{"as duas nos dois: vale a primeira aba", []BoardToken{piece("atk"), piece("alvo")},
			[]BoardToken{piece("atk"), piece("alvo")}, "taverna"},
		{"uma em cada: o do alvo", []BoardToken{piece("atk")}, []BoardToken{piece("alvo")}, "cripta"},
		{"só o alvo no mapa: o do alvo", nil, []BoardToken{piece("alvo")}, "cripta"},
		{"o alvo nos dois e quem ataca na cripta: a cripta", []BoardToken{piece("alvo")},
			[]BoardToken{piece("atk"), piece("alvo")}, "cripta"},
		{"só quem ataca no mapa: nenhum", []BoardToken{piece("atk")}, nil, ""},
	} {
		tavern.Tokens, crypt.Tokens = c.tavern, c.crypt
		got := ""
		if b := BoardForTheStrike([]*BoardState{tavern, crypt}, "atk", target); b != nil {
			got = b.ID
		}
		if got != c.want {
			t.Errorf("%s: o golpe leu %q; want %q", c.why, got, c.want)
		}
	}
}
