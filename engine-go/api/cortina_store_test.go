package api

import (
	"context"
	"testing"

	"t20engine/aovivo"
	"t20engine/tabuleiro"
)

// A cortina precisa SOBREVIVER ao reinício (ALE-202): o mestre fecha a cortina,
// monta a emboscada, fecha o laptop. Se a cortina mora só na memória, a sessão
// volta com o tabuleiro aberto e a mesa vê a emboscada montada — o modo de falha
// mais caro desta issue, porque acontece em silêncio e o mestre acha que está
// escondido.
func TestACortinaVoltaDoBanco(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))

	s.boards.Open(ctx, sid, "Taverna do Javali", "taverna")
	if _, _, err := s.boards.SetCurtain(ctx, sid, true); err != nil {
		t.Fatalf("fechar a cortina: %v", err)
	}
	s.boards.Persist(ctx, sid)

	// Um servidor novo sobre o MESMO banco: é o reinício, sem fingir.
	frio := tabuleiro.NewBoardStore(s.queries, aovivo.NewUUID)

	if voltou := frio.Get(ctx, sid); voltou == nil || !voltou.Curtained {
		t.Fatalf("a cortina não voltou do banco e a mesa veria a cena: %+v", voltou)
	}
}

// Fechar a cortina TEM de avançar a versão, e isto não é contabilidade: o
// `publishBoardState` usa `Version` como ordem do quadro e o hub descarta quadro
// cuja ordem não avançou (ALE-238 #1). Sem o bump, o mestre fecha a cortina, o
// servidor publica, o hub joga fora, e a mesa continua vendo o mapa — sem erro e
// sem log.
func TestFecharACortinaAvancaAVersaoDoQuadro(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))

	antes := s.boards.Open(ctx, sid, "Taverna do Javali", "taverna").Version

	fechada, mudou, err := s.boards.SetCurtain(ctx, sid, true)
	if err != nil {
		t.Fatalf("fechar a cortina: %v", err)
	}
	if !mudou {
		t.Error("fechar cortina aberta não foi reconhecido como mudança")
	}
	if fechada.Version <= antes {
		t.Errorf("a versão não avançou (%d → %d): o hub descartaria este quadro", antes, fechada.Version)
	}

	// Fechar o que já está fechado não é erro nem mutação — dois cliques no
	// telefone do mestre, ou duas abas — e publicar por não-mudança acordaria a
	// mesa inteira à toa.
	denovo, mudou, err := s.boards.SetCurtain(ctx, sid, true)
	if err != nil {
		t.Fatalf("fechar de novo: %v", err)
	}
	if mudou || denovo.Version != fechada.Version {
		t.Errorf("fechar cortina fechada mexeu no estado: mudou=%v versão %d → %d",
			mudou, fechada.Version, denovo.Version)
	}
}
