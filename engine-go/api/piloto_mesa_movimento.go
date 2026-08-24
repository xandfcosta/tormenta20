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
// impede o estouro do deslocamento não é um aviso depois do erro — são as casas
// oferecidas, que já vêm limitadas pelo que sobrou. Não dá para clicar no que o
// servidor recusaria.
//
// Nada de lista de paradas guardada: o CAMINHO proposto já é o acumulado, e a
// última parada é o último quadrado dele.

func (s *Server) rotasDoMovimento(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/tabuleiro/{tokenId}"
	r.Post(base+"/parada/{x}/{y}", s.comandoDaMesa(paraNoQuadrado))
	r.Post(base+"/confirmar", s.comandoDaMesa(confirmaOMovimento))
	r.Post(base+"/cancelar", s.comandoDaMesa(cancelaOMovimento))
}

// paraNoQuadrado acrescenta uma parada ao movimento — ou começa um.
//
// Estender é somar o segmento novo ao caminho que já existe, descartando o
// primeiro quadrado dele porque é o último do anterior — a mesma emenda do
// `CaminhoPorParadas`, que é quem a define.
func paraNoQuadrado(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	destino, err := quadradoDaURL(c.R)
	if err != nil {
		return nil, err
	}
	tokenID := chi.URLParam(c.R, "tokenId")
	b := st.boards.Get(c.R.Context(), c.SessionID)
	if b == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto nesta mesa")
	}
	peca := tabuleiro.FindToken(b, tokenID)
	if peca == nil {
		return nil, fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}

	caminho := []engine.Square{{X: peca.X, Y: peca.Y}}
	if p := b.Pending; p != nil && p.TokenID == tokenID && p.ByUserID == c.User.ID {
		caminho = p.Path
	}
	caminho = append(caminho, engine.CaminhoEntre(caminho[len(caminho)-1], destino)[1:]...)

	quem := st.quemMove(c)
	return st.boards.ProposeMove(c.R.Context(), c.SessionID,
		st.sessions.GetState(c.SessionID), tokenID, caminho, quem, 0)
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
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, sessionID, ok := mesaParams(w, r)
		if !ok {
			return
		}
		user := currentUser(r)
		if _, _, status, err := s.sessionForCaller(r.Context(), user, campaignID, sessionID); err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		sinais := map[string]any{}
		estado, err := mutar(s, mesaComando{
			R: r, User: user, CampaignID: campaignID, SessionID: sessionID, Sinais: sinais,
		})
		if estado != nil {
			s.publishBoardState(sessionID, estado)
		}
		s.respondeAoMestre(w, r, user, campaignID, sessionID, err, sinais)
	}
}

// intDoCaminho aceita o sinal de menos, porque o plano não tem bordas.
func intDoCaminho(bruto string) (int, error) {
	var n int
	_, err := fmt.Sscanf(bruto, "%d", &n)
	return n, err
}
