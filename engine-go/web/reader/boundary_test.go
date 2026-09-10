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
// A tentação daqui é o `os` e o `plataforma.Config`, e ela é concreta: a outra
// metade do arquivo original faz `os.Stat`, lê `LIVRO_PDF` e serve o arquivo com
// faixas. Essa metade ficou no `api` de propósito — uma cena que descobrisse
// onde o PDF está no disco teria o hospedeiro dentro dela, e o que ela precisa
// saber é só o ENDEREÇO.
//
// O guarda não recusa a biblioteca padrão, então um `os.ReadFile` aqui passaria
// por ele. O que o segura é a porta: sem caminho de arquivo atravessando a
// fronteira, não há o que ler.
var permitidos = map[string]bool{
	"t20engine/web/bookui": true, // o endereço do livro e a abertura dele
	"t20engine/web/ui":     true, // o kit e a casca
	"t20engine/web/routes": true, // o endereço desta cena, que o `bookui` cita
}

func TestTheReaderDoesNotImportItsHost(t *testing.T) {
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
				"o `api`, é ciclo, porque ele importa esta cena para montar rota.\n"+
				"Se for `t20engine/plataforma`, a resposta é outra: quem lê configuração e\n"+
				"serve o ARQUIVO é o hospedeiro; esta cena só desenha a página.",
				nome, caminho, caminho)
		}
	}

	if visitados < 3 {
		t.Fatalf("o guarda visitou só %d arquivos `.go` — ele está medindo o "+
			"diretório errado", visitados)
	}
}
