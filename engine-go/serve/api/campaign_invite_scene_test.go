package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"strconv"
)

// A CONFIGURAÇÃO MOSTRA O LINK, e ele é o caminho de entrar com o token da mesa.
//
// Este caso anda pela tela de verdade — abrir a campanha pelo endereço, na aba
// de configuração — porque o que ele prende é a COMPOSIÇÃO: a regra cunha, a
// vista carrega, e o painel desenha. Cada uma das três funcionando sozinha não
// diz nada sobre o mestre conseguir copiar um link.
func TestTheConfigTabShowsTheTableLink(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")

	create := postaFolhaNova(t, s, gm, "Mesa do Beco", "")
	if create.Code != http.StatusSeeOther {
		t.Fatalf("abrir campanha: esperado 303, veio %d", create.Code)
	}
	destination := create.Header().Get("Location")

	rec := pedeNaCronica(t, s, gm, http.MethodGet, destination+"?tab=config", "")
	body := rec.Body.String()

	if !strings.Contains(body, `id="invite-panel"`) {
		t.Fatalf("a aba de configuração não traz o painel do link:\n%s", primeiros(body, 400))
	}
	// O CAMINHO e não a URL: quem prefixa a origem é o navegador, e um `r.Host`
	// aqui daria um link morto quando houver proxy na frente (ver
	// `ui.MintedInvite`).
	if !strings.Contains(body, `data-caminho="/campanhas/entrar?token=`) {
		t.Errorf("o painel não trouxe o caminho do convite:\n%s", primeiros(body, 600))
	}
	if strings.Contains(body, "Esta mesa não tem link") {
		t.Error("a mesa recém-aberta apareceu SEM link — ela nasce com um desde a ALE-287")
	}
}

// GERAR OUTRO DERRUBA O ANTERIOR, e é o que o botão promete em letra miúda.
func TestGeneratingANewLinkDropsTheOldOne(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")
	id := campanhaAberta(t, s, gm)

	before := s.campaignLifecycle().InviteOf(context.Background(), id)
	rec := pedeNaCronica(t, s, gm, http.MethodPost, "/campanhas/"+strconv.FormatInt(id, 10)+"/convite", "")
	after := s.campaignLifecycle().InviteOf(context.Background(), id)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("gerar link: esperado 303, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if before == "" || after == "" {
		t.Fatalf("um dos dois links veio vazio (antes %q, depois %q)", before, after)
	}
	if before == after {
		t.Error("o link não mudou — quem já tinha o antigo continua entrando")
	}
}

// QUEM NÃO MESTRA NÃO GERA. A tela nem desenha o botão, mas isso é UX: a
// fronteira é o servidor, e é a mesma trava de editar e excluir — o
// `Access.OwnedCampaign`, dentro do caso de uso (ALE-348).
func TestOnlyTheOwnerRotatesTheLink(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")
	stranger := seedUser(t, s, "estranho@t20.local")
	id := campanhaAberta(t, s, gm)

	before := s.campaignLifecycle().InviteOf(context.Background(), id)
	rec := pedeNaCronica(t, s, stranger, http.MethodPost, "/campanhas/"+strconv.FormatInt(id, 10)+"/convite", "")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("um estranho recebeu %d ao gerar link da mesa alheia, esperado 403", rec.Code)
	}
	if after := s.campaignLifecycle().InviteOf(context.Background(), id); after != before {
		t.Error("o link mudou mesmo com a recusa")
	}
}

func campanhaAberta(t *testing.T, s *Server, ownerID int64) int64 {
	t.Helper()
	rec := postaFolhaNova(t, s, ownerID, "Mesa do Beco", "")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("abrir campanha: esperado 303, veio %d", rec.Code)
	}
	destination := rec.Header().Get("Location")
	id, err := strconv.ParseInt(strings.TrimPrefix(destination, "/campanhas/"), 10, 64)
	if err != nil {
		t.Fatalf("id da campanha em %q: %v", destination, err)
	}
	return id
}

func primeiros(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
