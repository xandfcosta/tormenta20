package api

import (
	"strings"
	"t20engine/book"
	"testing"
)

// O guarda dos três catálogos DO PERSONAGEM (ALE-264).

// TestOsModificadoresSaemNaOrdemDoLivro.
//
// O defeito que ele prende não é de conteúdo, é de DETERMINISMO: os
// modificadores vêm num `map[string]int`, e a ordem de um mapa em Go é aleatória
// por projeto. Imprimir direto do mapa daria uma ordem diferente a cada render —
// a mesma página mudando sozinha entre dois pedidos iguais, e qualquer teste
// sobre o texto ficaria intermitente.
//
// Rodar cem vezes é o que torna a asserção honesta: uma passada só teria 1/6 de
// chance de pegar a ordem errada por acaso.
func TestOsModificadoresSaemNaOrdemDoLivro(t *testing.T) {
	elfo := book.RaceAttribute{
		Kind: "fixed",
		Mods: map[string]int{"intelligence": 2, "dexterity": 1, "constitution": -1},
	}
	const esperado = "+1 Des, -1 Con, +2 Int"
	for i := 0; i < 100; i++ {
		if escrito := elfo.Escrito(); escrito != esperado {
			t.Fatalf("volta %d: %q — a ordem do livro é For, Des, Con, Int, Sab, Car", i, escrito)
		}
	}
}

// TestAEscolhaLivreNaoViraTresAtributosInventados: as duas formas do livro.
func TestAEscolhaLivreNaoViraTresAtributosInventados(t *testing.T) {
	humano := book.RaceAttribute{Kind: "floating", Count: 3, Value: 1}
	if escrito := humano.Escrito(); escrito != "+1 em três atributos" {
		t.Errorf("o humano escolhe onde põe os três +1, e a tela disse %q", escrito)
	}
}

// TestAsTresAbasNovasDesenhamOQueASuaEntradaTem.
//
// INTEGRAÇÃO e não asserção de componente porque o que se protege é a
// composição: catálogo novo (`classes.json`), tipo novo, aba nova e cartão novo
// — quatro saltos, e a cena passaria a existir vazia se qualquer um falhasse,
// sem erro nenhum.
func TestAsTresAbasNovasDesenhamOQueASuaEntradaTem(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	casos := []struct{ aba, esperado, porque string }{
		{"racas", "Graça de Glórienn", "a habilidade de raça vem do catálogo"},
		{"racas", "+2 Int", "o modificador de atributo é o que muda numa ficha"},
		{"classes", "poderes de classe", "a conta de poderes é derivada, não transcrita"},
		{"classes", "Treinado em", "as perícias saem de class-expertises"},
		{"deuses", "Arma preferida", "o clérigo saca a arma preferida na cena"},
		{"deuses", "Concede", "os poderes concedidos são o que o mestre consulta"},
	}
	for _, caso := range casos {
		corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/"+caso.aba, "").Body.String()
		if !strings.Contains(corpo, caso.esperado) {
			t.Errorf("a aba %q não traz %q — %s", caso.aba, caso.esperado, caso.porque)
		}
	}
}

// TestORotuloVazioNaoSaiSozinho: Lena e Marah não têm arma preferida no livro.
//
// Provado VERMELHO antes do conserto: a cena escrevia "Arma preferida:" seguido
// de nada, que parece dado perdido em vez de ausência com significado.
func TestORotuloVazioNaoSaiSozinho(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/deuses", "").Body.String()
	if strings.Contains(corpo, "Arma preferida: </span>") {
		t.Error("um rótulo saiu sem valor — Lena e Marah não têm arma preferida")
	}
	// O CONTROLE: quem TEM arma preferida continua mostrando.
	if !strings.Contains(corpo, "Arma preferida: Martelo de guerra") {
		t.Error("o rótulo sumiu de quem tem valor — o guarda acima passaria por ausência de tudo")
	}
}

// TestABuscaUnificadaAlcancaOsTresNovos: a busca sem aba varre os SETE.
func TestABuscaUnificadaAlcancaOsTresNovos(t *testing.T) {
	v := carregaCatalogos(criteriosDoAcervo{Busca: "allihanna", Aba: ""}, enderecoDoLivro{})
	if v.Achados == 0 {
		t.Fatal("a busca unificada não achou a deusa Allihanna")
	}
	achouODeus := false
	for _, g := range v.Grupos {
		if g.Rotulo == "Deuses" && len(g.Deuses) > 0 {
			achouODeus = true
		}
	}
	if !achouODeus {
		t.Error("o grupo de deuses não entra na busca unificada")
	}
}
