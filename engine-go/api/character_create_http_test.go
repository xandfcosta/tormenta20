package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A validação de criação de personagem pelo ROUTER real. Ela recusa PV atual
// acima do máximo, classe repetida e os campos obrigatórios — e nada disso
// tinha teste: a tabela de rotas só provava o 401 do anônimo.
//
// A tela também "protege" o PV com um `max` no input, que NÃO trava nada (o
// browser deixa digitar e marca o campo inválido em silêncio). Esta é a camada
// dona da regra; a da tela é UX.

func createBody(overrides map[string]any) string {
	body := map[string]any{
		"name":              "Herói de teste",
		"races":             []string{"Humano"},
		"origin":            "Soldado",
		"classes":           []map[string]any{{"className": "Guerreiro", "level": 1}},
		"hpMax":             20,
		"hpCurrent":         20,
		"mpMax":             4,
		"mpCurrent":         4,
		"strength":          1,
		"dexterity":         0,
		"constitution":      1,
		"intelligence":      0,
		"wisdom":            0,
		"charisma":          0,
		"size":              "Médio",
		"displacement":      9,
		"trainedExpertises": []string{},
		"items":             []any{},
	}
	for key, value := range overrides {
		body[key] = value
	}
	raw, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func fieldErrors(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload struct {
		FieldErrors map[string]any `json:"fieldErrors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("resposta não é JSON (%s): %v", rec.Body.String(), err)
	}
	return payload.FieldErrors
}

func TestCreateCharacterValidation(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "forja@t.com")

	t.Run("PV atual acima do máximo é recusado, nomeando o campo", func(t *testing.T) {
		rec := authed(t, s, owner, http.MethodPost, "/characters",
			createBody(map[string]any{"hpCurrent": 99, "hpMax": 25}))

		if rec.Code == http.StatusCreated {
			t.Fatalf("criou com PV 99/25")
		}
		if _, ok := fieldErrors(t, rec)["hpCurrent"]; !ok {
			t.Errorf("erros=%v, queria um erro no campo hpCurrent", rec.Body.String())
		}
	})

	t.Run("PM atual acima do máximo é recusado", func(t *testing.T) {
		rec := authed(t, s, owner, http.MethodPost, "/characters",
			createBody(map[string]any{"mpCurrent": 40, "mpMax": 4}))

		if rec.Code == http.StatusCreated {
			t.Fatalf("criou com PM 40/4")
		}
		if _, ok := fieldErrors(t, rec)["mpCurrent"]; !ok {
			t.Errorf("erros=%v, queria um erro no campo mpCurrent", rec.Body.String())
		}
	})

	// Duas entradas da mesma classe somariam níveis duas vezes na ficha; a
	// mensagem diz o que fazer em vez de só recusar.
	t.Run("classe repetida é recusada explicando o que fazer", func(t *testing.T) {
		rec := authed(t, s, owner, http.MethodPost, "/characters", createBody(map[string]any{
			"classes": []map[string]any{
				{"className": "Guerreiro", "level": 1},
				{"className": "Guerreiro", "level": 2},
			},
		}))

		if rec.Code == http.StatusCreated {
			t.Fatalf("criou com a mesma classe duas vezes")
		}
		// A mensagem diz o que fazer, não só que deu errado.
		if got := rec.Body.String(); !strings.Contains(got, "combine levels in one entry") {
			t.Errorf("corpo=%s, queria a instrução de combinar níveis", got)
		}
	})

	t.Run("sem nome, sem raça ou sem classe é recusado", func(t *testing.T) {
		for field, override := range map[string]map[string]any{
			"name":    {"name": "   "},
			"races":   {"races": []string{}},
			"classes": {"classes": []any{}},
		} {
			rec := authed(t, s, owner, http.MethodPost, "/characters", createBody(override))
			if rec.Code == http.StatusCreated {
				t.Errorf("%s: criou mesmo assim", field)
				continue
			}
			if _, ok := fieldErrors(t, rec)[field]; !ok {
				t.Errorf("%s: erros=%s, queria o campo nomeado", field, rec.Body.String())
			}
		}
	})
}
