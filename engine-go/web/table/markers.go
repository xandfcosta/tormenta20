package table

import (
	"errors"
	"fmt"

	"github.com/go-chi/chi/v5"

	"t20engine/tabuleiro"
)

// OS MARCADORES na Mesa em Datastar (ALE-264, item 5) — o ponto apontado no
// mapa, ver GLOSSARIO.md.
//
// O tabuleiro já DESENHAVA marcadores desde o `33380d6`; o que não existia era
// gesto para criar, revelar ou apagar um. As rotas JSON (`handleBoardMarker*`)
// respondiam só à SPA.
//
// O verbo que importa é REVELAR (ALE-195): o marcador nasce ESCONDIDO, porque
// marcar a armadilha na frente da mesa entrega a armadilha. Por isso criar e
// revelar são dois gestos e não um.

func (s Scene) MarkerRoutes(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/tabuleiro/marcadores"
	// O `novo` estático antes das coordenadas separa a criação dos gestos sobre
	// um marcador que já existe — sem ele, `{x}` e `{id}` disputariam a mesma
	// posição do caminho.
	r.Post(base+"/novo/{x}/{y}", s.gmBoardCommand(marcaOLugar))
	r.Post(base+"/{id}/revelar", s.gmBoardCommand(revealMarker))
	r.Post(base+"/{id}/cor/{cor}", s.gmBoardCommand(paintMarker))
	r.Post(base+"/{id}/remover", s.gmBoardCommand(eraseMarker))
}

// marcaOLugar põe um marcador novo na casa clicada.
//
// A LETRA vem do motor (`ProximaLetraDeMarcador`) e não da tela: na SPA era o
// cliente que escolhia "A", "B", "C" e mandava pronto, e duas telas escolhendo
// letra por conta própria é como nasce o segundo "C" no mesmo mapa.
func marcaOLugar(st Scene, c commandCtx) (*tabuleiro.BoardState, error) {
	casa, err := quadradoDaURL(c.R)
	if err != nil {
		return nil, err
	}
	b := st.deps.Boards().Get(c.R.Context(), c.SessionID, c.TabuleiroID)
	if b == nil {
		return nil, errors.New("não há tabuleiro aberto para marcar")
	}
	return st.deps.Boards().AddMarker(c.R.Context(), c.SessionID, c.TabuleiroID, tabuleiro.BoardMarker{
		X: casa.X, Y: casa.Y,
		Text:  tabuleiro.NextMarkerLetter(b.Markers),
		Color: tabuleiro.DefaultMarkerColor(),
		// ESCONDIDO ao nascer, e é a razão de o marcador existir.
		Hidden: true,
	})
}

// revealMarker alterna entre mostrar e esconder.
//
// ALTERNA e não "revela", apesar do nome do gesto: o mestre que revelou cedo
// demais precisa poder esconder de volta, e um segundo botão para desfazer o
// primeiro seria a mesma decisão em dois lugares.
func revealMarker(st Scene, c commandCtx) (*tabuleiro.BoardState, error) {
	marcador, err := urlMarker(st, c)
	if err != nil {
		return nil, err
	}
	return st.deps.Boards().UpdateMarker(c.R.Context(), c.SessionID, c.TabuleiroID, marcador.ID,
		tabuleiro.MarkerReveal(!marcador.Hidden))
}

// paintMarker troca a cor.
func paintMarker(st Scene, c commandCtx) (*tabuleiro.BoardState, error) {
	marcador, err := urlMarker(st, c)
	if err != nil {
		return nil, err
	}
	cor := chi.URLParam(c.R, "cor")
	// A RECUSA é aqui e explícita, apesar de o `UpdateMarker` ignorar cor
	// desconhecida: ignorar em silêncio é um clique que não faz nada e não diz
	// nada, que o mestre lê como tela travada.
	if !tabuleiro.KnownMarkerColor(cor) {
		return nil, fmt.Errorf("a cor %q não existe; as do mapa são %s", cor, coresEmPortugues())
	}
	return st.deps.Boards().UpdateMarker(c.R.Context(), c.SessionID, c.TabuleiroID, marcador.ID,
		tabuleiro.NewMarkerColor(cor))
}

// eraseMarker tira o ponto do mapa.
func eraseMarker(st Scene, c commandCtx) (*tabuleiro.BoardState, error) {
	marcador, err := urlMarker(st, c)
	if err != nil {
		return nil, err
	}
	return st.deps.Boards().RemoveMarker(c.R.Context(), c.SessionID, c.TabuleiroID, marcador.ID)
}

// urlMarker acha o marcador que o gesto aponta.
//
// Ele DEVOLVE O MARCADOR e não só o id porque os três gestos precisam do estado
// atual — revelar alterna, e alternar sem ler é escrever `true` por cima de
// `true`. E achar aqui é o que faz um id inventado virar recusa com frase em vez
// de mutação silenciosa que não acha ninguém.
func urlMarker(st Scene, c commandCtx) (tabuleiro.BoardMarker, error) {
	id := chi.URLParam(c.R, "id")
	b := st.deps.Boards().Get(c.R.Context(), c.SessionID, c.TabuleiroID)
	if b == nil {
		return tabuleiro.BoardMarker{}, errors.New("não há tabuleiro aberto")
	}
	for _, m := range b.Markers {
		if m.ID == id {
			return m, nil
		}
	}
	return tabuleiro.BoardMarker{}, fmt.Errorf("o marcador %q não está neste mapa", id)
}

// coresEmPortugues lista as cores para a frase da recusa — a mensagem tem de
// dizer o que era esperado, não só o que veio errado.
func coresEmPortugues() string {
	nomes := ""
	for i, c := range tabuleiro.MarkerColors {
		if i > 0 {
			nomes += ", "
		}
		nomes += c.Rotulo
	}
	return nomes
}
