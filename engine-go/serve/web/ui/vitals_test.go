package ui

import "testing"

// Os LIMIARES da escada, e SÓ eles: a tabela inteira de porcentagens seria a
// implementação reescrita.
//
// 25 e 50 são os mesmos do `hpFillVar` da SPA, e os números estão escritos à
// mão de propósito — derivá-los de `hpCritical`/`hpHurt` faria a asserção andar
// junto com o defeito.
//
// # Cada degrau tem as DUAS bordas
//
// Aqui morava a versão da Mesa, com seis casos. Ela prendia `{25, crítico}` e
// `{26, ferido}`, o que é a fronteira de baixo — e não tinha nada em 51: um
// `pct <= 51` no ramo do ferido passaria verde. Esta versão fecha os dois lados
// de cada degrau, que é a lição que a ALE-315 deixou escrita sobre o teto do
// traço.
func TestTheHpLadderTurnsAtTheThresholds(t *testing.T) {
	for _, c := range []struct {
		pct   int
		tom   string
		tinta string
	}{
		{0, "bg-hp-critical", "text-grimorio-crimson-bright"},
		{25, "bg-hp-critical", "text-grimorio-crimson-bright"},
		{26, "bg-hp-hurt", "text-hp-hurt"},
		{50, "bg-hp-hurt", "text-hp-hurt"},
		{51, "bg-hp-full", "text-hp-full"},
		{100, "bg-hp-full", "text-hp-full"},
	} {
		if got := HpFillTone(c.pct); got != c.tom {
			t.Errorf("HpFillTone(%d) = %q, queria %q", c.pct, got, c.tom)
		}
		// A TINTA é outra escada, e ela diverge da de preencher em UM degrau
		// (ALE-240): o crítico escreve com a tinta de perigo da casa, porque o
		// `--hp-critical` dá 4,11:1 como letra pequena.
		if got := HpInkTone(c.pct); got != c.tinta {
			t.Errorf("HpInkTone(%d) = %q, queria %q", c.pct, got, c.tinta)
		}
	}
	// E o degrau em que as duas escadas DIVERGEM, afirmado sozinho: sem isto,
	// uma `HpInkTone` que devolvesse a classe de preencher com outro prefixo
	// passaria na tabela acima inteira.
	if HpInkTone(10) == "text-hp-critical" {
		t.Error("a tinta do crítico é a de preencher com outro prefixo, e ela dá 4,11:1 como letra")
	}
}

// A PORCENTAGEM não divide por zero e não estoura a barra.
//
// O caso do máximo zero é o de quem não tem mana, e ele é a razão de a função
// existir separada: três superfícies escreviam esta mesma guarda à mão.
func TestTheVitalPercentIsPennedBetweenZeroAndOneHundred(t *testing.T) {
	for _, c := range []struct {
		atual, max int64
		quer       int
	}{
		{10, 57, 17},
		{57, 57, 100},
		{0, 57, 0},
		{5, 0, 0},   // sem pool: não é barra cheia nem vazia, é barra nenhuma
		{-3, 20, 0}, // o passo prende em zero, mas a barra não confia nele
		{30, 20, 100},
	} {
		if got := VitalPercent(c.atual, c.max); got != c.quer {
			t.Errorf("VitalPercent(%d, %d) = %d, queria %d", c.atual, c.max, got, c.quer)
		}
	}
}
