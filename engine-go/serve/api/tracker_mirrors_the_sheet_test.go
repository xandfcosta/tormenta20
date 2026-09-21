package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// A FILA ESPELHA A FICHA TAMBÉM QUANDO O GESTO VEM DA FICHA.
//
// # A promessa, e o sentido que faltava
//
// "O PV do rastreador É o PV da ficha" (ALE-122). Ela valia num sentido só: o
// mestre ferindo pela fila escreve na ficha — o `DeltaVitals` diz *"quem manda é
// a FICHA"* e a entrada espelha o resultado. O caminho inverso não existia.
//
// Sete gestos mudam o poço de um personagem e só DOIS contavam à fila: o dano
// pela própria fila e o mestre descansando o grupo. Os outros cinco são os da
// FICHA — os botões ±PV/±PM, beber uma dose, conjurar, entrar numa postura e o
// descanso de dia pedido pelo jogador.
//
// # O sintoma é SILENCIOSO, e é da pior família
//
// Nada falha. A fila desenha um número plausível, e o mestre escolhe alvo e
// ordem de iniciativa olhando um PV que não existe mais. Medido no navegador ao
// fechar a ALE-355: a ficha e o cartão do Grupo diziam 117/137 e o crachá da
// iniciativa, 137 de 137.
//
// # Ele afirma o que o MESTRE VÊ, e não o estado cru
//
// A fila viva mora em memória e tem vários sítios de escrita; o que o mestre lê
// é o que a CENA DA MESA desenha, e ela passa pelo `RefreshCharacterVitals` a
// cada desenho — inclusive no batimento de um segundo do stream. Afirmar sobre
// o `GetState` seria afirmar sobre um passo intermediário e deixar de fora
// justamente o passo que conserta.
//
// # O CONTROLE
//
// A primeira metade afirma que a linha ENTROU na fila com o PV cheio. Sem ela,
// "a linha mostra 132" poderia ser uma linha que nunca teve número nenhum —
// e o caso passaria sobre uma fila vazia.
func TestTheQueueMirrorsAWoundTakenOnTheSheet(t *testing.T) {
	f := newSceneFixture(t)
	entryID := f.tracker(t)

	full := poolsOf(t, f.s, f.charID)
	if pv := queueHp(t, f, entryID); pv != full.HpCurrent {
		t.Fatalf("a linha entrou na fila com %d e a ficha tem %d — o caso mediria "+
			"uma fila que já discordava antes do gesto", pv, full.HpCurrent)
	}

	// O GESTO DA FICHA: o botão −5 da própria tela do jogador.
	target := fmt.Sprintf("/personagens/%d/vitais/pv/-5", f.charID)
	if rec := f.pede(t, f.player, http.MethodPost, target, ""); rec.Code != http.StatusOK {
		t.Fatalf("ferir pela ficha deu %d", rec.Code)
	}

	onSheet := poolsOf(t, f.s, f.charID)
	if onSheet.HpCurrent != full.HpCurrent-5 {
		t.Fatalf("a FICHA ficou com %d e devia ter %d: o gesto não chegou nela, "+
			"e o resto do caso mediria outra coisa", onSheet.HpCurrent, full.HpCurrent-5)
	}

	if pv := queueHp(t, f, entryID); pv != onSheet.HpCurrent {
		t.Errorf("a fila ficou com %d PV e a ficha com %d.\n"+
			"O mestre escolhe alvo e ordem olhando a fila, e ela mostra o número de antes "+
			"do gesto do jogador — sem erro nenhum em lugar nenhum.", pv, onSheet.HpCurrent)
	}
}

// queueHp lê o PV da linha COMO A MESA A DESENHA, e FALHA se ela não existir ou
// vier sem número: entrada ausente e entrada sem PV se parecem com "o espelho
// não funcionou".
func queueHp(t *testing.T, f sceneFixture, entryID string) int64 {
	t.Helper()
	v, _, err := f.s.tableScene.LoadView(context.Background(), f.gm, f.campaignID, f.sessionID)
	if err != nil {
		t.Fatalf("carregar a cena da Mesa: %v", err)
	}
	for _, row := range v.Queue {
		if row.ID != entryID {
			continue
		}
		if row.PV == nil {
			t.Fatalf("a linha %s está na fila sem barra de PV nenhuma", entryID)
		}
		return row.PV.Current
	}
	t.Fatalf("a linha %s sumiu da fila", entryID)
	return 0
}
