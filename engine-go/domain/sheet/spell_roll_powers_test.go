package sheet

import (
	"reflect"
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// OS PODERES QUE MEXEM NO DADO DA MAGIA entram como parcela própria (ver GLOSSARY, **dado da magia**): o Arcano de
// Batalha soma o atributo-chave uma vez na rolagem de dano da conjuração (p38), e o Aspecto do Inverno soma +1 por
// dado de frio em todo dano de frio da magia, também depois da conjuração (p61).
func TestThePowersAddTheirOwnPartToTheSpellRoll(t *testing.T) {
	catalogs, err := book.EngineCatalogs()
	if err != nil {
		t.Fatalf("primar os catálogos: %v", err)
	}
	ruleset := engine.BookRuleset(catalogs)
	arcanist := CharacterDTO{Ruleset: ruleset,
		Classes: []ClassDTO{{ClassName: "Arcanista", Level: 10}}, Intelligence: 4,
		ClassPowers: `["class.arcanista.arcano-de-batalha"]`,
	}
	druid := CharacterDTO{Ruleset: ruleset,
		Classes: []ClassDTO{{ClassName: "Druida", Level: 10}}, Wisdom: 3,
		ClassPowers: `["class.druida.aspecto-do-inverno"]`,
	}
	powersOf := func(dto CharacterDTO) SpellRollPowers {
		t.Helper()
		computed, err := Compute(dto)
		if err != nil {
			t.Fatalf("computar a ficha: %v", err)
		}
		return SpellRollPowersOf(dto, computed)
	}
	fireball := []book.SpellRoll{{Dice: "6d6", Kind: "fogo"}}
	healing := []book.SpellRoll{{Dice: "2d8+2", Kind: "cura"}}
	frost := []book.SpellRoll{{Dice: "3d6", Kind: "frio"}, {Dice: "2d6", Kind: "corte"}}

	for _, c := range []struct {
		why   string
		got   []SpellBonus
		wants []SpellBonus
	}{
		{"o Arcano de Batalha na Bola de Fogo", powersOf(arcanist).CastBonuses(fireball),
			[]SpellBonus{{Name: "Arcano de Batalha", Amount: 4}}},
		{"o Arcano de Batalha não soma na cura", powersOf(arcanist).CastBonuses(healing), nil},
		{"o Arcano de Batalha é da conjuração, e não do dado de depois", powersOf(arcanist).LaterBonuses(fireball), nil},
		{"o Aspecto do Inverno soma por dado de frio", powersOf(druid).CastBonuses(frost),
			[]SpellBonus{{Name: "Aspecto do Inverno", Amount: 3, Kind: "frio", Note: "3 dados de frio"}}},
		{"o Aspecto do Inverno vale também no dado de depois", powersOf(druid).LaterBonuses(frost),
			[]SpellBonus{{Name: "Aspecto do Inverno", Amount: 3, Kind: "frio", Note: "3 dados de frio"}}},
		{"sem dado de frio, o Inverno não soma", powersOf(druid).CastBonuses(fireball), nil},
	} {
		if !reflect.DeepEqual(c.got, c.wants) {
			t.Errorf("%s: %+v; want %+v", c.why, c.got, c.wants)
		}
	}
}

// PV TEMPORÁRIOS não são dano: a Vitalidade Fantasma, que só os dá, não ganha o Arcano de Batalha (p38), e a Magia
// Sagrada/Profana não os troca por luz (p58).
func TestTemporaryHitPointsAreNotDamage(t *testing.T) {
	temporary := []book.SpellRoll{{Dice: "2d10", Kind: "PV temporários"}}
	powers := SpellRollPowers{battleArcane: 4}
	if bonuses := powers.CastBonuses(temporary); len(bonuses) != 0 {
		t.Errorf("a Vitalidade Fantasma ganhou as parcelas %+v; want nenhuma, porque PV temporários não são dano", bonuses)
	}
	if parts := (PoweredCast{DamageKind: "luz"}).Apply(temporary); parts[0].Kind != "PV temporários" {
		t.Errorf("a Magia Sagrada trocou os PV temporários por %q; want PV temporários", parts[0].Kind)
	}
}
