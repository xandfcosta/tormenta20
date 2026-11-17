package api

import (
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// A FILA DO JOGADOR E AS RESERVAS (ALE-662, fatia 5).
//
// A redação da fila não sabia quem olha: a linha do próprio jogador dizia "PM ocultos
// pelo mestre", e o Grupo, logo abaixo, mostrava o PM dele (a ficha é dele; decisão do
// dono). E a reserva de pontos temporários não aparecia na fileira do jogador nem no
// editor da folha do combatente, onde o mestre aplica o dano que come a reserva primeiro
// (p106).

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

// gmQueueRow recorta a fileira de um combatente na fila do mestre, sem a folha dele.
func gmQueueRow(t *testing.T, page, label string) string {
	t.Helper()
	for _, row := range strings.Split(page, `class="ds-row ds-gm-row"`)[1:] {
		row, _, _ = strings.Cut(row, "<dialog")
		if strings.Contains(row, label) {
			return row
		}
	}
	t.Fatalf("a fila do mestre não tem a fileira de %s", label)
	return ""
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

func TestTheReserveShowsOnTheQueueAndOnTheCombatantSheet(t *testing.T) {
	f := newSceneFixture(t)
	if rec := effect(t, f, f.charID, "aplica/campo-de-forca"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Campo de Força devolveu %d", rec.Code)
	}
	if rec := f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/vitais/pm/temporarios/5", f.charID), ""); rec.Code != http.StatusOK {
		t.Fatalf("dar PM temporário deu %d", rec.Code)
	}
	f.scene(t)
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/por-no-mapa", ""); rec.Code != http.StatusOK {
		t.Fatalf("pôr o grupo na fila deu %d", rec.Code)
	}
	entry := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Arcanista")

	// A FILEIRA DO JOGADOR diz a reserva, como a do mestre.
	if own := queueRow(t, html.UnescapeString(f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()), "Arcanista"); !strings.Contains(own, "+30") {
		t.Errorf("a fileira do jogador não diz os 30 PV temporários:\n%s", own)
	}

	gm := html.UnescapeString(f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String())
	// A FILEIRA do mestre, e não a folha do combatente, que tem o mesmo "+N" no editor e
	// vem ANTES da fileira no documento.
	gmRow := gmQueueRow(t, gm, "Arcanista")
	// O CONTROLE: a fileira do mestre já dizia o +30 do PV.
	if !strings.Contains(gmRow, `title="PV temporários">+30<`) {
		t.Fatalf("a fileira do mestre não diz o +30 do PV — o caso mediria outra coisa:\n%s", gmRow)
	}
	// A DICA diz o pool certo na linha do PM.
	if !strings.Contains(gmRow, `title="PM temporários">+5<`) {
		t.Errorf("o +5 da linha do PM do mestre não diz que são PM temporários:\n%s", gmRow)
	}
	// A FOLHA DO COMBATENTE, onde o mestre aplica o dano, diz a reserva.
	sheet := regexp.MustCompile(`(?s)<dialog[^>]*id="combatant-` + entry + `".*?</dialog>`).FindString(gm)
	if sheet == "" {
		t.Fatal("o mestre não tem a folha do Arcanista")
	}
	if !strings.Contains(sheet, "+30") {
		t.Errorf("a folha do combatente não diz os 30 PV temporários que o dano come primeiro")
	}
}
