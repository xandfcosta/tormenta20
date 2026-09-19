package campaigns

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO.
//
// É a cena de porta mais larga, e o guarda existe para que a largura pare aqui.
// (Sem número: ela está encolhendo fatia a fatia na ALE-348, e um número escrito
// à mão sobre uma família que muda envelhece sozinho — este já dizia ONZE com
// vinte na porta.) A tentação tem nome: **o `s.db`**, porque o caminho curto
// para qualquer coluna nova é pedir o banco cru de volta.
//
// A resposta certa é a PERGUNTA, e melhor ainda é o CASO DE USO: gravar o texto
// da campanha não atravessa mais esta porta — ele é `campaign.Lifecycle`, e o
// SQL mora lá, inteiro e visível (ALE-348). O `Queries` continua permitido porque três das quatro telas leem e
// escrevem as próprias tabelas — é a concessão da forja e da administração —,
// e o sinal de que ela está no lugar é nenhum handler tocar banco fora dele.
//
// **O `web/characters` na lista é cena lendo cena, e é deliberado.** A lista de
// campanhas desenha o herói de quem pede em cada mesa, e a linha de classes dele
// é a MESMA do cartão do elenco. A alternativa era o mesmo texto montado em dois
// lugares. A direção continua legal — quem importa é quem desenha depois — e é
// a mesma concessão que a Mesa faz com o bestiário do `web/master`.
var permitidos = map[string]bool{
	// O `app/` NÃO é concessão, é a porta encolhendo (ALE-348): ele está ABAIXO
	// desta cena, então não há ciclo para desviar e não há interface a declarar.
	// O vocabulário (`app.Caller`) e a TRAVA (`session.Access`) chegam por
	// parâmetro do construtor, como na Mesa e na ficha — e cada entrada que vira
	// caso de uso SAI da `Deps` em vez de ganhar um adaptador novo.
	"t20engine/app":                  true,
	"t20engine/app/campaign":         true,
	"t20engine/app/session":          true,
	"t20engine/domain/campaign":      true, // as REGRAS: nome, descrição, regras opcionais
	"t20engine/infra/db/sqlcgen":     true, // as linhas do banco, pelo `Queries` da porta
	"t20engine/infra/wire":           true,
	"t20engine/domain/search":        true, // o casamento da busca da lista
	"t20engine/domain/sheet":         true, // a forma do personagem que senta à mesa
	"t20engine/serve/web/characters": true, // a linha de classes do herói, uma só
	"t20engine/serve/web/routes":     true, // o endereço da Mesa, citado daqui
	"t20engine/serve/web/ui":         true, // o kit, a casca e a identidade visual
}

// SEM lista de recusa da biblioteca padrão, e a ausência é uma decisão.
//
// O guarda irmão do `campaign` tem uma, com UMA entrada, porque lá a tentação
// foi MEDIDA. Uma escrita aqui por precaução, recusando `os` e `path/filepath`,
// reprova o PRÓPRIO guarda — que importa `os` para ler o diretório. **Lista de
// perigo imaginado envelhece; lista de defeito acontecido, não.**
//
// O `database/sql` desta cena é legítimo: o `trimOrNull` dela traduz vazio para
// NULL antes de o valor atravessar. O que ela não pode é montar a instrução, e
// isso não é um import — é uma decisão que só a leitura do corpo mostra.

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
				"porta cresce com a PERGUNTA, e encolhe quando a pergunta vira CASO DE\n"+
				"USO (ver o `campaign.Lifecycle`). Ela nunca cresce com a tabela.",
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
