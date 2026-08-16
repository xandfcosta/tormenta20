package api

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"t20engine/db/sqlcgen"
)

// Gestão de membros pelo ROUTER real. `PATCH /campaigns/{cid}/members/{id}` promove
// alguém a mestre da mesa, e só tinha o 401 anônimo da tabela de rotas — nada provava
// que um membro comum não se promove sozinho, nem que um id de OUTRA mesa é recusado.

// memberFixture: uma mesa do dono, um jogador membro, e uma segunda mesa com o membro
// dela — o vizinho que os testes usam para tentar atravessar a fronteira.
type memberFixture struct {
	s          *Server
	owner      int64
	player     int64
	campaignID int64
	memberID   int64
	otherOwner int64
	otherCamp  int64
	otherMemID int64
}

func newMemberFixture(t *testing.T) memberFixture {
	t.Helper()
	s := newTestServer(t)
	ctx := context.Background()
	owner := seedUser(t, s, "dono@t.com")
	player := seedUser(t, s, "jogador@t.com")
	otherOwner := seedUser(t, s, "vizinho@t.com")

	campaignID := seedCampaign(t, s, owner)
	otherCamp := seedCampaign(t, s, otherOwner)
	pcID := seedCharacter(t, s, player, "Herói", 20, 30, 5, 10)
	otherPc := seedCharacter(t, s, otherOwner, "Vizinho", 20, 30, 5, 10)

	member, err := s.queries.CreateMember(ctx, sqlcgen.CreateMemberParams{
		Campaignid: campaignID, Characterid: pcID, Role: "player", Addedat: nowISO(),
	})
	if err != nil {
		t.Fatalf("seed member: %v", err)
	}
	otherMember, err := s.queries.CreateMember(ctx, sqlcgen.CreateMemberParams{
		Campaignid: otherCamp, Characterid: otherPc, Role: "player", Addedat: nowISO(),
	})
	if err != nil {
		t.Fatalf("seed other member: %v", err)
	}

	return memberFixture{
		s: s, owner: owner, player: player, campaignID: campaignID, memberID: member.ID,
		otherOwner: otherOwner, otherCamp: otherCamp, otherMemID: otherMember.ID,
	}
}

func (f memberFixture) roleOf(t *testing.T, memberID int64) string {
	t.Helper()
	m, err := f.s.queries.GetMember(context.Background(), memberID)
	if err != nil {
		t.Fatalf("ler membro %d: %v", memberID, err)
	}
	return m.Role
}

func (f memberFixture) patchRole(t *testing.T, caller, campaignID, memberID int64, role string) int {
	t.Helper()
	path := "/campaigns/" + strconv.FormatInt(campaignID, 10) + "/members/" + strconv.FormatInt(memberID, 10)
	return authed(t, f.s, caller, http.MethodPatch, path, `{"role":"`+role+`"}`).Code
}

func TestUpdateMemberRole(t *testing.T) {
	t.Run("o dono promove um membro a mestre", func(t *testing.T) {
		f := newMemberFixture(t)
		if code := f.patchRole(t, f.owner, f.campaignID, f.memberID, "gm"); code != http.StatusOK {
			t.Fatalf("code=%d, queria 200", code)
		}
		if got := f.roleOf(t, f.memberID); got != "gm" {
			t.Errorf("papel=%q, queria gm", got)
		}
	})

	// A escalação de privilégio: o próprio jogador se promovendo na mesa de outro.
	t.Run("o jogador não se promove", func(t *testing.T) {
		f := newMemberFixture(t)
		code := f.patchRole(t, f.player, f.campaignID, f.memberID, "gm")
		if code == http.StatusOK {
			t.Fatal("o jogador conseguiu se promover")
		}
		if got := f.roleOf(t, f.memberID); got != "player" {
			t.Errorf("papel=%q — a recusa devolveu %d mas GRAVOU mesmo assim", got, code)
		}
	})

	// O membro existe, mas é de outra mesa: o id sozinho não pode bastar.
	t.Run("membro de outra mesa é 404 e não muda", func(t *testing.T) {
		f := newMemberFixture(t)
		if code := f.patchRole(t, f.owner, f.campaignID, f.otherMemID, "gm"); code != http.StatusNotFound {
			t.Errorf("code=%d, queria 404", code)
		}
		if got := f.roleOf(t, f.otherMemID); got != "player" {
			t.Errorf("papel do vizinho=%q — mudou o membro de outra mesa", got)
		}
	})

	t.Run("papel fora da lista é recusado", func(t *testing.T) {
		f := newMemberFixture(t)
		if code := f.patchRole(t, f.owner, f.campaignID, f.memberID, "admin"); code == http.StatusOK {
			t.Fatal("papel inventado foi aceito")
		}
		if got := f.roleOf(t, f.memberID); got != "player" {
			t.Errorf("papel=%q, queria player intacto", got)
		}
	})
}
