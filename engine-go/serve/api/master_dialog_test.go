package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sinalQueAbre é o `data-signals` que o SERVIDOR redeclara no mesmo remendo do
// conteúdo, e não a palavra solta: procurar o nome do sinal cru acha o
// `data-show` do diálogo, que está sempre lá, e o guarda passa verde afirmando
// uma ordem que nunca mediu.
const sinalQueAbre = `sheet_open: true`
const sinalQueFecha = `sheet_open: false`

// O guarda da FICHA que abre na hora certa.
//
// Clicar numa linha NÃO selecionada abria a ficha na hora com a criatura
// ANTERIOR e trocava um quadro adiante (a 0ms "Bandido", a 16ms "Lobo"); na
// linha JÁ selecionada não piscava, porque o conteúdo já estava certo.
//
// A garantia é sobre ORDEM no fluxo, e é por isso que ela cabe num teste de
// handler: o conteúdo tem de sair ANTES do sinal que abre. Invertido, a ficha
// aparece com a criatura velha — e nenhum teste de "abriu?" pegaria isso.

func fluxoDaFicha(t *testing.T, f sceneFixture, target string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.gm))
	req.Header.Set("datastar-request", "true")
	rec := httptest.NewRecorder()
	f.s.WebRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s deu %d", target, rec.Code)
	}
	return rec.Body.String()
}

// A garantia é de ATOMICIDADE e não de ordem: mandar um EVENTO DE SINAL depois
// do conteúdo não funciona, porque o `data-signals` que abre a ficha mora no
// `#bestiary`, que É o elemento remendado — o remendo redeclara
// `sheet_open: false` por cima, o fio leva `{"sheet_open":true}` e o diálogo
// continua `display:none`. Com o servidor redeclarando o valor CERTO, o conteúdo
// e o estado de aberto chegam juntos e não existe janela entre eles.
func TestTheEntryCardIsBornOpenInTheSamePatchAsItsContent(t *testing.T) {
	f := newSceneFixture(t)
	body := fluxoDaFicha(t, f, "/mestre/bestiario?criatura=lobo&abrir=1")

	if !strings.Contains(body, "datastar-patch-elements") {
		t.Fatal("o fluxo não trouxe conteúdo nenhum — o guarda mediria a resposta errada")
	}
	if !strings.Contains(body, sinalQueAbre) {
		t.Errorf("a ficha não nasce aberta: o remendo declara %q", sinalQueFecha)
	}
	if strings.Contains(body, sinalQueFecha) {
		t.Error("o remendo redeclara a ficha FECHADA por cima: ela não abriria, ou abriria e fecharia")
	}
	// E o conteúdo é o da criatura PEDIDA, não o de qualquer uma.
	if !strings.Contains(body, "Lobo") {
		t.Error("o fluxo não trouxe a criatura escolhida")
	}
}

// A metade que faz a de cima significar alguma coisa. A MESMA rota serve a
// busca e os filtros de tipo, e os dois mandam os sinais TODOS — inclusive o
// `criatura` já escolhido. Se a decisão de abrir viesse de um sinal em vez da
// URL, digitar uma letra na busca abriria a ficha por cima da lista, a cada
// tecla.
func TestSearchAndFilterDoNotOpenTheEntryCard(t *testing.T) {
	f := newSceneFixture(t)

	// O CONTROLE: com `abrir` o sinal SAI. Sem ele, "não abriu" seria verdade
	// também sobre uma rota quebrada que não responde nada.
	withOpen := fluxoDaFicha(t, f, "/mestre/bestiario?criatura=lobo&abrir=1")
	if !strings.Contains(withOpen, sinalQueAbre) {
		t.Fatal("nem com abrir=1 a ficha abre — o guarda abaixo não mediria nada")
	}

	noOpen := fluxoDaFicha(t, f, "/mestre/bestiario?criatura=lobo&busca=lo")
	if strings.Contains(noOpen, sinalQueAbre) {
		t.Error("buscar abriu a ficha: a cada tecla o diálogo saltaria por cima da lista")
	}
}

// A regressão silenciosa deste conserto: devolver `$sheet_open = true` à
// expressão do clique faz a ficha voltar a abrir antes do conteúdo, e nada
// estoura — o defeito reaparece como um quadro piscando, que é o que ninguém
// atribui a um commit.
func TestClickingTheRowDoesNotOpenTheEntryCardOnItsOwn(t *testing.T) {
	f := newSceneFixture(t)
	screen := f.pede(t, f.gm, http.MethodGet, "/mestre/bestiario", "").Body.String()

	if !strings.Contains(screen, "criatura=") {
		t.Fatal("a lista não desenhou — o guarda mediria a tela errada")
	}
	if strings.Contains(screen, "$sheet_open = true") {
		t.Error("o clique abre a ficha pelo cliente: ela aparece com a criatura anterior por um quadro")
	}
	if !strings.Contains(screen, "abrir=1") {
		t.Error("o clique não pede ao servidor para abrir a ficha")
	}
}
