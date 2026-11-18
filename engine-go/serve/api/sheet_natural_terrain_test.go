package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"t20engine/domain/sheet"
)

// O TERRENO DIFÍCIL QUE NÃO REDUZ, sob a condição do livro:
//
//   - Caminho do Explorador (caçador, p51): "só funciona em terrenos nos quais você tenha a
//     habilidade Explorador" — o interruptor "no terreno" que o Explorador já tem;
//   - Caminho dos Ermos (druida, p63): "só funciona em terrenos naturais";
//   - À Prova de Tudo (origem Batedor, p88): "por terreno difícil natural".
//
// Quem sabe onde a cena está é a mesa, e por isso é interruptor de situação (decisão do dono,
// como o terreno do Explorador). O número medido é o que o tabuleiro lê: a ficha computada diz
// se o terreno difícil cobra desta criatura.

func ignoresDifficultTerrain(t *testing.T, f sceneFixture, id int64) bool {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler a ficha %d: %v", id, err)
	}
	computed, err := sheet.LoadAndCompute(context.Background(), f.s.queries, f.s.catalogs, row)
	if err != nil {
		t.Fatalf("computar a ficha %d: %v", id, err)
	}
	return computed.IgnoresDifficultTerrain
}

func turnsTheSituationOn(t *testing.T, f sceneFixture, id int64, key string) {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/efeitos/situacao", id)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, fmt.Sprintf(`{"conditional":%q}`, key)).Body.String()); refusal != "" {
		t.Fatalf("ligar %q: %q", key, refusal)
	}
}

func TestTheTerrainPowersIgnoreDifficultTerrainOnlyWhereTheBookSays(t *testing.T) {
	cases := []struct {
		name   string
		class  string
		level  int64
		origin string
		pick   string
		key    string
	}{
		{"Caminho dos Ermos, druida 2, em terreno natural", "Druida", 2, "", "", "flag:terreno-natural"},
		{"Caminho do Explorador, caçador 5, no terreno de Explorador", "Caçador", 5, "", "", "flag:explorador"},
		{"À Prova de Tudo, origem Batedor, em terreno natural", "Guerreiro", 1, "batedor", `["origin-batedor-unique"]`, "flag:terreno-natural"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSceneFixture(t)
			id := seedCharacterAtLevel(t, f.s, f.player, "Andarilho", c.class, c.level, 0, 0)
			if c.origin != "" {
				if _, err := f.s.db.ExecContext(context.Background(), "UPDATE characters SET origin = ? WHERE id = ?", c.origin, id); err != nil {
					t.Fatalf("pôr a origem: %v", err)
				}
				choiceCom(t, f, id, `[]`, c.pick)
			}
			if ignoresDifficultTerrain(t, f, id) {
				t.Fatal("a ficha já ignora o terreno difícil antes do interruptor — o caso não mede a condição")
			}
			turnsTheSituationOn(t, f, id, c.key)
			if !ignoresDifficultTerrain(t, f, id) {
				t.Errorf("com %q ligado, o terreno difícil ainda cobra desta ficha", c.key)
			}
		})
	}
}
