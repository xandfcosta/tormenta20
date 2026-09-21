package sheet

import "testing"

// O EXEMPLO TRABALHADO do livro, e ele é o único caso que separa duas leituras
// da mesma frase (T20 p106): Helior, 7º nível, recupera 7 numa estalagem e 3 ao
// relento. Metade de 7 é 3,5 — arredondar para cima daria 4, e o livro escreve
// 3. Uma asserção inventada não pegaria a diferença.
func TestHeliorRecoversWhatTheBookSays(t *testing.T) {
	if gain := NightRestGain(7, "normal"); gain != 7 {
		t.Errorf("o Helior na estalagem recuperou %d, e o livro diz 7 (p106)", gain)
	}
	if gain := NightRestGain(7, "ruim"); gain != 3 {
		t.Errorf("o Helior ao relento recuperou %d, e o livro diz 3 — metade de 7 DESCE (p106)", gain)
	}
}

func TestEachLodgingHasItsFactor(t *testing.T) {
	for _, tc := range []struct {
		tc        string
		condition string
		level     int64
		want      int64
	}{
		{"ruim é metade do nível", "ruim", 10, 5},
		{"normal é o nível", "normal", 10, 10},
		{"confortável é o dobro", "confortavel", 10, 20},
		{"luxuosa é o triplo", "luxuosa", 10, 30},
		{"condição que o servidor não conhece vira normal", "palacete", 10, 10},
		{"nível 1 ao relento não recupera nada, e isso é a regra", "ruim", 1, 0},
	} {
		t.Run(tc.tc, func(t *testing.T) {
			if gain := NightRestGain(tc.level, tc.condition); gain != tc.want {
				t.Errorf("nível %d em %q deu %d, e o caso pede %d",
					tc.level, tc.condition, gain, tc.want)
			}
		})
	}
}

// O TETO é do livro: "não pode ultrapassar seu máximo".
func TestTheNightNeverGoesPastTheMaximum(t *testing.T) {
	after := AfterNightRest(7, "luxuosa", RestedVitals{HpCurrent: 10, MpCurrent: 2}, 20, 5)
	if after.HpCurrent != 20 {
		t.Errorf("o PV parou em %d, e o máximo é 20", after.HpCurrent)
	}
	if after.MpCurrent != 5 {
		t.Errorf("o PM parou em %d, e o máximo é 5", after.MpCurrent)
	}
}

// Ficha ferida que não chega ao máximo recebe o ganho inteiro.
func TestAWoundedSheetGetsTheWholeGain(t *testing.T) {
	after := AfterNightRest(7, "normal", RestedVitals{HpCurrent: 1, MpCurrent: 0}, 40, 40)
	if after.HpCurrent != 8 || after.MpCurrent != 7 {
		t.Errorf("deu %+v, e o caso pede 8 PV e 7 PM", after)
	}
}
