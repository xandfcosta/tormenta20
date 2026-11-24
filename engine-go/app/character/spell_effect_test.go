package character

import (
	"os"
	"testing"

	"t20engine/domain/sheet"
	"t20engine/infra/db"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/db/testdb"
)

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }

// sheetWithoutEffects é uma ficha gravada num banco virgem, pronta para receber efeitos.
func sheetWithoutEffects(t *testing.T) (*sqlcgen.Queries, int64) {
	t.Helper()
	database, err := db.Open(testdb.Fresh(t))
	if err != nil {
		t.Fatalf("abrir o banco: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	q := sqlcgen.New(database)
	now := dbvalue.NowISO()
	owner, err := q.CreateUser(t.Context(), sqlcgen.CreateUserParams{
		Email: "efeito@t20.local", Passwordhash: "x", Createdat: now, Updatedat: now,
	})
	if err != nil {
		t.Fatalf("criar a conta: %v", err)
	}
	id, err := q.CreateCharacter(t.Context(), sqlcgen.CreateCharacterParams{
		OwnerId: owner.ID, Name: "Arcanista", Origin: "estudioso", Level: 9, Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassChoices: "{}", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("criar a ficha: %v", err)
	}
	return q, id
}

// effectsOf são as linhas em vigor da ficha como a ficha as lê.
func effectsOf(t *testing.T, q *sqlcgen.Queries, characterID int64) []sheet.EffectDTO {
	t.Helper()
	effects, err := sheet.LoadActiveEffects(t.Context(), q, characterID)
	if err != nil {
		t.Fatalf("ler os efeitos da ficha %d: %v", characterID, err)
	}
	return effects
}

const hasteBonus = `[{"target":{"k":"defense"},"amount":2,"bonusType":"untyped","note":"Velocidade"}]`

// O EFEITO DA MAGIA É UMA LINHA POR MAGIA NA FICHA (ver GLOSSARY, **efeito da magia**): o gesto mais recente diz a
// duração, os modificadores gravados ficam quando ele não traz nenhum, e a linha que o Conjurar tocou é a que a
// ficha sustenta. A ordem dos gestos não importa; o que importa é o que cada um pede.
func TestTheSpellEffectIsOneRowPerSpell(t *testing.T) {
	q, id := sheetWithoutEffects(t)
	write := func(spellID string, w spellEffectWrite) {
		t.Helper()
		if _, err := writeSpellEffect(t.Context(), q, id, spellID, w); err != nil {
			t.Fatalf("gravar %s %+v: %v", spellID, w, err)
		}
	}
	only := func(spellID string) sheet.EffectDTO {
		t.Helper()
		var found []sheet.EffectDTO
		for _, e := range effectsOf(t, q, id) {
			if e.CatalogID == spellID {
				found = append(found, e)
			}
		}
		if len(found) != 1 {
			t.Fatalf("a ficha tem %d linhas de %s: %+v; want uma", len(found), spellID, found)
		}
		return found[0]
	}

	// O Aplicar de cena, e depois o Conjurar sustentado.
	write("velocidade", spellEffectWrite{Scope: "scene", Modifiers: hasteBonus})
	write("velocidade", spellEffectWrite{Scope: "sustained", Cast: true, Augments: []sheet.AugmentPick{}})
	haste := only("velocidade")
	if haste.Scope != "sustained" || haste.Modifiers != hasteBonus {
		t.Errorf("depois do Conjurar, a Velocidade é %s com %s; want sustentada com o bônus do Aplicar", haste.Scope, haste.Modifiers)
	}
	if !haste.Sustaining() {
		t.Error("a Velocidade que esta ficha conjurou não é sustentada por ela, e a manutenção não a cobra")
	}

	// O Aplicar de novo, de cena: a duração é a dele, e a conjuração gravada fica.
	write("velocidade", spellEffectWrite{Scope: "scene", Modifiers: hasteBonus})
	if haste := only("velocidade"); haste.Scope != "scene" || haste.Sustaining() || !haste.CastHere {
		t.Errorf("depois do Aplicar de cena, a Velocidade é %s, sustentada aqui = %v, conjurada aqui = %v",
			haste.Scope, haste.Sustaining(), haste.CastHere)
	}

	// O Aplicar sozinho, sustentado: é o aliado, que recebe e não sustenta.
	write("oracao", spellEffectWrite{Scope: "sustained", Modifiers: hasteBonus})
	if prayer := only("oracao"); prayer.Sustaining() {
		t.Error("a Oração só aplicada nesta ficha é sustentada por ela; quem sustenta é quem conjurou")
	}
	only("velocidade")
}
