package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// AS CÓPIAS DE CAMPANHA NA LISTA: cada rascunho mostra as mesas em que ele joga, e a ficha diz qual é.

// seatsACopy senta o rascunho numa campanha nova do mestre, e devolve a campanha e a cópia.
func seatsACopy(t *testing.T, f sceneFixture, draft int64, name string) (campaignID, copyID int64) {
	t.Helper()
	campaignID, err := f.s.campaignLifecycle().Open(context.Background(), f.gm, name, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.s.campaignSeating().Seat(context.Background(), f.player, campaignID, draft, f.s.campaignLifecycle().InviteOf(context.Background(), campaignID)); err != nil {
		t.Fatalf("sentar o rascunho: %v", err)
	}
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT id FROM characters WHERE sourceCharacterId = ? AND campaignId = ?", draft, campaignID).Scan(&copyID); err != nil {
		t.Fatalf("achar a cópia: %v", err)
	}
	return campaignID, copyID
}

// pageFrom é o trecho da página a partir do marco, e reprova com nome quando o marco não está lá.
func pageFrom(t *testing.T, page, mark string) string {
	t.Helper()
	at := strings.Index(page, mark)
	if at < 0 {
		t.Fatalf("a página não traz %q", mark)
	}
	return page[at:]
}

func castPage(t *testing.T, f sceneFixture) string {
	t.Helper()
	return f.requests(t, f.player, http.MethodGet, "/personagens", "").Body.String()
}

func TestTheDraftCardListsTheTablesItPlaysAt(t *testing.T) {
	f := newSceneFixture(t)
	draft := seedCharacterAtLevel(t, f.s, f.player, "Kharvos", "Guerreiro", 3, 0, 0)
	_, first := seatsACopy(t, f, draft, "A Queda de Tauron")
	_, second := seatsACopy(t, f, draft, "Ordem do Escudo Partido")

	page := castPage(t, f)
	card := pageFrom(t, page, "Abrir ficha de Kharvos")
	card = card[:strings.Index(card, "</article>")]
	for _, want := range []string{"Nas mesas", "A Queda de Tauron", "Ordem do Escudo Partido",
		fmt.Sprintf(`href="/personagens/%d"`, first), fmt.Sprintf(`href="/personagens/%d"`, second)} {
		if !strings.Contains(card, want) {
			t.Errorf("o cartão de Kharvos não traz %q", want)
		}
	}
	if strings.Count(page, `aria-label="Abrir ficha de Kharvos"`) != 1 {
		t.Error("a cópia virou um cartão de rascunho, e devia ser uma linha dentro do cartão dele")
	}
}

// A cópia cujo rascunho foi apagado continua na lista, num grupo próprio.
func TestACopyWithoutItsDraftStaysOnTheList(t *testing.T) {
	f := newSceneFixture(t)
	draft := seedCharacterAtLevel(t, f.s, f.player, "Vargas", "Guerreiro", 3, 0, 0)
	seatsACopy(t, f, draft, "Clã do Machado Rubro")
	if _, err := f.s.db.ExecContext(context.Background(), "DELETE FROM characters WHERE id = ?", draft); err != nil {
		t.Fatal(err)
	}
	page := castPage(t, f)
	if !strings.Contains(pageFrom(t, page, "Sem rascunho"), "Vargas") {
		t.Error("a cópia sem rascunho sumiu da lista, ou não está no grupo Sem rascunho")
	}
}

// A faixa diz qual ficha é: a da mesa leva à mesa, e a do rascunho avisa que não é a das mesas.
func TestTheSheetSaysWhichSheetItIs(t *testing.T) {
	f := newSceneFixture(t)
	draft := seedCharacterAtLevel(t, f.s, f.player, "Kharvos", "Guerreiro", 3, 0, 0)
	campaignID, copyID := seatsACopy(t, f, draft, "A Queda de Tauron")

	sheetOf := func(id int64) string {
		return f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "").Body.String()
	}
	copySheet := sheetOf(copyID)
	if !strings.Contains(copySheet, "A Queda de Tauron") || !strings.Contains(copySheet, fmt.Sprintf(`href="/campanhas/%d"`, campaignID)) {
		t.Error("a ficha da cópia não diz de que mesa ela é, nem leva a ela")
	}
	if strings.Contains(copySheet, "não chega às mesas") {
		t.Error("a ficha da cópia diz que é rascunho")
	}
	if !strings.Contains(sheetOf(draft), "o que muda aqui não chega às mesas") {
		t.Error("a ficha do rascunho não avisa que ele não é a ficha das mesas")
	}
}

// O fluxo ao vivo da ficha é de quem a alcança, e quem não alcança é recusado ANTES de o fluxo abrir.
func TestTheSheetsLiveStreamIsTheOwners(t *testing.T) {
	f := newSceneFixture(t)
	stranger := seedUser(t, f.s, "estranho@t.com")
	rec := f.requests(t, stranger, http.MethodGet, fmt.Sprintf("/personagens/%d/ao-vivo", f.charID), "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("quem não é da mesa abriu o fluxo da ficha do jogador: %d", rec.Code)
	}
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", f.charID), "").Body.String()
	if !strings.Contains(page, fmt.Sprintf("/personagens/%d/ao-vivo", f.charID)) {
		t.Error("a ficha solta não abre o fluxo ao vivo")
	}
}
