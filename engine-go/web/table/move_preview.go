package table

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"t20engine/aovivo"
	"t20engine/engine"
	"t20engine/tabuleiro"
)

// A PRÉVIA do movimento DURANTE O ARRASTO (ALE-203, pedido do dono: *"durante o
// drag do token, mostre a seta apontando para o token movimentando e mostre a
// distância na seta"*).
//
// Até aqui o arrasto era CEGO: o `followsFinger` escrevia um deslocamento em pixels
// e o CSS empurrava a peça, e mais nada — nenhuma seta, nenhum número. A pessoa
// só descobria o custo depois de soltar, e se tivesse estourado, desfazia e
// tentava de novo. A conta que decide o gesto chegava depois do gesto.
//
// NÃO MUTA NADA, e é essa a diferença que a põe aqui e não no
// `move.go`: a prévia é uma PERGUNTA — "se eu soltar aqui,
// quanto custa?" — e responder com o mapa remendado trocaria a peça debaixo do
// dedo de quem está arrastando. É o mesmo argumento que separa a régua dos
// comandos, e por isso a resposta é do mesmo tamanho: sinais, e só.
//
// O CAMINHO TODO e não só a perna viva (decisão do dono): as paradas já postas
// continuam desenhadas e a perna viva ESTENDE o caminho, com as cores
// recalculadas sobre o TOTAL. É o que responde a pergunta de verdade — "se eu
// soltar aqui, quanto gastei?" —, que uma perna medida sozinha não responde.

func (s Scene) MovePreviewRoutes(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/tabuleiro/{tokenId}"
	r.Post(base+"/previa/{x}/{y}", s.handlePreviewMove)
}

// previewLegsMax é o teto de rótulos que o `.templ` reserva.
//
// Mesmo contrato do `stopsMax` da régua, e pela mesma razão: o Datastar
// não tem laço, então os nós são fixos e cada um se mostra conforme a lista. Um
// rótulo sem nó é um número que ninguém vê, então os dois números têm de ser o
// mesmo.
const previewLegsMax = 12

