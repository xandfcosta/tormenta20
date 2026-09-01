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

	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes", "").Body.String()
	if !strings.Contains(corpo, "/mestre/efeitos?entrada=medo") {
		t.Error("a tag da condição não leva ao tipo de efeito")
	}
	// E ela é escrita como o LIVRO escreve, não como o dado guarda: a tag é
	// `cansaco`, sem acento e em caixa baixa, porque é uma chave. O ponto final
	// vem junto porque é assim que o livro fecha a condição ("… *Medo.*").
	if !strings.Contains(corpo, ">Medo.<") {
		t.Error("a tag saiu com a cara de chave em vez do nome do tipo")
	}
	// E ela vem DEPOIS da explicação, não colada no nome: o dono leu "Abalado
	// Medo" como se as duas palavras fossem o mesmo verbete. A ordem no HTML é
	// a prova — a descrição primeiro, o tipo em seguida.
	abalado := corpo[strings.Index(corpo, ">Abalado<"):]
	if strings.Index(abalado, "em testes de perícia") > strings.Index(abalado, ">Medo.<") {
		t.Error("o tipo de efeito voltou para antes da explicação")
	}
	efeitos := pedeNoMestre(t, s, eu, "GET", "/mestre/efeitos", "").Body.String()
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
	if aba, id := eloDoDevoto("Elfos"); aba != "racas" || id != "elfo" {
		t.Errorf("“Elfos” devia levar à raça elfo, e levou a %q/%q", aba, id)
	}
	if aba, _ := eloDoDevoto("Bárbaros"); aba != "classes" {
		t.Errorf("“Bárbaros” devia levar às classes, e levou a %q", aba)
	}
	if aba, _ := eloDoDevoto("Quaisquer"); aba != "" {
		t.Errorf("“Quaisquer” virou elo para %q", aba)
	}
	if idDoPoder("Coragem Total") == "" {
		t.Error("um poder concedido de verdade não foi reconhecido")
	}
	if idDoPoder("Poder Que Não Existe") != "" {
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

// TestOEloEnderecaUmVerbeteENaoUmaBusca (ALE-264).
//
// PROVADO VERMELHO contra a primeira versão: o elo apontava para
// `?aba=efeitos&busca=Medo`, e busca com termo faz a cena mostrar os OITO grupos
// agrupados — quem clicava em "Medo" caía numa lista para procurar o que já
// tinha escolhido. O dono viu e disse: "aparece na quarta seção da busca".
func TestOEloEnderecaUmVerbeteENaoUmaBusca(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes", "").Body.String()
	if !strings.Contains(corpo, "/mestre/efeitos?entrada=medo") {
		t.Error("o elo não endereça o verbete — voltou a ser busca")
	}
	if strings.Contains(corpo, "/mestre/efeitos?busca=") {
		t.Error("sobrou elo apontando para uma busca")
	}
}

// TestOEnderecoDeUmVerbeteMostraSoEle, e oferece a saída.
func TestOEnderecoDeUmVerbeteMostraSoEle(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/efeitos?entrada=medo", "").Body.String()
	if !strings.Contains(corpo, "Medo capaz de prejudicar o alvo") {
		t.Fatal("o endereço do verbete não mostrou o verbete")
	}
	// O CONTROLE: os outros 17 tipos NÃO estão na tela. Sem ele, "achei o Medo"
	// seria verdade sobre a aba inteira.
	if strings.Contains(corpo, "Altera a forma ou composição corporal") {
		t.Error("a cena mostrou o acervo inteiro em vez do verbete pedido")
	}
	// E a saída: sem ela a pessoa fica sem entender por que a aba tem uma
	// entrada só.
	if !strings.Contains(corpo, "ver Efeitos inteiro") {
		t.Error("não há como voltar para a aba inteira")
	}
}

// TestACaixaDoVerbeteTrazOCartaoInteiro: o remendo que o elo pede.
func TestACaixaDoVerbeteTrazOCartaoInteiro(t *testing.T) {
	s := servidorComLivro(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/verbete?aba=efeitos&entrada=medo", "").Body.String()
	if !strings.Contains(corpo, `id="verbete-do-elo"`) {
		t.Fatal("o remendo não traz o id que ele substitui")
	}
	if !strings.Contains(corpo, "Medo capaz de prejudicar o alvo") {
		t.Error("a caixa não traz a definição")
	}
	// O MESMO cartão da aba, com o botão do livro: um segundo desenho "para a
	// caixa" divergiria do primeiro na terceira issue.
	if !strings.Contains(corpo, "/livro/ler?p=228") {
		t.Error("a caixa perdeu o botão do livro que o cartão tem")
	}

	// Id que não existe não inventa verbete — endereço se digita à mão.
	vazio := pedeNoMestre(t, s, eu, "GET", "/verbete?aba=efeitos&entrada=nao-existe", "").Body.String()
	if !strings.Contains(vazio, "não está no acervo") {
		t.Error("um id desconhecido não disse que não achou")
	}
}

// TestODevotoNoPluralAchaOVerbete (ALE-264).
//
// PROVADO VERMELHO: a primeira versão tentava só tirar "s" e "es", e o dono viu
// os buracos. Os quatro casos abaixo são os que faltavam, cada um por um motivo
// diferente do português — ou por não ser plural nenhum.
func TestODevotoNoPluralAchaOVerbete(t *testing.T) {
	racas, _, _ := catalogosDoPersonagem()
	nomePorID := map[string]string{}
	for _, r := range racas {
		nomePorID[r.ID] = r.Name
	}

	casos := []struct{ devoto, raca, porque string }{
		{"Anões", "Anão", "ões → ão, e não “Anõe”"},
		{"Golens", "Golem", "ns → m"},
		{"Sereias/Tritões", "Sereia/Tritão", "as DUAS metades vão para o plural"},
		{"Aggelus", "Suraggel", "não é plural: é a ascendência que o catálogo guarda"},
		{"Sulfure", "Suraggel", "a outra ascendência"},
		{"Elfos", "Elfo", "o caso simples continua valendo"},
	}
	for _, caso := range casos {
		aba, id := eloDoDevoto(caso.devoto)
		if aba != "racas" || nomePorID[id] != caso.raca {
			t.Errorf("%q levou a %q/%q, esperado a raça %q — %s",
				caso.devoto, aba, id, caso.raca, caso.porque)
		}
	}

	// O CONTROLE: o que não é verbete continua sem elo. Sem ele, uma regra
	// frouxa demais passaria verde ligando tudo a qualquer coisa.
	for _, naoEh := range []string{"Quaisquer", "Aventureiros (todas as classes)", "Qualquer duyshidakk"} {
		if aba, _ := eloDoDevoto(naoEh); aba != "" {
			t.Errorf("%q virou elo para %q, e não é verbete de nada", naoEh, aba)
		}
	}
}

// TestAReferenciaDePaginaNoTextoViraElo: o livro se cita, e o número levava a
// lugar nenhum.
func TestAReferenciaDePaginaNoTextoViraElo(t *testing.T) {
	pedacos := comElosDoTexto("Reduz os PV do alvo. Efeitos deste tipo são subdivididos em tipos de dano (veja a página 230).")
	var achou *trecho
	for i := range pedacos {
		if pedacos[i].Pagina > 0 {
			achou = &pedacos[i]
		}
	}
	if achou == nil {
		t.Fatalf("a referência não virou elo: %+v", pedacos)
	}
	if achou.Pagina != 230 {
		t.Errorf("a referência aponta para a p%d", achou.Pagina)
	}
	// O TEXTO do elo é a frase do livro, e não um "p230 ↗" inventado: trocá-la
	// reescreveria o livro na tela.
	if achou.Texto != "página 230" {
		t.Errorf("o elo mudou o texto para %q", achou.Texto)
	}
	// E o resto da frase continua inteiro, texto puro.
	inteiro := ""
	for _, p := range pedacos {
		inteiro += p.Texto
	}
	if !strings.Contains(inteiro, "Reduz os PV do alvo.") || !strings.Contains(inteiro, ").") {
		t.Errorf("a varredura comeu pedaço da frase: %q", inteiro)
	}
}

// TestUmNumeroSoltoNaoViraPagina: o controle da varredura.
func TestUmNumeroSoltoNaoViraPagina(t *testing.T) {
	for _, texto := range []string{"causa 3d6 de dano", "recebe +2 na Defesa e 230 de alcance", "20% de chance"} {
		for _, p := range comElosDoTexto(texto) {
			if p.Pagina > 0 {
				t.Errorf("%q: o número %d virou página", texto, p.Pagina)
			}
		}
	}
}

// TestOsAprimoramentosAbremNaCaixa: eram uma contagem que não se podia ler.
func TestOsAprimoramentosAbremNaCaixa(t *testing.T) {
	s := servidorComLivro(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")

	cena := pedeNoMestre(t, s, eu, "GET", "/mestre/magias?entrada=bola-de-fogo", "").Body.String()
	if !strings.Contains(cena, "aprimoramentos disponíveis") {
		t.Fatal("o cartão não oferece os aprimoramentos")
	}
	if !strings.Contains(cena, "parte=aprimoramentos") {
		t.Error("o botão não pede a parte dos aprimoramentos")
	}

	caixa := pedeNoMestre(t, s, eu, "GET", "/verbete?aba=magias&parte=aprimoramentos&entrada=bola-de-fogo", "").Body.String()
	if !strings.Contains(caixa, "Aumenta o dano em +2d6") {
		t.Error("a caixa não traz o texto do aprimoramento")
	}
	if !strings.Contains(caixa, "+2 PM") {
		t.Error("a caixa não traz o custo, que é por onde o mestre varre a lista")
	}
	// O CONTROLE: sem a `parte`, a mesma rota devolve o cartão inteiro.
	inteiro := pedeNoMestre(t, s, eu, "GET", "/verbete?aba=magias&entrada=bola-de-fogo", "").Body.String()
	if strings.Contains(inteiro, "Aumenta o dano em +2d6") {
		t.Error("o cartão inteiro veio com os aprimoramentos — a `parte` não separa nada")
	}
}

// TestTodoDeusLigaOsPoderesQueConcede (ALE-264).
//
// PROVADO VERMELHO: o dono mandou três cartões — Valkaria, Wynna e Thwor — em
// que a maior parte dos poderes concedidos era texto morto. A causa não estava
// no elo: o acervo lia o `granted-powers` (36 nomes) e não o `divine-powers`
// (72), por causa de um comentário que afirmava que os divinos "não têm texto de
// regra". Eles têm.
//
// AMOSTRAGEM sobre os VINTE deuses e não sobre os três que o dono viu: a lacuna
// era invisível na tela — a palavra continuava lá, só não levava a lugar nenhum
// —, e conferir só os relatados deixaria os outros dezessete no escuro.
func TestTodoDeusLigaOsPoderesQueConcede(t *testing.T) {
	_, _, deuses := catalogosDoPersonagem()
	if len(deuses) < 20 {
		t.Fatalf("só %d deuses — o guarda mediria quase nada", len(deuses))
	}
	concedidos := 0
	for _, d := range deuses {
		for _, poder := range d.PoderesConcedidos {
			concedidos++
			if idDoPoder(poder) == "" {
				t.Errorf("%s concede %q, que não tem verbete no acervo", d.Name, poder)
			}
		}
	}
	// O CONTROLE: havia poder para medir. Sem ele, um `poderesConcedidos` vazio
	// passaria verde.
	if concedidos < 60 {
		t.Errorf("só %d poderes concedidos no total — o catálogo não carregou", concedidos)
	}
}

// TestTodoDevotoQueEVerbeteViraElo: o outro lado do cartão do deus.
//
// Os três que ficam de fora estão NOMEADOS porque são exatamente os que não são
// verbete de nada — e prendê-los é o que faz o guarda acusar no dia em que um
// quarto aparecer por um defeito de casamento de plural.
func TestTodoDevotoQueEVerbeteViraElo(t *testing.T) {
	_, _, deuses := catalogosDoPersonagem()
	semVerbete := map[string]bool{
		"Quaisquer":                       true,
		"Qualquer duyshidakk":             true,
		"Aventureiros (todas as classes)": true,
	}
	for _, d := range deuses {
		for _, devoto := range d.Devotos {
			aba, _ := eloDoDevoto(devoto)
			if aba == "" && !semVerbete[devoto] {
				t.Errorf("%s tem o devoto %q sem elo — plural que o casamento não pega?", d.Name, devoto)
			}
			if aba != "" && semVerbete[devoto] {
				t.Errorf("%q virou elo para %q, e não é raça nem classe", devoto, aba)
			}
		}
	}
}
