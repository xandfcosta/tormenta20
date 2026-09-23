package api

import (
	"net/http"
	"sort"
	"testing"

	"github.com/go-chi/chi/v5"
)

// A SUPERFÍCIE DA API JSON É DECLARADA, ROTA POR ROTA, E DE QUAL LADO (ALE-343).
//
// A API tem dois grupos — ver o `json_api_routes.go` —, e uma separação que só
// existe no arranjo do código é um comentário com passos extras: o próximo autor
// registra a rota onde for mais perto e nada acusa. Este guarda é o que a torna
// real.
//
// # Ele PERCORRE, e é isso que lhe dá o denominador
//
// `chi.Walk` sobre o `Router()` devolve o que está REGISTRADO, com o padrão já
// resolvido — o `r.Route` pai concatenado com o filho, o registro quebrado em duas
// linhas lido como um. Um parser da fonte não vê nenhuma das duas formas, e as
// duas estão neste arquivo.
//
// E ele cobra as DUAS direções, que é o que impede as listas de apodrecer:
//
//   - rota registrada e não declarada FALHA com o endereço dela e com a pergunta
//     que o autor tem de responder — bancada ou pública?
//   - rota declarada que não existe mais FALHA também, senão a lista guardaria
//     endereços mortos e o guarda passaria a medir uma API imaginária.
//
// # O que ele NÃO é
//
// Ele não é o `TestEveryAddressAPostWritesExistsInTheRouter`, que vai na direção
// oposta: aquele parte do HTML servido e pergunta ao chi se o endereço existe.
// Este parte do ROTEADOR e pergunta à declaração. Um endereço pode existir sem
// ninguém escrevê-lo — a bancada inteira é assim, porque quem a chama é um spec
// de Playwright, e o HTML nunca a menciona.

// thePublicJSONSurface é o que existe para quem está FORA do repositório.
//
// UMA rota, e o número é o achado da ALE-343: o que sobrou da API JSON depois de a
// SPA sair é bancada de teste mais a sonda do compose. Se esta lista crescer, foi
// decisão de desenho — e é bom que ela custe uma linha aqui.
var thePublicJSONSurface = map[string]bool{
	"GET /health": true,
}

// theBenchJSONSurface é a bancada do e2e, e cada entrada diz qual spec a pede.
var theBenchJSONSurface = map[string]bool{
	"GET /campanhas/":                       true, // auth.setup.ts: varredura das descartáveis
	"POST /campanhas/":                      true, // board.spec.ts: a mesa da corrida
	"DELETE /campanhas/{id}":                true, // auth.setup.ts: a faxina
	"POST /campanhas/{campaignId}/sessoes/": true, // board.spec.ts: a sessão da mesa
	"GET /personagens/":                     true, // spec da sessão: varredura de condições
	"PATCH /personagens/{id}/conditions":    true, // spec da sessão: a faixa cheia
}

func TestEveryJSONRouteDeclaresWhichSideItIsOn(t *testing.T) {
	s := newTestServer(t)
	mux, ok := s.Router().(*chi.Mux)
	if !ok {
		t.Fatalf("a API JSON deixou de ser um *chi.Mux (%T) — sem isso não há a quem perguntar", s.Router())
	}

	registered := map[string]bool{}
	err := chi.Walk(mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered[method+" "+route] = true
		return nil
	})
	if err != nil {
		t.Fatalf("percorrer a API JSON: %v", err)
	}

	for address := range registered {
		public, bench := thePublicJSONSurface[address], theBenchJSONSurface[address]
		if public && bench {
			t.Errorf("%q está declarada nos DOIS grupos, e a pergunta é excludente", address)
		}
		if !public && !bench {
			t.Errorf("a rota %q não está declarada em nenhum dos dois grupos.\n"+
				"Responda a pergunta e ponha o endereço na lista do `json_api_surface_test.go`:\n"+
				"  · PÚBLICA, se existe para quem está FORA do repositório;\n"+
				"  · BANCADA, se existe para um spec — e diga qual, no comentário da linha.",
				address)
		}
	}
	for _, declared := range []map[string]bool{thePublicJSONSurface, theBenchJSONSurface} {
		for address := range declared {
			if !registered[address] {
				t.Errorf("%q está declarada e NÃO está registrada no roteador.\n"+
					"Endereço morto na lista faz este guarda medir uma API imaginária — apague a linha.\n"+
					"O que o roteador atende hoje: %v", address, jsonAddressesInOrder(registered))
			}
		}
	}

	// O DENOMINADOR: uma varredura que não casa com nada e uma API vazia são a
	// mesma linha verde. O piso é o tamanho declarado, porque aqui a lista É o
	// esperado — ao contrário dos guardas de amostragem, esta API é pequena o
	// bastante para ser enumerada inteira, e é justamente esse o achado.
	if want := len(thePublicJSONSurface) + len(theBenchJSONSurface); len(registered) != want {
		t.Errorf("o roteador atende %d rotas e as listas declaram %d — %v",
			len(registered), want, jsonAddressesInOrder(registered))
	}
}

// jsonAddressesInOrder devolve os endereços em ordem, para a mensagem de falha ser comparável
// entre duas execuções.
func jsonAddressesInOrder(addresses map[string]bool) []string {
	out := make([]string, 0, len(addresses))
	for address := range addresses {
		out = append(out, address)
	}
	sort.Strings(out)
	return out
}
