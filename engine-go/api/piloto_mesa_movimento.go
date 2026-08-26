package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"t20engine/engine"
	"t20engine/tabuleiro"
)

// O MOVIMENTO da peça na Mesa em Datastar (ALE-266).
//
// Uma parada por vez: a pessoa clica numa casa alcançável e a peça propõe ir até
// lá; clica de novo e o caminho ESTENDE, contornando o que ela quiser. O que
// impede o estouro do deslocamento é o ALCANCE desenhado, que já vem limitado
// pelo que sobrou e encolhe a cada parada — medido: 84 casas com o deslocamento
// inteiro, 12 depois de gastar 4 de 6.
//
// Mas "não dá para clicar no que o servidor recusaria" deixou de ser verdade
// quando o ARRASTO entrou: clique cai numa casa oferecida, soltura cai onde o
// dedo estiver, inclusive fora do alcance. Por isso a recusa também PRECISA
// falar — e fala em `erroDoMovimento`, no tabuleiro. O alcance continua sendo a
// realimentação principal; a frase é a rede embaixo dela.
//
// A LISTA DE PARADAS é guardada desde a ALE-269 (item 10), e a linha que estava
// aqui dizia o contrário — "o CAMINHO proposto já é o acumulado, e a última
// parada é o último quadrado dele". A primeira metade continua verdadeira; a
// segunda basta para o Cancelar e não basta para DESFAZER UMA: um trecho legítimo
// já tem uma dobra (a diagonal vem primeiro), e ela é indistinguível da dobra de
// uma parada. Quem guarda é o `PendingMove.Stops`, e o caminho passou a ser o que
// as paradas produzem em vez de o que se emenda à mão.

func (s *Server) rotasDoMovimento(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/tabuleiro/{tokenId}"
	r.Post(base+"/parada/{x}/{y}", s.comandoDaMesa(paraNoQuadrado))
	r.Post(base+"/desfazer-parada", s.comandoDaMesa(desfazAUltimaParada))
	r.Post(base+"/confirmar", s.comandoDaMesa(confirmaOMovimento))
	r.Post(base+"/cancelar", s.comandoDaMesa(cancelaOMovimento))
}

// paraNoQuadrado acrescenta uma parada ao movimento — ou começa um.
func paraNoQuadrado(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	destino, err := quadradoDaURL(c.R)
	if err != nil {
		return nil, err
	}
	tokenID := chi.URLParam(c.R, "tokenId")
	paradas, err := st.paradasDaProposta(c, tokenID)
	if err != nil {
		return nil, err
	}
	return st.propoePorParadas(c, tokenID, append(paradas, destino))
}

// desfazAUltimaParada corrige a última perna sem jogar a rota inteira fora
// (ALE-266, portado na ALE-269).
//
// É a ordem do arrependimento: primeiro se tira a perna errada, e só depois se
// cancela tudo. Sobrando só a origem, desfazer VIRA cancelar — uma proposta sem
// perna nenhuma não é proposta, e deixar um provisório de custo zero na mesa
// seria pedir um "Confirmar" que não move ninguém.
//
// Reconstrói pelas paradas que sobraram em vez de cortar o fim do caminho: o
// número de quadrados de um trecho não se deduz das paradas sem redesenhá-lo, e
// redesenhar é o que o `CaminhoPorParadas` faz de graça.
func desfazAUltimaParada(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	tokenID := chi.URLParam(c.R, "tokenId")
	paradas, err := st.paradasDaProposta(c, tokenID)
	if err != nil {
		return nil, err
	}
	if len(paradas) < 2 {
		return nil, fmt.Errorf("não há parada a desfazer em %q", tokenID)
	}
	if paradas = paradas[:len(paradas)-1]; len(paradas) < 2 {
		return cancelaOMovimento(st, c)
	}
	return st.propoePorParadas(c, tokenID, paradas)
}

