package api

import (
	"context"
	"strings"
	"testing"

	"t20engine/app/combat"
	"t20engine/domain/live"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// A NATUREZA DE QUEM ESTÁ NA FILA se lê num lugar só (ver GLOSSARY, **tipo de efeito**): o tipo e a Inteligência,
// pelo verbete do bestiário ou pelo bloco do mestre. Nula não é zero: o Zumbi não tem mente (o travessão do livro),
// a Aparição tem Int 0, e o bloco guarda sempre um número.
func TestTheRosterReadsTheNatureOfTheLine(t *testing.T) {
	f := newSceneFixture(t)
	ctx := context.Background()
	roster := combat.NewRoster(f.s.queries, f.s.catalogs)

	for _, c := range []struct {
		monster  string
		nullMind bool
	}{{"zumbi", true}, {"aparicao", false}} {
		monster := c.monster
		found, err := roster.Of(ctx, f.campaignID, live.InitiativeEntry{ID: monster, Label: monster, MonsterID: &monster})
		if err != nil {
			t.Fatalf("ler o verbete %s: %v", monster, err)
		}
		if found.Nature.Kind != "morto-vivo" || (found.Nature.Intelligence == nil) != c.nullMind {
			t.Errorf("%s tem a natureza %+v; want morto-vivo, com mente nula: %v", monster, found.Nature, c.nullMind)
		}
	}

	block := undeadBlock(t, f)
	line := linePointingAtBlock(t, f.s, f.sessionID, block)
	found, err := roster.Of(ctx, f.campaignID, line)
	if err != nil {
		t.Fatalf("ler o bloco: %v", err)
	}
	if found.Nature.Kind != "morto-vivo" || found.Nature.Intelligence == nil || *found.Nature.Intelligence != 0 {
		t.Errorf("o bloco tem a natureza %+v; want morto-vivo com Int 0, que o bloco guarda como número", found.Nature)
	}
}

// O BLOCO DO MESTRE responde à imunidade como o verbete: morto-vivo, ele recusa veneno pelo TIPO; com Int 0 (o bloco
// guarda número, nunca o travessão), ele sente medo.
func TestTheMastersUndeadBlockRefusesPoisonAndFeelsFear(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	line := linePointingAtBlock(t, f.s, f.sessionID, undeadBlock(t, f))
	if refusal := marcaCondicao(t, f, line.ID, "envenenado"); !strings.Contains(refusal, "veneno") {
		t.Errorf("envenenar o morto-vivo do bloco não foi recusado pelo tipo: %q", refusal)
	}
	if refusal := marcaCondicao(t, f, line.ID, "apavorado"); refusal != "" {
		t.Errorf("apavorar o morto-vivo do bloco, que tem Int 0, foi recusado: %q", refusal)
	}
}

// undeadBlock semeia na campanha da bancada um bloco de morto-vivo com Inteligência 0.
func undeadBlock(t *testing.T, f sceneFixture) int64 {
	t.Helper()
	row, err := f.s.queries.CreateCampaignCreature(context.Background(), sqlcgen.CreateCampaignCreatureParams{
		Campaignid: f.campaignID, Name: "Carniçal do Mestre",
		Block:     `{"name":"Carniçal do Mestre","tipo":"morto-vivo","inteligencia":0,"defesa":14,"hp":30}`,
		Createdat: dbvalue.NowISO(), Updatedat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o bloco: %v", err)
	}
	return row.ID
}
