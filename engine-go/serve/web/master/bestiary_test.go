package master

import (
	"slices"
	"strings"
	"t20engine/domain/book"
	"t20engine/serve/web/routes"
	"testing"
)

// O bestiário do livro lido pelo servidor.
//
// O guia manda validar catálogo por SCHEMA no despejo e prender só a EXCEÇÃO —
// a armadilha da tabela —, nunca repetir a tabela inteira num `expect` por
// campo. Aqui a exceção é o TRAVESSÃO, e ele merece guarda porque a perda dele
// é invisível: um `int` recebendo `null` vira 0, "+0" é um número plausível, e
// a tela fica mentindo sem erro em lugar nenhum.

// Ausência de atributo NÃO é zero.
//
// Provado VERMELHO por sabotagem, e o vermelho aqui é de COMPILAÇÃO, que é o
// melhor tipo: trocar `Inteligencia *int` por `int` em `verbete` faz este
// arquivo parar de compilar (`invalid operation: m.Inteligencia == nil`). O
// guarda transforma uma perda silenciosa — `null` virando 0 e "+0" afirmando
// que o Zumbi tem a média de um humano — num erro que impede o build.
//
// Sabotar só o LADO MEDIDO é o que faz a sabotagem significar alguma coisa:
// mudar a struct e o teste junto deixa o contador em zero por construção do
// próprio remendo, e não pelo parse.
func TestTheEmDashSurvivesTheParse(t *testing.T) {
	noIntelligence := 0
	noStrength := 0
	for _, m := range book.Creatures() {
		if m.Intelligence == nil {
			noIntelligence++
		}
		if m.Strength == nil {
			noStrength++
		}
	}
	// Os números vêm do dado de hoje. Eles são um DETECTOR de regressão do
	// parse, não uma transcrição do livro: se o embed voltar, os dois caem a
	// zero de uma vez.
	if noIntelligence == 0 || noStrength == 0 {
		t.Fatalf("nenhum travessão sobreviveu ao parse: %d sem Int, %d sem For — "+
			"o campo virou `int` e `null` virou zero", noIntelligence, noStrength)
	}
	if got := book.WithSignPtr(nil); got != "—" {
		t.Errorf("book.WithSignPtr(nil) = %q, quero o travessão %q", got, "—")
	}
}

// `bookPage`, `equipamento` e `tesouro`
// não existem no `CreatureBlock` (ou existem com OUTRO nome), e o
// `encoding/json` os deixaria vazios em silêncio.
func TestTheFieldsTheEmbedWouldLoseAreThere(t *testing.T) {
	all := book.Creatures()
	if len(all) == 0 {
		t.Fatal("bestiário vazio: o catálogo não carregou")
	}
	var noPage, withEquipment, withTreasure int
	for _, m := range all {
		if m.BookPage == 0 {
			noPage++
		}
		if m.Equipment != "" {
			withEquipment++
		}
		if m.Treasure != "" {
			withTreasure++
		}
	}
	if noPage > 0 {
		t.Errorf("%d criaturas sem `bookPage` — a linha mostra \"p0\"", noPage)
	}
	if withEquipment == 0 {
		t.Error("nenhuma criatura com `equipamento`: o campo do livro é `equipamento` " +
			"e o do bloco do mestre é `equipment` — nomes diferentes não casam")
	}
	if withTreasure == 0 {
		t.Error("nenhuma criatura com `tesouro`: mesmo motivo, o bloco do mestre diz `treasure`")
	}
}

// A ordem é REGRA, não apresentação — o
// mestre procura nível de ameaça primeiro.
func TestTheOrderIsByChallengeAndThenByName(t *testing.T) {
	outside := book.FilterCreatures(book.Creatures(), book.CreatureFilter{NDMin: book.CRMin, NDMax: book.CRMax})
	if len(outside) < 2 {
		t.Fatalf("o bestiário devolveu %d criaturas", len(outside))
	}
	for i := 1; i < len(outside); i++ {
		anterior, current := outside[i-1], outside[i]
		if current.ND < anterior.ND {
			t.Fatalf("ND fora de ordem em %d: %s (ND %v) depois de %s (ND %v)",
				i, current.Name, current.ND, anterior.Name, anterior.ND)
		}
	}
}

