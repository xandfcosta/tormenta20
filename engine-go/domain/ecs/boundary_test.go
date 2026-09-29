package ecs

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// O NÚCLEO É FOLHA, e este guarda é o que faz isso valer.
//
// O `world.go` afirma por escrito que aqui não há "nenhum import deste projeto
// — só a stdlib", e essa frase era só uma promessa: nada a cobrava. Ela passou
// a importar na ALE-413, quando o `domain/board` precisou do núcleo e entrou na
// lista de permitidos do guarda de fronteira de lá — a mesma lista que avisa
// que *acrescentar import a ela transforma a porta em enfeite*.
//
// O precedente é o `infra/events`, e a justificativa é idêntica: enquanto o
// núcleo não importar NADA do projeto, depender dele não pode criar fronteira
// errada nenhuma, porque não há para onde a dependência vazar. No dia em que
// alguém importar a ficha aqui para "dar um componente pronto ao mundo", o
// tabuleiro passa a alcançar a ficha de graça — e o guarda de lá continua
// verde, porque ele só olha os imports DELE.
//
// # O que PODE entrar aqui
//
// Entidade, componente, sistema e o armazenamento deles. Nada que saiba o que é
// Tormenta: não há atributo, perícia nem peça neste pacote, e é essa ausência
// que permite exercitar o mecanismo sem arranjar uma ficha inteira.
//
// # A única exceção é o AUTO-IMPORT
//
// O `world_test.go` mora no pacote externo `ecs_test` e importa o `ecs` por
// construção — é assim que ele exercita a API como um chamador de fora a vê.
//
// Ele é a exceção, e apenas ele: os arquivos de teste continuam varridos, de
// propósito. Os componentes de mentira daqui são `label` e `weight` justamente
// porque *"se uma palavra do livro aparecer neste pacote, o desenho vazou"* — e
// um teste que importasse o `domain/engine` para arranjar um componente pronto
// seria essa palavra, com a suíte verde.
const meuCaminho = "t20engine/domain/ecs"

func TestTheCoreImportsNothingOfTheProject(t *testing.T) {
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
			if !strings.HasPrefix(path, "t20engine/") || path == meuCaminho {
				continue
			}
			t.Errorf("%s importa %q — o núcleo de ECS é FOLHA.\n"+
				"Ele está na lista de permitidos do `tabuleiro` justamente porque\n"+
				"não alcança nada; com um import daqui, aquele contexto passa a\n"+
				"alcançar %q de graça, e o guarda de lá não vê.", name, path, path)
		}
	}

	// Sem isto, apagar o pacote deixaria o guarda VERDE — ausência lida como
	// aprovação.
	if visited == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
