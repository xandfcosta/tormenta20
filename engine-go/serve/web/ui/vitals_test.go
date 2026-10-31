package ui

import "testing"

// Os LIMIARES da escada, e SÓ eles: a tabela inteira de porcentagens seria a
// implementação reescrita.
//
// Os números 25 e 50 estão escritos à mão de propósito: derivá-los de
// `hpCritical`/`hpHurt` faria a asserção andar junto com o defeito.
//
// # Cada degrau tem as DUAS bordas
//
// Prender `{25, crítico}` e `{26, ferido}` fecha só a fronteira de BAIXO: sem
// nada em 51, um `pct <= 51` no ramo do ferido passa verde. Cada degrau é
// afirmado dos dois lados.
func TestTheHpLadderTurnsAtTheThresholds(t *testing.T) {
	for _, c := range []struct {
		pct int
		tom string
	}{
		{0, "bg-hp-critical"},
		{25, "bg-hp-critical"},
		{26, "bg-hp-hurt"},
		{50, "bg-hp-hurt"},
		{51, "bg-hp-full"},
		{100, "bg-hp-full"},
	} {
		if got := HpFillTone(c.pct); got != c.tom {
			t.Errorf("HpFillTone(%d) = %q, queria %q", c.pct, got, c.tom)
		}
	}
}

// A PORCENTAGEM não divide por zero e não estoura a barra.
//
// O caso do máximo zero é o de quem não tem mana, e ele é a razão de a função
// existir separada: três superfícies escreviam esta mesma guarda à mão.
func TestTheVitalPercentIsPennedBetweenZeroAndOneHundred(t *testing.T) {
	for _, c := range []struct {
		current, max int64
		want         int
	}{
		{10, 57, 17},
		{57, 57, 100},
		{0, 57, 0},
		{5, 0, 0},   // sem pool: não é barra cheia nem vazia, é barra nenhuma
		{-3, 20, 0}, // o passo prende em zero, mas a barra não confia nele
		{30, 20, 100},
	} {
		if got := VitalPercent(c.current, c.max); got != c.want {
			t.Errorf("VitalPercent(%d, %d) = %d, queria %d", c.current, c.max, got, c.want)
		}
	}
}