// paradasDaProposta devolve as paradas já acumuladas, ou só o lugar da peça.
//
// A proposta de OUTRA pessoa não conta: duas mãos empilhando pernas no mesmo
// movimento é o estado que o `ByUserID` existe para evitar, e sem esta conferência
// um segundo jogador estenderia o caminho que o primeiro está montando.
func (s *Server) paradasDaProposta(c mesaComando, tokenID string) ([]engine.Square, error) {
	b := s.boards.Get(c.R.Context(), c.SessionID)
	if b == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto nesta mesa")
	}
	peca := tabuleiro.FindToken(b, tokenID)
	if peca == nil {
		return nil, fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	if p := b.Pending; p != nil && p.TokenID == tokenID && p.ByUserID == c.User.ID && len(p.Stops) > 0 {
		// Cópia, e não a fatia do estado: o `append` do chamador escreveria na
		// memória do tabuleiro vivo antes de a proposta ser validada — e uma
		// proposta RECUSADA teria deixado a parada lá.
		return append([]engine.Square(nil), p.Stops...), nil
	}
	return []engine.Square{{X: peca.X, Y: peca.Y}}, nil
}

func (s *Server) propoePorParadas(c mesaComando, tokenID string, paradas []engine.Square) (*tabuleiro.BoardState, error) {
	return s.boards.ProposeMoveComParadas(c.R.Context(), c.SessionID,
		s.sessions.GetState(c.SessionID), tokenID, paradas, s.quemMove(c), 0)
}

func confirmaOMovimento(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	// Versão ZERO: o `CommitMove` só compara quando ela é positiva, e aqui quem
	// confirma acabou de ver a cena que o servidor desenhou — não há uma versão
	// vinda do cliente para conferir contra. A trava contra a mesa ter mudado
	// continua sendo a REVALIDAÇÃO da vez, que o `CommitMove` faz de novo.
	return st.boards.CommitMove(c.R.Context(), c.SessionID,
		st.sessions.GetState(c.SessionID), 0, st.quemMove(c))
}

func cancelaOMovimento(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	return st.boards.CancelMove(c.R.Context(), c.SessionID, st.quemMove(c))
}

// quadradoDaURL lê o destino do CAMINHO e não de um sinal.
//
// Coordenada NEGATIVA é lugar legítimo — o plano não tem bordas —, então o
// caminho carrega o número com sinal e o `strconv` o aceita. Vir pela URL é o
// mesmo argumento dos verbos da linha: o valor é do botão que foi clicado, e não
// de um sinal da página inteira que nove botões disputariam.
func quadradoDaURL(r *http.Request) (engine.Square, error) {
	x, errX := intDoCaminho(chi.URLParam(r, "x"))
	y, errY := intDoCaminho(chi.URLParam(r, "y"))
	if errX != nil || errY != nil {
		return engine.Square{}, fmt.Errorf("quadrado (%q,%q) não é um par de números",
			chi.URLParam(r, "x"), chi.URLParam(r, "y"))
	}
	return engine.Square{X: x, Y: y}, nil
}

// quemMove resolve a POSSE contra o banco, e nunca contra o cliente.
//
// O `Mover.OwnsCharacter` é o que separa "a peça é sua" de "você disse que é": a
// peça aponta para um personagem, e quem responde de quem ele é são as fichas da
// campanha — o mesmo caminho que o `mesaRoster` usa para saber quais são os MEUS
// (ALE-33).
func (s *Server) quemMove(c mesaComando) tabuleiro.Mover {
	_, papel, _, err := s.sessionForCaller(c.R.Context(), c.User, c.CampaignID, c.SessionID)
	if err != nil {
		papel = "player"
	}
	quem := tabuleiro.Mover{UserID: c.User.ID, Role: papel}
	if papel == "gm" {
		return quem
	}
	_, meus, _ := s.mesaRoster(c.R.Context(), c.User, c.CampaignID)
	b := s.boards.Get(c.R.Context(), c.SessionID)
	if peca := tabuleiro.FindToken(b, chi.URLParam(c.R, "tokenId")); peca != nil && peca.CharacterID != nil {
		quem.OwnsCharacter = meus[*peca.CharacterID]
	}
	return quem
}

