package api

import (
	"context"
	"testing"

	"t20engine/app"
	"t20engine/app/boards"
	"t20engine/domain/board"
)

/*
O TABULEIRO DE UMA SESSÃO QUE JÁ MORREU.

O tabuleiro vive em MEMÓRIA num mapa por sessão, e a gravação escreve em
`open_boards`, cuja chave estrangeira aponta para a sessão. Quando a sessão é
APAGADA, o mapa em memória continua lá — e a gravação seguinte bate na FK.

O estrago MUDOU DE FORMA com a ALE-375, e por isso estes casos continuam. Antes
ele era o `Dirty`: a mesa se declarava suja para sempre, porque só uma gravação
bem-sucedida apagava a marca e nenhuma ia suceder. Agora não há marca — há
RECUSA, e ela é pior de outro jeito: todo gesto no tabuleiro órfão passa a ser
recusado com a frase do banco, num mapa que ninguém quer que seja gravado.

Os dois estragos têm o mesmo conserto, e é ele que estes casos prendem: o
`SessionDeleted` tira o mapa da memória junto com a sessão.
*/

// deadSessionBoard abre um tabuleiro numa sessão e apaga a sessão por baixo dele.
func deadSessionBoard(t *testing.T) (*Server, int64, int64) {
	t.Helper()
	s, campaign, session, _ := deadSessionBoardOfOwner(t)
	return s, campaign, session
}

// deadSessionBoardOfOwner é o mesmo, dizendo QUEM é o dono — o caso que apaga a
// campanha precisa dele para chamar o caso de uso, que autoriza.
func deadSessionBoardOfOwner(t *testing.T) (*Server, int64, int64, int64) {
	t.Helper()
	s := newTestServer(t)
	owner := seedUser(t, s, "gm@t.com")
	campaign := seedCampaign(t, s, owner)
	session := seedSession(t, s, campaign)
	ctx := context.Background()

	if _, err := s.boards.Open(ctx, session, "Taverna do Javali", "tavern"); err != nil {
		t.Fatalf("abrir o tabuleiro: %v", err)
	}
	// A GRAVAÇÃO tem de dar certo ANTES, senão o caso mede uma sessão que nunca
	// gravou e a conclusão seria sobre outra coisa. Ela acontece DENTRO do
	// `Open` desde a ALE-375, então chegar aqui sem erro já é a prova.
	return s, campaign, session, owner
}

// A sessão apagada deixa de ter tabuleiro em memória, e a mesa não se declara
// suja por causa dela.
func TestADeletedSessionLeavesNoBoardBehind(t *testing.T) {
	s, campaign, session, owner := deadSessionBoardOfOwner(t)
	ctx := context.Background()

	// PELO CASO DE USO, como o caso da campanha logo abaixo. Ele fazia o DELETE e
	// o esquecimento à mão, e era esse chamador — um teste — que sustentava a
	// cópia da sequência que morava no hospedeiro (ALE-377).
	if err := s.sessionLifecycle().Delete(ctx, app.Caller{ID: owner}, campaign, session); err != nil {
		t.Fatalf("apagar a sessão: %v", err)
	}

	// O MAPA: o tabuleiro morreu com a sessão. Sem isto, toda gravação seguinte
	// bate na FK, para sempre.
	open := boardsRead(s.boards.OpenBoards(ctx, session))
	if len(open) != 0 {
		t.Errorf("a sessão apagada continuou com %d tabuleiros em memória", len(open))
	}
	// CONTROLE do controle: a campanha segue de pé, e outra sessão dela grava
	// normalmente. Sem isto, um esquecimento que limpasse o store INTEIRO
	// passaria neste caso.
	other := seedSession(t, s, campaign)
	if _, err := s.boards.Open(ctx, other, "Cripta", "crypt"); err != nil {
		t.Errorf("a sessão vizinha deixou de gravar: %v", err)
	}
}

// A CAMPANHA apagada leva os tabuleiros de TODAS as sessões dela.
//
// Caso próprio porque o caminho é outro: apagar a campanha derruba as sessões
// por CASCATA no banco, e nenhuma delas passa pelo caminho de apagar sessão.
func TestADeletedCampaignLeavesNoBoardBehind(t *testing.T) {
	s, campaign, session, owner := deadSessionBoardOfOwner(t)
	ctx := context.Background()
	second := seedSession(t, s, campaign)
	if _, err := s.boards.Open(ctx, second, "Cripta", "crypt"); err != nil {
		t.Fatalf("abrir o segundo tabuleiro: %v", err)
	}

	// PELO CASO DE USO, e não pelos dois passos à mão: a ordem — esquecer antes
	// de apagar — é dele, e um caso que a repetisse aqui seria uma terceira
	// cópia dela (ALE-359).
	if err := s.campaignLifecycle().Delete(ctx, app.Caller{ID: owner}, campaign); err != nil {
		t.Fatalf("apagar a campanha: %v", err)
	}

	for _, dead := range []int64{session, second} {
		open := boardsRead(s.boards.OpenBoards(ctx, dead))
		if len(open) != 0 {
			t.Errorf("a sessão %d da campanha apagada ficou com %d tabuleiros", dead, len(open))
		}
	}
}

