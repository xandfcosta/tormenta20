package sheet

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A FICHA E AS REGRAS DELA, e nenhum contexto (ALE-278).
//
// Irmão dos `fronteira_test.go` do `aovivo`, do `tabuleiro`, da `plataforma`, do
// `events` e do `creature`. A lista aqui não é vazia como a do `creature`: três
// entradas foram medidas antes da extração — o `character_dto.go` já importava
// exatamente aquelas e mais nada, o que foi o que provou que ele saía inteiro.
//
// **A quarta mudou o que este pacote é, e a prosa mudou junto.** Aqui morava "A
// FICHA É FORMA DE DADO", com a promessa de que quem precisasse de algo de fora
// receberia o dado por PARÂMETRO. Ela valeu enquanto o pacote guardava só os
// DTOs; deixou de valer quando as regras da ficha vieram morar aqui — equipar e
// PV temporário na fatia anterior, as escolhas e o círculo alcançável nesta —, e
// as duas últimas leem o LIVRO. Passá-lo por parâmetro seria fazer cada chamador
// montar a tabela do catálogo para entregá-la de volta.
//
// O que a lista IMPEDE continua sendo o que importa: `api` (que é HTTP),
// `catalog` (que é o arquivo cru) e `web/*` (que é tela). O livro entra TIPADO,
// pelo `book`, que é folha.
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
	// O LIVRO entrou na ALE-278, e é a única entrada desta lista que mudou o que
	// este pacote É. Decisão do dono.
	//
	// Duas regras não cabiam em lugar nenhum: quantas vagas de poder o nível
	// abre e qual círculo o personagem alcança leem o CATÁLOGO e a FICHA ao mesmo
	// tempo, e o `book` não pode importar daqui (ele é consultado por treze
	// famílias). Ficar no `api` era devolver regra ao pacote que a épica está
	// esvaziando; ficar na cena era a rota JSON ler regra de um pacote de
	// apresentação.
	//
	// O que isto CUSTA, escrito para quem vier depois: toda cena que importa a
	// ficha alcança o livro de graça. O preço é pequeno porque oito das dez já
	// importam `book` direto — mas ele é real, e a próxima entrada nesta lista
	// merece a mesma conta.
	"t20engine/book": true,
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
