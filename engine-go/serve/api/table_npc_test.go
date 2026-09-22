package api

import (
	"encoding/json"
	"strconv"
	"strings"
	"t20engine/domain/creature"
	"t20engine/infra/db/sqlcgen"
	"testing"
)

func TestStoringTheEntryCreatesTheGmBlock(t *testing.T) {
	f := newSceneFixture(t)

	f.posta(t, f.gm, f.tableUrl()+"/elenco/npc/do-verbete",
		`{"creature":"ogro","npc_name":"Ogro Capitão"}`)

	npcs := f.dbCast(t)
	if len(npcs) != 1 {
		t.Fatalf("o elenco tem %d NPCs, queria 1", len(npcs))
	}
	if npcs[0].Name != "Ogro Capitão" {
		t.Errorf("o nome guardado é %q", npcs[0].Name)
	}
	var block creature.Block
	if err := json.Unmarshal([]byte(npcs[0].Block), &block); err != nil {
		t.Fatalf("o bloco guardado está ilegível: %v", err)
	}
	if block.HP <= 0 || block.Defense <= 0 {
		t.Errorf("o bloco nasceu vazio: PV %d, Defesa %d", block.HP, block.Defense)
	}
	// A ORIGEM fica gravada, e é ela que deixa a tela dizer "cópia de ogro"
	// depois de o mestre renomear. Sem ela, "Ogro Capitão" perde o fio até o
	// livro no instante em que ganha nome próprio.
	if block.SourceMonsterID != "ogro" {
		t.Errorf("a origem não foi guardada: %q", block.SourceMonsterID)
	}
}

// Guardar "Ogro" como "Ogro" é o caso comum, e obrigar a digitar faria o mestre
// repetir o que a tela já mostra.
func TestAnEmptyNameFallsBackToTheBookName(t *testing.T) {
	f := newSceneFixture(t)

	f.posta(t, f.gm, f.tableUrl()+"/elenco/npc/do-verbete", `{"creature":"ogro","npc_name":"   "}`)

	npcs := f.dbCast(t)
	if len(npcs) != 1 || npcs[0].Name == "" {
		t.Fatalf("o NPC não nasceu com o nome do livro: %+v", npcs)
	}
	if strings.TrimSpace(npcs[0].Name) != npcs[0].Name {
		t.Errorf("o nome guardado veio com espaços: %q", npcs[0].Name)
	}
}

// "Os NPCs voltam semana que vem" só é verdade se eles não morrerem com a
// sessão. Guardado numa sessão, o NPC tem de aparecer na OUTRA da mesma
// campanha — e um guarda que olhasse só a sessão de origem passaria verde sobre
// um elenco que se perde toda noite.
func TestTheCastBelongsToTheCampaignAndNotToTheSession(t *testing.T) {
	f := newSceneFixture(t)
	otherSession := seedSession(t, f.s, f.campaignID)

	f.posta(t, f.gm, f.tableUrl()+"/elenco/npc/do-verbete", `{"creature":"ogro"}`)

	// A view da OUTRA sessão da mesma campanha tem de enxergar o mesmo NPC.
	view, _, err := f.s.tableScene.LoadView(t.Context(), f.gm, f.campaignID, otherSession)
	if err != nil {
		t.Fatalf("montar a view da outra sessão: %v", err)
	}
	if len(view.NPCs) != 1 {
		t.Errorf("a outra sessão da mesma campanha vê %d NPCs, queria 1", len(view.NPCs))
	}
}

// O id vem do CAMINHO, e o elenco guarda a PREPARAÇÃO da campanha, que é o material mais privado que o
// mestre tem: o chefe da semana que vem está ali. Alcançar o de outra mesa é
// pior que ver a fila dela.
func TestTheGmDoesNotReachAnotherCampaignsCast(t *testing.T) {
	f := newSceneFixture(t)
	otherCampaign := seedCampaign(t, f.s, f.player)
	now := "2026-01-01T00:00:00.000Z"
	foreign, err := f.s.queries.CreateCampaignCreature(t.Context(), sqlcgen.CreateCampaignCreatureParams{
		Campaignid: otherCampaign, Name: "Segredo alheio", Block: `{"nd":1,"tipo":"humanoide","size":"medio","hp":10}`,
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		t.Fatalf("semear o NPC alheio: %v", err)
	}

	body := f.posta(t, f.gm,
		f.tableUrl()+"/elenco/npc/"+strconv.FormatInt(foreign.ID, 10)+"/apagar", "{}")

	if !strings.Contains(body, "não é desta campanha") {
		t.Errorf("a recusa não veio: %s", firstRows(body, 5))
	}
	// O CONTROLE: recusar DEPOIS de apagar seria pior que não recusar.
	if _, err := f.s.queries.GetCampaignCreature(t.Context(), foreign.ID); err != nil {
		t.Error("o NPC da outra campanha foi apagado apesar da recusa")
	}
}

// A outra separação: elenco não é fila.
//
// Apagar o NPC do elenco e tirar a linha do combate respondem a duas perguntas
// — "ele não volta mais" e "ele saiu desta cena". Juntá-las faria o mestre
// perder o combatente EM CURSO ao arrumar a preparação, no meio da noite.
func TestDeletingFromTheCastDoesNotRemoveFromTheTracker(t *testing.T) {
	f := newSceneFixture(t)
	f.posta(t, f.gm, f.tableUrl()+"/elenco/npc/do-verbete", `{"creature":"ogro"}`)
	npcs := f.dbCast(t)
	if len(npcs) != 1 {
		t.Fatalf("o NPC não foi guardado")
	}
	route := f.tableUrl() + "/elenco/npc/" + strconv.FormatInt(npcs[0].ID, 10)
	f.posta(t, f.gm, route+"/na-fila", "{}")
	if n := len(stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative); n != 1 {
		t.Fatalf("o NPC não entrou na fila (%d linhas) — o resto do teste mediria nada", n)
	}

	f.posta(t, f.gm, route+"/apagar", "{}")

	if n := len(stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative); n != 1 {
		t.Errorf("apagar do elenco tirou o combatente da cena: a fila tem %d linhas", n)
	}
}

// O papel, no servidor.
func TestThePlayerDoesNotTouchTheCampaignCast(t *testing.T) {
	f := newSceneFixture(t)

	rec := f.pede(t, f.player, "POST", f.tableUrl()+"/elenco/npc/do-verbete", `{"creature":"ogro"}`)

	if rec.Code != 403 {
		t.Errorf("o jogador guardou NPC no elenco do mestre: %d", rec.Code)
	}
	if npcs := f.dbCast(t); len(npcs) != 0 {
		t.Error("a recusa veio depois da escrita")
	}
}

func (f sceneFixture) dbCast(t *testing.T) []sqlcgen.CampaignCreature {
	t.Helper()
	rows, err := f.s.queries.ListCampaignCreatures(t.Context(), f.campaignID)
	if err != nil {
		t.Fatalf("ler o elenco: %v", err)
	}
	return rows
}
