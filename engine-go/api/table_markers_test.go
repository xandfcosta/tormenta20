package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/tabuleiro"
	"testing"
)

func mapMarkers(t *testing.T, f pilotoFixture) []tabuleiro.BoardMarker {
	t.Helper()
	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if b == nil {
		t.Fatal("não há tabuleiro — o gesto não tinha onde acontecer")
	}
	return b.Markers
}

// TestTheMarkerIsBornHiddenAndWithTheFreeLetter — as duas garantias da ALE-195.
//
// ESCONDIDO é a razão de o marcador existir: marcar a armadilha na frente da
// mesa entrega a armadilha. E a LETRA vem do servidor, não da tela: na SPA era o
// cliente que escolhia "A", "B", "C", e duas telas escolhendo por conta própria
// é como nasce o segundo "C" no mesmo mapa.
func TestTheMarkerIsBornHiddenAndWithTheFreeLetter(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	base := f.tableUrl() + "/tabuleiro/marcadores"

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/novo/2/3", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/novo/4/5", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar o segundo deu %d", rec.Code)
	}

	marcadores := mapMarkers(t, f)
	if len(marcadores) != 2 {
		t.Fatalf("o mapa ficou com %d marcadores, esperado 2", len(marcadores))
	}
	if marcadores[0].Text != "A" || marcadores[1].Text != "B" {
		t.Errorf("as letras saíram %q e %q, esperado A e B", marcadores[0].Text, marcadores[1].Text)
	}
	for _, m := range marcadores {
		if !m.Hidden {
			t.Errorf("o marcador %q nasceu VISÍVEL — a armadilha foi entregue à mesa", m.Text)
		}
	}
	// A casa clicada é a casa marcada, e o plano não tem bordas: a table.Coordinate
	// atravessa o caminho inteira.
	if marcadores[0].X != 2 || marcadores[0].Y != 3 {
		t.Errorf("o marcador caiu em (%d,%d) e o clique foi em (2,3)", marcadores[0].X, marcadores[0].Y)
	}
}

// TestRevealTogglesInsteadOfOnlyRevealing.
//
// O mestre que revelou cedo demais precisa poder esconder de volta, e um segundo
// botão para desfazer o primeiro seria a mesma decisão em dois lugares.
func TestRevealTogglesInsteadOfOnlyRevealing(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	base := f.tableUrl() + "/tabuleiro/marcadores"
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/novo/1/1", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar deu %d", rec.Code)
	}
	id := mapMarkers(t, f)[0].ID

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/"+id+"/revelar", ""); rec.Code != http.StatusOK {
		t.Fatalf("revelar deu %d", rec.Code)
	}
	if mapMarkers(t, f)[0].Hidden {
		t.Fatal("revelar não revelou — e sem isto o resto não mede nada")
	}
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/"+id+"/revelar", ""); rec.Code != http.StatusOK {
		t.Fatalf("esconder de volta deu %d", rec.Code)
	}
	if !mapMarkers(t, f)[0].Hidden {
		t.Error("o gesto não esconde de volta: quem revelou cedo demais fica sem saída")
	}
}

// TestAColorOutsideTheListIsRefusedWithASentence.
//
// O `UpdateMarker` IGNORA cor desconhecida, e ignorar em silêncio é um clique
// que não faz nada e não diz nada — o mestre lê como tela travada. A recusa
// nomeia o valor recebido E o esperado, que é a regra da casa para mensagem de
// erro.
func TestAColorOutsideTheListIsRefusedWithASentence(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	base := f.tableUrl() + "/tabuleiro/marcadores"
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/novo/1/1", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar deu %d", rec.Code)
	}
	id := mapMarkers(t, f)[0].ID
	corAntes := mapMarkers(t, f)[0].Color

	corpo := f.posta(t, f.mestre, base+"/"+id+"/cor/gold", "")

	// A FRASE INTEIRA e não as palavras soltas, e esta linha custou uma
	// sabotagem: procurar "gold" e "Carmim" no corpo passava VERDE mesmo com a
	// recusa apagada, porque as duas aparecem no HTML por acidente — a classe
	// `grimorio-gold` e o `title` do botão de cor. Asserção sobre substring
	// comum mede a página, não a mensagem.
	// AS ASPAS VÊM ESCAPADAS, e isto foi MEDIDO e não suposto: a recusa viaja
	// dentro do sinal `erroDoComando`, num `data: signals {...}` JSON, então o
	// `%q` do servidor chega como `\"gold\"`. Procurar a frase com aspas
	// normais falhava sobre uma recusa que estava lá.
	if !strings.Contains(corpo, `\"gold\" não existe`) {
		t.Errorf("a recusa não nomeou a cor recebida; resposta: %.400s", corpo)
	}
	if !strings.Contains(corpo, "as do mapa são") {
		t.Errorf("a recusa não disse qual era a forma esperada; resposta: %.400s", corpo)
	}
	if depois := mapMarkers(t, f)[0].Color; depois != corAntes {
		t.Errorf("a cor mudou para %q apesar da recusa", depois)
	}

	// O CONTROLE: uma cor BOA passa. Sem ele, "a cor não mudou" seria verdade
	// também num gesto que nunca funciona.
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/"+id+"/cor/carmim", ""); rec.Code != http.StatusOK {
		t.Fatalf("pintar de carmim deu %d", rec.Code)
	}
	if got := mapMarkers(t, f)[0].Color; got != "carmim" {
		t.Errorf("a cor boa não entrou: ficou %q", got)
	}
}

