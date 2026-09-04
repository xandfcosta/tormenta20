package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/tabuleiro"
)

/*
O RASCUNHO DE LUGAR (ALE-292): a cena montada FORA da sessão.

A capacidade existia no domínio desde a ALE-191 e nenhum caminho chegava até
ela. O que estes casos prendem é o CAMINHO — a superfície do tabuleiro apontada
para o acervo — e as três coisas que ele não pode errar: quem entra, para onde
os gestos postam, e o que a tela promete sobre gravar.

Integração e não unitário porque é COMPOSIÇÃO que se está provando: a rota, a
trava, a view do tabuleiro reusada e a gravação no acervo. A regra de cada
mutação já está presa no `tabuleiro`, e reafirmá-la aqui seria a mesma
fronteira duas vezes.
*/

// draftPlace guarda um lugar no acervo e devolve o id dele.
func (f pilotoFixture) draftPlace(t *testing.T, nome, chao string) int64 {
	t.Helper()
	lugar, err := f.s.tableHost().Boards().NewPlace(context.Background(), f.campaignID, nome, chao)
	if err != nil {
		t.Fatalf("criar o lugar %q: %v", nome, err)
	}
	return lugar.ID
}

func (f pilotoFixture) draftUrl(placeID int64) string {
	return fmt.Sprintf("/campanhas/%d/lugares/%d", f.campaignID, placeID)
}

// A cena do rascunho desenha o TABULEIRO — o mesmo plano, o mesmo trilho — e
// diz, na tarja, que ninguém está vendo.
//
// A tarja é a razão de este caso existir: o mapa do rascunho é IGUALZINHO ao da
// mesa, e sem uma linha dizendo o contrário o mestre monta a emboscada sem saber
// de que lado do tempo ele está. É a lição da cortina (ALE-202) aplicada aqui.
func TestTheDraftDrawsTheBoardAndSaysNobodyIsWatching(t *testing.T) {
	f := novoPiloto(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "cripta")

	corpo := f.pede(t, f.mestre, http.MethodGet, f.draftUrl(lugar), "").Body.String()

	if !strings.Contains(corpo, "tabuleiro-plano") {
		t.Error("o rascunho não desenhou o plano do tabuleiro")
	}
	if !strings.Contains(corpo, "Cripta de Thwor") {
		t.Error("o rascunho não diz que lugar está sendo montado")
	}
	if !strings.Contains(corpo, "a mesa não vê") {
		t.Error("a tarja não diz que ninguém está vendo — o mestre não tem como saber em que tempo está")
	}
	if !strings.Contains(corpo, "chao-cripta") {
		t.Error("o chão escolhido não foi desenhado")
	}
}

// OS GESTOS POSTAM NO ACERVO, e não numa sessão que não existe.
//
// É o guarda do `BoardView.Base`, medido no HTML que sai. Sem ele, uma chamada
// que continuasse escrevendo o caminho da mesa postaria em `/mesa/N/0/tabuleiro`
// — um endereço que RESPONDE, com 403 ou 404, e devolve uma tela que não mudou.
// O sintoma seria "o pincel não pinta", sem uma linha em lugar nenhum.
func TestTheDraftGesturesPostToTheArchiveAndNotToATable(t *testing.T) {
	f := novoPiloto(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "cripta")

	corpo := f.pede(t, f.mestre, http.MethodGet, f.draftUrl(lugar), "").Body.String()

	esperado := fmt.Sprintf("/campanhas/%d/lugares/%d/tabuleiro", f.campaignID, lugar)
	if !strings.Contains(corpo, esperado) {
		t.Fatalf("nenhum gesto posta em %q", esperado)
	}
	// O CONTROLE, e ele é o que separa "não achei" de "não procurei": a mesma
	// página NÃO pode carregar o caminho da mesa. `/mesa/` sozinho apareceria
	// num link de navegação legítimo, então o que se procura é o caminho do
	// TABULEIRO de uma sessão.
	if strings.Contains(corpo, "/tabuleiro/terreno") && !strings.Contains(corpo, esperado+"/terreno") {
		t.Error("o pincel do rascunho posta num tabuleiro que não é o dele")
	}
	if strings.Contains(corpo, fmt.Sprintf("/mesa/%d/0/tabuleiro", f.campaignID)) {
		t.Error("um gesto escapou para a mesa com sessão ZERO — é o defeito que o `Base` existe para impedir")
	}
}

