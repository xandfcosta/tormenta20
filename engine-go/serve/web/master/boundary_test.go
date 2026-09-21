package master

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// A Mesa do Mestre é a maior cena a sair — treze arquivos, quase quatro mil
// linhas, trinta rotas — com a MENOR porta com dependências da série: dois
// métodos. O guarda existe para que ela continue assim, e a tentação aqui não é
// a mesma da administração.
//
// Lá o risco era pedir a `Config` inteira. Aqui é o CATÁLOGO: seis das sete
// entradas abaixo são leitura do livro, e o caminho curto para qualquer dado
// novo é chamar `catalog.Resource` direto, como o Improviso fazia até esta
// fatia — sem ciclo, sem erro, só a divisão vazando por baixo. É o mesmo achado
// que o guarda da forja pegou no primeiro dia, com `items.go`.
//
// A regra que decide, e que já está escrita no guia: **o destino de uma função
// é a DEPENDÊNCIA dela.** Leu catálogo, é do `book`.
var permitidos = map[string]bool{
	"t20engine/domain/book":      true, // o catálogo TIPADO: é o que esta cena desenha
	"t20engine/domain/creature":  true, // o bloco de criatura, para o verbete virar rascunho
	"t20engine/domain/engine":    true, // as contas do encontro e a rolagem do improviso
	"t20engine/domain/search":    true, // o casamento e a pontuação da busca do acervo
	"t20engine/serve/web/ui":     true, // o kit de apresentação e a casca
	"t20engine/serve/web/bookui": true, // o que sabe desenhar verbete, elo e selo de página
	"t20engine/serve/web/routes": true, // os endereços que outra cena cita desta
}

func TestTheMasterDoesNotImportItsHost(t *testing.T) {
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
				"Se for `t20engine/domain/catalog`, a resposta é outra: o leitor tipado vai para o\n"+
				"`book` e a cena o chama de lá. Foi o que o Improviso fez nesta fatia.",
				name, path, path)
		}
	}

	// O DENOMINADOR: uma lista de reprovados vazia e um diretório que não foi
	// lido se parecem no terminal. O piso é alto de propósito porque esta cena é
	// grande — se ela encolher para menos de dez arquivos `.go`, alguma coisa
	// saiu e o guarda precisa ser reolhado, não silenciado.
	if visited < 10 {
		t.Fatalf("o guarda visitou só %d arquivos `.go`, e o pacote tem mais que isso: "+
			"ele está medindo o diretório errado", visited)
	}
}
