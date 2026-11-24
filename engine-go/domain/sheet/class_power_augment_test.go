package sheet

import (
	"strings"
	"testing"
)

// OS PODERES DE CLASSE QUE SE PAGAM AO CONJURAR entram no diálogo de conjurar como os gerais: a Magia
// Pungente (p38), a Prece de Combate, a Magia Sagrada/Profana e a Comunhão Vital (p58), o Espírito dos
// Solstícios e a Liberdade da Pradaria (p62). O app tranca só pelo que a ficha e a magia dizem.

func powerLocksOf(t *testing.T, dto CharacterDTO, spellID string) map[string]string {
	t.Helper()
	locks := map[string]string{}
	for _, p := range CasterOf(dto).Spell(spellID).PowerAugments {
		locks[p.Power.Name] = p.Lock
	}
	return locks
}

func TestTheClericsPowersAreAugmentsOfTheDivineSpell(t *testing.T) {
	cleric := CharacterDTO{
		Classes:     []ClassDTO{{ClassName: "Clérigo", Level: 8}},
		ClassPowers: `["class.clerigo.prece-de-combate","class.clerigo.magia-sagrada-profana","class.clerigo.comunhao-vital"]`,
	}
	// A Curar Ferimentos é divina e de ação padrão (p189): as três se aplicam.
	locks := powerLocksOf(t, cleric, "curar-ferimentos")
	for _, name := range []string{"Prece de Combate", "Magia Sagrada/Profana", "Comunhão Vital"} {
		lock, listed := locks[name]
		if !listed {
			t.Fatalf("a Curar Ferimentos não lista %s: %v", name, locks)
		}
		if lock != "" {
			t.Errorf("%s tranca a Curar Ferimentos com %q", name, lock)
		}
	}
	// O Escudo da Fé é reação (p192): a Prece só vale na magia de ação padrão.
	if lock := powerLocksOf(t, cleric, "escudo-da-fe")["Prece de Combate"]; !strings.Contains(lock, "ação padrão") {
		t.Errorf("a Prece no Escudo da Fé tranca com %q", lock)
	}

	// "Quando lança uma magia divina" (p58): o Sono do multiclasse é arcano (p207).
	multiclass := cleric
	multiclass.Classes = []ClassDTO{{ClassName: "Clérigo", Level: 8}, {ClassName: "Arcanista", Level: 1}}
	if lock := powerLocksOf(t, multiclass, "sono")["Prece de Combate"]; lock != "apenas divinos" {
		t.Errorf("a Prece no Sono arcano tranca com %q; want \"apenas divinos\"", lock)
	}

	pm, refusal := CasterOf(cleric).Spell("curar-ferimentos").AugmentPrice(nil, []PowerAugmentPick{
		{PowerID: "class.clerigo.prece-de-combate", Stacks: 1}, {PowerID: "class.clerigo.comunhao-vital", Stacks: 1},
	})
	if pm != 4 || refusal != "" {
		t.Errorf("a Prece e a Comunhão custam %d PM, recusa %q; want 4 (2 + 2)", pm, refusal)
	}
}

func TestTheArcanistAndDruidPowersLockByTheSpell(t *testing.T) {
	arcanist := CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 8}}, ClassPowers: `["class.arcanista.magia-pungente"]`}
	// O Sono pede Vontade (p207); a Armadura Arcana não pede resistência nenhuma (p181).
	if lock := powerLocksOf(t, arcanist, "sono")["Magia Pungente"]; lock != "" {
		t.Errorf("a Pungente tranca o Sono com %q", lock)
	}
	if lock := powerLocksOf(t, arcanist, "armadura-arcana")["Magia Pungente"]; !strings.Contains(lock, "teste de resistência") {
		t.Errorf("a Pungente na Armadura Arcana tranca com %q", lock)
	}
	// Uma vez por conjuração (decisão do dono): +2 na CD por 1 PM, e não +4 por 2.
	if _, refusal := CasterOf(arcanist).Spell("sono").AugmentPrice(nil, []PowerAugmentPick{
		{PowerID: "class.arcanista.magia-pungente", Stacks: 2},
	}); !strings.Contains(refusal, "não se acumula") {
		t.Errorf("a Pungente paga duas vezes foi recusada com %q", refusal)
	}

	druid := CharacterDTO{
		Classes:      []ClassDTO{{ClassName: "Druida", Level: 9}},
		ClassPowers:  `["class.druida.liberdade-da-pradaria"]`,
		PowerChoices: `{"class.druida.magias-1-circulo":["convocacao","transmutacao","abjuracao"]}`,
	}
	// Toque, curto ou médio sobem um passo (p62); o alcance pessoal não tem passo.
	if lock := powerLocksOf(t, druid, "controlar-plantas")["Liberdade da Pradaria"]; lock != "" {
		t.Errorf("a Pradaria tranca o Controlar Plantas com %q", lock)
	}
	// A Arma Espiritual é pessoal (p180).
	if lock := powerLocksOf(t, druid, "arma-espiritual")["Liberdade da Pradaria"]; !strings.Contains(lock, "alcance") {
		t.Errorf("a Pradaria na Arma Espiritual tranca com %q", lock)
	}
}
