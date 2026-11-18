package table

import (
	"fmt"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/board"
)

// OS MARCADORES na Mesa em Datastar — o ponto apontado no mapa, ver
// GLOSSARY.md.
//
// O verbo que importa é REVELAR: o marcador nasce ESCONDIDO, porque marcar a
// armadilha na frente da mesa entrega a armadilha. Por isso criar e revelar são
// dois gestos e não um.

func (s Scene) MarkerRoutes(r chi.Router) {
	base := sessionPattern + "/tabuleiro/marcadores"
	// O `novo` estático antes das coordenadas separa a criação dos gestos sobre
	// um marcador que já existe — sem ele, `{x}` e `{id}` disputariam a mesma
	// posição do caminho.
	r.Post(base+"/novo", s.gmBoardCommand(marksTheSquare))
	r.Post(base+"/{id}/revelar", s.gmBoardCommand(revealMarker))
	r.Post(base+"/{id}/cor/{cor}", s.gmBoardCommand(paintMarker))
	r.Post(base+"/{id}/remover", s.gmBoardCommand(eraseMarker))
}

// marksTheSquare põe um marcador novo na casa clicada.
//
// A LETRA vem do motor e não da tela, e é decidida DENTRO da mutação (`board.AddNextMarker`): duas
// telas escolhendo letra por conta própria, ou dois cliques decidindo sobre a mesma leitura, é como
// nasce o segundo "C" no mesmo mapa.
func marksTheSquare(st Scene, c commandCtx) (*board.BoardState, error) {
	square, err := squareOnly(c.R)
	if err != nil {
		return nil, err
	}
	return st.deps.Boards().AddNextMarker(c.R.Context(), c.SessionID, c.BoardID, square)
}

// revealMarker alterna entre mostrar e esconder.
//
// ALTERNA e não "revela", apesar do nome do gesto: o mestre que revelou cedo
// demais precisa poder esconder de volta, e um segundo botão para desfazer o
// primeiro seria a mesma decisão em dois lugares.
func revealMarker(st Scene, c commandCtx) (*board.BoardState, error) {
	return st.deps.Boards().ToggleMarkerHidden(c.R.Context(), c.SessionID, c.BoardID, chi.URLParam(c.R, "id"))
}

// paintMarker troca a cor.
func paintMarker(st Scene, c commandCtx) (*board.BoardState, error) {
	color := chi.URLParam(c.R, "cor")
	// A RECUSA é aqui e explícita, apesar de o `UpdateMarker` ignorar cor
	// desconhecida: ignorar em silêncio é um clique que não faz nada e não diz
	// nada, que o mestre lê como tela travada.
	if !board.KnownMarkerColor(color) {
		return nil, fmt.Errorf("a cor %q não existe; as do mapa são %s", color, coresEmPortugues())
	}
	return st.deps.Boards().UpdateMarker(c.R.Context(), c.SessionID, c.BoardID, chi.URLParam(c.R, "id"),
		board.NewMarkerColor(color))
}

// eraseMarker tira o ponto do mapa.
func eraseMarker(st Scene, c commandCtx) (*board.BoardState, error) {
	return st.deps.Boards().EraseMarker(c.R.Context(), c.SessionID, c.BoardID, chi.URLParam(c.R, "id"))
}

// coresEmPortugues lista as cores para a frase da recusa — a mensagem tem de
// dizer o que era esperado, não só o que veio errado.
func coresEmPortugues() string {
	names := ""
	for i, c := range board.MarkerColors {
		if i > 0 {
			names += ", "
		}
		names += c.Label
	}
	return names
}
