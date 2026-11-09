package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/sheet"
)

// O PV MÁXIMO SE MOVE POR OUTROS GESTOS ALÉM DO NÍVEL E DO ANEL: a Constituição soma em todo nível (p17),
// e entrar numa classe soma os PV dela (p35). O PV atual é o máximo menos o dano, e quem chega a 0 ou
// menos cai, por qualquer caminho (p236) — a mesma regra que a queda de nível segue (decisão do dono).

// sheetGesture faz um gesto da ficha e para o teste se ele foi recusado.
func sheetGesture(t *testing.T, f sceneFixture, path string) {
	t.Helper()
	rec := f.requests(t, f.player, http.MethodPost, path+"?tab=combat", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("%s respondeu %d", path, rec.Code)
	}
	if refusal := sceneRefusal(rec.Body.String()); refusal != "" {
		t.Fatalf("%s foi recusado: %q", path, refusal)
	}
}

// activeConditionsOf é o que a ficha guarda como condição, cru.
func activeConditionsOf(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler a ficha: %v", err)
	}
	return row.Activeconditions
}

// isDown diz se a ficha tem as duas condições de quem caiu (p236).
func isDown(conditions string) bool {
	return strings.Contains(conditions, "inconsciente") && strings.Contains(conditions, "sangrando")
}

func TestLoweringConstitutionWhileHurtKnocksTheCharacterDown(t *testing.T) {
	f, id := arcanistaDeNivel(t, 5)
	constitution := func(value int) string { return fmt.Sprintf("/personagens/%d/atributos/constitution/%d", id, value) }
	sheetGesture(t, f, constitution(0))
	arrangePools(t, f.s, id, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.HpCurrent = 2
		return pools, nil
	})
	maxBefore := poolsOf(t, f.s, id).HpMax

	sheetGesture(t, f, constitution(-2))

	after := poolsOf(t, f.s, id)
	if lost := maxBefore - after.HpMax; lost <= 2 || after.HpCurrent != 2-lost {
		t.Fatalf("com 2 PV e o máximo indo de %d a %d, a ficha ficou com %d PV", maxBefore, after.HpMax, after.HpCurrent)
	}
	if got := activeConditionsOf(t, f, id); !isDown(got) {
		t.Errorf("a ficha caiu sem as condições da p236: %s", got)
	}

	// Subir de volta traz o PV para 2, e quem chega a 1 PV acorda (p236).
	sheetGesture(t, f, constitution(0))
	if got := poolsOf(t, f.s, id).HpCurrent; got != 2 {
		t.Fatalf("de volta à Constituição 0, a ficha tem %d PV, want 2", got)
	}
	if got := activeConditionsOf(t, f, id); strings.Contains(got, "inconsciente") || strings.Contains(got, "sangrando") {
		t.Errorf("de pé, a ficha continua com as condições de quem caiu: %s", got)
	}
}