// O gesto atravessa o caminho inteiro e o acervo muda.
//
// Pela porta de VERDADE (`posta`), num servidor HTTP real: o SDK do Datastar
// fecha o corpo do pedido ao criar o gerador SSE, e o par
// `httptest.NewRequest` + recorder não reproduz esse ciclo de vida — a ordem
// trocada passa verde na suíte e quebra toda escrita no servidor.
func TestADraftGestureChangesTheArchivedScene(t *testing.T) {
	f := novoPiloto(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "cripta")

	f.posta(t, f.mestre, f.draftUrl(lugar)+"/tabuleiro/pecas/nova/4/3",
		`{"novapecanome":"Porta da cripta","novapecatamanho":1,"novapecaaparencia":"object"}`)

	cena, err := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, lugar)
	if err != nil {
		t.Fatalf("reabrir o rascunho: %v", err)
	}
	if len(cena.Tokens) != 1 {
		t.Fatalf("o gesto não chegou ao acervo: %+v", cena.Tokens)
	}
	if cena.Tokens[0].Label != "Porta da cripta" || cena.Tokens[0].X != 4 || cena.Tokens[0].Y != 3 {
		t.Errorf("a peça não nasceu onde o clique disse: %+v", cena.Tokens[0])
	}
	// A peça nasce com id do SERVIDOR: sem ele nada consegue selecioná-la para
	// mover, editar ou remover no gesto seguinte.
	if cena.Tokens[0].ID == "" {
		t.Error("a peça do rascunho nasceu sem id")
	}
}

// O ACERVO É DO MESTRE. Um jogador da campanha não monta o rascunho — e a trava
// é do SERVIDOR, não do botão escondido.
//
// O que ele veria não é um detalhe: a cripta de sábado, com a emboscada
// posicionada e os marcadores que ainda não foram revelados.
func TestAStrangerDoesNotReachThePlaceDraft(t *testing.T) {
	f := novoPiloto(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "cripta")

	pagina := f.pede(t, f.jogador, http.MethodGet, f.draftUrl(lugar), "")
	if pagina.Code != http.StatusForbidden {
		t.Errorf("o jogador abriu o rascunho com %d", pagina.Code)
	}
	if strings.Contains(pagina.Body.String(), "Cripta de Thwor") {
		t.Error("a recusa vazou o nome do lugar que ela existe para esconder")
	}

	// E POSTANDO NA MÃO, que é o caso que o botão escondido não cobre.
	gesto := f.posta(t, f.jogador, f.draftUrl(lugar)+"/tabuleiro/pecas/nova/4/3",
		`{"novapecanome":"Intruso","novapecatamanho":1,"novapecaaparencia":"object"}`)
	if strings.Contains(gesto, "datastar") {
		t.Errorf("o gesto do jogador foi atendido: %q", gesto)
	}
	cena, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, lugar)
	if len(cena.Tokens) != 0 {
		t.Errorf("o jogador escreveu no acervo: %+v", cena.Tokens)
	}
}

// A cena do rascunho NÃO oferece os verbos da sessão.
//
// Não é cosmético: "Encerrar o tabuleiro" aqui arquivaria uma cena que já é
// acervo, e a cortina prometeria esconder de uma mesa que não existe. Cada um
// deles precisa de uma sessão do outro lado, e um botão que não pode funcionar é
// pior que a ausência dele — ele ensina um gesto errado.
func TestTheDraftDoesNotOfferTheSessionVerbs(t *testing.T) {
	f := novoPiloto(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "cripta")

	corpo := f.pede(t, f.mestre, http.MethodGet, f.draftUrl(lugar), "").Body.String()

	for _, verbo := range []string{
		"Encerrar o tabuleiro",
		"Abrir outro tabuleiro",
		"Lugares da campanha",
		"/tabuleiro/cortina/",
		"/tabuleiro/lente",
	} {
		if strings.Contains(corpo, verbo) {
			t.Errorf("o rascunho oferece %q, que precisa de uma mesa do outro lado", verbo)
		}
	}
	// O CONTROLE: o que o rascunho TEM continua lá. Sem ele, uma cena que
	// falhasse em desenhar o tabuleiro inteiro passaria neste caso — ausência de
	// botão e ausência de tela se parecem no `strings.Contains`.
	if !strings.Contains(corpo, "Ferramentas do mapa") {
		t.Fatal("o trilho de ferramentas sumiu junto: o guarda mediu uma tela vazia")
	}
}

