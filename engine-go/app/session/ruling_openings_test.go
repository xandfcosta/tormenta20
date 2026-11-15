package session

import (
	"testing"

	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O VERBETE E A MESA FALAM A MESMA LÍNGUA: cada desfecho que um verbete pode pedir (`engine.OpensOn…`) é aberto
// por alguma decisão que o mestre grava. O regime não importa o motor, e é aqui, onde os dois se encontram, que a
// tradução é conferida: um `On` novo no motor que a mesa não conhecesse armaria uma manobra que nunca abre.
func TestEveryFollowUpTriggerIsOpenedBySomeRuling(t *testing.T) {
	rulings := []string{live.RulingMiss, live.RulingHit, live.RulingCritical, live.RulingSuccess, live.RulingFailure}
	for _, on := range []string{engine.OpensOnHit, engine.OpensOnSuccess} {
		opened := false
		for _, ruling := range rulings {
			opened = opened || live.RulingOpens(ruling, on)
		}
		if !opened {
			t.Errorf("o desfecho %q do verbete não é aberto por nenhuma decisão do mestre: ensine o `live.RulingOpens`", on)
		}
	}
	if live.RulingOpens(live.RulingMiss, engine.OpensOnHit) || live.RulingOpens(live.RulingFailure, engine.OpensOnSuccess) {
		t.Error("o erro abriu o que pede acerto, ou a Falha abriu o que pede Sucesso")
	}
	if !live.RulingOpens(live.RulingCritical, engine.OpensOnHit) {
		t.Error("o crítico é um acerto, e não abriu o que pede acerto")
	}
}
