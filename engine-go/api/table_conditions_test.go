package api

import (
	"net/http"
	"strings"
	"t20engine/book"
	"t20engine/catalog"
	"testing"
)

func rowConditions(t *testing.T, f pilotoFixture, entryID string) []string {
	t.Helper()
	for _, e := range f.s.tableHost().Sessions().GetState(f.sessionID).Initiative {
		if e.ID == entryID {
			return e.Conditions
		}
	}
	t.Fatalf("o combatente %q sumiu da fila", entryID)
	return nil
}

// TestTheBadgeSaysTheBookWordAndNotTheId — o defeito que estava na tela.
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
	f := novoPiloto(t)
	f.scene(t)
	_, npc := sceneIds(t, f)

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/initiative/"+npc+"/condicao/caido", ""); rec.Code != http.StatusOK {
		t.Fatalf("aplicar deu %d", rec.Code)
	}

	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	// O SELETOR É O CRACHÁ (`</li>`) e não a palavra solta, e esta linha custou
	// uma sabotagem: procurar "Caído" na página passava VERDE com o crachá
	// imprimindo o id, porque o DIÁLOGO lista as 35 condições pelo nome — a
	// palavra estava lá, só não no lugar medido. É a terceira vez que substring
	// comum mente neste repositório; asserção de tela pede âncora de elemento.
	if !strings.Contains(tela, ">Caído</li>") {
		t.Error(`o crachá da fila não diz "Caído" — voltou a imprimir o id`)
	}
	// E o EFEITO viaja no `title`, porque a condição aqui é rastreio e não
	// regra: quem aplica é o mestre, e ele precisa ler o que ela faz.
	//
	// O texto sai do CATÁLOGO e não é transcrito aqui: eu tinha escrito de
	// cabeça "O personagem cai no chão" e o guarda ficou vermelho contra a
	// descrição de verdade. Esperado escrito à mão sobre dado que existe é
	// convite a testar a minha memória em vez do app.
	if !strings.Contains(tela, conditionEffectOf("caido")) {
		t.Error("o crachá não carrega o efeito da condição")
	}
	// O CONTROLE: o efeito não é string vazia, senão a asserção acima é
	// verdadeira sobre qualquer página.
	if conditionEffectOf("caido") == "" {
		t.Fatal("o catálogo não tem efeito para `caido` — o guarda acima não mede nada")
	}
}

// TestTogglingTurnsTheConditionOnAndOff.
//
// O clique carrega a INTENÇÃO ("mexe nesta") e não o conjunto: quem monta a
// lista nova é o servidor, lendo a atual. Uma tela que mandasse o conjunto
// inteiro apagaria a condição que outro remendo acabou de acrescentar.
func TestTogglingTurnsTheConditionOnAndOff(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	_, npc := sceneIds(t, f)
	base := f.tableUrl() + "/initiative/" + npc + "/condicao/"

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"abalado", ""); rec.Code != http.StatusOK {
		t.Fatalf("ligar deu %d", rec.Code)
	}
	if c := rowConditions(t, f, npc); len(c) != 1 || c[0] != "abalado" {
		t.Fatalf("depois de ligar a linha tem %v", c)
	}
	// Uma SEGUNDA condição não substitui a primeira.
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"caido", ""); rec.Code != http.StatusOK {
		t.Fatalf("ligar a segunda deu %d", rec.Code)
	}
	if c := rowConditions(t, f, npc); len(c) != 2 {
		t.Errorf("a segunda condição substituiu a primeira: %v", c)
	}
	// E o mesmo clique DESLIGA, sem tocar na vizinha.
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"abalado", ""); rec.Code != http.StatusOK {
		t.Fatalf("desligar deu %d", rec.Code)
	}
	c := rowConditions(t, f, npc)
	if len(c) != 1 || c[0] != "caido" {
		t.Errorf("desligar levou a vizinha junto: %v", c)
	}
}

// TestTheNewSetComesBackInTheSignal.
//
// Sem isto o diálogo aberto MENTE: os crachás dele são pintados a partir do
// sinal que a abertura escreveu, e depois de um clique aquele sinal descreve o
// estado de antes. O mestre aplicaria "abalado", veria o crachá apagado, e
// clicaria de novo — tirando o que acabou de pôr.
func TestTheNewSetComesBackInTheSignal(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	_, npc := sceneIds(t, f)

	corpo := f.posta(t, f.mestre, f.tableUrl()+"/initiative/"+npc+"/condicao/abalado", "")

	if !strings.Contains(corpo, `"condicoesdalinha":"abalado"`) {
		t.Errorf("o conjunto novo não voltou no sinal; resposta: %.300s", corpo)
	}
}

// TestAnInventedConditionIsRefusedWithThePage.
//
// A validação é do CATÁLOGO e não de uma lista escrita aqui: a API já teve 34
// ids ao lado das 35 do catálogo, e a que faltava — `enfeitiçado` — dava 400 ao
// ser aplicada (ALE-122). Uma cópia da tabela do livro é uma cópia que desvia.
func TestAnInventedConditionIsRefusedWithThePage(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	_, npc := sceneIds(t, f)

	corpo := f.posta(t, f.mestre, f.tableUrl()+"/initiative/"+npc+"/condicao/maldicao-inventada", "")

	if !strings.Contains(corpo, "p394-395") {
		t.Errorf("a recusa não cita a página da tabela; resposta: %.300s", corpo)
	}
	if c := rowConditions(t, f, npc); len(c) != 0 {
		t.Errorf("a condição inventada entrou na linha: %v", c)
	}

	// O CONTROLE, e ele é o que a ALE-122 pagou: a condição que a lista à mão
	// esquecia É aceita. Sem ele, "recusou a inventada" seria verdade também
	// numa validação que recusa tudo.
	if !catalog.IsCondition("enfeitiçado") {
		t.Fatal("o catálogo não tem `enfeitiçado` — o controle está medindo outra coisa")
	}
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/initiative/"+npc+"/condicao/enfeitiçado", ""); rec.Code != http.StatusOK {
		t.Errorf("a condição do livro foi recusada: %d", rec.Code)
	}
}

// TestThePlayerDoesNotApplyAConditionButton — a trava é do servidor.
func TestThePlayerDoesNotApplyAConditionButton(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	_, npc := sceneIds(t, f)

	rec := f.pede(t, f.jogador, http.MethodPost,
		f.tableUrl()+"/initiative/"+npc+"/condicao/abalado", "")

	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador aplicou condição: %d", rec.Code)
	}
	if c := rowConditions(t, f, npc); len(c) != 0 {
		t.Errorf("a condição entrou apesar do 403: %v", c)
	}
}

// TestTheDialogOffersTheCatalogConditions, por AMOSTRAGEM sobre o catálogo.
//
// A condição que entrar no livro amanhã já nasce oferecida — não há uma lista
// aqui para alguém esquecer de atualizar.
func TestTheDialogOffersTheCatalogConditions(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)

	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	condicoes := book.Catalogs().Condicoes
	if len(condicoes) == 0 {
		t.Fatal("o catálogo não tem condição nenhuma — o laço abaixo não mediria nada")
	}
	for _, c := range condicoes {
		if !strings.Contains(tela, c.Name) {
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
	for _, c := range book.Catalogs().Condicoes {
		if c.ID == id {
			return c.Description
		}
	}
	return ""
}
