package api

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

// A REGRA OPCIONAL CHEGA À FICHA ABERTA NA MESA.
//
// Desligar a carga muda a conta de cada ficha da campanha sem tocar na linha
// dela, e a "Minha ficha" não é região do fluxo: quem a repede é o cliente,
// quando o fluxo manda o aviso `sheet_version`, que é o carimbo da linha. O
// gesto carimba as fichas da mesa, como a regra da mesa
// (`TestAChangedRuleWakesTheSheetsAtTheTable`); sem isso a ficha aberta ficava
// com a carga velha.
func TestTheOptionalRuleReachesTheOpenSheet(t *testing.T) {
	f := newSceneFixture(t)
	srv := liveServer(t, f)
	table := opensStream(t, srv, f, f.player, f.tableUrl()+"/fluxo")
	// O CONTROLE: a Mesa do jogador abriu. Sem ele, o silêncio depois não
	// distinguiria o aviso que faltou de um fluxo que nunca abriu.
	table.waitsFor(t, 3*time.Second, "a Mesa do jogador não desenhou", containing("table-header"))

	f.posts(t, f.gm, "/campanhas/"+strconv.FormatInt(f.campaignID, 10)+"/regras/carga", "{}")
	table.waitsFor(t, eventReach, "o mestre desligou a carga e a ficha aberta do jogador não foi avisada",
		func(frame string) bool { return strings.Contains(frame, "sheet_version") })
}