// TestThePlayerDoesNotTouchTheMarkers — a trava é do servidor.
//
// Os três gestos numa varredura só: o botão escondido é cortesia, e cada rota
// nova é uma linha de registro que alguém pode trocar sem perceber.
func TestThePlayerDoesNotTouchTheMarkers(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	base := f.tableUrl() + "/tabuleiro/marcadores"
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/novo/1/1", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar deu %d", rec.Code)
	}
	id := mapMarkers(t, f)[0].ID

	for _, gesto := range []string{"novo/7/7", id + "/revelar", id + "/cor/azul", id + "/remover"} {
		rec := f.pede(t, f.jogador, http.MethodPost, base+"/"+gesto, "")
		if rec.Code != http.StatusForbidden {
			t.Errorf("o jogador passou em %q: %d", gesto, rec.Code)
		}
	}
	marcadores := mapMarkers(t, f)
	if len(marcadores) != 1 || !marcadores[0].Hidden || marcadores[0].Color != tabuleiro.DefaultMarkerColor() {
		t.Errorf("o mapa mudou apesar dos 403: %+v", marcadores)
	}
}

// TestTheGmSeesTheMarkerStateAndTheTableDoesNotSeeTheHiddenOne.
//
// Duas garantias que se parecem: a mesa não recebe o marcador escondido (isso o
// `BoardForRole` já fazia), e o MESTRE precisa distinguir o que ele vê do que a
// mesa vê — senão ele revela e a tela dele não muda, que é justamente a pergunta
// que o gesto de revelar existe para responder.
func TestTheGmSeesTheMarkerStateAndTheTableDoesNotSeeTheHiddenOne(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	base := f.tableUrl() + "/tabuleiro/marcadores"
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/novo/1/1", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar deu %d", rec.Code)
	}

	doMestre := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doMestre, "escondido da mesa") {
		t.Fatal("a tela do mestre não diz que o marcador está escondido — ele revela e nada muda para ele")
	}
	// O CONTROLE de que a busca acha o marcador quando ele ESTÁ lá, para o
	// "não achei" da tela do jogador significar alguma coisa.
	if !strings.Contains(doMestre, "Marcador A") {
		t.Fatal("o marcador não apareceu nem para o mestre — o guarda está medindo a tela errada")
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(doJogador, "Marcador A") {
		t.Error("o marcador ESCONDIDO chegou ao HTML do jogador")
	}

	id := mapMarkers(t, f)[0].ID
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/"+id+"/revelar", ""); rec.Code != http.StatusOK {
		t.Fatalf("revelar deu %d", rec.Code)
	}
	revelado := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(revelado, "Marcador A") {
		t.Error("revelado, o marcador continua sem chegar à mesa")
	}
}

// TestDeleteRemovesTheMarkerAndAnInventedIdIsRefused: apagar tira o marcador do mapa, e um id inventado RECUSA em vez de sumir.
//
// A recusa importa porque a alternativa é uma mutação que não acha ninguém e
// responde 200: a tela diria que apagou algo que continua lá.
func TestDeleteRemovesTheMarkerAndAnInventedIdIsRefused(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	base := f.tableUrl() + "/tabuleiro/marcadores"
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/novo/1/1", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar deu %d", rec.Code)
	}
	id := mapMarkers(t, f)[0].ID

	corpo := f.posta(t, f.mestre, base+"/nao-existe/remover", "")
	if !strings.Contains(corpo, "nao-existe") {
		t.Errorf("a recusa não nomeou o id inventado; resposta: %.200s", corpo)
	}
	if len(mapMarkers(t, f)) != 1 {
		t.Fatal("o id inventado mexeu no mapa")
	}

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/"+id+"/remover", ""); rec.Code != http.StatusOK {
		t.Fatalf("apagar deu %d", rec.Code)
	}
	if n := len(mapMarkers(t, f)); n != 0 {
		t.Errorf("o marcador continua no mapa (%d)", n)
	}
}
