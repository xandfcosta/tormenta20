package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"t20engine/domain/book"
	"t20engine/infra/db/sqlcgen"
	"testing"
)

// levelUpDialog é o diálogo que o `+` abre. Ele só existe na página quando há o que escolher.
const levelUpDialog = "subir-de-nivel"

func enterClass(t *testing.T, f sceneFixture, id int64, class string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/classes/%s/entra", id, url.PathEscape(class))
	return bagRefusal(t, f.requests(t, f.player, http.MethodPost, target, "").Body.String())
}

func stepClass(t *testing.T, f sceneFixture, id int64, class string, step int) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/nivel/%s/%d", id, url.PathEscape(class), step)
	return bagRefusal(t, f.requests(t, f.player, http.MethodPost, target, "").Body.String())
}

func classesOfTheSheet(t *testing.T, f sceneFixture, id int64) map[string]int64 {
	t.Helper()
	rows, err := f.s.queries.ListClassesByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar as classes: %v", err)
	}
	out := map[string]int64{}
	for _, row := range rows {
		out[row.Classname] = row.Level
	}
	return out
}

// p35: entrar numa classe nova é um nível a mais no personagem, com PV e PM dela, e sem as
// perícias nem as proficiências dela.
func TestEnteringAClassAddsALevelButNoSkillsNorProficiencies(t *testing.T) {
	f, id := guerreiro(t)
	before := poolsOf(t, f.s, id)
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ficha: %v", err)
	}
	skills, err := f.s.queries.ListExpertisesByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("perícias: %v", err)
	}

	if refusal := enterClass(t, f, id, "Arcanista"); refusal != "" {
		t.Fatalf("entrar na classe foi recusado: %q", refusal)
	}

	classes := classesOfTheSheet(t, f, id)
	if classes["Arcanista"] != 1 {
		t.Errorf("a classe nova entrou com o nível %d, esperava 1", classes["Arcanista"])
	}
	after, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ficha: %v", err)
	}
	if after.Level != row.Level+1 {
		t.Errorf("o nível do personagem foi de %d para %d, esperava mais um", row.Level, after.Level)
	}
	if pools := poolsOf(t, f.s, id); pools.HpMax <= before.HpMax || pools.MpMax <= before.MpMax {
		t.Errorf("os poços não subiram: PV %d→%d, PM %d→%d", before.HpMax, pools.HpMax, before.MpMax, pools.MpMax)
	}
	if after.Proficiencies != row.Proficiencies {
		t.Errorf("a classe nova mexeu nas proficiências: %s → %s", row.Proficiencies, after.Proficiencies)
	}
	skillsAfter, err := f.s.queries.ListExpertisesByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("perícias: %v", err)
	}
	if len(skillsAfter) != len(skills) {
		t.Errorf("a classe nova mexeu nas perícias: %d → %d", len(skills), len(skillsAfter))
	}
}

func TestEnteringAClassRefusesWhatTheBookDoesNotHave(t *testing.T) {
	f, id := guerreiro(t)
	for _, c := range []struct{ class, wants string }{
		{"Guerreiro", "já é"},
		{"Astronauta", "Astronauta"},
	} {
		if refusal := enterClass(t, f, id, c.class); !strings.Contains(refusal, c.wants) {
			t.Errorf("entrar em %s: a recusa foi %q, esperava citar %q", c.class, refusal, c.wants)
		}
	}
}

// O teto é do personagem (p35): 20 no total, e a classe nova contaria o 21º.
func TestEnteringAClassStopsAtLevelTwenty(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lenda", "Guerreiro", 20, 10, 5)
	if refusal := enterClass(t, f, id, "Arcanista"); !strings.Contains(refusal, "21") {
		t.Errorf("a recusa do 21º nível foi %q", refusal)
	}
	if _, found := classesOfTheSheet(t, f, id)["Arcanista"]; found {
		t.Error("a classe entrou mesmo recusada")
	}
}

// O `−` no 1º nível apaga a classe, e o personagem desce junto.
func TestLoweringASecondClassAtLevelOneRemovesIt(t *testing.T) {
	f, id := guerreiro(t)
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ficha: %v", err)
	}
	if refusal := enterClass(t, f, id, "Ladino"); refusal != "" {
		t.Fatalf("entrar: %q", refusal)
	}
	if refusal := stepClass(t, f, id, "Ladino", -1); refusal != "" {
		t.Fatalf("sair da classe foi recusado: %q", refusal)
	}
	if _, found := classesOfTheSheet(t, f, id)["Ladino"]; found {
		t.Error("a classe continuou na ficha")
	}
	after, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ficha: %v", err)
	}
	if after.Level != row.Level {
		t.Errorf("o nível ficou em %d, esperava voltar a %d", after.Level, row.Level)
	}
}

