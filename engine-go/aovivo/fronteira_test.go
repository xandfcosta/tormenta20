package aovivo

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A FRONTEIRA DO REGIME (ALE-254).
//
// O irmão deste guarda vive em `plataforma/` e a regra é a mesma: um bounded
// context vale pelo que o compilador IMPEDE. A lista aqui é maior porque o
// regime É domínio — ele pode falar com a persistência e com a plataforma — mas
// o que ele NÃO pode é conhecer os outros contextos.
//
// O caso concreto que este guarda protege já aconteceu durante a própria
// extração: o `sessionStore` chamava `applyCharacterVitals`, que usa as regras
// de dano da FICHA. A resposta certa foi declarar a porta `VitaisDaFicha` e
// receber quem a cumpre por parâmetro — e não importar a ficha daqui. Sem este
// guarda, o próximo a precisar de algo da ficha acrescenta o import, compila, e
// a porta vira enfeite.
//
// `api` não está na lista de propósito, e é o teste mais importante: `api` monta
// as rotas e importa o regime, então o regime importá-lo de volta seria ciclo —
// o compilador já pega. O que ele NÃO pega é `engine` ou `catalog` entrando
// aqui, e é isso que a lista fecha.

// permitidos é o que o regime pode importar de dentro do projeto.
var permitidos = map[string]bool{
	// A persistência é infraestrutura: o estado da fila é gravado no banco, e
	// isso é do regime — o `persist` mora aqui.
	"t20engine/db/sqlcgen": true,
	// A plataforma não é domínio nenhum, então depender dela não cria fronteira
	// errada nenhuma. É a direção que o guarda de lá garante ser de mão única.
	"t20engine/plataforma": true,
	// O VOCABULÁRIO DA MESA é shared kernel, e não um contexto (ALE-279).
	//
	// Ele entra nesta lista sabendo do aviso que está escrito acima — que
	// acrescentar import aqui transforma a porta em enfeite —, e a diferença
	// está no que ele PODE conter. Uma porta não resolve este caso: um
	// barramento TIPADO exige que os dois lados falem do mesmo tipo, e
	// interface declarada aqui só casaria com um tipo daqui, o que devolveria
	// um vocabulário por contexto e o problema inteiro.
	//
	// O que impede a porta dos fundos ("importe `events` para chegar na ficha")
	// não é confiança: é o `TestVocabularyImportsNothing`, que recusa
	// QUALQUER import do projeto dentro de `events/`. Enquanto ele for folha,
	// depender dele não cria fronteira errada nenhuma — que é exatamente a
	// justificativa da `plataforma`, por outro caminho.
	"t20engine/events": true,
}

func TestORegimeNaoConheceOsOutrosContextos(t *testing.T) {
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
			t.Errorf("%s importa %q — o regime não conhece outros contextos.\n"+
				"Se o regime precisa de algo de lá, DECLARE UMA PORTA aqui e receba\n"+
				"quem a cumpre por parâmetro, como `VitaisDaFicha` faz com a ficha.\n"+
				"Acrescentar o import à lista transforma a porta em enfeite.",
				nome, caminho)
		}
	}

	// Sem isto, apagar o pacote deixaria o guarda VERDE — ausência lida como
	// aprovação, que é a família que o `CLAUDE.md` da raiz descreve.
	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
