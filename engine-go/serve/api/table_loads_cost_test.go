package api

import (
	"net/http"
	"testing"
)

// A PRÉVIA DO ARRASTO NÃO CARREGA FICHA NENHUMA. Ela roda a cada casa que o dedo
// cruza e só pergunta de quem é a peça, e montava o Grupo inteiro, com a ficha de
// cada membro, para ler a posse: com um grupo de seis, 10 ms por casa, contra 1 ms
// sem as cargas.
//
// O teto é de ALOCAÇÕES, pela razão do `TestTheExpertisesPageDoesNotRebuildThePowersPerSkill`:
// tempo varia com a máquina, e a contagem não. Ele fica longe dos dois lados — a
// prévia sem cargas aloca ~5 mil vezes, e com as seis fichas ~15,6 mil —, porque
// cada fatia que mexe na ficha soma algumas centenas.
//
// A página da mesa, que também passou a carregar cada ficha uma vez em vez de duas,
// não tem teto aqui: a carga a menos por membro custa ~900 alocações, perto demais
// do que a ficha cresce de uma fatia para outra.
func TestTheDragPreviewLoadsNoSheet(t *testing.T) {
	f := newSceneFixture(t)
	for _, name := range []string{"Ana", "Bento", "Caio", "Dora", "Enzo"} {
		seedMember(t, f.s, f.campaignID, seedCharacterAtLevel(t, f.s, f.player, name, "Guerreiro", 5, 0, 0))
	}
	tokenID := f.onBoardAt(t, 4, 2)
	f.turnPlayer(t)
	path, body := f.tableUrl()+"/tabuleiro/"+tokenID+"/previa", `{"from":{"X":9,"Y":2}}`
	if rec := f.requests(t, f.player, http.MethodPost, path, body); rec.Code != http.StatusOK {
		t.Fatalf("a prévia deu %d", rec.Code)
	}

	const ceiling = 8000
	if allocs := testing.AllocsPerRun(5, func() { f.requests(t, f.player, http.MethodPost, path, body) }); allocs > ceiling {
		t.Errorf("a prévia do arrasto alocou %.0f vezes com um grupo de seis, e o teto é %d: "+
			"ela voltou a carregar a ficha dos membros para saber de quem é a peça", allocs, ceiling)
	}
}
