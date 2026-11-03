package api

import (
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// O anel vestido soma na ficha, e a tela avisa o que o livro pede: "somente após um
// dia de uso" (p342). O prazo é do mestre, porque o app não conta dias de uso.
func TestAVestedRingShowsItsBonusAndTheDayOfUseNotice(t *testing.T) {
	f, id := arcanistaDeNivel(t, 3)
	maxBefore := poolsOf(t, f.s, id).HpMax
	itemSemeia(t, f, id, "anel-da-vitalidade", "Anel da vitalidade", "vested")

	if got := poolsOf(t, f.s, id).HpMax; got != maxBefore+10 {
		t.Errorf("com o anel o máximo é %d, want %d", got, maxBefore+10)
	}
	const notice = "somente após um dia de uso"
	combat := html.UnescapeString(f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=combat", id), "").Body.String())
	account := regexp.MustCompile(`(?s)<dialog[^>]*id="vital-pv-levels".*?</dialog>`).FindString(combat)
	if !strings.Contains(account, "Anel da vitalidade") || !strings.Contains(account, notice) {
		t.Errorf("a conta de PV não mostra o anel com o aviso:\n%s", account)
	}
	bag := html.UnescapeString(f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=bag", id), "").Body.String())
	if !strings.Contains(bag, notice) {
		t.Error("a Mochila não avisa o dia de uso do anel")
	}
}
