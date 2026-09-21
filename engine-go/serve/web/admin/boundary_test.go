package admin

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// A administração é a cena com a MAIOR porta das quatro — treze métodos —, e o
// guarda é o que impede a lista de crescer por conveniência. A tentação aqui é
// concreta e tem nome: `s.cfg`. Metade do painel de servidor vem da
// configuração, e pedir a `Config` inteira resolveria três métodos de uma vez —
// ao preço de a cena conhecer trinta campos para mostrar dois, e de o tipo ser
// do hospedeiro.
//
// O que ela alcança hoje é o kit, as linhas do banco e nada mais. `time` e
// `context` são biblioteca padrão e não passam por aqui.
var permitidos = map[string]bool{
	// O `app/accounts` NÃO é concessão, é a porta encolhendo (ALE-349): ele está
	// ABAIXO desta cena, então não há ciclo para desviar e não há interface a
	// declarar. Apagar conta, cunhar convite e cunhar link de senha chegam por
	// PARÂMETRO do `New` — e o predicado que distinguia "conta inexistente"
	// deixou de existir, porque `accounts.ErrUnknownAccount` é valor exportado.
	"t20engine/app/accounts":     true,
	"t20engine/serve/web/ui":     true, // o kit de apresentação e a casca
	"t20engine/infra/db/sqlcgen": true, // as linhas do banco
	"t20engine/infra/db/dbvalue": true,
}

func TestTheAdminDoesNotImportItsHost(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ler o pacote: %v", err)
	}

	set := token.NewFileSet()
	visited := 0
	for _, entry := range files {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		visited++
		file, err := parser.ParseFile(set, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ler %s: %v", name, err)
		}
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(path, "t20engine/") || permitidos[path] {
				continue
			}
			t.Errorf("%s importa %q.\n"+
				"Se a cena precisa de algo de lá, DECLARE na `Deps` e receba de quem monta.\n"+
				"Acrescentar o import à lista transforma a porta em enfeite — e se %q for\n"+
				"o `api`, é ciclo, porque ele importa esta cena para montar rota.",
				name, path, path)
		}
	}

	if visited == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
