package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// A marca de arma de categoria maior mora no item da Mochila, e fica guardada
// (decisão do dono). O livro não dá tamanho por item (p142), e a arma "ocupa um
// espaço a mais" (p106).
func oversizedAddress(f sceneFixture, item int64) string {
	return fmt.Sprintf("/personagens/%d/itens/%d/categoria-maior?tab=bag", f.charID, item)
}

func TestMarkingAWeaponAsOversizedKeepsTheMarkAndTheExtraSlot(t *testing.T) {
	f := newSceneFixture(t)
	item := itemSemeia(t, f, f.charID, "adaga", "Adaga", "wielded")
	slotsOf := func() float64 {
		t.Helper()
		row, err := f.s.queries.GetItem(context.Background(), item)
		if err != nil {
			t.Fatalf("ler o item: %v", err)
		}
		return row.Slots
	}
	before := slotsOf()

	if refusal := bagRefusal(t, f.requests(t, f.player, http.MethodPost, oversizedAddress(f, item), "").Body.String()); refusal != "" {
		t.Fatalf("marcar foi recusado: %q", refusal)
	}
	if got := slotsOf(); got != before+1 {
		t.Errorf("marcada, a arma ocupa %v espaços, want %v", got, before+1)
	}
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=bag", f.charID), "").Body.String()
	if !strings.Contains(page, "Arma de categoria maior") {
		t.Error("a folha do item não oferece a marca")
	}
	// A adaga é 1d4, e um passo acima é 1d6 (Tabela 3-2, p143). O crachá da mão descreve
	// o item, e o item é outro depois da marca.
	if !strings.Contains(page, "Dano 1d6") {
		t.Error("o crachá da Mochila não andou o dado da arma marcada")
	}
	combat := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=combat", f.charID), "").Body.String()
	if !strings.Contains(combat, "arma de categoria maior") {
		t.Error("a decomposição do ataque não diz de onde veio o -5")
	}

	// Desmarcar devolve o espaço: o gesto é o mesmo, e quem sabe o estado é o servidor.
	if refusal := bagRefusal(t, f.requests(t, f.player, http.MethodPost, oversizedAddress(f, item), "").Body.String()); refusal != "" {
		t.Fatalf("desmarcar foi recusado: %q", refusal)
	}
	if got := slotsOf(); got != before {
		t.Errorf("desmarcada, a arma ocupa %v espaços, want %v", got, before)
	}
}

func TestOnlyACatalogWeaponTakesTheOversizedMark(t *testing.T) {
	f := newSceneFixture(t)
	for _, c := range []struct{ name, catalog, label string }{
		{"armadura", "armadura-couro", "Armadura de couro"},
		{"item feito à mão", "", "Espada do avô"},
	} {
		t.Run(c.name, func(t *testing.T) {
			item := itemSemeia(t, f, f.charID, c.catalog, c.label, "")
			refusal := bagRefusal(t, f.requests(t, f.player, http.MethodPost, oversizedAddress(f, item), "").Body.String())
			if !strings.Contains(refusal, c.label) {
				t.Errorf("a recusa diz %q; devia nomear o item", refusal)
			}
		})
	}
}

// cloneForCampaignTest senta o herói numa mesa nova, pelo gesto de entrar, e devolve
// a cópia que nasceu lá.
func cloneForCampaignTest(t *testing.T, f sceneFixture) int64 {
	t.Helper()
	table := seedCampanha(t, f.s, f.player, "Mesa da cópia", "")
	if rec := postaCarta(t, f.s, f.player, url.Values{
		"campaignId":  {strconv.FormatInt(table, 10)},
		"characterId": {strconv.FormatInt(f.charID, 10)},
	}); rec.Code >= http.StatusBadRequest {
		t.Fatalf("entrar na mesa respondeu %d: %s", rec.Code, rec.Body.String())
	}
	var copied int64
	if err := f.s.db.QueryRowContext(context.Background(),
		`SELECT id FROM characters WHERE sourceCharacterId = ? AND campaignId = ?`, f.charID, table).Scan(&copied); err != nil {
		t.Fatalf("a cópia não nasceu na mesa: %v", err)
	}
	return copied
}

// "Marcou uma vez e já era": a marca entra na mesa com a cópia da ficha.
func TestTheCampaignCopyKeepsTheOversizedMark(t *testing.T) {
	f := newSceneFixture(t)
	itemSemeia(t, f, f.charID, "bandana", "Bandana", "vested")
	item := itemSemeia(t, f, f.charID, "adaga", "Adaga", "wielded")
	if refusal := bagRefusal(t, f.requests(t, f.player, http.MethodPost, oversizedAddress(f, item), "").Body.String()); refusal != "" {
		t.Fatalf("marcar foi recusado: %q", refusal)
	}

	copied := cloneForCampaignTest(t, f)

	var catalog string
	if err := f.s.db.QueryRowContext(context.Background(), `
		SELECT i.catalogId FROM character_oversized_weapons o
		JOIN character_items i ON i.id = o.itemId WHERE i.characterId = ?`, copied).Scan(&catalog); err != nil {
		t.Fatalf("a cópia não trouxe a marca: %v", err)
	}
	if catalog != "adaga" {
		t.Errorf("a marca foi parar em %q, want adaga", catalog)
	}
}
