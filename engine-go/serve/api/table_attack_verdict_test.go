package api

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// O VEREDITO DO ATAQUE É SÓ DO MESTRE (ALE-532, decisão do dono): acertar ou
// errar é interpretação dele, e o app a mostra ao mestre para facilitar a
// leitura dos números. O jogador vê a própria rolagem e ouve o resto em voz —
// no papel do personagem, ele não conhece a Defesa do alvo.
//
// O predicado é o PAPEL de quem olha, e por isso o caso desenha a MESMA mesa,
// com o MESMO provisório, para o mestre e para o jogador.

// attackSection devolve o cartão do ataque de uma tela da mesa.
func attackSection(t *testing.T, screen string) string {
	t.Helper()
	i := strings.Index(screen, `id="table-attack"`)
	if i < 0 {
		t.Fatal("a mesa não desenhou o cartão do ataque")
	}
	rest := screen[i:]
	return rest[:strings.Index(rest, "</section>")]
}

func TestOnlyTheGMSeesTheAttackVerdict(t *testing.T) {
	f, goblin := attackOnTurn(t)
	rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/atacar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("propor o ataque deu %d", rec.Code)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil {
		t.Fatalf("o ataque não virou provisório: %q", tableRefusal(t, rec.Body.String()))
	}
	verdicts := []string{"Acertou", "Errou", "Crítico"}
	defense := "vs " + strconv.Itoa(pending.Defense)

	gm := attackSection(t, f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String())
	sawVerdict := false
	for _, word := range verdicts {
		sawVerdict = sawVerdict || strings.Contains(gm, word)
	}
	if !sawVerdict {
		t.Errorf("o mestre não vê o veredito do ataque:\n%s", gm)
	}
	if !strings.Contains(gm, defense) {
		t.Errorf("o mestre não vê a conta contra a Defesa (%q):\n%s", defense, gm)
	}

	player := attackSection(t, f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String())
	for _, word := range verdicts {
		if strings.Contains(player, word) {
			t.Errorf("o jogador vê o veredito %q — ele é do mestre:\n%s", word, player)
		}
	}
	if strings.Contains(player, defense) {
		t.Errorf("o jogador vê a Defesa do alvo (%q):\n%s", defense, player)
	}
	// O CONTROLE: a faixa do jogador não ficou vazia — ela diz a rolagem dele.
	own := "= " + strconv.Itoa(pending.Total)
	if !strings.Contains(player, own) {
		t.Errorf("o jogador não vê a própria rolagem (%q):\n%s", own, player)
	}
}
