package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// npcAtND é o rascunho de um NPC no ND pedido, com Defesa, PV e um ataque escritos à mão — o que o "Preencher pelo
// ND" tem de sobrescrever.
func npcAtND(nd string) string {
	return bodyDraft(`"id":0,"nome":"Capanga","conjura":false,"bloco":{"nd":` + nd + `,"tipo":"humanoide","size":"medio",` +
		`"hp":1,"defesa":99,"deslocamento":"9m (6q)","forca":3,` +
		`"attacks":[{"name":"Clava","attackBonus":2,"damage":"1d6","special":""}],"skills":[],"specialAbilities":[]}`)
}

// PREENCHER PELO ND (ver GLOSSARY, **estatísticas pelo ND**): o rascunho volta com a Defesa, os PV e os ataques da
// Tabela 7-2 (p323), sobrescritos, e com o GUIA do que o livro deixa ao mestre — as perícias focadas e as outras, a
// CD e o pacote de atributos do patamar (p322). Como o acrescentar um ataque, o gesto não grava nada.
func TestFillingTheNPCByTheNDWritesTheDraftAndSavesNothing(t *testing.T) {
	f := newSceneFixture(t)
	response := f.posts(t, f.gm, f.tableUrl()+"/elenco/npc/rascunho/preencher-pelo-nd", npcAtND("5"))

	if cast := f.s.tableScene.CampaignCast(context.Background(), f.campaignID); len(cast) != 0 {
		t.Errorf("preencher pelo ND GRAVOU no elenco: %+v", cast)
	}
	block, _ := responseDraft(t, response)["bloco"].(map[string]any)
	if block["defesa"] != float64(28) || block["hp"] != float64(150) {
		t.Errorf("o ND 5 voltou com Defesa %v e PV %v; want 28 e 150", block["defesa"], block["hp"])
	}
	attacks, _ := block["attacks"].([]any)
	if len(attacks) != 2 {
		t.Fatalf("o ND 5 faz 2 ataques por rodada, e o rascunho voltou com %v", block["attacks"])
	}
	for _, a := range attacks {
		attack, _ := a.(map[string]any)
		if attack["attackBonus"] != float64(18) || attack["damage"] != "1d12+11" {
			t.Errorf("o ataque do ND 5 voltou %v; want +18 (1d12+11)", attack)
		}
	}
	if block["forca"] != float64(3) {
		t.Errorf("preencher mexeu na Força, que é do mestre: %v", block["forca"])
	}
	// A LINHA de ataque redesenhada volta como HTML, como no acrescentar: sem ela os dois ataques seriam sinal sem
	// campo na tela.
	if !strings.Contains(response, "draft.bloco.attacks.1.name") {
		t.Errorf("a segunda linha de ataque não foi desenhada:\n%.600s", response)
	}
	for _, want := range []string{"Tabela 7-2", "ND 5", "Veterano", "+14 nas focadas", "+5 nas outras", "CD 20", "4, 3, 2, 1, 0, 0"} {
		if !strings.Contains(response, want) {
			t.Errorf("o guia não diz %q:\n%.800s", want, response)
		}
	}
}

// FORA DA TABELA não há número: o ND ¼ do Bandido e o 21 não têm linha na Tabela 7-2, e a recusa diz por quê,
// dentro do editor (decisão do dono).
func TestFillingOutsideTheTableIsRefusedWithTheReason(t *testing.T) {
	f := newSceneFixture(t)
	for _, nd := range []string{"0.25", "21"} {
		response := f.posts(t, f.gm, f.tableUrl()+"/elenco/npc/rascunho/preencher-pelo-nd", npcAtND(nd))
		if !strings.Contains(response, "draft_error") || !strings.Contains(response, "Tabela 7-2") || !strings.Contains(response, "½ ao 20") {
			t.Errorf("o ND %s não foi recusado dizendo que a Tabela 7-2 vai do ND ½ ao 20:\n%.500s", nd, response)
		}
	}
}

// A trava é do SERVIDOR, como no resto do elenco.
func TestOnlyTheGmFillsTheNPCByTheND(t *testing.T) {
	f := newSceneFixture(t)
	rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/elenco/npc/rascunho/preencher-pelo-nd", npcAtND("5"))
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador preencheu o rascunho do mestre: %d", rec.Code)
	}
}

// O BOTÃO mora no editor, ao lado do ND, e fica apagado fora da tabela pelo mesmo motivo que o servidor recusa.
func TestTheEditorOffersFillingByTheND(t *testing.T) {
	f := newSceneFixture(t)
	// O EDITOR mora na página do Elenco, e não na mesa.
	page := f.requests(t, f.gm, http.MethodGet, f.tableUrl()+"/elenco", "").Body.String()
	if !strings.Contains(page, "Nome do NPC") {
		t.Fatalf("o controle falhou: a página do Elenco não desenhou o editor do bloco:\n%.400s", page)
	}
	if !strings.Contains(page, "Preencher pelo ND") || !strings.Contains(page, "/elenco/npc/rascunho/preencher-pelo-nd") {
		t.Error("o editor do bloco não oferece o Preencher pelo ND")
	}
	if !strings.Contains(page, "[0.5,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20].includes") {
		t.Error("o botão não se apaga fora dos NDs da Tabela 7-2")
	}
}
