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
	"t20engine/web/ui":     true, // o kit de apresentação e a casca
	"t20engine/db/sqlcgen": true, // as linhas do banco
	"t20engine/plataforma": true, // não é domínio nenhum: aqui, o carimbo ISO
}

func TestTheAdminDoesNotImportItsHost(t *testing.T) {
	arquivos, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ler o pacote: %v", err)
	}

	conjunto := token.NewFileSet()
	visitados := 0
	for _, entrada := range arquivos {
		nome := entrada.Name()
		if !strings.HasSuffix(nome, ".go") {
			continue
		}
		visitados++
		arquivo, err := parser.ParseFile(conjunto, nome, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ler %s: %v", nome, err)
		}
		for _, imp := range arquivo.Imports {
			caminho := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(caminho, "t20engine/") || permitidos[caminho] {
				continue
			}
			t.Errorf("%s importa %q.\n"+
				"Se a cena precisa de algo de lá, DECLARE na `Deps` e receba de quem monta.\n"+
				"Acrescentar o import à lista transforma a porta em enfeite — e se %q for\n"+
				"o `api`, é ciclo, porque ele importa esta cena para montar rota.",
				nome, caminho, caminho)
		}
	}

	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