// handlePreviewMove responde "se eu soltar aqui, como fica" em sinais.
func (s Scene) handlePreviewMove(w http.ResponseWriter, r *http.Request) {
	papel, sessionID, tabuleiroID, ok := s.whoMeasuresTheTable(w, r)
	if !ok {
		return
	}
	destino, err := quadradoDoCaminho(r, "x", "y")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tokenID := chi.URLParam(r, "tokenId")
	b := tabuleiro.BoardForRole(papel, s.deps.Boards().Get(r.Context(), sessionID, tabuleiroID))
	previa, err := dragPreview(b, s.deps.Sessions().GetState(sessionID), tokenID, destino,
		s.whoDragsInPreview(r, papel, tabuleiro.FindToken(b, tokenID)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeSignals(w, r, previa)
}

// whoDragsInPreview monta o `Mover` de quem pergunta.
//
// O ORÇAMENTO da prévia sai do mesmo `PodeMoverCom` que o desenho usa, então
// quem não pode mover aquela peça recebe uma prévia sem faixas — e não uma
// prévia mentindo o deslocamento de uma peça que não é dele.
//
// A POSSE é resolvida contra o BANCO (o `meus` do roster), como no `moveWho`, e
// nunca assumida: a primeira versão desta função escrevia `OwnsCharacter: true`
// direto, e isso teria dado a qualquer jogador o deslocamento da peça de
// qualquer outro — não pela tela, que só oferece o arrasto da peça dele, mas
// pela ROTA, que é onde a fronteira mora.
func (s Scene) whoDragsInPreview(r *http.Request, papel string, peca *tabuleiro.BoardToken) tabuleiro.Mover {
	userID := s.deps.CurrentUserID(r)
	quem := tabuleiro.Mover{UserID: userID, Role: papel}
	if papel == "gm" || peca == nil || peca.CharacterID == nil {
		return quem
	}
	campaignID, err := intDoCaminho(chi.URLParam(r, "campaignId"))
	if err != nil {
		return quem
	}
	_, meus, _ := s.tableRoster(r.Context(), userID, int64(campaignID))
	quem.OwnsCharacter = meus[*peca.CharacterID]
	return quem
}

// dragPreview mede o caminho ATUAL mais a casa sob o dedo.
//
// A lista de paradas sai do provisório quando ele existe — é o que faz a perna
// viva ESTENDER o caminho em vez de recomeçá-lo — e da posição da peça quando
// não. É a mesma leitura do `paradasDaProposta`, e ela é refeita aqui em vez de
// reusada porque aquela vive num `commandCtx` (o caminho da MUTAÇÃO) e esta não
// pode ter direito de escrita nenhum.
func dragPreview(b *tabuleiro.BoardState, st *aovivo.SessionRuntimeState, tokenID string, destino engine.Square, quem tabuleiro.Mover) (map[string]any, error) {
	if b == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto nesta mesa")
	}
	peca := tabuleiro.FindToken(b, tokenID)
	if peca == nil {
		return nil, fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	dobras := append(progressStops(b, tokenID, peca), destino)
	_, orcamento := tabuleiro.CanMoveWith(b, st, tokenID, quem)
	custos := legsCosts(dobras, moveTerrain(b))
	cabe, segundo, alem := moveWires(dobras, custos, orcamento)
	return map[string]any{
		"previafiocabe":    cabe,
		"previafiosegundo": segundo,
		"previafioalem":    alem,
		"previarotulos":    previewLabels(dobras, custos),
		"previatexto":      previewPhrase(custos, orcamento),
	}, nil
}

// progressStops são as dobras do caminho já desenhado, ou a casa da peça.
func progressStops(b *tabuleiro.BoardState, tokenID string, peca *tabuleiro.BoardToken) []engine.Square {
	if p := b.Pending; p != nil && p.TokenID == tokenID && len(p.Stops) > 0 {
		return append([]engine.Square(nil), p.Stops...)
	}
	return []engine.Square{{X: peca.X, Y: peca.Y}}
}

// previewLabels empacota cada perna num trio de números que a tela desenha:
// o rótulo, e onde ele pousa.
//
// Trio e não três listas paralelas: três listas se desalinham no dia em que uma
// delas for filtrada, e o desalinhamento aparece como um número pousado sobre a
// perna errada — que é uma mentira convincente, não um erro.
func previewLabels(dobras []engine.Square, custos []int) []map[string]any {
	pernas := moveLegs(dobras, custos)
	if len(pernas) > previewLegsMax {
		pernas = pernas[:previewLegsMax]
	}
	out := make([]map[string]any, 0, len(pernas))
	for _, p := range pernas {
		out = append(out, map[string]any{"t": p.Rotulo, "x": p.MeioX, "y": p.MeioY})
	}
	return out
}

// previewSignals declaram a seta viva no navegador, com valores INICIAIS.
//
// Não é o que faz a prévia existir — o sinal do Datastar é um proxy e nasce na
// primeira leitura, e a prévia MEDIDA continua funcionando com esta linha
// removida (conferido sabotando o declarante e rodando o e2e do arrasto). O que
// ela dá é o valor de partida explícito, que é o que separa "vazio porque ainda
// não mediu" de "vazio porque o sinal não existe" para quem for ler daqui a um
// ano.
//
// `previax`/`previay` nascem NULOS e não zero, e essa parte muda comportamento:
// zero é uma casa legítima do plano, e um arrasto que começasse nela cairia na
// trava do "só pede quando o quadrado muda" e não pediria a primeira prévia.
const previewSignals = "previafiocabe: '', previafiosegundo: '', previafioalem: '', " +
	"previarotulos: [], previatexto: '', previax: null, previay: null"

// legsReserve é a contagem que o `.templ` percorre para desenhar os nós
// fixos dos rótulos. Sai do MESMO teto que o servidor corta — escritos em dois
// lugares, uma perna nasceria medida e sem rótulo.
func legsReserve() []int {
	reserva := make([]int, previewLegsMax)
	for i := range reserva {
		reserva[i] = i
	}
	return reserva
}

// existsPreviewLabel esconde o nó da perna que a prévia não tem.
//
// VAZIO também esconde, e não é a mesma pergunta que "existe": a perna de zero
// quadrado devolve texto vazio de propósito (ver `metersLeg`), e um `<text>`
// sem conteúdo continuaria ocupando o nó com o halo do contorno.
func existsPreviewLabel(i int) string {
	return list("previarotulos", fmt.Sprintf("(lista[%d]?.t ?? '') !== ''", i))
}

// previewText e previewMid leem o trio que o servidor mandou.
func previewText(i int) string {
	return list("previarotulos", fmt.Sprintf("lista[%d]?.t ?? ''", i))
}

// previewMid é o eixo `x` ou `y` do meio da perna, em QUADRADOS.
//
// Ele sai multiplicado pelo `--quadrado` e menos a vista na própria expressão do
// atributo, porque o rótulo mora FORA do grupo que escala — se morasse dentro, o
// `scale` multiplicaria a fonte e 12px virariam 1000px no zoom máximo.
func previewMid(i int, eixo string) string {
	return list("previarotulos", fmt.Sprintf("lista[%d]?.%s ?? 0", i, eixo))
}

// previewPhrase diz o custo e a faixa, na mesma língua do rodapé.
//
// O `moveView` é montado aqui à mão porque a prévia não tem provisório: ela
// mede uma proposta que ainda não existe. O que ela NÃO faz é reescrever a
// regra — o `spentActions` é o mesmo do rodapé, e é ele que garante que soltar
// a peça não mude a frase que a pessoa acabou de ler.
func previewPhrase(custos []int, orcamento int) string {
	total := 0
	for _, c := range custos {
		total += c
	}
	metros := meters(float64(total)*engine.SquareMetres) + "m"
	if orcamento < 0 {
		return fmt.Sprintf("%d %s (%s)", total, quadradosEmPortugues(total), metros)
	}
	return fmt.Sprintf("%d de %d quadrados (%s) · %s",
		total, orcamento, metros, spentActions(&moveView{Custo: total, Orcamento: orcamento}))
}
