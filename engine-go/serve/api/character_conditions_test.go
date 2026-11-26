package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/catalog"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
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
	character := seedCharacter(t, s, owner, "Herói")
	path := "/personagens/" + id64(character) + "/conditions"

	ids := catalog.ConditionIDs()
	if len(ids) < 30 {
		t.Fatalf("o catálogo devolveu %d condições — o parse falhou", len(ids))
	}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			body, _ := json.Marshal(map[string][]string{"activeConditions": {id}})

			rec := authed(t, s, owner, http.MethodPatch, path, string(body))

			// A que o app deriva (Sobrecarregado, da carga) não entra por porta nenhuma (`engine.HandMarkRefusal`).
			if engine.HandMarkRefusal(id) != nil {
				if rec.Code != http.StatusBadRequest {
					t.Errorf("condição derivada %q aceita com %d: %s", id, rec.Code, rec.Body.String())
				}
				return
			}
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
	character := seedCharacter(t, s, owner, "Herói")

	rec := authed(t, s, owner, http.MethodPatch, "/personagens/"+id64(character)+"/conditions",
		`{"activeConditions":["caido","enfeiticado","voando"]}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "voando") {
		t.Errorf("a mensagem tem de nomear o id ofensivo, veio %s", rec.Body.String())
	}
}

// A TROCA DA LISTA PERGUNTA O MESMO que os gestos (ver GLOSSARY, **condição**), e só pelo que ENTRA: o osteon não
// recebe Envenenado (p228), e o que já estava gravado passa direto.
func TestTheConditionListAsksWhoCanReceiveWhatEnters(t *testing.T) {
	f := newSceneFixture(t)
	seedRaca(t, f.s, f.charID, "Osteon")
	path := "/personagens/" + id64(f.charID) + "/conditions"

	rec := authed(t, f.s, f.player, http.MethodPatch, path, `{"activeConditions":["envenenado"]}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "veneno") {
		t.Errorf("a lista deu Envenenado ao osteon: %d %s", rec.Code, rec.Body.String())
	}
	if got := conditions(t, f, f.charID); len(got) != 0 {
		t.Fatalf("a recusa gravou assim mesmo: %v", got)
	}

	if err := sheet.SaveConditions(t.Context(), f.s.queries, f.charID, []string{"envenenado"}); err != nil {
		t.Fatalf("gravar a marca antiga: %v", err)
	}
	rec = authed(t, f.s, f.player, http.MethodPatch, path, `{"activeConditions":["envenenado","caido"]}`)
	if rec.Code != http.StatusOK {
		t.Errorf("a marca antiga barrou a lista que só acrescenta Caído: %d %s", rec.Code, rec.Body.String())
	}
	if got := conditions(t, f, f.charID); len(got) != 2 {
		t.Errorf("a ficha ficou com %v; want a marca antiga e o Caído", got)
	}
}