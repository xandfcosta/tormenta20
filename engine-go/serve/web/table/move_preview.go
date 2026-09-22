package table

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// A PRÉVIA do movimento DURANTE O ARRASTO: a seta apontando para a peça que se
// move e a distância escrita nela.
//
// Sem ela o arrasto é CEGO — o `followsFinger` escreve um deslocamento em pixels
// e o CSS empurra a peça, e mais nada. A pessoa só descobre o custo depois de
// soltar, e a conta que decide o gesto chega depois do gesto.
//
// NÃO MUTA NADA, e é essa a diferença que a põe aqui e não no `move.go`: a
// prévia é uma PERGUNTA — "se eu soltar aqui, quanto custa?" — e responder com o
// mapa remendado trocaria a peça debaixo do dedo de quem está arrastando. É o
// mesmo argumento que separa a régua dos comandos, e por isso a resposta é do
// mesmo tamanho: sinais, e só.
//
// O CAMINHO TODO e não só a perna viva: as paradas já postas continuam
// desenhadas e a perna viva ESTENDE o caminho, com as cores recalculadas sobre o
// TOTAL. É o que responde a pergunta de verdade — "se eu soltar aqui, quanto
// gastei?" —, que uma perna medida sozinha não responde.

func (s Scene) MovePreviewRoutes(r chi.Router) {
	base := sessionPattern + "/tabuleiro/{tokenId}"
	r.Post(base+"/previa", s.handlePreviewMove)
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
	role, sessionID, boardID, ok := s.whoMeasuresTheTable(w, r)
	if !ok {
		return
	}
	destination, err := squareOnly(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tokenID := chi.URLParam(r, "tokenId")
	onBoard, err := s.deps.Boards().Get(r.Context(), sessionID, boardID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b := board.BoardForRole(role, onBoard)
	state, err := s.deps.Sessions().State(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	preview, err := dragPreview(b, state, tokenID, destination,
		s.whoDragsInPreview(r, role, board.FindToken(b, tokenID)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeSignals(w, r, preview)
}

// whoDragsInPreview monta o `Mover` de quem pergunta.
//
// O ORÇAMENTO da prévia sai do mesmo `board.CanMoveWith` que o desenho usa, então
// quem não pode mover aquela peça recebe uma prévia sem faixas — e não uma
// prévia mentindo o deslocamento de uma peça que não é dele.
//
// A POSSE é resolvida contra o BANCO (o `mine` do roster), como no `moveWho`, e
// nunca assumida: um `OwnsCharacter: true` escrito direto daria a qualquer
// jogador o deslocamento da peça de qualquer outro — não pela tela, que só
// oferece o arrasto da peça dele, mas pela ROTA, que é onde a fronteira mora.
func (s Scene) whoDragsInPreview(r *http.Request, role string, token *board.BoardToken) board.Mover {
	userID := s.deps.CurrentUserID(r)
	who := board.Mover{UserID: userID, Role: role}
	if role == "gm" || token == nil || token.CharacterID == nil {
		return who
	}
	campaignID, err := intDoCaminho(chi.URLParam(r, "campaignId"))
	if err != nil {
		return who
	}
	_, mine, _ := s.tableRoster(r.Context(), userID, int64(campaignID))
	who.OwnsCharacter = mine[*token.CharacterID]
	return who
}

// dragPreview mede o caminho ATUAL mais a casa sob o dedo.
//
// A lista de paradas sai do provisório quando ele existe — é o que faz a perna
// viva ESTENDER o caminho em vez de recomeçá-lo — e da posição da peça quando
// não. É a mesma leitura do `paradasDaProposta`, e ela é refeita aqui em vez de
// reusada porque aquela vive num `commandCtx` (o caminho da MUTAÇÃO) e esta não
// pode ter direito de escrita nenhum.
func dragPreview(b *board.BoardState, st *live.SessionRuntimeState, tokenID string, destination engine.Square, who board.Mover) (map[string]any, error) {
	if b == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto nesta mesa")
	}
	token := board.FindToken(b, tokenID)
	if token == nil {
		return nil, fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	folds := append(progressStops(b, tokenID, token), destination)
	_, budget := board.CanMoveWith(b, st, tokenID, who)
	costs := legsCosts(folds, moveTerrain(b))
	fits, segundo, beyond := moveWires(folds, costs, budget)
	return map[string]any{
		"preview_arrow_fits":   fits,
		"preview_arrow_second": segundo,
		"preview_arrow_beyond": beyond,
		"preview_labels":       previewLabels(folds, costs),
		"preview_text":         previewPhrase(costs, budget),
	}, nil
}

// progressStops são as dobras do caminho já desenhado, ou a casa da peça.
func progressStops(b *board.BoardState, tokenID string, token *board.BoardToken) []engine.Square {
	if p := b.Pending; p != nil && p.TokenID == tokenID && len(p.Stops) > 0 {
		return append([]engine.Square(nil), p.Stops...)
	}
	return []engine.Square{{X: token.X, Y: token.Y}}
}

// previewLabels empacota cada perna num trio de números que a tela desenha:
// o rótulo, e onde ele pousa.
//
// Trio e não três listas paralelas: três listas se desalinham no dia em que uma
// delas for filtrada, e o desalinhamento aparece como um número pousado sobre a
// perna errada — que é uma mentira convincente, não um erro.
func previewLabels(folds []engine.Square, costs []int) []map[string]any {
	legs := moveLegs(folds, costs)
	if len(legs) > previewLegsMax {
		legs = legs[:previewLegsMax]
	}
	out := make([]map[string]any, 0, len(legs))
	for _, p := range legs {
		out = append(out, map[string]any{"t": p.Label, "x": p.MidX, "y": p.MidY})
	}
	return out
}

// previewSignals declaram a seta viva no navegador, com valores INICIAIS.
//
// Não é o que faz a prévia existir — o sinal do Datastar é um proxy e nasce na
// primeira leitura. O que ela dá é o valor de partida explícito, que separa
// "vazio porque ainda não mediu" de "vazio porque o sinal não existe".
//
// `preview_x`/`preview_y` nascem NULOS e não zero, e essa parte muda
// comportamento: zero é uma casa legítima do plano, e um arrasto que começasse
// nela cairia na trava do "só pede quando o quadrado muda" e não pediria a
// primeira prévia.
const previewSignals = "preview_arrow_fits: '', preview_arrow_second: '', preview_arrow_beyond: '', " +
	"preview_labels: [], preview_text: '', preview_x: null, preview_y: null"

// legsReserve é a contagem que o `.templ` percorre para desenhar os nós
// fixos dos rótulos. Sai do MESMO teto que o servidor corta — escritos em dois
// lugares, uma perna nasceria medida e sem rótulo.
func legsReserve() []int {
	reserve := make([]int, previewLegsMax)
	for i := range reserve {
		reserve[i] = i
	}
	return reserve
}

// existsPreviewLabel esconde o nó da perna que a prévia não tem.
//
// VAZIO também esconde, e não é a mesma pergunta que "existe": a perna de zero
// quadrado devolve texto vazio de propósito (ver `metersLeg`), e um `<text>`
// sem conteúdo continuaria ocupando o nó com o halo do contorno.
func existsPreviewLabel(i int) string {
	return list("preview_labels", fmt.Sprintf("(lista[%d]?.t ?? '') !== ''", i))
}

// previewText e previewMid leem o trio que o servidor mandou.
func previewText(i int) string {
	return list("preview_labels", fmt.Sprintf("lista[%d]?.t ?? ''", i))
}

// previewMid é o eixo `x` ou `y` do meio da perna, em QUADRADOS.
//
// Ele sai multiplicado pelo `--quadrado` e menos a vista na própria expressão do
// atributo, porque o rótulo mora FORA do grupo que escala — se morasse dentro, o
// `scale` multiplicaria a fonte e 12px virariam 1000px no zoom máximo.
func previewMid(i int, axis string) string {
	return list("preview_labels", fmt.Sprintf("lista[%d]?.%s ?? 0", i, axis))
}

// previewPhrase diz o custo e a faixa, na mesma língua do rodapé.
//
// O `moveView` é montado aqui à mão porque a prévia não tem provisório: ela
// mede uma proposta que ainda não existe. O que ela NÃO faz é reescrever a
// regra — o `spentActions` é o mesmo do rodapé, e é ele que garante que soltar
// a peça não mude a frase que a pessoa acabou de ler.
func previewPhrase(costs []int, budget int) string {
	total := 0
	for _, c := range costs {
		total += c
	}
	metres := meters(float64(total)*engine.SquareMetres) + "m"
	if budget < 0 {
		return fmt.Sprintf("%d %s (%s)", total, quadradosEmPortugues(total), metres)
	}
	return fmt.Sprintf("%d de %d quadrados (%s) · %s",
		total, budget, metres, spentActions(&moveView{Cost: total, Budget: budget}))
}
