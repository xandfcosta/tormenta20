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
func pageSystemAndStream(view View) (ui.DesignSystem, string) {
	if view.BoardWindow {
		return ui.SystemTempestade, view.SessionBase() + "/fluxo?janela=tabuleiro"
	}
	if view.Tempestade {
		return ui.SystemTempestade, view.SessionBase() + "/fluxo"
	}
	return ui.SystemGrimoire, view.SessionBase() + "/fluxo"
}

// A MESA DO MESTRE EM PARALELO (ALE-522): `?tempestade=1` grava um cookie, e
// é o cookie que a página, o stream e os comandos leem. Uma marca no endereço
// teria de viajar em cada gesto, e a da sessão do jogador esqueceu três
// caminhos antes de dar certo; o cookie viaja sozinho. `?tempestade=0` apaga.
const GmTableCookie = "mesa_mestre_tempestade"

// rememberGmTable grava ou apaga o cookie pela marca do endereço.
func rememberGmTable(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Query().Get("tempestade") {
	case "1":
		http.SetCookie(w, &http.Cookie{Name: GmTableCookie, Value: "1", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
	case "0":
		http.SetCookie(w, &http.Cookie{Name: GmTableCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
}

// gmTableAsked diz se o pedido quer a mesa nova do mestre: a marca do
// endereço vence, e sem ela vale o cookie.
func gmTableAsked(r *http.Request) bool {
	switch r.URL.Query().Get("tempestade") {
	case "1":
		return true
	case "0":
		return false
	}
	c, err := r.Cookie(GmTableCookie)
	return err == nil && c.Value == "1"
}

// dressForGm veste a view do MESTRE no Tempestade quando ele pediu.
func (v *View) dressForGm(r *http.Request) {
	if v.GM != nil {
		v.Tempestade = gmTableAsked(r)
		// A JANELA À PARTE do tabuleiro (ALE-522) é a mesma sessão desenhando
		// só o mapa; o fluxo dela leva a marca para remendar só o mapa.
		v.BoardWindow = v.Tempestade && r.URL.Query().Get("janela") == "tabuleiro"
	}
}
