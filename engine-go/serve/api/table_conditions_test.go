package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/catalog"
	"t20engine/domain/sheet"
)

func rowConditions(t *testing.T, f sceneFixture, entryID string) []string {
	t.Helper()
	for _, e := range f.s.tableHost().Sessions().GetState(f.sessionID).Initiative {
		if e.ID == entryID {
			return e.Conditions
		}
	}
	t.Fatalf("o combatente %q sumiu da fila", entryID)
	return nil
}

// O defeito que estava na tela.
//
// O crachá desenhava o `id` cru e o `uppercase` do CSS disfarçava: 31 das 35
// condições saem iguais em maiúsculas, e as outras quatro apareciam como CAIDO,
// EM-CHAMAS, IMOVEL e VULNERAVEL. Identificador não é texto de gente, e uma tela
// que imprime id erra só onde o id não coincide — que é a forma mais fácil de
// não se notar.
//
// A escolha da condição é DELIBERADA: `caido` é uma das quatro em que id e nome
// divergem. Uma que coincidisse passaria verde sobre o defeito.
func TestTheBadgeSaysTheBookWordAndNotTheId(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	_, npc := sceneIds(t, f)

	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/iniciativa/"+npc+"/condicao/caido", ""); rec.Code != http.StatusOK {
		t.Fatalf("aplicar deu %d", rec.Code)
	}

	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	// O SELETOR É O CRACHÁ (`</li>`) e não a palavra solta, e esta linha custou
	// uma sabotagem: procurar "Caído" na página passava VERDE com o crachá
	// imprimindo o id, porque o DIÁLOGO lista as 35 condições pelo nome — a
	// palavra estava lá, só não no lugar medido. É a terceira vez que substring
	// comum mente neste repositório; asserção de tela pede âncora de elemento.
	if !strings.Contains(screen, ">Caído</li>") {
		t.Error(`o crachá da fila não diz "Caído" — voltou a imprimir o id`)
	}
	// E o EFEITO viaja no `title`, porque a condição aqui é rastreio e não
	// regra: quem aplica é o mestre, e ele precisa ler o que ela faz.
	//
	// O texto sai do CATÁLOGO e não é transcrito aqui: eu tinha escrito de
	// cabeça "O personagem cai no chão" e o guarda ficou vermelho contra a
	// descrição de verdade. Esperado escrito à mão sobre dado que existe é
	// convite a testar a minha memória em vez do app.
	if !strings.Contains(screen, conditionEffectOf("caido")) {
		t.Error("o crachá não carrega o efeito da condição")
	}
	// O CONTROLE: o efeito não é string vazia, senão a asserção acima é
	// verdadeira sobre qualquer página.
	if conditionEffectOf("caido") == "" {
		t.Fatal("o catálogo não tem efeito para `caido` — o guarda acima não mede nada")
	}
}

// O clique ALTERNA a condição.
//
// O clique carrega a INTENÇÃO ("mexe nesta") e não o conjunto: quem monta a
// lista nova é o servidor, lendo a atual. Uma tela que mandasse o conjunto
// inteiro apagaria a condição que outro remendo acabou de acrescentar.
func TestTogglingTurnsTheConditionOnAndOff(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	_, npc := sceneIds(t, f)
	base := f.tableUrl() + "/iniciativa/" + npc + "/condicao/"

	if rec := f.pede(t, f.gm, http.MethodPost, base+"abalado", ""); rec.Code != http.StatusOK {
		t.Fatalf("ligar deu %d", rec.Code)
	}
	if c := rowConditions(t, f, npc); len(c) != 1 || c[0] != "abalado" {
		t.Fatalf("depois de ligar a linha tem %v", c)
	}
	// Uma SEGUNDA condição não substitui a primeira.
	if rec := f.pede(t, f.gm, http.MethodPost, base+"caido", ""); rec.Code != http.StatusOK {
		t.Fatalf("ligar a segunda deu %d", rec.Code)
	}
	if c := rowConditions(t, f, npc); len(c) != 2 {
		t.Errorf("a segunda condição substituiu a primeira: %v", c)
	}
	// E o mesmo clique DESLIGA, sem tocar na vizinha.
	if rec := f.pede(t, f.gm, http.MethodPost, base+"abalado", ""); rec.Code != http.StatusOK {
		t.Fatalf("desligar deu %d", rec.Code)
	}
	c := rowConditions(t, f, npc)
	if len(c) != 1 || c[0] != "caido" {
		t.Errorf("desligar levou a vizinha junto: %v", c)
	}
}

