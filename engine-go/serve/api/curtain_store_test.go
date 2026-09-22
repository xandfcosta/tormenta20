package api

import (
	"context"
	"t20engine/app/boards"
	"testing"

	"t20engine/domain/live"
	"t20engine/infra/events"
)

// A cortina precisa SOBREVIVER ao reinício (ALE-202): o mestre fecha a cortina,
// monta a emboscada, fecha o laptop. Se a cortina mora só na memória, a sessão
// volta com o tabuleiro aberto e a mesa vê a emboscada montada — o modo de falha
// mais caro desta issue, porque acontece em silêncio e o mestre acha que está
// escondido.
func TestTheCurtainComesBackFromTheDatabase(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))

	opensBoard(t, s, sid, "Taverna do Javali", "tavern")
	if _, _, err := s.boards.SetCurtain(ctx, sid, defaultTab, true); err != nil {
		t.Fatalf("fechar a cortina: %v", err)
	}

	// Um servidor novo sobre o MESMO banco: é o reinício, sem fingir.
	cold := boards.NewStore(boards.NewSnapshots(s.queries), s.queries, live.NewUUID, &events.Bus{})

	if returned := boardRead(cold.Get(ctx, sid, defaultTab)); returned == nil || !returned.Curtained {
		t.Fatalf("a cortina não voltou do banco e a mesa veria a cena: %+v", returned)
	}
}

// Fechar a cortina avança a versão, porque `Version` significa "o tabuleiro
// mudou" e toda mutação aceita a sobe. Quem lê esse número são o descarte de
// quadro atrasado do hub (ALE-238 #1) e o `commitMove`, que recusa confirmar um
// movimento proposto sobre um tabuleiro que mudou desde então.
//
// O que este teste NÃO prova, e eu escrevi errado antes de medir: que sem o bump
// a mesa não veria a cortina. O `EmitOrdered` descarta com `Seq < lastSeq` —
// estritamente menor —, então versão repetida passa. Tirei o bump, subi o
// servidor e o e2e de dois clientes seguiu verde. O guarda continua valendo pelo
// primeiro motivo; a consequência dramática é que era invenção minha.
func TestClosingTheCurtainAdvancesTheBoardVersion(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))

	before := opensBoard(t, s, sid, "Taverna do Javali", "tavern").Version

	closed, changed, err := s.boards.SetCurtain(ctx, sid, defaultTab, true)
	if err != nil {
		t.Fatalf("fechar a cortina: %v", err)
	}
	if !changed {
		t.Error("fechar cortina aberta não foi reconhecido como mudança")
	}
	if closed.Version <= before {
		t.Errorf("a versão não avançou (%d → %d): o contador deixou de dizer que o tabuleiro mudou", before, closed.Version)
	}

	// Fechar o que já está fechado não é erro nem mutação — dois cliques no
	// telefone do mestre, ou duas abas — e publicar por não-mudança acordaria a
	// mesa inteira à toa.
	again, changed, err := s.boards.SetCurtain(ctx, sid, defaultTab, true)
	if err != nil {
		t.Fatalf("fechar de novo: %v", err)
	}
	if changed || again.Version != closed.Version {
		t.Errorf("fechar cortina fechada mexeu no estado: mudou=%v versão %d → %d",
			changed, closed.Version, again.Version)
	}
}
