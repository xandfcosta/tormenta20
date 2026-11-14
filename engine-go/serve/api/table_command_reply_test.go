package api

import (
	"strings"
	"testing"
	"time"
)

// O COMANDO DA MESA RESPONDE SÓ OS SINAIS, e quem entrega a cena é o fluxo.
//
// O fluxo acorda em todo gesto (o piso `SessionChanged` do `respondGm`) e manda
// só a região que mudou. Uma resposta que trouxesse a Mesa inteira desenharia o
// gesto duas vezes — eram ~353 KB por clique, e no arrasto do pincel, por casa
// cruzada. Decisão do dono: o gesto do mestre depende do fluxo de pé.
func TestATableCommandRepliesOnlyItsSignals(t *testing.T) {
	cases := []struct{ name, path string }{
		{"o gesto avulso", "/iniciativa/por-no-mapa"},
		{"a recusa", "/iniciativa/proxima-vez"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSceneFixture(t)
			reply := f.posts(t, f.gm, f.tableUrl()+c.path, "{}")
			// O CONTROLE: a resposta chegou e diz se houve recusa. Sem ele, uma
			// resposta vazia passaria por "não trouxe a cena".
			if !strings.Contains(reply, "command_error") {
				t.Fatalf("a resposta de %s não trouxe o sinal da recusa:\n%.300s", c.path, reply)
			}
			if strings.Contains(reply, "datastar-patch-elements") {
				t.Errorf("a resposta de %s trouxe HTML da cena (%d bytes): quem desenha é o fluxo", c.path, len(reply))
			}
		})
	}
}

// O GESTO CONTÍNUO também: o pincel publica a cada casa, e o fluxo já desenha cada uma.
func TestTheBrushStrokeRepliesOnlyItsSignals(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	reply := f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 1, 1, 3, 1))
	if !strings.Contains(reply, "command_error") {
		t.Fatalf("a resposta do pincel não trouxe o sinal da recusa:\n%.300s", reply)
	}
	if strings.Contains(reply, "datastar-patch-elements") {
		t.Errorf("a resposta do pincel trouxe HTML (%d bytes): quem desenha é o fluxo", len(reply))
	}
}

// O GESTO CHEGA À TELA DE QUEM CLICOU PELO FLUXO DELA, na hora: é a metade que a
// resposta sem cena passou a depender. O avanço da cena é o botão mais clicado
// da sessão, e o mestre que inicia a cena tem de ver o "Encerrar a cena".
func TestTheGestureReachesTheClickersOwnStream(t *testing.T) {
	f := newSceneFixture(t)
	srv := liveServer(t, f)
	table := opensStream(t, srv, f, f.gm, f.tableUrl()+"/fluxo")
	// O CONTROLE: a cena nasce fechada. Sem ele, "Encerrar a cena" depois não
	// distinguiria o gesto de uma tela que já estava assim.
	table.waitsFor(t, 3*time.Second, "a Mesa do mestre não desenhou a cena fechada", containing("Iniciar uma cena de Ação"))

	f.posts(t, f.gm, f.tableUrl()+"/cena/iniciar/acao", "{}")
	table.waitsFor(t, eventReach, "o mestre iniciou a cena e a tela dele não a mostrou aberta", containing("Encerrar a cena"))
}

// A CASA PINTADA CHEGA PELO FLUXO, e só a região do mapa: o acervo, a maior região
// da Mesa, não muda quando alguém pinta, e o hash do fluxo a cala.
func TestTheBrushStrokeReachesTheStreamAlone(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	srv := liveServer(t, f)
	table := opensStream(t, srv, f, f.gm, f.tableUrl()+"/fluxo")
	table.waitsFor(t, 3*time.Second, "a Mesa do mestre não desenhou o mapa", containing(`id="table-board"`))
	// Esvazia a carga fria: cada região veio num quadro, e o que se mede é o que
	// chega DEPOIS do traço.
	drain := time.After(200 * time.Millisecond)
	for draining := true; draining; {
		select {
		case <-table.frames:
		case <-drain:
			draining = false
		}
	}

	f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 1, 1, 3, 1))
	table.waitsFor(t, eventReach, "a casa pintada não chegou ao mapa pelo fluxo", func(frame string) bool {
		if strings.Contains(frame, `id="table-archive"`) {
			t.Errorf("o traço repintou o acervo, que não muda ao pintar uma casa")
		}
		return strings.Contains(frame, `id="table-board"`) && strings.Contains(frame, "board-difficult")
	})
}
