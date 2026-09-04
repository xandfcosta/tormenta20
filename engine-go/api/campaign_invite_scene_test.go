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
	mestre := seedUser(t, s, "mestre@t20.local")

	criar := postaFolhaNova(t, s, mestre, "Mesa do Beco", "")
	if criar.Code != http.StatusSeeOther {
		t.Fatalf("abrir campanha: esperado 303, veio %d", criar.Code)
	}
	destino := criar.Header().Get("Location")

	rec := pedeNaCronica(t, s, mestre, http.MethodGet, destino+"?tab=config", "")
	corpo := rec.Body.String()

	if !strings.Contains(corpo, `id="painel-convite"`) {
		t.Fatalf("a aba de configuração não traz o painel do link:\n%s", primeiros(corpo, 400))
	}
	// O CAMINHO e não a URL: quem prefixa a origem é o navegador, e um `r.Host`
	// aqui daria um link morto quando houver proxy na frente (ver
	// `ui.MintedInvite`).
	if !strings.Contains(corpo, `data-caminho="/campanhas/entrar?token=`) {
		t.Errorf("o painel não trouxe o caminho do convite:\n%s", primeiros(corpo, 600))
	}
	if strings.Contains(corpo, "Esta mesa não tem link") {
		t.Error("a mesa recém-aberta apareceu SEM link — ela nasce com um desde a ALE-287")
	}
}

// GERAR OUTRO DERRUBA O ANTERIOR, e é o que o botão promete em letra miúda.
func TestGeneratingANewLinkDropsTheOldOne(t *testing.T) {
	s := newTestServer(t)
	mestre := seedUser(t, s, "mestre@t20.local")
	id := campanhaAberta(t, s, mestre)

	antes := s.campaignRules().inviteOf(context.Background(), id)
	rec := pedeNaCronica(t, s, mestre, http.MethodPost, "/campanhas/"+strconv.FormatInt(id, 10)+"/convite", "")
	depois := s.campaignRules().inviteOf(context.Background(), id)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("gerar link: esperado 303, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if antes == "" || depois == "" {
		t.Fatalf("um dos dois links veio vazio (antes %q, depois %q)", antes, depois)
	}
	if antes == depois {
		t.Error("o link não mudou — quem já tinha o antigo continua entrando")
	}
}

// QUEM NÃO MESTRA NÃO GERA. A tela nem desenha o botão, mas isso é UX: a
// fronteira é o servidor, e é o mesmo `ownerOrRefuse` de editar e excluir.
func TestOnlyTheOwnerRotatesTheLink(t *testing.T) {
	s := newTestServer(t)
	mestre := seedUser(t, s, "mestre@t20.local")
	estranho := seedUser(t, s, "estranho@t20.local")
	id := campanhaAberta(t, s, mestre)

	antes := s.campaignRules().inviteOf(context.Background(), id)
	rec := pedeNaCronica(t, s, estranho, http.MethodPost, "/campanhas/"+strconv.FormatInt(id, 10)+"/convite", "")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("um estranho recebeu %d ao gerar link da mesa alheia, esperado 403", rec.Code)
	}
	if depois := s.campaignRules().inviteOf(context.Background(), id); depois != antes {
		t.Error("o link mudou mesmo com a recusa")
	}
}

func campanhaAberta(t *testing.T, s *Server, donoID int64) int64 {
	t.Helper()
	rec := postaFolhaNova(t, s, donoID, "Mesa do Beco", "")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("abrir campanha: esperado 303, veio %d", rec.Code)
	}
	destino := rec.Header().Get("Location")
	id, err := strconv.ParseInt(strings.TrimPrefix(destino, "/campanhas/"), 10, 64)
	if err != nil {
		t.Fatalf("id da campanha em %q: %v", destino, err)
	}
	return id
}

func primeiros(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
