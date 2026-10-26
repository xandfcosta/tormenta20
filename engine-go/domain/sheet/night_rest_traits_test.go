package sheet

import "testing"

// O DESCANSO OBEDECE A QUEM DESCANSA (p106, ALE-488).
//
// Três regras do livro mudam a noite, cada uma escrita em mais de um verbete:
//
//   - "sua condição de descanso é uma categoria acima do padrão pela situação"
//     — Estoico (p94) e Pajem (p54). As duas NÃO acumulam (decisão do dono):
//     é o mesmo efeito escrito duas vezes;
//   - "sua recuperação de PV e PM nunca é inferior ao seu nível" — Rato das
//     Ruas (p23) e Vida Rústica (p94);
//   - "recupera PV e PM por descanso em condições normais (… não são afetados
//     por condições boas ou ruins de descanso)" — golem (p27) e osteon (p29).
//
// O Helior de 7º nível da p106 é a medida: 3 ao relento, 7 na estalagem, 14
// confortável, 21 luxuosa.
func TestTheNightBendsToWhoSleeps(t *testing.T) {
	for _, tt := range []struct {
		name      string
		condition string
		traits    RestTraits
		want      int64
	}{
		{"sem nada, ao relento", "ruim", RestTraits{}, 3},
		{"Estoico ao relento dorme como na estalagem", "ruim", RestTraits{OneCategoryUp: true}, 7},
		{"Estoico na estalagem dorme confortável", "normal", RestTraits{OneCategoryUp: true}, 14},
		{"Estoico confortável dorme luxuoso", "confortavel", RestTraits{OneCategoryUp: true}, 21},
		{"Estoico no luxo não passa do luxo", "luxuosa", RestTraits{OneCategoryUp: true}, 21},
		{"Rato das Ruas ao relento recupera o nível", "ruim", RestTraits{NeverBelowLevel: true}, 7},
		{"Rato das Ruas confortável segue o dobro", "confortavel", RestTraits{NeverBelowLevel: true}, 14},
		{"golem ao relento descansa normal", "ruim", RestTraits{AlwaysNormal: true}, 7},
		{"golem no luxo descansa normal", "luxuosa", RestTraits{AlwaysNormal: true}, 7},
		{"golem com Estoico continua normal", "normal", RestTraits{AlwaysNormal: true, OneCategoryUp: true}, 7},
	} {
		if got := NightRestGainFor(7, tt.condition, tt.traits); got != tt.want {
			t.Errorf("%s: %d, want %d", tt.name, got, tt.want)
		}
	}
}
