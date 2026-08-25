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
	if !strings.Contains(corpo, "aba=efeitos&amp;entrada=medo") {
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

	corpo := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/catalogos?aba=condicoes", "").Body.String()
	if !strings.Contains(corpo, "aba=efeitos&amp;entrada=medo") {
		t.Error("o elo não endereça o verbete — voltou a ser busca")
	}
	if strings.Contains(corpo, "aba=efeitos&amp;busca=") {
		t.Error("sobrou elo apontando para uma busca")
	}
}

// TestOEnderecoDeUmVerbeteMostraSoEle, e oferece a saída.
func TestOEnderecoDeUmVerbeteMostraSoEle(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/catalogos?aba=efeitos&entrada=medo", "").Body.String()
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

	corpo := pedeNoMestre(t, s, eu, "GET", "/piloto/verbete?aba=efeitos&entrada=medo", "").Body.String()
	if !strings.Contains(corpo, `id="verbete-do-elo"`) {
		t.Fatal("o remendo não traz o id que ele substitui")
	}
	if !strings.Contains(corpo, "Medo capaz de prejudicar o alvo") {
		t.Error("a caixa não traz a definição")
	}
	// O MESMO cartão da aba, com o botão do livro: um segundo desenho "para a
	// caixa" divergiria do primeiro na terceira issue.
	if !strings.Contains(corpo, "/piloto/livro/ler?p=228") {
		t.Error("a caixa perdeu o botão do livro que o cartão tem")
	}

	// Id que não existe não inventa verbete — endereço se digita à mão.
	vazio := pedeNoMestre(t, s, eu, "GET", "/piloto/verbete?aba=efeitos&entrada=nao-existe", "").Body.String()
	if !strings.Contains(vazio, "não está no acervo") {
		t.Error("um id desconhecido não disse que não achou")
	}
}
