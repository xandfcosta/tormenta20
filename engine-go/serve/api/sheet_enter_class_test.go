package api

import (
	"fmt"
	"strings"
	"testing"

	"t20engine/domain/sheet"
)

// ENTRAR NUMA CLASSE (p35) soma os PV dela, e quem estava caído pode levantar (p236), pela mesma regra da
// queda de nível (decisão do dono).
func TestEnteringAClassWhileDownGetsTheCharacterUp(t *testing.T) {
	f, id := arcanistaDeNivel(t, 5)
	// A ficha cai pelo caminho de verdade: com 3 PV, a Constituição baixa de 0 a −1 e tira 1 PV por nível
	// do arcanista 5 — ela fica com −2, e o 1º nível de Guerreiro (5 + Con, p37) a põe de pé com 2.
	constitution := func(value int) string { return fmt.Sprintf("/personagens/%d/atributos/constitution/%d", id, value) }
	sheetGesture(t, f, constitution(0))
	arrangePools(t, f.s, id, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.HpCurrent = 3
		return pools, nil
	})
	sheetGesture(t, f, constitution(-1))
	if got := activeConditionsOf(t, f, id); !isDown(got) {
		t.Fatalf("o controle falhou: a ficha devia estar caída, e as condições são %s", got)
	}
	down := poolsOf(t, f.s, id).HpCurrent

	sheetGesture(t, f, fmt.Sprintf("/personagens/%d/classes/Guerreiro/entra", id))

	if got := poolsOf(t, f.s, id).HpCurrent; got <= down || got < 1 {
		t.Fatalf("o Guerreiro devia somar PV à ficha caída com %d, e ela ficou com %d", down, got)
	}
	if got := activeConditionsOf(t, f, id); strings.Contains(got, "inconsciente") || strings.Contains(got, "sangrando") {
		t.Errorf("de pé, a ficha continua com as condições de quem caiu: %s", got)
	}
}

// A DEVOÇÃO SEGUE A CLASSE MAIS ESTRITA (p96): o devoto de Tenebra não entra em Paladino, que não serve a
// ela, e a recusa diz o deus, a classe e o que fazer. O mestre contorna editando a ficha (decisão do dono).
// Quem não tem deus entra, e a devoção fica pendente, como a de quem nasce paladino.
func TestEnteringAClassThatDoesNotServeTheGodIsRefused(t *testing.T) {
	cases := []struct {
		god     string
		refused bool
	}{
		{"tenebra", true},
		{"khalmyr", false},
		{"", false},
	}
	for _, c := range cases {
		t.Run("deus "+c.god, func(t *testing.T) {
			f, id := guerreiro(t)
			if c.god != "" {
				if refusal := powerCommand(t, f, id, devotionPath(c.god), ""); refusal != "" {
					t.Fatalf("escolher %s: %q", c.god, refusal)
				}
			}
			rec := f.requests(t, f.player, "POST", fmt.Sprintf("/personagens/%d/classes/Paladino/entra?tab=combat", id), "")
			refusal := sceneRefusal(rec.Body.String())
			entered := strings.Contains(storedClassesOf(t, f, id), "Paladino")
			switch {
			case c.refused && entered:
				t.Errorf("o devoto de %s entrou em Paladino", c.god)
			case c.refused && !(strings.Contains(refusal, "Tenebra") && strings.Contains(refusal, "Paladino") && strings.Contains(refusal, "devoção")):
				t.Errorf("a recusa não diz o deus, a classe e a devoção a trocar: %q", refusal)
			case !c.refused && !entered:
				t.Errorf("com deus %q, entrar em Paladino foi recusado: %q", c.god, refusal)
			}
		})
	}
}

// storedClassesOf são os níveis de classe gravados da ficha, um por linha, na ordem: "Guerreiro, Guerreiro, …".
func storedClassesOf(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	rows, err := f.s.db.Query(`SELECT ref FROM character_level_choices WHERE characterId = ? AND kind = 'class' ORDER BY id`, id)
	if err != nil {
		t.Fatalf("ler as classes: %v", err)
	}
	defer rows.Close()
	written := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		written = append(written, name)
	}
	return strings.Join(written, ", ")
}
