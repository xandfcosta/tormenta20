package api

import (
	"context"
	"net/http"
	"strconv"
	"testing"
)

// O dinheiro do personagem pelo ROUTER real (ALE-215).
//
// O tibar já existia na criação (Tabela 3-1, p140) e não tinha caminho de
// edição nenhum: a Forja o escrevia uma vez e ninguém mais o tocava. O que
// estes casos guardam é a metade que a tela NÃO pode guardar — o campo é
// dinheiro, e dinheiro que aceita negativo ou infinito volta pela carga (cada
// mil moedas ocupam um espaço, p141) e envenena a ficha inteira.

func tibarPath(id int64) string {
	return "/characters/" + strconv.FormatInt(id, 10) + "/tibar"
}

func tibarOf(t *testing.T, s *Server, id int64) float64 {
	t.Helper()
	row, err := s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler personagem %d: %v", id, err)
	}
	return row.Tibar
}

func TestUpdateTibarHTTP(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono-tibar@t.com")

	t.Run("grava o valor e responde com ele", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Rico", 10, 10, 0, 0)
		rec := authed(t, s, owner, http.MethodPatch, tibarPath(char), `{"tibar":1250.5}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
		}
		if got := jsonField(t, rec, "tibar"); got != 1250.5 {
			t.Errorf("a resposta trouxe %v, queria 1250.5", got)
		}
		if got := tibarOf(t, s, char); got != 1250.5 {
			t.Errorf("gravado %v, queria 1250.5", got)
		}
	})

	// Zerar a bolsa é um valor legítimo, e não "campo ausente": quem gastou tudo
	// precisa poder dizer isso.
	t.Run("zero é um valor, não um campo vazio", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Falido", 10, 10, 0, 0)
		if rec := authed(t, s, owner, http.MethodPatch, tibarPath(char), `{"tibar":300}`); rec.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
		}
		if rec := authed(t, s, owner, http.MethodPatch, tibarPath(char), `{"tibar":0}`); rec.Code != http.StatusOK {
			t.Fatalf("recusou zerar: code=%d body=%s", rec.Code, rec.Body.String())
		}
		if got := tibarOf(t, s, char); got != 0 {
			t.Errorf("gravado %v, queria 0", got)
		}
	})

	// Dívida não existe na ficha: T$ negativo daria carga de moeda negativa, que
	// COMPRARIA espaço na mochila em vez de ocupar.
	t.Run("negativo é recusado e nada é gravado", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Devedor", 10, 10, 0, 0)
		if rec := authed(t, s, owner, http.MethodPatch, tibarPath(char), `{"tibar":500}`); rec.Code != http.StatusOK {
			t.Fatalf("preparo falhou: %s", rec.Body.String())
		}
		rec := authed(t, s, owner, http.MethodPatch, tibarPath(char), `{"tibar":-1}`)
		if rec.Code == http.StatusOK {
			t.Fatal("aceitou T$ negativo")
		}
		if _, ok := fieldErrors(t, rec)["tibar"]; !ok {
			t.Errorf("erros=%s, queria o campo tibar", rec.Body.String())
		}
		if got := tibarOf(t, s, char); got != 500 {
			t.Errorf("gravado %v — a recusa escreveu mesmo assim", got)
		}
	})

	// Um número absurdo não pode chegar ao banco: a carga lê o dinheiro (cada
	// mil moedas ocupam um espaço, p141), e T$ 1e12 viraria um bilhão de espaços
	// na mochila. Quem recusa `1e999` é o próprio decodificador, então o caso que
	// exercita o TETO é um número que decodifica bem.
	t.Run("um valor absurdo é recusado", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Dragão", 10, 10, 0, 0)
		if rec := authed(t, s, owner, http.MethodPatch, tibarPath(char), `{"tibar":1e12}`); rec.Code == http.StatusOK {
			t.Fatal("aceitou T$ 1e12")
		}
		if rec := authed(t, s, owner, http.MethodPatch, tibarPath(char), `{"tibar":1e999}`); rec.Code == http.StatusOK {
			t.Fatal("aceitou T$ 1e999")
		}
		if got := tibarOf(t, s, char); got != 0 {
			t.Errorf("gravado %v, queria 0 intacto", got)
		}
	})

	t.Run("corpo sem o campo é recusado", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Vazio", 10, 10, 0, 0)
		if rec := authed(t, s, owner, http.MethodPatch, tibarPath(char), `{}`); rec.Code == http.StatusOK {
			t.Fatal("aceitou corpo sem tibar")
		}
	})
}
