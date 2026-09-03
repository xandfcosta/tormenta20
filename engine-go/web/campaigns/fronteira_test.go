package campaigns

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// Nona cena, e a de porta mais larga: ONZE métodos. O guarda existe para que a
// largura pare aqui, e a tentação tem nome — **o `s.db`**.
//
// Esta cena montava `setBuilder` + `execTouched` + `"UPDATE campaigns"` à mão,
// e o caminho curto para qualquer coluna nova é pedir o banco cru de volta.
// A resposta certa é a PERGUNTA: `SaveText` existe porque o hospedeiro é que
// sabe o nome da coluna, que vazio é NULL e que a linha tem um `updatedAt` a
// tocar. O `Queries` continua permitido porque três das quatro telas leem e
// escrevem as próprias tabelas — é a concessão da forja e da administração —,
// e o sinal de que ela está no lugar é nenhum handler tocar banco fora dele.
//
// **O `web/characters` na lista é cena lendo cena, e é deliberado.** A lista de
// campanhas desenha o herói de quem pede em cada mesa, e a linha de classes dele
// é a MESMA do cartão do elenco. A alternativa era o mesmo texto montado em dois
// lugares. A direção continua legal — quem importa é quem desenha depois — e é
// a mesma concessão que a Mesa faz com o bestiário do `web/master`.
var permitidos = map[string]bool{
	"t20engine/campaign":       true, // as REGRAS: nome, descrição, regras opcionais
	"t20engine/db/sqlcgen":     true, // as linhas do banco, pelo `Queries` da porta
	"t20engine/plataforma":     true, // o mapa de erro por campo e o carimbo ISO
	"t20engine/search":         true, // o casamento da busca da lista
	"t20engine/sheet":          true, // a forma do personagem que senta à mesa
	"t20engine/web/characters": true, // a linha de classes do herói, uma só
	"t20engine/web/routes":     true, // o endereço da Mesa, citado daqui
	"t20engine/web/ui":         true, // o kit, a casca e a identidade visual
}

// SEM lista de recusa da biblioteca padrão, e a ausência é uma decisão.
//
// O guarda irmão do `campaign` ganhou uma, com UMA entrada, porque lá havia uma
// tentação MEDIDA: a versão anterior devolvia `sql.NullString` de verdade. Aqui
// eu escrevi uma recusando `os` e `path/filepath` por precaução — e ela reprovou
// o PRÓPRIO guarda, que importa `os` para ler o diretório.
//
// Fica registrado porque é a regra que eu mesmo tinha acabado de escrever
// falhando na fatia seguinte: **lista de perigo imaginado envelhece; lista de
// defeito acontecido, não.** O `database/sql` desta cena, aliás, é legítimo —
// ela GRAVA, e o `trimOrNull` traduz vazio para NULL. O que ela não pode é
// montar a instrução, e isso não é um import: é uma decisão que só a leitura do
// `SaveText` mostra.

func TestTheCampaignsSceneDoesNotImportItsHost(t *testing.T) {
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
				"Se a vontade for o banco cru para uma coluna nova, a resposta é outra: a\n"+
				"porta cresce com a PERGUNTA (ver `SaveText`), não com a tabela.",
				nome, caminho, caminho)
		}
	}

	// O DENOMINADOR: um diretório não lido e uma lista de reprovados vazia se
	// parecem no terminal. Esta cena tem oito arquivos `.go` de produção.
	if visitados < 6 {
		t.Fatalf("o guarda visitou só %d arquivos `.go` — ele está medindo o "+
			"diretório errado", visitados)
	}
}
