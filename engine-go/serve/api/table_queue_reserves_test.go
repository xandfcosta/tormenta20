package api

import (
	"html"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// A FILA DO JOGADOR (ALE-662, fatia 5).
//
// A redação da fila não sabia quem olha: a linha do próprio jogador dizia "PM ocultos
// pelo mestre", e o Grupo, logo abaixo, mostrava o PM dele (a ficha é dele; decisão do
// dono).

var nextRowStart = regexp.MustCompile(`class="ds-row[ "]`)

// queueRow recorta a linha de um combatente da fila.
func queueRow(t *testing.T, page, label string) string {
	t.Helper()
	_, queue, ok := strings.Cut(page, `id="table-tracker"`)
	if !ok {
		t.Fatal("a mesa não desenhou a fila")
	}
	at := strings.Index(queue, label)
	if at < 0 {
		t.Fatalf("a fila não tem a linha de %s", label)
	}
	row := queue[at:]
	// A PRÓXIMA LINHA começa num `ds-row` inteiro: o `ds-row-meta` de dentro da linha,
	// onde mora a frase de oculto, não é fim de nada.
	if end := nextRowStart.FindStringIndex(row); end != nil {
		row = row[:end[0]]
	}
	return row
}

func TestTheQueueShowsThePlayerHisOwnPools(t *testing.T) {
	f := newSceneFixture(t)
	withAColleague(t, f)
	f.scene(t)
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/por-no-mapa", ""); rec.Code != http.StatusOK {
		t.Fatalf("pôr o grupo na fila deu %d", rec.Code)
	}
	page := html.UnescapeString(f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String())

	// O CONTROLE: o PM da colega continua escondido.
	if colleague := queueRow(t, page, "Guerreira"); !strings.Contains(colleague, "PM ocultos pelo mestre") {
		t.Fatalf("o PM da colega não está escondido na fila — o caso mediria outra coisa:\n%s", colleague)
	}
	own := queueRow(t, page, "Arcanista")
	if strings.Contains(own, "PM ocultos pelo mestre") || !regexp.MustCompile(`PM \d+/\d+`).MatchString(own) {
		t.Errorf("a fila esconde do jogador o PM do próprio personagem:\n%s", own)
	}
}
