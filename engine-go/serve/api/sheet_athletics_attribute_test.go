package api

import (
	"context"
	"testing"

	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
)

// "PODE USAR DESTREZA em vez de Força em Atletismo": o Pequeno e Rechonchudo do hynne (p28) e o
// Acrobático (p129). O app usa o MAIOR dos dois (decisão do dono): ninguém escolheria o menor, e a
// conta diz qual atributo entrou e por quê.

func athleticsOf(t *testing.T, sheet engine.ComputedSheet) engine.ExpertiseBreakdown {
	t.Helper()
	for _, ex := range sheet.Expertises {
		if ex.Name == "Atletismo" {
			return ex
		}
	}
	t.Fatal("a ficha não tem Atletismo")
	return engine.ExpertiseBreakdown{}
}

func withStrengthAndDexterity(t *testing.T, f sceneFixture, id, strength, dexterity int64) {
	t.Helper()
	if _, err := f.s.db.ExecContext(context.Background(),
		"UPDATE characters SET strength = ?, dexterity = ? WHERE id = ?", strength, dexterity, id); err != nil {
		t.Fatalf("pôr os atributos: %v", err)
	}
}

func TestDexterityReplacesStrengthInAthleticsWhenItIsHigher(t *testing.T) {
	cases := []struct {
		name  string
		grant func(t *testing.T, f sceneFixture, id int64)
	}{
		{"Pequeno e Rechonchudo, hynne", func(t *testing.T, f sceneFixture, id int64) {
			if err := f.s.queries.CreateRace(context.Background(), sqlcgen.CreateRaceParams{Characterid: id, Race: "hynne"}); err != nil {
				t.Fatalf("pôr a raça: %v", err)
			}
		}},
		{"Acrobático", func(t *testing.T, f sceneFixture, id int64) {
			seedPowerChoices(t, f.s, id, `["acrobatico"]`, `{}`)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSceneFixture(t)
			id := seedCharacterAtLevel(t, f.s, f.player, "Saltador", "Guerreiro", 1, 0, 0)
			// A LINHA DA PERÍCIA, como a criação de ficha a grava: Atletismo é de Força (p114).
			if _, err := f.s.queries.CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
				Characterid: id, Name: "Atletismo", Attribute: "strength",
			}); err != nil {
				t.Fatalf("pôr o Atletismo: %v", err)
			}
			withStrengthAndDexterity(t, f, id, 1, 4)
			before := athleticsOf(t, computedOf(t, f, id))
			if before.Attribute != "strength" {
				t.Fatalf("sem o verbete o Atletismo já usa %q — o caso não mede a troca", before.Attribute)
			}
			c.grant(t, f, id)
			after := athleticsOf(t, computedOf(t, f, id))
			if after.Attribute != "dexterity" || after.Total-before.Total != after.AttrValue-before.AttrValue || after.AttrValue <= before.AttrValue {
				t.Errorf("com Força 1 e Destreza 4, o Atletismo usou %q (%d → %d); devia usar a Destreza", after.Attribute, before.Total, after.Total)
			}
			if after.AttributeSource == "" {
				t.Error("a conta não diz por que a Destreza entrou no Atletismo")
			}

			// O MAIOR: com a Força mais alta, ela continua.
			withStrengthAndDexterity(t, f, id, 5, 2)
			if strong := athleticsOf(t, computedOf(t, f, id)); strong.Attribute != "strength" {
				t.Errorf("com Força 5 e Destreza 2, o Atletismo usou %q; o maior é a Força", strong.Attribute)
			}
		})
	}
}
