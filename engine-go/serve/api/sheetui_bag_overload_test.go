package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// A mochila nunca impede adicionar: passar do limite tem regra (p141), e quem aplica é a
// ficha. E o aviso diz o que a regra faz com ESTE personagem: o anão não perde
// deslocamento (p20), mas a penalidade de armadura continua valendo para ele.
func TestOverloadNeverRefusesAndTheNoticeTellsTheDwarfApart(t *testing.T) {
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

			// Cem espaços passam do limite e do dobro dele, para qualquer Força daqui.
			if refusal := customItem(t, f, id, `{"item_name":"Barril","item_qty":1,"item_slots":100}`); refusal != "" {
				t.Fatalf("a mochila recusou passar do limite: %q", refusal)
			}
			if item := sheetNameItem(t, f, id, "Barril"); item.Slots != 100 {
				t.Fatalf("o item entrou com %v espaços, esperava 100", item.Slots)
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
		})
	}
}
