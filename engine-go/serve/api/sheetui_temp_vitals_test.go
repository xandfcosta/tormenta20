package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/app"
	"t20engine/app/rest"
)

// OS PV E PM TEMPORÁRIOS À MÃO (decisão do dono): somar faz a poça da ficha crescer, e
// tirar drena da maior para a menor, como o gasto (p106).

func touchesTemp(t *testing.T, f sceneFixture, id int64, vital string, step int) string {
	t.Helper()
	return sheetPost(t, f, id, fmt.Sprintf("/vitais/%s/temporarios/%d", vital, step), "")
}

func TestAddingTemporaryPointsByHandFillsThePool(t *testing.T) {
	f, id := barbaro(t, 5)
	before := poolsOf(t, f.s, id)

	for _, step := range []int{4, 3} {
		if refusal := touchesTemp(t, f, id, "pv", step); refusal != "" {
			t.Fatalf("somar %d PV temporários foi recusado: %q", step, refusal)
		}
		if refusal := touchesTemp(t, f, id, "pm", step); refusal != "" {
			t.Fatalf("somar %d PM temporários foi recusado: %q", step, refusal)
		}
	}
	if got := tempHpOf(t, f, id); got != 7 {
		t.Errorf("a ficha tem %d PV temporários, e são 7", got)
	}
	if got := tempManaOf(t, f, id); got != 7 {
		t.Errorf("a ficha tem %d PM temporários, e são 7", got)
	}
	after := poolsOf(t, f.s, id)
	if after.HpCurrent != before.HpCurrent || after.MpCurrent != before.MpCurrent {
		t.Errorf("o temporário é parcela à parte (p106), e o poço mudou: %+v virou %+v", before, after)
	}
	var lines int
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM active_effects WHERE characterId = ?", id).Scan(&lines); err != nil {
		t.Fatalf("contar os efeitos: %v", err)
	}
	if lines != 2 {
		t.Errorf("somar à mão devia crescer uma poça por vital, e há %d linhas", lines)
	}
}

// A poça de outra fonte não é reescrita: quem soma à mão fica na própria linha.
func TestAddingByHandKeepsTheOtherPools(t *testing.T) {
	f, id := barbaro(t, 5)
	seedTempHpPool(t, f.s, id, 5)
	if refusal := touchesTemp(t, f, id, "pv", 2); refusal != "" {
		t.Fatalf("somar foi recusado: %q", refusal)
	}
	if got := tempHpOf(t, f, id); got != 7 {
		t.Errorf("as poças somam (p106), e a ficha tem %d de 7", got)
	}
}

func TestTakingTemporaryPointsDrainsTheLargestPoolFirst(t *testing.T) {
	f, id := barbaro(t, 5)
	seedTempHpPool(t, f.s, id, 5)
	touchesTemp(t, f, id, "pv", 2)
	before := poolsOf(t, f.s, id)

	if refusal := touchesTemp(t, f, id, "pv", -4); refusal != "" {
		t.Fatalf("tirar foi recusado: %q", refusal)
	}
	if got := tempHpOf(t, f, id); got != 3 {
		t.Errorf("a ficha ficou com %d PV temporários, e são 3", got)
	}
	var byHand string
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT modifiers FROM active_effects WHERE characterId = ? AND catalogId = 'manual.temp-hp'", id).Scan(&byHand); err != nil {
		t.Fatalf("ler a poça feita à mão: %v", err)
	}
	if !strings.Contains(byHand, `"amount":2`) {
		t.Errorf("a maior poça paga primeiro, e a de 2 mudou: %s", byHand)
	}
	if after := poolsOf(t, f.s, id); after.HpCurrent != before.HpCurrent {
		t.Errorf("tirar o temporário mexeu no PV: %d virou %d", before.HpCurrent, after.HpCurrent)
	}
}

// Como o toque no PV: o gesto prende no zero, e não sobra nada para o poço pagar.
func TestTakingMoreThanThereIsEmptiesThePoolOnly(t *testing.T) {
	f, id := barbaro(t, 5)
	touchesTemp(t, f, id, "pm", 2)
	before := poolsOf(t, f.s, id)
	if refusal := touchesTemp(t, f, id, "pm", -9); refusal != "" {
		t.Fatalf("tirar foi recusado: %q", refusal)
	}
	if got := tempManaOf(t, f, id); got != 0 {
		t.Errorf("sobraram %d PM temporários", got)
	}
	if after := poolsOf(t, f.s, id); after.MpCurrent != before.MpCurrent {
		t.Errorf("tirar o temporário gastou o poço: %d virou %d", before.MpCurrent, after.MpCurrent)
	}
}

// O temporário à mão vale até o fim do dia (p106): a cena acabar não o leva, e o descanso sim.
func TestTheHandmadePoolLastsUntilTheEndOfTheDay(t *testing.T) {
	f, id := barbaro(t, 5)
	touchesTemp(t, f, id, "pv", 4)
	var scope string
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT scope FROM active_effects WHERE characterId = ? AND catalogId = 'manual.temp-hp'", id).Scan(&scope); err != nil {
		t.Fatalf("ler a poça: %v", err)
	}
	if scope != "day" {
		t.Errorf("a poça dura %q, e o padrão do livro é o dia", scope)
	}
	screen := sheetTab(t, f, id, "conditionals")
	for _, want := range []string{"PV temporários · à mão", "até o fim do dia"} {
		if !strings.Contains(screen, want) {
			t.Errorf("a aba Efeitos não diz %q", want)
		}
	}
	if strings.Contains(screen, "manual.temp-hp") {
		t.Error("a aba Efeitos escreve a chave crua da poça")
	}
	scopes := rest.NewScopes(f.s.queries, f.s.catalogs)
	if err := scopes.EndScene(context.Background(), app.Caller{ID: f.player}, id); err != nil {
		t.Fatalf("encerrar a cena: %v", err)
	}
	if got := tempHpOf(t, f, id); got != 4 {
		t.Errorf("a cena acabou, e a poça do dia ficou com %d de 4", got)
	}
	if err := scopes.EndDay(context.Background(), app.Caller{ID: f.player}, id); err != nil {
		t.Fatalf("encerrar o dia: %v", err)
	}
	if got := tempHpOf(t, f, id); got != 0 {
		t.Errorf("o dia acabou, e sobraram %d PV temporários", got)
	}
}

