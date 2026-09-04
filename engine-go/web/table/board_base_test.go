package table

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// NENHUM GESTO DO TABULEIRO ESCREVE O PRÓPRIO CAMINHO (ALE-292).
//
// O tabuleiro tem DUAS superfícies desde o rascunho de lugar: a cena que a mesa
// está jogando e a cena que o mestre monta no acervo, fora da sessão. O mesmo
// desenho serve às duas, e o que muda é para ONDE os gestos postam — que é o
// `BoardView.Base`.
//
// Este guarda existe porque o modo de falha é SILENCIOSO e assimétrico. Uma
// chamada que continuasse escrevendo `/mesa/%d/%d/tabuleiro` compilaria, passaria
// em toda revisão de diff e funcionaria perfeitamente na Mesa — o defeito só
// apareceria no rascunho, onde `v.SessionID` é ZERO: o gesto postaria em
// `/mesa/12/0/tabuleiro/…`, um endereço que existe, responde 404 ou 403, e
// devolve uma tela que não mudou. Pintar não pinta, e nada explica por quê.
//
// É a forma da seção "Como uma convenção passa a valer": a regra é mecanizável
// com o que já roda, então ela é guarda e não parágrafo — e falha com o nome do
// arquivo e da linha, que é a diferença entre "conserte isto" e "procure".
func TestNoBoardRouteIsHandwritten(t *testing.T) {
	// Os dois ÚNICOS lugares onde o caminho do tabuleiro pode ser escrito. Eles
	// são a definição do prefixo; proibi-los seria proibir a regra de existir.
	const ondeOPrefixoMora = "board_view.go"

	arquivos, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("varrer o pacote: %v", err)
	}
	templs, err := filepath.Glob("*.templ")
	if err != nil {
		t.Fatalf("varrer os templs: %v", err)
	}
	arquivos = append(arquivos, templs...)

	medidos, usosDaBase := 0, 0
	for _, arquivo := range arquivos {
		// O GERADO não conta: ele é a saída do `.templ`, e uma violação nele já
		// foi acusada na fonte. O de teste também não — este arquivo cita o
		// literal proibido para poder proibi-lo.
		if strings.HasSuffix(arquivo, "_templ.go") || strings.HasSuffix(arquivo, "_test.go") {
			continue
		}
		bruto, err := os.ReadFile(arquivo)
		if err != nil {
			t.Fatalf("ler %s: %v", arquivo, err)
		}
		medidos++
		for numero, linha := range strings.Split(string(bruto), "\n") {
			usosDaBase += strings.Count(linha, "v.Base")
			// O que se procura é um CAMINHO, e não a palavra: o import do
			// pacote `t20engine/tabuleiro` casa com ela e não é rota nenhuma.
			// Por isso a linha só conta quando o caminho vem montado — com o
			// `/mesa/` na frente ou com um `%d` para o id.
			ehCaminho := strings.Contains(linha, "/tabuleiro") &&
				(strings.Contains(linha, "/mesa/") || strings.Contains(linha, "%d"))
			if !ehCaminho || arquivo == ondeOPrefixoMora {
				continue
			}
			// A REGISTRAÇÃO da rota é o chi, e ela escreve o padrão com os
			// parâmetros nomeados (`{campaignId}`) — não é um gesto postando.
			if strings.Contains(linha, "{campaignId}") || strings.Contains(linha, "{placeId}") {
				continue
			}
			t.Errorf("%s:%d escreve o caminho do tabuleiro à mão: use o `v.Base`, senão o gesto do rascunho posta na mesa\n\t%s",
				arquivo, numero+1, strings.TrimSpace(linha))
		}
	}

	// O DENOMINADOR, nas duas pontas. Uma lista de reprovados vazia e uma
	// varredura que não abriu arquivo nenhum se parecem no terminal — e um
	// `Base` que ninguém usasse diria que o campo é enfeite.
	if medidos < 20 {
		t.Fatalf("só %d arquivos varridos — o guarda ficou cego", medidos)
	}
	if usosDaBase < 15 {
		t.Fatalf("só %d usos de `v.Base` — o prefixo voltou a ser literal em algum lugar", usosDaBase)
	}
}
