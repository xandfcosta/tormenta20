package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"t20engine/domain/sheet"
	"t20engine/infra/config"
	"t20engine/infra/db/dbvalue"
	"testing"

	"t20engine/domain/engine"
	"t20engine/infra/db"
	"t20engine/infra/db/sqlcgen"
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
// O exemplo trabalhado da p171 exercita a composição toda de uma vez, e é o que
// separa "o teto vale sobre o TOTAL" de "o teto vale só sobre os
// aprimoramentos".

func newCastServer(t *testing.T) *Server {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "cast.db"))
	if err != nil {
		t.Fatalf("abrir banco: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	raw, err := os.ReadFile(filepath.Join("..", "..", "parity", "_catalogs.json"))
	if err != nil {
		t.Fatalf("ler catálogos: %v (gere com `go run ./cmd/genoracle`)", err)
	}
	catalogs, err := engine.PrimeEngineCatalogs(raw)
	if err != nil {
		t.Fatalf("primar catálogos: %v", err)
	}
	return NewServer(config.Config{JWTSecret: "test-secret", CookieName: "t20_session"}, database, catalogs)
}

// seedCaster insere um conjurador com o nível de classe, os PM e uma magia
// aprendida — as três entradas de que toda regra abaixo depende.
func seedCaster(t *testing.T, s *Server, ownerID int64, className string, classLevel, mpCurrent int, spellID string) int64 {
	t.Helper()
	return seedCasterWithPowers(t, s, ownerID, className, classLevel, mpCurrent, spellID, "[]")
}

// seedCasterWithPowers é o mesmo, com poderes de classe escolhidos — o que traz
// modificadores de custo de PM para dentro da conta.
func seedCasterWithPowers(t *testing.T, s *Server, ownerID int64, className string, classLevel, mpCurrent int, spellID, classPowers string) int64 {
	t.Helper()
	ctx := context.Background()
	id, err := s.queries.CreateCharacter(ctx, sqlcgen.CreateCharacterParams{
		OwnerId: ownerID, Name: "Conjurador", Origin: "Estudioso", Level: int64(classLevel),
		HpMax: 50, HpCurrent: 50, MpMax: int64(mpCurrent), MpCurrent: int64(mpCurrent),
		Intelligence: 4, Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: classPowers, ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
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
		Characterid: id, Catalogspellid: spellID, Prepared: 1, Learnedat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("semear magia: %v", err)
	}
	return id
}

// castSpell chama a REGRA direto, e não uma rota.
//
// O que estes casos prendem nunca foi o transporte: é o teto de PM da p171, o
// empilhamento de aprimoramento da p224 e a ressalva do custo mínimo. **Teste de
// regra vive junto da regra**, e o caminho até ela é o mesmo que a cena da ficha
// usa — `castSpellForCharacter`, pelo `CastSpell` da porta.
//
// Devolve ERRO em vez de status: a recusa aqui é uma frase para uma pessoa, e é
// o handler que a traduz em 400.
func castSpell(t *testing.T, s *Server, userID, characterID int64, spellID, body string) error {
	t.Helper()
	var corpo struct {
		Augments []sheet.AugmentPick `json:"augments"`
	}
	if err := json.Unmarshal([]byte(body), &corpo); err != nil {
		t.Fatalf("corpo do caso inválido: %v", err)
	}
	row, err := s.queries.GetCharacter(context.Background(), characterID)
	if err != nil {
		t.Fatalf("ler personagem %d: %v", characterID, err)
	}
	dto, err := s.sheetRules().LoadCharacter(context.Background(), row)
	if err != nil {
		t.Fatalf("montar a ficha %d: %v", characterID, err)
	}
	// Sem `httptest.NewRequest`: a regra recebe CONTEXTO, e montar um pedido só
	// para entregá-lo era o sintoma que a ALE-347 veio tirar.
	return s.sheetRules().castSpellForCharacter(context.Background(), dto, spellID, corpo.Augments)
}

func mpOf(t *testing.T, s *Server, characterID int64) int64 {
	t.Helper()
	row, err := s.queries.GetCharacter(context.Background(), characterID)
	if err != nil {
		t.Fatalf("reler personagem: %v", err)
	}
	return row.Mpcurrent
}

// A Tabela 4-1 (p170) é prendida no `sheet`, junto da tabela — aqui ficam só os
// casos que montam um `Server` de verdade.

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
		if err := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[{"augmentIndex":0,"stacks":4}]}`); err != nil {
			t.Fatalf("conjurar devolveu %v", err)
		}
		if got := mpOf(t, s, char); got != 29 {
			t.Errorf("PM restante = %d, want 29 (40 − 11)", got)
		}
	})

	t.Run("um acúmulo a mais estoura o teto e nada é gasto", func(t *testing.T) {
		antes := mpOf(t, s, char)
		if err := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[{"augmentIndex":0,"stacks":5}]}`); err == nil {
			t.Fatal("13 PM com teto 11 devolveu — e não foi recusado")
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
		if err := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[{"augmentIndex":0,"stacks":3}]}`); err != nil {
			t.Fatalf("devolveu %v", err)
		}
		// 3 de base + 3 × 2 = 9.
		if got := antes - mpOf(t, s, char); got != 9 {
			t.Errorf("gastou %d PM, want 9 (3 + 3×2)", got)
		}
	})

	// O índice 1 da Bola de Fogo é a esfera flamejante, um "muda".
	t.Run("o aprimoramento que MUDA não acumula", func(t *testing.T) {
		if err := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[{"augmentIndex":1,"stacks":2}]}`); err == nil {
			t.Fatal("dois acúmulos de um 'muda' devolveram — e não foi recusado")
		}
	})

	t.Run("o mesmo aprimoramento duas vezes na lista é recusado", func(t *testing.T) {
		body := `{"augments":[{"augmentIndex":0,"stacks":1},{"augmentIndex":0,"stacks":1}]}`
		if err := castSpell(t, s, owner, char, "bola-de-fogo", body); err == nil {
			t.Fatal("índice repetido devolveu — e não foi recusado")
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
		if err := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[]}`); err != nil {
			t.Fatalf("custo mínimo devolveu %v", err)
		}
		if got := mpOf(t, s, char); got != 17 {
			t.Errorf("PM restante = %d, want 17 (20 − 3)", got)
		}
	})

	// A ressalva cobre o MÍNIMO, e não mais que isso: um aprimoramento em cima
	// continua barrado.
	t.Run("um aprimoramento acima do mínimo continua barrado", func(t *testing.T) {
		antes := mpOf(t, s, char)
		if err := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[{"augmentIndex":0,"stacks":1}]}`); err == nil {
			t.Fatal("devolveu — e não foi recusado")
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

	if err := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[]}`); err == nil {
		t.Fatal("3 PM com 2 no bolso devolveu — e não foi recusado")
	}
	if got := mpOf(t, s, char); got != 2 {
		t.Errorf("PM = %d, want 2 — cobrou numa recusa", got)
	}
}

// "Reduções de Custo. Reduções no custo de PM não são cumulativas. Uma
// habilidade nunca pode ter seu custo reduzido para menos de 1 PM." (p226)
//
// O defeito que isto prende é o portão de conjurar IGNORAR o modificador de
// custo que o motor calcula: um Druida de 20º nível com Força da Natureza
// ("diminui o custo de todas as suas magias em −2 PM", p63) paga preço cheio, e
// o mosaico "Custo PM" da ficha diz o contrário.
func TestPmCostReductionIsAppliedAndFloored(t *testing.T) {
	s := newCastServer(t)
	owner := seedUser(t, s, "druida@t20.local")
	char := seedCasterWithPowers(t, s, owner, "Druida", 20, 40, "bola-de-fogo",
		`["class.druida.forca-da-natureza"]`)

	t.Run("a redução sai do custo (3 PM de base − 2)", func(t *testing.T) {
		antes := mpOf(t, s, char)
		if err := castSpell(t, s, owner, char, "bola-de-fogo", `{"augments":[]}`); err != nil {
			t.Fatalf("devolveu %v", err)
		}
		if gasto := antes - mpOf(t, s, char); gasto != 1 {
			t.Errorf("gastou %d PM, want 1 (3 de base − 2 da Força da Natureza)", gasto)
		}
	})

	// O piso da p226: a redução nunca leva o custo abaixo de 1 PM. Uma magia de
	// 1º círculo custa 1, e −2 não a torna gratuita.
	t.Run("o piso de 1 PM segura a redução", func(t *testing.T) {
		if _, err := s.queries.CreateSpell(context.Background(), sqlcgen.CreateSpellParams{
			Characterid: char, Catalogspellid: "luz", Prepared: 1, Learnedat: dbvalue.NowISO(),
		}); err != nil {
			t.Fatalf("semear magia: %v", err)
		}
		antes := mpOf(t, s, char)
		if err := castSpell(t, s, owner, char, "luz", `{"augments":[]}`); err != nil {
			t.Fatalf("devolveu %v", err)
		}
		if gasto := antes - mpOf(t, s, char); gasto != 1 {
			t.Errorf("gastou %d PM, want 1 — o piso da p226 não segurou", gasto)
		}
	})
}

// O TRUQUE, p171: "Este aprimoramento transforma a magia em uma versão mais
// simples e reduz seu custo em PM para zero. Truques não podem ser usados em
// conjunto com outros aprimoramentos."
//
// São duas regras numa frase, e o servidor não aplicava nenhuma das duas
// (ALE-339). Havia um guarda com o nome certo — `spell.Circle == 0`, "truques
// não recebem aprimoramentos" — apontado para a OUTRA representação de truque, a
// magia de círculo 0, que nenhuma das 198 magias do catálogo tem. O truque de
// verdade é o APRIMORAMENTO, e são catorze.
//
// A Explosão de Chamas serve porque tem os três casos numa magia só: o truque no
// índice 0 e dois aprimoramentos comuns de 1 PM.
func TestATruqueCostsNothingAndRefusesCompany(t *testing.T) {
	s := newCastServer(t)
	owner := seedUser(t, s, "truque@t20.local")
	char := seedCaster(t, s, owner, "Arcanista", 5, 20, "explosao-de-chamas")

	// "reduz seu custo em PM para zero" é a magia INTEIRA, e não o aprimoramento:
	// modelado como um `+0 PM` o custo base de 1 PM ficaria de pé, e a versão mais
	// simples da magia sairia pelo preço da normal.
	t.Run("o truque zera o custo da conjuração", func(t *testing.T) {
		antes := mpOf(t, s, char)
		if err := castSpell(t, s, owner, char, "explosao-de-chamas", `{"augments":[{"augmentIndex":0,"stacks":1}]}`); err != nil {
			t.Fatalf("o truque devolveu %v", err)
		}
		if gasto := antes - mpOf(t, s, char); gasto != 0 {
			t.Errorf("o truque gastou %d PM, want 0 (p171)", gasto)
		}
	})

	t.Run("o truque acompanhado é recusado", func(t *testing.T) {
		antes := mpOf(t, s, char)
		corpo := `{"augments":[{"augmentIndex":0,"stacks":1},{"augmentIndex":1,"stacks":1}]}`
		if err := castSpell(t, s, owner, char, "explosao-de-chamas", corpo); err == nil {
			t.Fatal("truque + aprimoramento devolveu — e não foi recusado")
		}
		if depois := mpOf(t, s, char); depois != antes {
			t.Errorf("PM foi de %d para %d numa recusa", antes, depois)
		}
	})

	// A ordem da lista não pode decidir nada: o truque em SEGUNDO lugar é a mesma
	// combinação, e um laço que só olhasse o primeiro pedido passaria verde.
	t.Run("a ordem do pedido não escapa da regra", func(t *testing.T) {
		corpo := `{"augments":[{"augmentIndex":1,"stacks":1},{"augmentIndex":0,"stacks":1}]}`
		if err := castSpell(t, s, owner, char, "explosao-de-chamas", corpo); err == nil {
			t.Fatal("aprimoramento + truque devolveu — e não foi recusado")
		}
	})

	// O CONTROLE: sem ele, "foi recusado" também seria verdade num servidor que
	// recusa qualquer combinação. Os dois aprimoramentos comuns somam 1 + 1 sobre
	// o 1 de base.
	t.Run("dois aprimoramentos comuns continuam passando juntos", func(t *testing.T) {
		antes := mpOf(t, s, char)
		corpo := `{"augments":[{"augmentIndex":1,"stacks":1},{"augmentIndex":2,"stacks":1}]}`
		if err := castSpell(t, s, owner, char, "explosao-de-chamas", corpo); err != nil {
			t.Fatalf("dois aprimoramentos comuns devolveram %v", err)
		}
		if gasto := antes - mpOf(t, s, char); gasto != 3 {
			t.Errorf("gastou %d PM, want 3 (1 de base + 1 + 1)", gasto)
		}
	})
}
