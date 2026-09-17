package api

import (
	"context"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/campaigns"
)

// UMA MESA CRIADA PELA TELA ACEITA GENTE.
//
// Se o `CreateCampaign` não escrever `inviteToken`, a mesa nasce com a coluna
// NULA — e o `joinTable` recusa quem não é o dono já no `!c.Invitetoken.Valid`,
// antes de olhar o que a pessoa digitou. A mesa fica com ZERO membros, com
// convite vazio ou com um token qualquer.
//
// Por que uma suíte inteira passa por cima disto: a bancada semeia campanha com
// o token DADO, e todo caso de entrar usa essa porta. Um dado de fixture que o
// código sob teste não sabe produzir esconde exatamente o defeito de quem o
// produz. Por isso este caso chama o `CreateCampaign` do jeito que a CENA chama,
// com os mesmos parâmetros e mais nada.
func TestACampaignBornOnScreenLetsAPlayerIn(t *testing.T) {
	s := newTestServer(t)
	mestre := seedUser(t, s, "mestre@t20.local")
	jogador := seedUser(t, s, "jogador@t20.local")
	heroi := seedCharacter(t, s, jogador, "Visitante", 10, 10, 0, 0)

	agora := dbvalue.NowISO()
	c, err := s.campaignRules().createCampaign(context.Background(), sqlcgen.CreateCampaignParams{
		Ownerid: mestre, Name: "Mesa Nova", Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		t.Fatalf("criar campanha: %v", err)
	}

	if !c.Invitetoken.Valid || len(c.Invitetoken.String) < 16 {
		t.Fatalf("a mesa nasceu sem link de convite (%q) — sem ele ninguém entra", c.Invitetoken.String)
	}
	if recusa := s.campaignsHost().Join(
		context.Background(), c.ID, heroi, jogador, c.Invitetoken.String,
	); recusa != campaigns.JoinOK {
		t.Fatalf("o jogador foi recusado (%v) com o link que a própria mesa cunhou", recusa)
	}
	if n := membrosDaMesa(t, s, c.ID); n != 1 {
		t.Errorf("a mesa ficou com %d membros, esperado 1", n)
	}
}

// O link SÓ vale para a mesa que o cunhou — o par do caso acima, e o que impede
// "cunhar no nascimento" de virar "um token abre qualquer porta".
func TestALinkOnlyOpensItsOwnTable(t *testing.T) {
	s := newTestServer(t)
	mestre := seedUser(t, s, "mestre@t20.local")
	jogador := seedUser(t, s, "jogador@t20.local")
	heroi := seedCharacter(t, s, jogador, "Visitante", 10, 10, 0, 0)

	agora := dbvalue.NowISO()
	minha, err := s.campaignRules().createCampaign(context.Background(), sqlcgen.CreateCampaignParams{
		Ownerid: mestre, Name: "A minha", Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	outra, err := s.campaignRules().createCampaign(context.Background(), sqlcgen.CreateCampaignParams{
		Ownerid: mestre, Name: "A outra", Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		t.Fatalf("criar: %v", err)
	}

	if minha.Invitetoken.String == outra.Invitetoken.String {
		t.Fatal("duas mesas nasceram com o MESMO link")
	}
	if recusa := s.campaignsHost().Join(
		context.Background(), minha.ID, heroi, jogador, outra.Invitetoken.String,
	); recusa == campaigns.JoinOK {
		t.Error("o link de uma mesa abriu a porta de OUTRA")
	}
}

func membrosDaMesa(t *testing.T, s *Server, campanhaID int64) int {
	t.Helper()
	var n int
	if err := s.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM campaign_members WHERE campaignId = ?", campanhaID).Scan(&n); err != nil {
		t.Fatalf("contar membros: %v", err)
	}
	return n
}
