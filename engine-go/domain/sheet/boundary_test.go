package sheet

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A FICHA E AS REGRAS DELA, e nenhum contexto.
//
// Este pacote não é só FORMA DE DADO: as regras da ficha moram aqui, e duas
// delas leem o LIVRO. Passá-lo por parâmetro faria cada chamador montar a tabela
// do catálogo para entregá-la de volta.
//
// O que a lista IMPEDE continua sendo o que importa: `api` (que é HTTP),
// `catalog` (que é o arquivo cru) e `web/*` (que é tela). O livro entra TIPADO,
// pelo `book`, que é folha.
var permitidos = map[string]bool{
	// A forma vem das linhas do banco: o `CharacterDTO` nasce de um
	// `sqlcgen.Character`, e é isso que o `CharacterScalarsFrom` faz.
	"t20engine/infra/db/sqlcgen": true,
	// Os tipos do MOTOR aparecem nos campos computados. É a direção certa: a
	// ficha conhece a regra, a regra não conhece a ficha.
	"t20engine/domain/engine": true,
	// Não é domínio nenhum, então depender dela não cria fronteira errada — a
	// mesma justificativa dos irmãos.
	"t20engine/infra/db/dbvalue": true,
	// O LIVRO é a entrada que mudou o que este pacote É (decisão do dono): duas
	// regras — quantas vagas de poder o nível abre e qual círculo o personagem
	// alcança — leem o CATÁLOGO e a FICHA ao mesmo tempo, e o `book` não pode
	// importar daqui.
	//
	// O que isto CUSTA: toda cena que importa a ficha alcança o livro de graça. O
	// preço é pequeno porque quase todas já importam `book` direto — mas ele é
	// real, e a próxima entrada nesta lista merece a mesma conta.
	"t20engine/domain/book": true,
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
			t.Errorf("%s importa %q — a ficha é o DADO e as REGRAS dele, e nada mais.\n"+
				"HTTP, catálogo cru e tela ficam de fora: se ela precisa de algo de lá,\n"+
				"o dado entra por PARÂMETRO de quem monta.\n"+
				"Acrescentar o import à lista dá a %q a todas as cenas que importam este pacote.",
				nome, caminho, caminho)
		}
	}

	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
