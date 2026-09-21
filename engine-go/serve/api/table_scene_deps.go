package api

import (
	"context"
	"net/http"
	"t20engine/app/boards"
	"t20engine/app/session"

	"t20engine/domain/board"
	"t20engine/domain/live"
	"t20engine/infra/events"
	"t20engine/serve/web/sheetui"
)

// A MESA, com adaptador próprio, e a maior porta do projeto.
//
// O adaptador é o núcleo mais um `tableRules`, e o `tableRules` toca quase todos
// os campos do `*Server` — o que está certo, e a razão está escrita lá: a Mesa É
// a mesa ao vivo, e a mesa ao vivo é o que aqueles stores guardam. O que ela NÃO
// tem é o livro, os trincos por personagem, a espera do desligamento e os
// métodos das outras cenas.
type tableHost struct {
	sceneCore
	rules tableRules
}

func (s *Server) tableHost() tableHost {
	return tableHost{sceneCore: s.sceneCore(), rules: s.tableRules()}
}

// O adaptador cumprindo a porta da MESA (`table.Deps`).
//
// É a porta mais larga da série, e o arquivo é o lugar de dizer por quê: esta é
// a única cena que MOVIMENTA estado ao vivo. As outras leem o banco e desenham;
// esta abre e encerra cena, move peça, pinta terreno, vira turno e empurra tudo
// para quem está olhando.
//
// O sinal de que a fronteira está no lugar: nenhum destes métodos desenha nada,
// e nenhum handler da cena toca banco fora do `Queries`.

// Os quatro STORES do estado ao vivo, inteiros.
//
// Eles são tipos de OUTROS pacotes, e é isso que os deixa atravessar: a cena
// recebe o vocabulário do domínio ao vivo, não o hospedeiro com outro nome.
// Embrulhá-los método a método daria oitenta entradas na porta e nenhuma
// fronteira a mais — é a mesma concessão do `Queries`, e ela tem o mesmo sinal
// de estar no lugar.
func (h tableHost) Boards() *boards.Store            { return h.rules.boards }
func (h tableHost) Sessions() *session.Store         { return h.rules.sessions }
func (h tableHost) Presence() *live.PresenceRegistry { return h.rules.presence }
func (h tableHost) SSE() *live.SSEHub                { return h.rules.sse }

// CharacterChanged avisa que uma ficha da mesa mexeu. A regra é da FICHA e a
// Mesa a pede emprestada, que é o que o campo `sheet` do `tableRules` diz.
func (h tableHost) CharacterChanged(characterID int64) {
	h.rules.sheet.characterChanged(characterID)
}
func (h tableHost) Bus() *events.Bus { return h.rules.bus }

// ── o estado AO VIVO ─────────────────────────────────────────────────────────

// ── PUBLICAR, que é do hospedeiro ────────────────────────────────────────────

// Os DOIS passos ficam escritos aqui, separados: o disco primeiro, o fio depois.
// Com a gravação escondida dentro do publicador, apagar o publicador — que é a
// leitura natural de "isto emite para ninguém", já que o `SSEHub` não tem
// ouvinte em produção — levaria a gravação junto, e a mesa passaria a viver só
// em memória.
func (h tableHost) PublishSessionState(sessionID int64, state *live.SessionRuntimeState) {
	h.rules.saveSession(sessionID)
	h.rules.publishSessionState(sessionID, state)
}

func (h tableHost) PublishBoardState(sessionID int64, board *board.BoardState) {
	h.rules.saveBoard(sessionID, board)
	h.rules.publishBoardState(sessionID, board)
}

func (h tableHost) PublishWhatIsLeft(ctx context.Context, sessionID int64) {
	h.rules.publishWhatIsLeft(ctx, sessionID)
}

// ── a escrita montada em SQL ─────────────────────────────────────────────────
//
// A coluna `notes` não tem query própria no sqlc — quem escreve é um SET
// montado. Ela mora no hospedeiro e não na cena porque cena que compõe SQL é
// cena com o banco dentro. O TÍTULO era o irmão dela e saiu na ALE-344: ele
// mora no `app/session`, que é onde uma escrita sem consulta gerada pode morar.

// ── a casca e a ficha embutida ───────────────────────────────────────────────
//
// O `BookAddress` NÃO está aqui: ele vem embutido do `sceneCore`
// (`scene_core.go`), junto com as outras cinco assinaturas que MAIS DE UMA cena
// pede. Declarar um segundo daria ao `*Server` dois nomes para a mesma coisa.

// PlayerSheet é a ficha EMBUTIDA de quem senta à mesa.
//
// A Mesa pede o painel PRONTO em vez de montar a cena da ficha: montá-la lá
// obrigaria a Mesa a cumprir a `sheetui.Deps` INTEIRA — e ela não usa nenhum
// dos métodos — só para desenhar um painel. (Sem número aqui de propósito: a
// porta está encolhendo fatia a fatia, e um número escrito à mão sobre uma
// família que muda envelhece sozinho.) Nulo é caminho normal, e a falha é
// silenciosa de propósito: estar numa mesa é mais importante que ver a própria
// ficha dentro dela.
func (h tableHost) PlayerSheet(r *http.Request, characterID int64) *sheetui.View {
	sheet, _, err := sheetui.New(h.rules.sheetScene, h.rules.sheetPlays).Load(
		r.Context(), currentUser(r).ID, characterID, sheetui.AskedTab(""), "", sheetui.Signals{})
	if err != nil {
		return nil
	}
	sheet.Embedded = true
	return &sheet
}
