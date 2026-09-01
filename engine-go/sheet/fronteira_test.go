package sheet

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A FICHA É FORMA DE DADO, e não alcança contexto nenhum (ALE-278).
//
// Irmão dos `fronteira_test.go` do `aovivo`, do `tabuleiro`, da `plataforma`, do
// `events` e do `creature`. A lista aqui não é vazia como a do `creature`, e os
// três que ela tem foram medidos antes da extração — o `character_dto.go` já
// importava exatamente estes e mais nada, o que foi o que provou que ele saía
// inteiro.
//
// O que a lista IMPEDE é o que importa: `api` (que é HTTP), `catalog` (que é o
// livro), `web/*` (que é tela). Este pacote descreve a forma que a ficha tem no
// fio e no banco; no dia em que ele souber ler catálogo, toda cena que o importa
// passa a ler catálogo junto — e cada guarda de fronteira continua verde, porque
// cada um só olha os imports dele.
var permitidos = map[string]bool{
	// A forma vem das linhas do banco: o `CharacterDTO` nasce de um
	// `sqlcgen.Character`, e é isso que o `CharacterScalarsFrom` faz.
	"t20engine/db/sqlcgen": true,
	// Os tipos do MOTOR aparecem nos campos computados. É a direção certa: a
	// ficha conhece a regra, a regra não conhece a ficha.
	"t20engine/engine": true,
	// Não é domínio nenhum, então depender dela não cria fronteira errada — a
	// mesma justificativa dos irmãos.
	"t20engine/plataforma": true,
}

func TestTheSheetDoesNotReachTheContexts(t *testing.T) {
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
			t.Errorf("%s importa %q — a ficha é forma de DADO.\n"+
				"Se ela precisa de algo de lá, o dado entra por PARÂMETRO de quem monta.\n"+
				"Acrescentar o import à lista dá a %q a todas as cenas que importam este pacote.",
				nome, caminho, caminho)
		}
	}

	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
