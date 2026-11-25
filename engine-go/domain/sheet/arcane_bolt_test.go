package sheet

import (
	"strings"
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// O RAIO ARCANO (p39): 1d8 de essência mais 1d8 por círculo acima do 1º que o arcanista lança, em alcance curto, e
// Reflexos reduz à metade. O Raio Poderoso troca para d12 e alcance médio; o Raio Elemental troca o tipo, e quem
// falha sofre a condição do tipo.
func TestTheArcaneBoltGrowsWithTheCircleAndThePowers(t *testing.T) {
	catalogs, err := book.EngineCatalogs()
	if err != nil {
		t.Fatalf("primar os catálogos: %v", err)
	}
	boltOf := func(powers string) (ArcaneBolt, bool) {
		t.Helper()
		dto := CharacterDTO{
			Ruleset: engine.BookRuleset(catalogs), Classes: []ClassDTO{{ClassName: "Arcanista", Level: 10}},
			Intelligence: 4, ClassPowers: powers,
		}
		computed, err := Compute(dto)
		if err != nil {
			t.Fatalf("computar a ficha: %v", err)
		}
		return ArcaneBoltOf(dto, computed)
	}

	if _, has := boltOf(`[]`); has {
		t.Fatal("quem não tem o Raio Arcano o dispara")
	}
	plain, has := boltOf(`["class.arcanista.raio-arcano"]`)
	if !has || plain.Dice != "3d8" || plain.Range != "curto" || plain.DC == 0 || plain.Elemental {
		t.Fatalf("o Raio Arcano do arcanista de 10º (3º círculo) é %+v; want 3d8, alcance curto, com CD, sem o elemental", plain)
	}
	if parts, err := plain.Roll(""); err != nil || len(parts) != 1 || parts[0].Dice != "3d8" || parts[0].Kind != "essência" {
		t.Errorf("o Raio Arcano rola %+v (%v); want 3d8 de essência", parts, err)
	}
	if _, err := plain.Roll("fogo"); err == nil {
		t.Error("o Raio Arcano sem o Raio Elemental mudou para fogo")
	}

	mighty, _ := boltOf(`["class.arcanista.raio-arcano","class.arcanista.raio-poderoso"]`)
	if mighty.Dice != "3d12" || mighty.Range != "médio" {
		t.Errorf("o Raio Poderoso é %+v; want 3d12, alcance médio", mighty)
	}

	elemental, _ := boltOf(`["class.arcanista.raio-arcano","class.arcanista.raio-elemental"]`)
	parts, err := elemental.Roll("fogo")
	if err != nil || parts[0].Kind != "fogo" || !strings.Contains(parts[0].Note, "em chamas") {
		t.Errorf("o Raio Elemental de fogo rola %+v (%v); want fogo, e quem falha fica em chamas", parts, err)
	}
	if _, err := elemental.Roll("luz"); err == nil {
		t.Error("o Raio Elemental aceitou luz, que não é um dos cinco tipos")
	}
}