// A faixa vem da URL, que qualquer um
// edita à mão. Um 999 ou um texto esconderia TODAS as criaturas, e a tela leria
// como "bestiário vazio" em vez de "filtro absurdo".
func TestAnAbsurdRangeDoesNotEmptyTheBestiary(t *testing.T) {
	cases := []struct{ name, min, max string }{
		{"texto no lugar do número", "abc", "xyz"},
		{"acima do teto do livro", "999", "9999"},
		{"negativo", "-5", ""},
		{"vazio", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			min, max := book.CRRange(c.min, c.max)
			if min != book.CRMin || max != book.CRMax {
				t.Fatalf("faixaDeND(%q, %q) = %v..%v, quero a faixa inteira %v..%v",
					c.min, c.max, min, max, book.CRMin, book.CRMax)
			}
		})
	}
}

// Min 10 e max 2 devolve lista VAZIA, e a tela diz "Nenhuma criatura casa com os
// filtros".
//
// "Consertar" isso devolvendo a faixa inteira faz o filtro MENTIR: pedir 10..2 e
// receber as 80 é pior que receber nenhuma.
func TestAnInvertedRangeReturnsEmpty(t *testing.T) {
	min, max := book.CRRange("10", "2")
	outside := book.FilterCreatures(book.Creatures(), book.CreatureFilter{NDMin: min, NDMax: max})
	if len(outside) != 0 {
		t.Fatalf("faixa invertida devolveu %d criaturas, quero nenhuma", len(outside))
	}
}

// Tipo vazio quer dizer TODO tipo, não nenhum: sem crachá aceso o filtro não filtra
// por tipo, e tratar vazio como "nenhum" mostraria bestiário vazio a quem não
// escolheu nada.
func TestAnEmptyTypeMeansEveryType(t *testing.T) {
	all := book.Creatures()
	noKind := book.FilterCreatures(all, book.CreatureFilter{NDMax: book.CRMax})
	if len(noKind) != len(all) {
		t.Fatalf("sem tipo escolhido vieram %d de %d criaturas", len(noKind), len(all))
	}
	oneKind := book.FilterCreatures(all, book.CreatureFilter{Kinds: []string{"animal"}, NDMax: book.CRMax})
	if len(oneKind) == 0 || len(oneKind) == len(all) {
		t.Fatalf("filtrar por animal devolveu %d de %d — o filtro não filtrou", len(oneKind), len(all))
	}
	for _, m := range oneKind {
		if m.Kind != "animal" {
			t.Fatalf("%s é %q e passou pelo filtro de animal", m.Name, m.Kind)
		}
	}
}

// "ND 0.25" não existe em Tormenta 20 — a mesa
// diz "ND 1/4", e um decimal na linha lê como artefato de arredondamento.
func TestAChallengeBelowOneComesOutAsAFraction(t *testing.T) {
	cases := map[float64]string{0.25: "1/4", 0.5: "1/2", 1: "1", 3: "3", 20: "20"}
	for nd, want := range cases {
		if got := book.CRWritten(nd); got != want {
			t.Errorf("ndEscrito(%v) = %q, quero %q", nd, got, want)
		}
	}
}

// ── a cena pelo fio ──────────────────────────────────────────────────────────

