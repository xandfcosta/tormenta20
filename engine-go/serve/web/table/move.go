package table

import (
	"fmt"
	"net/http"
	"t20engine/app"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// O MOVIMENTO da peça na Mesa.
//
// Uma parada por vez: a pessoa clica numa casa alcançável e a peça propõe ir até
// lá; clica de novo e o caminho ESTENDE, contornando o que ela quiser. O que
// impede o estouro do deslocamento é o ALCANCE desenhado, que já vem limitado
// pelo que sobrou e encolhe a cada parada.
//
// Mas "não dá para clicar no que o servidor recusaria" deixa de valer com o
// ARRASTO: clique cai numa casa oferecida, soltura cai onde o dedo estiver,
// inclusive fora do alcance. Por isso a recusa também PRECISA falar — e fala em
// `move_error`, no tabuleiro. O alcance continua sendo a realimentação
// principal; a frase é a rede embaixo dela.
//
// A LISTA DE PARADAS é guardada no `PendingMove.Stops`, e o caminho é o que elas
// produzem. Deduzir a última parada do fim do caminho basta para o Cancelar e
// NÃO basta para desfazer UMA: um trecho legítimo já tem uma dobra (a diagonal
// vem primeiro), e ela é indistinguível da dobra de uma parada.

func (s Scene) MoveRoutes(r chi.Router) {
	base := sessionPattern + "/tabuleiro/{tokenId}"
	r.Post(base+"/parada", s.tableCommand(paraNoQuadrado))
	r.Post(base+"/desfazer-parada", s.tableCommand(undoLastStop))
	r.Post(base+"/confirmar", s.tableCommand(confirmMove))
	r.Post(base+"/cancelar", s.tableCommand(cancelMove))
}

// paraNoQuadrado acrescenta uma parada ao movimento — ou começa um.
func paraNoQuadrado(st Scene, c commandCtx) (*board.BoardState, error) {
	destination, err := squareOnly(c.R)
	if err != nil {
		return nil, err
	}
	tokenID := chi.URLParam(c.R, "tokenId")
	stops, err := st.paradasDaProposta(c, tokenID)
	if err != nil {
		return nil, err
	}
	return st.propoePorParadas(c, tokenID, append(stops, destination))
}

// undoLastStop corrige a última perna sem jogar a rota inteira fora.
//
// É a ordem do arrependimento: primeiro se tira a perna errada, e só depois se
// cancela tudo. Sobrando só a origem, desfazer VIRA cancelar — uma proposta sem
// perna nenhuma não é proposta, e deixar um provisório de custo zero na mesa
// seria pedir um "Confirmar" que não move ninguém.
//
// Reconstrói pelas paradas que sobraram em vez de cortar o fim do caminho: o
// número de quadrados de um trecho não se deduz das paradas sem redesenhá-lo, e
// redesenhar é o que o `PathThroughStops` faz de graça.
func undoLastStop(st Scene, c commandCtx) (*board.BoardState, error) {
	tokenID := chi.URLParam(c.R, "tokenId")
	stops, err := st.paradasDaProposta(c, tokenID)
	if err != nil {
		return nil, err
	}
	if len(stops) < 2 {
		return nil, fmt.Errorf("não há parada a desfazer em %q", tokenID)
	}
	if stops = stops[:len(stops)-1]; len(stops) < 2 {
		return cancelMove(st, c)
	}
	return st.propoePorParadas(c, tokenID, stops)
}

// paradasDaProposta devolve as paradas já acumuladas, ou só o lugar da peça.
//
// A proposta de OUTRA pessoa não conta: duas mãos empilhando pernas no mesmo
// movimento é o estado que o `ByUserID` existe para evitar, e sem esta conferência
// um segundo jogador estenderia o caminho que o primeiro está montando.
func (s Scene) paradasDaProposta(c commandCtx, tokenID string) ([]engine.Square, error) {
	b, err := s.deps.Boards().Get(c.R.Context(), c.SessionID, c.BoardID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto nesta mesa")
	}
	token := board.FindToken(b, tokenID)
	if token == nil {
		return nil, fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	if p := b.Pending; p != nil && p.TokenID == tokenID && p.ByUserID == c.User && len(p.Stops) > 0 {
		// Cópia, e não a fatia do estado: o `append` do chamador escreveria na
		// memória do tabuleiro vivo antes de a proposta ser validada — e uma
		// proposta RECUSADA teria deixado a parada lá.
		return append([]engine.Square(nil), p.Stops...), nil
	}
	return []engine.Square{{X: token.X, Y: token.Y}}, nil
}

func (s Scene) propoePorParadas(c commandCtx, tokenID string, stops []engine.Square) (*board.BoardState, error) {
	state, err := s.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	return s.deps.Boards().ProposeMoveWithStops(c.R.Context(), c.SessionID, c.BoardID,
		state, tokenID, stops, s.moveWho(c), 0)
}

// confirmMove entrega o gesto ao caso de uso, que abre a transação.
//
// O corpo dele morava aqui e foi para o `app/boards` na ALE-376: eram duas
// gravações em transações diferentes — a peça pousava e só então a ação era
// cobrada —, e uma falha na cobrança deixava o movimento feito e o turno
// intacto. A cena não abre transação; quem desenha o contorno de um gesto é o
// caso de uso.
//
// O `moveWho` fica: ele resolve a POSSE lendo o papel e o id da peça DO PEDIDO,
// e isso é leitura de requisição — o caso de uso recebe a resposta pronta.
func confirmMove(st Scene, c commandCtx) (*board.BoardState, error) {
	return st.gestures.ConfirmMove(c.R.Context(), c.SessionID, c.BoardID, st.moveWho(c))
}

func cancelMove(st Scene, c commandCtx) (*board.BoardState, error) {
	return st.deps.Boards().CancelMove(c.R.Context(), c.SessionID, c.BoardID, st.moveWho(c))
}

// moveWho resolve a POSSE contra o banco, e nunca contra o cliente.
//
// O `Mover.OwnsCharacter` é o que separa "a peça é sua" de "você disse que é": a
// peça aponta para um personagem, e quem responde de quem ele é são as fichas da
// campanha — o mesmo caminho que o `tableRoster` usa para saber quais são os
// MEUS.
func (s Scene) moveWho(c commandCtx) board.Mover {
	_, role, err := s.access.Session(c.R.Context(), app.Caller{ID: c.User}, c.CampaignID, c.SessionID)
	if err != nil {
		role = "player"
	}
	who := board.Mover{UserID: c.User, Role: role}
	if role == "gm" {
		return who
	}
	_, mine, _ := s.tableRoster(c.R.Context(), c.User, c.CampaignID)
	// A LEITURA QUE FALHA DEIXA A POSSE EM FALSO, e este é o lado seguro: sem
	// saber de quem é a peça, a resposta é "não é sua", e o `assertMovable` do
	// tabuleiro recusa com a frase dele. O erro não some da mesa — o gesto que
	// vem a seguir lê o mesmo tabuleiro pela porta que DEVOLVE erro, e é ele
	// quem conta o que houve.
	b, err := s.deps.Boards().Get(c.R.Context(), c.SessionID, c.BoardID)
	if err != nil {
		return who
	}
	if token := board.FindToken(b, chi.URLParam(c.R, "tokenId")); token != nil && token.CharacterID != nil {
		who.OwnsCharacter = mine[*token.CharacterID]
	}
	return who
}

// tableCommand é o irmão do `gmCommand` para o que o JOGADOR também faz.
//
// Mover não é do mestre: o jogador anda com a própria peça, e quem decide isso é
// o `assertMovable` do `tabuleiro` — três regras que já existem e que este
// caminho não pode reescrever. Por isso ele NÃO exige papel: a recusa vem da
// regra, com a frase que ela escreve ("não é a vez de Arwen"), e não de um 403
// que diria a coisa errada.
func (s Scene) tableCommand(
	mutate func(Scene, commandCtx) (*board.BoardState, error),
) http.HandlerFunc {
	return s.boardCommand(mutate, false)
}

// gmBoardCommand é abrir e encerrar a cena: mutação de TABULEIRO,
// mas só o mestre monta e desmonta a mesa.
//
// As duas diferenças andam JUNTAS e por isso são um parâmetro só. Quem pode agir
// decide onde a recusa fala: comando que só o mestre emite fala no RODAPÉ dele,
// que é a superfície que ele tem na tela; comando que o jogador também emite
// fala no TABULEIRO, porque jogador não renderiza rodapé nenhum — foi assim que
// uma recusa de movimento ficou muda por meia sessão.
func (s Scene) gmBoardCommand(
	mutate func(Scene, commandCtx) (*board.BoardState, error),
) http.HandlerFunc {
	return s.boardCommand(mutate, true)
}

// gmContinuousCommand é o irmão do de cima para o gesto que se REPETE enquanto
// o dedo está no botão: pintar e apagar terreno arrastando.
//
// A diferença é o TAMANHO DA RESPOSTA, e ela é medida: o `respondGm` repinta
// TODAS as regiões da Mesa, e um clique de pintura devolvia **353 KB**. No gesto
// avulso isso é seguro — ninguém está no meio de um arrasto no instante em que
// pediu outra coisa —, e a frase deixa de valer exatamente aqui: no gesto
// contínuo a pessoa ESTÁ no meio de um arrasto, e um traço de vinte casas
// custaria sete megabytes.
//
// A resposta fica no `mesa-tabuleiro` porque terreno não aparece em mais lugar
// nenhum. Quem precisar de outra região não usa este atalho — é uma lista
// explícita, não um padrão.
func (s Scene) gmContinuousCommand(
	mutate func(Scene, commandCtx) (*board.BoardState, error),
) http.HandlerFunc {
	return s.boardCommand(mutate, true, "table-board")
}

// boardCommand é o corpo dos dois. Separá-los em duas cópias seria repetir
// resolver a mesa, mutar, publicar e redesenhar — e é numa delas que alguém
// esquece de publicar e a mesa fica vendo a cena velha.
func (s Scene) boardCommand(
	mutate func(Scene, commandCtx) (*board.BoardState, error),
	gmOnly bool,
	onlyRegions ...string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, sessionID, ok := tableParams(w, r)
		if !ok {
			return
		}
		userID := s.deps.CurrentUserID(r)
		_, role, err := s.access.Session(r.Context(), app.Caller{ID: userID}, campaignID, sessionID)
		status := statusOf(err)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		// A trava é aqui e não na tela: quem postar na mão leva 403, e o botão
		// escondido é só cortesia para quem não pode.
		if gmOnly && role != "gm" {
			http.Error(w, "só o mestre monta a cena", http.StatusForbidden)
			return
		}
		signals := map[string]any{}
		aba, err := s.chosenTabOf(r.Context(), sessionID, userID)
		if err != nil {
			s.respondGm(w, r, userID, campaignID, sessionID, err, signals, onlyRegions...)
			return
		}
		state, err := mutate(s, commandCtx{
			R: r, User: userID, CampaignID: campaignID, SessionID: sessionID,
			// A ABA de quem clicou, resolvida aqui e uma vez só: é ela que diz em
			// QUAL tabuleiro o gesto acontece. Resolver dentro de cada mutação seria
			// a mesma pergunta escrita vinte vezes, e a vigésima primeira é a que
			// esquece.
			BoardID: aba,
			Signals: signals,
		})
		if state != nil {
			s.deps.PublishBoardState(r.Context(), sessionID, state)
		}
		if gmOnly {
			// O `respondGm` escreve o `command_error` do rodapé sozinho.
			s.respondGm(w, r, userID, campaignID, sessionID, err, signals, onlyRegions...)
			return
		}
		// A recusa vai para `move_error` e NÃO para o `command_error` do
		// rodapé, que é do mestre: quem move é o jogador, e ele não tem rodapé
		// nenhum — a frase cairia num elemento que a tela dele nem renderiza.
		// Escrita nos DOIS caminhos pelo mesmo motivo do outro sinal: só acender
		// deixa a recusa de duas paradas atrás acesa sobre uma que deu certo.
		sentence := ""
		if err != nil {
			sentence = err.Error()
		}
		signals["move_error"] = sentence
		s.respondGm(w, r, userID, campaignID, sessionID, nil, signals)
	}
}

// intDoCaminho aceita o sinal de menos, porque o plano não tem bordas.
func intDoCaminho(raw string) (int, error) {
	var n int
	_, err := fmt.Sscanf(raw, "%d", &n)
	return n, err
}