// O lugar que está numa MESA AO VIVO não se monta — e a recusa chega à tela.
//
// A trava mora no `EditPlace` e está presa lá; o que este caso prende é o
// CAMINHO até ela: a sessão viva da campanha é resolvida pelo gateway e chega
// à regra. Sem isso a regra existiria e nunca seria consultada.
func TestTheDraftOfAPlaceOnALiveTableIsRefused(t *testing.T) {
	f := novoPiloto(t)
	// A taverna é aberta na sessão da fixture — o estado normal de uma partida
	// em andamento. O status da sessão NÃO importa para a trava, e é de
	// propósito: uma sessão encerrada guarda os tabuleiros dela e reabre com
	// eles, então "está na mesa" é sobre o tabuleiro, não sobre a partida.
	f.seedOpenBoard(t, "taverna")
	if err := f.s.tableHost().Boards().Archive(context.Background(), f.campaignID,
		f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}
	lugar := placeNamed(t, f.s.tableHost().Boards().Places(context.Background(), f.campaignID),
		"Taverna do Javali")

	resposta := f.posta(t, f.mestre, f.draftUrl(lugar.ID)+"/tabuleiro/pecas/nova/4/3",
		`{"novapecanome":"Fantasma","novapecatamanho":1,"novapecaaparencia":"object"}`)

	if !strings.Contains(resposta, "está aberto numa mesa agora") {
		t.Errorf("a recusa não chegou à tela: %q", resposta)
	}
	cena, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, lugar.ID)
	for _, peca := range cena.Tokens {
		if peca.Label == "Fantasma" {
			t.Error("o gesto passou por cima da trava e escreveu no acervo")
		}
	}
	// CONTROLE: o mesmo gesto num lugar que NÃO está na mesa passa. Sem ele,
	// uma recusa por qualquer outro motivo — id errado, rota que não existe —
	// seria lida como "a trava funcionou".
	outro := f.draftPlace(t, "Cripta de Thwor", "cripta")
	f.posta(t, f.mestre, f.draftUrl(outro)+"/tabuleiro/pecas/nova/4/3",
		`{"novapecanome":"Porta","novapecatamanho":1,"novapecaaparencia":"object"}`)
	if livre, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, outro); len(livre.Tokens) != 1 {
		t.Fatalf("o gesto foi recusado no lugar que NÃO está na mesa: %+v", livre.Tokens)
	}
}

// A peça se MOVE no rascunho, direto, sem proposta.
//
// A diferença é o que a issue nomeia: na mesa o arrasto propõe um movimento com
// custo e vez, e alguém confirma; aqui não há vez para gastar. Uma proposta
// pendurada num rascunho seria um movimento que ninguém pode confirmar — e o
// `PlaceScene` a descarta na leitura seguinte, então ela sumiria em silêncio.
func TestTheDraftMovesThePieceWithoutAProposal(t *testing.T) {
	f := novoPiloto(t)
	lugar := f.draftPlace(t, "Cripta de Thwor", "cripta")
	// O ID vem do SERVIDOR e não do teste: o `AddToken` cunha um sempre, e
	// escolher um aqui seria arranjar um dado que a produção nunca produz.
	semeada, err := f.s.tableHost().Boards().EditPlace(context.Background(), f.campaignID, lugar,
		func(b *tabuleiro.BoardState) error {
			return tabuleiro.AddToken(b, tabuleiro.BoardToken{
				Label: "Porta", X: 1, Y: 1, Footprint: 1,
			}, f.s.tableHost().Boards().NewID)
		})
	if err != nil {
		t.Fatalf("semear a peça: %v", err)
	}
	id := semeada.Tokens[0].ID

	f.posta(t, f.mestre, f.draftUrl(lugar)+"/tabuleiro/pecas/"+id+"/mover/6/2", "{}")

	cena, _ := f.s.tableHost().Boards().PlaceScene(context.Background(), f.campaignID, lugar)
	peca := tabuleiro.FindToken(cena, id)
	if peca == nil {
		t.Fatal("a peça sumiu do rascunho")
	}
	if peca.X != 6 || peca.Y != 2 {
		t.Errorf("a peça não foi para a casa clicada: está em (%d,%d)", peca.X, peca.Y)
	}
	if cena.Pending != nil {
		t.Error("o rascunho ficou com um movimento PROPOSTO, que ninguém pode confirmar")
	}
}
