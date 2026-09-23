package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/table"
	"testing"
)

func mapToken(t *testing.T, f sceneFixture, label string, x, y int) string {
	t.Helper()
	placed, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: label, X: x, Y: y, Kind: "npc"})
	if err != nil {
		t.Fatalf("pôr a peça %q: %v", label, err)
	}
	return placed.Tokens[len(placed.Tokens)-1].ID
}

func nowBoard(t *testing.T, f sceneFixture) *board.BoardState {
	t.Helper()
	b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	if b == nil {
		t.Fatal("não há tabuleiro — o gesto não tinha onde acontecer")
	}
	return b
}

// Esconder a peça é o que faz "ver como jogador" dizer alguma coisa: sem um
// gesto de esconder, aquela superfície responde sempre "nenhuma peça escondida
// nesta cena".
func TestHidingTheTokenIsTheGestureThatWasMissing(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")
	id := mapToken(t, f, "Ogro", 4, 4)
	base := f.tableUrl() + "/tabuleiro/pecas/" + id

	if rec := f.requests(t, f.gm, http.MethodPost, base+"/visibilidade", ""); rec.Code != http.StatusOK {
		t.Fatalf("esconder deu %d", rec.Code)
	}
	if !board.FindToken(nowBoard(t, f), id).Hidden {
		t.Fatal("a peça não foi escondida")
	}
	// A MESA deixa de vê-la, que é o ponto inteiro: a trava é o `BoardForRole`, e
	// este caso afirma que o gesto passa por ele em vez de só pintar diferente.
	forPlayer := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(forPlayer, "Ogro") {
		t.Error("a peça escondida continuou na tela do jogador")
	}

	// ALTERNA: o mestre que escondeu cedo demais precisa poder mostrar de volta, e
	// um segundo botão para desfazer o primeiro seria a mesma decisão em dois
	// lugares.
	if rec := f.requests(t, f.gm, http.MethodPost, base+"/visibilidade", ""); rec.Code != http.StatusOK {
		t.Fatalf("mostrar deu %d", rec.Code)
	}
	if board.FindToken(nowBoard(t, f), id).Hidden {
		t.Error("mostrar não devolveu a peça à mesa")
	}
}

// São duas perguntas diferentes — "ele saiu do mapa" e "ele saiu do combate" —, e
// juntá-las faria o mestre perder o combatente ao arrumar a cena.
func TestTakingOffTheMapDoesNotTakeOutOfCombat(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	entryID := f.tracker(t)
	placed, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: "Arcanista", X: 0, Y: 0, EntryID: &entryID})
	if err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	id := placed.Tokens[len(placed.Tokens)-1].ID

	if rec := f.requests(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/remover", ""); rec.Code != http.StatusOK {
		t.Fatalf("remover deu %d", rec.Code)
	}
	if board.FindToken(nowBoard(t, f), id) != nil {
		t.Error("a peça continuou no tabuleiro")
	}
	// A LINHA fica: quem estava no combate continua no combate.
	tracker := false
	for _, e := range stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative {
		tracker = tracker || e.ID == entryID
	}
	if !tracker {
		t.Error("tirar a peça do mapa tirou o combatente da fila")
	}
}

