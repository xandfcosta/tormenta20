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
// Ver `AmendmentsFrom`: perder um bônus que o mestre escreveu não tem aparência
// nenhuma na tela.
func TestTheCampaignAmendmentRefusesModifiersItCannotRead(t *testing.T) {
	ok, err := AmendmentsFrom(map[string]string{
		"medalhao-de-prata": `[{"target":{"k":"expertise","name":"Luta"},"amount":1}]`,
	})
	if err != nil {
		t.Fatalf("a emenda bem formada foi recusada: %v", err)
	}
	if len(ok.Entries["medalhao-de-prata"]) != 1 {
		t.Fatalf("a emenda bem formada veio com %d modificadores", len(ok.Entries["medalhao-de-prata"]))
	}

	if _, err := AmendmentsFrom(map[string]string{"medalhao-de-prata": `{"amount":1}`}); err == nil {
		t.Fatal("um objeto onde se esperava lista passou — a emenda teria sumido em silêncio")
	}
}
