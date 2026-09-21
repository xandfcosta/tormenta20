package characters

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// A lista de personagens é a sétima cena a sair, com três métodos na porta. A
// tentação daqui tem nome e endereço: **o BANCO**.
//
// O cartão do herói mostra a Defesa, que vem do motor sobre a ficha computada, e
// o caminho curto para qualquer campo novo é pedir o `Queries()` — que a forja
// pede, e legitimamente, porque ela GRAVA. Esta cena não grava nada: ela faz uma
// pergunta (`CharacterList`) e desenha a resposta. Pedir o banco aqui a faria
// montar consulta, e no dia em que a lista precisar de um filtro novo a decisão
// certa é a porta crescer com a PERGUNTA, não com a tabela.
//
// O `engine` está na lista por causa de UM tipo — o `*engine.Catalogs` que a
// porta devolve. Ele não é usado para calcular nada aqui: quem calcula é o
// `sheet.Compute`.
var permitidos = map[string]bool{
	"t20engine/domain/book":   true, // as habilidades de raça que o dossiê escreve
	"t20engine/domain/engine": true, // só o tipo `*Catalogs`, que a porta repassa
	"t20engine/domain/search": true, // o casamento da busca da cena
	"t20engine/domain/sheet":  true, // a forma da ficha e o `Compute` que dá a Defesa
	"t20engine/serve/web/ui":  true, // o kit de apresentação, a casca e a identidade visual
}

func TestTheCharacterListDoesNotImportItsHost(t *testing.T) {
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
				"o `api`, é ciclo, porque ele importa esta cena para montar rota.\n"+
				"Se for `t20engine/infra/db/sqlcgen`, a resposta é outra: esta cena não grava,\n"+
				"então o que ela precisa é de uma PERGUNTA nova na porta, não do banco.",
				name, path, path)
		}
	}

	// O DENOMINADOR: um diretório não lido e uma lista de reprovados vazia se
	// parecem no terminal.
	if visited < 3 {
		t.Fatalf("o guarda visitou só %d arquivos `.go` — ele está medindo o "+
			"diretório errado", visited)
	}
}
