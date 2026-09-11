package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/board"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/live"
	"t20engine/web/table"
	"testing"
)

func mapToken(t *testing.T, f pilotoFixture, rotulo string, x, y int) string {
	t.Helper()
	posto, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: rotulo, X: x, Y: y, Kind: "npc"})
	if err != nil {
		t.Fatalf("pôr a peça %q: %v", rotulo, err)
	}
	return posto.Tokens[len(posto.Tokens)-1].ID
}

func nowBoard(t *testing.T, f pilotoFixture) *board.BoardState {
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
	f.seedOpenBoard(t, "crypt")
	id := mapToken(t, f, "Ogro", 4, 4)
	base := f.tableUrl() + "/tabuleiro/pecas/" + id

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/visibilidade", ""); rec.Code != http.StatusOK {
		t.Fatalf("esconder deu %d", rec.Code)
	}
	if !board.FindToken(nowBoard(t, f), id).Hidden {
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
	if board.FindToken(nowBoard(t, f), id).Hidden {
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
	f.seedOpenBoard(t, "stone")
	entryID := f.tracker(t)
	posto, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: "Arcanista", X: 0, Y: 0, EntryID: &entryID})
	if err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	id := posto.Tokens[len(posto.Tokens)-1].ID

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/remover", ""); rec.Code != http.StatusOK {
		t.Fatalf("remover deu %d", rec.Code)
	}
	if board.FindToken(nowBoard(t, f), id) != nil {
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
	f.seedOpenBoard(t, "stone")
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
	if rec := f.pede(t, f.mestre, http.MethodPost, mover+"/parada", `{"from":{"X":5,"Y":1}}`); rec.Code != http.StatusOK {
		t.Fatalf("a parada deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, http.MethodPost, mover+"/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if peca := board.FindToken(nowBoard(t, f), id); peca.X != 5 {
		t.Fatalf("a peça não andou: está em (%d,%d)", peca.X, peca.Y)
	}

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/voltar", ""); rec.Code != http.StatusOK {
		t.Fatalf("voltar deu %d", rec.Code)
	}
	peca := board.FindToken(nowBoard(t, f), id)
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
	f.seedOpenBoard(t, "stone")
	id := mapToken(t, f, "Dragão", 2, 2)
	mover := f.tableUrl() + "/tabuleiro/" + id
	for _, passo := range []struct{ rota, corpo string }{
		{"/parada", `{"from":{"X":8,"Y":8}}`},
		{"/confirmar", ""},
	} {
		if rec := f.pede(t, f.mestre, http.MethodPost, mover+passo.rota, passo.corpo); rec.Code != http.StatusOK {
			t.Fatalf("%s deu %d", passo.rota, rec.Code)
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
	f.seedOpenBoard(t, "stone")
	id := mapToken(t, f, "Zumbi", 3, 3)

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/peca", ""); rec.Code != http.StatusOK {
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

// ── OS TRÊS DUPLICARES (ALE-206) ─────────────────────────────────────────────
//
// A distinção é o que a cópia faz com a LINHA DA FILA, e ela se prova AQUI e não
// no `tabuleiro`: a regra de qual vínculo a cópia leva já está presa lá, e o que
// esta faixa cobre é a COMPOSIÇÃO — a linha nova nasce, ela nasce CHEIA, e a
// peça aponta para ela e não para outra.

// tokenOnTheQueue põe no mapa uma peça amarrada à linha de um combatente.
//
// O `mapToken` põe peça SOLTA, que é o caso do cenário; sem esta o teste dos
// modos mediria sempre a recusa.
func tokenOnTheQueue(t *testing.T, f pilotoFixture, rotulo string) (string, string) {
	t.Helper()
	estado := f.s.sessions.GetState(f.sessionID)
	var linha string
	for i := range estado.Initiative {
		if estado.Initiative[i].Label == rotulo {
			linha = estado.Initiative[i].ID
		}
	}
	if linha == "" {
		t.Fatalf("%q não está na fila: o resto do caso não mediria nada", rotulo)
	}
	posto, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: rotulo, X: 3, Y: 3, Kind: "npc", EntryID: &linha})
	if err != nil {
		t.Fatalf("pôr a peça de %q: %v", rotulo, err)
	}
	return posto.Tokens[len(posto.Tokens)-1].ID, linha
}

// TestTheCopyWithItsOwnLineEntersTheQueueWhole.
//
// É o "mais um zumbi" de montar encontro, e o número que importa é o PV: o ogro
// da bancada está com 12 de 130, e o segundo ogro chega INTEIRO. Copiar o PV
// atual daria um irmão que já nasce sangrando pela porrada que o primeiro levou
// — quem quer isso está pedindo "sangra junto", que é o outro verbo.
func TestTheCopyWithItsOwnLineEntersTheQueueWhole(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	id, linhaOriginal := tokenOnTheQueue(t, f, "Ogro cansado")
	antes := len(f.s.sessions.GetState(f.sessionID).Initiative)

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/sozinha", ""); rec.Code != http.StatusOK {
		t.Fatalf("duplicar com PV próprio deu %d", rec.Code)
	}

	fila := f.s.sessions.GetState(f.sessionID)
	if len(fila.Initiative) != antes+1 {
		t.Fatalf("a fila ficou com %d linhas, esperado %d", len(fila.Initiative), antes+1)
	}
	b := nowBoard(t, f)
	copia := b.Tokens[len(b.Tokens)-1]
	if copia.EntryID == nil {
		t.Fatal("a cópia nasceu sem linha: ela não teria barra de PV nenhuma")
	}
	if *copia.EntryID == linhaOriginal {
		t.Fatal("a cópia ficou na linha da ORIGINAL — isso é o 'sangra junto', não o PV próprio")
	}
	var nova *live.InitiativeEntry
	for i := range fila.Initiative {
		if fila.Initiative[i].ID == *copia.EntryID {
			nova = &fila.Initiative[i]
		}
	}
	if nova == nil {
		t.Fatal("a peça aponta para uma linha que não está na fila")
	}
	// O NÚMERO ESCRITO NA MÃO, e não derivado da linha original: derivá-lo do
	// código sob teste esconderia justamente a troca de "cheio" por "atual".
	if live.DerefOr(nova.HpCurrent, 0) != 130 || live.DerefOr(nova.HpMax, 0) != 130 {
		t.Errorf("o segundo ogro entrou com %d/%d, esperado 130/130 — ele chega inteiro",
			live.DerefOr(nova.HpCurrent, 0), live.DerefOr(nova.HpMax, 0))
	}
}

// TestTheCopySharingTheLineAddsNoLine: as duas peças, uma barra só.
func TestTheCopySharingTheLineAddsNoLine(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	id, linha := tokenOnTheQueue(t, f, "Ogro cansado")
	antes := len(f.s.sessions.GetState(f.sessionID).Initiative)

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/junto", ""); rec.Code != http.StatusOK {
		t.Fatalf("duplicar sangrando junto deu %d", rec.Code)
	}

	if depois := len(f.s.sessions.GetState(f.sessionID).Initiative); depois != antes {
		t.Errorf("a fila ganhou linha: %d → %d, e o ponto deste modo é NÃO ganhar", antes, depois)
	}
	b := nowBoard(t, f)
	copia := b.Tokens[len(b.Tokens)-1]
	if copia.EntryID == nil || *copia.EntryID != linha {
		t.Errorf("a cópia aponta para %v, esperado a linha da original — sem isso o dano não aparece nas duas", copia.EntryID)
	}
}