// O conjunto novo volta no SINAL.
//
// Sem isto o diálogo aberto MENTE: os crachás dele são pintados a partir do
// sinal que a abertura escreveu, e depois de um clique aquele sinal descreve o
// estado de antes. O mestre aplicaria "abalado", veria o crachá apagado, e
// clicaria de novo — tirando o que acabou de pôr.
func TestTheNewSetComesBackInTheSignal(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	_, npc := sceneIds(t, f)

	body := f.posta(t, f.gm, f.tableUrl()+"/iniciativa/"+npc+"/condicao/abalado", "")

	if !strings.Contains(body, `"row_conditions":"abalado"`) {
		t.Errorf("o conjunto novo não voltou no sinal; resposta: %.300s", body)
	}
}

// A validação é do CATÁLOGO e não de uma lista escrita à mão: uma lista assim já
// teve 34 ids para as 35 do catálogo, e a que faltava dava 400 ao ser aplicada.
// Cópia da tabela do livro é cópia que desvia.
func TestAnInventedConditionIsRefusedWithThePage(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	_, npc := sceneIds(t, f)

	body := f.posta(t, f.gm, f.tableUrl()+"/iniciativa/"+npc+"/condicao/maldicao-inventada", "")

	if !strings.Contains(body, "p394-395") {
		t.Errorf("a recusa não cita a página da tabela; resposta: %.300s", body)
	}
	if c := rowConditions(t, f, npc); len(c) != 0 {
		t.Errorf("a condição inventada entrou na linha: %v", c)
	}

	// O CONTROLE: a condição que a lista à mão esquecia É aceita. Sem ele,
	// "recusou a inventada" seria verdade também numa validação que recusa tudo.
	if !catalog.IsCondition("enfeiticado") {
		t.Fatal("o catálogo não tem `enfeiticado` — o controle está medindo outra coisa")
	}
	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/iniciativa/"+npc+"/condicao/enfeiticado", ""); rec.Code != http.StatusOK {
		t.Errorf("a condição do livro foi recusada: %d", rec.Code)
	}
}

// A trava é do servidor.
func TestThePlayerDoesNotApplyAConditionButton(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	_, npc := sceneIds(t, f)

	rec := f.pede(t, f.player, http.MethodPost,
		f.tableUrl()+"/iniciativa/"+npc+"/condicao/abalado", "")

	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador aplicou condição: %d", rec.Code)
	}
	if c := rowConditions(t, f, npc); len(c) != 0 {
		t.Errorf("a condição entrou apesar do 403: %v", c)
	}
}

// Por AMOSTRAGEM sobre o catálogo.
//
// A condição que entrar no livro amanhã já nasce oferecida — não há uma lista
// aqui para alguém esquecer de atualizar.
func TestTheDialogOffersTheCatalogConditions(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)

	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	conditions := book.Catalogs().Conditions
	if len(conditions) == 0 {
		t.Fatal("o catálogo não tem condição nenhuma — o laço abaixo não mediria nada")
	}
	for _, c := range conditions {
		if !strings.Contains(screen, c.Name) {
			t.Errorf("a condição %q (%s) não é oferecida na Mesa", c.Name, c.ID)
		}
	}
}

// conditionEffectOf lê o efeito no LIVRO, e não pelo ajudante da cena.
//
// A diferença importa: chamar o `conditionEffect` de `web/table` faria o
// esperado sair do código sob teste, e os dois andariam juntos com o defeito. O
// catálogo é a fonte dos dois lados, e é dele que a asserção lê.
func conditionEffectOf(id string) string {
	for _, c := range book.Catalogs().Conditions {
		if c.ID == id {
			return c.Description
		}
	}
	return ""
}

// sheetConditions lê as condições gravadas NA FICHA.
func sheetConditions(t *testing.T, f sceneFixture, charID int64) []string {
	t.Helper()
	row, err := f.s.queries.GetCharacter(t.Context(), charID)
	if err != nil {
		t.Fatalf("ler a ficha %d: %v", charID, err)
	}
	return sheet.UnmarshalStrings(row.Activeconditions)
}

