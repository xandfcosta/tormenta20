package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// Entre o limite e o dobro a mochila aceita, porque há regra para quem passa (p141), e o
// aviso diz o que ela faz com ESTE personagem: o anão não perde deslocamento (p20), mas a
// penalidade de armadura vale para ele. Acima do dobro o livro proíbe, e ela recusa.
func TestOverloadTellsTheDwarfApartAndTheBagRefusesPastTheDouble(t *testing.T) {
	for _, c := range []struct {
		race, wants, refuses string
	}{
		{"anao", "deslocamento sem redução (Devagar e Sempre)", "-3m de deslocamento"},
		{"elfo", "-3m de deslocamento", "Devagar e Sempre"},
	} {
		t.Run(c.race, func(t *testing.T) {
			f := newSceneFixture(t)
			form := aFolhaPreenchida()
			form.Set("race", c.race)
			id := oIDDoDestino(t, postaAForja(t, f, f.player, "/personagens/nova", form).Header().Get("Location"))

			// Força 0: limite 10, dobro 20. O kit desta folha ocupa 8, então 6 a mais dão 14.
			if refusal := customItem(t, f, id, `{"item_name":"Barril","item_qty":1,"item_slots":6}`); refusal != "" {
				t.Fatalf("a mochila recusou passar do limite, e isso tem regra: %q", refusal)
			}
			page := fmt.Sprintf("/personagens/%s?tab=bag", strconv.FormatInt(id, 10))
			body := f.requests(t, f.player, http.MethodGet, page, "").Body.String()
			if !strings.Contains(body, "Sobrecarregado (p141): -5 em Acrobacia, Furtividade e Ladinagem") {
				t.Fatal("o aviso de sobrecarga não saiu com a penalidade de armadura: o caso não mede nada")
			}
			if !strings.Contains(body, c.wants) {
				t.Errorf("o aviso não diz %q", c.wants)
			}
			if strings.Contains(body, c.refuses) {
				t.Errorf("o aviso diz %q", c.refuses)
			}

			// 14 + 7 = 21 passa do dobro; a recusa diz o número e o teto.
			refusal := customItem(t, f, id, `{"item_name":"Bigorna","item_qty":1,"item_slots":7}`)
			if !strings.Contains(refusal, "21") || !strings.Contains(refusal, "20") {
				t.Errorf("passar do dobro não foi recusado com os números: %q", refusal)
			}
			// 14 + 6 = 20 é exatamente o dobro, e cabe.
			if refusal := customItem(t, f, id, `{"item_name":"Saco","item_qty":1,"item_slots":6}`); refusal != "" {
				t.Errorf("o dobro exato foi recusado: %q", refusal)
			}

			// EDITAR para crescer além do dobro também é recusado; para encolher, nunca.
			barrel := sheetNameItem(t, f, id, "Barril")
			edit := fmt.Sprintf("/personagens/%d/itens/%d/edita?tab=bag", id, barrel.ID)
			grown := f.requests(t, f.player, http.MethodPost, edit, `{"item_name":"Barril","item_qty":1,"item_slots":7}`)
			if refusal := bagRefusal(t, grown.Body.String()); refusal == "" {
				t.Error("editar o item para passar do dobro foi aceito")
			}
			shrunk := f.requests(t, f.player, http.MethodPost, edit, `{"item_name":"Barril","item_qty":1,"item_slots":1}`)
			if refusal := bagRefusal(t, shrunk.Body.String()); refusal != "" {
				t.Errorf("encolher o item foi recusado: %q", refusal)
			}
		})
	}
}
