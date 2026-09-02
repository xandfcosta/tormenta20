package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Os guardas da FOLHA EM BRANCO (ALE-246).
//
// O que se protege é o que uma recusa custa: o texto digitado. O resto da tela
// é marcação, e marcação se confere olhando.

func postaFolhaNova(t *testing.T, s *Server, userID int64, nome, descricao string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"name": {nome}, "description": {descricao}}
	u, err := s.queries.GetUserByID(t.Context(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.signToken(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/campanhas/nova", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	return rec
}

// A RECUSA NÃO PODE COMER O TEXTO. Um nome de 121 caracteres devolvendo a folha
// vazia levaria junto a descrição inteira — que é o campo caro de reescrever, e
// o único que alguém digita por minutos.
func TestTheRefusalGivesBackWhatWasTyped(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	const texto = "A caravana parte de Valkaria ao amanhecer."

	rec := postaFolhaNova(t, s, dono, "   ", texto)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, queria 422", rec.Code)
	}
	corpo := rec.Body.String()
	if !strings.Contains(corpo, texto) {
		t.Error("a descrição sumiu na recusa — o trabalho da pessoa foi embora junto com o erro")
	}
	if !strings.Contains(corpo, "O nome é obrigatório") {
		t.Errorf("a recusa não diz o que houve:\n%s", corpo)
	}
}

// O caminho feliz vai para a crônica recém-aberta, e com 303. O `See Other` é o
// que garante que o navegador siga com GET: com 302, recarregar a página de
// destino reenviaria o formulário e abriria uma segunda campanha igual.
func TestOpeningACampaignLandsOnItWithACleanHistoryView(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")

	rec := postaFolhaNova(t, s, dono, "A Queda de Tauron", "")

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, queria 303 — com 302 recarregar reenviaria o POST", rec.Code)
	}
	// A crônica virou cena do servidor na ALE-255, então abrir uma campanha
	// leva à página DELA e não mais à da SPA.
	destino := rec.Header().Get("Location")
	if !strings.HasPrefix(destino, "/campanhas/") {
		t.Errorf("destino = %q, queria a crônica recém-aberta", destino)
	}

	// E ela existe de verdade, com o nome aparado.
	lista, err := s.queries.ListCampaignsForUser(t.Context(), dono)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(lista) != 1 || lista[0].Name != "A Queda de Tauron" {
		t.Errorf("campanhas do dono = %+v", lista)
	}
}

// A folha usa a MESMA regra da rota JSON, e o caso que prova isso é o do teto
// da descrição: ele não existia no servidor antes desta fatia, e é o que a
// virada teria apagado ao levar o formulário da SPA embora.
func TestTheFormRefusesADescriptionAboveTheCeiling(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")

	rec := postaFolhaNova(t, s, dono, "Nome bom", strings.Repeat("a", descricaoDeCampanhaMax+1))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, queria 422 — o teto de 2000 vivia só na tela da SPA", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "2000 caracteres") {
		t.Error("a recusa não diz qual é o limite")
	}
}
