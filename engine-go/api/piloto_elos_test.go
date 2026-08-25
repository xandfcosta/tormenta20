package api

import (
	"strings"
	"testing"
)

// O guarda dos ELOS entre entradas (ALE-264).
//
// O que se protege é uma REDE: a condição cita o tipo de efeito, agrava para
// outra condição, e a descrição dela nomeia uma terceira. Cada elo é um `href`
// montado a partir de dois catálogos diferentes, e um deles mudar de nome
// quebraria o elo em silêncio — a palavra continuaria na tela, só que morta.

// TestOTipoDeEfeitoDaCondicaoVirouElo: o caso que o dono pediu com todas as
// letras — "a palavra Medo na página de catálogo é um link para mostrar o Medo".
func TestOTipoDeEfeitoDaCondicaoVirouElo(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/catalogos?aba=condicoes", "").Body.String()
	if !strings.Contains(corpo, "aba=efeitos&amp;busca=Medo") {
		t.Error("a tag da condição não leva ao tipo de efeito")
	}
	// E ela é escrita como o LIVRO escreve, não como o dado guarda: a tag é
	// `cansaco`, sem acento e em caixa baixa, porque é uma chave.
	if !strings.Contains(corpo, ">Medo<") {
		t.Error("a tag saiu com a cara de chave em vez do nome do tipo")
	}
	efeitos := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/catalogos?aba=efeitos", "").Body.String()
	if !strings.Contains(efeitos, "Medo capaz de prejudicar o alvo") {
		t.Error("o destino do elo não tem a definição do livro — o elo levaria a lugar nenhum")
	}
}

// TestACondicaoCitadaNaDescricaoVirouElo, e o que ela NÃO faz.
//
// O controle é a segunda metade: uma condição não vira elo para SI MESMA. Um
// elo que aponta para a página em que já se está é ruído com cara de saída, e
// era o que a primeira varredura fazia.
func TestACondicaoCitadaNaDescricaoVirouElo(t *testing.T) {
	pedacos := comElosParaCondicoes("Desprevenido e imóvel; -2 em ataques", "Agarrado")
	if len(pedacos) < 2 || pedacos[0].Texto != "Desprevenido" || pedacos[0].Aba != "condicoes" {
		t.Fatalf("a citação não virou elo: %+v", pedacos)
	}
	if pedacos[1].Aba != "" {
		t.Errorf("o resto da frase virou elo também: %+v", pedacos[1])
	}

	proprio := comElosParaCondicoes("Desprevenido e não pode fazer ações.", "Desprevenido")
	for _, p := range proprio {
		if p.Aba != "" {
			t.Errorf("a condição virou elo para si mesma: %+v", p)
		}
	}
}

// TestOEloRespeitaPalavraInteiraECaixa.
//
// Duas regras numa: no texto do livro a condição vem com MAIÚSCULA ("fica
// Abalado") e a palavra comum não ("um efeito de medo"). Casar sem caixa, ou
// casar pedaço de palavra, encheria a tela de elos que não são citação.
func TestOEloRespeitaPalavraInteiraECaixa(t *testing.T) {
	casos := []struct {
		texto  string
		espera bool
		porque string
	}{
		{"O alvo fica Abalado.", true, "citação de verdade"},
		{"imune a efeitos de medo", false, "palavra comum, caixa baixa"},
		{"Abaladocom", false, "pedaço de palavra maior"},
		{"deixa Abalados os inimigos", false, "plural não é o nome da condição"},
	}
	for _, caso := range casos {
		temElo := false
		for _, p := range comElosParaCondicoes(caso.texto, "") {
			if p.Aba != "" {
				temElo = true
			}
		}
		if temElo != caso.espera {
			t.Errorf("%q: elo=%v, esperado %v — %s", caso.texto, temElo, caso.espera, caso.porque)
		}
	}
}

// TestOsElosDoDeusSoApontamParaQuemTemVerbete.
//
// "Quaisquer" é devoto de Aharadak e não é raça nem classe; "Elfos" vem no
// PLURAL e a raça é "Elfo". Elo que aponta para o vazio é pior que texto puro:
// ele promete uma página que não existe.
func TestOsElosDoDeusSoApontamParaQuemTemVerbete(t *testing.T) {
	if aba := eloDoDevoto("Elfos"); aba != "racas" {
		t.Errorf("“Elfos” devia levar às raças, e levou a %q", aba)
	}
	if aba := eloDoDevoto("Bárbaros"); aba != "classes" {
		t.Errorf("“Bárbaros” devia levar às classes, e levou a %q", aba)
	}
	if aba := eloDoDevoto("Quaisquer"); aba != "" {
		t.Errorf("“Quaisquer” virou elo para %q", aba)
	}
	if !ehPoderConhecido("Coragem Total") {
		t.Error("um poder concedido de verdade não foi reconhecido")
	}
	if ehPoderConhecido("Poder Que Não Existe") {
		t.Error("um nome inventado passou por poder do acervo")
	}
}

// TestTodaTagDeCondicaoTemTipoDeEfeito: a rede não pode ter ponta solta.
//
// O `scripts/paginas-do-livro.py` já recusa gravar com tag órfã; este guarda
// cobra o mesmo do lado de cá, porque quem edita `conditions.json` à mão não
// passa pelo script.
func TestTodaTagDeCondicaoTemTipoDeEfeito(t *testing.T) {
	conhecidos := map[string]bool{}
	for _, e := range tiposDeEfeito() {
		conhecidos[e.ID] = true
	}
	if len(conhecidos) < 15 {
		t.Fatalf("só %d tipos de efeito — o catálogo não carregou", len(conhecidos))
	}
	for _, c := range catalogosDoLivro().Condicoes {
		for _, tag := range c.Tags {
			if !conhecidos[tag] {
				t.Errorf("a condição %q carrega o tipo %q, que não tem verbete", c.Name, tag)
			}
		}
	}
}
