package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
