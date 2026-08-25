package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

// fluxoDaFicha pede como o Datastar pede: com o cabeçalho que faz o handler
// responder FLUXO em vez de página. Sem ele a rota devolve o documento inteiro e
// a asserção sobre ordem de eventos não teria eventos para ordenar.
// sinalQueAbre é o `data-signals` que o SERVIDOR redeclara no mesmo remendo do
// conteúdo, e não a palavra solta: procurar `fichaAberta` cru acha o `data-show`
// do diálogo, que está sempre lá. A primeira versão deste guarda procurava a
// palavra, achava o atributo, e passava verde afirmando uma ordem que nunca
// mediu — quem denunciou foi o guarda da busca, falhando pelo mesmo motivo.
const sinalQueAbre = `fichaAberta: true`
const sinalQueFecha = `fichaAberta: false`

func fluxoDaFicha(t *testing.T, f pilotoFixture, alvo string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, alvo, nil)
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.mestre))
	req.Header.Set("datastar-request", "true")
	rec := httptest.NewRecorder()
	http.StripPrefix("/piloto", f.s.PilotoRouter()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s deu %d", alvo, rec.Code)
	}
	return rec.Body.String()
}

// TestAfichaNASCEabertaNOmesmoREMENDOdoConteudo.
//
// A garantia é de ATOMICIDADE e não de ordem, e a diferença foi medida: a
// primeira versão do conserto mandava um EVENTO DE SINAL depois do conteúdo, e
// ele não funcionava — o `data-signals` que abre a ficha mora no `#bestiario`,
// que É o elemento remendado, e o remendo redeclarava `fichaAberta: false` por
// cima. O fio levava `{"fichaAberta":true}` e o diálogo continuava
// `display:none`. Com o servidor redeclarando o valor CERTO, o conteúdo e o
// estado de aberto chegam juntos e não existe janela entre eles.
func TestAFichaNasceAbertaNoMesmoRemendoDoConteudo(t *testing.T) {
	f := novoPiloto(t)
	corpo := fluxoDaFicha(t, f, "/piloto/mestre/bestiario?criatura=lobo&abrir=1")

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

// TestBUSCAeFILTROnaoABREMaFicha.
//
// A metade que faz a de cima significar alguma coisa. A MESMA rota serve a
// busca e os filtros de tipo, e os dois mandam os sinais TODOS — inclusive o
// `criatura` já escolhido. Se a decisão de abrir viesse de um sinal em vez da
// URL, digitar uma letra na busca abriria a ficha por cima da lista, a cada
// tecla.
func TestBuscaEFiltroNaoAbremAFicha(t *testing.T) {
	f := novoPiloto(t)

	// O CONTROLE: com `abrir` o sinal SAI. Sem ele, "não abriu" seria verdade
	// também sobre uma rota quebrada que não responde nada.
	comAbrir := fluxoDaFicha(t, f, "/piloto/mestre/bestiario?criatura=lobo&abrir=1")
	if !strings.Contains(comAbrir, sinalQueAbre) {
		t.Fatal("nem com abrir=1 a ficha abre — o guarda abaixo não mediria nada")
	}

	semAbrir := fluxoDaFicha(t, f, "/piloto/mestre/bestiario?criatura=lobo&busca=lo")
	if strings.Contains(semAbrir, sinalQueAbre) {
		t.Error("buscar abriu a ficha: a cada tecla o diálogo saltaria por cima da lista")
	}
}

// TestOcliqueNAlinhaNAOabreAfichaSOZINHO.
//
// A regressão silenciosa deste conserto: devolver `$fichaAberta = true` à
// expressão do clique faz a ficha voltar a abrir antes do conteúdo, e nada
// estoura — o defeito reaparece como um quadro piscando, que é o que ninguém
// atribui a um commit.
func TestOCliqueNaLinhaNaoAbreAFichaSozinho(t *testing.T) {
	f := novoPiloto(t)
	tela := f.pede(t, f.mestre, http.MethodGet, "/piloto/mestre/bestiario", "").Body.String()

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
