package api

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// O QUE O MESTRE ESCONDE NÃO VAZA POR UMA TELA IRMÃ.
//
// A regra é a da fila (`live.RedactForPlayers`): o PV de personagem nasce à vista, todo PM
// nasce escondido, e o mestre muda isso linha a linha. O cartão Grupo da tela do jogador
// montava o mesmo dado por OUTRO caminho, que não passava pela regra: na mesma tela, a
// fila dizia "PM ocultos pelo mestre" e o Grupo, logo abaixo, "PM 12/20".

var poolNumber = regexp.MustCompile(`P[VM] \d+/\d+`)

// withAColleague põe na campanha a Guerreira, de OUTRA jogadora: é dela que a regra
// esconde. O Arcanista da bancada é de quem olha, e a ficha dele é dele.
func withAColleague(t *testing.T, f sceneFixture) int64 {
	t.Helper()
	colleague := seedUser(t, f.s, "colega@t.com")
	id := seedCharacterAtLevel(t, f.s, colleague, "Guerreira", "Guerreiro", 3, 0, 0)
	seedMember(t, f.s, f.campaignID, id)
	return id
}

// partyCard recorta do Grupo da tela do jogador o cartão de um membro.
func partyCard(t *testing.T, page, name string) string {
	t.Helper()
	for _, card := range strings.Split(page, `class="ds-row ds-party-row"`)[1:] {
		if strings.Contains(card, name) {
			card, _, _ = strings.Cut(card, `class="ds-row ds-party-row"`)
			return card
		}
	}
	t.Fatalf("o Grupo não tem o cartão de %s", name)
	return ""
}

func poolsIn(card string) []string {
	return poolNumber.FindAllString(card, -1)
}

func TestTheGroupCardHidesWhatTheQueueHides(t *testing.T) {
	t.Run("fora de cena, o PM da colega é escondido e o PV não", func(t *testing.T) {
		f := newSceneFixture(t)
		withAColleague(t, f)
		page := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

		colleague := partyCard(t, page, "Guerreira")
		pools := strings.Join(poolsIn(colleague), " ")
		if !strings.Contains(pools, "PV ") {
			t.Errorf("o PV da colega, que a mesa vê por padrão, sumiu do Grupo: %q", pools)
		}
		if strings.Contains(pools, "PM ") {
			t.Errorf("o Grupo mostra o PM da colega, que nasce escondido: %q", pools)
		}
		if !strings.Contains(colleague, "PM ocultos pelo mestre") {
			t.Error("o Grupo não diz que o PM da colega existe e está escondido")
		}
		// A ficha de quem olha é dele: o Arcanista mostra os dois.
		if mine := strings.Join(poolsIn(partyCard(t, page, "Arcanista")), " "); !strings.Contains(mine, "PV ") || !strings.Contains(mine, "PM ") {
			t.Errorf("o Grupo escondeu de quem olha o próprio personagem: %q", mine)
		}
	})

	t.Run("em cena, o PV que o mestre esconde some do Grupo", func(t *testing.T) {
		f := newSceneFixture(t)
		withAColleague(t, f)
		f.scene(t)
		if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/por-no-mapa", ""); rec.Code != http.StatusOK {
			t.Fatalf("pôr o grupo na fila deu %d", rec.Code)
		}
		entry := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Guerreira")

		// O CONTROLE: antes do gesto, o PV dela está no Grupo.
		before := partyCard(t, f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String(), "Guerreira")
		if !strings.Contains(strings.Join(poolsIn(before), " "), "PV ") {
			t.Fatalf("o PV da colega não estava no Grupo antes de o mestre escondê-lo — o caso mediria o vazio:\n%s", before)
		}
		if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/"+entry+"/vitais/hp/oculto", ""); rec.Code != http.StatusOK {
			t.Fatalf("esconder o PV deu %d", rec.Code)
		}
		after := partyCard(t, f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String(), "Guerreira")
		if pools := strings.Join(poolsIn(after), " "); strings.Contains(pools, "PV ") {
			t.Errorf("o mestre escondeu o PV da Guerreira e o Grupo do jogador o mostra: %q", pools)
		}
	})

	// O MESTRE vê tudo: a regra é da mesa, e a página Grupo dele continua com os números.
	t.Run("o mestre vê o PM de todos", func(t *testing.T) {
		f := newSceneFixture(t)
		withAColleague(t, f)
		page := gmRequest(t, f, f.tableUrl()+"/grupo").Body.String()
		if got := len(regexp.MustCompile(`PM \d+/\d+`).FindAllString(page, -1)); got < 2 {
			t.Errorf("o Grupo do mestre mostra %d números de PM, e o grupo tem dois membros com PM", got)
		}
		// O olho riscado é o que a marca desenharia aqui, e ele cobriria o − do membro.
		if strings.Contains(page, "A mesa não vê os") {
			t.Error("o Grupo do mestre ganhou o olho riscado, que nesta página cobre o −")
		}
	})
}
