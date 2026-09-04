package api

import (
	"context"
	"t20engine/aovivo"
	"t20engine/tabuleiro"
)

// CONTEXTO: O QUE A MESA AO VIVO PUBLICA — o quadro do tabuleiro, o estado da
// sessão, e a gravação que anda junto dos dois.
//
// Este arquivo é o que sobrou de `board_commands.go` e `session_commands.go`,
// apagados na ALE-277. Eles traduziam o socket da SPA para HTTP (ALE-253): 36
// manipuladores, mil linhas, e ZERO chamadores desde que as cenas passaram a
// mutar o estado direto pela porta delas. O que não morreu com as rotas foi a
// PUBLICAÇÃO — a cena da Mesa a chama pelo `table_scene_deps.go`, e ela é a
// mesma para os dois papéis desde antes do HTTP.
//
// O `liveCtx` fica aqui pelo mesmo motivo: `board_rules.go` e `vitals_rules.go`
// recebem um, e quem os chama hoje é a cena.

// defaultTab é o id vazio, e ele significa "o tabuleiro de quem não escolheu
// nenhum" — o primeiro aberto da sessão (ALE-205, ver `BoardStore.achaLocked`).
//
// Aqui morava "TODA rota deste arquivo o usa", uma afirmação sobre a
// `SessionTrackerPage` da SPA: um slot de tabuleiro, nenhuma barra de abas,
// sempre o mesmo quadro. Não há rota neste arquivo desde a ALE-277, e a SPA não
// existe desde a ALE-272. Quem o usa hoje é o `publishWhatIsLeft` — e a razão
// mudou junto: a Mesa em Datastar TEM barra de abas, então o padrão não é mais
// "o único", é o que a mesa vê quando quem fecha uma aba não disse qual olhar.
//
// Escrever a constante em vez de um `""` solto continua sendo o que faz essa
// decisão ser lida por quem passar aqui.
const defaultTab = ""

// publishBoardState transmite às duas salas por papel e persiste.
//
// # A tela antiga tem UM slot, e por isso ela não recebe as outras abas
//
// O fluxo SSE daqui é o da SPA, que desenha "o tabuleiro da sessão" e não tem
// barra de abas (ALE-205). Mandar-lhe o quadro de uma aba que não é a padrão
// trocaria a cena na tela dela sem gesto nenhum e sem volta — a taverna viraria
// a cripta porque o mestre pintou uma casa numa aba que a SPA nem sabe que
// existe. Descartar é estritamente melhor: o que ela mostra continua sendo o
// que ela mostrava.
//
// A GRAVAÇÃO acontece sempre, e é essa a divisão: quem não vê a aba não precisa
// do quadro, mas o disco precisa de todas. Trocar as duas de lugar seria perder
// em silêncio a cena de quem não está na aba padrão, que é a ALE-154 outra vez.
func (tr tableRules) publishBoardState(sessionID int64, board *tabuleiro.BoardState) {
	if board != nil {
		go tr.persistBoardAndWarn(sessionID, board.ID)
		if board.ID != tr.boards.DefaultBoardID(context.Background(), sessionID) {
			return
		}
	}
	// O tabuleiro já numera as próprias mutações, então a ordem sai de graça —
	// `Version` sobe a cada mutação aceita. Fechar o tabuleiro manda `nil` e cai
	// no caminho "sem ordem", que reinicia o destino de propósito.
	var ordem uint64
	if board != nil {
		ordem = uint64(board.Version)
	}
	tr.sse.EmitOrdered(sessionID, "gm", "board-state", ordem, board)
	tr.sse.EmitOrdered(sessionID, "player", "board-state", ordem, tabuleiro.BoardForRole("player", board))
}

// publishWhatIsLeft é o quadro DEPOIS de fechar uma aba (ALE-205).
//
// Fechar publicava `nil`, e `nil` é a frase "esta sessão não tem tabuleiro". Com
// várias abas ela passou a poder ser MENTIRA: fechar a cripta com a taverna
// aberta apagaria a cena da tela antiga, que continuaria sem tabuleiro nenhum
// até alguém recarregar — a mesa vendo a grade sumir por uma aba que ela nem
// sabia que existia.
//
// Então quem responde é o estado: sobrou aba, vai a PADRÃO; não sobrou, vai o
// `nil` de sempre, que continua sendo a verdade.
func (tr tableRules) publishWhatIsLeft(ctx context.Context, sessionID int64) {
	tr.publishBoardState(sessionID, tr.boards.Get(ctx, sessionID, defaultTab))
}

func (tr tableRules) persistBoardAndWarn(sessionID int64, tabuleiroID string) {
	if Dirty, changed := tr.boards.Persist(context.Background(), sessionID, tabuleiroID); changed {
		tr.warnPersistenceOnBoard(sessionID, Dirty)
	}
}

func (tr tableRules) warnPersistenceOnBoard(sessionID int64, Dirty bool) {
	tr.sse.Emit(sessionID, "", "persistence-warning", map[string]any{
		"sessionId": sessionID, "Dirty": Dirty,
	})
}

// liveCtx é o que o socket chamava de `msgCtx`: quem pediu, em que mesa, com
// que papel. Resolvido uma vez por requisição.
type liveCtx struct {
	UserID     int64
	campaignID int64
	sessionID  int64
	Role       string
}

// publishSessionState transmite o estado às duas salas por papel e persiste.
// Espelha o `emitSessionState` do gateway.
func (tr tableRules) publishSessionState(sessionID int64, state *aovivo.SessionRuntimeState) {
	tr.sse.EmitOrdered(sessionID, "gm", "session-state", state.Seq, state)
	tr.sse.EmitOrdered(sessionID, "player", "session-state", state.Seq, aovivo.RedactForPlayers(state))
	tr.emSegundoPlano.Add(1)
	go func() {
		defer tr.emSegundoPlano.Done()
		tr.persistSessionAndWarn(sessionID)
	}()
}

// persistSessionAndWarn persiste e avisa a mesa SÓ quando o sinal de sujeira
// vira — primeira falha, ou uma tentativa que se recuperou. Espelha o
// `persistAndWarn` do gateway; quem é dono do sinal é o store.
func (tr tableRules) persistSessionAndWarn(sessionID int64) {
	Dirty, changed := tr.sessions.Persist(context.Background(), sessionID)
	if !changed {
		return
	}
	tr.sse.Emit(sessionID, "", "persistence-warning", map[string]any{
		"sessionId": sessionID, "Dirty": Dirty,
	})
}
