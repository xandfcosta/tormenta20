package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/campaigns"
	"testing"
)

// Os guardas da cena de CAMPANHAS.
//
// O que se protege é o que o SERVIDOR decide: quem entra na lista, qual
// campanha aparece ao vivo, e que o cursor nasce numa que existe. O desenho é
// do e2e — ele mede contraste e o cursor andando, que são as coisas que só o
// navegador testemunha.

type cenaFixture struct {
	s     *Server
	owner int64
}

func novaCena(t *testing.T, admins ...string) cenaFixture {
	t.Helper()
	s := newTestServer(t, admins...)
	return cenaFixture{s: s, owner: seedUser(t, s, "mestre@t20.local")}
}

func (f cenaFixture) eu(t *testing.T) AuthUser {
	t.Helper()
	u, err := f.s.queries.GetUserByID(context.Background(), f.owner)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	return f.s.accountRules().authUser(u)
}

// ── a lista ──────────────────────────────────────────────────────────────────

func (f cenaFixture) campanha(t *testing.T, name, synopsis string) int64 {
	t.Helper()
	c, err := f.s.queries.CreateCampaign(context.Background(), sqlcgen.CreateCampaignParams{
		Ownerid: f.owner, Name: name, Createdat: dbvalue.NowISO(), Updatedat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("criar campanha: %v", err)
	}
	if synopsis != "" {
		if _, err := f.s.db.ExecContext(context.Background(),
			`UPDATE campaigns SET description = ? WHERE id = ?`, synopsis, c.ID); err != nil {
			t.Fatalf("sinopse: %v", err)
		}
	}
	return c.ID
}

// A busca é do SERVIDOR nesta cena, e a regra é a mesma do `casaBusca`. Este
// guarda é a costura: que a cena de fato APLICA a regra, sobre o nome E a
// sinopse.
func TestTheSceneFiltersBySearchOverNameAndSynopsis(t *testing.T) {
	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")
	f.campanha(t, "Segredos de Wynlla", "Uma trama sobre a Tormenta")

	byName, err := campaigns.New(f.s.campaignsHost(), f.s.sessionAccess(), f.s.campaignDirectory(), f.s.campaignLifecycle(), f.s.campaignSeating(), f.s.boards).LoadList(context.Background(), f.owner, f.s.ehAdmin(t, f.owner), "queda", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(byName.Campanhas) != 1 || byName.Campanhas[0].Nome != "A Queda de Tauron" {
		t.Errorf("busca por nome devolveu %d resultados", len(byName.Campanhas))
	}

	bySynopsis, err := campaigns.New(f.s.campaignsHost(), f.s.sessionAccess(), f.s.campaignDirectory(), f.s.campaignLifecycle(), f.s.campaignSeating(), f.s.boards).LoadList(context.Background(), f.owner, f.s.ehAdmin(t, f.owner), "tormenta", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(bySynopsis.Campanhas) != 1 || bySynopsis.Campanhas[0].Nome != "Segredos de Wynlla" {
		t.Errorf("busca por sinopse devolveu %d resultados", len(bySynopsis.Campanhas))
	}
}

// O cursor tem de nascer numa campanha que EXISTE na lista filtrada. Se ele
// ficasse na primeira da lista COMPLETA, uma busca que a filtrasse fora
// deixaria o palco vazio com o trilho cheio — e a tela pareceria quebrada.
func TestTheCursorIsBornOnTheFirstOfTheFilteredList(t *testing.T) {
	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")
	second := f.campanha(t, "Segredos de Wynlla", "")

	v, err := campaigns.New(f.s.campaignsHost(), f.s.sessionAccess(), f.s.campaignDirectory(), f.s.campaignLifecycle(), f.s.campaignSeating(), f.s.boards).LoadList(context.Background(), f.owner, f.s.ehAdmin(t, f.owner), "wynlla", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if v.CursorID != second {
		t.Errorf("cursor = %d, queria %d (a única que sobrou)", v.CursorID, second)
	}
}

// Buscar e não achar nada é DIFERENTE de não ter campanha nenhuma: uma pede
// para limpar o filtro, a outra para criar a primeira.
func TestTheSceneTellsAnEmptyListFromASearchWithNoResult(t *testing.T) {
	empty := novaCena(t)
	nothing, err := campaigns.New(empty.s.campaignsHost(), empty.s.sessionAccess(), empty.s.campaignDirectory(), empty.s.campaignLifecycle(), empty.s.campaignSeating(), empty.s.boards).LoadList(context.Background(), empty.eu(t).ID, empty.eu(t).IsAdmin, "", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if nothing.TemAlguma || nothing.FiltrouTudo {
		t.Errorf("lista vazia: TemAlguma=%v FiltrouTudo=%v", nothing.TemAlguma, nothing.FiltrouTudo)
	}

	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")
	noResult, err := campaigns.New(f.s.campaignsHost(), f.s.sessionAccess(), f.s.campaignDirectory(), f.s.campaignLifecycle(), f.s.campaignSeating(), f.s.boards).LoadList(context.Background(), f.owner, f.s.ehAdmin(t, f.owner), "zzzzz", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if !noResult.TemAlguma || !noResult.FiltrouTudo {
		t.Errorf("busca sem resultado: TemAlguma=%v FiltrouTudo=%v", noResult.TemAlguma, noResult.FiltrouTudo)
	}
}

// ── a sessão viva ────────────────────────────────────────────────────────────

// A consulta única substitui uma fan-out de N+1, e o que ela tem de acertar é
// ATRIBUIR a sessão à campanha certa: trocar duas faz o "Continuar" levar para a
// mesa errada.
func TestALiveSessionGoesToTheRightCampaign(t *testing.T) {
	f := novaCena(t)
	stop := f.campanha(t, "A Queda de Tauron", "")
	scrolling := f.campanha(t, "Segredos de Wynlla", "")
	f.campanha(t, "O Chamado", "")

	stoppedSession := seedSession(t, f.s, stop)
	liveSession := seedSession(t, f.s, scrolling)
	_ = stoppedSession
	if _, err := f.s.queries.StartSessionFresh(context.Background(), sqlcgen.StartSessionFreshParams{
		UpdatedAt: dbvalue.NowISO(), ID: liveSession,
	}); err != nil {
		t.Fatalf("iniciar: %v", err)
	}

	v, err := campaigns.New(f.s.campaignsHost(), f.s.sessionAccess(), f.s.campaignDirectory(), f.s.campaignLifecycle(), f.s.campaignSeating(), f.s.boards).LoadList(context.Background(), f.owner, f.s.ehAdmin(t, f.owner), "", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	for _, c := range v.Campanhas {
		wantLive := c.ID == scrolling
		if c.AoVivo != wantLive {
			t.Errorf("%q: AoVivo=%v, queria %v", c.Nome, c.AoVivo, wantLive)
		}
		if c.AoVivo && c.SessaoID != liveSession {
			t.Errorf("%q aponta para a sessão %d, queria %d", c.Nome, c.SessaoID, liveSession)
		}
	}
}

// ── o filtro de papel ────────────────────────────────────────────────────────

// Um valor de papel que não existe — vindo de uma URL editada à mão — vira
// "todas". Esconder a lista inteira por causa de um parâmetro estranho é a
// tela mentindo sobre o que a pessoa tem.
func TestAnInvalidRoleInTheUrlDoesNotHideTheList(t *testing.T) {
	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")

	for _, role := range []string{"", "mestre", "GM", "'; drop table"} {
		v, err := campaigns.New(f.s.campaignsHost(), f.s.sessionAccess(), f.s.campaignDirectory(), f.s.campaignLifecycle(), f.s.campaignSeating(), f.s.boards).LoadList(context.Background(), f.owner, f.s.ehAdmin(t, f.owner), "", role)
		if err != nil {
			t.Fatalf("carregar: %v", err)
		}
		if v.Papel != "todas" || len(v.Campanhas) != 1 {
			t.Errorf("papel %q virou %q com %d campanhas", role, v.Papel, len(v.Campanhas))
		}
	}
}

func TestTheRoleFilterSeparatesRunningFromPlaying(t *testing.T) {
	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")

	mastering, err := campaigns.New(f.s.campaignsHost(), f.s.sessionAccess(), f.s.campaignDirectory(), f.s.campaignLifecycle(), f.s.campaignSeating(), f.s.boards).LoadList(context.Background(), f.owner, f.s.ehAdmin(t, f.owner), "", "gm")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(mastering.Campanhas) != 1 {
		t.Errorf("mestrando devolveu %d — o dono mestra a própria mesa", len(mastering.Campanhas))
	}

	playing, err := campaigns.New(f.s.campaignsHost(), f.s.sessionAccess(), f.s.campaignDirectory(), f.s.campaignLifecycle(), f.s.campaignSeating(), f.s.boards).LoadList(context.Background(), f.owner, f.s.ehAdmin(t, f.owner), "", "player")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(playing.Campanhas) != 0 {
		t.Errorf("jogando devolveu %d — o dono não JOGA na própria mesa", len(playing.Campanhas))
	}
}

// ── a rota ───────────────────────────────────────────────────────────────────

// A carga fria devolve a PÁGINA; o pedido do Datastar devolve o REMENDO. Uma
// rota só serve os dois, e quem distingue é o cabeçalho que o cliente põe.
func TestTheSceneAnswersPageOrPatchDependingOnWhoAsks(t *testing.T) {
	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")
	tok, err := f.s.accountGate().SignSession(sqlcgen.User{ID: f.owner, Email: "mestre@t20.local"})
	if err != nil {
		t.Fatalf("assinar: %v", err)
	}
	asks := func(datastar bool) string {
		req := httptest.NewRequest(http.MethodGet, "/campanhas", nil)
		req.AddCookie(&http.Cookie{Name: f.s.cfg.CookieName, Value: tok})
		if datastar {
			req.Header.Set("datastar-request", "true")
		}
		rec := httptest.NewRecorder()
		f.s.WebRouter().ServeHTTP(rec, req)
		return rec.Body.String()
	}

	page := asks(false)
	if !strings.Contains(page, "<!doctype html>") && !strings.Contains(page, "<!DOCTYPE html>") {
		t.Error("a carga fria não devolveu o documento inteiro")
	}
	patch := asks(true)
	if strings.Contains(patch, "<html") {
		t.Error("o pedido do Datastar recebeu a página inteira — o remendo viraria um documento dentro do outro")
	}
	if !strings.Contains(patch, "campaigns-scene") {
		t.Error("o remendo não trouxe o id que o morph casa")
	}
}
