package table

import (
	"net/http"

	"t20engine/serve/web/sheetui"
	"t20engine/serve/web/ui"
)

// A SESSÃO PARALELA (ALE-473): a marca `?tempestade=1`, quem a lê, e a ficha
// que a sessão desenha.

// parallelSessionAsked diz se a sessão é a paralela (`?tempestade=1`): só
// para o JOGADOR com personagem nela. O mestre segue com o palco de hoje, e
// quem não tem personagem não tem a Ficha que esta fatia migra.
//
// TRÊS caminhos a leem, e os três desenham a mesa: a página, cada quadro do
// stream e a resposta dos comandos. Um que a esquecesse devolvia o desenho de
// hoje dentro da sessão paralela — o stream e os comandos já esqueceram.
func parallelSessionAsked(r *http.Request, view View) bool {
	return sheetui.TempestadeAsked(r) && view.GM == nil && view.Eu != nil
}

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
// O FLUXO LEVA A MARCA: é ele que redesenha o cabeçalho a cada tique.
func pageSystemAndStream(view View) (ui.DesignSystem, string) {
	if view.Tempestade {
		return ui.SystemTempestade, view.SessionBase() + "/fluxo?tempestade=1"
	}
	return ui.SystemGrimoire, view.SessionBase() + "/fluxo"
}

// wearParallel veste a view da sessão paralela quando ela foi pedida: a marca
// vai para o cabeçalho e para os gestos do mapa, que respondem redesenhando o
// tabuleiro. As abas e a volta do puxão chegam montadas sem ela, e são
// remontadas aqui.
func (v *View) wearParallel(asked bool) {
	v.Tempestade = asked
	if !asked {
		return
	}
	v.Board.Mark = markedIfParallel("", true)
	for i := range v.Board.Tabs {
		v.Board.Tabs[i].Command = tabCommand(v.Board, v.Board.Tabs[i].ID)
	}
	if v.Board.Pulled != nil {
		v.Board.Pulled.ReturnCommand = tabCommand(v.Board, v.Board.Pulled.ReturnTab)
	}
}
