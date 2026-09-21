package table

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// NENHUM GESTO DO TABULEIRO ESCREVE O PRÓPRIO CAMINHO.
//
// O tabuleiro tem DUAS superfícies desde o rascunho de lugar: a cena que a mesa
// está jogando e a cena que o mestre monta no acervo, fora da sessão. O mesmo
// desenho serve às duas, e o que muda é para ONDE os gestos postam — que é o
// `BoardView.Base`.
//
// Este guarda existe porque o modo de falha é SILENCIOSO e assimétrico. Uma
// chamada que continuasse escrevendo `/campanhas/%d/sessoes/%d/tabuleiro` compilaria, passaria
// em toda revisão de diff e funcionaria perfeitamente na Mesa — o defeito só
// apareceria no rascunho, onde `v.SessionID` é ZERO: o gesto postaria em
// `/campanhas/12/sessoes/0/tabuleiro/…`, um endereço que existe, responde 404 ou 403, e
// devolve uma tela que não mudou. Pintar não pinta, e nada explica por quê.
//
// A regra é mecanizável com o que já roda, então ela é guarda e não parágrafo —
// e falha com o nome do arquivo e da linha, que é a diferença entre "conserte
// isto" e "procure".
func TestNoBoardRouteIsHandwritten(t *testing.T) {
	// Os dois ÚNICOS lugares onde o caminho do tabuleiro pode ser escrito. Eles
	// são a definição do prefixo; proibi-los seria proibir a regra de existir.
	const ondeOPrefixoMora = "board_view.go"

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("varrer o pacote: %v", err)
	}
	templs, err := filepath.Glob("*.templ")
	if err != nil {
		t.Fatalf("varrer os templs: %v", err)
	}
	files = append(files, templs...)

	measured, baseUses := 0, 0
	for _, file := range files {
		// O GERADO não conta: ele é a saída do `.templ`, e uma violação nele já
		// foi acusada na fonte. O de teste também não — este arquivo cita o
		// literal proibido para poder proibi-lo.
		if strings.HasSuffix(file, "_templ.go") || strings.HasSuffix(file, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("ler %s: %v", file, err)
		}
		measured++
		for number, row := range strings.Split(string(raw), "\n") {
			baseUses += strings.Count(row, "v.Base")
			// O que se procura é um CAMINHO, e não a palavra: o import do
			// pacote `t20engine/domain/board` casa com ela e não é rota nenhuma.
			// Por isso a linha só conta quando o caminho vem montado — com o
			// `/campanhas/` na frente ou com um `%d` para o id.
			isPath := strings.Contains(row, "/tabuleiro") &&
				(strings.Contains(row, "/campanhas/") || strings.Contains(row, "%d"))
			if !isPath || file == ondeOPrefixoMora {
				continue
			}
			// A REGISTRAÇÃO da rota é o chi, e ela escreve o padrão com os
			// parâmetros nomeados (`{campaignId}`) — não é um gesto postando.
			if strings.Contains(row, "{campaignId}") || strings.Contains(row, "{placeId}") {
				continue
			}
			t.Errorf("%s:%d escreve o caminho do tabuleiro à mão: use o `v.Base`, senão o gesto do rascunho posta na mesa\n\t%s",
				file, number+1, strings.TrimSpace(row))
		}
	}

	// O DENOMINADOR, nas duas pontas. Uma lista de reprovados vazia e uma
	// varredura que não abriu arquivo nenhum se parecem no terminal — e um
	// `Base` que ninguém usasse diria que o campo é enfeite.
	if measured < 20 {
		t.Fatalf("só %d arquivos varridos — o guarda ficou cego", measured)
	}
	if baseUses < 15 {
		t.Fatalf("só %d usos de `v.Base` — o prefixo voltou a ser literal em algum lugar", baseUses)
	}
}
