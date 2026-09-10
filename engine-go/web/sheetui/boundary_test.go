package sheetui

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// Décima cena, e a maior de todas: 36 arquivos de produção, sete abas, mais de
// trinta mutações. A porta tem dezoito métodos — cinco deles ESCRITA —, e o
// guarda existe para que ela pare de crescer pela tabela em vez de pela
// pergunta.
//
// A tentação tem nome, e aqui ela não é hipótese: **o `s.db`**. Esta cena
// montava `setBuilder` + `"UPDATE character_items"` e `"UPDATE characters"` em
// QUATRO lugares, e um deles abria a transação inteira — `BeginTx`, plano,
// escritas, `Commit`. Cena que compõe SQL é cena com o banco dentro. As quatro
// viraram `SaveItemOverlays`, `SaveChoices` e `ApplyPowerTempHp`, e a última
// desfez de quebra uma transação escrita DUAS vezes: o `applyPool` da rota JSON
// tinha a mesma sequência. (A rota foi apagada na ALE-277 e o `applyPool` com
// ela — o que sobrou é a cópia única, aqui.)
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
	"t20engine/book":           true, // o catálogo TIPADO: poder, origem, deus, condição, ativação
	"t20engine/catalog":        true, // ver a nota abaixo — é o LookupSpell, não o Resource
	"t20engine/db/sqlcgen":     true, // as linhas do banco, pelo `Queries` da porta
	"t20engine/engine":         true, // os tipos computados que os painéis desenham
	"t20engine/plataforma":     true, // o carimbo ISO e os conversores de coluna anulável
	"t20engine/sheet":          true, // a ficha: a forma do dado E as regras dela
	"t20engine/web/characters": true, // a linha de classes do herói, uma só
	"t20engine/web/ui":         true, // o kit, a casca e a identidade visual
}

// O `catalog` está na lista, e a razão é MEDIDA e não confortável.
//
// O que saiu daqui nesta fatia foram as TRÊS leituras cruas — `Resource("class-powers")`
// duas vezes e `Resource("activations")` uma —, que são a forma que o `items.go`
// da forja e o improviso do trilho do mestre já tinham mostrado: quem lê o
// catálogo é do livro. Elas viraram `book.ClassPowerFlags`,
// `book.PowersThatTeachSpells` e `book.Activations`.
//
// O que FICOU são três chamadas de `catalog.LookupSpell` e uma de
// `catalog.IsCondition`, mais os tipos `catalog.Spell` e `catalog.Augment` em
// três assinaturas. Elas não são a mesma coisa: são o acessor tipado,
// e o HOSPEDEIRO usa exatamente o mesmo — o `validateAugments` recebe um
// `catalog.Spell`. Unificá-lo com o `book.Spell`, que existe e tem outros
// campos, é trabalho próprio e mexe nos dois lados; fingir que ele coube aqui
// seria pior. Fica contado: **quatro chamadas e três assinaturas.**
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
