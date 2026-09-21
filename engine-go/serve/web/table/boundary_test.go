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
// `title` e `notes` não têm query própria no sqlc, e montar o `UPDATE` deles
// aqui dentro é pôr o banco dentro da cena. O `database/sql` fica fora da
// lista, e não é por precaução: é a tentação MEDIDA.
var permitidos = map[string]bool{
	// O vocabulário do grupo: quem pede e as recusas tipadas.
	"t20engine/app": true,
	// O `app/boards` é o store dos tabuleiros abertos — a orquestração que a
	// ALE-344 tirou de `domain/board`. A cena o recebe pela porta, como recebia
	// antes: o que mudou é de onde ele vem, não o que ele é.
	"t20engine/app/boards": true,
	// O `app/initiative` é quem entra na fila, pela mesma razão.
	"t20engine/app/initiative": true,
	// O `app/campaign` é o ELENCO de NPCs, e entra pela mesma razão dos dois
	// acima: é caso de uso, chega por parâmetro, e está ABAIXO desta cena.
	//
	// Ele veio junto com a tabela ganhando dono: `campaign_creatures` é acervo
	// da CAMPANHA — o NPC preparado na quinta sobrevive à sessão de sábado —, e
	// esta cena escrevia nela direto, com uma segunda cópia da trava. As quatro
	// escritas saíram daqui (ALE-353).
	"t20engine/app/campaign": true,
	// O `app/rest` é o descanso e a expiração de escopo, pela mesma razão.
	"t20engine/app/rest": true,
	// O `app/session` NÃO é concessão, é a razão do guarda existir ficar menor
	// (ALE-344): ele está ABAIXO desta cena e do `serve/api`, então não há ciclo
	// para desviar — e por isso não há interface. Cinco entradas da porta saíram
	// com ele. A direção continua legal: quem importa é quem desenha depois.
	"t20engine/app/session":       true,
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
			if recusados[path] {
				t.Errorf("%s importa %q — esta cena NÃO monta SQL.\n"+
					"As duas colunas sem query no sqlc (`title` e `notes`) são gravadas\n"+
					"pelo hospedeiro, que sabe o nome delas, que vazio é NULL e que a\n"+
					"linha tem um `updatedAt` a carimbar. Ver `SaveNotes` na porta.",
					name, path)
				continue
			}
			if !strings.HasPrefix(path, "t20engine/") || permitidos[path] {
				continue
			}
			t.Errorf("%s importa %q.\n"+
				"Se a cena precisa de algo de lá, DECLARE na `Deps` e receba de quem monta.\n"+
				"Acrescentar o import à lista transforma a porta em enfeite — e se %q for\n"+
				"o `api`, é ciclo, porque ele importa esta cena para montar o roteador.\n"+
				"Se a vontade for o banco cru para uma coluna sem query, a resposta é\n"+
				"outra: a porta cresce com a PERGUNTA (ver `SaveNotes`), não com a tabela.",
				name, path, path)
		}
	}

	// O DENOMINADOR: um diretório não lido e uma lista de reprovados vazia se
	// parecem no terminal. O piso é folgado de propósito — ele existe para acusar
	// o guarda medindo o diretório ERRADO, não para contar arquivos.
	if visited < 40 {
		t.Fatalf("o guarda visitou só %d arquivos `.go` — ele está medindo o "+
			"diretório errado", visited)
	}
}
