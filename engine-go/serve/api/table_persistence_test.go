package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// A GRAVAÇÃO DO TABULEIRO, pelo caminho inteiro: comando da Mesa, regra, disco.
//
// Aqui moravam o `TestTheGmIsWarnedWhenSavingFails` e o
// `TestThePlayerIsNotWarnedAboutSaving`, que mediam a tarja "a mesa não está
// sendo salva" no cabeçalho do mestre. **A tarja saiu com a ALE-375**, e o que
// ela avisava deixou de poder acontecer: a gravação mora dentro da mutação, e o
// gesto que o disco recusa volta RECUSADO em vez de mudar a tela sobre um disco
// que não recebeu nada.
//
// O que aqueles dois protegiam — "a mesa não pode rodar de memória em silêncio"
// — continua protegido, e pelos dois casos abaixo: um diz que a mutação CHEGA ao
// disco, o outro que a mutação que NÃO chega é recusada com a frase.

// UMA MUTAÇÃO PELA CENA CHEGA AO DISCO.
//
// Os casos do `board_store_test.go` dirigem o STORE direto; nenhum mede o
// caminho inteiro. Sem este, apagar a gravação não quebra teste nenhum.
func TestACommandFromTheTableReachesTheDisk(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	sheet, _ := sceneIds(t, f)

	f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/pecas", `{"map_selection":"`+sheet+`"}`)

	// O CONTROLE: a peça entrou na memória. Sem isto, um disco vazio não
	// distingue "não gravou" de "não havia o que gravar".
	if b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)); len(b.Tokens) == 0 {
		t.Fatal("a peça não entrou no mapa — o guarda mediria o vazio")
	}
	// SEM SONDAGEM, e isto é a fatia: a gravação era em goroutine e ler uma vez
	// logo depois do comando era uma corrida, então este guarda sondava o disco
	// por até dois segundos. Hoje o comando só responde depois de gravar.
	var rows int
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM open_boards WHERE sessionId = ?", f.sessionID).Scan(&rows); err != nil {
		t.Fatalf("consultar o disco: %v", err)
	}
	if rows == 0 {
		t.Error("o tabuleiro mexeu e o disco não recebeu nada — a mesa está vivendo só em memória")
	}
}

// A MUTAÇÃO QUE O DISCO RECUSA VOLTA RECUSADA, com a frase na cena.
//
// É o substituto da tarja, e ele é melhor pelo mesmo motivo que a fatia inteira:
// a tarja só podia aparecer DEPOIS de a peça já ter andado na tela sobre um
// disco vazio. Aqui a peça não anda.
//
// A recusa volta em 200 com a cena redesenhada, e não em 4xx: o Datastar
// DESCARTA o remendo de toda resposta não-2xx, e uma recusa em 400 não
// apareceria na tela. Por isso o que se afirma é a FRASE.
func TestACommandTheDiskRefusesComesBackRefused(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	sheet, _ := sceneIds(t, f)
	ctx := context.Background()

	// O CONTROLE primeiro: com o disco saudável o gesto passa e a peça entra.
	// Sem esta metade, "a peça não entrou" não distingue a recusa de um pedido
	// que nunca funcionou.
	f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/pecas", `{"map_selection":"`+sheet+`"}`)
	before := len(boardRead(f.s.tableHost().Boards().Get(ctx, f.sessionID, defaultTab)).Tokens)
	if before == 0 {
		t.Fatal("o controle falhou: a peça não entrou com o disco saudável")
	}

	// A SABOTAGEM é derrubar a TABELA, e as duas óbvias não servem: sem
	// tabuleiro aberto não há o que gravar, e fechar o `*sql.DB` mata a LEITURA
	// junto — a página nem renderiza. O defeito de verdade é escrita falhando
	// com leitura funcionando, que é como ele fica invisível.
	if _, err := f.s.db.ExecContext(ctx, "DROP TABLE open_boards"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}

	body := f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/pecas", `{"map_selection":"`+sheet+`"}`)

	if !strings.Contains(body, "open_boards") {
		t.Errorf("a gravação falhou e a cena não disse nada ao mestre.\ncorpo: %s", primeiros(body, 400))
	}
	// E A MESA NÃO MUDOU: a peça não pode ter andado sobre um disco que recusou.
	after := len(boardRead(f.s.tableHost().Boards().Get(ctx, f.sessionID, defaultTab)).Tokens)
	if after != before {
		t.Errorf("o mapa foi de %d para %d peças sobre uma gravação recusada", before, after)
	}
}

// O JOGADOR continua vendo a mesa — o controle de que a cena não quebrou.
func TestThePlayerStillSeesTheTable(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)

	body := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(body, "Arcanista") {
		t.Error("o jogador não viu a própria mesa")
	}
}
