package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// Os guardas do estado de JOGO da ficha (ALE-222).
//
// O que vale proteger aqui não é o CRUD — é o CICLO DE VIDA. Os três estados
// saíram do `localStorage` justamente porque cada um se limpa numa hora
// diferente, e é a hora errada que estraga uma mesa: um "1/dia" que zera no
// descanso de cena devolve ao jogador um poder que o livro não devolve.

type playFixture struct {
	s     *Server
	dono  int64
	heroi int64
}

func novoPlayState(t *testing.T) playFixture {
	t.Helper()
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t.com")
	heroi := seedCharacter(t, s, dono, "Bárbaro", 30, 30, 5, 10)
	return playFixture{s: s, dono: dono, heroi: heroi}
}

// gasta usa o caminho HTTP de verdade, e não a query: o que se quer provar é o
// que o jogador consegue, não o que o SQL aceita.
func (f playFixture) gasta(t *testing.T, powerID, scope string) {
	t.Helper()
	corpo := `{"powerId":"` + powerID + `","scope":"` + scope + `"}`
	if rec := authed(t, f.s, f.dono, http.MethodPost, "/characters/"+id64(f.heroi)+"/power-uses", corpo); rec.Code != http.StatusOK {
		t.Fatalf("gastar %s/%s: code=%d — %s", powerID, scope, rec.Code, rec.Body.String())
	}
}

func (f playFixture) usos(t *testing.T) map[string]int64 {
	t.Helper()
	linhas, err := f.s.queries.ListCharacterPowerUses(context.Background(), f.heroi)
	if err != nil {
		t.Fatalf("ler usos: %v", err)
	}
	out := map[string]int64{}
	for _, l := range linhas {
		out[l.Powerid+"/"+l.Scope] = l.Used
	}
	return out
}

// O uso SOMA. É a razão de o corpo mandar "gastei mais um" em vez do total:
// dois cliques rápidos com o total gravariam o mesmo número e perderiam um uso.
func TestUsoDePoderSoma(t *testing.T) {
	f := novoPlayState(t)

	f.gasta(t, "furia", "day")
	f.gasta(t, "furia", "day")
	f.gasta(t, "furia", "day")

	if got := f.usos(t)["furia/day"]; got != 3 {
		t.Fatalf("usos = %d, queria 3 — o upsert está sobrescrevendo em vez de somar", got)
	}
}

// As duas contas do MESMO poder são independentes: um "1/cena" gasto três vezes
// no dia soma 3 no dia e 1 na cena. É por isso que o escopo entra na chave.
func TestUsoDePoderSeparaCenaDeDia(t *testing.T) {
	f := novoPlayState(t)

	f.gasta(t, "furia", "scene")
	f.gasta(t, "furia", "day")
	f.gasta(t, "furia", "day")

	usos := f.usos(t)
	if usos["furia/scene"] != 1 || usos["furia/day"] != 2 {
		t.Fatalf("cena=%d dia=%d, queria 1 e 2 — as contas se misturaram", usos["furia/scene"], usos["furia/day"])
	}
}

// O CICLO DE VIDA, que é o guarda que justifica três tabelas em vez de uma.
//
// O descanso de CENA leva os "1/cena" e as posturas, e a segunda metade é a que
// importa: ele NÃO pode levar os "1/dia". Um teste que só afirmasse "limpou"
// passaria verde com um `DELETE` sem `WHERE scope`.
func TestDescansoDeCenaLimpaSoOQueEDaCena(t *testing.T) {
	f := novoPlayState(t)
	ctx := context.Background()
	f.gasta(t, "furia", "scene")
	f.gasta(t, "milagre", "day")
	if rec := authed(t, f.s, f.dono, http.MethodPut, "/characters/"+id64(f.heroi)+"/stances/furia", `{"steps":2,"pmPaid":4}`); rec.Code != http.StatusOK {
		t.Fatalf("entrar na postura: code=%d — %s", rec.Code, rec.Body.String())
	}

	// Pelo HELPER de domínio e não pela rota: o que este teste protege é o CICLO
	// DE VIDA do estado de jogo, e a rota carrega uma autorização que não é
	// assunto dele — desde a ALE-223 ela pede um MESTRE em sessão viva, e montar
	// uma mesa aqui só para descansar mediria a regra errada. O helper é também
	// por onde o descanso da SESSÃO passa, então é o caminho mais verdadeiro.
	if status, err := f.s.EndScene(ctx, AuthUser{ID: f.dono}, f.heroi); err != nil {
		t.Fatalf("descanso de cena: status=%d — %v", status, err)
	}

	usos := f.usos(t)
	if _, ainda := usos["furia/scene"]; ainda {
		t.Error("o uso de cena sobreviveu ao descanso de cena")
	}
	if usos["milagre/day"] != 1 {
		t.Error("o descanso de CENA levou um uso de DIA junto — o jogador ganhou um poder que o livro não devolve")
	}
	posturas, err := f.s.queries.ListCharacterStances(ctx, f.heroi)
	if err != nil {
		t.Fatalf("ler posturas: %v", err)
	}
	if len(posturas) != 0 {
		t.Errorf("a postura sobreviveu ao descanso de cena: %+v", posturas)
	}
}

