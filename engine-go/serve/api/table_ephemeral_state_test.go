package api

import (
	"context"
	"net/http"
	"testing"

	"t20engine/app"
)

// A MEMÓRIA EFÊMERA DA MESA MORRE COM A SESSÃO.
//
// A lente acesa e a aba que cada pessoa escolheu são estado do SERVIDOR, num
// mapa por `(sessão, pessoa)`. O único lugar que os esvaziava era o `endBoard`,
// e ele só o faz quando a ÚLTIMA cena fecha: certo para fechar uma aba, e
// nenhuma cobertura para a sessão APAGADA.
//
// Este vazamento não tinha sintoma na TELA, e é por isso que sobreviveu — quem
// lê a aba escolhida confere contra os tabuleiros abertos, e a lente só acende
// botão de mestre. Era memória que nunca voltava, e o guarda é de LIGAÇÃO: a
// Mesa está na lista de mapas que o fim de uma sessão alcança (ALE-377).
func TestADeletedSessionLeavesNoEphemeralTableState(t *testing.T) {
	f := newSceneFixture(t)
	ctx := context.Background()
	f.seedOpenBoard(t, "stone")
	crypt := f.openSecond(t, "Cripta")

	// TRÊS entradas de propósito, uma de cada natureza: a lente do mestre, a
	// escolha de aba do jogador, e o puxão — que é chaveado pela SESSÃO sozinha e
	// escaparia de um esquecimento que só varresse as chaves compostas.
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/tabuleiro/aba/"+crypt.ID, "")
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/lente", "")
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/aba/"+crypt.ID+"/mostrar", "")

	// O DENOMINADOR, antes de apagar: sem ele, "não sobrou nada" não se distingue
	// de um caso que nunca guardou nada — as duas coisas são zero.
	before := f.s.tableMemory.Watching(f.sessionID)
	if before == 0 {
		t.Fatalf("os gestos não guardaram memória nenhuma: o caso mediria o nada")
	}

	if err := f.s.sessionLifecycle().Delete(ctx, app.Caller{ID: f.gm}, f.campaignID, f.sessionID); err != nil {
		t.Fatalf("apagar a sessão: %v", err)
	}

	if left := f.s.tableMemory.Watching(f.sessionID); left != 0 {
		t.Errorf("a sessão apagada deixou %d entradas de memória efêmera (eram %d)", left, before)
	}
}

// E O ESQUECIMENTO É DE UMA SESSÃO, não do processo.
//
// Caso próprio porque o defeito simétrico é pior que o vazamento: um `Erase` que
// limpasse o mapa inteiro derrubaria a lente e a aba de toda mesa que estivesse
// no ar naquele instante, e isso a suíte só acusa com DUAS sessões vivas.
func TestForgettingOneSessionKeepsTheOtherTableLooking(t *testing.T) {
	f := newSceneFixture(t)
	ctx := context.Background()
	f.seedOpenBoard(t, "stone")
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/lente", "")

	neighbor := seedSession(t, f.s, f.campaignID)
	if _, err := f.s.tableHost().Boards().Open(ctx, neighbor, "Cripta", "stone"); err != nil {
		t.Fatalf("abrir o tabuleiro da sessão vizinha: %v", err)
	}
	f.s.sessionLifecycle().ForgetSession(neighbor)

	if left := f.s.tableMemory.Watching(f.sessionID); left == 0 {
		t.Error("esquecer a sessão vizinha apagou a lente da sessão que continua no ar")
	}
}
