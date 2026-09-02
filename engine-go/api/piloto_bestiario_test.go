package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"t20engine/book"
	"testing"
)

// O bestiário do livro lido pelo servidor (ALE-257).
//
// O guia manda validar catálogo por SCHEMA no despejo e prender só a EXCEÇÃO —
// a armadilha da tabela —, nunca repetir a tabela inteira num `expect` por
// campo. Aqui a exceção é o TRAVESSÃO, e ele merece guarda porque a perda dele
// é invisível: um `int` recebendo `null` vira 0, "+0" é um número plausível, e
// a tela fica mentindo sem erro em lugar nenhum.

// TestTheEmDashSurvivesTheParse: ausência de atributo NÃO é zero.
//
// Provado VERMELHO por sabotagem, e o vermelho aqui é de COMPILAÇÃO, que é o
// melhor tipo: trocar `Inteligencia *int` por `int` em `verbete` faz este
// arquivo parar de compilar (`invalid operation: m.Inteligencia == nil`). O
// guarda transforma uma perda silenciosa — `null` virando 0 e "+0" afirmando
// que o Zumbi tem a média de um humano (ALE-151) — num erro que impede o build.
//
// Tentei primeiro provar o vermelho mudando a struct E o teste junto, para o
// teste continuar compilando com `int`. Não vale: o contador ficava zero por
// construção do meu próprio remendo, não pelo parse. Sabotar só o lado medido é
// o que faz a sabotagem significar alguma coisa.
func TestTheEmDashSurvivesTheParse(t *testing.T) {
	semInteligencia := 0
	semForca := 0
	for _, m := range book.Creatures() {
		if m.Inteligencia == nil {
			semInteligencia++
		}
		if m.Forca == nil {
			semForca++
		}
	}
	// Os números vêm do dado de hoje. Eles são um DETECTOR de regressão do
	// parse, não uma transcrição do livro: se o embed voltar, os dois caem a
	// zero de uma vez.
	if semInteligencia == 0 || semForca == 0 {
		t.Fatalf("nenhum travessão sobreviveu ao parse: %d sem Int, %d sem For — "+
			"o campo virou `int` e `null` virou zero", semInteligencia, semForca)
	}
	if got := book.WithSignPtr(nil); got != "—" {
		t.Errorf("comSinal(nil) = %q, quero o travessão %q", got, "—")
	}
}

// TestTheFieldsTheEmbedWouldLoseAreThere: `bookPage`, `equipamento` e `tesouro`
// não existem no `CreatureBlock` (ou existem com OUTRO nome), e o
// `encoding/json` os deixaria vazios em silêncio.
func TestTheFieldsTheEmbedWouldLoseAreThere(t *testing.T) {
	todas := book.Creatures()
	if len(todas) == 0 {
		t.Fatal("bestiário vazio: o catálogo não carregou")
	}
	var semPagina, comEquipamento, comTesouro int
	for _, m := range todas {
		if m.BookPage == 0 {
			semPagina++
		}
		if m.Equipamento != "" {
			comEquipamento++
		}
		if m.Tesouro != "" {
			comTesouro++
		}
	}
	if semPagina > 0 {
		t.Errorf("%d criaturas sem `bookPage` — a linha mostra \"p0\"", semPagina)
	}
	if comEquipamento == 0 {
		t.Error("nenhuma criatura com `equipamento`: o campo do livro é `equipamento` " +
			"e o do bloco do mestre é `equipment` — nomes diferentes não casam")
	}
	if comTesouro == 0 {
		t.Error("nenhuma criatura com `tesouro`: mesmo motivo, o bloco do mestre diz `treasure`")
	}
}

// TestTheOrderIsByChallengeAndThenByName: a ordem é REGRA, não apresentação — o
// mestre procura nível de ameaça primeiro.
func TestTheOrderIsByChallengeAndThenByName(t *testing.T) {
	fora := book.FilterCreatures(book.Creatures(), book.CreatureFilter{NDMin: book.CRMin, NDMax: book.CRMax})
	if len(fora) < 2 {
		t.Fatalf("o bestiário devolveu %d criaturas", len(fora))
	}
	for i := 1; i < len(fora); i++ {
		anterior, atual := fora[i-1], fora[i]
		if atual.ND < anterior.ND {
			t.Fatalf("ND fora de ordem em %d: %s (ND %v) depois de %s (ND %v)",
				i, atual.Name, atual.ND, anterior.Name, anterior.ND)
		}
	}
}

