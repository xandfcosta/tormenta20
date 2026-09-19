package table

import (
	"github.com/go-chi/chi/v5"

	"t20engine/domain/board"
)

// A CORTINA na Mesa — ver GLOSSARY.md.
//
// O tabuleiro EXISTE para o mestre e a mesa vê uma cortina no lugar dele: montar
// a taverna enquanto eles olham a cripta. Quem esvazia a cena para quem não é
// mestre é o `BoardForRole`; o que mora aqui é o gesto que a liga.

func (s Scene) CurtainRoutes(r chi.Router) {
	// O ESTADO no caminho, e não um alternar: ver o comentário do `runsCurtain`.
	r.Post(sessionPattern+"/tabuleiro/cortina/{estado}",
		s.gmBoardCommand(runsCurtain))
}

// runsCurtain fecha ou abre, conforme o pedido do caminho.
//
// O DESTINO vem na URL e não é um alternar cego, e a diferença aparece nos dois
// caminhos que a tela oferece: o botão do cabeçalho alterna, mas a tira de aviso
// só ABRE. Um alternar cego faria a tira fechar a cortina de novo se ela
// chegasse a ser desenhada com a cortina já aberta — e ela é justamente o que o
// mestre clica com pressa, no meio da cena.
func runsCurtain(st Scene, c commandCtx) (*board.BoardState, error) {
	fechada := chi.URLParam(c.R, "estado") == "fechar"
	board, mudou, err := st.deps.Boards().SetCurtain(c.R.Context(), c.SessionID, c.TabuleiroID, fechada)
	if err != nil {
		return board, err
	}
	// SEM MUDANÇA NÃO SE PUBLICA, e devolver nil é como se diz isso aqui: quem
	// transmite é o `boardCommand`, e ele só o faz com estado em mãos. O
	// mestre continua recebendo o redesenho dele — o `respondGm` roda de
	// qualquer jeito, porque a resposta É a confirmação do gesto.
	//
	// Publicar um quadro que não mudou custa um `BoardForRole` e um remendo em
	// cada tela da mesa para dizer exatamente o que elas já sabiam.
	if !mudou {
		return nil, nil
	}
	return board, nil
}
