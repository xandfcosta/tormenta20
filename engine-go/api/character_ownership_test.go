package api

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

// A POSSE da ficha, rota por rota (ALE-186, bloco 2).
//
// `assertCharacterOwner`/`characterFor` guarda cada mutação, e a tabela antiga
// (`TestCharacterWritesRejectNonOwner`) disparava 4 das 26 rotas por requisição:
// aprender magia, preparar, aplicar efeito, habilidades, perícias e itens
// nunca tinham sido chamadas por um intruso.
//
// Por isso isto não é uma tabela e sim uma INVARIANTE, na forma do `gmGate`: as
// rotas são LIDAS do registro do router, e uma rota nova entra vermelha até
// alguém escrever a linha dela. O risco real nunca foi a linha existente estar
// errada — é a rota do mês que vem esquecer a porta.

// `Put` entrou na ALE-222, e a ausência dele era um PONTO CEGO: a rota
// `PUT /{id}/stances/{flag}` foi registrada e esta invariante não a viu, porque
// o verbo não estava na alternância — das quatro rotas novas ela cobrou três.
// Uma invariante que lê o router vale exatamente o que vale a lista de verbos
// que ela reconhece.
var characterRoute = regexp.MustCompile(`r\.(Get|Post|Put|Patch|Delete)\("(/\{id\}[^"]*)", s\.(\w+)\)`)

// registeredCharacterRoutes lê o `r.Route("/characters", …)` do server.go.
func registeredCharacterRoutes(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("ler server.go: %v", err)
	}
	block := charactersBlock(t, string(raw))
	found := map[string]bool{}
	for _, m := range characterRoute.FindAllStringSubmatch(block, -1) {
		found[strings.ToUpper(m[1])+" "+m[2]] = true
	}
	if len(found) == 0 {
		t.Fatal("nenhuma rota de /characters/{id} reconhecida — o registro mudou de forma e este teste ficou cego")
	}
	return found
}

// charactersBlock recorta SÓ o grupo `/characters`: outros grupos (campanhas,
// sessões) também registram `/{id}`, e sem o recorte a invariante cobraria a
// posse da ficha em rotas que não são de ficha.
func charactersBlock(t *testing.T, source string) string {
	t.Helper()
	start := strings.Index(source, `r.Route("/characters"`)
	if start < 0 {
		t.Fatal("não achei o grupo /characters no server.go")
	}
	end := strings.Index(source[start:], "\n\treturn r")
	if end < 0 {
		t.Fatal("o grupo /characters não termina onde este teste espera")
	}
	return source[start : start+end]
}

// intruderCase é uma rota da ficha com um corpo que passaria pela validação —
// o 403 tem de vir da POSSE, e não de um campo faltando.
type intruderCase struct {
	method, template, path, body string
}