// comandoDaMesa é o irmão do `comandoDoMestre` para o que o JOGADOR também faz.
//
// Mover não é do mestre: o jogador anda com a própria peça, e quem decide isso é
// o `assertMovable` do `tabuleiro` — três regras que já existem e que este
// caminho não pode reescrever. Por isso ele NÃO exige papel: a recusa vem da
// regra, com a frase que ela escreve ("não é a vez de Arwen"), e não de um 403
// que diria a coisa errada.
func (s *Server) comandoDaMesa(
	mutar func(*Server, mesaComando) (*tabuleiro.BoardState, error),
) http.HandlerFunc {
	return s.comandoDoTabuleiro(mutar, false)
}

// comandoDoMestreNoTabuleiro é abrir e encerrar a cena: mutação de TABULEIRO,
// mas só o mestre monta e desmonta a mesa.
//
// As duas diferenças andam JUNTAS e por isso são um parâmetro só. Quem pode agir
// decide onde a recusa fala: comando que só o mestre emite fala no RODAPÉ dele,
// que é a superfície que ele tem na tela; comando que o jogador também emite
// fala no TABULEIRO, porque jogador não renderiza rodapé nenhum — foi assim que
// uma recusa de movimento ficou muda por meia sessão.
func (s *Server) comandoDoMestreNoTabuleiro(
	mutar func(*Server, mesaComando) (*tabuleiro.BoardState, error),
) http.HandlerFunc {
	return s.comandoDoTabuleiro(mutar, true)
}

// comandoDoTabuleiro é o corpo dos dois. Separá-los em duas cópias seria repetir
// resolver a mesa, mutar, publicar e redesenhar — e é numa delas que alguém
// esquece de publicar e a mesa fica vendo a cena velha.
func (s *Server) comandoDoTabuleiro(
	mutar func(*Server, mesaComando) (*tabuleiro.BoardState, error),
	soODoMestre bool,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, sessionID, ok := mesaParams(w, r)
		if !ok {
			return
		}
		user := currentUser(r)
		_, papel, status, err := s.sessionForCaller(r.Context(), user, campaignID, sessionID)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		// A trava é aqui e não na tela: quem postar na mão leva 403, e o botão
		// escondido é só cortesia para quem não pode.
		if soODoMestre && papel != "gm" {
			http.Error(w, "só o mestre monta a cena", http.StatusForbidden)
			return
		}
		sinais := map[string]any{}
		estado, err := mutar(s, mesaComando{
			R: r, User: user, CampaignID: campaignID, SessionID: sessionID, Sinais: sinais,
		})
		if estado != nil {
			s.publishBoardState(sessionID, estado)
		}
		if soODoMestre {
			// O `respondeAoMestre` escreve o `erroDoComando` do rodapé sozinho.
			s.respondeAoMestre(w, r, user, campaignID, sessionID, err, sinais)
			return
		}
		// A recusa vai para `erroDoMovimento` e NÃO para o `erroDoComando` do
		// rodapé, que é do mestre: quem move é o jogador, e ele não tem rodapé
		// nenhum — a frase cairia num elemento que a tela dele nem renderiza.
		// Escrita nos DOIS caminhos pelo mesmo motivo do outro sinal: só acender
		// deixa a recusa de duas paradas atrás acesa sobre uma que deu certo.
		frase := ""
		if err != nil {
			frase = err.Error()
		}
		sinais["erroDoMovimento"] = frase
		s.respondeAoMestre(w, r, user, campaignID, sessionID, nil, sinais)
	}
}

// intDoCaminho aceita o sinal de menos, porque o plano não tem bordas.
func intDoCaminho(bruto string) (int, error) {
	var n int
	_, err := fmt.Sscanf(bruto, "%d", &n)
	return n, err
}
