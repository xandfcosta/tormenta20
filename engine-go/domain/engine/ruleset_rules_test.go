package engine

import (
	"path/filepath"
	"testing"
)

// AQUI MORAVA O TestEveryCatalogEntryPointAppliesTheCampaignView, que varria as
// portas de `*Catalogs` e exigia que cada uma abrisse aplicando a vista da mesa.
//
// Ele saiu porque o TIPO faz o trabalho dele, e faz melhor: computar uma ficha
// pede um `*Ruleset`, então quem tem só o livro na mão NÃO COMPILA. Um guarda
// precisa ser rodado para pegar o esquecimento; o compilador não.

// A EMENDA DA MESA ACRESCENTA AO VERBETE DO LIVRO, SEM SUBSTITUÍ-LO (ALE-387).
//
// O caso é real: numa mesa, o medalhão de prata concede +1 em Luta além do que
// o livro dá. O verbete da p160 tem UM modificador — `pmLimit +1`, condicionado
// a empunhar — e a mesa quer esse MAIS o de Luta.
//
// O que se prende aqui é que o do LIVRO sobrevive. Uma emenda que substituísse
// entregaria só o +1 de Luta, e o jogador perderia o limite de PM sem ninguém
// notar — o item continuaria na mochila, com o nome certo, concedendo menos.
//
// E a ORDEM: o do livro primeiro, o da mesa depois. A ordem dos modificadores é
// o que o oráculo compara byte a byte, e uma emenda que entrasse na frente
// mudaria a decomposição de todo item emendado.
func TestTheCampaignAmendmentAddsToTheBookEntryWithoutReplacingIt(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	book := primeFromDump(t, dir)

	doLivro := book.Item("medalhao-de-prata")
	if doLivro == nil || len(doLivro.Modifiers) != 1 {
		t.Fatalf("o verbete do livro mudou de forma; este caso partia de 1 modificador: %+v", doLivro)
	}

	daMesa := Modifier{
		Target:    ModifierTarget{K: "expertise", Name: "Luta"},
		Amount:    1,
		BonusType: "untyped",
		Note:      "Regra da mesa",
	}
	mesa := RulesetOf(book, Amendments{Entries: map[string][]Modifier{
		"medalhao-de-prata": {daMesa},
	}})

	emendado := mesa.itemOf("medalhao-de-prata")
	if len(emendado.Modifiers) != 2 {
		t.Fatalf("o item emendado tem %d modificadores, esperava 2 (o do livro + o da mesa): %+v",
			len(emendado.Modifiers), emendado.Modifiers)
	}
	if emendado.Modifiers[0].Target.K != "pmLimit" {
		t.Fatalf("o modificador do LIVRO não veio primeiro: %+v", emendado.Modifiers[0])
	}
	if emendado.Modifiers[1].Target.Name != "Luta" {
		t.Fatalf("o modificador da MESA não veio por último: %+v", emendado.Modifiers[1])
	}

	// O CONTROLE que importa: o mundo não pode sujar o livro. Todas as mesas
	// leem o mesmo `Catalogs` primado, e uma emenda que escrevesse no mapa dele
	// vazaria para as outras — e para o molde, que deve ver o livro puro.
	if depois := book.Item("medalhao-de-prata"); len(depois.Modifiers) != 1 {
		t.Fatalf("o mundo SUJOU o livro: ele agora tem %d modificadores.\n"+
			"A emenda tem de viver na vista, nunca no mapa compartilhado.", len(depois.Modifiers))
	}
	// E um item que a mesa não emendou continua sendo o ponteiro do livro.
	if mesa.itemOf("armadura-completa") != book.Item("armadura-completa") {
		t.Error("um item NÃO emendado voltou como cópia — a vista está copiando o que não precisa")
	}
}

// A EMENDA CHEGA À FICHA, e o MUNDO é quem a leva.
//
// O arranjo é o de produção: o livro é um só, e o que muda entre as duas contas
// é o MUNDO em que a ficha é computada. Um caso que passasse os modificadores
// pelo personagem provaria o transporte que esta fatia justamente trocou.
func TestTheCampaignAmendmentReachesTheComputedSheet(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	book := primeFromDump(t, dir)

	wielded := "wielded"
	id := "medalhao-de-prata"
	ch := Character{
		Expertises: []CharacterExpertise{{Name: "Luta", Attribute: "strength"}},
		Items:      []CharacterItem{{Name: "Medalhão de prata", CatalogID: &id, Equipped: &wielded}},
	}

	lutaEm := func(world *Ruleset) int {
		for _, ex := range world.ComputeSheet(ch, nil).Expertises {
			if ex.Name == "Luta" {
				return ex.Total
			}
		}
		t.Fatal("a perícia Luta não apareceu na ficha")
		return 0
	}

	mesa := RulesetOf(book, Amendments{Entries: map[string][]Modifier{
		id: {{Target: ModifierTarget{K: "expertise", Name: "Luta"}, Amount: 1, BonusType: "untyped"}},
	}})

	foraDeMesa, naMesa := lutaEm(BookRuleset(book)), lutaEm(mesa)
	if naMesa != foraDeMesa+1 {
		t.Fatalf("Luta na mesa deu %d e fora dela %d — esperava exatamente +1.\n"+
			"Se os dois forem iguais, a emenda não chegou à coleta.", naMesa, foraDeMesa)
	}

	// O CONTROLE: os dois mundos dividem o MESMO livro. Se o primeiro o tivesse
	// sujado, o segundo levaria o bônus dele — e computar na outra ordem
	// esconderia isso.
	if depois := lutaEm(BookRuleset(book)); depois != foraDeMesa {
		t.Fatalf("computar na mesa SUJOU o livro: Luta fora dela foi %d e voltou %d", foraDeMesa, depois)
	}
}