// O CLIENTE QUE VAI EMBORA NÃO FECHA A ABA PELA METADE.
//
// ESTE CASO INVERTEU COM A ALE-375, e a inversão é o desenho. O `Close` tirava a
// aba da memória primeiro e apagava a linha depois, com `WithoutCancel` — porque
// com o contexto da REQUISIÇÃO um cliente que fosse embora deixava a linha no
// banco e o `Dirty` aceso, sem ninguém para tentar de novo.
//
// Hoje o DELETE mora dentro do gesto e vem ANTES: o cliente que vai embora não
// fecha nada. É o tudo-ou-nada da fila aplicado ao tabuleiro — a aba continua
// aberta, a linha continua no banco, e o mestre clica de novo. O que não pode
// acontecer é o meio do caminho, que era justamente o que o `WithoutCancel`
// existia para evitar por outro lado.
func TestALeavingClientDoesNotHalfCloseABoard(t *testing.T) {
	s, _, session := deadSessionBoard(t)
	canceled, cancels := context.WithCancel(context.Background())
	cancels() // o cliente foi embora ANTES de a gravação acontecer

	if err := s.boards.Close(canceled, session, defaultTab); err == nil {
		t.Error("fechar o tabuleiro com o cliente já embora foi aceito, e o gesto não pôde ser gravado")
	}

	// E A ABA CONTINUA INTEIRA, dos dois lados. Na MEMÓRIA, porque o gesto
	// recusado não pode ter tirado nada dela; e no BANCO, que é o que um store
	// novo enxerga — sem isto, a memória poderia estar certa sobre uma linha que
	// sumiu.
	open := boardsRead(s.boards.OpenBoards(context.Background(), session))
	if len(open) != 1 {
		t.Errorf("a memória ficou com %d abas depois de um fechar RECUSADO, e era 1", len(open))
	}
	after := boards.NewStore(boards.NewSnapshots(s.queries), s.queries, s.boards.NewID, s.bus)
	reread := boardsRead(after.OpenBoards(context.Background(), session))
	if len(reread) != 1 {
		t.Errorf("o banco ficou com %d abas depois de um fechar RECUSADO, e era 1", len(reread))
	}
}

// O CONTROLE DA AUSÊNCIA, e a issue o exige com todas as letras: os casos acima
// ficariam verdes tanto pelo conserto quanto por nunca chegarem ao lugar onde o
// defeito mora.
//
// Este caso prova que o CANAL EXISTE — com a sessão apagada e o tabuleiro ainda
// em memória (o estado de antes do conserto, montado à mão), a chave estrangeira
// morde e o gesto é recusado.
//
// Aqui ele media o `Dirty` acendendo e não saindo mais. A marca não existe desde
// a ALE-375; o que se mede é a RECUSA, que é a forma que o mesmo estrago tomou.
func TestTheForeignKeyStillBitesWhenTheBoardOutlivesTheSession(t *testing.T) {
	s, _, session := deadSessionBoard(t)
	ctx := context.Background()

	// Apaga a sessão SEM avisar os stores — que é exatamente o que a produção
	// fazia antes desta issue.
	if err := s.queries.DeleteSession(ctx, session); err != nil {
		t.Fatalf("apagar a sessão: %v", err)
	}

	_, err := s.boards.AddMarker(ctx, session, defaultTab, board.BoardMarker{X: 2, Y: 3})

	if err == nil {
		t.Fatal("gravar num tabuleiro órfão passou — o canal não existe, e os outros casos não provam nada")
	}
	// E NÃO SAI SOZINHO: a sessão não volta a existir, então todo gesto seguinte
	// bate na mesma chave. É o que torna o mapa órfão um estrago e não um susto.
	if _, again := s.boards.AddMarker(ctx, session, defaultTab, board.BoardMarker{X: 4, Y: 5}); again == nil {
		t.Error("o segundo gesto passou: o defeito não é o que a issue descreve")
	}
}