// TestAnAbsurdRangeDoesNotEmptyTheBestiary: a faixa vem da URL, que qualquer um
// edita à mão. Um 999 ou um texto esconderia TODAS as criaturas, e a tela leria
// como "bestiário vazio" em vez de "filtro absurdo".
func TestAnAbsurdRangeDoesNotEmptyTheBestiary(t *testing.T) {
	casos := []struct{ nome, min, max string }{
		{"texto no lugar do número", "abc", "xyz"},
		{"acima do teto do livro", "999", "9999"},
		{"negativo", "-5", ""},
		{"vazio", "", ""},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			min, max := book.CRRange(c.min, c.max)
			if min != book.CRMin || max != book.CRMax {
				t.Fatalf("faixaDeND(%q, %q) = %v..%v, quero a faixa inteira %v..%v",
					c.min, c.max, min, max, book.CRMin, book.CRMax)
			}
		})
	}
}

// TestAnInvertedRangeReturnsEmpty prende o PORTE, não uma melhoria.
//
// Min 10 e max 2 devolve lista vazia, e a tela diz "Nenhuma criatura casa com
// os filtros". A primeira versão desta camada "consertava" isso devolvendo a
// faixa inteira — o que faz o filtro MENTIR: pedir 10..2 e receber as 80 é pior
// que receber nenhuma. Se alguém quiser mudar, que mude nas DUAS telas.
func TestAnInvertedRangeReturnsEmpty(t *testing.T) {
	min, max := book.CRRange("10", "2")
	fora := book.FilterCreatures(book.Creatures(), book.CreatureFilter{NDMin: min, NDMax: max})
	if len(fora) != 0 {
		t.Fatalf("faixa invertida devolveu %d criaturas, quero nenhuma", len(fora))
	}
}

// TestAnEmptyTypeMeansEveryType, não nenhum: sem crachá aceso o filtro não filtra
// por tipo, e tratar vazio como "nenhum" mostraria bestiário vazio a quem não
// escolheu nada.
func TestAnEmptyTypeMeansEveryType(t *testing.T) {
	todas := book.Creatures()
	semTipo := book.FilterCreatures(todas, book.CreatureFilter{NDMax: book.CRMax})
	if len(semTipo) != len(todas) {
		t.Fatalf("sem tipo escolhido vieram %d de %d criaturas", len(semTipo), len(todas))
	}
	umTipo := book.FilterCreatures(todas, book.CreatureFilter{Tipos: []string{"animal"}, NDMax: book.CRMax})
	if len(umTipo) == 0 || len(umTipo) == len(todas) {
		t.Fatalf("filtrar por animal devolveu %d de %d — o filtro não filtrou", len(umTipo), len(todas))
	}
	for _, m := range umTipo {
		if m.Tipo != "animal" {
			t.Fatalf("%s é %q e passou pelo filtro de animal", m.Name, m.Tipo)
		}
	}
}

// TestAChallengeBelowOneComesOutAsAFraction: "ND 0.25" não existe em Tormenta 20 — a mesa
// diz "ND 1/4", e um decimal na linha lê como artefato de arredondamento.
func TestAChallengeBelowOneComesOutAsAFraction(t *testing.T) {
	casos := map[float64]string{0.25: "1/4", 0.5: "1/2", 1: "1", 3: "3", 20: "20"}
	for nd, quero := range casos {
		if got := book.CRWritten(nd); got != quero {
			t.Errorf("ndEscrito(%v) = %q, quero %q", nd, got, quero)
		}
	}
}

// ── a cena pelo fio ──────────────────────────────────────────────────────────