// TestTheModesThatNeedALineRefuseALoosePiece.
//
// Peça de cenário não tem PV, e os dois modos que falam de PV não têm o que
// fazer com ela. A recusa é ESCRITA porque o silêncio ali produziria uma cópia
// idêntica à do peão mudo com outro nome — o mestre clicaria em "com PV próprio"
// e receberia exatamente o que "só a peça" dá.
func TestTheModesThatNeedALineRefuseALoosePiece(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "stone")
	id := mapToken(t, f, "Baú", 1, 1)

	for _, modo := range []string{"junto", "sozinha"} {
		recusa := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/"+modo, "")
		if !strings.Contains(recusa, "não é um combatente da fila") {
			t.Errorf("o modo %q não recusou a peça solta:\n%s", modo, recusa)
		}
	}
	// O CONTROLE: o peão mudo, na mesma peça, PASSA. Sem ele as duas recusas
	// acima seriam verdade também numa rota que recusa tudo.
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/peca", ""); rec.Code != http.StatusOK {
		t.Fatalf("o peão mudo também foi recusado: %d", rec.Code)
	}
	if b := nowBoard(t, f); len(b.Tokens) != 2 {
		t.Errorf("o mapa ficou com %d peças, esperado 2", len(b.Tokens))
	}
}

