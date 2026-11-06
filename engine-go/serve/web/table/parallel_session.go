package table

import (
	"net/http"

	"t20engine/serve/web/sheetui"
)

// A SESSÃO DO JOGADOR (ALE-473): a ficha que ela desenha e o sistema que a
// página veste.

// tablePlayerSheet carrega a ficha que a superfície "Minha ficha" desenha.
//
// Só na CARGA FRIA e não no stream: a ficha é sete painéis
// computados, e ela muda pelos comandos DELA — que remendam o `#sheet-scene`
// direto. Recomputá-la a cada tique da sessão seria pagar o preço mais caro da
// página para descobrir que nada mudou.
//
// Falha em silêncio de propósito: uma ficha que não carrega tira a aba da tela,
// e não derruba a sessão. Estar numa mesa é mais importante que ver a própria
// ficha dentro dela, e o jogador continua tendo o elenco.
func (s Scene) tablePlayerSheet(r *http.Request, view View) *sheetui.View {
	if view.GM != nil || view.Eu == nil {
		return nil
	}
	return s.deps.PlayerSheet(r, view.Eu.CharacterID)
}

// streamOfPage é o fluxo que a página abre, com a marca da página à parte: o
// fluxo de uma página remenda só as regiões dela.
func streamOfPage(view View) string {
	return view.SessionBase() + "/fluxo" + view.pageMark()
}