func pedeNoMestre(t *testing.T, s *Server, userID int64, metodo, caminho string, sinais string) *httptest.ResponseRecorder {
	t.Helper()
	u, err := s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.signToken(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(metodo, caminho, strings.NewReader(sinais))
	req.Header.Set("Authorization", "Bearer "+token)
	if sinais != "" {
		// É assim que o Datastar manda os sinais num POST: corpo JSON e o
		// cabeçalho que distingue o remendo da carga fria.
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("datastar-request", "true")
	}
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	return rec
}

// TestTheBestiaryOpensWithTheWholeBook: a carga fria desenha a lista, e o painel
// já vem com uma criatura. Painel vazio ao lado de lista cheia parece defeito.
func TestTheBestiaryOpensWithTheWholeBook(t *testing.T) {
	s := newTestServer(t)
	quemQuerQueSeja := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, quemQuerQueSeja, "GET", "/mestre/bestiario", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	corpo := rec.Body.String()
	primeira := book.FilterCreatures(book.Creatures(), book.CreatureFilter{NDMax: book.CRMax})[0]
	if !strings.Contains(corpo, primeira.Name) {
		t.Errorf("a primeira criatura (%s) não está na página", primeira.Name)
	}
	// O bloco à direita, não só a linha da lista: o `Deslocamento` só aparece lá.
	if !strings.Contains(corpo, primeira.Deslocamento) {
		t.Error("o painel da criatura escolhida não foi desenhado")
	}
	if !strings.Contains(corpo, "Ferramentas do mestre") {
		t.Error("a trilha de ferramentas não foi desenhada")
	}
}

// TestTheSearchIsAnAddress: `?busca=` na URL tem de valer na carga FRIA, senão o
// link colado no chat da mesa abre o bestiário inteiro.
func TestTheSearchIsAnAddress(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/mestre/bestiario?busca=ogro", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	esperado := book.FilterCreatures(book.Creatures(), book.CreatureFilter{Busca: "ogro", NDMax: book.CRMax})
	if len(esperado) == 0 {
		t.Fatal("a busca por ogro não casa com nada: o dado mudou e este teste perdeu o sentido")
	}
	corpo := rec.Body.String()
	if !strings.Contains(corpo, fmt.Sprintf("%d de %d", len(esperado), len(book.Creatures()))) {
		t.Errorf("a contagem não reflete a busca; queria %d de %d", len(esperado), len(book.Creatures()))
	}
}

// TestAnInventedTypeIsRefusedByTheGesture: no POST a recusa é DURA, porque ali
// alguém está agindo. Na URL o tipo desconhecido é só descartado — ver
// `tiposConhecidos`.
func TestAnInventedTypeIsRefusedByTheGesture(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/bestiario/tipo/dragao-roxo", "{}")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, queria 400 — tipo inventado passou", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "dragao-roxo") {
		t.Error("a recusa não diz qual tipo foi recusado")
	}
}

// TestTheTypeBadgeTogglesWithoutNavigating: o POST devolve um REMENDO (SSE), não uma
// página — recarregar no meio de uma lista perderia a posição de quem lê.
func TestTheTypeBadgeTogglesWithoutNavigating(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/bestiario/tipo/animal", `{"tipos":[]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type %q — o gesto navegou em vez de remendar", ct)
	}
	corpo := rec.Body.String()
	sos := book.FilterCreatures(book.Creatures(), book.CreatureFilter{Tipos: []string{"animal"}, NDMax: book.CRMax})
	if !strings.Contains(corpo, fmt.Sprintf("%d de %d", len(sos), len(book.Creatures()))) {
		t.Errorf("o remendo não filtrou por animal; queria %d de %d", len(sos), len(book.Creatures()))
	}
}

// TestTurningTheBadgeOffGoesBackToTheWholeBook: a álgebra do conjunto é do servidor,
// e o crachá aceso que chega nos sinais tem de SAIR.
func TestTurningTheBadgeOffGoesBackToTheWholeBook(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/bestiario/tipo/animal", `{"tipos":["animal"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	total := len(book.Creatures())
	if !strings.Contains(rec.Body.String(), fmt.Sprintf("%d de %d", total, total)) {
		t.Errorf("desligar o crachá não devolveu as %d criaturas", total)
	}
}

// TestTheGmAloneStillReachesTheBestiary: `/mestre` não é tela — a trilha sempre tem
// uma ferramenta em cena, e é a mesma que a SPA abre.
func TestTheGmAloneStillReachesTheBestiary(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "GET", "/mestre", "")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d, queria 303", rec.Code)
	}
	if destino := rec.Header().Get("Location"); destino != "/mestre/bestiario" {
		t.Errorf("foi para %q", destino)
	}
}

