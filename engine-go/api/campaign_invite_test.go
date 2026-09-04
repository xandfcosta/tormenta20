package api

import (
	"context"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
	"t20engine/web/campaigns"
)

// UMA MESA CRIADA PELA TELA ACEITA GENTE (ALE-287).
//
// Ela não aceitava, e não por falta de gesto: o `CreateCampaign` não escrevia
// `inviteToken`, então a mesa nascia com a coluna NULA — e o `joinTable` recusa
// quem não é o dono já no `!c.Invitetoken.Valid`, antes de olhar o que a pessoa
// digitou. Com convite vazio ou com um token qualquer, as duas tentativas
// devolviam `JoinNeedsInvite` e a mesa ficava com ZERO membros.
//
// As únicas mesas em que alguém entrava eram as seis da `seed.sql`, que trazem
// `seedtoken-0N` escrito à mão.
//
// # Por que a suíte inteira passava por cima disto
//
// A bancada semeia campanha com o token DADO (`seedCampanha(t, s, dono, nome,
// convite)`), e todos os casos de entrar usavam essa porta. O teste fornecia o
// que a produção nunca fornecia — é a família do "esperado calculado", com o
// arranjo no lugar do valor esperado: um dado de fixture que o código sob teste
// não sabe produzir esconde exatamente o defeito de quem o produz.
//
// Por isso este caso chama o `CreateCampaign` do jeito que a CENA chama, com os
// mesmos parâmetros e mais nada.
func TestACampaignBornOnScreenLetsAPlayerIn(t *testing.T) {
	s := newTestServer(t)
	mestre := seedUser(t, s, "mestre@t20.local")
	jogador := seedUser(t, s, "jogador@t20.local")
	heroi := seedCharacter(t, s, jogador, "Visitante", 10, 10, 0, 0)

	agora := plataforma.NowISO()
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

	agora := plataforma.NowISO()
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