// O descanso de DIA leva os dois escopos.
func TestDescansoDeDiaLimpaOsDoisEscopos(t *testing.T) {
	f := novoPlayState(t)
	f.gasta(t, "furia", "scene")
	f.gasta(t, "milagre", "day")

	// Helper de domínio, mesma razão do teste irmão acima (ALE-223).
	if status, err := f.s.endDay(context.Background(), AuthUser{ID: f.dono}, f.heroi); err != nil {
		t.Fatalf("descanso de dia: status=%d — %v", status, err)
	}

	if usos := f.usos(t); len(usos) != 0 {
		t.Errorf("o descanso de dia deixou usos para trás: %+v", usos)
	}
}

// O situacional é um CONJUNTO substituído inteiro, como o irmão das condições do
// livro. As duas metades: o que entrou entrou, e o que não veio no corpo saiu.
func TestSituacionaisSubstituemOConjunto(t *testing.T) {
	f := novoPlayState(t)
	url := "/characters/" + id64(f.heroi) + "/conditionals"

	if rec := authed(t, f.s, f.dono, http.MethodPatch, url, `{"conditionals":["furia","ataque-poderoso"]}`); rec.Code != http.StatusOK {
		t.Fatalf("ligar: code=%d — %s", rec.Code, rec.Body.String())
	}
	if rec := authed(t, f.s, f.dono, http.MethodPatch, url, `{"conditionals":["furia"]}`); rec.Code != http.StatusOK {
		t.Fatalf("desligar um: code=%d — %s", rec.Code, rec.Body.String())
	}

	ligados, err := f.s.queries.ListCharacterConditionals(context.Background(), f.heroi)
	if err != nil {
		t.Fatalf("ler situacionais: %v", err)
	}
	if len(ligados) != 1 || ligados[0] != "furia" {
		t.Fatalf("ligados = %v, queria só [furia] — o PATCH somou em vez de substituir", ligados)
	}
}

// Situacional HOMEBREW passa. Não há catálogo fechado aqui, e recusar um id
// desconhecido no meio do combate seria pior que ignorá-lo: o motor não casa
// modificador nenhum com ele e o efeito é nada.
func TestSituacionalHomebrewNaoERecusado(t *testing.T) {
	f := novoPlayState(t)
	rec := authed(t, f.s, f.dono, http.MethodPatch, "/characters/"+id64(f.heroi)+"/conditionals",
		`{"conditionals":["bencao-caseira-do-mestre"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d — %s", rec.Code, rec.Body.String())
	}
}

// A ficha carrega o estado de jogo junto. Sem isto ela abriria com a Fúria
// desligada e a ligaria um instante depois, piscando os números que ela muda.
func TestFichaTrazOEstadoDeJogoJunto(t *testing.T) {
	f := novoPlayState(t)
	f.gasta(t, "furia", "day")
	if rec := authed(t, f.s, f.dono, http.MethodPatch, "/characters/"+id64(f.heroi)+"/conditionals", `{"conditionals":["furia"]}`); rec.Code != http.StatusOK {
		t.Fatalf("ligar: code=%d", rec.Code)
	}

	corpo := authed(t, f.s, f.dono, http.MethodGet, "/characters/"+id64(f.heroi), "").Body.String()

	for _, pedaco := range []string{`"conditionals":["furia"]`, `"powerId":"furia"`} {
		if !strings.Contains(corpo, pedaco) {
			t.Errorf("a ficha não trouxe %s:\n%s", pedaco, corpo)
		}
	}
}
