package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/engine"
	"t20engine/tabuleiro"
	"t20engine/web/table"
	"testing"
)

func mapToken(t *testing.T, f pilotoFixture, rotulo string, x, y int) string {
	t.Helper()
	posto, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{Label: rotulo, X: x, Y: y, Kind: "npc"}, true)
	if err != nil {
		t.Fatalf("pôr a peça %q: %v", rotulo, err)
	}
	return posto.Tokens[len(posto.Tokens)-1].ID
}

func nowBoard(t *testing.T, f pilotoFixture) *tabuleiro.BoardState {
	t.Helper()
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if b == nil {
		t.Fatal("não há tabuleiro — o gesto não tinha onde acontecer")
	}
	return b
}

// TestHidingTheTokenIsTheGestureThatWasMissing — o buraco que esta fatia fecha.
//
// A capacidade estava no `BoardStore` desde a ALE-178 e não tinha rota nenhuma na
// Mesa em Datastar: a mesma forma da cortina, no ar e invisível. E a ausência
// dela deixava OUTRA superfície mentindo — "ver como jogador" (ALE-193) existe
// para conferir a emboscada, e sem um gesto de esconder ela respondia sempre
// "nenhuma peça escondida nesta cena".
func TestHidingTheTokenIsTheGestureThatWasMissing(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "cripta")
	id := mapToken(t, f, "Ogro", 4, 4)
	base := f.tableUrl() + "/tabuleiro/pecas/" + id

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/visibilidade", ""); rec.Code != http.StatusOK {
		t.Fatalf("esconder deu %d", rec.Code)
	}
	if !tabuleiro.FindToken(nowBoard(t, f), id).Hidden {
		t.Fatal("a peça não foi escondida")
	}
	// A MESA deixa de vê-la, que é o ponto inteiro: a trava é o `BoardForRole`, e
	// este caso afirma que o gesto passa por ele em vez de só pintar diferente.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(doJogador, "Ogro") {
		t.Error("a peça escondida continuou na tela do jogador")
	}

	// ALTERNA: o mestre que escondeu cedo demais precisa poder mostrar de volta, e
	// um segundo botão para desfazer o primeiro seria a mesma decisão em dois
	// lugares.
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/visibilidade", ""); rec.Code != http.StatusOK {
		t.Fatalf("mostrar deu %d", rec.Code)
	}
	if tabuleiro.FindToken(nowBoard(t, f), id).Hidden {
		t.Error("mostrar não devolveu a peça à mesa")
	}
}

// TestTakingOffTheMapDoesNotTakeOutOfCombat — a separação que a cena promete.
//
// São duas perguntas diferentes — "ele saiu do mapa" e "ele saiu do combate" —, e
// juntá-las faria o mestre perder o combatente ao arrumar a cena. É a mesma
// separação que o elenco e a fila já têm (superfície 6b).
func TestTakingOffTheMapDoesNotTakeOutOfCombat(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	entryID := f.tracker(t)
	posto, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{Label: "Arcanista", X: 0, Y: 0, EntryID: &entryID}, true)
	if err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	id := posto.Tokens[len(posto.Tokens)-1].ID

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/remover", ""); rec.Code != http.StatusOK {
		t.Fatalf("remover deu %d", rec.Code)
	}
	if tabuleiro.FindToken(nowBoard(t, f), id) != nil {
		t.Error("a peça continuou no tabuleiro")
	}
	// A LINHA fica: quem estava no combate continua no combate.
	tracker := false
	for _, e := range f.s.tableHost().Sessions().GetState(f.sessionID).Initiative {
		tracker = tracker || e.ID == entryID
	}
	if !tracker {
		t.Error("tirar a peça do mapa tirou o combatente da fila")
	}
}

