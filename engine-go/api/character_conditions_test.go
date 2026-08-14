package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"t20engine/catalog"
)

// A validação de condições tinha uma CÓPIA da tabela do livro escrita à mão na
// API: 34 ids ao lado das 35 do catálogo. A que faltava era `enfeitiçado`, e
// aplicá-la dava 400 — na ficha do jogador e na do mestre.
//
// Este teste PERCORRE o catálogo em vez de repetir a lista, que é o que o
// repositório faz com dado transcrito do livro: repetir a tabela num `expect`
// só recria a cópia que desviou.
func TestEveryCatalogConditionIsAccepted(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	character := seedCharacter(t, s, owner, "Herói", 20, 20, 10, 10)
	path := "/characters/" + id64(character) + "/conditions"

	ids := catalog.ConditionIDs()
	if len(ids) < 30 {
		t.Fatalf("o catálogo devolveu %d condições — o parse falhou", len(ids))
	}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			body, _ := json.Marshal(map[string][]string{"activeConditions": {id}})

			rec := authed(t, s, owner, http.MethodPatch, path, string(body))

			if rec.Code != http.StatusOK {
				t.Errorf("condição %q recusada com %d: %s", id, rec.Code, rec.Body.String())
			}
		})
	}
}

// O outro lado da regra: um id que não é do livro continua sendo recusado, e a
// mensagem diz qual.
func TestAnUnknownConditionIsRejected(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	character := seedCharacter(t, s, owner, "Herói", 20, 20, 10, 10)

	rec := authed(t, s, owner, http.MethodPatch, "/characters/"+id64(character)+"/conditions",
		`{"activeConditions":["caido","enfeitiçado","voando"]}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "voando") {
		t.Errorf("a mensagem tem de nomear o id ofensivo, veio %s", rec.Body.String())
	}
}
