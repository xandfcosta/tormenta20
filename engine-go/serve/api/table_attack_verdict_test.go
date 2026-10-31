package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// O VEREDITO DO ATAQUE É SÓ DO MESTRE (ALE-532, ALE-536): ele decide com o
// duelo à vista e a sugestão do livro, e o jogador vê a própria rolagem — no
// papel do personagem, ele não conhece a Defesa do alvo.
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
	roll := fmt.Sprintf("d20 (%d)", pending.Roll)

	// O MESTRE vê o duelo — a rolagem na notação contra a Defesa — e decide.
	gm := attackSection(t, f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String())
	for _, word := range verdicts {
		if !strings.Contains(gm, word) {
			t.Errorf("o mestre não tem o botão %q para decidir:\n%s", word, gm)
		}
	}
	if !strings.Contains(gm, roll) || !strings.Contains(gm, "Defesa") ||
		!strings.Contains(gm, "<b>"+strconv.Itoa(pending.Defense)+"</b>") {
		t.Errorf("o duelo do mestre não mostra %q contra a Defesa %d:\n%s", roll, pending.Defense, gm)
	}

	// O JOGADOR vê a própria rolagem, e nada que julgue o golpe.
	player := attackSection(t, f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String())
	for _, word := range append(verdicts, "Defesa", "sugere") {
		if strings.Contains(player, word) {
			t.Errorf("o jogador vê %q — o julgamento é do mestre:\n%s", word, player)
		}
	}
	// O CONTROLE: a faixa do jogador não ficou vazia — ela diz a rolagem dele.
	if !strings.Contains(player, roll) || !strings.Contains(player, "= "+strconv.Itoa(pending.Total)) {
		t.Errorf("o jogador não vê a própria rolagem (%s … = %d):\n%s", roll, pending.Total, player)
	}
}
