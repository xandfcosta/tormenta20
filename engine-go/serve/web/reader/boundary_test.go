package reader

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// A tentação daqui é o `os` e o `config.Config`, e ela é concreta: a outra
// metade do arquivo original faz `os.Stat`, lê `LIVRO_PDF` e serve o arquivo com
// faixas. Essa metade ficou no `api` de propósito — uma cena que descobrisse
// onde o PDF está no disco teria o hospedeiro dentro dela, e o que ela precisa
// saber é só o ENDEREÇO.
//
// O guarda não recusa a biblioteca padrão, então um `os.ReadFile` aqui passaria
// por ele. O que o segura é a porta: sem caminho de arquivo atravessando a
// fronteira, não há o que ler.
var permitidos = map[string]bool{
	"t20engine/serve/web/bookui": true, // o endereço do livro e a abertura dele
	"t20engine/serve/web/ui":     true, // o kit e a casca
	"t20engine/serve/web/routes": true, // o endereço desta cena, que o `bookui` cita
}

func TestTheReaderDoesNotImportItsHost(t *testing.T) {
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
				"Se for `t20engine/infra/config`, a resposta é outra: quem lê configuração e\n"+
				"serve o ARQUIVO é o hospedeiro; esta cena só desenha a página.",
				name, path, path)
		}
	}

	if visited < 3 {
		t.Fatalf("o guarda visitou só %d arquivos `.go` — ele está medindo o "+
			"diretório errado", visited)
	}
}