// TestEditingRefusesASizeTheBookDoesNotHave.
//
// O livro define 1, 2, 3 e 6 (T20 p107, Tab. 1-21) — não existe 4 nem 5. O
// número vem do cliente, e uma peça de lado 4 mentiria sobre quem o gabarito pega
// e sobre onde cabe passar.
func TestEditingRefusesASizeTheBookDoesNotHave(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "stone")
	id := mapToken(t, f, "Ogro", 1, 1)
	base := f.tableUrl() + "/tabuleiro/pecas/" + id + "/editar"

	recusa := f.posta(t, f.mestre, base, `{"token_name":"Ogro","token_size":4}`)
	if !strings.Contains(recusa, "1, 2, 3 ou 6") {
		t.Errorf("o lado 4 não foi recusado:\n%s", recusa)
	}
	semNome := f.posta(t, f.mestre, base, `{"token_name":"  ","token_size":1}`)
	if !strings.Contains(semNome, "precisa de um nome") {
		t.Errorf("o nome vazio não foi recusado:\n%s", semNome)
	}
	// E o caso positivo, sem o qual as duas recusas acima seriam verdade também
	// numa rota que recusa tudo.
	f.posta(t, f.mestre, base, `{"token_name":"Ogro Capitão","token_size":2}`)
	peca := board.FindToken(nowBoard(t, f), id)
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
	f.seedOpenBoard(t, "stone")
	id := mapToken(t, f, "Ogro", 1, 1)
	base := f.tableUrl() + "/tabuleiro/pecas/" + id

	for _, verbo := range []string{"/visibilidade", "/duplicar/peca", "/duplicar/junto", "/duplicar/sozinha", "/voltar", "/remover"} {
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

// ── COPIAR E COLAR (ALE-206) ─────────────────────────────────────────────────
//
// O colar faz três coisas que o duplicar não faz: repete sem perguntar de novo,
// pousa onde a pessoa está OLHANDO, e ATRAVESSA AS ABAS. A terceira é a que não
// tinha caminho nenhum antes, e é a que estes casos prendem.

// colaNaAba manda o comando de colar com a área apontando para outra aba.
func colaNaAba(t *testing.T, f pilotoFixture, deOndeVeio, peca, modo string, x, y int) string {
	t.Helper()
	area := fmt.Sprintf(`{"area_token":%q,"area_board":%q,"area_mode":%q}`, peca, deOndeVeio, modo)
	return f.posta(t, f.mestre,
		fmt.Sprintf("%s/tabuleiro/colar/%d/%d", f.tableUrl(), x, y), area)
}

// TestThePasteCrossesTheTabs.
//
// Copiar o zumbi na Cripta e colá-lo na Taverna. O servidor procura a original
// no tabuleiro que a ÁREA nomeia, e não no que está na tela — são diferentes
// justamente quando o colar mais serve.
func TestThePasteCrossesTheTabs(t *testing.T) {
	f := novoPiloto(t)
	cripta := f.seedOpenBoard(t, "stone")
	posto, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, cripta.ID,
		board.BoardToken{Label: "Zumbi", X: 1, Y: 1, Kind: "npc"})
	if err != nil {
		t.Fatalf("pôr a peça na cripta: %v", err)
	}
	naCripta := posto.Tokens[len(posto.Tokens)-1].ID

	// A SEGUNDA ABA, e o comando age nela porque é a que o mestre está olhando.
	taverna, err := f.s.tableHost().Boards().Open(context.Background(), f.sessionID, "Taverna", "madeira")
	if err != nil {
		t.Fatalf("abrir a segunda aba: %v", err)
	}
	if taverna.ID == cripta.ID {
		t.Fatal("as duas abas têm o mesmo id: o caso não mediria travessia nenhuma")
	}
	// A ABA é escolhida pela PORTA de verdade, e não mexendo no campo do
	// servidor: é o mesmo gesto de clicar na aba, e ele é quem decide em qual
	// tabuleiro o comando age.
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/aba/"+taverna.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("escolher a aba da taverna deu %d", rec.Code)
	}

	if recusa := colaNaAba(t, f, cripta.ID, naCripta, "peca", 6, 4); strings.Contains(recusa, "não há peça na área") {
		t.Fatalf("o colar recusou:\n%s", recusa)
	}

	naTaverna := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, taverna.ID)
	if len(naTaverna.Tokens) != 1 {
		t.Fatalf("a taverna ficou com %d peças, esperado 1", len(naTaverna.Tokens))
	}
	colada := naTaverna.Tokens[0]
	if colada.X != 6 || colada.Y != 4 {
		t.Errorf("a cópia pousou em (%d,%d), esperado o quadrado pedido (6,4)", colada.X, colada.Y)
	}
	// E a CRIPTA não perdeu a original: colar copia, não move.
	if depois := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, cripta.ID); len(depois.Tokens) != 1 {
		t.Errorf("a cripta ficou com %d peças — o colar levou a original junto", len(depois.Tokens))
	}
}

