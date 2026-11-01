package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/domain/sheet"
	"testing"
)

// p35: "não ganha as perícias treinadas ou proficiências da nova classe". Restaurar o padrão
// conta só a PRIMEIRA classe: o arcanista que virou guerreiro não veste armadura pesada.
func TestRestoringTheDefaultProficienciesCountsOnlyTheFirstClass(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Zaled", "Arcanista", 3, 10, 5)
	if refusal := enterClass(t, f, id, "Guerreiro"); refusal != "" {
		t.Fatalf("entrar na classe foi recusado: %q", refusal)
	}
	f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/proficiencias/padrao", id), "")

	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ficha: %v", err)
	}
	if !strings.Contains(row.Proficiencies, "armas-simples") {
		t.Fatalf("o padrão nem saiu: %s", row.Proficiencies)
	}
	for _, category := range []string{"armas-marciais", "armaduras-pesadas", "escudos"} {
		if strings.Contains(row.Proficiencies, category) {
			t.Errorf("o padrão deu %q, que é da segunda classe: %s", category, row.Proficiencies)
		}
	}
}

// A ficha da semente nasce como nasceria pelo gesto, e a p35 vale para ela também.
func TestTheSeededMulticlassGetsOnlyTheFirstClassProficiencies(t *testing.T) {
	f := newSceneFixture(t)
	id, err := f.s.Seeder().CreateCharacter(context.Background(), f.player, sheet.CreateBody{
		Name: "Zaled", Origin: "soldado",
		Classes: []sheet.ClassEntry{{ClassName: "Arcanista", Level: 3}, {ClassName: "Guerreiro", Level: 1}},
	})
	if err != nil {
		t.Fatalf("semear: %v", err)
	}
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ficha: %v", err)
	}
	if strings.Contains(row.Proficiencies, "armaduras-pesadas") {
		t.Errorf("a semente deu as proficiências da segunda classe: %s", row.Proficiencies)
	}
}
