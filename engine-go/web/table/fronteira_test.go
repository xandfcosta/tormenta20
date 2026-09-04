package table

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// A décima primeira e ÚLTIMA cena, e a maior: 47 arquivos de produção, ~14.000
// linhas, vinte rotas — e a porta mais larga da série, com trinta e um métodos.
//
// **O tamanho é o que a cena É, e o contraste que ensina é com o trilho do
// mestre**, que tem DOIS: aquele desenha o livro embutido e não toca banco;
// esta é a única cena que MOVIMENTA estado ao vivo — abre e encerra cena, move
// peça, pinta terreno, mede distância, vira turno e empurra tudo para quem está
// olhando, por dois stores em memória, um hub de SSE e um barramento.
//
// # O que a lista PERMITE, e por quê
//
// Os quatro stores (`tabuleiro`, `aovivo`, `events`) atravessam INTEIROS pela
// porta, e é isso que os deixa entrar aqui: são tipos de outros pacotes, o
// vocabulário do domínio ao vivo, não o hospedeiro com outro nome. Embrulhá-los
// método a método daria oitenta entradas na porta e nenhuma fronteira a mais.
//
// **O `web/routes` entrou na ALE-292**, e é a última das cinco cenas a
// importá-lo. Ele não é concessão: é o pacote que não importa NADA, e o guarda
// dele (`TestTheAddressesImportNothing`) é quem garante que a permissão daqui
// não vira uma porta para o resto do projeto. Esta cena passou a citá-lo porque
// o rascunho de lugar mora num endereço da CAMPANHA, e ele é citado dos dois
// lados — a crônica leva ao rascunho, e o rascunho volta para a crônica.
//
// **O `web/master` e o `web/sheetui` são cena lendo cena**, e as duas são
// concessões declaradas: o bestiário da Mesa é o MESMO desenho da cena do
// mestre, parametrizado pelo endereço, e a ficha embutida é o MESMO painel da
// ficha. A alternativa nos dois casos era um segundo desenho mantido em dois
// lugares. A direção continua legal — quem importa é quem desenha depois.
//
// # A tentação tem nome: o `s.db`
//
// Duas gravações compunham `setBuilder` + `"UPDATE sessions"` aqui dentro,
// porque `title` e `notes` não têm query própria no sqlc. Cena que compõe SQL é
// cena com o banco dentro; viraram `SaveSessionTitle` e `SaveNotes`. O
// `database/sql` não está na lista, e não é por precaução: é a tentação MEDIDA.
var permitidos = map[string]bool{
	"t20engine/aovivo":      true, // a fila, a cena e a presença, pela porta
	"t20engine/book":        true, // o catálogo tipado do bestiário e das condições
	"t20engine/catalog":     true, // ver a nota abaixo — é o IsCondition, não o Resource
	"t20engine/creature":    true, // o bloco de criatura que o NPC edita
	"t20engine/db/sqlcgen":  true, // as linhas do banco, pelo `Queries` da porta
	"t20engine/engine":      true, // a medição de área e a ficha computada
	"t20engine/events":      true, // o barramento, para o stream saber o que houve
	"t20engine/markdown":    true, // as notas do mestre, que saíram daqui na fatia 1
	"t20engine/plataforma":  true, // o carimbo ISO e o envelope de resposta
	"t20engine/tabuleiro":   true, // o mapa: peça, marcador, terreno, lugar
	"t20engine/web/bookui":  true, // o endereço do livro que o bestiário linka
	"t20engine/web/master":  true, // o MESMO desenho do bestiário do mestre
	"t20engine/web/routes":  true, // os endereços que ela cita da campanha (ALE-292)
	"t20engine/web/sheetui": true, // o MESMO painel da ficha, embutido (ALE-275)
	"t20engine/web/ui":      true, // o kit, a casca e a identidade visual
}

// A lista de RECUSA da biblioteca padrão, com UMA entrada.
//
// Ela existe porque a prosa acima nomeia o `s.db` como a tentação, e **um guarda
// de fronteira que filtra por prefixo do módulo é cego para tudo que não tem
// esse prefixo** — o `database/sql` passaria limpo. Isso não é teoria: sabotei
// com um `sql.NullString` neste pacote, o build passou E o guarda passou.
//
// É o mesmo achado que o guarda do `campaign` registrou, e a regra dele vale
// aqui inteira: **lista de perigo imaginado envelhece; lista de defeito
// acontecido, não.** Esta tem uma entrada porque houve UM defeito — duas
// gravações compondo SQL — e não porque `os` e `path/filepath` parecem
// perigosos. A lista curta é o que a mantém honesta.
var recusados = map[string]bool{
	"database/sql": true,
}

// O `catalog` está na lista por UMA chamada: `IsCondition`.
//
// Ele NÃO é o `Resource` cru, que é a família que já saiu daqui três vezes
// nesta épica — e a nota vale porque a diferença some quando alguém lê só o
// import. `IsCondition` é o acessor tipado; `Resource` devolve bytes, e quem
// desempacota bytes de catálogo é do livro.
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
	// parecem no terminal. Esta cena tem 47 arquivos de produção mais os de
	// teste — e o piso é folgado porque ele existe para acusar o guarda medindo o
	// diretório ERRADO, que é como três outros quebraram nesta mesma fatia.
	if visitados < 40 {
		t.Fatalf("o guarda visitou só %d arquivos `.go` — ele está medindo o "+
			"diretório errado", visitados)
	}
}
