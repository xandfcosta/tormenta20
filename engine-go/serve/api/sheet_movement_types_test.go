package api

import (
	"context"
	"testing"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// OS DESLOCAMENTOS POR TIPO das raças, em METROS como o voo (p23, p30):
//
//   - Espelunqueiro (goblin): "deslocamento de escalada igual ao seu deslocamento terrestre";
//   - Transformação Anfíbia (sereia/tritão): "deslocamento de natação 12m";
//   - Asas de Borboleta (sílfide): paira com deslocamento 9m, o que "permite que você ignore
//     terreno difícil" — sempre ligado (decisão do dono) —, e voa 12m gastando 1 PM por rodada.

func raceSheet(t *testing.T, f sceneFixture, race string) (int64, engine.ComputedSheet) {
	t.Helper()
	id := seedCharacterAtLevel(t, f.s, f.player, "Viajante", "Guerreiro", 1, 0, 0)
	if err := f.s.queries.CreateRace(context.Background(), sqlcgen.CreateRaceParams{Characterid: id, Race: race}); err != nil {
		t.Fatalf("pôr a raça %q: %v", race, err)
	}
	return id, computedOf(t, f, id)
}

func computedOf(t *testing.T, f sceneFixture, id int64) engine.ComputedSheet {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler a ficha %d: %v", id, err)
	}
	computed, err := sheet.LoadAndCompute(context.Background(), f.s.queries, f.s.catalogs, row)
	if err != nil {
		t.Fatalf("computar a ficha %d: %v", id, err)
	}
	return computed
}

func TestTheGoblinClimbsAsFastAsHeWalks(t *testing.T) {
	_, goblin := raceSheet(t, newSceneFixture(t), "goblin")
	walks := goblin.Displacement.Total * 3 / 2
	if walks == 0 || goblin.ClimbSpeed != walks {
		t.Errorf("o goblin anda %dm e escala %dm; o Espelunqueiro (p23) os iguala", walks, goblin.ClimbSpeed)
	}
}

func TestTheMermaidSwimsTwelveMetres(t *testing.T) {
	_, mermaid := raceSheet(t, newSceneFixture(t), "sereia-tritao")
	if mermaid.SwimSpeed != 12 {
		t.Errorf("a sereia nada %dm; a Transformação Anfíbia (p30) dá 12m", mermaid.SwimSpeed)
	}
}

func TestTheSylphHoversOverDifficultTerrainAndFliesForOnePM(t *testing.T) {
	f := newSceneFixture(t)
	id, sylph := raceSheet(t, f, "silfide")
	if !sylph.IgnoresDifficultTerrain {
		t.Error("a sílfide paira (p30) e ainda paga o terreno difícil")
	}
	if sylph.FlySpeed != 0 {
		t.Fatalf("a sílfide já voa %dm antes de gastar o PM — o caso não mede o gesto", sylph.FlySpeed)
	}
	before := poolsOf(t, f.s, id).MpCurrent
	if refusal := powerCommand(t, f, id, "usa/silfide-asas-de-borboleta", ""); refusal != "" {
		t.Fatalf("voar foi recusado: %q", refusal)
	}
	if got := computedOf(t, f, id).FlySpeed; got != 12 {
		t.Errorf("a sílfide gastou o PM e voa %dm; as Asas de Borboleta dão 12m", got)
	}
	if spent := before - poolsOf(t, f.s, id).MpCurrent; spent != 1 {
		t.Errorf("voar custou %d PM, e o livro diz 1 PM por rodada", spent)
	}
}
