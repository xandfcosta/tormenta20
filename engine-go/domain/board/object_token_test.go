package board

import "testing"

// A PEÇA DE OBJETO carrega as estatísticas da Tab. 5-4 (p239), e a pergunta que
// estes casos prendem é a da DIVERGÊNCIA.
//
// A peça já tinha `Footprint`, e ele não consegue carregar a categoria de
// tamanho: Minúsculo, Pequeno e Médio ocupam 1 quadrado e se defendem com 15, 12
// e 10. Então `Size` entrou como FONTE e o `Footprint` virou derivado — e o que
// se mede aqui é que os dois não conseguem discordar, por nenhum caminho.
//
// Isso importa porque o menu da peça deixa o mestre trocar o lado dela DEPOIS de
// posta. Sem a costura, uma porta Colossal viraria 1×1 no mapa continuando a se
// defender com 0, e a tela e a regra estariam dizendo coisas diferentes sobre a
// mesma peça — calado, que é como esta família de defeito sempre chega.

func umTabuleiro() *BoardState { return NewBoard("b1", "cripta", "pedra") }

func idFixo() func() string { return func() string { return "t1" } }

// A CRIAÇÃO deriva: o mestre escolhe "Colossal" e a peça nasce 6×6.
func TestAnObjectBornWithASizeGetsItsFootprintFromIt(t *testing.T) {
	b := umTabuleiro()
	porta := BoardToken{
		Label: "Celeiro", Kind: "object",
		Size: "Colossal", Material: "madeira", HpMax: 200, HpCurrent: 200,
		// O footprint vem ERRADO de propósito: é o que um cliente antigo
		// mandaria, e a derivação tem de vencer o que chegou no fio.
		Footprint: 1,
	}
	if err := AddToken(b, porta, idFixo()); err != nil {
		t.Fatalf("pôr o celeiro: %v", err)
	}
	if lado := b.Tokens[0].Footprint; lado != 6 {
		t.Errorf("o celeiro Colossal nasceu com lado %d, e a Tab. 1-21 (p107) dá 6", lado)
	}
}

// O MENU DA PEÇA não consegue desencaixar: trocar o lado de um objeto que tem
// tamanho volta ao lado do tamanho dele.
func TestChangingTheFootprintOfAnObjectCannotContradictItsSize(t *testing.T) {
	b := umTabuleiro()
	if err := AddToken(b, BoardToken{
		Label: "Porta da cripta", Kind: "object",
		Size: "Grande", Material: "madeira", HpMax: 20, HpCurrent: 20,
	}, idFixo()); err != nil {
		t.Fatalf("pôr a porta: %v", err)
	}
	// O CONTROLE: ela nasceu 2×2, senão o caso abaixo passaria sobre nada.
	if lado := b.Tokens[0].Footprint; lado != 2 {
		t.Fatalf("o controle já estava errado: a porta Grande nasceu com lado %d, e não 2", lado)
	}

	umQuadrado := 1
	if err := UpdateToken(b, "t1", TokenPatch{Footprint: &umQuadrado}); err != nil {
		t.Fatalf("remendar a porta: %v", err)
	}
	if lado := b.Tokens[0].Footprint; lado != 2 {
		t.Errorf("o lado da porta Grande virou %d por um remendo de footprint — a Defesa "+
			"dela continuaria sendo 8, de um objeto Grande, com 1×1 desenhado no mapa", lado)
	}
}

// E O CAMINHO INVERSO: trocar o TAMANHO redesenha a peça.
func TestChangingTheSizeOfAnObjectRedrawsIt(t *testing.T) {
	b := umTabuleiro()
	if err := AddToken(b, BoardToken{
		Label: "Barril", Kind: "object",
		Size: "Médio", Material: "madeira", HpMax: 10, HpCurrent: 10,
	}, idFixo()); err != nil {
		t.Fatalf("pôr o barril: %v", err)
	}
	enorme := "Enorme"
	if err := UpdateToken(b, "t1", TokenPatch{Size: &enorme}); err != nil {
		t.Fatalf("remendar o barril: %v", err)
	}
	if lado := b.Tokens[0].Footprint; lado != 3 {
		t.Errorf("o barril virou Enorme e continuou com lado %d — a p107 dá 3", lado)
	}
}

// O NPC TAMBÉM DERIVA, e o que protege a peça alheia é o tamanho VAZIO.
//
// Um ogro Grande ocupa 2×2 igual a uma porta Grande (p107) — a categoria não é
// uma propriedade de objeto, é do livro. Um ramo por `Kind` aqui faria a tira
// perguntar a categoria para as duas aparências e honrá-la só numa.
func TestAnyPieceWithASizeGetsItsFootprintFromIt(t *testing.T) {
	b := umTabuleiro()
	if err := AddToken(b, BoardToken{
		Label: "Ogro", Kind: "npc", Size: "Grande", Footprint: 1,
	}, idFixo()); err != nil {
		t.Fatalf("pôr o ogro: %v", err)
	}
	if lado := b.Tokens[0].Footprint; lado != 2 {
		t.Errorf("o ogro Grande ficou com lado %d, e a p107 dá 2 para qualquer Grande", lado)
	}
}

// E A PEÇA SEM CATEGORIA sai intocada: é a que o `Populate` traz da fila, com o
// lado que o chamador escolheu e nenhum tamanho para derivar de.
func TestAPieceWithoutASizeKeepsTheFootprintItArrivedWith(t *testing.T) {
	b := umTabuleiro()
	if err := AddToken(b, BoardToken{Label: "Herói", Kind: "character", Footprint: 3}, idFixo()); err != nil {
		t.Fatalf("pôr o herói: %v", err)
	}
	if lado := b.Tokens[0].Footprint; lado != 3 {
		t.Errorf("a peça sem categoria teve o lado trocado para %d", lado)
	}
}

// SEM ESTATÍSTICA é estado legítimo, e diferente de destruída.
//
// Uma mancha de musgo é cenário e não se ataca. Se `HasObjectStats` dissesse sim
// para ela, o ataque mediria contra Defesa de um tamanho vazio — e `IsDestroyed`
// diria que ela já está destruída, porque 0 PV é 0 PV.
func TestSceneryWithoutStatsIsNeitherAttackableNorDestroyed(t *testing.T) {
	musgo := BoardToken{Label: "Musgo", Kind: "object"}
	if musgo.HasObjectStats() {
		t.Errorf("uma peça de cenário sem tamanho, material nem PV se diz atacável")
	}
	if musgo.IsDestroyed() {
		t.Errorf("uma peça de cenário sem PV se diz destruída — ela nunca teve PV para perder")
	}
	porta := BoardToken{
		Label: "Porta", Kind: "object",
		Size: "Grande", Material: "madeira", HpMax: 20, HpCurrent: 20,
	}
	if !porta.HasObjectStats() {
		t.Fatalf("o controle já estava errado: a porta com os três campos não se diz atacável")
	}
	if porta.IsDestroyed() {
		t.Errorf("a porta inteira, com 20 de 20 PV, se diz destruída")
	}
	porta.HpCurrent = 0
	if !porta.IsDestroyed() {
		t.Errorf("a porta a 0 PV não se diz destruída, e a p239 diz que ela é")
	}
}
