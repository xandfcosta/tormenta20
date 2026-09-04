package api

import (
	"context"
	"testing"

	"t20engine/tabuleiro"
)

/*
O TABULEIRO DE UMA SESSÃO QUE JÁ MORREU (ALE-270).

O tabuleiro vive em MEMÓRIA num mapa por sessão, e o `Persist` grava o blob em
`open_boards`, cuja chave estrangeira aponta para a sessão. Quando a sessão é
APAGADA, o mapa em memória continua lá — e a gravação seguinte bate na FK.

O estrago não é a linha de log. É o `Dirty`: ele existe desde a ALE-154 para a
mesa SABER quando parou de gravar, e só um `Persist` bem-sucedido o apaga.
Nenhum vai suceder, porque a sessão não volta a existir. **Um alarme construído
para gritar "PARE, não estou gravando" passa a gritar por um tabuleiro que
ninguém quer que seja gravado** — e um alarme que toca sozinho é como se aprende
a ignorar o alarme.
*/

// deadSessionBoard abre um tabuleiro numa sessão e apaga a sessão por baixo dele.
func deadSessionBoard(t *testing.T) (*Server, int64, int64) {
	t.Helper()
	s := newTestServer(t)
	campanha := seedCampaign(t, s, seedUser(t, s, "gm@t.com"))
	sessao := seedSession(t, s, campanha)
	ctx := context.Background()

	if _, err := s.boards.Open(ctx, sessao, "Taverna do Javali", "taverna"); err != nil {
		t.Fatalf("abrir o tabuleiro: %v", err)
	}
	// A GRAVAÇÃO tem de dar certo ANTES, senão o caso mede uma sessão que nunca
	// gravou e a conclusão seria sobre outra coisa.
	if dirty, _ := s.boards.Persist(ctx, sessao, defaultTab); dirty {
		t.Fatal("a gravação já falhava com a sessão VIVA — o caso mediria outro defeito")
	}
	return s, campanha, sessao
}

// A sessão apagada deixa de ter tabuleiro em memória, e a mesa não se declara
// suja por causa dela.
func TestADeletedSessionLeavesNoBoardBehind(t *testing.T) {
	s, campanha, sessao := deadSessionBoard(t)
	ctx := context.Background()

	if err := s.queries.DeleteSession(ctx, sessao); err != nil {
		t.Fatalf("apagar a sessão: %v", err)
	}
	s.SessionDeleted(sessao)

	// O MAPA: o tabuleiro morreu com a sessão. Sem isto, todo `Persist` seguinte
	// bate na FK, para sempre.
	if abertos := s.boards.OpenBoards(ctx, sessao); len(abertos) != 0 {
		t.Errorf("a sessão apagada continuou com %d tabuleiros em memória", len(abertos))
	}
	// E A MARCA saiu: um `Dirty` que ninguém pode limpar é um alarme travado.
	if s.boards.SaveFailed(sessao) {
		t.Error("a mesa continuou se declarando suja por uma sessão que não existe")
	}
	// CONTROLE do controle: a campanha segue de pé, e outra sessão dela grava
	// normalmente. Sem isto, um esquecimento que limpasse o store INTEIRO
	// passaria neste caso.
	outra := seedSession(t, s, campanha)
	if _, err := s.boards.Open(ctx, outra, "Cripta", "cripta"); err != nil {
		t.Fatalf("abrir tabuleiro na sessão vizinha: %v", err)
	}
	if dirty, _ := s.boards.Persist(ctx, outra, defaultTab); dirty {
		t.Error("a sessão vizinha deixou de gravar")
	}
}

