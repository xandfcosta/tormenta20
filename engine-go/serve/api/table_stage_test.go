package api

import (
	"strings"
	"t20engine/serve/web/table"
	"testing"
)

// gmRegionNames são os ids que o stream remenda na mesa do MESTRE. A lista é a
// mesma do `TableRegions`, escrita aqui de novo DE PROPÓSITO: derivá-la da
// produção faria o teste concordar com o defeito — uma região removida sumiria
// dos dois lados e o guarda ficaria verde sobre nada.
var gmRegionNames = []string{
	"table-header",
	"table-gm-menu",
	"table-attack",
	"table-skill-test",
	"table-tracker",
	"table-commands",
	"table-board",
	"table-populate",
	"table-archive",
	"table-npcs",
	"table-session-config",
}

// QUE REGIÕES existem no documento do mestre, e cada uma UMA VEZ SÓ.
//
// "Uma vez só" é a metade que pega o erro provável: mover uma região e esquecer
// de tirá-la de onde estava deixa DUAS raízes com o mesmo id. O `PatchElements`
// acerta a primeira, a segunda envelhece na tela, e as duas discordam sobre a
// mesma fila.
func TestTheGmTableHasEveryRegionExactlyOnce(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)

	page := f.requests(t, f.gm, "GET", f.tableUrl(), "").Body.String()

	for _, id := range gmRegionNames {
		mark := `id="` + id + `"`
		if n := strings.Count(page, mark); n != 1 {
			t.Errorf("a região %q aparece %d vezes na mesa do mestre, e o remendo precisa de exatamente 1", id, n)
		}
	}
	// E a lista escrita à mão é a do stream, nos dois sentidos: uma região que o
	// stream manda e a lista esqueceu é uma região que este guarda não mede.
	streamed := idsDasRegioes(t, f, f.gm)
	for _, id := range gmRegionNames {
		if !streamed[id] {
			t.Errorf("a mesa do mestre desenha %q e o stream não a remenda: ela nasce e nunca mais muda", id)
		}
		delete(streamed, id)
	}
	for id := range streamed {
		t.Errorf("o stream remenda %q e esta lista não a conhece: acrescente-a acima", id)
	}
}

// As DUAS pontas juntas, para os dois papéis, que é onde a divergência nasce: a
// lista de regiões do stream e a página são escritas em lugares diferentes e
// têm de concordar sobre quem existe. Discordar não dá erro em lugar nenhum — o
// remendo simplesmente não pousa.
func TestTheStreamOnlySendsTheRegionTheDocumentHas(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)

	for _, who := range []struct {
		name string
		id   int64
	}{{"mestre", f.gm}, {"jogador", f.player}} {
		page := f.requests(t, who.id, "GET", f.tableUrl(), "").Body.String()
		regions := idsDasRegioes(t, f, who.id)
		// O CONTROLE: sem região nenhuma, o laço abaixo é verde sobre nada.
		if len(regions) < 5 {
			t.Fatalf("o stream do %s tem só %d regiões", who.name, len(regions))
		}
		for id := range regions {
			if n := strings.Count(page, `id="`+id+`"`); n != 1 {
				t.Errorf("o stream manda %q ao %s e a página dele a tem %d vezes", id, who.name, n)
			}
		}
	}
	// E o que é do mestre não vai ao jogador: o elenco e as configurações da
	// sessão são dele.
	forPlayer := idsDasRegioes(t, f, f.player)
	for _, id := range []string{"table-npcs", "table-session-config", "table-gm-menu"} {
		if forPlayer[id] {
			t.Errorf("o stream manda ao jogador a região %q, do mestre", id)
		}
	}
}

// idsDasRegioes pergunta ao MESMO `table.TableRegions` da produção quais regiões o
// stream mandaria para aquele papel, montando a view pelo caminho de sempre.
func idsDasRegioes(t *testing.T, f sceneFixture, userID int64) map[string]bool {
	t.Helper()
	view, _, err := f.s.tableScene.LoadView(t.Context(), userID, f.campaignID, f.sessionID)
	if err != nil {
		t.Fatalf("montar a view: %v", err)
	}
	ids := map[string]bool{}
	for _, r := range table.TableRegions(view) {
		ids[r.ID] = true
	}
	return ids
}