// TestThePasteWithoutAClipboardSaysSo: a recusa é escrita.
//
// `CTRL + V` sem nada na área é o gesto mais provável de todos — a tecla existe
// no dedo de quem usa qualquer outro programa. O silêncio ali seria a mesma tela
// de antes, e a pessoa apertaria de novo.
func TestThePasteWithoutAClipboardSaysSo(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "stone")

	recusa := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/colar/2/2", `{"area_token":""}`)
	if !strings.Contains(recusa, "não há peça na área") {
		t.Errorf("a área vazia não foi recusada:\n%s", recusa)
	}
}

// TestThePasteOfAPieceThatIsGoneSaysSo.
//
// A área é do CLIENTE e vive mais que a peça: o mestre copia o zumbi, tira o
// zumbi do mapa, e aperta CTRL+V. Sem esta frase o colar sairia calado.
func TestThePasteOfAPieceThatIsGoneSaysSo(t *testing.T) {
	f := novoPiloto(t)
	b := f.seedOpenBoard(t, "stone")

	recusa := colaNaAba(t, f, b.ID, "peca-que-nao-existe", "peca", 2, 2)
	if !strings.Contains(recusa, "não está mais no tabuleiro de origem") {
		t.Errorf("a peça sumida não foi recusada:\n%s", recusa)
	}
}

// TestThePasteWithItsOwnLineAlsoFillsTheQueue: o modo grudado na área vale
// igual no colar.
//
// Ele é o mesmo `bondForMode` do duplicar, e este caso é quem prova que os dois
// verbos concordam sobre o que "com PV próprio" significa.
func TestThePasteWithItsOwnLineAlsoFillsTheQueue(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	b := f.seedOpenBoard(t, "stone")
	id, _ := tokenOnTheQueue(t, f, "Ogro cansado")
	antes := len(f.s.sessions.GetState(f.sessionID).Initiative)

	if recusa := colaNaAba(t, f, b.ID, id, "sozinha", 8, 8); strings.Contains(recusa, "não há peça") {
		t.Fatalf("o colar recusou:\n%s", recusa)
	}

	fila := f.s.sessions.GetState(f.sessionID)
	if len(fila.Initiative) != antes+1 {
		t.Fatalf("a fila ficou com %d linhas, esperado %d", len(fila.Initiative), antes+1)
	}
	mapa := nowBoard(t, f)
	colada := mapa.Tokens[len(mapa.Tokens)-1]
	if colada.EntryID == nil {
		t.Fatal("a cópia colada nasceu sem linha: ela não teria barra de PV")
	}
	var nova *live.InitiativeEntry
	for i := range fila.Initiative {
		if fila.Initiative[i].ID == *colada.EntryID {
			nova = &fila.Initiative[i]
		}
	}
	if nova == nil {
		t.Fatal("a peça colada aponta para uma linha que não está na fila")
	}
	if live.DerefOr(nova.HpCurrent, 0) != 130 {
		t.Errorf("o ogro colado entrou com %d de PV, esperado 130 — ele chega inteiro",
			live.DerefOr(nova.HpCurrent, 0))
	}
}

// ── O CHEFE QUE GANHA NOME (ALE-206) ─────────────────────────────────────────

