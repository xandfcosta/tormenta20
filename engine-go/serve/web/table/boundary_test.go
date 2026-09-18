package table

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO.
//
// Esta é a maior cena e a única que MOVIMENTA estado ao vivo — abre e encerra
// cena, move peça, pinta terreno, mede distância, vira turno e empurra tudo para
// quem está olhando, por dois stores em memória, um hub de SSE e um barramento.
//
// # O que a lista PERMITE, e por quê
//
// Os stores (`board`, `live`, `events`) atravessam INTEIROS pela porta, e é isso
// que os deixa entrar aqui: são tipos de outros pacotes, o vocabulário do
// domínio ao vivo, não o hospedeiro com outro nome. Embrulhá-los método a método
// daria oitenta entradas na porta e nenhuma fronteira a mais.
//
// O `web/routes` não é concessão: é o pacote que não importa NADA, e o guarda
// dele (`TestTheAddressesImportNothing`) é quem garante que a permissão daqui
// não vira uma porta para o resto do projeto.
//
// **O `web/master` e o `web/sheetui` são cena lendo cena**, e as duas são
// concessões declaradas: o bestiário da Mesa é o MESMO desenho da cena do
// mestre, parametrizado pelo endereço, e a ficha embutida é o MESMO painel da
// ficha. A alternativa nos dois casos era um segundo desenho mantido em dois
// lugares. A direção continua legal — quem importa é quem desenha depois.
//
// # A tentação tem nome: o `s.db`
//
// `title` e `notes` não têm query própria no sqlc, e compor `setBuilder` +
// `"UPDATE sessions"` aqui dentro é pôr o banco dentro da cena. O
// `database/sql` fica fora da lista, e não é por precaução: é a tentação
// MEDIDA.
var permitidos = map[string]bool{
	"t20engine/domain/live":       true, // a fila, a cena e a presença, pela porta
	"t20engine/domain/book":       true, // o catálogo tipado do bestiário e das condições
	"t20engine/domain/catalog":    true, // ver a nota abaixo — é o IsCondition, não o Resource
	"t20engine/domain/creature":   true, // o bloco de criatura que o NPC edita
	"t20engine/infra/db/sqlcgen":  true, // as linhas do banco, pelo `Queries` da porta
	"t20engine/domain/engine":     true, // a medição de área e a ficha computada
	"t20engine/domain/sheet":      true, // o `LoadAndCompute` que dá a Defesa do Grupo
	"t20engine/infra/events":      true, // o barramento, para o stream saber o que houve
	"t20engine/domain/markdown":   true, // as notas do mestre, que saíram daqui na fatia 1
	"t20engine/infra/db/dbvalue":  true,
	"t20engine/domain/board":      true, // o mapa: peça, marcador, terreno, lugar
	"t20engine/serve/web/bookui":  true, // o endereço do livro que o bestiário linka
	"t20engine/serve/web/master":  true, // o MESMO desenho do bestiário do mestre
	"t20engine/serve/web/routes":  true, // os endereços que ela cita da campanha
	"t20engine/serve/web/sheetui": true, // o MESMO painel da ficha, embutido
	"t20engine/serve/web/ui":      true, // o kit, a casca e a identidade visual
}

// A lista de RECUSA da biblioteca padrão, com UMA entrada.
//
// **Um guarda de fronteira que filtra por prefixo do módulo é cego para tudo que
// não tem esse prefixo** — o `database/sql` passa limpo. Provado por sabotagem:
// com um `sql.NullString` neste pacote, o build passou E o guarda passou.
//
// **Lista de perigo imaginado envelhece; lista de defeito acontecido, não.**
// Esta tem uma entrada porque houve UM defeito, e não porque `os` e
// `path/filepath` parecem perigosos. A lista curta é o que a mantém honesta.
var recusados = map[string]bool{
	"database/sql": true,
}

// O `catalog` está na lista por UMA chamada: `IsCondition`.
//
// Ele NÃO é o `Resource` cru, e a diferença some quando alguém lê só o import:
// `IsCondition` é o acessor tipado; `Resource` devolve bytes, e quem desempacota
// bytes de catálogo é do livro.
func TestTheTableSceneDoesNotImportItsHost(t *testing.T) {
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
			if recusados[caminho] {
				t.Errorf("%s importa %q — esta cena NÃO monta SQL.\n"+
					"As duas colunas sem query no sqlc (`title` e `notes`) são gravadas\n"+
					"pelo hospedeiro, que sabe o nome delas, que vazio é NULL e que a\n"+
					"linha tem um `updatedAt` a carimbar. Ver `SaveNotes` na porta.",
					nome, caminho)
				continue
			}
			if !strings.HasPrefix(caminho, "t20engine/") || permitidos[caminho] {
				continue
			}
			t.Errorf("%s importa %q.\n"+
				"Se a cena precisa de algo de lá, DECLARE na `Deps` e receba de quem monta.\n"+
				"Acrescentar o import à lista transforma a porta em enfeite — e se %q for\n"+
				"o `api`, é ciclo, porque ele importa esta cena para montar o roteador.\n"+
				"Se a vontade for o banco cru para uma coluna sem query, a resposta é\n"+
				"outra: a porta cresce com a PERGUNTA (ver `SaveNotes`), não com a tabela.",
				nome, caminho, caminho)
		}
	}

	// O DENOMINADOR: um diretório não lido e uma lista de reprovados vazia se
	// parecem no terminal. O piso é folgado de propósito — ele existe para acusar
	// o guarda medindo o diretório ERRADO, não para contar arquivos.
	if visitados < 40 {
		t.Fatalf("o guarda visitou só %d arquivos `.go` — ele está medindo o "+
			"diretório errado", visitados)
	}
}