func TestAStrangeVitalOrStepIsRefused(t *testing.T) {
	f, id := barbaro(t, 5)
	if refusal := sheetPost(t, f, id, "/vitais/xp/temporarios/2", ""); !strings.Contains(refusal, "xp") {
		t.Errorf("o vital que não existe devia ser recusado pelo nome, e veio %q", refusal)
	}
	if refusal := sheetPost(t, f, id, "/vitais/pv/temporarios/0", ""); refusal == "" {
		t.Error("o passo zero devia ser recusado")
	}
}

func TestOnlyTheOwnerTouchesTheTemporaryPoints(t *testing.T) {
	f, id := barbaro(t, 5)
	rec := f.requests(t, seedUser(t, f.s, "estranho@t.com"), http.MethodPost, fmt.Sprintf("/personagens/%d/vitais/pv/temporarios/2", id), "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("a ficha de outra pessoa respondeu %d", rec.Code)
	}
}

// O diálogo tem o campo de somar sempre, e o de tirar só quando há o que tirar.
func TestTheVitalDialogOffersTheTemporaryPoints(t *testing.T) {
	f, id := barbaro(t, 5)
	empty := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "").Body.String()
	if !strings.Contains(empty, "/vitais/pv/temporarios/") {
		t.Fatal("o diálogo do PV não oferece somar temporários")
	}
	if strings.Contains(empty, "Tirar") {
		t.Error("sem temporários, o diálogo oferece Tirar")
	}
	touchesTemp(t, f, id, "pv", 4)
	filled := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "").Body.String()
	if !strings.Contains(filled, "Tirar") {
		t.Error("com temporários, o diálogo não oferece Tirar")
	}
}

// "Dano" e "Gastar" são gasto, e os temporários pagam primeiro (p106). O que sobra sai do poço.
func TestTheDialogDamageAndSpendDrainThePoolFirst(t *testing.T) {
	for _, c := range []struct {
		name, vital string
		temp        func(*testing.T, sceneFixture, int64) int
		current     func(*testing.T, *Server, int64) int64
	}{
		{"PV", "pv", tempHpOf, func(t *testing.T, s *Server, id int64) int64 { return poolsOf(t, s, id).HpCurrent }},
		{"PM", "pm", tempManaOf, mpOf},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id := barbaro(t, 5)
			touchesTemp(t, f, id, c.vital, 5)
			before := c.current(t, f.s, id)
			if refusal := sheetPost(t, f, id, "/vitais/"+c.vital+"/gasto/3", ""); refusal != "" {
				t.Fatalf("gastar 3 foi recusado: %q", refusal)
			}
			if got, temp := c.current(t, f.s, id), c.temp(t, f, id); got != before || temp != 2 {
				t.Errorf("a poça paga primeiro: poço %d (era %d), poça %d de 2", got, before, temp)
			}
			if refusal := sheetPost(t, f, id, "/vitais/"+c.vital+"/gasto/4", ""); refusal != "" {
				t.Fatalf("gastar 4 foi recusado: %q", refusal)
			}
			if got, temp := c.current(t, f.s, id), c.temp(t, f, id); got != before-2 || temp != 0 {
				t.Errorf("o que sobra sai do poço: poço %d de %d, poça %d", got, before-2, temp)
			}
		})
	}
}

// O −1 e o +1 são correção, e a Cura e o Recuperar não mexem na poça.
func TestTheCorrectionsKeepThePool(t *testing.T) {
	f, id := barbaro(t, 5)
	touchesTemp(t, f, id, "pv", 5)
	before := poolsOf(t, f.s, id).HpCurrent
	for _, step := range []string{"-1", "1"} {
		if refusal := sheetPost(t, f, id, "/vitais/pv/"+step, ""); refusal != "" {
			t.Fatalf("corrigir %s foi recusado: %q", step, refusal)
		}
	}
	if got, temp := poolsOf(t, f.s, id).HpCurrent, tempHpOf(t, f, id); got != before || temp != 5 {
		t.Errorf("a correção mexeu na poça: PV %d de %d, poça %d de 5", got, before, temp)
	}
}

func TestTheDialogOffersTheSpendGesture(t *testing.T) {
	f, id := barbaro(t, 5)
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "").Body.String()
	for _, want := range []string{"/vitais/pv/gasto/", "/vitais/pm/gasto/"} {
		if !strings.Contains(page, want) {
			t.Errorf("o diálogo não usa %s", want)
		}
	}
}

func TestAStrangeSpendIsRefused(t *testing.T) {
	f, id := barbaro(t, 5)
	before := poolsOf(t, f.s, id)
	for _, path := range []string{"/vitais/xp/gasto/2", "/vitais/pv/gasto/0", "/vitais/pv/gasto/-3"} {
		if refusal := sheetPost(t, f, id, path, ""); refusal == "" {
			t.Errorf("%s devia ser recusado", path)
		}
	}
	if after := poolsOf(t, f.s, id); after.HpCurrent != before.HpCurrent {
		t.Errorf("o gasto recusado mexeu no PV: %d virou %d", before.HpCurrent, after.HpCurrent)
	}
}
