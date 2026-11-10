package api

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
	"testing"
)

// A BOLA DE FOGO DA FLAMEJANTE (p335): "Uma vez por rodada, quando ataca, você pode gastar 2 PM. Se fizer
// isso, em vez do ataque normal você dispara uma bola de fogo contra um alvo em alcance médio. O alvo sofre
// 6d6 pontos de dano. Um teste de Reflexos (CD For ou Des, à sua escolha) reduz à metade."
//
// Ela é um gesto próprio da superfície Ações, como conjurar (decisão do dono): cobra 2 PM e a ação padrão do
// ataque, uma vez por rodada, e põe na mesa os 6d6 rolados — no app, ou o total que a mesa rolou. Quem aplica
// o dano e a resistência do alvo é o mestre. O que o encanto dá é do motor (`enchant_boost_rules_test.go`);
// aqui se prende a LIGAÇÃO, da linha até a faixa.
func TestTheFlamingBlastGoesFromTheActionsToTheTable(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	sword := itemSemeia(t, f, barbaro, "espada-longa", "Espada longa", "wielded")
	if err := f.s.characterPlays().EnchantItem(context.Background(), sword, []string{"encanto-flamejante"}, nil); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}

	// A LINHA diz o que a bola de fogo faz e o que custa, com a CD da ficha: nível 6 e Força 4, a maior entre
	// Força e Destreza, dão 10 + 3 + 4 = 17 (p227).
	standard := entre(t, superficieDeAcoes(t, html.UnescapeString(acoesDaMesa(t, f))), "Ação padrão", "Ação de movimento")
	for _, want := range []string{"Bola de fogo", "6d6 de fogo", "alcance médio", "Reflexos CD 17 reduz à metade", "2 PM · uma vez por rodada"} {
		if !strings.Contains(standard, want) {
			t.Errorf("a linha da bola de fogo na ação padrão não diz %q:\n%s", want, recorte(standard))
		}
	}

	fire := func(body string) string {
		return sceneRefusal(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/bola-de-fogo/%d?embutida=1", barbaro, sword), body))
	}
	before := poolsOf(t, f.s, barbaro)
	// O TOTAL DA MESA fora do que 6d6 tira é recusado, e a recusa não cobra nada.
	if refusal := fire(fmt.Sprintf(`{"turn_target":%q,"blast_total":40}`, goblin)); !strings.Contains(refusal, "6 a 36") {
		t.Errorf("o total 40 em 6d6 foi aceito, ou a recusa não diz o intervalo: %q", refusal)
	}
	if refusal := fire(fmt.Sprintf(`{"turn_target":%q,"blast_total":21}`, goblin)); refusal != "" {
		t.Fatalf("a bola de fogo com o total 21 da mesa foi recusada: %q", refusal)
	}
	state := stateOf(t, f.s.sessions, f.sessionID)
	blast := state.LastBlast
	if blast == nil || blast.Total != 21 || !blast.ByHand || blast.TargetEntryID != goblin || blast.DC != 17 {
		t.Fatalf("a mesa guardou %+v; quer os 21 da mesa contra o Goblin, com a CD 17", blast)
	}
	if after := poolsOf(t, f.s, barbaro); after.MpCurrent != before.MpCurrent-2 {
		t.Errorf("a bola de fogo deixou %d PM, want %d − 2", after.MpCurrent, before.MpCurrent)
	}
	if state.Scene.StandardLeft {
		t.Error("a bola de fogo não gastou a ação padrão, e ela é em vez do ataque")
	}

	// A FAIXA DA MESA diz a rolagem e a resistência, para o mestre aplicar e para o jogador ler.
	for role, user := range map[string]int64{"mestre": f.gm, "jogador": f.player} {
		page := f.requests(t, user, http.MethodGet, f.tableUrl(), "").Body.String()
		band := entre(t, page, `id="table-blast"`, "</section>")
		for _, want := range []string{"Bola de fogo", "Goblin", "21", "de fogo", "Reflexos CD 17 reduz à metade", "da mesa"} {
			if !strings.Contains(band, want) {
				t.Errorf("a faixa do %s não diz %q:\n%s", role, want, band)
			}
		}
	}

	// UMA VEZ POR RODADA, com a iniciativa correndo: a vez seguinte da mesma rodada é recusada pela regra
	// do encanto, e não por falta de ação — ela vem antes.
	if refusal := fire(fmt.Sprintf(`{"turn_target":%q}`, goblin)); !strings.Contains(refusal, "uma vez por rodada") {
		t.Errorf("a segunda bola de fogo da rodada foi aceita, ou a recusa não diz a regra: %q", refusal)
	}
}
