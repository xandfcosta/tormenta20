package api

import (
	"strings"
	"t20engine/domain/book"
	"t20engine/serve/web/bookui"
	"testing"
)

// O guarda dos ELOS entre entradas.
//
// O que se protege é uma REDE: a condição cita o tipo de efeito, agrava para
// outra condição, e a descrição dela nomeia uma terceira. Cada elo é um `href`
// montado a partir de dois catálogos diferentes, e um deles mudar de nome
// quebraria o elo em silêncio — a palavra continuaria na tela, só que morta.

// A palavra do tipo de efeito, na página de condições, é um elo para o verbete
// dele.
func TestTheConditionEffectTypeBecameALink(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	body := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes", "").Body.String()
	if !strings.Contains(body, "/mestre/efeitos?entrada=medo") {
		t.Error("a tag da condição não leva ao tipo de efeito")
	}
	// E ela é escrita como o LIVRO escreve, não como o dado guarda: a tag é
	// `cansaco`, sem acento e em caixa baixa, porque é uma chave. O ponto final
	// vem junto porque é assim que o livro fecha a condição ("… *Medo.*").
	if !strings.Contains(body, ">Medo.<") {
		t.Error("a tag saiu com a cara de chave em vez do nome do tipo")
	}
	// E ela vem DEPOIS da explicação, não colada no nome: "Abalado Medo" se lê
	// como se as duas palavras fossem o mesmo verbete.
	shaken := body[strings.Index(body, ">Abalado<"):]
	if strings.Index(shaken, "em testes de perícia") > strings.Index(shaken, ">Medo.<") {
		t.Error("o tipo de efeito voltou para antes da explicação")
	}
	effects := pedeNoMestre(t, s, eu, "GET", "/mestre/efeitos", "").Body.String()
	if !strings.Contains(effects, "Medo capaz de prejudicar o alvo") {
		t.Error("o destino do elo não tem a definição do livro — o elo levaria a lugar nenhum")
	}
}

// A citação vira elo, e o que ela NÃO faz.
//
// O controle é a segunda metade: uma condição não vira elo para SI MESMA — um
// elo que aponta para a página em que já se está é ruído com cara de saída.
func TestTheConditionCitedInTheDescriptionBecameALink(t *testing.T) {
	chunks := book.WithConditionLinks("Desprevenido e imóvel; -2 em ataques", "Agarrado")
	if len(chunks) < 2 || chunks[0].Text != "Desprevenido" || chunks[0].Aba != "condicoes" {
		t.Fatalf("a citação não virou elo: %+v", chunks)
	}
	if chunks[1].Aba != "" {
		t.Errorf("o resto da frase virou elo também: %+v", chunks[1])
	}

	own := book.WithConditionLinks("Desprevenido e não pode fazer ações.", "Desprevenido")
	for _, p := range own {
		if p.Aba != "" {
			t.Errorf("a condição virou elo para si mesma: %+v", p)
		}
	}
}

// Duas regras numa: no texto do livro a condição vem com MAIÚSCULA ("fica
// Abalado") e a palavra comum não ("um efeito de medo"). Casar sem caixa, ou
// casar pedaço de palavra, encheria a tela de elos que não são citação.
func TestTheLinkRespectsWholeWordsAndCase(t *testing.T) {
	cases := []struct {
		text   string
		waits  bool
		reason string
	}{
		{"O alvo fica Abalado.", true, "citação de verdade"},
		{"imune a efeitos de medo", false, "palavra comum, caixa baixa"},
		{"Abaladocom", false, "pedaço de palavra maior"},
		{"deixa Abalados os inimigos", false, "plural não é o nome da condição"},
	}
	for _, tc := range cases {
		hasLink := false
		for _, p := range book.WithConditionLinks(tc.text, "") {
			if p.Aba != "" {
				hasLink = true
			}
		}
		if hasLink != tc.waits {
			t.Errorf("%q: elo=%v, esperado %v — %s", tc.text, hasLink, tc.waits, tc.reason)
		}
	}
}

