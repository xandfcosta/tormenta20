package ui

// LiveStream é a expressão que abre um FLUXO LONGO — o `@get` que fica de pé enquanto a página vive — e
// que volta sempre, em cerca de um segundo, depois de cada queda.
//
// O padrão do Datastar 1.0.5 conta as novas tentativas pela vida inteira do `@get`: a espera dobra até
// 30 s, e depois da décima queda ele desiste até recarregarem a página. Numa sessão de horas, o deploy no
// meio do jogo e o Wi-Fi do celular derrubam o fluxo mais do que isso. O `retryScaler: 1` mantém a espera
// no segundo inicial, e o `retryMaxCount: Infinity` tira o teto. Ver "O fluxo longo reconecta" em
// `docs/datastar.md`.
//
//	ui.Page{Init: ui.LiveStream("/campanhas/1/sessoes/5/fluxo")}
func LiveStream(address string) string {
	return "@get('" + address + "', {retryMaxCount: Infinity, retryScaler: 1})"
}
