package api

import (
	"fmt"
	"net/http"
	"testing"
)

// O ALQUEBRADO NA FICHA (ALE-647): "o custo em pontos de mana das habilidades do personagem aumenta em +1" (p394).
// Entrar na Fúria cobra um PM a mais do bárbaro Alquebrado do que do mesmo bárbaro sem a condição.
func TestTheBrokenConditionChargesOneMoreToEnterTheRage(t *testing.T) {
	spent := func(broken bool) int64 {
		f, id := barbaro(t, 10)
		if broken {
			if rec := effect(t, f, id, "condicao/alquebrado"); rec.Code != http.StatusOK {
				t.Fatalf("aplicar o Alquebrado: status %d", rec.Code)
			}
		}
		before := mpOf(t, f.s, id)
		rage := f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/poderes/postura/furia/entra?tab=abilities", id), "")
		if refusal := sceneRefusal(rage.Body.String()); rage.Code != http.StatusOK || refusal != "" {
			t.Fatalf("entrar na Fúria (Alquebrado: %v): status %d, recusa %q", broken, rage.Code, refusal)
		}
		return before - mpOf(t, f.s, id)
	}
	plain, broken := spent(false), spent(true)
	if plain <= 0 || broken != plain+1 {
		t.Errorf("a Fúria custou %d PM, e Alquebrado %d; want um PM a mais", plain, broken)
	}
}
