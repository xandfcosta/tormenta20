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
	porRota := map[string][]string{}
	err := chi.Walk(mux, func(metodo, rota string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		porRota[metodo+" "+rota] = nil
		return nil
	})
	if err != nil {
		t.Fatalf("percorrer o roteador: %v", err)
	}
	return porRota
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
	bruto, err := os.ReadFile(routeParamsFile)
	if err != nil {
		t.Fatalf("ler %s: %v", routeParamsFile, err)
	}
	permitidos := map[string]bool{}
	for _, linha := range strings.Split(string(bruto), "\n") {
		if linha = strings.TrimSpace(linha); linha != "" && !strings.HasPrefix(linha, "#") {
			permitidos[linha] = true
		}
	}

	rotas := walkTheRouter(t, newTestServer(t))
	vistos := map[string]bool{}
	desconhecidos := map[string]string{}
	for rota := range rotas {
		for _, m := range routeParam.FindAllStringSubmatch(rota, -1) {
			nome := m[1]
			// O `{*}` e o `{param:regex}` do chi não são nome de parâmetro.
			if nome == "*" {
				continue
			}
			if corte := strings.IndexByte(nome, ':'); corte >= 0 {
				nome = nome[:corte]
			}
			vistos[nome] = true
			if !permitidos[nome] {
				desconhecidos[nome] = rota
			}
		}
	}

	// O DENOMINADOR, e ele é EXATO porque vem do roteador. O piso denuncia um
	// roteador que deixou de montar — que é como este guarda ficaria inerte.
	if len(rotas) < 150 || len(vistos) < 30 {
		t.Fatalf("a varredura achou %d rotas e %d parâmetros — o roteador é o primeiro suspeito",
			len(rotas), len(vistos))
	}

	var lista []string
	for nome, rota := range desconhecidos {
		lista = append(lista, nome+" — em "+rota)
	}
	sort.Strings(lista)
	if len(lista) > 0 {
		t.Errorf("parâmetro de caminho que o PERMITIDOS não conhece — %d de %d, em %d rotas:\n  %s\n"+
			"A coordenada de um gesto viaja no CORPO, pelo `payload` do `@post` (ver "+
			"`engine-go/CLAUDE.md`). Se este parâmetro NÃO vem do ponteiro, escreva a linha em "+
			routeParamsFile+" — o guarda falha no que não conhece de propósito, porque a lista de "+
			"PROIBIDOS que ele substituiu deixou nove grafias de coordenada passarem verdes.",
			len(lista), len(vistos), len(rotas), strings.Join(lista, "\n  "))
		return
	}
	t.Logf("rotas: %d, parâmetros: %d, nenhum desconhecido", len(rotas), len(vistos))
}

// A lista SÓ PODE ENCOLHER sozinha.
//
// Uma entrada que deixou de existir no roteador é uma permissão pendurada, e
// pendurada é exatamente onde ela não incomoda ninguém: no dia em que alguém
// registrar `/tabuleiro/{linha}/{col}` achando que `linha` é a da lista de
// tarefas, a lista velha o aprova em silêncio. É a mesma regra da linha de base
// de idioma: arquivo de permissões que não encolhe vira mentira sozinho.
func TestNoAllowedRouteParamIsStale(t *testing.T) {
	bruto, err := os.ReadFile(routeParamsFile)
	if err != nil {
		t.Fatalf("ler %s: %v", routeParamsFile, err)
	}
	rotas := walkTheRouter(t, newTestServer(t))
	vivos := map[string]bool{}
	for rota := range rotas {
		for _, m := range routeParam.FindAllStringSubmatch(rota, -1) {
			nome := m[1]
			if corte := strings.IndexByte(nome, ':'); corte >= 0 {
				nome = nome[:corte]
			}
			vivos[nome] = true
		}
	}

	var pendurados []string
	for _, linha := range strings.Split(string(bruto), "\n") {
		linha = strings.TrimSpace(linha)
		if linha == "" || strings.HasPrefix(linha, "#") {
			continue
		}
		if !vivos[linha] {
			pendurados = append(pendurados, linha)
		}
	}
	sort.Strings(pendurados)
	if len(pendurados) > 0 {
		t.Errorf("parâmetro permitido que NENHUMA rota usa — %d:\n  %s\n"+
			"Tire a linha. Uma permissão pendurada aprova em silêncio o dia em que alguém "+
			"reusar aquele nome para outra coisa.", len(pendurados), strings.Join(pendurados, "\n  "))
	}
}
