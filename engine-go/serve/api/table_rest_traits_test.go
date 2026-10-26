package api

import (
	"net/http"
	"testing"
)

// O DESCANSO DO GRUPO OBEDECE A QUEM DORME (p106, ALE-488).
//
// O gesto é o do mestre, "descanso de dia", e a conta é a da ficha de cada um:
// o goblin nunca recupera menos que o nível (Rato das Ruas, p23), o golem
// descansa sempre em condição normal (Criatura Artificial, p27) e o cavaleiro
// com Pajem dorme uma categoria acima (p54). O guerreiro humano é o controle:
// sem ele, uma noite que desse o nível a todo mundo passaria verde.
//
// Todos de 5º nível e feridos, e a medida é QUANTO cada noite devolveu — o
// ponto de partida varia com a raça (o golem tem outro PV máximo). Os números
// estão escritos à mão: ruim é 2 (metade de 5, para baixo), normal 5,
// confortável 10, luxuosa 15.
func TestTheGroupRestFollowsEachSleepersAbilities(t *testing.T) {
	f := newSceneFixture(t)
	wounded := func(name, class, race string) int64 {
		id := seedCharacterAtLevel(t, f.s, f.player, name, class, 5, bookPools(t, f.s, class, 5).PvMax-1, 0)
		if race != "" {
			seedRaca(t, f.s, id, race)
		}
		seedMember(t, f.s, f.campaignID, id)
		return id
	}
	control := wounded("Controle", "Guerreiro", "")
	goblin := wounded("Rato", "Guerreiro", "goblin")
	golem := wounded("Golem", "Guerreiro", "golem")
	knight := wounded("Cavaleira", "Cavaleiro", "")
	if refusal := powerCommand(t, f, knight, "escolhe/class.cavaleiro.pajem", ""); refusal != "" {
		t.Fatalf("escolher o Pajem: %q", refusal)
	}

	ids := []int64{control, goblin, golem, knight}
	rests := func(quality string) map[int64]int64 {
		t.Helper()
		before := map[int64]int64{}
		for _, id := range ids {
			before[id] = poolsOf(t, f.s, id).HpCurrent
		}
		if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/descanso/dia",
			`{"rest_quality":"`+quality+`"}`); rec.Code != http.StatusOK {
			t.Fatalf("descanso %s deu %d: %s", quality, rec.Code, rec.Body.String())
		}
		gained := map[int64]int64{}
		for _, id := range ids {
			gained[id] = poolsOf(t, f.s, id).HpCurrent - before[id]
		}
		return gained
	}

	gained := rests("ruim")
	for _, tt := range []struct {
		who  string
		id   int64
		want int64
	}{
		{"o controle, ao relento, recupera metade", control, 2},
		{"o goblin nunca recupera menos que o nível", goblin, 5},
		{"o golem descansa em condição normal", golem, 5},
		{"a cavaleira com Pajem dorme como na estalagem", knight, 5},
	} {
		if got := gained[tt.id]; got != tt.want {
			t.Errorf("noite ruim — %s: +%d PV, want +%d", tt.who, got, tt.want)
		}
	}

	gained = rests("confortavel")
	for _, tt := range []struct {
		who  string
		id   int64
		want int64
	}{
		{"o controle recupera o dobro", control, 10},
		{"o golem não melhora com o conforto", golem, 5},
		{"a cavaleira com Pajem dorme no luxo", knight, 15},
	} {
		if got := gained[tt.id]; got != tt.want {
			t.Errorf("noite confortável — %s: +%d PV, want +%d", tt.who, got, tt.want)
		}
	}
}