// Um botão que não faz nada é pior que nenhum, e aqui ele seria pior ainda:
// "voltar para onde estava" numa peça que nunca se moveu promete desfazer algo
// que ninguém lembra de ter feito.
func TestUndoOnlyExistsWhereThereIsSomewhereToGoBackTo(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	id := mapToken(t, f, "Ogro", 1, 1)
	base := f.tableUrl() + "/tabuleiro/pecas/" + id

	// Sem movimento nenhum: o servidor recusa E a tela não desenha o verbo.
	rec := f.requests(t, f.gm, http.MethodPost, base+"/voltar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("o comando deu %d — a recusa é uma frase, não um status", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "não há para onde voltar") {
		t.Errorf("voltar sem movimento não recusou:\n%s", rec.Body.String())
	}
	if screen := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); strings.Contains(screen, "Voltar Ogro para") {
		t.Error("a tela ofereceu voltar numa peça que não se moveu")
	}

	// Agora com um movimento CONFIRMADO: o mestre move sem orçamento.
	mover := f.tableUrl() + "/tabuleiro/" + id
	if rec := f.requests(t, f.gm, http.MethodPost, mover+"/parada", `{"from":{"X":5,"Y":1}}`); rec.Code != http.StatusOK {
		t.Fatalf("a parada deu %d", rec.Code)
	}
	if rec := f.requests(t, f.gm, http.MethodPost, mover+"/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if token := board.FindToken(nowBoard(t, f), id); token.X != 5 {
		t.Fatalf("a peça não andou: está em (%d,%d)", token.X, token.Y)
	}

	if rec := f.requests(t, f.gm, http.MethodPost, base+"/voltar", ""); rec.Code != http.StatusOK {
		t.Fatalf("voltar deu %d", rec.Code)
	}
	piece := board.FindToken(nowBoard(t, f), id)
	if piece.X != 1 || piece.Y != 1 {
		t.Errorf("a peça voltou para (%d,%d), esperado (1,1)", piece.X, piece.Y)
	}
	// UMA vez e não uma pilha: voltar LIMPA o registro, então o botão some. Um
	// "voltar" que continuasse disponível andaria para trás na cena com um botão
	// que não diz até onde vai.
	if piece.CameFrom != nil {
		t.Errorf("o voltar continuou disponível, apontando para %v", piece.CameFrom)
	}
}

// O desfazer mora no SERVIDOR e não na memória da aba: o gesto que ele conserta
// — "arrastei o dragão para o lugar errado na frente de seis pessoas" — é
// justamente o que se quer desfazer de qualquer tela, inclusive depois do F5.
func TestUndoSurvivesAReload(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	id := mapToken(t, f, "Dragão", 2, 2)
	mover := f.tableUrl() + "/tabuleiro/" + id
	for _, step := range []struct{ route, body string }{
		{"/parada", `{"from":{"X":8,"Y":8}}`},
		{"/confirmar", ""},
	} {
		if rec := f.requests(t, f.gm, http.MethodPost, mover+step.route, step.body); rec.Code != http.StatusOK {
			t.Fatalf("%s deu %d", step.route, rec.Code)
		}
	}

	// Uma carga fria, como quem apertou F5: nada do navegador anterior viaja.
	screen := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(screen, "Voltar Dragão para "+table.Coordinate(2, 2)) {
		t.Error("a página recarregada perdeu o voltar — ele não sobreviveu ao F5")
	}
}

// Duas telas escolhendo o número por conta própria é como nasce o segundo
// "Zumbi 3" no mesmo mapa. E a cópia nasce AO LADO da original: quem duplica o
// zumbi do canto espera o irmão dele ali, não na fileira de entrada.
func TestDuplicateNumbersOnTheServer(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	id := mapToken(t, f, "Zumbi", 3, 3)

	if rec := f.requests(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/peca", ""); rec.Code != http.StatusOK {
		t.Fatalf("duplicar deu %d", rec.Code)
	}
	b := nowBoard(t, f)
	if len(b.Tokens) != 2 {
		t.Fatalf("o mapa ficou com %d peças, esperado 2", len(b.Tokens))
	}
	dup := b.Tokens[1]
	if dup.Label == "Zumbi" {
		t.Error("a cópia ficou com o mesmo nome — dois 'Zumbi' no mesmo mapa")
	}
	if near := engine.RangeSquares(engine.Square{X: 3, Y: 3}, engine.Square{X: dup.X, Y: dup.Y}); near > 2 {
		t.Errorf("a cópia nasceu a %d quadrados da original", near)
	}
}

// ── OS TRÊS DUPLICARES ───────────────────────────────────────────────────────
//
// A distinção é o que a cópia faz com a LINHA DA FILA, e ela se prova AQUI e não
// no `tabuleiro`: a regra de qual vínculo a cópia leva já está presa lá, e o que
// esta faixa cobre é a COMPOSIÇÃO — a linha nova nasce, ela nasce CHEIA, e a
// peça aponta para ela e não para outra.