// TestEveryBookTypeIsOnTheRailAndHasALabel — o guarda que faltava, e que teria pego um
// defeito que os outros sete não pegaram.
//
// Um `sed` meu de renomear tipo (`monstro` → `verbete`, para seguir o
// glossário) casou DENTRO das strings e trocou a chave do mapa de rótulos e a
// entrada do trilho. Efeito: 27 das 80 criaturas — um terço do bestiário —
// passaram a mostrar o tipo em caixa baixa, e o crachá "Monstro" filtrava por
// um tipo que nenhuma criatura tem, devolvendo bestiário vazio.
//
// Nenhum dos guardas existentes pegou, e o motivo é instrutivo: todos usavam
// "animal" como tipo de exemplo. **Um guarda só mede o que ele VISITA**, e
// nenhum visitava a tabela inteira.
//
// A direção da asserção importa. Não é "todo tipo do trilho tem criatura" —
// `planar` existe no domínio e tem ZERO verbetes hoje, e exigir criatura o
// faria falhar por um vazio legítimo. É o contrário: **todo tipo que o LIVRO
// usa precisa estar no trilho e ter rótulo**, senão existe criatura que ninguém
// consegue filtrar e que mostra o dado cru na linha.
func TestEveryBookTypeIsOnTheRailAndHasALabel(t *testing.T) {
	noLivro := map[string]int{}
	for _, m := range book.Creatures() {
		noLivro[m.Tipo]++
	}
	if len(noLivro) == 0 {
		t.Fatal("bestiário vazio: o catálogo não carregou")
	}
	for tipo, quantas := range noLivro {
		if !slices.Contains(book.CreatureTypes, tipo) {
			t.Errorf("%d criaturas são do tipo %q e o trilho não o oferece — ninguém consegue filtrá-las",
				quantas, tipo)
		}
		if rotulo, ok := book.TypeLabels[tipo]; !ok || rotulo == tipo {
			t.Errorf("o tipo %q (%d criaturas) sai na tela como o dado cru %q",
				tipo, quantas, book.TypeName(tipo))
		}
	}
	// E o outro lado: crachá do trilho sem rótulo é botão com nome de campo de
	// banco. Aqui é o trilho inteiro, `planar` incluído.
	for _, tipo := range book.CreatureTypes {
		if _, ok := book.TypeLabels[tipo]; !ok {
			t.Errorf("o trilho oferece %q e não há rótulo para ele", tipo)
		}
	}
}

// TestTheBestiaryBaseHasNoDefault.
//
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
	_ = bestiarioView{}.bestiarioBase()
}

// E a cena do mestre continua falando para a rota dela: o refator trocou o
// literal por um campo, e este guarda prende que o campo chegou preenchido.
func TestTheGmSceneTalksToTheGmRoute(t *testing.T) {
	v := (&Server{}).carregaBestiario(criteriosDoBestiario{NDMax: 20})
	if v.Base != rotaDoBestiarioDoMestre {
		t.Errorf("a cena do mestre nasceu com Base %q", v.Base)
	}
}

// TestTheRailHasOneStopPerCatalog: o trilho lista TODOS os catálogos.
//
// Por AMOSTRAGEM sobre `abasDoAcervo` e não por lista escrita à mão: o e2e que
// media isto contava "onze paradas", e as duas que nasceram depois (escolas e
// perícias) só o denunciaram quando ele ficou vermelho por um número velho.
// Aqui o catálogo novo entra medido no dia em que entra na lista — que é o que
// devolve a amostragem no lugar da enumeração.
//
// A GEOMETRIA (nenhuma parada escapa da janela, em qualquer largura) fica no
// e2e: é caixa contra caixa, e em jsdom todo elemento mede zero.
func TestTheRailHasOneStopPerCatalog(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/condicoes", "").Body.String()

	for _, a := range abasDoAcervo {
		if !strings.Contains(corpo, `href="/mestre/`+a.ID+`"`) {
			t.Errorf("o catálogo %q não tem parada no trilho", a.ID)
		}
	}
	// O bestiário é catálogo como os outros (ALE-264), e as duas ferramentas
	// são a outra seção do trilho — se alguma sumir, o mestre perde a porta.
	for _, parada := range []string{"bestiario", "encontros", "improviso"} {
		if !strings.Contains(corpo, `href="/mestre/`+parada+`"`) {
			t.Errorf("a parada %q sumiu do trilho", parada)
		}
	}
}