// The templates below mirror the router 1:1; `path` is the same route with the
// seeded ids substituted.
func characterIntruderCases(hero, item, effect int64) []intruderCase {
	base := "/characters/" + id64(hero)
	return []intruderCase{
		{http.MethodGet, "/{id}", base, ""},
		{http.MethodGet, "/{id}/sheet", base + "/sheet", ""},
		{http.MethodGet, "/{id}/campaigns", base + "/campaigns", ""},
		{http.MethodPost, "/{id}/active-effects", base + "/active-effects", `{"catalogId":"cosmetico","scope":"scene"}`},
		{http.MethodPatch, "/{id}/active-effects/{effectId}", base + "/active-effects/" + id64(effect), `{"scope":"day"}`},
		{http.MethodDelete, "/{id}/active-effects/{effectId}", base + "/active-effects/" + id64(effect), ""},
		{http.MethodPost, "/{id}/end-scene", base + "/end-scene", ""},
		{http.MethodPost, "/{id}/end-day", base + "/end-day", ""},
		{http.MethodPatch, "/{id}/vitals", base + "/vitals", `{"hpCurrent":1}`},
		{http.MethodPatch, "/{id}/tibar", base + "/tibar", `{"tibar":10}`},
		{http.MethodPost, "/{id}/damage", base + "/damage", `{"amount":5}`},
		{http.MethodPatch, "/{id}/level", base + "/level", `{"level":20}`},
		{http.MethodPatch, "/{id}/classes/level", base + "/classes/level", `{"className":"Guerreiro","level":2}`},
		{http.MethodPatch, "/{id}/abilities", base + "/abilities", `{"strength":18}`},
		{http.MethodPatch, "/{id}/proficiencies", base + "/proficiencies", `{"proficiencies":["Marciais"]}`},
		{http.MethodPost, "/{id}/items", base + "/items", `{"name":"Corda","quantity":1,"slots":1}`},
		{http.MethodPatch, "/{id}/items/{itemId}", base + "/items/" + id64(item), `{"quantity":2}`},
		{http.MethodDelete, "/{id}/items/{itemId}", base + "/items/" + id64(item), ""},
		{http.MethodPost, "/{id}/items/{itemId}/consume", base + "/items/" + id64(item) + "/consume", "{}"},
		{http.MethodPatch, "/{id}/conditions", base + "/conditions", `{"activeConditions":["caido"]}`},
		{http.MethodPost, "/{id}/expertises", base + "/expertises", `{"name":"Luta","trained":true}`},
		{http.MethodPatch, "/{id}/expertises", base + "/expertises", `{"name":"Luta","trained":false}`},
		{http.MethodDelete, "/{id}/expertises/{name}", base + "/expertises/Luta", ""},
		{http.MethodPost, "/{id}/spells", base + "/spells", `{"catalogSpellId":"bola-de-fogo"}`},
		{http.MethodDelete, "/{id}/spells/{catalogSpellId}", base + "/spells/bola-de-fogo", ""},
		{http.MethodPatch, "/{id}/spells/{catalogSpellId}/prepared", base + "/spells/bola-de-fogo/prepared", `{"prepared":true}`},
		{http.MethodPost, "/{id}/spells/{catalogSpellId}/cast", base + "/spells/bola-de-fogo/cast", "{}"},
		// O estado de JOGO da ficha (ALE-222).
		{http.MethodPatch, "/{id}/conditionals", base + "/conditionals", `{"conditionals":["furia"]}`},
		{http.MethodPost, "/{id}/power-uses", base + "/power-uses", `{"powerId":"furia","scope":"day"}`},
		{http.MethodPut, "/{id}/stances/{flag}", base + "/stances/furia", `{"steps":1,"pmPaid":2}`},
		{http.MethodDelete, "/{id}/stances/{flag}", base + "/stances/furia", ""},
	}
}

func seedEffectRow(t *testing.T, s *Server, charID int64) int64 {
	t.Helper()
	eff, err := s.queries.CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: charID, Catalogid: "cosmetico", Scope: "scene", Modifiers: "[]", Createdat: nowISO(),
	})
	if err != nil {
		t.Fatalf("semear efeito: %v", err)
	}
	return eff.ID
}

func TestEveryCharacterRouteIsCoveredByTheOwnershipTable(t *testing.T) {
	registered := registeredCharacterRoutes(t)
	covered := map[string]bool{}
	for _, c := range characterIntruderCases(1, 2, 3) {
		covered[c.method+" "+c.template] = true
	}

	for route := range registered {
		if !covered[route] {
			t.Errorf("a rota %q é nova e ninguém provou que ela recusa um intruso: escreva a linha dela", route)
		}
	}
	for route := range covered {
		if !registered[route] {
			t.Errorf("a tabela guarda %q, mas o router não registra mais essa rota", route)
		}
	}
}

func TestEveryCharacterRouteRejectsAnIntruder(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	intruder := seedUser(t, s, "intruso@t20.local")
	hero := seedCharacter(t, s, owner, "Herói", 8, 20, 3, 5)
	item := seedConsumable(t, s, hero, "balsamo-restaurador", "Bálsamo restaurador", 2)
	effect := seedEffectRow(t, s, hero)

	for _, c := range characterIntruderCases(hero, item, effect) {
		t.Run(c.method+" "+c.template, func(t *testing.T) {
			rec := authed(t, s, intruder, c.method, c.path, c.body)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("esperado 403, veio %d (%s)", rec.Code, rec.Body.String())
			}
			assertHeroUntouched(t, s, hero, item)
		})
	}
}

