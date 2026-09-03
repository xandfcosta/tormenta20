package api

import (
	"strings"
	"testing"
)

// O guarda dos FILTROS de cada catálogo (ALE-264).
//
// O que se protege é a regra de COMBINAÇÃO — OU dentro de um filtro, E entre
// eles — e o fato de que cada catálogo oferece os seus. Um filtro que some da
// cena não estoura nada: a lista continua desenhando, só que inteira.

// TestTheFilterIsAnAddress: `?circulo=3` abre a cena já filtrada.
func TestTheFilterIsAnAddress(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/magias?circulo=3&escola=evocacao", "").Body.String()
	if !strings.Contains(corpo, "6 entradas") {
		t.Error("a cena não abriu filtrada pelo endereço")
	}
	// E os crachás nascem ACESOS: sem isso a lista está filtrada e a tela não
	// diz por quê — o pior estado possível.
	if !strings.Contains(corpo, `aria-pressed="true"`) {
		t.Error("nenhum crachá aceso numa cena que abriu filtrada")
	}
}

// TestAFilterFromAnotherTabDoesNotApply: `?circulo=3` nas condições não filtra nada.
//
// Aceitá-lo faria a cena carregar um estado que ela não sabe desenhar — filtro
// aplicado sem crachá para desligá-lo, e a pessoa presa numa lista curta sem
// entender.
func TestAFilterFromAnotherTabDoesNotApply(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes?circulo=3", "").Body.String()
	if !strings.Contains(corpo, "35 entradas") {
		t.Error("um filtro de outra aba mexeu na lista das condições")
	}
}

// TestSearchHidesTheFilters — com termo digitado a cena responde outra
// pergunta, e um crachá de círculo sobre uma lista com itens e condições diria
// que ele filtra o que não filtra.
func TestSearchHidesTheFilters(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	semBusca := pedeNoMestre(t, s, eu, "GET", "/mestre/magias", "").Body.String()
	comBusca := pedeNoMestre(t, s, eu, "GET", "/mestre/magias?busca=fogo", "").Body.String()

	if !strings.Contains(semBusca, "Filtrar por Círculo") {
		t.Error("a cena das magias não desenha os filtros")
	}
	if strings.Contains(comBusca, "Filtrar por Círculo") {
		t.Error("os filtros continuaram na tela durante a busca")
	}
}

// TestTheSpellCardTellsAndLetsYouOpenTheSchool: o elo, ponta a ponta.
func TestTheSpellCardTellsAndLetsYouOpenTheSchool(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/magias?entrada=bola-de-fogo", "").Body.String()
	if !strings.Contains(corpo, "Evocação") {
		t.Error("o cartão da magia não diz a escola dela")
	}
	if !strings.Contains(corpo, "/mestre/escolas?entrada=evocacao") {
		t.Error("a escola não virou elo")
	}
	// E o destino existe, com a definição do livro.
	escola := pedeNoMestre(t, s, eu, "GET", "/mestre/escolas?entrada=evocacao", "").Body.String()
	if !strings.Contains(escola, "manipulam ou geram energia pura") {
		t.Error("o verbete da escola não traz a definição do livro")
	}
	// A ABREVIATURA que as tabelas do livro imprimem.
	if !strings.Contains(escola, "Evoc") {
		t.Error("o verbete perdeu a abreviatura, que é como o livro escreve nas tabelas")
	}
}

// TestTheClassLinksTheExpertisesItTrains: citação sem destino é texto morto.
func TestTheClassLinksTheExpertisesItTrains(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/classes?entrada=bardo", "").Body.String()
	if !strings.Contains(corpo, "/mestre/pericias?entrada=atuacao") {
		t.Error("o Bardo não liga a perícia que ele treina")
	}
	// E o caminho de volta: a perícia diz quem a treina de saída.
	pericia := pedeNoMestre(t, s, eu, "GET", "/mestre/pericias?entrada=atuacao", "").Body.String()
	if !strings.Contains(pericia, "/mestre/classes?entrada=bardo") {
		t.Error("Atuação não diz que o Bardo a treina de saída")
	}
}
