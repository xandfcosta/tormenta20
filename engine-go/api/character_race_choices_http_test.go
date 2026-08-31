package api

import (
	"context"
	"net/http"
	"strconv"
	"testing"
)

// A escolha de atributo da raça pelo PATCH de habilidades (ALE-169).
//
// A forja oferece criar o personagem com o bônus por colocar — o passo de
// Resumo diz, por escrito, "dá para criar assim e terminar na ficha". A ficha
// não tinha como terminar: este endpoint aceitava as outras cinco escolhas e
// não esta, então o personagem ficava para sempre sem os +1 que o livro manda
// aplicar ("Sua raça modifica seus atributos", p18), sem outro conserto além
// de refazer a forja inteira.

func abilitiesPath(id int64) string {
	return "/characters/" + strconv.FormatInt(id, 10) + "/abilities"
}

func raceAttrChoicesOf(t *testing.T, s *Server, id int64) string {
	t.Helper()
	row, err := s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler personagem %d: %v", id, err)
	}
	return row.Raceattributechoices
}

func TestUpdateRaceAttributeChoicesHTTP(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono-raca@t.com")

	t.Run("grava os picks e devolve o que gravou", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Humano sem os +1", 20, 20, 5, 5)
		body := `{"raceAttributeChoices":{"floatingPicks":["strength","dexterity","constitution"]}}`

		rec := authed(t, s, owner, http.MethodPatch, abilitiesPath(char), body)

		if rec.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
		}
		got := raceAttrChoicesOf(t, s, char)
		want := `{"floatingPicks":["strength","dexterity","constitution"]}`
		if got != want {
			t.Errorf("guardado=%s, queria %s", got, want)
		}
	})

	// O endpoint escreve um SUBCONJUNTO: mandar a escolha de raça não pode
	// apagar as outras cinco colunas de escolha.
	//
	// O poder é o `ataque-poderoso`, que EXISTE no livro. Aqui morava um
	// `ataque-especial` inventado, e ele parou de passar quando a fatia 8 da
	// ALE-272 fechou a fronteira das escolhas: um id que o catálogo não tem
	// agora é recusado, e o preparo do teste era justamente um.
	t.Run("não apaga as escolhas vizinhas", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Com poderes", 20, 20, 5, 5)
		// A CLASSE é o que abre a VAGA de poder. Ela entrou na fatia 8 da
		// ALE-272: com a fronteira das escolhas fechada, um personagem sem
		// classe tem zero vagas, e escolher um poder passou a ser recusado —
		// esta ficha de teste era ilegal pelo livro e ninguém reclamava.
		seedClasse(t, s, char, "Guerreiro", 2)
		if rec := authed(t, s, owner, http.MethodPatch, abilitiesPath(char),
			`{"classPowers":["ataque-poderoso"]}`); rec.Code != http.StatusOK {
			t.Fatalf("preparo: code=%d body=%s", rec.Code, rec.Body.String())
		}

		if rec := authed(t, s, owner, http.MethodPatch, abilitiesPath(char),
			`{"raceAttributeChoices":{"floatingPicks":["wisdom"]}}`); rec.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
		}

		row, err := s.queries.GetCharacter(context.Background(), char)
		if err != nil {
			t.Fatalf("ler personagem: %v", err)
		}
		if row.Classpowers != `["ataque-poderoso"]` {
			t.Errorf("classPowers=%s — a escolha vizinha foi apagada", row.Classpowers)
		}
	})

	// Guardar como veio é seguro porque quem RECUSA é o motor: uma escolha
	// inválida não vira bônus nenhum (engine/collect_rules.go, resolveFloating).
	// Este teste prende o contrato do armazenamento, não a validação.
	t.Run("a ascendência também passa", func(t *testing.T) {
		char := seedCharacter(t, s, owner, "Suraggel", 20, 20, 5, 5)

		rec := authed(t, s, owner, http.MethodPatch, abilitiesPath(char),
			`{"raceAttributeChoices":{"ascendencia":"aggelus"}}`)

		if rec.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
		}
		if got := raceAttrChoicesOf(t, s, char); got != `{"ascendencia":"aggelus"}` {
			t.Errorf("guardado=%s", got)
		}
	})
}
