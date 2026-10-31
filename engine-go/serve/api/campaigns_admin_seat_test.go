package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

/*
O ASSENTO VENCE O ADMIN (ALE-527).

O admin passava como mestre de TODA campanha, e a semente tem o caso que mordeu:
o admin JOGA na mesa de outra pessoa, e a página da campanha lhe abria a
configuração e o acervo, enquanto a mesa da mesma campanha o
tratava como jogador. Sem assento ele continua mestrando — é o que o
`TestTheAdminEditsSomeoneElsesCampaignAndAPlayerStillCannot` prende.
*/

// O admin com herói na mesa de outra pessoa é JOGADOR nela: a página não lhe
// abre a configuração, e as escritas o recusam.
func TestTheAdminSeatedAsAPlayerIsAPlayer(t *testing.T) {
	s := newTestServer(t, "admin@t20.local")
	owner := seedUser(t, s, "dono@t20.local")
	admin := seedUser(t, s, "admin@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")
	hero := seedCharacterAtLevel(t, s, admin, "Bardo", "Bardo", 1, 0, 5)
	seedMember(t, s, campaign, hero)

	body := pedeNaCronica(t, s, admin, http.MethodGet, placesTabUrl(campaign), "").Body.String()
	// CONTROLE: ele recebeu a página da campanha, e não uma recusa.
	if !strings.Contains(body, "A Queda de Tauron") {
		t.Fatal("o admin não recebeu a página da campanha: o guarda mediu outra coisa")
	}
	if !strings.Contains(body, "Jogando") {
		t.Error("a página não diz \"Jogando\" para o admin que joga nesta mesa")
	}
	if strings.Contains(body, "Novo lugar") {
		t.Error("o acervo de lugares foi aberto para o admin que só joga nesta mesa")
	}

	for _, write := range []struct{ name, path, form string }{
		{"criar lugar", strings.TrimSuffix(placesTabUrl(campaign), "?tab=lugares") + "/lugares/novo", "name=Cripta&ground=crypt"},
		{"editar", strings.TrimSuffix(placesTabUrl(campaign), "?tab=lugares") + "/editar", "name=Outro+nome"},
	} {
		if rec := pedeNaCronica(t, s, admin, http.MethodPost, write.path, write.form); rec.Code != http.StatusForbidden {
			t.Errorf("%s: o admin que joga nesta mesa recebeu %d; só o dono escreve (403)", write.name, rec.Code)
		}
	}
	c, _ := s.queries.GetCampaign(context.Background(), campaign)
	if c.Name != "A Queda de Tauron" {
		t.Errorf("o admin-jogador renomeou a campanha para %q", c.Name)
	}
}
