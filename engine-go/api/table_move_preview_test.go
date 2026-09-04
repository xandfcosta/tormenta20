package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestThePreviewDrawsWithoutTouchingTheScene(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoardAt(t, 4, 2)
	f.turnPlayer(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	rec := f.pede(t, f.jogador, http.MethodPost, base+"/previa/9/2", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("a prévia deu %d", rec.Code)
	}
	sinais := trechoDeSinais(rec.Body.String())

	// Cinco casas para o leste custam 5, que cabem no deslocamento de 6: fio de
	// uma faixa só, e a frase nomeando a ação.
	if !strings.Contains(sinais, `"previafiocabe":"M 4.5 2.5 L 9 2.5"`) {
		t.Errorf("a prévia não desenhou a seta da perna viva; sinais = %s", sinais)
	}
	if !strings.Contains(sinais, "5 de 6 quadrados") || !strings.Contains(sinais, "ação de movimento") {
		t.Errorf("a prévia não diz o custo nem a ação; sinais = %s", sinais)
	}
	// A DISTÂNCIA EM METROS sobre a linha, que é o pedido ao pé da letra.
	if !strings.Contains(sinais, `"t":"7,5m"`) {
		t.Errorf("a prévia não põe a distância em metros na seta; sinais = %s", sinais)
	}

	// E A CENA NÃO MUDOU: nem a peça andou, nem nasceu proposta.
	tela := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(tela, "tabuleiro-peca-fantasma") {
		t.Error("a prévia deixou uma proposta na cena: arrastar viraria uma proposta por casa")
	}
	if !strings.Contains(tela, "Arcanista em 4, 2") {
		t.Error("a peça saiu do lugar por causa de uma prévia")
	}
}

// A prévia ESTENDE o caminho já desenhado em vez de recomeçá-lo.
//
// Decisão do dono ("o caminho todo + a perna viva"), e é ela que faz a leitura
// responder à pergunta do gesto: "se eu soltar aqui, quanto GASTEI?" — que uma
// perna medida sozinha não responde.
func TestThePreviewExtendsThePathAlreadyDrawn(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoardAt(t, 0, 0)
	f.turnPlayer(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID

	if rec := f.pede(t, f.jogador, http.MethodPost, base+"/parada/3/0", ""); rec.Code != http.StatusOK {
		t.Fatalf("a primeira parada deu %d", rec.Code)
	}
	sinais := trechoDeSinais(f.pede(t, f.jogador, http.MethodPost, base+"/previa/6/0", "").Body.String())

	// Três mais três: o total é 6, e não 3. Recomeçar daria "3 de 6".
	if !strings.Contains(sinais, "6 de 6 quadrados") {
		t.Errorf("a prévia recomeçou o caminho em vez de estendê-lo; sinais = %s", sinais)
	}
	// DOIS rótulos, um por perna: a parada posta continua tendo o número dela.
	if strings.Count(sinais, `"t":"4,5m"`) != 2 {
		t.Errorf("as duas pernas não ganharam rótulo próprio; sinais = %s", sinais)
	}
}

// A PRÉVIA PINTA AS TRÊS FAIXAS, com a mesma conta da seta guardada.
//
// O CONTROLE está no caso acima (um caminho que cabe sai de uma cor só), então
// aqui basta o caro: quinze quadrados sobre um deslocamento de seis passam das
// duas ações e têm de acender as três.
func TestThePreviewPaintsTheThreeBands(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoardAt(t, 0, 0)
	f.turnPlayer(t)

	sinais := trechoDeSinais(f.pede(t, f.jogador, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/previa/15/0", "").Body.String())

	for _, fio := range []string{"previafiocabe", "previafiosegundo", "previafioalem"} {
		if strings.Contains(sinais, `"`+fio+`":""`) {
			t.Errorf("a faixa %q saiu vazia num caminho que passa das duas ações; sinais = %s", fio, sinais)
		}
	}
	if !strings.Contains(sinais, "não cabe no turno") {
		t.Errorf("a prévia não diz que o caminho não cabe; sinais = %s", sinais)
	}
}

// FORA DE COMBATE a prévia mede e não pinta faixa nenhuma.
//
// O CONTROLE do guarda acima: sem vez não há ação de movimento a gastar, e as
// três cores diriam respeito a um teto que a cena não tem.
func TestOutOfCombatThePreviewMeasuresWithoutBands(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.onBoardAt(t, 0, 0)

	sinais := trechoDeSinais(f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/previa/15/0", "").Body.String())

	if !strings.Contains(sinais, `"previafiosegundo":""`) || !strings.Contains(sinais, `"previafioalem":""`) {
		t.Errorf("fora de combate a prévia pintou faixa de ação; sinais = %s", sinais)
	}
	// E MEDE do mesmo jeito: o canal está aberto, então o vazio acima é o
	// resultado e não a ausência de resposta.
	if strings.Contains(sinais, `"previafiocabe":""`) {
		t.Errorf("a prévia não desenhou seta nenhuma fora de combate; sinais = %s", sinais)
	}
	if !strings.Contains(sinais, "15 quadrados") {
		t.Errorf("a prévia não mediu fora de combate; sinais = %s", sinais)
	}
}