// NA LINHA DE PERSONAGEM, A MESA GRAVA NA FICHA (decisão do dono, ALE-368).
//
// A condição de um personagem mora na ficha — é lá que o motor a lê para mexer
// na Defesa e no instante de agir. A fila guardava uma lista própria também
// para PC, e o Atordoado marcado pela Mesa não atordoava ninguém.
func TestMarkingAConditionOnACharacterRowWritesTheSheet(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	pc, _ := sceneIds(t, f)
	base := f.tableUrl() + "/iniciativa/" + pc + "/condicao/"

	if rec := f.pede(t, f.gm, http.MethodPost, base+"atordoado", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar deu %d", rec.Code)
	}
	if got := sheetConditions(t, f, f.charID); len(got) != 1 || got[0] != "atordoado" {
		t.Fatalf("a Mesa marcou Atordoado e a ficha tem %v", got)
	}
	if got := rowConditions(t, f, pc); len(got) != 0 {
		t.Errorf("a linha do PC guardou %v na fila: a condição dele mora na ficha", got)
	}
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(screen, ">Atordoado</li>") {
		t.Error("a fila não mostra o crachá da condição que está na ficha")
	}

	// O mesmo gesto desliga, na ficha.
	if rec := f.pede(t, f.gm, http.MethodPost, base+"atordoado", ""); rec.Code != http.StatusOK {
		t.Fatalf("desmarcar deu %d", rec.Code)
	}
	if got := sheetConditions(t, f, f.charID); len(got) != 0 {
		t.Errorf("o segundo toque não tirou a condição da ficha: %v", got)
	}
}

// A FILA MOSTRA O QUE A FICHA TEM, venha de onde vier: o jogador que se marca
// Caído pela aba Efeitos aparece Caído na Mesa.
func TestTheTableRowShowsTheSheetConditions(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	if rec := f.pede(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/efeitos/condicao/caido?tab=conditionals", f.charID), ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar pela ficha deu %d", rec.Code)
	}
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(screen, ">Caído</li>") {
		t.Error("a ficha está Caída e a fila da Mesa não mostra o crachá")
	}
}

// O NPC CONTINUA NA FILA: ele não tem ficha, e a lista da linha é o lugar dele.
func TestMarkingAConditionOnAnNPCRowStaysOnTheQueue(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	_, npc := sceneIds(t, f)
	if rec := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/"+npc+"/condicao/abalado", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar deu %d", rec.Code)
	}
	if got := rowConditions(t, f, npc); len(got) != 1 || got[0] != "abalado" {
		t.Errorf("o NPC marcado Abalado tem %v na linha", got)
	}
}

// O ATORDOADO MARCADO PELA MESA VALE PARA A REGRA: o personagem "não pode fazer
// ações" (p394), e a conjuração de ação padrão na vez dele é recusada. Era isto
// que a lista própria da fila deixava passar — ela não mexia em número nenhum.
func TestAStunMarkedAtTheTableStopsTheCharacterFromActing(t *testing.T) {
	f := newSceneFixture(t)
	startCombatWithSomeoneElseOnTurn(t, f)
	if _, err := f.s.sessions.NextTurn(f.sessionID); err != nil {
		t.Fatalf("passar a vez ao personagem: %v", err)
	}
	var pc string
	for _, e := range f.s.sessions.GetState(f.sessionID).Initiative {
		if e.CharacterID != nil {
			pc = e.ID
		}
	}
	// O CONTROLE: de pé e na vez, a mesma magia sai.
	if rec := learnAndCast(t, f, "luz"); sceneRefusal(rec.Body) != "" {
		t.Fatalf("o controle falhou: a magia foi recusada antes do Atordoado: %q", sceneRefusal(rec.Body))
	}
	if _, err := f.s.sessions.NextTurn(f.sessionID); err != nil { // volta ao goblin
		t.Fatalf("girar: %v", err)
	}
	if _, err := f.s.sessions.NextTurn(f.sessionID); err != nil { // e ao personagem, com o turno inteiro
		t.Fatalf("girar: %v", err)
	}
	if rec := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/"+pc+"/condicao/atordoado", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar Atordoado deu %d", rec.Code)
	}
	refusal := sceneRefusal(learnAndCast(t, f, "luz").Body)
	if !strings.Contains(refusal, "não dá para agir") {
		t.Errorf("atordoado pela Mesa, o personagem conjurou mesmo assim (recusa: %q)", refusal)
	}
}
