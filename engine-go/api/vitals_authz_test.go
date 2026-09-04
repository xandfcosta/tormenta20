package api

import "t20engine/aovivo"

import (
	"context"
	"database/sql"
	"strings"
	"t20engine/plataforma"
	"testing"

	"t20engine/db/sqlcgen"
)

// A fronteira de quem pode mexer no PV de quem vive no gateway de socket, e nunca
// tinha executado sob teste: a MESMA regra estava afirmada no componente do front,
// que é a camada que não é dona dela — gating de UI é UX, o servidor é a trava.
//
// O gateway aqui é construído sem `io` de propósito: `assertVitalsEditable` decide
// e devolve erro, quem emite é o chamador. Se um dia ela precisar do socket para
// decidir, este teste para de compilar — e essa é a intenção.

// vitalsFixture monta uma mesa com mestre, dois jogadores com personagem e um NPC na
// iniciativa, devolvendo o gateway e os ids que os testes usam.
type vitalsFixture struct {
	srv       *Server
	sessionID int64
	gmUser    int64
	player    int64
	other     int64
	pcEntry   string
	otherPc   string
	npcEntry  string
}

func newVitalsFixture(t *testing.T) vitalsFixture {
	t.Helper()
	s := newTestServer(t)
	ctx := context.Background()
	gmUser := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	other := seedUser(t, s, "outro@t.com")
	campaignID := seedCampaign(t, s, gmUser)

	pcID := seedCharacter(t, s, player, "Herói", 20, 30, 5, 10)
	otherID := seedCharacter(t, s, other, "Colega", 20, 30, 5, 10)
	seedMember(t, s, campaignID, pcID, "player")
	seedMember(t, s, campaignID, otherID, "player")

	sess, err := s.queries.CreateSession(ctx, sqlcgen.CreateSessionParams{
		Campaignid: campaignID, Sessionnumber: 1, Title: sql.NullString{String: "S1", Valid: true},
		Createdat: plataforma.NowISO(), Updatedat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}

	srv := s
	// O id da entrada é do SERVIDOR (`aovivo.AddEntry` sobrescreve o que vem do cliente),
	// então o teste lê de volta o que ele gerou em vez de inventar um.
	Add := func(label, kind string, characterID *int64) string {
		state, err := srv.sessions.AddInitiativeEntry(sess.ID, aovivo.InitiativeEntry{
			Label: label, Initiative: 10, Type: kind, CharacterID: characterID,
		})
		if err != nil {
			t.Fatalf("seed entry %q: %v", label, err)
		}
		for _, e := range state.Initiative {
			if e.Label == label {
				return e.ID
			}
		}
		t.Fatalf("entrada %q não voltou no estado", label)
		return ""
	}

	return vitalsFixture{
		srv: srv, sessionID: sess.ID, gmUser: gmUser, player: player, other: other,
		pcEntry:  Add("Herói", "character", &pcID),
		otherPc:  Add("Colega", "character", &otherID),
		npcEntry: Add("Ogro", "npc", nil),
	}
}

func (f vitalsFixture) ctx(UserID int64, Role string) liveCtx {
	return liveCtx{UserID: UserID, sessionID: f.sessionID, Role: Role}
}

func TestAssertVitalsEditable(t *testing.T) {
	f := newVitalsFixture(t)

	t.Run("o mestre edita qualquer combatente", func(t *testing.T) {
		for _, entry := range []string{f.pcEntry, f.otherPc, f.npcEntry} {
			if err := f.srv.assertVitalsEditableFor(context.Background(), f.ctx(f.gmUser, "gm"), entry); err != nil {
				t.Errorf("mestre em %q: %v", entry, err)
			}
		}
	})

	t.Run("o jogador edita o próprio personagem", func(t *testing.T) {
		if err := f.srv.assertVitalsEditableFor(context.Background(), f.ctx(f.player, "player"), f.pcEntry); err != nil {
			t.Errorf("jogador no próprio PC: %v", err)
		}
	})

	// O caso que dói na mesa: um jogador tirando PV do personagem de outro.
	t.Run("o jogador não edita o personagem de outro", func(t *testing.T) {
		err := f.srv.assertVitalsEditableFor(context.Background(), f.ctx(f.player, "player"), f.otherPc)
		if err == nil {
			t.Fatal("jogador editou o PC de outro jogador")
		}
	})

	t.Run("o jogador não edita NPC", func(t *testing.T) {
		err := f.srv.assertVitalsEditableFor(context.Background(), f.ctx(f.player, "player"), f.npcEntry)
		if err == nil || !strings.Contains(err.Error(), "NPC") {
			t.Fatalf("err=%v, queria recusa citando NPC", err)
		}
	})

	// A mensagem carrega o id recusado: sem ele, o mestre lê "não encontrado" e
	// não sabe qual linha o cliente pediu.
	t.Run("entrada inexistente é recusada nomeando o id", func(t *testing.T) {
		err := f.srv.assertVitalsEditableFor(context.Background(), f.ctx(f.player, "player"), "e-fantasma")
		if err == nil || !strings.Contains(err.Error(), "e-fantasma") {
			t.Fatalf("err=%v, queria recusa citando e-fantasma", err)
		}
	})
}

// O comentário do `access` declara a ameaça — "um sessionId forjado num socket velho
// não sequestra outra mesa" — e ela dependia de `sessionForCaller` conferir que a
// sessão pertence à campanha pedida. Um mestre é mestre da PRÓPRIA mesa, e o socket
// re-resolve o papel a cada mensagem: o par (campanha minha, sessão de outro) tem de
// morrer aqui, senão o papel resolvido é "gm" e ele comanda a mesa alheia.
func TestSessionForCallerRejectsForeignSession(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	mine := seedUser(t, s, "meu@t.com")
	theirs := seedUser(t, s, "alheio@t.com")
	myCampaign := seedCampaign(t, s, mine)
	theirCampaign := seedCampaign(t, s, theirs)

	foreign, err := s.queries.CreateSession(ctx, sqlcgen.CreateSessionParams{
		Campaignid: theirCampaign, Sessionnumber: 1, Title: sql.NullString{String: "Alheia", Valid: true},
		Createdat: plataforma.NowISO(), Updatedat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed foreign session: %v", err)
	}

	_, Role, status, err := s.campaignRules().sessionForCaller(ctx, AuthUser{ID: mine}, myCampaign, foreign.ID)
	if err == nil || status == 200 {
		t.Fatalf("status=%d Role=%q err=%v — a sessão de outra mesa foi aceita", status, Role, err)
	}
}