// "Quaisquer" é devoto de Aharadak e não é raça nem classe; "Elfos" vem no
// PLURAL e a raça é "Elfo". Elo que aponta para o vazio é pior que texto puro:
// ele promete uma página que não existe.
func TestTheGodLinksOnlyPointAtWhoHasAnEntry(t *testing.T) {
	if aba, id := book.DevoteeLink("Elfos"); aba != "racas" || id != "elfo" {
		t.Errorf("“Elfos” devia levar à raça elfo, e levou a %q/%q", aba, id)
	}
	if aba, _ := book.DevoteeLink("Bárbaros"); aba != "classes" {
		t.Errorf("“Bárbaros” devia levar às classes, e levou a %q", aba)
	}
	if aba, _ := book.DevoteeLink("Quaisquer"); aba != "" {
		t.Errorf("“Quaisquer” virou elo para %q", aba)
	}
	if bookui.PowerID("Coragem Total") == "" {
		t.Error("um poder concedido de verdade não foi reconhecido")
	}
	if bookui.PowerID("Poder Que Não Existe") != "" {
		t.Error("um nome inventado passou por poder do acervo")
	}
}

// A rede não pode ter ponta solta.
//
// O `scripts/book-pages.py` já recusa gravar com tag órfã; este guarda
// cobra o mesmo do lado de cá, porque quem edita `conditions.json` à mão não
// passa pelo script.
func TestEveryConditionTagHasAnEffectType(t *testing.T) {
	known := map[string]bool{}
	for _, e := range book.EffectKinds() {
		known[e.ID] = true
	}
	if len(known) < 15 {
		t.Fatalf("só %d tipos de efeito — o catálogo não carregou", len(known))
	}
	for _, c := range book.Catalogs().Conditions {
		for _, tag := range c.Tags {
			if !known[tag] {
				t.Errorf("a condição %q carrega o tipo %q, que não tem verbete", c.Name, tag)
			}
		}
	}
}

// O elo endereça o VERBETE e não uma busca: com `?busca=Medo` a cena mostra os
// oito grupos agrupados, e quem clica em "Medo" cai numa lista para procurar o
// que já tinha escolhido.
func TestTheLinkAddressesAnEntryAndNotASearch(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	body := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes", "").Body.String()
	if !strings.Contains(body, "/mestre/efeitos?entrada=medo") {
		t.Error("o elo não endereça o verbete — voltou a ser busca")
	}
	if strings.Contains(body, "/mestre/efeitos?busca=") {
		t.Error("sobrou elo apontando para uma busca")
	}
}

// O endereço do verbete mostra só ele, e oferece a saída.
func TestAnEntryAddressShowsOnlyThatEntry(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	body := pedeNoMestre(t, s, eu, "GET", "/mestre/efeitos?entrada=medo", "").Body.String()
	if !strings.Contains(body, "Medo capaz de prejudicar o alvo") {
		t.Fatal("o endereço do verbete não mostrou o verbete")
	}
	// O CONTROLE: os outros 17 tipos NÃO estão na tela. Sem ele, "achei o Medo"
	// seria verdade sobre a aba inteira.
	if strings.Contains(body, "Altera a forma ou composição corporal") {
		t.Error("a cena mostrou o acervo inteiro em vez do verbete pedido")
	}
	// E a saída: sem ela a pessoa fica sem entender por que a aba tem uma
	// entrada só.
	if !strings.Contains(body, "ver Efeitos inteiro") {
		t.Error("não há como voltar para a aba inteira")
	}
}

