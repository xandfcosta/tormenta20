package api

import (
	"context"
	"testing"
)

// UMA MESA CRIADA PELA TELA ACEITA GENTE.
//
// Se o `CreateCampaign` não escrever `inviteToken`, a mesa nasce com a coluna
// NULA — e o `Seat` recusa quem não é o dono já no `!c.Invitetoken.Valid`,
// antes de olhar o que a pessoa digitou. A mesa fica com ZERO membros, com
// convite vazio ou com um token qualquer.
//
// Por que uma suíte inteira passa por cima disto: a bancada semeia campanha com
// o token DADO, e todo caso de entrar usa essa porta. Um dado de fixture que o
// código sob teste não sabe produzir esconde exatamente o defeito de quem o
// produz. Por isso este caso abre a mesa pelo MESMO caso de uso que a cena usa,
// com os mesmos argumentos e mais nada.
func TestACampaignBornOnScreenLetsAPlayerIn(t *testing.T) {
	s := newTestServer(t)
	mestre := seedUser(t, s, "mestre@t20.local")
	jogador := seedUser(t, s, "jogador@t20.local")
	heroi := seedCharacter(t, s, jogador, "Visitante", 10, 10, 0, 0)

	id, err := s.campaignLifecycle().Open(context.Background(), mestre, "Mesa Nova", "")
	if err != nil {
		t.Fatalf("abrir campanha: %v", err)
	}

	convite := s.campaignLifecycle().InviteOf(context.Background(), id)
	if len(convite) < 16 {
		t.Fatalf("a mesa nasceu sem link de convite (%q) — sem ele ninguém entra", convite)
	}
	if err := s.campaignSeating().Seat(context.Background(), jogador, id, heroi, convite); err != nil {
		t.Fatalf("o jogador foi recusado (%v) com o link que a própria mesa cunhou", err)
	}
	if n := membrosDaMesa(t, s, id); n != 1 {
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

	minha, err := s.campaignLifecycle().Open(context.Background(), mestre, "A minha", "")
	if err != nil {
		t.Fatalf("abrir: %v", err)
	}
	outra, err := s.campaignLifecycle().Open(context.Background(), mestre, "A outra", "")
	if err != nil {
		t.Fatalf("abrir: %v", err)
	}

	conviteDaMinha := s.campaignLifecycle().InviteOf(context.Background(), minha)
	conviteDaOutra := s.campaignLifecycle().InviteOf(context.Background(), outra)
	if conviteDaMinha == conviteDaOutra {
		t.Fatal("duas mesas nasceram com o MESMO link")
	}
	if err := s.campaignSeating().Seat(context.Background(), jogador, minha, heroi, conviteDaOutra); err == nil {
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
