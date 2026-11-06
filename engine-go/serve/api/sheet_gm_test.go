package api

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// O MESTRE ABRE A CÓPIA DO JOGADOR na campanha dele, e pode editar (decisão do dono). O rascunho continua
// só do jogador, e o mestre de outra mesa não alcança a cópia.

func TestTheGMOpensAndEditsThePlayersCopy(t *testing.T) {
	f := newSceneFixture(t)
	draft := seedCharacterAtLevel(t, f.s, f.player, "Kharvos", "Guerreiro", 3, 0, 0)
	_, copyID := seatsACopy(t, f, draft, "A Queda de Tauron")

	rec := f.requests(t, f.gm, http.MethodGet, fmt.Sprintf("/personagens/%d", copyID), "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Kharvos") {
		t.Fatalf("o mestre não abriu a cópia do jogador: %d", rec.Code)
	}
	before := poolsOf(t, f.s, copyID).HpCurrent
	edit := f.requests(t, f.gm, http.MethodPost, fmt.Sprintf("/personagens/%d/vitais/pv/-1?tab=combat", copyID), "{}")
	if edit.Code != http.StatusOK || sceneRefusal(edit.Body.String()) != "" {
		t.Fatalf("o mestre não editou a cópia do jogador: %d %q", edit.Code, sceneRefusal(edit.Body.String()))
	}
	if after := poolsOf(t, f.s, copyID).HpCurrent; after != before-1 {
		t.Errorf("o PV da cópia foi de %d a %d, e o mestre tirou 1", before, after)
	}
}

func TestTheGMDoesNotReachThePlayersDraft(t *testing.T) {
	f := newSceneFixture(t)
	draft := seedCharacterAtLevel(t, f.s, f.player, "Kharvos", "Guerreiro", 3, 0, 0)
	seatsACopy(t, f, draft, "A Queda de Tauron")
	if rec := f.requests(t, f.gm, http.MethodGet, fmt.Sprintf("/personagens/%d", draft), ""); rec.Code != http.StatusForbidden {
		t.Errorf("o mestre abriu o rascunho do jogador: %d", rec.Code)
	}
	if rec := f.requests(t, f.gm, http.MethodPost, fmt.Sprintf("/personagens/%d/vitais/pv/-1", draft), "{}"); rec.Code != http.StatusForbidden {
		t.Errorf("o mestre mexeu no rascunho do jogador: %d", rec.Code)
	}
}

func TestTheGMOfAnotherTableDoesNotReachTheCopy(t *testing.T) {
	f := newSceneFixture(t)
	draft := seedCharacterAtLevel(t, f.s, f.player, "Kharvos", "Guerreiro", 3, 0, 0)
	_, copyID := seatsACopy(t, f, draft, "A Queda de Tauron")
	stranger := seedUser(t, f.s, "outro-mestre@t.com")
	seedCampaign(t, f.s, stranger)
	if rec := f.requests(t, stranger, http.MethodGet, fmt.Sprintf("/personagens/%d", copyID), ""); rec.Code != http.StatusForbidden {
		t.Errorf("o mestre de outra mesa abriu a cópia: %d", rec.Code)
	}
	if rec := f.requests(t, stranger, http.MethodPost, fmt.Sprintf("/personagens/%d/vitais/pv/-1", copyID), "{}"); rec.Code != http.StatusForbidden {
		t.Errorf("o mestre de outra mesa mexeu na cópia: %d", rec.Code)
	}
	if rec := f.requests(t, stranger, http.MethodGet, fmt.Sprintf("/personagens/%d/ao-vivo", copyID), ""); rec.Code != http.StatusForbidden {
		t.Errorf("o mestre de outra mesa abriu o fluxo da cópia: %d", rec.Code)
	}
}

// O fluxo ao vivo da cópia abre para o mestre, que é quem a abre em outra aba pela fila.
func TestTheGMOpensTheCopysLiveStream(t *testing.T) {
	f := newSceneFixture(t)
	draft := seedCharacterAtLevel(t, f.s, f.player, "Kharvos", "Guerreiro", 3, 0, 0)
	_, copyID := seatsACopy(t, f, draft, "A Queda de Tauron")
	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()
	req, err := http.NewRequest(http.MethodGet, srv.URL+fmt.Sprintf("/personagens/%d/ao-vivo", copyID), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.gm))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("o fluxo da cópia respondeu %d ao mestre", resp.StatusCode)
	}
	if frame := lerQuadro(t, bufio.NewReader(resp.Body)); frame == "" {
		t.Error("o fluxo da cópia não mandou a versão de agora ao mestre")
	}
}