// assertHeroUntouched: o 403 tem de ser recusa, e não recusa DEPOIS de gravar.
func assertHeroUntouched(t *testing.T, s *Server, hero, item int64) {
	t.Helper()
	row, err := s.queries.GetCharacter(context.Background(), hero)
	if err != nil {
		t.Fatalf("ler personagem: %v", err)
	}
	if row.Hpcurrent != 8 || row.Level != 1 {
		t.Fatalf("o personagem mudou apesar do 403: pv=%d nível=%d", row.Hpcurrent, row.Level)
	}
	if qty, err := s.queries.GetItem(context.Background(), item); err == sql.ErrNoRows {
		t.Fatal("o item sumiu apesar do 403")
	} else if err == nil && qty.Quantity != 2 {
		t.Fatalf("a quantidade do item mudou apesar do 403: %d", qty.Quantity)
	}
}

// As regras de EQUIPAR pelo router (ALE-186, bloco 2).
//
// O `equip_test.go` prova os limites como HELPERS — a função certa recebendo a
// lista certa. O que faltava é a ligação: nada provava que o handler CHAMA a
// função, e um handler que esquecesse a chamada deixaria a mesa com três
// escudos na mão com a suíte inteira verde. É a mesma forma da ALE-124 fatia 4,
// onde a regra do terreno difícil estava implementada, testada e morta por
// falta de quem a chamasse.

func equipItem(t *testing.T, s *Server, owner, hero, item int64, state string) int {
	t.Helper()
	path := "/characters/" + id64(hero) + "/items/" + id64(item)
	return authed(t, s, owner, http.MethodPatch, path, `{"equipped":"`+state+`"}`).Code
}

func TestEquipLimitsAreEnforcedByTheHandler(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	hero := seedCharacter(t, s, owner, "Herói", 20, 20, 0, 0)
	first := seedConsumable(t, s, hero, "escudo-leve", "Escudo leve", 1)
	second := seedConsumable(t, s, hero, "escudo-leve", "Escudo leve", 1)
	third := seedConsumable(t, s, hero, "escudo-leve", "Escudo leve", 1)

	if code := equipItem(t, s, owner, hero, first, "wielded"); code != http.StatusOK {
		t.Fatalf("o primeiro escudo devia entrar na mão: %d", code)
	}
	if code := equipItem(t, s, owner, hero, second, "wielded"); code != http.StatusOK {
		t.Fatalf("o segundo escudo devia entrar na outra mão: %d", code)
	}

	code := equipItem(t, s, owner, hero, third, "wielded")

	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, esperado 400: são duas mãos, e a terceira não existe", code)
	}
	// E a recusa tem de ser recusa: 400 que grava é pior que não validar,
	// porque a tela mostra o erro e o banco fica com o terceiro escudo.
	row, err := s.queries.GetItem(context.Background(), third)
	if err != nil {
		t.Fatalf("ler o terceiro escudo: %v", err)
	}
	if row.Equipped.Valid {
		t.Fatalf("o terceiro escudo ficou equipado como %q apesar do 400", row.Equipped.String)
	}
}

// withEquipCatalog dá ao servidor o catálogo MÍNIMO de que o eixo precisa. O
// `newTestServer` sobe com catálogo nil de propósito (a maioria dos handlers não
// o usa), e sem ele o `equipAxisError` desiste em silêncio: "item desconhecido"
// vale como válido. Isso não é do teste — é como a produção se comporta quando
// o arquivo de catálogo não é encontrado, e o `main.go` até loga
// "mutation validators disabled" ao cair nesse caso.
func withEquipCatalog(t *testing.T, s *Server) {
	t.Helper()
	catalogs, err := engine.PrimeEngineCatalogs([]byte(`{"items":[
		{"id":"cota-malha","name":"Cota de malha","equip":"vested","slots":5},
		{"id":"escudo-leve","name":"Escudo leve","equip":"wielded","slots":1}
	]}`))
	if err != nil {
		t.Fatalf("preparar catálogo: %v", err)
	}
	s.catalogs = catalogs
}

func TestEquipAxisIsEnforcedByTheHandler(t *testing.T) {
	s := newTestServer(t)
	withEquipCatalog(t, s)
	owner := seedUser(t, s, "dono@t20.local")
	hero := seedCharacter(t, s, owner, "Herói", 20, 20, 0, 0)
	armor := seedConsumable(t, s, hero, "cota-malha", "Cota de malha", 1)

	// Uma armadura se VESTE; empunhá-la não é um estado que ela ocupe.
	code := equipItem(t, s, owner, hero, armor, "wielded")

	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, esperado 400 ao empunhar uma cota de malha", code)
	}
}
