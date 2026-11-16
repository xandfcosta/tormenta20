package api

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O GESTO QUE MEXE EM NÚMERO DE QUEM O USA aplica o número: o PM comprou o efeito, e a conta é do app (o que
// o efeito faz nos OUTROS fica com o mestre).

// paladinWithCharisma é um paladino de `level` com Carisma 3, com o PM cheio e na mesa — a ÚNICA ficha do
// jogador, para a mesa abrir na dela.
func paladinWithCharisma(t *testing.T, level int64) (sceneFixture, int64) {
	t.Helper()
	s := newTestServer(t)
	f := sceneFixture{s: s, gm: seedUser(t, s, "mestre@t.com"), player: seedUser(t, s, "jogador@t.com")}
	f.campaignID = seedCampaign(t, s, f.gm)
	f.sessionID = seedSession(t, s, f.campaignID)
	id, err := f.s.sceneCore().Queries().CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.player, Name: "Devota", Origin: "batedor", Level: level,
		Strength: 3, Dexterity: 1, Constitution: 2, Intelligence: 0, Wisdom: 1, Charisma: 3,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassChoices: "{}", CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear a paladina: %v", err)
	}
	seedClasse(t, f.s, id, "Paladino", level)
	arrangePools(t, f.s, id, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.HpCurrent, pools.MpCurrent = pools.HpMax, pools.MpMax
		return pools, nil
	})
	seedMember(t, f.s, f.campaignID, id)
	f.charID = id
	return f, id
}

// A AURA SAGRADA soma o Carisma nas resistências de quem está nela (p84), e a paladina está nela.
func TestTheHolyAuraAddsCharismaToThePaladinsResistances(t *testing.T) {
	f, id := paladinWithCharisma(t, 3)
	before := statOf(t, f, id, engine.ModifierTarget{K: "resistance"})
	if refusal := powerCommand(t, f, id, "usa/class.paladino.aura-sagrada", ""); refusal != "" {
		t.Fatalf("usar a Aura Sagrada foi recusado: %q", refusal)
	}
	if got := statOf(t, f, id, engine.ModifierTarget{K: "resistance"}) - before; got != 3 {
		t.Errorf("as resistências subiram %d com a aura, e o Carisma é 3 (p84)", got)
	}
}

// O VINGADOR SAGRADO dá voo 18m e RD 20 até o fim da cena (p84).
func TestTheHolyAvengerGivesFlightAndDamageReduction(t *testing.T) {
	f, id := paladinWithCharisma(t, 20)
	reduction := statOf(t, f, id, engine.ModifierTarget{K: "damageReduction"})
	flight := statOf(t, f, id, engine.ModifierTarget{K: "flySpeed"})
	if refusal := powerCommand(t, f, id, "usa/class.paladino.vingador-sagrado", ""); refusal != "" {
		t.Fatalf("usar o Vingador Sagrado foi recusado: %q", refusal)
	}
	if got := statOf(t, f, id, engine.ModifierTarget{K: "damageReduction"}) - reduction; got != 20 {
		t.Errorf("a RD subiu %d na forma de vingador, e o livro dá 20 (p84)", got)
	}
	if got := statOf(t, f, id, engine.ModifierTarget{K: "flySpeed"}) - flight; got != 18 {
		t.Errorf("o voo subiu %d na forma de vingador, e o livro dá 18m (p84)", got)
	}
}

// A ARMADURA DE ALLIHANNA não tem limite de usos: o livro diz só "gastar uma ação de movimento e 1 PM" (p21).
func TestAllihannasArmorCanBeUsedAgainInTheScene(t *testing.T) {
	f, id := barbaro(t, 5)
	seedRaca(t, f.s, id, "Dahllan")
	for use := 1; use <= 2; use++ {
		if refusal := powerCommand(t, f, id, "usa/race.dahllan.armadura-de-allihanna", ""); refusal != "" {
			t.Fatalf("o %dº uso na cena foi recusado: %q", use, refusal)
		}
	}
}

// O DEGRAU DA PALAVRAS AFIADAS diz o que compra: cada +1 PM é a linha seguinte da tabela do nobre, +2d6 (p80).
func TestTheSharpWordsStepSaysItsDice(t *testing.T) {
	spec := book.ActivationOf("class.nobre.palavras-afiadas-2d6", "")
	if spec == nil || spec.Scaling == nil {
		t.Fatal("a Palavras Afiadas não tem o degrau")
	}
	if !strings.Contains(spec.Scaling.StepLabel, "+2d6") {
		t.Errorf("o degrau da Palavras Afiadas diz %q; want o dado que ele compra, +2d6", spec.Scaling.StepLabel)
	}
}

// A COMPAIXÃO MUDA O QUE A CURA PELAS MÃOS DIZ: "você pode usar Cura pelas Mãos em alcance curto e, para cada PM
// que gastar, cura 2d6+1 (em vez de 1d8+1)" (p83). O app não rola a cura (decisão do dono): o que muda é o texto
// do gesto — o aviso do alcance e do dado na linha da Cura, e o que cada degrau compra no diálogo de usar.
//
// Pela LINHA da Cura, como o aviso do Tambor das Profundezas, e não pela página: a folha da própria Compaixão
// repete "2d6+1" no verbete, e a página inteira o conteria com ou sem a mudança.
func TestCompassionTakesTheHealingByHandsToShortRangeWithTwoD6PerPM(t *testing.T) {
	const warning = "alcance curto e 2d6+1 PV por PM (Virtude Paladinesca: Compaixão)"
	for _, c := range []struct {
		name, powers string
		compassion   bool
	}{
		{"com a Compaixão", `["class.paladino.virtude-paladinesca-compaixao"]`, true},
		{"sem a Compaixão", `[]`, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id := paladinWithCharisma(t, 6)
			seedPowerChoices(t, f.s, id, c.powers, "{}")
			f.actionSceneWithTheGoblinFirst(t, "Devota", id, 2)
			for where, screen := range map[string]string{
				"na aba Poderes": sheetTab(t, f, id, "abilities"), "nas Ações da mesa": acoesDaMesa(t, f),
			} {
				line := regexp.MustCompile(`<div class="ds-row-title">\s*Cura pelas Mãos\s*</div>\s*(?:<div class="ds-row-meta">([^<]*)<)?|` +
					`<span class="ds-row-title">Cura pelas Mãos</span> <span class="ds-row-meta">([^<]*)<`).FindStringSubmatch(screen)
				if line == nil {
					t.Fatalf("não há linha da Cura pelas Mãos %s", where)
				}
				if says := line[1] + line[2]; strings.Contains(says, warning) != c.compassion {
					t.Errorf("%s, a linha da Cura diz %q; o aviso da Compaixão tinha de estar lá: %v", where, says, c.compassion)
				}
			}
			step := "Cada +1 PM compra +1d8+1 PV curados."
			if c.compassion {
				step = "Cada +1 PM compra +2d6+1 PV curados."
			}
			if screen := sheetTab(t, f, id, "abilities"); !strings.Contains(screen, step) {
				t.Errorf("o diálogo de usar a Cura não diz %q", step)
			}
		})
	}
}
