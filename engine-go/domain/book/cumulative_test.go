package book

import "testing"

// O VOCABULÁRIO DO BÔNUS CUMULATIVO é fechado, e o guarda é quem o fecha.
//
// A família inteira casa por TEXTO — o gatilho, o teto, o alvo —, e texto que
// não casa não estoura: o poder simplesmente nunca acumula, com a suíte verde e
// a ficha mostrando um verbete que não faz nada. Era exatamente esse o estado
// da Sangue dos Inimigos antes da ALE-423, e o guarda existe para que o
// PRÓXIMO poder cumulativo não repita a história em silêncio.
//
// DENOMINADOR: ele diz quantos olhou. Com zero declarações, "nada reprovou" e
// "não mediu nada" são a mesma cor.
func TestEveryCumulativeBonusDeclaresATriggerTheEngineKnows(t *testing.T) {
	measured := 0
	for _, spec := range Activations() {
		if spec.Cumulative == nil {
			continue
		}
		measured++
		if !CumulativeTriggerIsKnown(spec.Cumulative.On) {
			t.Errorf("%s declara o gatilho %q, que o motor não sabe disparar",
				spec.ID, spec.Cumulative.On)
		}
		// O TETO SEM ESPÉCIE CONHECIDA devolve zero, e zero quer dizer "não
		// acumula" — um poder declarado e inerte, que é o defeito que esta
		// fatia veio consertar.
		if CumulativeCap(*spec.Cumulative, 20) <= 0 {
			t.Errorf("%s tem teto %q, que o motor lê como zero: o bônus nunca sai do lugar",
				spec.ID, spec.Cumulative.CapPer)
		}
		if spec.Cumulative.Amount <= 0 {
			t.Errorf("%s acumula %d por gatilho", spec.ID, spec.Cumulative.Amount)
		}
		if len(spec.Cumulative.Targets) == 0 {
			t.Errorf("%s não diz que número ele move", spec.ID)
		}
		for _, target := range spec.Cumulative.Targets {
			if target.K == "" {
				t.Errorf("%s declara um alvo sem espécie", spec.ID)
			}
		}
	}
	if measured == 0 {
		t.Fatal("nenhuma ativação declara bônus cumulativo — o guarda não mediu nada")
	}
}

// O TETO É TETO, e o caso prende a aritmética que a p42 pede — *"um bônus
// cumulativo de +1… limitado pelo seu nível"*.
//
// UNITÁRIO porque isto é REGRA, e porque o caso de borda que importa é o que
// não se alcança de fora sem sete ataques: o gatilho no teto não devolve o
// teto mais um, e também não devolve menos.
func TestTheCumulativeBonusNeverPassesTheCap(t *testing.T) {
	spec := CumulativeBonus{On: "criticalOrDrop", Amount: 1, CapPer: "level"}
	for _, caso := range []struct {
		current, level, want int
	}{
		{0, 6, 1},
		{5, 6, 6},
		{6, 6, 6},
		{9, 6, 6}, // já passou do teto por outro caminho: ele não sobe mais
		{0, 1, 1},
	} {
		if got := CumulativeNext(spec, caso.current, caso.level); got != caso.want {
			t.Errorf("de +%d no nível %d esperava +%d e veio +%d",
				caso.current, caso.level, caso.want, got)
		}
	}
}
