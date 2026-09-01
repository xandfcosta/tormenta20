package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"t20engine/plataforma"
	"testing"

	"t20engine/db/sqlcgen"
)

// Os guardas da cena de CAMPANHAS (ALE-234).
//
// O que se protege é o que o SERVIDOR decide: quem entra na lista, qual
// campanha aparece ao vivo, e que o cursor nasce numa que existe. O desenho é
// do e2e — ele mede contraste e o cursor andando, que são as coisas que só o
// navegador testemunha.

type cenaFixture struct {
	s    *Server
	dono int64
}

func novaCena(t *testing.T, admins ...string) cenaFixture {
	t.Helper()
	s := newTestServer(t, admins...)
	return cenaFixture{s: s, dono: seedUser(t, s, "mestre@t20.local")}
}

func (f cenaFixture) campanha(t *testing.T, nome, sinopse string) int64 {
	t.Helper()
	c, err := f.s.queries.CreateCampaign(context.Background(), sqlcgen.CreateCampaignParams{
		Ownerid: f.dono, Name: nome, Createdat: plataforma.NowISO(), Updatedat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("criar campanha: %v", err)
	}
	if sinopse != "" {
		if _, err := f.s.db.ExecContext(context.Background(),
			`UPDATE campaigns SET description = ? WHERE id = ?`, sinopse, c.ID); err != nil {
			t.Fatalf("sinopse: %v", err)
		}
	}
	return c.ID
}

func (f cenaFixture) eu(t *testing.T) AuthUser {
	t.Helper()
	u, err := f.s.queries.GetUserByID(context.Background(), f.dono)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	return f.s.authUser(u)
}

// ── a lista ──────────────────────────────────────────────────────────────────

// A busca é do SERVIDOR nesta cena, e a regra é a mesma do `casaBusca`. Este
// guarda é a costura: que a cena de fato APLICA a regra, sobre o nome E a
// sinopse, que são os dois campos que a SPA indexa.
func TestCenaFiltraPelaBuscaNoNomeENaSinopse(t *testing.T) {
	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")
	f.campanha(t, "Segredos de Wynlla", "Uma trama sobre a Tormenta")

	porNome, err := f.s.carregaCampanhas(context.Background(), f.eu(t), "queda", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(porNome.Campanhas) != 1 || porNome.Campanhas[0].Nome != "A Queda de Tauron" {
		t.Errorf("busca por nome devolveu %d resultados", len(porNome.Campanhas))
	}

	porSinopse, err := f.s.carregaCampanhas(context.Background(), f.eu(t), "tormenta", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(porSinopse.Campanhas) != 1 || porSinopse.Campanhas[0].Nome != "Segredos de Wynlla" {
		t.Errorf("busca por sinopse devolveu %d resultados", len(porSinopse.Campanhas))
	}
}

// O cursor tem de nascer numa campanha que EXISTE na lista filtrada. Se ele
// ficasse na primeira da lista COMPLETA, uma busca que a filtrasse fora
// deixaria o palco vazio com o trilho cheio — e a tela pareceria quebrada.
func TestCursorNasceNaPrimeiraDaListaFILTRADA(t *testing.T) {
	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")
	segunda := f.campanha(t, "Segredos de Wynlla", "")

	v, err := f.s.carregaCampanhas(context.Background(), f.eu(t), "wynlla", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if v.CursorID != segunda {
		t.Errorf("cursor = %d, queria %d (a única que sobrou)", v.CursorID, segunda)
	}
}

// Buscar e não achar nada é DIFERENTE de não ter campanha nenhuma: uma pede
// para limpar o filtro, a outra para criar a primeira.
func TestCenaDistingueListaVaziaDeBuscaSemResultado(t *testing.T) {
	vazia := novaCena(t)
	semNada, err := vazia.s.carregaCampanhas(context.Background(), vazia.eu(t), "", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if semNada.TemAlguma || semNada.FiltrouTudo {
		t.Errorf("lista vazia: TemAlguma=%v FiltrouTudo=%v", semNada.TemAlguma, semNada.FiltrouTudo)
	}

	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")
	semResultado, err := f.s.carregaCampanhas(context.Background(), f.eu(t), "zzzzz", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if !semResultado.TemAlguma || !semResultado.FiltrouTudo {
		t.Errorf("busca sem resultado: TemAlguma=%v FiltrouTudo=%v", semResultado.TemAlguma, semResultado.FiltrouTudo)
	}
}

// ── a sessão viva ────────────────────────────────────────────────────────────

// A consulta única substituiu uma fan-out de N+1 (a SEGUNDA da migração), e o
// que ela tem de acertar é ATRIBUIR a sessão à campanha certa: trocar duas
// faria o "Continuar" levar para a mesa errada.
func TestSessaoVivaVaiParaACampanhaCerta(t *testing.T) {
	f := novaCena(t)
	parada := f.campanha(t, "A Queda de Tauron", "")
	rolando := f.campanha(t, "Segredos de Wynlla", "")
	f.campanha(t, "O Chamado", "")

	sessaoParada := seedSession(t, f.s, parada)
	sessaoViva := seedSession(t, f.s, rolando)
	_ = sessaoParada
	if _, err := f.s.queries.StartSessionFresh(context.Background(), sqlcgen.StartSessionFreshParams{
		UpdatedAt: plataforma.NowISO(), ID: sessaoViva,
	}); err != nil {
		t.Fatalf("iniciar: %v", err)
	}

	v, err := f.s.carregaCampanhas(context.Background(), f.eu(t), "", "todas")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	for _, c := range v.Campanhas {
		querAoVivo := c.ID == rolando
		if c.AoVivo != querAoVivo {
			t.Errorf("%q: AoVivo=%v, queria %v", c.Nome, c.AoVivo, querAoVivo)
		}
		if c.AoVivo && c.SessaoID != sessaoViva {
			t.Errorf("%q aponta para a sessão %d, queria %d", c.Nome, c.SessaoID, sessaoViva)
		}
	}
}

// ── o filtro de papel ────────────────────────────────────────────────────────

// Um valor de papel que não existe — vindo de uma URL editada à mão — vira
// "todas". Esconder a lista inteira por causa de um parâmetro estranho é a
// tela mentindo sobre o que a pessoa tem.
func TestPapelInvalidoNaURLNaoEscondeALista(t *testing.T) {
	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")

	for _, papel := range []string{"", "mestre", "GM", "'; drop table"} {
		v, err := f.s.carregaCampanhas(context.Background(), f.eu(t), "", papel)
		if err != nil {
			t.Fatalf("carregar: %v", err)
		}
		if v.Papel != "todas" || len(v.Campanhas) != 1 {
			t.Errorf("papel %q virou %q com %d campanhas", papel, v.Papel, len(v.Campanhas))
		}
	}
}

func TestFiltroDePapelSeparaMestrandoDeJogando(t *testing.T) {
	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")

	mestrando, err := f.s.carregaCampanhas(context.Background(), f.eu(t), "", "gm")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(mestrando.Campanhas) != 1 {
		t.Errorf("mestrando devolveu %d — o dono mestra a própria mesa", len(mestrando.Campanhas))
	}

	jogando, err := f.s.carregaCampanhas(context.Background(), f.eu(t), "", "player")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(jogando.Campanhas) != 0 {
		t.Errorf("jogando devolveu %d — o dono não JOGA na própria mesa", len(jogando.Campanhas))
	}
}

// ── a rota ───────────────────────────────────────────────────────────────────

// A carga fria devolve a PÁGINA; o pedido do Datastar devolve o REMENDO. Uma
// rota só serve os dois, e quem distingue é o cabeçalho que o cliente põe.
func TestCenaRespondePaginaOuRemendoConformeQuemPergunta(t *testing.T) {
	f := novaCena(t)
	f.campanha(t, "A Queda de Tauron", "")
	tok, err := f.s.signToken(sqlcgen.User{ID: f.dono, Email: "mestre@t20.local"})
	if err != nil {
		t.Fatalf("assinar: %v", err)
	}
	pede := func(datastar bool) string {
		req := httptest.NewRequest(http.MethodGet, "/campanhas", nil)
		req.AddCookie(&http.Cookie{Name: f.s.cfg.CookieName, Value: tok})
		if datastar {
			req.Header.Set("datastar-request", "true")
		}
		rec := httptest.NewRecorder()
		f.s.WebRouter().ServeHTTP(rec, req)
		return rec.Body.String()
	}

	pagina := pede(false)
	if !strings.Contains(pagina, "<!doctype html>") && !strings.Contains(pagina, "<!DOCTYPE html>") {
		t.Error("a carga fria não devolveu o documento inteiro")
	}
	remendo := pede(true)
	if strings.Contains(remendo, "<html") {
		t.Error("o pedido do Datastar recebeu a página inteira — o remendo viraria um documento dentro do outro")
	}
	if !strings.Contains(remendo, "cena-campanhas") {
		t.Error("o remendo não trouxe o id que o morph casa")
	}
}
