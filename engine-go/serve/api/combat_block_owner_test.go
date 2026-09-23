package api

import (
	"context"
	"errors"
	"testing"

	"t20engine/app"
	"t20engine/app/combat"
	"t20engine/domain/live"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O `CreatureID` de uma linha da fila vem do CLIENTE, e o `initiative.npcEntry`
// grava o que mandarem — de propósito: um id desconhecido vira "sem bloco" na
// tela em vez de derrubar a adição no meio do combate. O comentário lá diz, com
// todas as letras, que quem confere o dono do bloco é a rota que o SERVE.
//
// O `combat.Roster` é uma rota que serve, e não conferia. MEDIDO antes do
// conserto: uma linha apontando o bloco de outra campanha devolvia a Defesa
// dela. Um inteiro só, mas atravessando a fronteira entre duas mesas que não se
// conhecem, e alcançável pelo gesto de atacar (ALE-377).
func TestTheCombatRefusesACreatureBlockFromAnotherCampaign(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	roster := combat.NewRoster(s.queries, s.catalogs)

	strangerCampaign := seedCampaign(t, s, seedUser(t, s, "vizinho@t.com"))
	block, err := s.queries.CreateCampaignCreature(ctx, sqlcgen.CreateCampaignCreatureParams{
		Campaignid: strangerCampaign, Name: "Segredo do Vizinho",
		Block:     `{"name":"Segredo do Vizinho","defesa":77,"hp":30}`,
		Createdat: dbvalue.NowISO(), Updatedat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o bloco alheio: %v", err)
	}

	mine := seedCampaign(t, s, seedUser(t, s, "mestre@t.com"))
	line := linePointingAtBlock(t, s, seedSession(t, s, mine), block.ID)

	if _, err := roster.Of(ctx, mine, line); err == nil {
		t.Error("o combate leu o bloco de OUTRA campanha")
	} else if !errors.Is(err, app.ErrForbidden) {
		t.Errorf("a recusa tem de ser de PERMISSÃO e veio: %v", err)
	}

	// O CONTROLE. Sem ele, "recusou" não distingue a fronteira de um caminho que
	// nunca leu bloco nenhum — e o 77 é o mesmo número que vazou, lido agora por
	// quem tem direito a ele.
	own := linePointingAtBlock(t, s, seedSession(t, s, strangerCampaign), block.ID)
	found, err := roster.Of(ctx, strangerCampaign, own)
	if err != nil {
		t.Fatalf("o controle falhou: a campanha DONA não leu o próprio bloco: %v", err)
	}
	if found.Defense != 77 {
		t.Errorf("o controle falhou: a Defesa veio %d e o bloco diz 77", found.Defense)
	}
}

// linePointingAtBlock põe na fila da sessão uma linha de NPC apontando o bloco, e
// devolve a linha como o estado a guardou — com o id que o store escolheu.
func linePointingAtBlock(t *testing.T, s *Server, sessionID, blockID int64) live.InitiativeEntry {
	t.Helper()
	state, err := s.sessions.AddInitiativeEntry(context.Background(), sessionID, live.InitiativeEntry{
		Label: "Alvo", Initiative: 10, Type: "npc", CreatureID: &blockID,
	})
	if err != nil {
		t.Fatalf("pôr a linha na fila da sessão %d: %v", sessionID, err)
	}
	return state.Initiative[0]
}