// TestTheCopyWithItsOwnBlockClonesTheCreature.
//
// Duas linhas dividem um bloco sem problema — ele é um MOLDE. Clonar só importa
// quando o mestre vai EDITAR uma das duas: sem a cópia, dar 30 PV a mais ao
// chefe daria aos outros três zumbis também.
//
// É o BLOCO e não a ficha de personagem, e a diferença é do modelo: neste app
// `characterId` é PC de jogador e `creatureId` é a criatura que o mestre
// escreveu. Clonar personagem exigiria matricular a cópia na campanha, e todo
// membro aparece no painel do Grupo — um zumbi duplicado entraria lá.
func TestTheCopyWithItsOwnBlockClonesTheCreature(t *testing.T) {
	f := novoPiloto(t)
	agora := "2026-01-01T00:00:00Z"
	bloco, err := f.s.queries.CreateCampaignCreature(context.Background(), sqlcgen.CreateCampaignCreatureParams{
		Campaignid: f.campaignID, Name: "Zumbi",
		Block:     `{` + blocoMinimo + `}`,
		Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		t.Fatalf("semear o bloco: %v", err)
	}
	if _, err := f.s.sessions.StartScene(f.sessionID); err != nil {
		t.Fatalf("iniciar cena: %v", err)
	}
	pv := int64(20)
	if _, err := f.s.sessions.AddInitiativeEntry(f.sessionID, live.InitiativeEntry{
		Label: "Zumbi", Initiative: 10, Type: "npc",
		HpCurrent: &pv, HpMax: &pv, CreatureID: &bloco.ID,
	}); err != nil {
		t.Fatalf("semear a linha: %v", err)
	}
	f.seedOpenBoard(t, "stone")
	id, linhaOriginal := tokenOnTheQueue(t, f, "Zumbi")

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/bloco", ""); rec.Code != http.StatusOK {
		t.Fatalf("duplicar com bloco próprio deu %d", rec.Code)
	}

	fila := f.s.sessions.GetState(f.sessionID)
	var nova *live.InitiativeEntry
	for i := range fila.Initiative {
		if fila.Initiative[i].ID != linhaOriginal {
			nova = &fila.Initiative[i]
		}
	}
	if nova == nil {
		t.Fatal("a fila não ganhou linha")
	}
	if nova.CreatureID == nil {
		t.Fatal("a linha nova nasceu sem bloco: o chefe não teria ficha para editar")
	}
	if *nova.CreatureID == bloco.ID {
		t.Fatal("a linha nova aponta para o bloco da ORIGINAL — editar um mexeria no outro")
	}
	copia, err := f.s.queries.GetCampaignCreature(context.Background(), *nova.CreatureID)
	if err != nil {
		t.Fatalf("o bloco copiado não está no acervo: %v", err)
	}
	if copia.Block != `{`+blocoMinimo+`}` {
		t.Errorf("o bloco copiado veio diferente do original:\n%s", copia.Block)
	}
	// O NOME do bloco acompanha o da linha, e este é o ponto: com dois "Zumbi"
	// no acervo, o olho da fila abriria um bloco chamado como o outro.
	if copia.Name != nova.Label {
		t.Errorf("o bloco se chama %q e a linha %q — dois nomes para a mesma criatura",
			copia.Name, nova.Label)
	}
	if copia.Campaignid != f.campaignID {
		t.Errorf("o bloco copiado caiu na campanha %d", copia.Campaignid)
	}
}

// TestTheOwnBlockModeRefusesWhoHasNone: herói e NPC digitado à mão não têm bloco.
//
// O menu já esconde o verbo nesses casos; a trava é do servidor, e o botão
// escondido nunca foi prova de trava.
func TestTheOwnBlockModeRefusesWhoHasNone(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	id, _ := tokenOnTheQueue(t, f, "Ogro cansado")

	recusa := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/bloco", "")
	if !strings.Contains(recusa, "não tem bloco de criatura") {
		t.Errorf("o modo do bloco aceitou uma linha sem bloco:\n%s", recusa)
	}
	// O CONTROLE: o "com PV próprio", na MESMA peça, passa. Sem ele a recusa
	// acima seria verdade também numa rota que recusa tudo.
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/sozinha", ""); rec.Code != http.StatusOK {
		t.Fatalf("o modo com PV próprio também foi recusado: %d", rec.Code)
	}
}
