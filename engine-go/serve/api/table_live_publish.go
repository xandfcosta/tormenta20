package api

import (
	"context"
	"log"

	"t20engine/domain/board"
	"t20engine/domain/live"
)

// O QUE A MESA AO VIVO PUBLICA — o quadro do tabuleiro, o estado da sessão, e a
// gravação que anda junto dos dois. Quem chama é a cena da Mesa, pelo
// `table_scene_deps.go`, e a publicação é a mesma para os dois papéis.

// defaultTab é o id vazio, e ele significa "o tabuleiro de quem não escolheu
// nenhum" — o primeiro aberto da sessão (ver `boards.Store`). É o que
// a mesa vê quando quem fecha uma aba não disse qual olhar.
//
// Constante em vez de um `""` solto porque é o que faz essa decisão ser lida
// por quem passar aqui.
const defaultTab = ""

// Aqui moravam o `saveBoard` — a gravação do tabuleiro numa goroutine — e o
// `persistBoardAndWarn`, que acendia a tarja quando ela falhava.
//
// Eles eram o ÚLTIMO trabalho em segundo plano do app. O argumento era o tempo:
// "o mestre não espera o disco no meio do turno", e num prato girante o toque
// chegava a centenas de milissegundos. O que derrubou o argumento foi a medição
// da ALE-273: com `synchronous=NORMAL` o toque é de 1,7ms, e não há o que
// economizar adiando. O que se pagava por esses milissegundos era uma mesa que
// podia estar rodando de memória sem ninguém saber (ALE-375).
//
// Hoje quem grava é a MUTAÇÃO, dentro do gesto (`boards.Store.applyLocked`), e
// a publicação só publica.

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
// > Aqui morava um parágrafo sobre a GRAVAÇÃO acontecer sempre e a publicação
// > não, avisando que trocar as duas de lugar perderia em silêncio a cena de
// > quem não está na aba padrão. Ele deixou de valer: esta função não grava
// > mais nada (ALE-375), e o `return` abaixo é só do publicador porque não há
// > outro passo do qual separá-lo.
func (tr tableRules) publishBoardState(ctx context.Context, sessionID int64, state *board.BoardState) {
	defaultID, err := tr.boards.DefaultBoardID(ctx, sessionID)
	if err != nil {
		// A publicação é AVISO e não gesto: quem mutou já recebeu o resultado, e
		// não há a quem recusar aqui. Sem saber qual é a aba padrão não há como
		// decidir se este quadro é o dela, e mandá-lo a todos trocaria a cena na
		// tela de quem está noutra aba — o dano que o `return` abaixo existe para
		// evitar.
		log.Printf("session %d: não deu para saber a aba padrão para publicar (%v)", sessionID, err)
		return
	}
	if state != nil && state.ID != defaultID {
		return
	}
	// O tabuleiro já numera as próprias mutações, então a ordem sai de graça —
	// `Version` sobe a cada mutação aceita. Fechar o tabuleiro manda `nil` e cai
	// no caminho "sem ordem", que reinicia o destino de propósito.
	var order uint64
	if state != nil {
		order = uint64(state.Version)
	}
	tr.sse.EmitOrdered(sessionID, "gm", "board-state", order, state)
	tr.sse.EmitOrdered(sessionID, "player", "board-state", order, board.BoardForRole("player", state))
}

// publishWhatIsLeft é o quadro DEPOIS de fechar uma aba.
//
// Publicar `nil` seco seria a frase "esta sessão não tem tabuleiro", e com
// várias abas ela é MENTIRA: fechar a cripta com a taverna aberta apagaria a
// grade da tela de quem nem sabia que a cripta existia. Quem responde é o
// estado — sobrou aba, vai a PADRÃO; não sobrou, vai o `nil`, que aí é verdade.
//
// Ela só PUBLICA. Aqui ela também regravava a aba que sobrou, e isso deixou de
// ter sentido: o `Close` grava o que precisa gravar dentro do gesto, e a aba que
// sobrou não mudou (ALE-375).
func (tr tableRules) publishWhatIsLeft(ctx context.Context, sessionID int64) {
	left, err := tr.boards.Get(ctx, sessionID, defaultTab)
	if err != nil {
		log.Printf("session %d: não deu para ler o que sobrou para publicar (%v)", sessionID, err)
		return
	}
	tr.publishBoardState(ctx, sessionID, left)
}

// liveCtx é quem pediu, em que mesa, com que papel. Resolvido uma vez por
// requisição.
type liveCtx struct {
	UserID     int64
	campaignID int64
	sessionID  int64
	Role       string
}

// publishSessionState transmite o estado às duas salas por papel. Ela não grava
// porque não há o que gravar: quando o estado chega aqui, ele JÁ está no banco
// (ALE-371).
func (tr tableRules) publishSessionState(sessionID int64, state *live.SessionRuntimeState) {
	tr.sse.EmitOrdered(sessionID, "gm", "session-state", state.Seq, state)
	tr.sse.EmitOrdered(sessionID, "player", "session-state", state.Seq, live.RedactForPlayers(state))
}
