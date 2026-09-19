package sheet

import "testing"

// A DIFERENÇA entre as duas é o que cada teste tem de separar, e o caso que a
// separa é o herói FERIDO ganhando máximo.
func TestTheWoundedHeroLevelsUpWithoutBeingHealed(t *testing.T) {
	ferido := Vitals{HpMax: 20, HpCurrent: 5, MpMax: 10, MpCurrent: 2}

	// Subir de nível: o máximo sobe 10, e o atual sobe os MESMOS 10.
	subiu, mudou := ShiftedByNewMax(ferido, 30, 15)
	if !mudou || subiu.HpCurrent != 15 || subiu.MpCurrent != 7 {
		t.Errorf("subir de nível deu %+v; o ferido ganha o delta, não a cura", subiu)
	}

	// Nascer/curar: o máximo sobe e o atual fica ONDE ESTAVA.
	preso, mudou := ClampedToNewMax(ferido, 30, 15)
	if !mudou || preso.HpCurrent != 5 || preso.MpCurrent != 2 {
		t.Errorf("prender na faixa deu %+v; o atual não acompanha", preso)
	}
}

// O CICLO `−` e `+` do passo de atributo tem de voltar ao mesmo lugar. Com
// "prende na faixa" nos dois sentidos ele devolveria PV de graça a cada volta.
func TestTheAttributeStepIsReversible(t *testing.T) {
	inicio := Vitals{HpMax: 20, HpCurrent: 12, MpMax: 10, MpCurrent: 4}

	menos, _ := ShiftedByNewMax(inicio, 18, 9)
	volta, _ := ShiftedByNewMax(menos, 20, 10)

	if volta != inicio {
		t.Errorf("a volta deu %+v e o começo era %+v — o ciclo criou pontos", volta, inicio)
	}
}

// O ATUAL nunca passa do máximo nem cai abaixo de zero, nas duas.
func TestNeitherRuleLeavesTheCurrentOutsideTheRange(t *testing.T) {
	cheio := Vitals{HpMax: 20, HpCurrent: 20, MpMax: 10, MpCurrent: 10}
	encolheu, mudou := ClampedToNewMax(cheio, 8, 3)
	if !mudou || encolheu.HpCurrent != 8 || encolheu.MpCurrent != 3 {
		t.Errorf("máximo menor deu %+v; o atual tem de ser preso no teto novo", encolheu)
	}

	quaseMorto := Vitals{HpMax: 20, HpCurrent: 1, MpMax: 10, MpCurrent: 0}
	caiu, _ := ShiftedByNewMax(quaseMorto, 5, 2)
	if caiu.HpCurrent < 0 || caiu.MpCurrent < 0 {
		t.Errorf("o delta negativo levou o atual abaixo de zero: %+v", caiu)
	}
}

// NADA MUDOU é resposta, e quem chama grava só quando mudou: um `UPDATE` sem
// mudança carimbaria um `updatedAt` que a Mesa lê para repedir a ficha.
func TestNothingChangedIsAnAnswer(t *testing.T) {
	mesmo := Vitals{HpMax: 20, HpCurrent: 12, MpMax: 10, MpCurrent: 4}
	if _, mudou := ClampedToNewMax(mesmo, 20, 10); mudou {
		t.Error("prender na faixa disse que mudou, e nada mudou")
	}
	if _, mudou := ShiftedByNewMax(mesmo, 20, 10); mudou {
		t.Error("acompanhar o delta disse que mudou, e nada mudou")
	}
}