// A caixa do verbete traz o cartão INTEIRO — é o remendo que o elo pede.
func TestTheEntryBoxCarriesTheWholeCard(t *testing.T) {
	s := serverWithBook(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")

	body := pedeNoMestre(t, s, eu, "GET", "/verbete?aba=efeitos&entrada=medo", "").Body.String()
	if !strings.Contains(body, `id="crossref-entry"`) {
		t.Fatal("o remendo não traz o id que ele substitui")
	}
	if !strings.Contains(body, "Medo capaz de prejudicar o alvo") {
		t.Error("a caixa não traz a definição")
	}
	// O MESMO cartão da aba, com o botão do livro: um segundo desenho "para a
	// caixa" divergiria do primeiro na terceira issue.
	if !strings.Contains(body, "/livro/ler?p=228") {
		t.Error("a caixa perdeu o botão do livro que o cartão tem")
	}

	// Id que não existe não inventa verbete — endereço se digita à mão.
	empty := pedeNoMestre(t, s, eu, "GET", "/verbete?aba=efeitos&entrada=nao-existe", "").Body.String()
	if !strings.Contains(empty, "não está no acervo") {
		t.Error("um id desconhecido não disse que não achou")
	}
}

// Tirar só "s" e "es" não dá conta: os casos abaixo falham cada um por um motivo
// diferente do português — ou por não ser plural nenhum.
func TestTheDevotoInThePluralFindsTheEntry(t *testing.T) {
	races, _, _ := book.CharacterCatalogs()
	nameByID := map[string]string{}
	for _, r := range races {
		nameByID[r.ID] = r.Name
	}

	cases := []struct{ devotee, race, reason string }{
		{"Anões", "Anão", "ões → ão, e não “Anõe”"},
		{"Golens", "Golem", "ns → m"},
		{"Sereias/Tritões", "Sereia/Tritão", "as DUAS metades vão para o plural"},
		{"Aggelus", "Suraggel", "não é plural: é a ascendência que o catálogo guarda"},
		{"Sulfure", "Suraggel", "a outra ascendência"},
		{"Elfos", "Elfo", "o caso simples continua valendo"},
	}
	for _, tc := range cases {
		aba, id := book.DevoteeLink(tc.devotee)
		if aba != "racas" || nameByID[id] != tc.race {
			t.Errorf("%q levou a %q/%q, esperado a raça %q — %s",
				tc.devotee, aba, id, tc.race, tc.reason)
		}
	}

	// O CONTROLE: o que não é verbete continua sem elo. Sem ele, uma regra
	// frouxa demais passaria verde ligando tudo a qualquer coisa.
	for _, isNot := range []string{"Quaisquer", "Aventureiros (todas as classes)", "Qualquer duyshidakk"} {
		if aba, _ := book.DevoteeLink(isNot); aba != "" {
			t.Errorf("%q virou elo para %q, e não é verbete de nada", isNot, aba)
		}
	}
}

// O livro se cita, e o número tem de levar a algum lugar.
func TestAPageReferenceInTheTextBecomesALink(t *testing.T) {
	chunks := book.WithLinks("Reduz os PV do alvo. Efeitos deste tipo são subdivididos em tipos de dano (veja a página 230).")
	var found *book.Chunk
	for i := range chunks {
		if chunks[i].Page > 0 {
			found = &chunks[i]
		}
	}
	if found == nil {
		t.Fatalf("a referência não virou elo: %+v", chunks)
	}
	if found.Page != 230 {
		t.Errorf("a referência aponta para a p%d", found.Page)
	}
	// O TEXTO do elo é a frase do livro, e não um "p230 ↗" inventado: trocá-la
	// reescreveria o livro na tela.
	if found.Text != "página 230" {
		t.Errorf("o elo mudou o texto para %q", found.Text)
	}
	// E o resto da frase continua inteiro, texto puro.
	whole := ""
	for _, p := range chunks {
		whole += p.Text
	}
	if !strings.Contains(whole, "Reduz os PV do alvo.") || !strings.Contains(whole, ").") {
		t.Errorf("a varredura comeu pedaço da frase: %q", whole)
	}
}

// O controle da varredura: número solto não vira página.
func TestALooseNumberDoesNotBecomeAPage(t *testing.T) {
	for _, text := range []string{"causa 3d6 de dano", "recebe +2 na Defesa e 230 de alcance", "20% de chance"} {
		for _, p := range book.WithLinks(text) {
			if p.Page > 0 {
				t.Errorf("%q: o número %d virou página", text, p.Page)
			}
		}
	}
}

// Os aprimoramentos abrem na caixa, em vez de serem uma contagem ilegível.
func TestAugmentsOpenInTheBox(t *testing.T) {
	s := serverWithBook(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")

	scene := pedeNoMestre(t, s, eu, "GET", "/mestre/magias?entrada=bola-de-fogo", "").Body.String()
	if !strings.Contains(scene, "aprimoramentos disponíveis") {
		t.Fatal("o cartão não oferece os aprimoramentos")
	}
	if !strings.Contains(scene, "parte=aprimoramentos") {
		t.Error("o botão não pede a parte dos aprimoramentos")
	}

	box := pedeNoMestre(t, s, eu, "GET", "/verbete?aba=magias&parte=aprimoramentos&entrada=bola-de-fogo", "").Body.String()
	if !strings.Contains(box, "Aumenta o dano em +2d6") {
		t.Error("a caixa não traz o texto do aprimoramento")
	}
	if !strings.Contains(box, "+2 PM") {
		t.Error("a caixa não traz o custo, que é por onde o mestre varre a lista")
	}
	// O CONTROLE: sem a `parte`, a mesma rota devolve o cartão inteiro.
	whole := pedeNoMestre(t, s, eu, "GET", "/verbete?aba=magias&entrada=bola-de-fogo", "").Body.String()
	if strings.Contains(whole, "Aumenta o dano em +2d6") {
		t.Error("o cartão inteiro veio com os aprimoramentos — a `parte` não separa nada")
	}
}

// AMOSTRAGEM sobre os VINTE deuses, e não sobre os que alguém relatou: a lacuna
// é invisível na tela — a palavra continua lá, só não leva a lugar nenhum.
func TestEveryGodLinksThePowersItGrants(t *testing.T) {
	_, _, gods := book.CharacterCatalogs()
	if len(gods) < 20 {
		t.Fatalf("só %d deuses — o guarda mediria quase nada", len(gods))
	}
	granted := 0
	for _, d := range gods {
		for _, power := range d.GrantedPowers {
			granted++
			if bookui.PowerID(power) == "" {
				t.Errorf("%s concede %q, que não tem verbete no acervo", d.Name, power)
			}
		}
	}
	// O CONTROLE: havia poder para medir. Sem ele, um `poderesConcedidos` vazio
	// passaria verde.
	if granted < 60 {
		t.Errorf("só %d poderes concedidos no total — o catálogo não carregou", granted)
	}
}

// O outro lado do cartão do deus.
//
// Os três que ficam de fora estão NOMEADOS porque são exatamente os que não são
// verbete de nada — e prendê-los é o que faz o guarda acusar no dia em que um
// quarto aparecer por um defeito de casamento de plural.
func TestEveryDevotoThatIsAnEntryBecomesALink(t *testing.T) {
	_, _, gods := book.CharacterCatalogs()
	noEntry := map[string]bool{
		"Quaisquer":                       true,
		"Qualquer duyshidakk":             true,
		"Aventureiros (todas as classes)": true,
	}
	for _, d := range gods {
		for _, devotee := range d.Devotees {
			aba, _ := book.DevoteeLink(devotee)
			if aba == "" && !noEntry[devotee] {
				t.Errorf("%s tem o devoto %q sem elo — plural que o casamento não pega?", d.Name, devotee)
			}
			if aba != "" && noEntry[devotee] {
				t.Errorf("%q virou elo para %q, e não é raça nem classe", devotee, aba)
			}
		}
	}
}
