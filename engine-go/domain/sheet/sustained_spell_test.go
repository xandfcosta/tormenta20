package sheet

import (
	"strings"
	"testing"
)

// "Você pode manter diversas habilidades sustentadas, pagando o custo de cada uma, mas apenas uma magia
// sustentada por vez" (p227). Com a Tempestade Divina de pé, o conjurador recusa a SEGUNDA sustentada e
// nomeia a que está de pé; a mesma magia, a que não é sustentada e a que um aprimoramento tira da
// sustentada passam.
func TestOnlyOneSustainedSpellIsCastAtATime(t *testing.T) {
	cleric := CharacterDTO{
		Classes:       []ClassDTO{{ClassName: "Clérigo", Level: 8}},
		ActiveEffects: []EffectDTO{{Source: "spell", CatalogID: "tempestade-divina", Scope: "sustained", CastHere: true}},
	}
	const clash = "Tempestade Divina já está sustentada"
	caster := CasterOf(cleric)

	if lock := caster.Spell("silencio").CastLock; !strings.Contains(lock, clash) {
		t.Errorf("o Conjurar do Silêncio, sustentado, tranca com %q; want a frase com %q", lock, clash)
	}
	for _, c := range []struct {
		why, spell string
		picks      []AugmentPick
	}{
		{"reconjurar a mesma não é uma segunda", "tempestade-divina", nil},
		{"a magia que não é sustentada", "curar-ferimentos", nil},
		// O aprimoramento 1 do Silêncio muda a duração para cena.
		{"o aprimoramento que tira o Silêncio da sustentada", "silencio", []AugmentPick{{AugmentIndex: 1, Stacks: 1}}},
	} {
		if refusal := caster.Spell(c.spell).CastRefusal(c.picks); refusal != "" {
			t.Errorf("%s: o conjurador recusa com %q", c.why, refusal)
		}
	}

	// A sustentada que DURA A CENA não é magia sustentada de pé: só o escopo conta.
	byScene := CasterOf(CharacterDTO{
		Classes:       cleric.Classes,
		ActiveEffects: []EffectDTO{{Source: "spell", CatalogID: "tempestade-divina", Scope: "scene", CastHere: true}},
	})
	if lock := byScene.Spell("silencio").CastLock; lock != "" {
		t.Errorf("com a Tempestade Divina valendo a cena, o Silêncio tranca com %q", lock)
	}

	// A sustentada SÓ APLICADA aqui veio de outro conjurador, e é ele quem a sustenta (decisão do dono).
	received := CasterOf(CharacterDTO{
		Classes:       cleric.Classes,
		ActiveEffects: []EffectDTO{{Source: "spell", CatalogID: "velocidade", Scope: "sustained"}},
	})
	if lock := received.Spell("silencio").CastLock; lock != "" {
		t.Errorf("com a Velocidade de outro conjurador aplicada aqui, o Silêncio tranca com %q", lock)
	}

	// O PODER SUSTENTADO não é magia: "você pode manter diversas habilidades sustentadas, pagando o custo de cada
	// uma, mas apenas uma magia sustentada por vez" (p227). A Aura Sagrada do paladino paga a manutenção e não
	// tranca a magia.
	aura := CasterOf(CharacterDTO{
		Classes:       cleric.Classes,
		ActiveEffects: []EffectDTO{{Source: "power", CatalogID: "class.paladino.aura-sagrada", Scope: "sustained"}},
	})
	if lock := aura.Spell("silencio").CastLock; lock != "" {
		t.Errorf("com a Aura Sagrada ligada, o Silêncio tranca com %q", lock)
	}
	if !aura.effects[0].Sustaining() {
		t.Error("a Aura Sagrada deixou de ser sustentada pela ficha, e a manutenção não a cobra")
	}
}

// O APRIMORAMENTO QUE TORNA A MAGIA SUSTENTADA passa pelo mesmo limite: a Invisibilidade é definida, e o
// aprimoramento 2 dela "muda a duração para sustentada" (p195). Sem ele ela se conjura; com ele, o
// conjurador recusa, e a folha tranca o aprimoramento.
func TestTheAugmentThatSustainsTheSpellMeetsTheLimit(t *testing.T) {
	arcanist := CharacterDTO{
		Classes:       []ClassDTO{{ClassName: "Arcanista", Level: 9}},
		ActiveEffects: []EffectDTO{{Source: "spell", CatalogID: "velocidade", Scope: "sustained", CastHere: true}},
	}
	spell := CasterOf(arcanist).Spell("invisibilidade")
	if refusal := spell.CastRefusal(nil); refusal != "" {
		t.Fatalf("a Invisibilidade sem aprimoramento não é sustentada, e o conjurador recusa com %q", refusal)
	}
	sphere := []AugmentPick{{AugmentIndex: 2, Stacks: 1}}
	if !spell.Sustains(sphere) {
		t.Fatal("o aprimoramento 2 da Invisibilidade não a torna sustentada")
	}
	if refusal := spell.CastRefusal(sphere); !strings.Contains(refusal, "Velocidade já está sustentada") {
		t.Errorf("com a Velocidade de pé, a esfera da Invisibilidade é recusada com %q", refusal)
	}
	if lock := spell.AugmentLock(2); lock == "" {
		t.Error("a folha deixa a esfera da Invisibilidade aberta com a Velocidade de pé")
	}
}
