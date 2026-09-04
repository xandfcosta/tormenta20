package table

import (
	"testing"

	"github.com/go-chi/chi/v5"
)

// TODA FERRAMENTA DESENHADA NO RASCUNHO TEM ROTA NELE (ALE-293).
//
// O rascunho reusa o `boardTable` inteiro, e com ele o TRILHO inteiro: as
// ferramentas que não são `SoMestre` — a régua e o gabarito — aparecem lá
// numeradas, com atalho de teclado, sem ninguém as ter escolhido para aquela
// tela. Foi assim que a ALE-292 entregou dois gestos que caíam em 404.
//
// **O modo de falha é o pior desta casa: o gesto oferecido que o servidor não
// atende.** Nada estoura, nada aparece no console do servidor além de um 404, e
// a tela simplesmente não muda — quem clica conclui que a ferramenta não faz
// nada, ou que travou.
//
// O guarda varre o TRILHO DE VERDADE (`MapTools`) contra o ROTEADOR DE VERDADE,
// e é essa dupla que o torna útil: uma lista escrita à mão dos dois lados
// concordaria consigo mesma para sempre. A ferramenta que nascer amanhã entra
// aqui sozinha, e quem esquecer a rota dela descobre pelo nome dela.
func TestEveryDraftToolHasARoute(t *testing.T) {
	roteador := chi.NewRouter()
	Scene{}.DraftRoutes(roteador)

	// O que cada ferramenta POSTA, com um caminho de exemplo. Não é a expressão
	// que o `.templ` monta — ela é JavaScript com concatenação, e reimplementá-la
	// aqui mediria a reimplementação. É o VERBO de cada uma, e ele muda junto com
	// a rota: uma rota renomeada derruba este caso.
	const base = "/campanhas/12/lugares/7/tabuleiro"
	posta := map[string]string{
		"":                   base + "/pecas/alguma/mover/1/1",
		ViewTool:             "", // a mão é do NAVEGADOR: ela arrasta a vista e não fala com o servidor
		FerramentaDaRegua:    base + "/regua",
		FerramentaDoGabarito: base + "/gabarito/esfera/6/0/0/1/1",
		MarkTool:             base + "/marcadores/novo/1/1",
		EraserTool:           base + "/terreno/limpar/1/1/ate/1/1",
		NewPieceTool:         base + "/pecas/nova/1/1",
	}

	medidas := 0
	for _, f := range rail(true) {
		caminho, declarada := posta[f.ID]
		if !declarada {
			// O PINCEL de terreno: um por espécie, e eles nascem da lista de
			// espécies. Escrevê-los à mão aqui seria a lista que fica para trás
			// quando a quinta espécie chegar.
			caminho = base + "/terreno/" + f.ID + "/1/1/ate/1/1"
		}
		if caminho == "" {
			continue
		}
		medidas++
		if !rotaExiste(roteador, caminho) {
			t.Errorf("a ferramenta %q (%s) é desenhada no rascunho e posta em %s, que não existe lá",
				f.Rotulo, f.ID, caminho)
		}
	}
	// A peça avulsa fica FORA da fileira numerada (ver `toolsRail`), então o
	// `rail` não a devolve — e ela é desenhada no rascunho do mesmo jeito.
	medidas++
	if !rotaExiste(roteador, posta[NewPieceTool]) {
		t.Errorf("a peça avulsa é desenhada no rascunho e posta em %s, que não existe lá", posta[NewPieceTool])
	}

	// O DENOMINADOR. Uma lista de reprovados vazia e um trilho que não devolveu
	// ferramenta nenhuma se parecem no terminal. Eram dez em setembro de 2026.
	if medidas < 8 {
		t.Fatalf("só %d ferramentas medidas — o guarda ficou cego", medidas)
	}
}

// rotaExiste pergunta ao ROTEADOR, e não a uma lista de strings.
//
// Pelo `chi.RouteContext` e não por um `ServeHTTP`: servir exigiria uma `Scene`
// com porta de verdade, e o que se quer saber é se o caminho CASA — quem
// responde 403 ou 500 depois já é outra pergunta, presa noutro caso.
func rotaExiste(roteador chi.Router, caminho string) bool {
	ctx := chi.NewRouteContext()
	return roteador.Match(ctx, "POST", caminho)
}

// A TRAVA das duas rotas de medir NÃO mora aqui, e é deliberado.
//
// Este arquivo pergunta ao ROTEADOR se o caminho casa, e mais nada. Quem prova
// que um estranho não mede o rascunho é o `api`, com servidor de verdade —
// `TestAStrangerDoesNotMeasureThePlaceDraft`. Uma tentativa de provar isso aqui
// entrou em PÂNICO em vez de reprovar: uma `Scene{}` sem porta estoura assim que
// o handler chama a porta, e um teste que mede o pânico mede o dublê.
