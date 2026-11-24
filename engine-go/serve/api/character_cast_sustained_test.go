package api

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O CONJURAR DA MAGIA SUSTENTADA abre o efeito em curso na ficha de quem conjurou, e é ele que a manutenção
// cobra e o limite de uma conta (p227). Antes só o Aplicar da aba Efeitos o abria, e só para as magias com
// bônus: a Tempestade Divina se conjurava, e ninguém pagava o PM do turno.

// sustainedEffectsOf são os efeitos sustentados da ficha, pelo id da magia, com os modificadores.
func sustainedEffectsOf(t *testing.T, s *Server, characterID int64) map[string]string {
	t.Helper()
	rows, err := s.queries.ListActiveEffectsByCharacter(context.Background(), characterID)
	if err != nil {
		t.Fatalf("listar efeitos da ficha %d: %v", characterID, err)
	}
	out := map[string]string{}
	for _, r := range rows {
		if r.Scope == "sustained" {
			out[r.Catalogid] = r.Modifiers
		}
	}
	return out
}

func learnPrepared(t *testing.T, s *Server, characterID int64, spellID string) {
	t.Helper()
	if _, err := s.queries.CreateSpell(context.Background(), sqlcgen.CreateSpellParams{
		Characterid: characterID, Catalogspellid: spellID, Prepared: 1, Learnedat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("aprender %s: %v", spellID, err)
	}
}

func TestCastingASustainedSpellOpensItsUpkeep(t *testing.T) {
	s := newCastServer(t)
	owner := seedUser(t, s, "clerigo@t20.local")
	cleric := seedCaster(t, s, owner, "Clérigo", 8, 30, "tempestade-divina")
	learnPrepared(t, s, cleric, "silencio")
	learnPrepared(t, s, cleric, "curar-ferimentos")

	if err := castSpell(t, s, owner, cleric, "curar-ferimentos", `{}`); err != nil {
		t.Fatalf("conjurar a Curar Ferimentos: %v", err)
	}
	if got := sustainedEffectsOf(t, s, cleric); len(got) != 0 {
		t.Fatalf("a magia instantânea abriu efeito sustentado: %v", got)
	}

	if err := castSpell(t, s, owner, cleric, "tempestade-divina", `{}`); err != nil {
		t.Fatalf("conjurar a Tempestade Divina: %v", err)
	}
	if _, open := sustainedEffectsOf(t, s, cleric)["tempestade-divina"]; !open {
		t.Fatal("conjurar a Tempestade Divina não abriu o efeito sustentado, e a manutenção não tem o que cobrar")
	}
	upkeep, err := sheetVitals{q: s.queries}.SustainedOf(context.Background(), cleric)
	if err != nil || len(upkeep) != 1 || upkeep[0].Label != "Tempestade Divina" {
		t.Fatalf("a manutenção do turno lê %v (%v); want a Tempestade Divina", upkeep, err)
	}

	t.Run("a segunda sustentada é recusada e não cobra", func(t *testing.T) {
		before := mpOf(t, s, cleric)
		err := castSpell(t, s, owner, cleric, "silencio", `{}`)
		if err == nil || !strings.Contains(err.Error(), "Tempestade Divina já está sustentada") {
			t.Fatalf("conjurar o Silêncio com a Tempestade de pé devolveu %v", err)
		}
		if after := mpOf(t, s, cleric); after != before {
			t.Errorf("a recusa cobrou: PM %d → %d", before, after)
		}
	})

	t.Run("o aprimoramento que tira a magia da sustentada passa, e não abre efeito", func(t *testing.T) {
		// O aprimoramento 1 do Silêncio muda a duração para cena.
		if err := castSpell(t, s, owner, cleric, "silencio", `{"augments":[{"augmentIndex":1,"stacks":1}]}`); err != nil {
			t.Fatalf("o Silêncio de cena foi recusado: %v", err)
		}
		if _, open := sustainedEffectsOf(t, s, cleric)["silencio"]; open {
			t.Error("o Silêncio de cena abriu efeito sustentado")
		}
	})

	t.Run("reconjurar a mesma não é uma segunda", func(t *testing.T) {
		if err := castSpell(t, s, owner, cleric, "tempestade-divina", `{}`); err != nil {
			t.Fatalf("reconjurar a Tempestade Divina: %v", err)
		}
		if got := sustainedEffectsOf(t, s, cleric); len(got) != 1 {
			t.Errorf("a ficha ficou com %d sustentadas: %v", len(got), got)
		}
	})
}

// O APLICAR E O CONJURAR ABREM A MESMA LINHA: a Oração conjurada abre a manutenção sem modificadores, e o
// Aplicar grava os modificadores nela. Na ordem inversa, o Conjurar não apaga o que o Aplicar gravou.
func TestTheBuffAndTheCastShareTheSustainedEffect(t *testing.T) {
	s := newCastServer(t)
	owner := seedUser(t, s, "oracao@t20.local")
	for _, castFirst := range []bool{true, false} {
		cleric := seedCaster(t, s, owner, "Clérigo", 8, 30, "oracao")
		gestures := []func() error{
			func() error { return castSpell(t, s, owner, cleric, "oracao", `{}`) },
			func() error {
				_, err := s.characterPlays().ApplySpellBuff(context.Background(), cleric, "oracao", nil)
				return err
			},
		}
		if !castFirst {
			gestures[0], gestures[1] = gestures[1], gestures[0]
		}
		for _, gesture := range gestures {
			if err := gesture(); err != nil {
				t.Fatalf("conjurar primeiro = %v: %v", castFirst, err)
			}
		}
		effects := sustainedEffectsOf(t, s, cleric)
		if len(effects) != 1 {
			t.Fatalf("conjurar primeiro = %v: %d sustentadas na ficha, want a Oração só", castFirst, len(effects))
		}
		if mods := effects["oracao"]; !strings.Contains(mods, "target") {
			t.Errorf("conjurar primeiro = %v: a Oração ficou sem os modificadores: %s", castFirst, mods)
		}
	}
}

// O APRIMORAMENTO QUE TORNA A MAGIA SUSTENTADA abre a manutenção: a esfera da Invisibilidade (p195).
func TestTheAugmentThatSustainsTheSpellOpensItsUpkeep(t *testing.T) {
	s := newCastServer(t)
	owner := seedUser(t, s, "invisivel@t20.local")
	arcanist := seedCaster(t, s, owner, "Arcanista", 9, 40, "invisibilidade")

	if err := castSpell(t, s, owner, arcanist, "invisibilidade", `{}`); err != nil {
		t.Fatalf("conjurar a Invisibilidade: %v", err)
	}
	if got := sustainedEffectsOf(t, s, arcanist); len(got) != 0 {
		t.Fatalf("a Invisibilidade definida abriu efeito sustentado: %v", got)
	}
	if err := castSpell(t, s, owner, arcanist, "invisibilidade", `{"augments":[{"augmentIndex":2,"stacks":1}]}`); err != nil {
		t.Fatalf("conjurar a esfera da Invisibilidade: %v", err)
	}
	if _, open := sustainedEffectsOf(t, s, arcanist)["invisibilidade"]; !open {
		t.Error("a esfera da Invisibilidade, sustentada, não abriu o efeito")
	}
}

// A SEGUNDA SUSTENTADA TRANCA O CONJURAR NAS DUAS ABAS, com a frase do servidor, e a linha do grimório a marca.
// A magia que não é sustentada fica de pé.
func TestTheSecondSustainedSpellLocksTheCastOnBothTabs(t *testing.T) {
	f := newSceneFixture(t)
	// A Tempestade Divina que ESTA ficha conjurou: a marca da conjuração é o que a faz sustentada por ela.
	storm, err := f.s.queries.UpsertActiveEffect(context.Background(), sqlcgen.UpsertActiveEffectParams{
		Characterid: f.charID, Source: "spell", Catalogid: "tempestade-divina", Scope: "sustained",
		Modifiers: "[]", Createdat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("pôr a Tempestade Divina de pé: %v", err)
	}
	if err := f.s.queries.SaveEffectCastAugments(context.Background(), sqlcgen.SaveEffectCastAugmentsParams{
		Effectid: storm.ID, Augments: "[]",
	}); err != nil {
		t.Fatalf("marcar a Tempestade Divina como conjurada aqui: %v", err)
	}
	for _, spell := range []string{"velocidade", "explosao-de-chamas"} {
		learn(t, f, f.charID, spell)
	}
	const clash = "Tempestade Divina já está sustentada"

	actions := entre(t, acoesDaMesa(t, f), "Ação padrão", "Ação de movimento")
	if line := castLineOf(t, actions, "Velocidade"); !strings.Contains(line, clash) {
		t.Errorf("na aba Ações, o Conjurar · Velocidade não diz %q:\n%s", clash, recorte(line))
	}
	if line := castLineOf(t, actions, "Explosão de Chamas"); strings.Contains(line, clash) {
		t.Errorf("na aba Ações, a Explosão de Chamas, que não é sustentada, tranca:\n%s", recorte(line))
	}

	grimoire := spellScreen(t, f, f.charID)
	for _, c := range []struct {
		name   string
		locked bool
	}{{"Velocidade", true}, {"Explosão de Chamas", false}} {
		button := regexp.MustCompile(`<button[^>]*aria-label="Conjurar ` + c.name + `"[^>]*>`).FindString(grimoire)
		if button == "" {
			t.Fatalf("a aba Magias não desenhou o Conjurar %s", c.name)
		}
		if got := strings.Contains(button, "disabled"); got != c.locked {
			t.Errorf("na aba Magias, o Conjurar %s trancado = %v; want %v", c.name, got, c.locked)
		}
	}
	if !strings.Contains(grimoire, "outra sustentada de pé") {
		t.Error("a linha da Velocidade no grimório não marca a sustentada que a tranca")
	}
}
