package api

import (
	"context"
	"net/http"
	"testing"
)

// A MANUTENÇÃO É DE QUEM CONJUROU (ver GLOSSARY, **magia sustentada**).

// QUEM RECEBE NÃO SUSTENTA: o aliado que aplica a Velocidade na própria ficha não paga a manutenção, que é de quem
// conjurou, nem fica impedido de receber a Oração de outro conjurador (decisão do dono).
func TestTheAllyWhoReceivesASustainedSpellDoesNotSustainIt(t *testing.T) {
	f, ally := fighterFixture(t)
	for _, path := range []string{"aplica/velocidade", "aplica/oracao"} {
		if rec := effect(t, f, ally, path); rec.Code != http.StatusOK || sceneRefusal(rec.Body) != "" {
			t.Fatalf("o aliado aplicar %s respondeu %d: %s", path, rec.Code, sceneRefusal(rec.Body))
		}
	}
	upkeep, err := (sheetVitals{q: f.s.queries}).SustainedOf(context.Background(), ally)
	if err != nil || len(upkeep) != 0 {
		t.Errorf("o aliado paga a manutenção de %v (%v); quem paga é quem conjurou", upkeep, err)
	}
}
