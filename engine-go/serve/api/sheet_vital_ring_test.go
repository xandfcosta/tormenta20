package api

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"t20engine/domain/sheet"
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

// Tirar o anel com pouca vida derruba (decisão do dono): o PV atual é o máximo menos o
// dano, e quem chega a 0 ou menos cai, por qualquer caminho (p236). Vestir de volta faz
// o movimento inverso.
func TestTakingOffARingWhileHurtKnocksTheCharacterDown(t *testing.T) {
	for _, c := range []struct{ name, path string }{
		{"guardando", "guarda"},
		{"tirando da ficha", "remover"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id := arcanistaDeNivel(t, 3)
			ring := itemSemeia(t, f, id, "anel-da-vitalidade", "Anel da vitalidade", "vested")
			arrangePools(t, f.s, id, func(pools sheet.Pools) (sheet.Pools, error) {
				pools.HpCurrent = 5
				return pools, nil
			})
			conditions := func() string {
				t.Helper()
				row, err := f.s.queries.GetCharacter(context.Background(), id)
				if err != nil {
					t.Fatalf("ler a ficha: %v", err)
				}
				return row.Activeconditions
			}
			if got := conditions(); strings.Contains(got, "inconsciente") {
				t.Fatalf("o controle falhou: a ficha já começou caída: %s", got)
			}

			address := fmt.Sprintf("/personagens/%d/itens/%d/%s?tab=bag", id, ring, c.path)
			if rec := f.requests(t, f.player, http.MethodPost, address, ""); rec.Code != http.StatusOK {
				t.Fatalf("o gesto respondeu %d", rec.Code)
			}

			if got := poolsOf(t, f.s, id).HpCurrent; got != -5 {
				t.Errorf("sem o anel a ficha tem %d PV, want -5", got)
			}
			got := conditions()
			if !strings.Contains(got, "inconsciente") || !strings.Contains(got, "sangrando") {
				t.Errorf("a ficha caiu sem as condições da p236: %s", got)
			}
		})
	}
}

func TestPuttingTheRingBackOnGetsTheCharacterUp(t *testing.T) {
	f, id := arcanistaDeNivel(t, 3)
	ring := itemSemeia(t, f, id, "anel-da-vitalidade", "Anel da vitalidade", "vested")
	arrangePools(t, f.s, id, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.HpCurrent = 5
		return pools, nil
	})
	base := fmt.Sprintf("/personagens/%d/itens/%d/", id, ring)
	f.requests(t, f.player, http.MethodPost, base+"guarda?tab=bag", "")
	if rec := f.requests(t, f.player, http.MethodPost, base+"equipa/vested?tab=bag", ""); rec.Code != http.StatusOK {
		t.Fatalf("vestir respondeu %d", rec.Code)
	}

	if got := poolsOf(t, f.s, id).HpCurrent; got != 5 {
		t.Errorf("de volta com o anel a ficha tem %d PV, want 5", got)
	}
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler a ficha: %v", err)
	}
	if strings.Contains(row.Activeconditions, "inconsciente") || strings.Contains(row.Activeconditions, "sangrando") {
		t.Errorf("de pé, a ficha continua com as condições de quem caiu: %s", row.Activeconditions)
	}
}