// A EMENDA QUE NÃO DECODIFICA RECUSA ALTO, em vez de virar uma emenda vazia.
//
// Ver `ParseModifiers`: perder um bônus que o mestre escreveu não tem aparência
// nenhuma na tela.
func TestTheCampaignAmendmentRefusesModifiersItCannotRead(t *testing.T) {
	ok, err := ParseModifiers(`[{"target":{"k":"expertise","name":"Luta"},"amount":1}]`)
	if err != nil {
		t.Fatalf("a emenda bem formada foi recusada: %v", err)
	}
	if len(ok) != 1 {
		t.Fatalf("a emenda bem formada veio com %d modificadores", len(ok))
	}

	if _, err := ParseModifiers(`{"amount":1}`); err == nil {
		t.Fatal("um objeto onde se esperava lista passou — a emenda teria sumido em silêncio")
	}
}

// O ESCOPO DE UMA CONCESSÃO É RESPEITADO, e o vizinho de mesa não leva junto.
//
// É a metade que o desenho anterior não conseguia nem representar: com as
// emendas viajando no personagem, "a mesa inteira" e "só este herói" chegavam
// ao motor já misturados.
//
// O caso põe DOIS heróis no mesmo mundo de propósito. Com um só, a concessão
// específica e a de mesa dão o mesmo resultado, e o guarda mediria a metade em
// que o defeito é invisível por construção.
func TestACampaignGrantReachesOnlyWhoTheSelectorNames(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	book := primeFromDump(t, dir)

	heroi := func(id int) Character {
		return Character{ID: id, Expertises: []CharacterExpertise{{Name: "Luta", Attribute: "strength"}}}
	}
	umaLuta := []Modifier{{Target: ModifierTarget{K: "expertise", Name: "Luta"}, Amount: 1, BonusType: "untyped"}}

	mundo := RulesetOf(book, Amendments{Grants: []CampaignGrant{
		{ID: "g-mesa", Applies: EveryoneIn(), Label: "Bênção da mesa", Modifiers: umaLuta},
		{ID: "g-so-do-7", Applies: OnlyCharacter(7), Label: "Pacto do Sétimo", Modifiers: umaLuta},
	}})

	lutaEm := func(world *Ruleset, ch Character) int {
		for _, ex := range world.ComputeSheet(ch, nil).Expertises {
			if ex.Name == "Luta" {
				return ex.Total
			}
		}
		t.Fatal("a perícia Luta não apareceu na ficha")
		return 0
	}
	// A base é o LIVRO, e não o mundo: medi-la dentro do mundo já traria a
	// concessão da mesa embutida, e as duas contas se cancelariam.
	base := lutaEm(BookRuleset(book), heroi(7))
	lutaDe := func(ch Character) int { return lutaEm(mundo, ch) }

	if got := lutaDe(heroi(7)); got != base+2 {
		t.Errorf("o herói 7 tirou %d e esperava %d: a concessão da mesa MAIS a dele", got, base+2)
	}
	if got := lutaDe(heroi(9)); got != base+1 {
		t.Errorf("o herói 9 tirou %d e esperava %d.\n"+
			"Só a concessão da MESA o alcança; a do 7 é de outro personagem.", got, base+1)
	}
}

// O PERSONAGEM SEM ID NÃO É ALCANÇADO POR UMA CONCESSÃO ESPECÍFICA.
//
// O `Character{}` dos fixtures e do oráculo tem `ID` zero, e um seletor que
// casasse o zero faria a emenda de uma mesa aparecer no oráculo — que é a rede
// de regressão da ficha inteira. Erra para o lado do LIVRO, de propósito.
func TestACharacterGrantNeverMatchesTheZeroId(t *testing.T) {
	if OnlyCharacter(0).Matches(Character{}) {
		t.Error("um seletor de personagem sem id casou com o personagem sem id")
	}
	if OnlyCharacter(3).Matches(Character{}) {
		t.Error("um seletor do herói 3 casou com um personagem sem id")
	}
	if !EveryoneIn().Matches(Character{}) {
		t.Error("o seletor da mesa inteira deixou de fora um personagem — ele alcança todos")
	}
	if (Selector{Kind: "uma-espécie-do-futuro"}).Matches(Character{ID: 1}) {
		t.Error("uma espécie de seletor desconhecida alcançou alguém; o lado seguro é não alcançar")
	}
}
