package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"t20engine/db"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

// Custo em PM de uma magia — a composição inteira, pelo HTTP, com o motor real.
//
// Tabela 4-1 (p170): 1º = 1 PM, 2º = 3, 3º = 6, 4º = 10, 5º = 15; truque = 0.
// Aprimoramentos (p171): "para aprimoramentos que AUMENTAM um valor você pode
// gastar aquela quantidade de PM várias vezes para acumular o aumento";
// os que MUDAM a magia nunca se acumulam.
// Teto (p224): "o máximo de PM que você pode gastar por uso é igual ao seu nível
// na classe que fornece a habilidade (mas você sempre pode usar a habilidade em
// seu CUSTO MÍNIMO)".
//
// O handler de conjurar não tinha um único teste — era a maior regra do capítulo
// 4 sem rede (ALE-105). O exemplo trabalhado da p171 exercita a composição toda
// de uma vez, e é o que separa "o teto vale sobre o TOTAL" de "o teto vale só
// sobre os aprimoramentos".

func newCastServer(t *testing.T) *Server {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "cast.db"))
	if err != nil {
		t.Fatalf("abrir banco: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	raw, err := os.ReadFile(filepath.Join("..", "parity", "_catalogs.json"))
	if err != nil {
		t.Fatalf("ler catálogos: %v (rode o harness GEN_ORACLE do front)", err)
	}
	catalogs, err := engine.PrimeEngineCatalogs(raw)
	if err != nil {
		t.Fatalf("primar catálogos: %v", err)
	}
	return NewServer(Config{JWTSecret: "test-secret", CookieName: "t20_session"}, database, catalogs)
}

// seedCaster inserts a caster with the class level, PM and one learned spell —
// the three inputs every rule below turns on.
func seedCaster(t *testing.T, s *Server, ownerID int64, className string, classLevel, mpCurrent int, spellID string) int64 {
	t.Helper()
	ctx := context.Background()
	id, err := s.queries.CreateCharacter(ctx, sqlcgen.CreateCharacterParams{
		OwnerId: ownerID, Name: "Conjurador", Origin: "Estudioso", Level: int64(classLevel),
		HpMax: 50, HpCurrent: 50, MpMax: int64(mpCurrent), MpCurrent: int64(mpCurrent),
		Intelligence: 4, Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: nowISO(), UpdatedAt: nowISO(),
	})
	if err != nil {
		t.Fatalf("semear personagem: %v", err)
	}
	if err := s.queries.CreateClass(ctx, sqlcgen.CreateClassParams{
		Characterid: id, Classname: className, Level: int64(classLevel),
	}); err != nil {
		t.Fatalf("semear classe: %v", err)
	}
	if _, err := s.queries.CreateSpell(ctx, sqlcgen.CreateSpellParams{
		Characterid: id, Catalogspellid: spellID, Prepared: 1, Learnedat: nowISO(),
	}); err != nil {
		t.Fatalf("semear magia: %v", err)
	}
	return id
}

func castSpell(t *testing.T, s *Server, userID, characterID int64, spellID, body string) *httptest.ResponseRecorder {
	t.Helper()
	path := fmt.Sprintf("/characters/%d/spells/%s/cast", characterID, spellID)
	return authed(t, s, userID, http.MethodPost, path, body)
}

func mpOf(t *testing.T, s *Server, characterID int64) int64 {
	t.Helper()
	row, err := s.queries.GetCharacter(context.Background(), characterID)
	if err != nil {
		t.Fatalf("reler personagem: %v", err)
	}
	return row.Mpcurrent
}

// Tabela 4-1. Uma tabela de seis linhas que decide o custo de TODA magia do
// jogo — cara demais para ficar só implícita nos oráculos.
func TestSpellBasePmCostTable(t *testing.T) {
	want := map[int]int{0: 0, 1: 1, 2: 3, 3: 6, 4: 10, 5: 15}
	for circle, pm := range want {
		if got := spellBasePmCost[circle]; got != pm {
			t.Errorf("círculo %d custa %d PM, want %d (Tabela 4-1, p170)", circle, got, pm)
		}
	}
	if len(spellBasePmCost) != len(want) {
		t.Errorf("a tabela tem %d círculos, want %d", len(spellBasePmCost), len(want))
	}
}

// O exemplo trabalhado do livro, p171 (quadro "Aprimoramentos Cumulativos"):
//
//	"A magia Bola de Fogo causa 6d6 pontos de dano e tem um aprimoramento que
//	 aumenta esse dano em +2d6 por +2 PM. Um arcanista de 11º nível pode gastar
//	 até 11 PM ao lançar essa magia, causando 14d6 pontos de dano."
//
// Bola de Fogo é de 2º círculo: 3 PM de base. 14d6 = 6d6 + 8d6, ou seja quatro
// acúmulos de +2 PM. 3 + 8 = 11 = o nível. A conta só fecha se o teto valer
// sobre o TOTAL — se valesse só sobre os aprimoramentos, 11 PM de acréscimo
// dariam 16d6.
func TestBolaDeFogoWorkedExample(t *testing.T) {
	s := newCastServer(t)
	owner := seedUser(t, s, "arcanista@t20.local")
	char := seedCaster(t, s, owner, "Arcanista", 11, 40, "bola-de-fogo")

	t.Run("quatro acúmulos gastam exatamente os 11 PM do teto", func(t *testing.T) {
		rec := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[{"augmentIndex":0,"stacks":4}]}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("conjurar devolveu %d: %s", rec.Code, rec.Body.String())
		}
		if got := mpOf(t, s, char); got != 29 {
			t.Errorf("PM restante = %d, want 29 (40 − 11)", got)
		}
	})

	t.Run("um acúmulo a mais estoura o teto e nada é gasto", func(t *testing.T) {
		antes := mpOf(t, s, char)
		rec := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[{"augmentIndex":0,"stacks":5}]}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("13 PM com teto 11 devolveu %d, want 400", rec.Code)
		}
		if depois := mpOf(t, s, char); depois != antes {
			t.Errorf("PM foi de %d para %d — a recusa cobrou mesmo assim", antes, depois)
		}
	})
}