// TestUndoOnlyExistsWhereThereIsSomewhereToGoBackTo.
//
// Um botão que não faz nada é pior que nenhum, e aqui ele seria pior ainda:
// "voltar para onde estava" numa peça que nunca se moveu promete desfazer algo
// que ninguém lembra de ter feito.
func TestUndoOnlyExistsWhereThereIsSomewhereToGoBackTo(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	id := mapToken(t, f, "Ogro", 1, 1)
	base := f.tableUrl() + "/tabuleiro/pecas/" + id

	// Sem movimento nenhum: o servidor recusa E a tela não desenha o verbo.
	rec := f.pede(t, f.mestre, http.MethodPost, base+"/voltar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("o comando deu %d — a recusa é uma frase, não um status", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "não há para onde voltar") {
		t.Errorf("voltar sem movimento não recusou:\n%s", rec.Body.String())
	}
	if tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String(); strings.Contains(tela, "Voltar Ogro para") {
		t.Error("a tela ofereceu voltar numa peça que não se moveu")
	}

	// Agora com um movimento CONFIRMADO: o mestre move sem orçamento.
	mover := f.tableUrl() + "/tabuleiro/" + id
	if rec := f.pede(t, f.mestre, http.MethodPost, mover+"/parada/5/1", ""); rec.Code != http.StatusOK {
		t.Fatalf("a parada deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, http.MethodPost, mover+"/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if peca := tabuleiro.FindToken(nowBoard(t, f), id); peca.X != 5 {
		t.Fatalf("a peça não andou: está em (%d,%d)", peca.X, peca.Y)
	}

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/voltar", ""); rec.Code != http.StatusOK {
		t.Fatalf("voltar deu %d", rec.Code)
	}
	peca := tabuleiro.FindToken(nowBoard(t, f), id)
	if peca.X != 1 || peca.Y != 1 {
		t.Errorf("a peça voltou para (%d,%d), esperado (1,1)", peca.X, peca.Y)
	}
	// UMA vez e não uma pilha: voltar LIMPA o registro, então o botão some. Um
	// "voltar" que continuasse disponível andaria para trás na cena com um botão
	// que não diz até onde vai.
	if peca.DeOndeVeio != nil {
		t.Errorf("o voltar continuou disponível, apontando para %v", peca.DeOndeVeio)
	}
}

// TestUndoSurvivesAReload.
//
// É a divergência DELIBERADA em relação à SPA: lá o desfazer do posicionamento
// mora na memória da aba e morre no F5 (`ondeEstava`, em `board-region`). O gesto
// que ele conserta — "arrastei o dragão para o lugar errado na frente de seis
// pessoas" — é justamente o que se quer desfazer de qualquer tela.
func TestUndoSurvivesAReload(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	id := mapToken(t, f, "Dragão", 2, 2)
	mover := f.tableUrl() + "/tabuleiro/" + id
	for _, passo := range []string{"/parada/8/8", "/confirmar"} {
		if rec := f.pede(t, f.mestre, http.MethodPost, mover+passo, ""); rec.Code != http.StatusOK {
			t.Fatalf("%s deu %d", passo, rec.Code)
		}
	}

	// Uma carga fria, como quem apertou F5: nada do navegador anterior viaja.
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(tela, "Voltar Dragão para "+table.Coordinate(2, 2)) {
		t.Error("a página recarregada perdeu o voltar — ele não sobreviveu ao F5")
	}
}

// TestDuplicateNumbersOnTheServer (ALE-192).
//
// Duas telas escolhendo o número por conta própria é como nasce o segundo
// "Zumbi 3" no mesmo mapa. E a cópia nasce AO LADO da original: quem duplica o
// zumbi do canto espera o irmão dele ali, não na fileira de entrada.
func TestDuplicateNumbersOnTheServer(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	id := mapToken(t, f, "Zumbi", 3, 3)

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar", ""); rec.Code != http.StatusOK {
		t.Fatalf("duplicar deu %d", rec.Code)
	}
	b := nowBoard(t, f)
	if len(b.Tokens) != 2 {
		t.Fatalf("o mapa ficou com %d peças, esperado 2", len(b.Tokens))
	}
	copia := b.Tokens[1]
	if copia.Label == "Zumbi" {
		t.Error("a cópia ficou com o mesmo nome — dois 'Zumbi' no mesmo mapa")
	}
	if perto := engine.RangeSquares(engine.Square{X: 3, Y: 3}, engine.Square{X: copia.X, Y: copia.Y}); perto > 2 {
		t.Errorf("a cópia nasceu a %d quadrados da original", perto)
	}
}

// TestEditingRefusesASizeTheBookDoesNotHave.
//
// O livro define 1, 2, 3 e 6 (T20 p107, Tab. 1-21) — não existe 4 nem 5. O
// número vem do cliente, e uma peça de lado 4 mentiria sobre quem o gabarito pega
// e sobre onde cabe passar.
func TestEditingRefusesASizeTheBookDoesNotHave(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	id := mapToken(t, f, "Ogro", 1, 1)
	base := f.tableUrl() + "/tabuleiro/pecas/" + id + "/editar"

	recusa := f.posta(t, f.mestre, base, `{"pecanome":"Ogro","pecatamanho":4}`)
	if !strings.Contains(recusa, "1, 2, 3 ou 6") {
		t.Errorf("o lado 4 não foi recusado:\n%s", recusa)
	}
	semNome := f.posta(t, f.mestre, base, `{"pecanome":"  ","pecatamanho":1}`)
	if !strings.Contains(semNome, "precisa de um nome") {
		t.Errorf("o nome vazio não foi recusado:\n%s", semNome)
	}
	// E o caso positivo, sem o qual as duas recusas acima seriam verdade também
	// numa rota que recusa tudo.
	f.posta(t, f.mestre, base, `{"pecanome":"Ogro Capitão","pecatamanho":2}`)
	peca := tabuleiro.FindToken(nowBoard(t, f), id)
	if peca.Label != "Ogro Capitão" || peca.Footprint != 2 {
		t.Errorf("a edição válida não pegou: %q, lado %d", peca.Label, peca.Footprint)
	}
}

// TestOnlyTheGmTouchesTheToken: a trava é do servidor, e não o menu escondido.
//
// O menu é do mestre porque quem monta a mesa é ele, e o botão que o jogador não
// vê nunca foi prova de trava — quem postar na mão leva 403.
func TestOnlyTheGmTouchesTheToken(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	id := mapToken(t, f, "Ogro", 1, 1)
	base := f.tableUrl() + "/tabuleiro/pecas/" + id

	for _, verbo := range []string{"/visibilidade", "/duplicar", "/voltar", "/remover"} {
		if rec := f.pede(t, f.jogador, http.MethodPost, base+verbo, ""); rec.Code != http.StatusForbidden {
			t.Errorf("o jogador alcançou %s: %d", verbo, rec.Code)
		}
	}
	// E o MENU não é desenhado para ele — cortesia, não trava.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doJogador, "Ogro") {
		t.Fatal("o jogador não viu nem a peça — a página não é o que este teste pensa que é")
	}
	if strings.Contains(doJogador, "O que fazer com Ogro") {
		t.Error("o menu do mestre apareceu na tela do jogador")
	}
}
