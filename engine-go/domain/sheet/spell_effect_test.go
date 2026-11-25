package sheet

import (
	"strings"
	"testing"
)

// O EFEITO DA MAGIA, lido (ver GLOSSARY, **efeito da magia**): o conjurador acha as magias em vigor da ficha e
// responde por cada uma com os aprimoramentos da conjuração gravada — o dado da vez, as ações, o dado de cada ação e
// as cargas. A linha que não é magia fica de fora, e a que não existe é recusada.
func TestTheSpellEffectAnswersWithTheCastItRecorded(t *testing.T) {
	caster := CasterOf(CharacterDTO{
		Classes: []ClassDTO{{ClassName: "Arcanista", Level: 20}},
		ActiveEffects: []EffectDTO{
			{ID: 1, Source: "power", CatalogID: "class.paladino.aura-sagrada", Scope: "sustained"},
			// O aprimoramento 0 do Ferver Sangue aumenta o dano em +1d8 (p193).
			{ID: 2, Source: "spell", CatalogID: "ferver-sangue", Scope: "sustained", CastHere: true,
				CastAugments: []AugmentPick{{AugmentIndex: 0, Stacks: 1}}},
			{ID: 3, Source: "spell", CatalogID: "mao-poderosa-de-talude", Scope: "sustained", CastHere: true,
				CastAugments: []AugmentPick{}},
			// O aprimoramento 1 da Lança Ígnea cria os quatro dardos (p196); este já gastou o último.
			{ID: 4, Source: "spell", CatalogID: "lanca-ignea-de-aleph", Scope: "scene", CastHere: true,
				CastAugments: []AugmentPick{{AugmentIndex: 1, Stacks: 1}}, Charges: 0},
		},
	})

	var ids []int64
	for _, e := range caster.SpellEffects() {
		ids = append(ids, e.Effect.ID)
	}
	if len(ids) != 3 || ids[0] != 2 || ids[1] != 3 || ids[2] != 4 {
		t.Errorf("as magias em vigor são as linhas %v; want [2 3 4], sem a Aura Sagrada, que é poder", ids)
	}

	fever, err := caster.SpellEffect(2)
	if err != nil {
		t.Fatalf("achar o Ferver Sangue: %v", err)
	}
	if parts, err := fever.TurnRoll(""); err != nil || rollsWritten(parts) != "[5d8 fogo]" {
		t.Errorf("o dado da vez do Ferver Sangue é %v (%v); want 5d8 de fogo, com o aumento da conjuração", rollsWritten(parts), err)
	}

	hand, err := caster.SpellEffect(3)
	if err != nil {
		t.Fatalf("achar a Mão Poderosa: %v", err)
	}
	if actions := hand.Actions(); actions == nil || actions.Options[0].Name != "Esmagar" {
		t.Fatalf("as ações da Mão Poderosa são %+v; want o Esmagar primeiro", actions)
	}
	if action, parts, err := hand.Action(0); err != nil || action.Name != "Esmagar" || rollsWritten(parts) != "[2d6+10 impacto]" {
		t.Errorf("o Esmagar rola %v (%v); want 2d6+10 de impacto (p198)", rollsWritten(parts), err)
	}

	lance, err := caster.SpellEffect(4)
	if err != nil {
		t.Fatalf("achar a Lança Ígnea: %v", err)
	}
	if left, counted := lance.Charges(); !counted || left != 0 {
		t.Errorf("a Lança diz %d cargas, contadas = %v; want 0, contadas", left, counted)
	}
	if _, _, err := lance.Action(0); err == nil || !strings.Contains(err.Error(), "descarregada") {
		t.Errorf("o dardo da Lança descarregada devolveu %v; want a recusa", err)
	}

	for _, id := range []int64{1, 99} {
		if _, err := caster.SpellEffect(id); err == nil {
			t.Errorf("a linha %d não é magia em vigor, e o conjurador a achou", id)
		}
	}
}

// A MÃO AGARRA E EMPURRA COMO MANOBRA (p198, p234; ver GLOSSARY, **ação da magia**): a opção não rola dado, diz a
// manobra do livro e o bônus que a mão soma ao Misticismo de quem conjurou, e o aprimoramento de 5º círculo muda
// esse bônus para +20.
func TestTheMightyHandGrabsAndPushesAsManeuvers(t *testing.T) {
	handWith := func(picks []AugmentPick) SpellEffect {
		t.Helper()
		caster := CasterOf(CharacterDTO{
			Classes: []ClassDTO{{ClassName: "Arcanista", Level: 20}},
			ActiveEffects: []EffectDTO{{ID: 7, Source: "spell", CatalogID: "mao-poderosa-de-talude", Scope: "sustained",
				CastHere: true, CastAugments: picks}},
		})
		hand, err := caster.SpellEffect(7)
		if err != nil {
			t.Fatalf("achar a Mão Poderosa: %v", err)
		}
		return hand
	}
	optionNamed := func(hand SpellEffect, name string) int {
		t.Helper()
		for i, o := range hand.Actions().Options {
			if o.Name == name {
				return i
			}
		}
		t.Fatalf("a Mão Poderosa não tem a ação %q", name)
		return -1
	}
	for _, c := range []struct {
		option, maneuver string
		picks            []AugmentPick
		bonus            int
	}{
		{"Agarrar", "agarrar", nil, 10},
		{"Empurrar", "empurrar", nil, 10},
		// O aprimoramento 1 da Mão muda o bônus em Misticismo para +20.
		{"Agarrar", "agarrar", []AugmentPick{{AugmentIndex: 1, Stacks: 1}}, 20},
	} {
		hand := handWith(c.picks)
		action, parts, err := hand.Action(optionNamed(hand, c.option))
		if err != nil {
			t.Fatalf("%s: %v", c.option, err)
		}
		if action.Maneuver != c.maneuver || action.Skill != "Misticismo" || action.Bonus != c.bonus || action.Action != "movimento" || len(parts) != 0 {
			t.Errorf("%s com %v é %+v e rola %v; want a manobra %s com Misticismo +%d, na ação de movimento, sem dado",
				c.option, c.picks, action, parts, c.maneuver, c.bonus)
		}
	}
}
