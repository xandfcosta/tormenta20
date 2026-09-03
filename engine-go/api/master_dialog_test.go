package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sinalQueAbre é o `data-signals` que o SERVIDOR redeclara no mesmo remendo do
// conteúdo, e não a palavra solta: procurar `fichaAberta` cru acha o `data-show`
// do diálogo, que está sempre lá. A primeira versão deste guarda procurava a
// palavra, achava o atributo, e passava verde afirmando uma ordem que nunca
// mediu — quem denunciou foi o guarda da busca, falhando pelo mesmo motivo.
const sinalQueAbre = `fichaAberta: true`
const sinalQueFecha = `fichaAberta: false`

// O guarda da FICHA que abre na hora certa (ALE-264).
//
// O defeito foi visto pelo dono e medido depois: clicar numa linha NÃO
// selecionada abria a ficha na hora com a criatura ANTERIOR e trocava um quadro
// adiante. Amostrado no navegador — a 0ms dizia "Bandido", a 16ms dizia "Lobo".
// Clicar na linha JÁ selecionada não piscava, e foi essa diferença que isolou a
// causa: lá o conteúdo já estava certo, então não havia troca para ver.
//
// A garantia é sobre ORDEM no fluxo, e é por isso que ela cabe num teste de
// handler: o conteúdo tem de sair ANTES do sinal que abre. Invertido, a ficha
// aparece com a criatura velha — e nenhum teste de "abriu?" pegaria isso.

func fluxoDaFicha(t *testing.T, f pilotoFixture, alvo string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, alvo, nil)
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.mestre))
	req.Header.Set("datastar-request", "true")
	rec := httptest.NewRecorder()
	f.s.WebRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s deu %d", alvo, rec.Code)
	}
	return rec.Body.String()
}

// TestTheEntryCardIsBornOpenInTheSamePatchAsItsContent.
//
// A garantia é de ATOMICIDADE e não de ordem, e a diferença foi medida: a
// primeira versão do conserto mandava um EVENTO DE SINAL depois do conteúdo, e
// ele não funcionava — o `data-signals` que abre a ficha mora no `#bestiario`,
// que É o elemento remendado, e o remendo redeclarava `fichaAberta: false` por
// cima. O fio levava `{"fichaAberta":true}` e o diálogo continuava
// `display:none`. Com o servidor redeclarando o valor CERTO, o conteúdo e o
// estado de aberto chegam juntos e não existe janela entre eles.
func TestTheEntryCardIsBornOpenInTheSamePatchAsItsContent(t *testing.T) {
	f := novoPiloto(t)
	corpo := fluxoDaFicha(t, f, "/mestre/bestiario?criatura=lobo&abrir=1")

	if !strings.Contains(corpo, "datastar-patch-elements") {
		t.Fatal("o fluxo não trouxe conteúdo nenhum — o guarda mediria a resposta errada")
	}
	if !strings.Contains(corpo, sinalQueAbre) {
		t.Errorf("a ficha não nasce aberta: o remendo declara %q", sinalQueFecha)
	}
	if strings.Contains(corpo, sinalQueFecha) {
		t.Error("o remendo redeclara a ficha FECHADA por cima: ela não abriria, ou abriria e fecharia")
	}
	// E o conteúdo é o da criatura PEDIDA, não o de qualquer uma.
	if !strings.Contains(corpo, "Lobo") {
		t.Error("o fluxo não trouxe a criatura escolhida")
	}
}

// TestSearchAndFilterDoNotOpenTheEntryCard.
//
// A metade que faz a de cima significar alguma coisa. A MESMA rota serve a
// busca e os filtros de tipo, e os dois mandam os sinais TODOS — inclusive o
// `criatura` já escolhido. Se a decisão de abrir viesse de um sinal em vez da
// URL, digitar uma letra na busca abriria a ficha por cima da lista, a cada
// tecla.
func TestSearchAndFilterDoNotOpenTheEntryCard(t *testing.T) {
	f := novoPiloto(t)

	// O CONTROLE: com `abrir` o sinal SAI. Sem ele, "não abriu" seria verdade
	// também sobre uma rota quebrada que não responde nada.
	comAbrir := fluxoDaFicha(t, f, "/mestre/bestiario?criatura=lobo&abrir=1")
	if !strings.Contains(comAbrir, sinalQueAbre) {
		t.Fatal("nem com abrir=1 a ficha abre — o guarda abaixo não mediria nada")
	}

	semAbrir := fluxoDaFicha(t, f, "/mestre/bestiario?criatura=lobo&busca=lo")
	if strings.Contains(semAbrir, sinalQueAbre) {
		t.Error("buscar abriu a ficha: a cada tecla o diálogo saltaria por cima da lista")
	}
}

// TestClickingTheRowDoesNotOpenTheEntryCardOnItsOwn.
//
// A regressão silenciosa deste conserto: devolver `$fichaAberta = true` à
// expressão do clique faz a ficha voltar a abrir antes do conteúdo, e nada
// estoura — o defeito reaparece como um quadro piscando, que é o que ninguém
// atribui a um commit.
func TestClickingTheRowDoesNotOpenTheEntryCardOnItsOwn(t *testing.T) {
	f := novoPiloto(t)
	tela := f.pede(t, f.mestre, http.MethodGet, "/mestre/bestiario", "").Body.String()

	if !strings.Contains(tela, "criatura=") {
		t.Fatal("a lista não desenhou — o guarda mediria a tela errada")
	}
	if strings.Contains(tela, "$fichaAberta = true") {
		t.Error("o clique abre a ficha pelo cliente: ela aparece com a criatura anterior por um quadro")
	}
	if !strings.Contains(tela, "abrir=1") {
		t.Error("o clique não pede ao servidor para abrir a ficha")
	}
}

// TestNoFocusAsksTheServerWithoutAKeyboardGuard — a VARREDURA de um defeito
// que só aparece em máquina carregada, e que o CI pegou duas vezes seguidas
// enquanto a bancada passava verde (ALE-272).
//
// O clique do mouse TAMBÉM foca. Um nó que pede ao servidor no foco E no clique
// manda DOIS pedidos por um gesto só, e os dois remendam a mesma cena — que
// redeclara os sinais dela a cada remendo. Quem chega por último manda, e a
// ordem de chegada não é a de saída: no bestiário o pedido do foco não leva
// `abrir=1`, então chegando por último ele FECHAVA a ficha que o clique tinha
// aberto. A criatura ficava escolhida e a ficha não abria.
//
// É a família "duas escritas no mesmo lugar sem ordem garantida", prima do
// `data-show` com `data-attr:style` — e como aquela, não deixa erro nenhum
// para trás.
//
// A REGRA: pedido disparado por FOCO é afordância de TECLADO, e por isso ele
// pede `:focus-visible`. Medido no navegador: o clique dá `false`, o Tab dá
// `true`, e o foco PROGRAMÁTICO do driver de setas também dá `true` — o guarda
// preserva a prévia da seta e tira só o pedido que o mouse mandava à toa.
//
// Ele varre a FONTE e não uma cena servida de propósito. A lição da ALE-237 e
// da ALE-252 é que um guarda só mede o que ele VISITA, e enumerar cena por cena
// deixaria a próxima nascer sem medição, em silêncio. Como a regra cabe num
// atributo só, a fonte inteira é alcançável de uma vez.
