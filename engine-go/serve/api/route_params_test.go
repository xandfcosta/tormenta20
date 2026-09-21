package api

import (
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

const routeParamsFile = "testdata/route_params.txt"

var routeParam = regexp.MustCompile(`\{([^}]*)\}`)

// walkTheRouter visita TODA rota registrada no processo.
//
// UM roteador e não dois: a API JSON é montada em `/api` dentro do `WebRouter`,
// e percorrer os dois contaria cada rota dela duas vezes, com e sem o prefixo.
//
// `chi.Walk` e não um regex sobre o código-fonte, e essa é a decisão inteira
// desta fatia: o padrão que ele devolve já está RESOLVIDO — o `base :=` juntado,
// o `r.Route` pai concatenado com o filho, o registro quebrado em duas linhas
// lido como uma. Nenhuma dessas três formas era visível para o guarda anterior,
// e as três são forma que este repositório usa.
//
// É também o que se ganha em ter um roteador só: o que o processo atende passou
// a caber inteiro nesta varredura. Enquanto as fontes, o favicon e a saúde
// moravam num roteador da biblioteca padrão, no `cmd`, nenhum guarda conseguia
// perguntar por elas.
func walkTheRouter(t *testing.T, s *Server) map[string][]string {
	t.Helper()
	mux, ok := s.WebRouter().(*chi.Mux)
	if !ok {
		t.Fatalf("o roteador deixou de ser um *chi.Mux (%T) — sem ele não há a quem perguntar", s.WebRouter())
	}
	byRoute := map[string][]string{}
	err := chi.Walk(mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		byRoute[method+" "+route] = nil
		return nil
	})
	if err != nil {
		t.Fatalf("percorrer o roteador: %v", err)
	}
	return byRoute
}

// NENHUMA ROTA CARREGA COORDENADA NO CAMINHO.
//
// A coordenada de um gesto viaja no CORPO. O que este guarda impede não é a
// decisão — é o ESQUECIMENTO dela.
//
// As duas metades do desenho são o que o torna confiável, e cada uma responde a
// um jeito de ele mentir:
//
// **Perguntar ao ROTEADOR**, e não a um regex sobre a fonte. Um parser de LINHA
// não vê a coordenada declarada num `base :=` (a forma dominante em `web/table`),
// nem o registro quebrado em duas linhas, nem o `r.Route`/`r.Handle` — e
// sabotando um `r.Route` o guarda PASSA com o denominador SUBINDO, porque ele
// conta o `r.Post` filho e ignora o pai.
//
// **Inverter a lista**: parâmetro que o PERMITIDOS não conhece REPROVA. Uma
// lista de proibidos subconta em silêncio — nove grafias alternativas de
// coordenada passaram verdes por ela, incluindo as que o cliente usa hoje.
// Escrever a linha é o ato de declarar que aquele parâmetro não vem do ponteiro.
func TestNoRouteCarriesACoordinateInThePath(t *testing.T) {
	raw, err := os.ReadFile(routeParamsFile)
	if err != nil {
		t.Fatalf("ler %s: %v", routeParamsFile, err)
	}
	allowed := map[string]bool{}
	for _, row := range strings.Split(string(raw), "\n") {
		if row = strings.TrimSpace(row); row != "" && !strings.HasPrefix(row, "#") {
			allowed[row] = true
		}
	}

	routes := walkTheRouter(t, newTestServer(t))
	seen := map[string]bool{}
	unknown := map[string]string{}
	for route := range routes {
		for _, m := range routeParam.FindAllStringSubmatch(route, -1) {
			name := m[1]
			// O `{*}` e o `{param:regex}` do chi não são nome de parâmetro.
			if name == "*" {
				continue
			}
			if cut := strings.IndexByte(name, ':'); cut >= 0 {
				name = name[:cut]
			}
			seen[name] = true
			if !allowed[name] {
				unknown[name] = route
			}
		}
	}

	// O DENOMINADOR, e ele é EXATO porque vem do roteador. O piso denuncia um
	// roteador que deixou de montar — que é como este guarda ficaria inerte.
	if len(routes) < 150 || len(seen) < 30 {
		t.Fatalf("a varredura achou %d rotas e %d parâmetros — o roteador é o primeiro suspeito",
			len(routes), len(seen))
	}

	var list []string
	for name, route := range unknown {
		list = append(list, name+" — em "+route)
	}
	sort.Strings(list)
	if len(list) > 0 {
		t.Errorf("parâmetro de caminho que o PERMITIDOS não conhece — %d de %d, em %d rotas:\n  %s\n"+
			"A coordenada de um gesto viaja no CORPO, pelo `payload` do `@post` (ver "+
			"`engine-go/CLAUDE.md`). Se este parâmetro NÃO vem do ponteiro, escreva a linha em "+
			routeParamsFile+" — o guarda falha no que não conhece de propósito, porque a lista de "+
			"PROIBIDOS que ele substituiu deixou nove grafias de coordenada passarem verdes.",
			len(list), len(seen), len(routes), strings.Join(list, "\n  "))
		return
	}
	t.Logf("rotas: %d, parâmetros: %d, nenhum desconhecido", len(routes), len(seen))
}

// A lista SÓ PODE ENCOLHER sozinha.
//
// Uma entrada que deixou de existir no roteador é uma permissão pendurada, e
// pendurada é exatamente onde ela não incomoda ninguém: no dia em que alguém
// registrar `/tabuleiro/{linha}/{col}` achando que `row` é a da lista de
// tarefas, a lista velha o aprova em silêncio. É a mesma regra da linha de base
// de idioma: arquivo de permissões que não encolhe vira mentira sozinho.
func TestNoAllowedRouteParamIsStale(t *testing.T) {
	raw, err := os.ReadFile(routeParamsFile)
	if err != nil {
		t.Fatalf("ler %s: %v", routeParamsFile, err)
	}
	routes := walkTheRouter(t, newTestServer(t))
	alive := map[string]bool{}
	for route := range routes {
		for _, m := range routeParam.FindAllStringSubmatch(route, -1) {
			name := m[1]
			if cut := strings.IndexByte(name, ':'); cut >= 0 {
				name = name[:cut]
			}
			alive[name] = true
		}
	}

	var dangling []string
	for _, row := range strings.Split(string(raw), "\n") {
		row = strings.TrimSpace(row)
		if row == "" || strings.HasPrefix(row, "#") {
			continue
		}
		if !alive[row] {
			dangling = append(dangling, row)
		}
	}
	sort.Strings(dangling)
	if len(dangling) > 0 {
		t.Errorf("parâmetro permitido que NENHUMA rota usa — %d:\n  %s\n"+
			"Tire a linha. Uma permissão pendurada aprova em silêncio o dia em que alguém "+
			"reusar aquele nome para outra coisa.", len(dangling), strings.Join(dangling, "\n  "))
	}
}
