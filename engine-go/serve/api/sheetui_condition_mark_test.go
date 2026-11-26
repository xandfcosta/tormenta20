package api

import (
	"strings"
	"testing"
)

// A FICHA RESPONDE COMO A MESA (decisão do dono): o mesmo personagem não recebe pela própria ficha a condição que a
// Mesa recusaria — o osteon é morto-vivo e imune a veneno (p29, p228), e o ladino de 4º nível nunca fica
// surpreendido (Esquiva Sobrenatural, p74).
func TestTheSheetRefusesWhatTheTableRefuses(t *testing.T) {
	t.Run("o osteon e o veneno", func(t *testing.T) {
		f := newSceneFixture(t)
		seedRaca(t, f.s, f.charID, "Osteon")
		if refusal := sceneRefusal(effect(t, f, f.charID, "condicao/envenenado").Body); !strings.Contains(refusal, "veneno") {
			t.Errorf("a ficha do osteon aceitou Envenenado, ou recusou sem dizer o tipo: %q", refusal)
		}
		if got := conditions(t, f, f.charID); len(got) != 0 {
			t.Errorf("a recusa gravou assim mesmo: %v", got)
		}
		// O CONTROLE: o osteon tem mente, e o medo entra nele.
		if refusal := sceneRefusal(effect(t, f, f.charID, "condicao/apavorado").Body); refusal != "" {
			t.Errorf("a ficha do osteon recusou Apavorado: %q", refusal)
		}
	})
	t.Run("o ladino e a surpresa", func(t *testing.T) {
		f := newSceneFixture(t)
		seedRaca(t, f.s, f.charID, "Humano")
		seedClasse(t, f.s, f.charID, "Ladino", 4)
		if refusal := sceneRefusal(effect(t, f, f.charID, "condicao/surpreendido").Body); !strings.Contains(refusal, "Esquiva Sobrenatural") {
			t.Errorf("a ficha do ladino de 4º nível aceitou Surpreendido, ou recusou sem dizer o poder: %q", refusal)
		}
		if got := conditions(t, f, f.charID); len(got) != 0 {
			t.Errorf("a recusa gravou assim mesmo: %v", got)
		}
	})
}
