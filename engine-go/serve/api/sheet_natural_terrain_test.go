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
//     habilidade Explorador" — o interruptor de CADA terreno escolhido no Explorador, e nenhum
//     interruptor a mais: um "no terreno:" sem terreno foi o que a tela mostrou na primeira versão;
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

// situationRefusal liga o interruptor `key` pela rota da ficha e devolve a recusa, vazia se ligou.
func situationRefusal(t *testing.T, f sceneFixture, id int64, key string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/efeitos/situacao", id)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, fmt.Sprintf(`{"conditional":%q}`, key)).Body.String())
}

func TestTheTerrainPowersIgnoreDifficultTerrainOnlyWhereTheBookSays(t *testing.T) {
	cases := []struct {
		name    string
		class   string
		level   int64
		origin  string
		pick    string
		terrain string
		key     string
		orphan  string
	}{
		{name: "Caminho dos Ermos, druida 2, em terreno natural", class: "Druida", level: 2, key: "flag:terreno-natural"},
		{
			name: "Caminho do Explorador, caçador 5, na floresta escolhida no Explorador", class: "Caçador", level: 5,
			terrain: `{"class.cacador.explorador":["floresta"]}`, key: "flag:explorador:floresta", orphan: "flag:explorador",
		},
		{name: "À Prova de Tudo, origem Batedor, em terreno natural", class: "Guerreiro", level: 1, origin: "batedor", pick: `["origin-batedor-unique"]`, key: "flag:terreno-natural"},
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
			if c.terrain != "" {
				seedPowerChoices(t, f.s, id, `[]`, c.terrain)
			}
			if c.orphan != "" {
				if refusal := situationRefusal(t, f, id, c.orphan); refusal == "" {
					t.Errorf("a ficha oferece %q, um interruptor de terreno sem terreno", c.orphan)
				}
			}
			if ignoresDifficultTerrain(t, f, id) {
				t.Fatal("a ficha já ignora o terreno difícil antes do interruptor — o caso não mede a condição")
			}
			if refusal := situationRefusal(t, f, id, c.key); refusal != "" {
				t.Fatalf("ligar %q: %q", c.key, refusal)
			}
			if !ignoresDifficultTerrain(t, f, id) {
				t.Errorf("com %q ligado, o terreno difícil ainda cobra desta ficha", c.key)
			}
		})
	}
}

// O caçador que ainda não tem o Caminho do Explorador (5º nível, p51) liga a floresta do Explorador
// e ganha os bônus dela, mas o terreno difícil continua cobrando: o termo pede a flag do Caminho.
func TestTheExplorerTerrainSwitchIgnoresDifficultTerrainOnlyWithTheExplorerPath(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Batedora", "Caçador", 3, 0, 0)
	seedPowerChoices(t, f.s, id, `[]`, `{"class.cacador.explorador":["floresta"]}`)
	if refusal := situationRefusal(t, f, id, "flag:explorador:floresta"); refusal != "" {
		t.Fatalf("ligar a floresta: %q", refusal)
	}
	if ignoresDifficultTerrain(t, f, id) {
		t.Error("o caçador 3, sem o Caminho do Explorador, atravessa terreno difícil sem redução na floresta")
	}
}
