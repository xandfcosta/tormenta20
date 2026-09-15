package api

import (
	"context"
	"t20engine/domain/board"
	"t20engine/domain/live"
)

// O QUE A MESA AO VIVO PUBLICA — o quadro do tabuleiro, o estado da sessão, e a
// gravação que anda junto dos dois. Quem chama é a cena da Mesa, pelo
// `table_scene_deps.go`, e a publicação é a mesma para os dois papéis.

// defaultTab é o id vazio, e ele significa "o tabuleiro de quem não escolheu
// nenhum" — o primeiro aberto da sessão (ver `BoardStore.achaLocked`). É o que
// a mesa vê quando quem fecha uma aba não disse qual olhar.
//
// Constante em vez de um `""` solto porque é o que faz essa decisão ser lida
// por quem passar aqui.
const defaultTab = ""

// saveBoard GRAVA o tabuleiro no disco, e não faz mais nada.
//
// Separada do `publishBoardState` e não uma linha dentro dele: o `SSEHub` não
// tem ouvinte em produção, então a leitura natural de "publicar para ninguém" é
// apagar a função — e ela levaria a gravação junto. **A mesa passaria a viver
// só em memória.** Duas funções com nomes que dizem o que fazem custam uma
// linha no chamador e tiram essa possibilidade do mapa.
//
// Em GOROUTINE porque o mestre não espera o disco no meio do turno: num prato
// girante o toque chega a centenas de milissegundos.
func (tr tableRules) saveBoard(sessionID int64, board *board.BoardState) {
	if board == nil {
		return
	}
	go tr.persistBoardAndWarn(sessionID, board.ID)
}

// saveSession GRAVA o estado da sessão. O irmão do `saveBoard`, e pela mesma
// razão — ver lá.
//
// Ela conta no `emSegundoPlano` e o tabuleiro não: quem espera no `Shutdown` é
// este contador. Diferença conhecida e não "arrumada" de passagem — mexer no
// que o desligamento espera é decisão de quem mediu o desligamento.
func (tr tableRules) saveSession(sessionID int64) {
	tr.emSegundoPlano.Add(1)
	go func() {
		defer tr.emSegundoPlano.Done()
		tr.persistSessionAndWarn(sessionID)
	}()
}

// publishBoardState transmite às duas salas por papel. Ela NÃO grava — ver o
// `saveBoard`.
//
// > **O canal dela não tem ouvinte em produção**: quem abria conexão no
// > `SSEHub` era a rota `/events`, e a Mesa em Datastar tem fluxo próprio pelo
// > `events.Bus`. Ela fica porque desmontar o hub é trabalho com desenho
// > próprio.
//
// SÓ A ABA PADRÃO é publicada. O destino deste fluxo desenha "o tabuleiro da
// sessão" e não tem barra de abas, então mandar-lhe o quadro de outra aba
// trocaria a cena na tela dele sem gesto nenhum e sem volta — a taverna viraria
// a cripta porque o mestre pintou uma casa numa aba que ele nem sabe que existe.
//
// A GRAVAÇÃO acontece SEMPRE e a publicação não: quem não vê a aba não precisa
// do quadro, mas o disco precisa de todas. Por isso o `return` abaixo é do
// publicador e nunca do gravador — trocar as duas de lugar perderia em silêncio
// a cena de quem não está na aba padrão.
func (tr tableRules) publishBoardState(sessionID int64, state *board.BoardState) {
	if state != nil && state.ID != tr.boards.DefaultBoardID(context.Background(), sessionID) {
		return
	}
	// O tabuleiro já numera as próprias mutações, então a ordem sai de graça —
	// `Version` sobe a cada mutação aceita. Fechar o tabuleiro manda `nil` e cai
	// no caminho "sem ordem", que reinicia o destino de propósito.
	var ordem uint64
	if state != nil {
		ordem = uint64(state.Version)
	}
	tr.sse.EmitOrdered(sessionID, "gm", "board-state", ordem, state)
	tr.sse.EmitOrdered(sessionID, "player", "board-state", ordem, board.BoardForRole("player", state))
}

// publishWhatIsLeft é o quadro DEPOIS de fechar uma aba.
//
// Publicar `nil` seco seria a frase "esta sessão não tem tabuleiro", e com
// várias abas ela é MENTIRA: fechar a cripta com a taverna aberta apagaria a
// grade da tela de quem nem sabia que a cripta existia. Quem responde é o
// estado — sobrou aba, vai a PADRÃO; não sobrou, vai o `nil`, que aí é verdade.
//
// Ela GRAVA e publica, os dois passos escritos. O `Close` do store já apagou a
// linha da aba fechada; o que esta regravação alcança é a que SOBROU.
func (tr tableRules) publishWhatIsLeft(ctx context.Context, sessionID int64) {
	sobrou := tr.boards.Get(ctx, sessionID, defaultTab)
	tr.saveBoard(sessionID, sobrou)
	tr.publishBoardState(sessionID, sobrou)
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

// liveCtx é quem pediu, em que mesa, com que papel. Resolvido uma vez por
// requisição.
type liveCtx struct {
	UserID     int64
	campaignID int64
	sessionID  int64
	Role       string
}

// publishSessionState transmite o estado às duas salas por papel. Ela NÃO grava
// — ver o `saveSession`.
func (tr tableRules) publishSessionState(sessionID int64, state *live.SessionRuntimeState) {
	tr.sse.EmitOrdered(sessionID, "gm", "session-state", state.Seq, state)
	tr.sse.EmitOrdered(sessionID, "player", "session-state", state.Seq, live.RedactForPlayers(state))
}

// persistSessionAndWarn persiste e avisa a mesa SÓ quando o sinal de sujeira
// vira — primeira falha, ou uma tentativa que se recuperou. Quem é dono do
// sinal é o store.
func (tr tableRules) persistSessionAndWarn(sessionID int64) {
	Dirty, changed := tr.sessions.Persist(context.Background(), sessionID)
	if !changed {
		return
	}
	tr.sse.Emit(sessionID, "", "persistence-warning", map[string]any{
		"sessionId": sessionID, "Dirty": Dirty,
	})
}
