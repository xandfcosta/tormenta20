package api

import (
	"context"
	"net/http"
	"strconv"
	"testing"
)

// O dano e o ajuste de vitais pelo ROUTER real.
//
// As duas regras — dano nunca passa de zero PV, e PV atual não passa do máximo —
// existiam só como funções alcançáveis por chamada direta: nenhum teste mandava
// uma REQUISIÇÃO. Uma rota registrada fora do grupo protegido, ou um handler que
// esquece o dono do personagem, não aparecia em lugar nenhum.

func vitalsPath(id int64) string {
	return "/characters/" + strconv.FormatInt(id, 10) + "/vitals"
}

func damagePath(id int64) string {
	return "/characters/" + strconv.FormatInt(id, 10) + "/damage"
}

func vitalsOf(t *testing.T, s *Server, id int64) (hp, mp int64) {
	t.Helper()
	row, err := s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler personagem %d: %v", id, err)
	}
	return row.Hpcurrent, row.Mpcurrent
}

func TestApplyDamageHTTP(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono-vitais@t.com")
	estranho := seedUser(t, s, "estranho@t.com")

	t.Run("o dano tira PV e responde 200", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Alvo", 20, 30, 5, 10)
		rec := authed(t, s, owner, http.MethodPost, damagePath(char), `{"amount":7}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
		}
		if hp, _ := vitalsOf(t, s, char); hp != 13 {
			t.Errorf("PV=%d, queria 13", hp)
		}
	})

	// Overkill não vira PV negativo: a ficha mostraria "-12/30" e o jogador não
	// sabe se morreu ou se a conta quebrou.
	t.Run("dano maior que a vida para em zero", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Quase morto", 5, 30, 5, 10)
		if rec := authed(t, s, owner, http.MethodPost, damagePath(char), `{"amount":40}`); rec.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
		}
		if hp, _ := vitalsOf(t, s, char); hp != 0 {
			t.Errorf("PV=%d, queria 0", hp)
		}
	})

	t.Run("dano zero ou negativo é recusado, nomeando o campo", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Intacto", 20, 30, 5, 10)
		for _, corpo := range []string{`{"amount":0}`, `{"amount":-3}`, `{}`} {
			rec := authed(t, s, owner, http.MethodPost, damagePath(char), corpo)
			if rec.Code == http.StatusOK {
				t.Errorf("%s: aceitou", corpo)
				continue
			}
			if _, ok := fieldErrors(t, rec)["amount"]; !ok {
				t.Errorf("%s: erros=%s, queria o campo amount", corpo, rec.Body.String())
			}
		}
		if hp, _ := vitalsOf(t, s, char); hp != 20 {
			t.Errorf("PV=%d — a recusa mexeu na ficha mesmo assim", hp)
		}
	})

	// A trava de verdade: personagem de outro dono.
	t.Run("estranho não fere personagem alheio", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Do dono", 20, 30, 5, 10)
		rec := authed(t, s, estranho, http.MethodPost, damagePath(char), `{"amount":5}`)
		if rec.Code == http.StatusOK {
			t.Fatal("um estranho feriu personagem alheio")
		}
		if hp, _ := vitalsOf(t, s, char); hp != 20 {
			t.Errorf("PV=%d — recusou com %d e gravou mesmo assim", hp, rec.Code)
		}
	})
}

func TestUpdateVitalsHTTP(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono-patch@t.com")

	t.Run("ajusta PV e PM", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Curado", 5, 30, 2, 10)
		rec := authed(t, s, owner, http.MethodPatch, vitalsPath(char), `{"hpCurrent":28,"mpCurrent":9}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
		}
		hp, mp := vitalsOf(t, s, char)
		if hp != 28 || mp != 9 {
			t.Errorf("PV=%d PM=%d, queria 28/9", hp, mp)
		}
	})

	// O mesmo teto da criação, agora na edição: a ficha não pode guardar 99/30.
	t.Run("PV acima do máximo é recusado e nada é gravado", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Teto", 10, 30, 2, 10)
		rec := authed(t, s, owner, http.MethodPatch, vitalsPath(char), `{"hpCurrent":99}`)
		if rec.Code == http.StatusOK {
			t.Fatal("aceitou PV acima do máximo")
		}
		if _, ok := fieldErrors(t, rec)["hpCurrent"]; !ok {
			t.Errorf("erros=%s, queria o campo hpCurrent", rec.Body.String())
		}
		if hp, _ := vitalsOf(t, s, char); hp != 10 {
			t.Errorf("PV=%d — a recusa gravou mesmo assim", hp)
		}
	})

	t.Run("PV negativo é recusado", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Negativo", 10, 30, 2, 10)
		if rec := authed(t, s, owner, http.MethodPatch, vitalsPath(char), `{"hpCurrent":-1}`); rec.Code == http.StatusOK {
			t.Fatal("aceitou PV negativo")
		}
		if hp, _ := vitalsOf(t, s, char); hp != 10 {
			t.Errorf("PV=%d, queria 10 intacto", hp)
		}
	})
}