// "Aprimoramentos Cumulativos. Para aprimoramentos que aumentam um valor […]
// você pode gastar aquela quantidade de PM várias vezes" — e os que MUDAM a
// magia "nunca se acumulam".
func TestAugmentStackingRules(t *testing.T) {
	s := newCastServer(t)
	owner := seedUser(t, s, "acumulos@t20.local")
	char := seedCaster(t, s, owner, "Arcanista", 11, 40, "bola-de-fogo")

	t.Run("o aprimoramento que AUMENTA acumula", func(t *testing.T) {
		antes := mpOf(t, s, char)
		rec := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[{"augmentIndex":0,"stacks":3}]}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("devolveu %d: %s", rec.Code, rec.Body.String())
		}
		// 3 de base + 3 × 2 = 9.
		if got := antes - mpOf(t, s, char); got != 9 {
			t.Errorf("gastou %d PM, want 9 (3 + 3×2)", got)
		}
	})

	// O índice 1 da Bola de Fogo é a esfera flamejante, um "muda".
	t.Run("o aprimoramento que MUDA não acumula", func(t *testing.T) {
		rec := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[{"augmentIndex":1,"stacks":2}]}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("dois acúmulos de um 'muda' devolveram %d, want 400", rec.Code)
		}
	})

	t.Run("o mesmo aprimoramento duas vezes na lista é recusado", func(t *testing.T) {
		body := `{"augments":[{"augmentIndex":0,"stacks":1},{"augmentIndex":0,"stacks":1}]}`
		if rec := castSpell(t, s, owner, char, "bola-de-fogo", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("índice repetido devolveu %d, want 400", rec.Code)
		}
	})
}

// "(mas você sempre pode usar a habilidade em seu CUSTO MÍNIMO)" — a ressalva
// entre parênteses da p224. O teto limita o que se gasta A MAIS; ele nunca torna
// a magia inconjurável.
//
// Alcançável hoje: o teto é o nível NA CLASSE que fornece a magia, e uma magia
// que não vem de classe nenhuma usa o nível de personagem. Um personagem de 2º
// nível com uma magia de 2º círculo concedida por outra fonte tem teto 2 e custo
// base 3 — sem a ressalva, ele nunca mais lança a magia que possui.
func TestMinimumCostIsAlwaysAllowed(t *testing.T) {
	s := newCastServer(t)
	owner := seedUser(t, s, "minimo@t20.local")
	// Bárbaro: a Bola de Fogo não está na lista da classe, então o teto cai no
	// nível de personagem (2), abaixo dos 3 PM de base.
	char := seedCaster(t, s, owner, "Bárbaro", 2, 20, "bola-de-fogo")

	t.Run("o custo base passa mesmo acima do teto", func(t *testing.T) {
		rec := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[]}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("custo mínimo devolveu %d: %s", rec.Code, rec.Body.String())
		}
		if got := mpOf(t, s, char); got != 17 {
			t.Errorf("PM restante = %d, want 17 (20 − 3)", got)
		}
	})

	// A ressalva cobre o MÍNIMO, e não mais que isso: um aprimoramento em cima
	// continua barrado.
	t.Run("um aprimoramento acima do mínimo continua barrado", func(t *testing.T) {
		antes := mpOf(t, s, char)
		rec := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[{"augmentIndex":0,"stacks":1}]}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("devolveu %d, want 400", rec.Code)
		}
		if depois := mpOf(t, s, char); depois != antes {
			t.Errorf("PM foi de %d para %d numa recusa", antes, depois)
		}
	})
}

// "você gasta os PM mesmo em caso de falha" (p224) pressupõe ter os PM: sem
// eles, a magia não sai e nada é cobrado.
func TestCastRefusedWithoutEnoughPm(t *testing.T) {
	s := newCastServer(t)
	owner := seedUser(t, s, "sempm@t20.local")
	char := seedCaster(t, s, owner, "Arcanista", 11, 2, "bola-de-fogo")

	rec := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("3 PM com 2 no bolso devolveu %d, want 400", rec.Code)
	}
	if got := mpOf(t, s, char); got != 2 {
		t.Errorf("PM = %d, want 2 — cobrou numa recusa", got)
	}
}