// tokenOnTheQueue põe no mapa uma peça amarrada à linha de um combatente.
//
// O `mapToken` põe peça SOLTA, que é o caso do cenário; sem esta o teste dos
// modos mediria sempre a recusa.
func tokenOnTheQueue(t *testing.T, f sceneFixture, label string) (string, string) {
	t.Helper()
	state := stateOf(t, f.s.sessions, f.sessionID)
	var row string
	for i := range state.Initiative {
		if state.Initiative[i].Label == label {
			row = state.Initiative[i].ID
		}
	}
	if row == "" {
		t.Fatalf("%q não está na fila: o resto do caso não mediria nada", label)
	}
	placed, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: label, X: 3, Y: 3, Kind: "npc", EntryID: &row})
	if err != nil {
		t.Fatalf("pôr a peça de %q: %v", label, err)
	}
	return placed.Tokens[len(placed.Tokens)-1].ID, row
}

// É o "mais um zumbi" de montar encontro, e o número que importa é o PV: o ogro
// da bancada está com 12 de 130, e o segundo ogro chega INTEIRO. Copiar o PV
// atual daria um irmão que já nasce sangrando pela porrada que o primeiro levou
// — quem quer isso está pedindo "sangra junto", que é o outro verbo.
func TestTheCopyWithItsOwnLineEntersTheQueueWhole(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	id, originalLine := tokenOnTheQueue(t, f, "Ogro cansado")
	before := len(stateOf(t, f.s.sessions, f.sessionID).Initiative)

	if rec := f.requests(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/sozinha", ""); rec.Code != http.StatusOK {
		t.Fatalf("duplicar com PV próprio deu %d", rec.Code)
	}

	queue := stateOf(t, f.s.sessions, f.sessionID)
	if len(queue.Initiative) != before+1 {
		t.Fatalf("a fila ficou com %d linhas, esperado %d", len(queue.Initiative), before+1)
	}
	b := nowBoard(t, f)
	dup := b.Tokens[len(b.Tokens)-1]
	if dup.EntryID == nil {
		t.Fatal("a cópia nasceu sem linha: ela não teria barra de PV nenhuma")
	}
	if *dup.EntryID == originalLine {
		t.Fatal("a cópia ficou na linha da ORIGINAL — isso é o 'sangra junto', não o PV próprio")
	}
	var nova *live.InitiativeEntry
	for i := range queue.Initiative {
		if queue.Initiative[i].ID == *dup.EntryID {
			nova = &queue.Initiative[i]
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

// As duas peças, uma barra só.
func TestTheCopySharingTheLineAddsNoLine(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	id, row := tokenOnTheQueue(t, f, "Ogro cansado")
	before := len(stateOf(t, f.s.sessions, f.sessionID).Initiative)

	if rec := f.requests(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/junto", ""); rec.Code != http.StatusOK {
		t.Fatalf("duplicar sangrando junto deu %d", rec.Code)
	}

	if after := len(stateOf(t, f.s.sessions, f.sessionID).Initiative); after != before {
		t.Errorf("a fila ganhou linha: %d → %d, e o ponto deste modo é NÃO ganhar", before, after)
	}
	b := nowBoard(t, f)
	dup := b.Tokens[len(b.Tokens)-1]
	if dup.EntryID == nil || *dup.EntryID != row {
		t.Errorf("a cópia aponta para %v, esperado a linha da original — sem isso o dano não aparece nas duas", dup.EntryID)
	}
}

// Peça de cenário não tem PV, e os dois modos que falam de PV não têm o que
// fazer com ela. A recusa é ESCRITA porque o silêncio ali produziria uma cópia
// idêntica à do peão mudo com outro nome — o mestre clicaria em "com PV próprio"
// e receberia exatamente o que "só a peça" dá.
func TestTheModesThatNeedALineRefuseALoosePiece(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	id := mapToken(t, f, "Baú", 1, 1)

	for _, mode := range []string{"junto", "sozinha"} {
		refusal := f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/"+mode, "")
		if !strings.Contains(refusal, "não é um combatente da fila") {
			t.Errorf("o modo %q não recusou a peça solta:\n%s", mode, refusal)
		}
	}
	// O CONTROLE: o peão mudo, na mesma peça, PASSA. Sem ele as duas recusas
	// acima seriam verdade também numa rota que recusa tudo.
	if rec := f.requests(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/peca", ""); rec.Code != http.StatusOK {
		t.Fatalf("o peão mudo também foi recusado: %d", rec.Code)
	}
	if b := nowBoard(t, f); len(b.Tokens) != 2 {
		t.Errorf("o mapa ficou com %d peças, esperado 2", len(b.Tokens))
	}
}

// O livro define 1, 2, 3 e 6 (T20 p107, Tab. 1-21) — não existe 4 nem 5. O
// número vem do cliente, e uma peça de lado 4 mentiria sobre quem o gabarito pega
// e sobre onde cabe passar.
func TestEditingRefusesASizeTheBookDoesNotHave(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	id := mapToken(t, f, "Ogro", 1, 1)
	base := f.tableUrl() + "/tabuleiro/pecas/" + id + "/editar"

	refusal := f.posts(t, f.gm, base, `{"token_name":"Ogro","token_size":4}`)
	if !strings.Contains(refusal, "1, 2, 3 ou 6") {
		t.Errorf("o lado 4 não foi recusado:\n%s", refusal)
	}
	noName := f.posts(t, f.gm, base, `{"token_name":"  ","token_size":1}`)
	if !strings.Contains(noName, "precisa de um nome") {
		t.Errorf("o nome vazio não foi recusado:\n%s", noName)
	}
	// E o caso positivo, sem o qual as duas recusas acima seriam verdade também
	// numa rota que recusa tudo.
	f.posts(t, f.gm, base, `{"token_name":"Ogro Capitão","token_size":2}`)
	token := board.FindToken(nowBoard(t, f), id)
	if token.Label != "Ogro Capitão" || token.Footprint != 2 {
		t.Errorf("a edição válida não pegou: %q, lado %d", token.Label, token.Footprint)
	}
}

// A trava é do SERVIDOR: o botão que o jogador não vê nunca foi prova de trava —
// quem postar na mão leva 403.
func TestOnlyTheGmTouchesTheToken(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	id := mapToken(t, f, "Ogro", 1, 1)
	base := f.tableUrl() + "/tabuleiro/pecas/" + id

	for _, verb := range []string{"/visibilidade", "/duplicar/peca", "/duplicar/junto", "/duplicar/sozinha", "/voltar", "/remover"} {
		if rec := f.requests(t, f.player, http.MethodPost, base+verb, ""); rec.Code != http.StatusForbidden {
			t.Errorf("o jogador alcançou %s: %d", verb, rec.Code)
		}
	}
	// E o MENU não é desenhado para ele — cortesia, não trava.
	forPlayer := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forPlayer, "Ogro") {
		t.Fatal("o jogador não viu nem a peça — a página não é o que este teste pensa que é")
	}
	if strings.Contains(forPlayer, "O que fazer com Ogro") {
		t.Error("o menu do mestre apareceu na tela do jogador")
	}
}

// ── COPIAR E COLAR ───────────────────────────────────────────────────────────
//
// O colar faz três coisas que o duplicar não faz: repete sem perguntar de novo,
// pousa onde a pessoa está OLHANDO, e ATRAVESSA AS ABAS. A terceira é a que não
// tinha caminho nenhum antes, e é a que estes casos prendem.

// colaNaAba manda o comando de colar com a área apontando para outra aba.
func colaNaAba(t *testing.T, f sceneFixture, cameFrom, token, mode string, x, y int) string {
	t.Helper()
	area := fmt.Sprintf(`{"area_token":%q,"area_board":%q,"area_mode":%q,"from":{"X":%d,"Y":%d}}`,
		token, cameFrom, mode, x, y)
	return f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/colar", area)
}

// Copiar o zumbi na Cripta e colá-lo na Taverna. O servidor procura a original
// no tabuleiro que a ÁREA nomeia, e não no que está na tela — são diferentes
// justamente quando o colar mais serve.
func TestThePasteCrossesTheTabs(t *testing.T) {
	f := newSceneFixture(t)
	crypt := f.seedOpenBoard(t, "stone")
	placed, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, crypt.ID,
		board.BoardToken{Label: "Zumbi", X: 1, Y: 1, Kind: "npc"})
	if err != nil {
		t.Fatalf("pôr a peça na cripta: %v", err)
	}
	inCrypt := placed.Tokens[len(placed.Tokens)-1].ID

	// A SEGUNDA ABA, e o comando age nela porque é a que o mestre está olhando.
	tavern, err := f.s.tableHost().Boards().Open(context.Background(), f.sessionID, "Taverna", "madeira")
	if err != nil {
		t.Fatalf("abrir a segunda aba: %v", err)
	}
	if tavern.ID == crypt.ID {
		t.Fatal("as duas abas têm o mesmo id: o caso não mediria travessia nenhuma")
	}
	// A ABA é escolhida pela PORTA de verdade, e não mexendo no campo do
	// servidor: é o mesmo gesto de clicar na aba, e ele é quem decide em qual
	// tabuleiro o comando age.
	if rec := f.requests(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/aba/"+tavern.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("escolher a aba da taverna deu %d", rec.Code)
	}

	if refusal := colaNaAba(t, f, crypt.ID, inCrypt, "peca", 6, 4); strings.Contains(refusal, "não há peça na área") {
		t.Fatalf("o colar recusou:\n%s", refusal)
	}

	inTavern := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, tavern.ID))
	if len(inTavern.Tokens) != 1 {
		t.Fatalf("a taverna ficou com %d peças, esperado 1", len(inTavern.Tokens))
	}
	pasted := inTavern.Tokens[0]
	if pasted.X != 6 || pasted.Y != 4 {
		t.Errorf("a cópia pousou em (%d,%d), esperado o quadrado pedido (6,4)", pasted.X, pasted.Y)
	}
	// E a CRIPTA não perdeu a original: colar copia, não move.
	if after := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, crypt.ID)); len(after.Tokens) != 1 {
		t.Errorf("a cripta ficou com %d peças — o colar levou a original junto", len(after.Tokens))
	}
}

