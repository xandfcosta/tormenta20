package api

import (
	"github.com/go-chi/chi/v5"

	"t20engine/tabuleiro"
)

// A CORTINA na Mesa em Datastar (ALE-269, superfície 1) — ver GLOSSARIO.md.
//
// O tabuleiro EXISTE para o mestre e a mesa vê uma cortina no lugar dele
// (ALE-202): montar a taverna enquanto eles olham a cripta. O servidor já sabia
// fazer isso inteiro — `SetCurtain` no store, `BoardForRole` esvaziando a cena
// para quem não é mestre, e o piloto até DESENHAVA o aviso. O que não existia
// era o gesto: a feature estava no ar e invisível, sem um botão que a ligasse.
//
// Encontrada no levantamento que precede a virada da sessão, cruzando os rótulos
// da SPA com o piloto. É a razão de aquele levantamento existir.

func (s *Server) rotasDaCortina(r chi.Router) {
	// O ESTADO no caminho, e não um alternar: ver o comentário do `correACortina`.
	r.Post("/mesa/{campaignId}/{sessionId}/tabuleiro/cortina/{estado}",
		s.comandoDoMestreNoTabuleiro(correACortina))
}

// correACortina fecha ou abre, conforme o pedido do caminho.
//
// O DESTINO vem na URL e não é um alternar cego, e a diferença aparece nos dois
// caminhos que a tela oferece: o botão do cabeçalho alterna, mas a tira de aviso
// só ABRE. Um alternar cego faria a tira fechar a cortina de novo se ela
// chegasse a ser desenhada com a cortina já aberta — e ela é justamente o que o
// mestre clica com pressa, no meio da cena.
func correACortina(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	fechada := chi.URLParam(c.R, "estado") == "fechar"
	board, mudou, err := st.boards.SetCurtain(c.R.Context(), c.SessionID, c.TabuleiroID, fechada)
	if err != nil {
		return board, err
	}
	// SEM MUDANÇA NÃO SE PUBLICA, e devolver nil é como se diz isso aqui: quem
	// transmite é o `comandoDoTabuleiro`, e ele só o faz com estado em mãos. O
	// mestre continua recebendo o redesenho dele — o `respondeAoMestre` roda de
	// qualquer jeito, porque a resposta É a confirmação do gesto.
	//
	// É a mesma escolha do `handleBoardCurtain` da SPA. Publicar um quadro que
	// não mudou custa um `BoardForRole` e um remendo em cada tela da mesa para
	// dizer exatamente o que elas já sabiam.
	if !mudou {
		return nil, nil
	}
	return board, nil
}
