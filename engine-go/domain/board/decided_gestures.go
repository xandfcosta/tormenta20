package board

import (
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// Os gestos que DECIDEM sobre o tabuleiro que leram — a próxima letra, o oposto do que está lá, a
// parada a tirar. A decisão mora aqui, e não na cena, porque é aqui que ela roda sobre o tabuleiro que
// a mutação acabou de ler: decidida antes, sobre uma leitura à parte, dois cliques quase juntos liam a
// mesma versão e gravavam a mesma decisão (dois "C", ou dois toques que se anulam em vez de se
// desfazerem).

// AddNextMarker põe um marcador na casa, com a primeira letra LIVRE, na cor padrão e ESCONDIDO.
//
// Escondido ao nascer é a razão de o marcador existir: marcar a armadilha na frente da mesa entrega a
// armadilha.
//
//	AddNextMarker(b, engine.Square{X: 2, Y: 3}, newID) // o mapa com "A" e "B" ganha o "C"
func AddNextMarker(b *BoardState, square engine.Square, newID func() string) error {
	return AddMarker(b, BoardMarker{
		X: square.X, Y: square.Y,
		Text:   NextMarkerLetter(b.Markers),
		Color:  DefaultMarkerColor(),
		Hidden: true,
	}, newID)
}

// ToggleMarkerHidden mostra o marcador escondido e esconde o que está à mostra.
//
// ALTERNA e não "revela": o mestre que revelou cedo demais precisa poder esconder de volta.
func ToggleMarkerHidden(b *BoardState, markerID string) error {
	for i := range b.Markers {
		if b.Markers[i].ID == markerID {
			return UpdateMarker(b, markerID, MarkerReveal(!b.Markers[i].Hidden))
		}
	}
	return fmt.Errorf("o marcador %q não está neste mapa", markerID)
}

// RemoveMarkerOrRefuse tira o marcador, e RECUSA o que não existe: o `RemoveMarker` some em silêncio,
// e para o botão de apagar um id que não acha nada é recusa com frase, e não clique mudo.
func RemoveMarkerOrRefuse(b *BoardState, markerID string) error {
	for _, m := range b.Markers {
		if m.ID == markerID {
			RemoveMarker(b, markerID)
			return nil
		}
	}
	return fmt.Errorf("o marcador %q não está neste mapa", markerID)
}

// ToggleTokenHidden esconde a peça à mostra e mostra a escondida.
//
// ALTERNA e não recebe o estado desejado: mandar o valor da tela faria dois cliques rápidos com a
// resposta atrasada apagarem um ao outro — e o empate é a emboscada aparecendo para a mesa.
func ToggleTokenHidden(b *BoardState, tokenID string) error {
	token := FindToken(b, tokenID)
	if token == nil {
		return fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	return UpdateToken(b, tokenID, ParseTokenPatch(map[string]any{"hidden": !token.Hidden}))
}

// ToggleTokenMoving liga e desliga o "em movimento" de uma peça de cenário (p239).
//
// SÓ PEÇA COM ESTATÍSTICAS: "em movimento" é uma linha da Tabela 5-3 sobre a Defesa de um OBJETO, e
// marcá-la num ogro não muda nada — um botão que não faz nada é pior que um botão que não existe.
func ToggleTokenMoving(b *BoardState, tokenID string) error {
	token := FindToken(b, tokenID)
	if token == nil {
		return fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	if !token.HasObjectStats() {
		return fmt.Errorf(
			"%s não tem estatísticas de objeto: o +5 por estar em movimento é da "+
				"Defesa de um objeto (p239)", token.Label)
	}
	return UpdateToken(b, tokenID, ParseTokenPatch(map[string]any{"moving": !token.Moving}))
}

// RemoveTokenOrRefuse tira a peça, e RECUSA a que não existe, pela mesma razão do
// `RemoveMarkerOrRefuse`: a frase diz o id porque é ele que o botão carregava.
func RemoveTokenOrRefuse(b *BoardState, tokenID string) error {
	if FindToken(b, tokenID) == nil {
		return fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	RemoveToken(b, tokenID)
	return nil
}

// UndoLastStop tira a última perna do movimento que esta pessoa monta com a peça.
//
// É a ordem do arrependimento: primeiro se tira a perna errada, e só depois se cancela tudo. Sobrando
// só a origem, desfazer VIRA cancelar — uma proposta sem perna nenhuma não é proposta, e deixar um
// provisório de custo zero na mesa seria pedir um "Confirmar" que não move ninguém.
//
// Reconstrói pelas paradas que sobraram em vez de cortar o fim do caminho: o número de quadrados de um
// trecho não se deduz das paradas sem redesenhá-lo, e redesenhar é o que o `PathThroughStops` faz.
func UndoLastStop(b *BoardState, st *live.SessionRuntimeState, tokenID string, by Mover) error {
	stops, err := StopsSoFar(b, tokenID, by.UserID)
	if err != nil {
		return err
	}
	if len(stops) < 2 {
		return fmt.Errorf("não há parada a desfazer em %q", tokenID)
	}
	if stops = stops[:len(stops)-1]; len(stops) < 2 {
		return CancelMove(b, by)
	}
	return ProposeMoveWithStops(b, st, tokenID, stops, by)
}
