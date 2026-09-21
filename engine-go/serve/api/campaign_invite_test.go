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
	gm := seedUser(t, s, "mestre@t20.local")
	player := seedUser(t, s, "jogador@t20.local")
	hero := seedCharacter(t, s, player, "Visitante")

	id, err := s.campaignLifecycle().Open(context.Background(), gm, "Mesa Nova", "")
	if err != nil {
		t.Fatalf("abrir campanha: %v", err)
	}

	invite := s.campaignLifecycle().InviteOf(context.Background(), id)
	if len(invite) < 16 {
		t.Fatalf("a mesa nasceu sem link de convite (%q) — sem ele ninguém entra", invite)
	}
	if err := s.campaignSeating().Seat(context.Background(), player, id, hero, invite); err != nil {
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
	gm := seedUser(t, s, "mestre@t20.local")
	player := seedUser(t, s, "jogador@t20.local")
	hero := seedCharacter(t, s, player, "Visitante")

	mine, err := s.campaignLifecycle().Open(context.Background(), gm, "A minha", "")
	if err != nil {
		t.Fatalf("abrir: %v", err)
	}
	other, err := s.campaignLifecycle().Open(context.Background(), gm, "A outra", "")
	if err != nil {
		t.Fatalf("abrir: %v", err)
	}

	myInvite := s.campaignLifecycle().InviteOf(context.Background(), mine)
	otherInvite := s.campaignLifecycle().InviteOf(context.Background(), other)
	if myInvite == otherInvite {
		t.Fatal("duas mesas nasceram com o MESMO link")
	}
	if err := s.campaignSeating().Seat(context.Background(), player, mine, hero, otherInvite); err == nil {
		t.Error("o link de uma mesa abriu a porta de OUTRA")
	}
}

func membrosDaMesa(t *testing.T, s *Server, campaignID int64) int {
	t.Helper()
	var n int
	if err := s.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM campaign_members WHERE campaignId = ?", campaignID).Scan(&n); err != nil {
		t.Fatalf("contar membros: %v", err)
	}
	return n
}