// TODO tipo que o livro usa está no trilho e tem rótulo.
//
// O defeito é um renome que casa DENTRO das strings e troca a chave do mapa de
// rótulos: o crachá passa a filtrar por um tipo que nenhuma criatura tem e
// devolve bestiário vazio. Nenhum guarda que use "animal" como tipo de exemplo
// pega isso — **um guarda só mede o que ele VISITA**, e a tabela inteira é o que
// precisa ser visitada.
//
// A direção da asserção importa. Não é "todo tipo do trilho tem criatura" —
// `planar` existe no domínio e tem ZERO verbetes hoje, e exigir criatura o
// faria falhar por um vazio legítimo. É o contrário: **todo tipo que o LIVRO
// usa precisa estar no trilho e ter rótulo**, senão existe criatura que ninguém
// consegue filtrar e que mostra o dado cru na linha.
func TestEveryBookTypeIsOnTheRailAndHasALabel(t *testing.T) {
	inBook := map[string]int{}
	for _, m := range book.Creatures() {
		inBook[m.Kind]++
	}
	if len(inBook) == 0 {
		t.Fatal("bestiário vazio: o catálogo não carregou")
	}
	for kind, howMany := range inBook {
		if !slices.Contains(book.CreatureTypes, kind) {
			t.Errorf("%d criaturas são do tipo %q e o trilho não o oferece — ninguém consegue filtrá-las",
				howMany, kind)
		}
		if label, ok := book.TypeLabels[kind]; !ok || label == kind {
			t.Errorf("o tipo %q (%d criaturas) sai na tela como o dado cru %q",
				kind, howMany, book.TypeName(kind))
		}
	}
	// E o outro lado: crachá do trilho sem rótulo é botão com nome de campo de
	// banco. Aqui é o trilho inteiro, `planar` incluído.
	for _, kind := range book.CreatureTypes {
		if _, ok := book.TypeLabels[kind]; !ok {
			t.Errorf("o trilho oferece %q e não há rótulo para ele", kind)
		}
	}
}

// Uma base vazia produz `@get(”)`, que o navegador resolve para a página ATUAL:
// o filtro pareceria funcionar — a página recarrega — e não filtraria nada. É o
// defeito silencioso desta forma, e a resposta é recusar em vez de escolher um
// padrão que só um dos dois chamadores quer.
func TestTheBestiaryBaseHasNoDefault(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("uma cena sem Base foi montada em silêncio")
		}
	}()
	_ = BestiaryView{}.BestiaryBase()
}

// E a cena do mestre continua falando para a rota dela: o campo tem de chegar
// preenchido, senão cai no caso de cima.
func TestTheGmSceneTalksToTheGmRoute(t *testing.T) {
	v := cenaSemLivro().loadBestiary(BestiaryCriteria{CRMax: 20})
	if v.Base != routes.MasterBestiary {
		t.Errorf("a cena do mestre nasceu com Base %q", v.Base)
	}
}

// O trilho lista TODOS os catálogos.
//
// Por AMOSTRAGEM sobre `collectionTabs` e não por lista escrita à mão: um número
// de paradas escrito à mão fica vermelho por envelhecer, e não por defeito. Aqui
// o catálogo novo entra medido no dia em que entra na lista.
//
// A GEOMETRIA (nenhuma parada escapa da janela, em qualquer largura) fica no
// e2e: é caixa contra caixa, e em jsdom todo elemento mede zero.
func TestTheRailHasOneStopPerCatalog(t *testing.T) {
	body := pedeNaCena(t, "/mestre/condicoes").Body.String()

	for _, a := range collectionTabs {
		if !strings.Contains(body, `href="/mestre/`+a.ID+`"`) {
			t.Errorf("o catálogo %q não tem parada no trilho", a.ID)
		}
	}
	// O bestiário é catálogo como os outros, e as duas ferramentas
	// são a outra seção do trilho — se alguma sumir, o mestre perde a porta.
	for _, stop := range []string{"bestiario", "encontros", "improviso"} {
		if !strings.Contains(body, `href="/mestre/`+stop+`"`) {
			t.Errorf("a parada %q sumiu do trilho", stop)
		}
	}
}