// O `+` sempre abre a lista, com a classe nova à vista, e o teto o desliga.
func TestTheLevelStepOffersTheOtherClasses(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Novato", "Guerreiro", 1, 10, 5)
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "").Body.String()
	if !strings.Contains(page, levelUpDialog) {
		t.Error("o + não abre a lista para quem tem uma classe só")
	}
	if !strings.Contains(page, "Outra classe") || !strings.Contains(page, "/classes/Ladino/entra") {
		t.Error("a lista não oferece entrar em outra classe")
	}
	if strings.Contains(page, "/classes/Guerreiro/entra") {
		t.Error("a lista oferece entrar na classe que ele já tem")
	}

	legend := seedCharacterAtLevel(t, f.s, f.player, "Lenda", "Guerreiro", 20, 10, 5)
	full := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", legend), "").Body.String()
	if strings.Contains(full, levelUpDialog) {
		t.Error("no nível 20 o + ainda abre a lista")
	}
}

// A classe que tem algo escolhido não sai, e a recusa diz o quê.
func TestAClassWithSomethingChosenCannotLeave(t *testing.T) {
	t.Run("poder da classe", func(t *testing.T) {
		f, id := guerreiro(t)
		if refusal := enterClass(t, f, id, "Ladino"); refusal != "" {
			t.Fatalf("entrar: %q", refusal)
		}
		var picked book.ClassPower
		for _, power := range book.ClassPowers() {
			if power.ClassName == "Ladino" && power.GrantedAtLevel == nil {
				picked = power
				break
			}
		}
		if picked.Name == "" {
			t.Fatal("o catálogo não tem poder eletivo de Ladino: o caso não mede nada")
		}
		seedPowerChoices(t, f.s, id, fmt.Sprintf("[%q]", picked.ID), "{}")
		if refusal := stepClass(t, f, id, "Ladino", -1); !strings.Contains(refusal, picked.Name) {
			t.Errorf("a recusa não citou o poder %q: %q", picked.Name, refusal)
		}
		if classesOfTheSheet(t, f, id)["Ladino"] != 1 {
			t.Error("a classe saiu mesmo recusada")
		}
	})

	t.Run("magia que só ela concede", func(t *testing.T) {
		f, id := guerreiro(t)
		if refusal := enterClass(t, f, id, "Arcanista"); refusal != "" {
			t.Fatalf("entrar: %q", refusal)
		}
		learn(t, f, id, "luz")
		if refusal := stepClass(t, f, id, "Arcanista", -1); !strings.Contains(refusal, "Luz") {
			t.Errorf("a recusa não citou a magia: %q", refusal)
		}
	})

	// A ESCOLHA DA CLASSE: a recusa diz o rótulo que a pessoa escolheu, e não o valor gravado.
	for _, c := range []struct{ class, blob, wants string }{
		{"Clérigo", `{"Clérigo":{"devoto":"khalmyr"}}`, "Khalmyr"},
		{"Arcanista", `{"Arcanista":{"caminho":"bruxo"}}`, "Bruxo"},
	} {
		t.Run("escolha de "+c.class, func(t *testing.T) {
			f, id := guerreiro(t)
			if refusal := enterClass(t, f, id, c.class); refusal != "" {
				t.Fatalf("entrar: %q", refusal)
			}
			if _, err := f.s.db.ExecContext(context.Background(),
				"UPDATE characters SET classChoices = ? WHERE id = ?", c.blob, id,
			); err != nil {
				t.Fatalf("gravar a escolha: %v", err)
			}
			if refusal := stepClass(t, f, id, c.class, -1); !strings.Contains(refusal, c.wants) {
				t.Errorf("a recusa não citou %q: %q", c.wants, refusal)
			}
			if classesOfTheSheet(t, f, id)[c.class] != 1 {
				t.Error("a classe saiu mesmo recusada")
			}
		})
	}

	// O CONTROLE: a magia que outra classe da ficha também concede não segura ninguém.
	t.Run("magia que a outra classe também concede", func(t *testing.T) {
		f := newSceneFixture(t)
		id := seedCharacterAtLevel(t, f.s, f.player, "Zaled", "Arcanista", 3, 10, 5)
		if refusal := enterClass(t, f, id, "Clérigo"); refusal != "" {
			t.Fatalf("entrar: %q", refusal)
		}
		learn(t, f, id, "luz")
		if refusal := stepClass(t, f, id, "Clérigo", -1); refusal != "" {
			t.Errorf("a magia da outra classe segurou a saída: %q", refusal)
		}
	})
}

func learn(t *testing.T, f sceneFixture, id int64, spell string) {
	t.Helper()
	if _, err := f.s.queries.CreateSpell(context.Background(), sqlcgen.CreateSpellParams{
		Characterid: id, Catalogspellid: spell,
	}); err != nil {
		t.Fatalf("aprender %s: %v", spell, err)
	}
}