// A CAMPANHA apagada leva os tabuleiros de TODAS as sessões dela.
//
// Caso próprio porque o caminho é outro: apagar a campanha derruba as sessões
// por CASCATA no banco, e nenhuma delas passa pelo caminho de apagar sessão.
func TestADeletedCampaignLeavesNoBoardBehind(t *testing.T) {
	s, campanha, sessao := deadSessionBoard(t)
	ctx := context.Background()
	segunda := seedSession(t, s, campanha)
	if _, err := s.boards.Open(ctx, segunda, "Cripta", "cripta"); err != nil {
		t.Fatalf("abrir o segundo tabuleiro: %v", err)
	}
	if dirty, _ := s.boards.Persist(ctx, segunda, defaultTab); dirty {
		t.Fatal("a segunda sessão já não gravava")
	}

	s.CampaignDeleted(ctx, campanha)
	if err := s.queries.DeleteCampaign(ctx, campanha); err != nil {
		t.Fatalf("apagar a campanha: %v", err)
	}

	for _, morta := range []int64{sessao, segunda} {
		if abertos := s.boards.OpenBoards(ctx, morta); len(abertos) != 0 {
			t.Errorf("a sessão %d da campanha apagada ficou com %d tabuleiros", morta, len(abertos))
		}
		if s.boards.SaveFailed(morta) {
			t.Errorf("a sessão %d da campanha apagada ficou marcada como suja", morta)
		}
	}
}

// A LIMPEZA NÃO DEPENDE DE O CLIENTE ESPERAR.
//
// O `Close` apaga a linha de `open_boards` com o contexto que recebeu, e na
// produção esse é o contexto da REQUISIÇÃO — cancelado quando quem clicou vai
// embora. A linha ficava no banco, o `Dirty` acendia, e não havia quem tentasse
// de novo: `board delete failed (context canceled)` foi medido numa corrida de
// e2e (ALE-270).
func TestClosingABoardSurvivesTheClientLeaving(t *testing.T) {
	s, _, sessao := deadSessionBoard(t)
	cancelado, cancela := context.WithCancel(context.Background())
	cancela() // o cliente foi embora ANTES de a gravação acontecer

	dirty, _ := s.boards.Close(cancelado, sessao, defaultTab)

	if dirty {
		t.Error("fechar o tabuleiro com o cliente já embora declarou a mesa suja")
	}
	// E A LINHA SAIU do banco: sem isto, a próxima hidratação traz de volta um
	// tabuleiro que o mestre encerrou.
	depois := tabuleiro.NewBoardStore(s.queries, s.boards.NewID, s.bus)
	if abertos := depois.OpenBoards(context.Background(), sessao); len(abertos) != 0 {
		t.Errorf("a linha do tabuleiro fechado ficou no banco: %d aberto(s) depois do reinício", len(abertos))
	}
}

// O CONTROLE DA AUSÊNCIA, e a issue o exige com todas as letras: o log fica
// quieto tanto no conserto quanto num caminho que nunca chega ao `Persist`.
//
// Este caso prova que o canal EXISTE — com a sessão apagada e o tabuleiro ainda
// em memória (o estado de antes do conserto, montado à mão), a gravação falha e
// o `Dirty` acende. É a garantia de que os casos acima não passam por não terem
// chegado ao lugar onde o defeito morava.
func TestTheForeignKeyStillBitesWhenTheBoardOutlivesTheSession(t *testing.T) {
	s, _, sessao := deadSessionBoard(t)
	ctx := context.Background()

	// Apaga a sessão SEM avisar os stores — que é exatamente o que a produção
	// fazia antes desta issue.
	if err := s.queries.DeleteSession(ctx, sessao); err != nil {
		t.Fatalf("apagar a sessão: %v", err)
	}

	dirty, _ := s.boards.Persist(ctx, sessao, defaultTab)

	if !dirty {
		t.Fatal("a gravação de um tabuleiro órfão passou — o canal não existe, e os outros casos não provam nada")
	}
	if !s.boards.SaveFailed(sessao) {
		t.Error("a marca de suja não acendeu")
	}
	// E ELA NÃO SAI SOZINHA: é o que torna o alarme travado, e não um susto.
	if dirtyDeNovo, _ := s.boards.Persist(ctx, sessao, defaultTab); !dirtyDeNovo {
		t.Error("a segunda gravação passou: o defeito não é o alarme travado que a issue descreve")
	}
}
