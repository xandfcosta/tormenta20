package sheetui

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO.
//
// É a maior cena do app — 36 arquivos de produção, sete abas, mais de trinta
// mutações —, e o guarda existe para a porta parar de crescer pela TABELA em vez
// de pela PERGUNTA.
//
// A tentação tem nome: **o `s.db`**. Cena que compõe SQL é cena com o banco
// dentro, e o remédio é um método que nomeia a pergunta (`SaveItemOverlays`,
// `SaveChoices`).
//
// O `Queries` continua permitido — as sete abas leem e escrevem a mesma linha de
// personagem, e é a concessão da forja, da administração e das campanhas. O
// sinal de que ela está no lugar é nenhum handler daqui tocar banco fora dele.
//
// **O `web/characters` na lista é cena lendo cena**, e é a mesma concessão que a
// Mesa faz com o bestiário do `web/master`: a ficha reaproveita quatro campos do
// cartão do herói. A direção continua legal — quem importa é quem desenha
// depois.
var permitidos = map[string]bool{
	// O `app/character` NÃO é concessão: ele está ABAIXO desta cena, então não há
	// ciclo para desviar — e por isso não há interface. As entradas da porta que
	// viraram gesto dele saíram da `Deps` em vez de ganhar um adaptador. A
	// direção continua legal: quem importa é quem desenha depois.
	"t20engine/app/character":        true,
	"t20engine/domain/book":          true, // o catálogo TIPADO: poder, origem, deus, condição, ativação
	"t20engine/domain/catalog":       true, // ver a nota abaixo — é o LookupSpell, não o Resource
	"t20engine/infra/db/sqlcgen":     true, // as linhas do banco, pelo `Queries` da porta
	"t20engine/domain/engine":        true, // os tipos computados que os painéis desenham
	"t20engine/infra/db/dbvalue":     true,
	"t20engine/domain/sheet":         true, // a ficha: a forma do dado E as regras dela
	"t20engine/serve/web/characters": true, // a linha de classes do herói, uma só
	"t20engine/serve/web/ui":         true, // o kit, a casca e a identidade visual
}

// O `catalog` está na lista, e a linha é entre LEITURA CRUA e ACESSOR TIPADO.
//
// `Resource(…)` não entra: quem lê o catálogo cru é o `domain/book`, e as
// leituras que existiam aqui viraram `book.ClassPowerFlags`,
// `book.PowersThatTeachSpells` e `book.Activations`.
//
// O que fica são `catalog.LookupSpell`, `catalog.IsCondition` e os tipos
// `catalog.Spell` e `catalog.Augment` em três assinaturas — o mesmo acessor que
// o CASO DE USO usa (o `validateAugments` recebe um `catalog.Spell`, e é por
// isso que ele não desce para o `domain/sheet`). Unificá-lo com o `book.Spell`
// é trabalho próprio, que mexe nos dois lados.
func TestTheSheetSceneDoesNotImportItsHost(t *testing.T) {
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
				"o `api`, é ciclo, porque ele importa esta cena para montar a Mesa.\n"+
				"Se a vontade for o banco cru para uma coluna nova, a resposta é outra: a\n"+
				"porta cresce com a PERGUNTA (ver `SaveChoices`), não com a tabela.",
				nome, caminho, caminho)
		}
	}

	// O DENOMINADOR: um diretório não lido e uma lista de reprovados vazia se
	// parecem no terminal. Esta cena tem 36 arquivos `.go` de produção e mais os
	// de teste; o piso é folgado de propósito, para ele acusar o guarda medindo o
	// diretório errado e não uma fatia que junta dois arquivos.
	if visitados < 30 {
		t.Fatalf("o guarda visitou só %d arquivos `.go` — ele está medindo o "+
			"diretório errado", visitados)
	}
}
