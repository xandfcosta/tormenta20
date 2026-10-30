package table

import (
	"net/http"

	"t20engine/serve/web/sheetui"
	"t20engine/serve/web/ui"
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

// pageSystemAndStream é o sistema que a página veste e o fluxo que ela abre.
// A janela à parte do tabuleiro leva a marca no fluxo, para remendar só o mapa.
func pageSystemAndStream(view View) (ui.DesignSystem, string) {
	if view.BoardWindow {
		return ui.SystemTempestade, view.SessionBase() + "/fluxo?janela=tabuleiro"
	}
	return ui.SystemTempestade, view.SessionBase() + "/fluxo"
}

// dressBoardWindow liga a JANELA À PARTE do tabuleiro do mestre (ALE-522): a
// mesma sessão desenhando só o mapa.
func (v *View) dressBoardWindow(r *http.Request) {
	v.BoardWindow = v.GM != nil && r.URL.Query().Get("janela") == "tabuleiro"
}