// A recusa é ESCRITA: `CTRL + V` sem nada na área é o gesto mais provável de todos — a tecla existe
// no dedo de quem usa qualquer outro programa. O silêncio ali seria a mesma tela
// de antes, e a pessoa apertaria de novo.
func TestThePasteWithoutAClipboardSaysSo(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	refusal := f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/colar", `{"area_token":"","from":{"X":2,"Y":2}}`)
	if !strings.Contains(refusal, "não há peça na área") {
		t.Errorf("a área vazia não foi recusada:\n%s", refusal)
	}
}

// A área é do CLIENTE e vive mais que a peça: o mestre copia o zumbi, tira o
// zumbi do mapa, e aperta CTRL+V. Sem esta frase o colar sairia calado.
func TestThePasteOfAPieceThatIsGoneSaysSo(t *testing.T) {
	f := newSceneFixture(t)
	b := f.seedOpenBoard(t, "stone")

	refusal := colaNaAba(t, f, b.ID, "peca-que-nao-existe", "peca", 2, 2)
	if !strings.Contains(refusal, "não está mais no tabuleiro de origem") {
		t.Errorf("a peça sumida não foi recusada:\n%s", refusal)
	}
}

// O modo grudado na área vale igual no colar: é o mesmo `bondForMode` do
// duplicar, e este caso prova que os dois verbos concordam sobre o que "com PV
// próprio" significa.
func TestThePasteWithItsOwnLineAlsoFillsTheQueue(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	b := f.seedOpenBoard(t, "stone")
	id, _ := tokenOnTheQueue(t, f, "Ogro cansado")
	before := len(stateOf(t, f.s.sessions, f.sessionID).Initiative)

	if refusal := colaNaAba(t, f, b.ID, id, "sozinha", 8, 8); strings.Contains(refusal, "não há peça") {
		t.Fatalf("o colar recusou:\n%s", refusal)
	}

	queue := stateOf(t, f.s.sessions, f.sessionID)
	if len(queue.Initiative) != before+1 {
		t.Fatalf("a fila ficou com %d linhas, esperado %d", len(queue.Initiative), before+1)
	}
	board := nowBoard(t, f)
	pasted := board.Tokens[len(board.Tokens)-1]
	if pasted.EntryID == nil {
		t.Fatal("a cópia colada nasceu sem linha: ela não teria barra de PV")
	}
	var nova *live.InitiativeEntry
	for i := range queue.Initiative {
		if queue.Initiative[i].ID == *pasted.EntryID {
			nova = &queue.Initiative[i]
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

// ── O CHEFE QUE GANHA NOME ───────────────────────────────────────────────────

// Duas linhas dividem um bloco sem problema — ele é um MOLDE. Clonar só importa
// quando o mestre vai EDITAR uma das duas: sem a cópia, dar 30 PV a mais ao
// chefe daria aos outros três zumbis também.
//
// É o BLOCO e não a ficha de personagem, e a diferença é do modelo: neste app
// `characterId` é PC de jogador e `creatureId` é a criatura que o mestre
// escreveu. Clonar personagem exigiria matricular a cópia na campanha, e todo
// membro aparece no painel do Grupo — um zumbi duplicado entraria lá.
func TestTheCopyWithItsOwnBlockClonesTheCreature(t *testing.T) {
	f := newSceneFixture(t)
	now := "2026-01-01T00:00:00Z"
	block, err := f.s.queries.CreateCampaignCreature(context.Background(), sqlcgen.CreateCampaignCreatureParams{
		Campaignid: f.campaignID, Name: "Zumbi",
		Block:     `{` + blocoMinimo + `}`,
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		t.Fatalf("semear o bloco: %v", err)
	}
	if _, err := f.s.sessions.StartScene(context.Background(), f.sessionID, live.SceneAction); err != nil {
		t.Fatalf("iniciar cena: %v", err)
	}
	pv := int64(20)
	if _, err := f.s.sessions.AddInitiativeEntry(context.Background(), f.sessionID, live.InitiativeEntry{
		Label: "Zumbi", Initiative: 10, Type: "npc",
		HpCurrent: &pv, HpMax: &pv, CreatureID: &block.ID,
	}); err != nil {
		t.Fatalf("semear a linha: %v", err)
	}
	f.seedOpenBoard(t, "stone")
	id, originalLine := tokenOnTheQueue(t, f, "Zumbi")

	if rec := f.requests(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/bloco", ""); rec.Code != http.StatusOK {
		t.Fatalf("duplicar com bloco próprio deu %d", rec.Code)
	}

	queue := stateOf(t, f.s.sessions, f.sessionID)
	var nova *live.InitiativeEntry
	for i := range queue.Initiative {
		if queue.Initiative[i].ID != originalLine {
			nova = &queue.Initiative[i]
		}
	}
	if nova == nil {
		t.Fatal("a fila não ganhou linha")
	}
	if nova.CreatureID == nil {
		t.Fatal("a linha nova nasceu sem bloco: o chefe não teria ficha para editar")
	}
	if *nova.CreatureID == block.ID {
		t.Fatal("a linha nova aponta para o bloco da ORIGINAL — editar um mexeria no outro")
	}
	dup, err := f.s.queries.GetCampaignCreature(context.Background(), *nova.CreatureID)
	if err != nil {
		t.Fatalf("o bloco copiado não está no acervo: %v", err)
	}
	if dup.Block != `{`+blocoMinimo+`}` {
		t.Errorf("o bloco copiado veio diferente do original:\n%s", dup.Block)
	}
	// O NOME do bloco acompanha o da linha, e este é o ponto: com dois "Zumbi"
	// no acervo, o olho da fila abriria um bloco chamado como o outro.
	if dup.Name != nova.Label {
		t.Errorf("o bloco se chama %q e a linha %q — dois nomes para a mesma criatura",
			dup.Name, nova.Label)
	}
	if dup.Campaignid != f.campaignID {
		t.Errorf("o bloco copiado caiu na campanha %d", dup.Campaignid)
	}
}

// Herói e NPC digitado à mão não têm bloco. O menu já esconde o verbo nesses
// casos; a trava é do servidor, e o botão escondido nunca foi prova de trava.
func TestTheOwnBlockModeRefusesWhoHasNone(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	id, _ := tokenOnTheQueue(t, f, "Ogro cansado")

	refusal := f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/bloco", "")
	if !strings.Contains(refusal, "não tem bloco de criatura") {
		t.Errorf("o modo do bloco aceitou uma linha sem bloco:\n%s", refusal)
	}
	// O CONTROLE: o "com PV próprio", na MESMA peça, passa. Sem ele a recusa
	// acima seria verdade também numa rota que recusa tudo.
	if rec := f.requests(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/sozinha", ""); rec.Code != http.StatusOK {
		t.Fatalf("o modo com PV próprio também foi recusado: %d", rec.Code)
	}
}

// DUPLICAR COM LINHA PRÓPRIA NÃO DEIXA LINHA ÓRFÃ NA FILA (ALE-376).
//
// Este gesto escreve nos DOIS donos de dado, e na ordem INVERSA à do movimento:
// a linha entra na FILA primeiro e a peça nasce no TABULEIRO depois. Em duas
// transações, uma falha na segunda deixava **um combatente na fila sem peça no
// mapa** — e tirá-lo é um gesto que o mestre não sabe que precisa fazer, porque
// nada na tela diz que aquela linha não tem corpo.
//
// # A sabotagem é do outro lado
//
// No caso do movimento, o gatilho aborta o UPDATE da `sessions`; aqui ele tem
// de abortar o do `open_boards`, que é o segundo passo. É o que faz este caso
// medir a MESMA garantia pela outra ponta: com a ordem invertida, quem tem de
// ser desfeito é o primeiro.
func TestDuplicatingWithAFailedBoardWriteLeavesNoOrphanLine(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	id, _ := tokenOnTheQueue(t, f, "Ogro cansado")

	// O CONTROLE: com o disco saudável o gesto passa e a fila cresce.
	before := len(stateOf(t, f.s.sessions, f.sessionID).Initiative)
	if rec := f.requests(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/sozinha", ""); rec.Code != http.StatusOK {
		t.Fatalf("o controle falhou: duplicar deu %d", rec.Code)
	}
	if now := len(stateOf(t, f.s.sessions, f.sessionID).Initiative); now != before+1 {
		t.Fatalf("o controle falhou: a fila foi de %d para %d", before, now)
	}

	if _, err := f.s.db.Exec(`CREATE TRIGGER o_tabuleiro_recusa BEFORE UPDATE ON open_boards
		BEGIN SELECT RAISE(ABORT, 'o disco recusou a gravação do tabuleiro'); END;`); err != nil {
		t.Fatalf("armar o gatilho: %v", err)
	}

	queueBefore := len(stateOf(t, f.s.sessions, f.sessionID).Initiative)
	rec := f.requests(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+id+"/duplicar/sozinha", "")

	if now := len(stateOf(t, f.s.sessions, f.sessionID).Initiative); now != queueBefore {
		t.Errorf("a peça não pôde ser gravada e a fila cresceu de %d para %d — "+
			"sobrou um combatente sem peça no mapa", queueBefore, now)
	}
	if !strings.Contains(rec.Body.String(), "recusou a gravação do tabuleiro") {
		t.Errorf("o gesto foi recusado e a cena não disse nada: %.200s", rec.Body.String())
	}
}
