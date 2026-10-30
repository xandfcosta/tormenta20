package table

import (
	"fmt"
	"strings"

	"t20engine/domain/board"
	"t20engine/domain/book"

	"github.com/a-h/templ"
)

// superficieDasConsultas é a terceira superfície do MESTRE: notas, grupo,
// elenco e bestiário (ALE-522).
const superficieDasConsultas = "consultas"

// gmSurfaces são as três superfícies do mestre, na ordem da barra. A Fila usa o
// valor da Mesa do jogador porque é ela que abre, e o sinal nasce nele.
var gmSurfaces = []playerSurface{
	{superficieDaMesa, "Fila", "ListOrdered"},
	{superficieDoTabuleiro, "Tabuleiro", "LayoutGrid"},
	{superficieDasConsultas, "Consultas", "BookOpen"},
}

// gmWhere é a linha de onde a mesa está: "Sessão 1 · Rodada 2".
func gmWhere(v View) string {
	if v.Turn.Kind == "idle" {
		return fmt.Sprintf("Sessão %d", v.SessionNum)
	}
	return fmt.Sprintf("Sessão %d · Rodada %d", v.SessionNum, v.Round)
}

// gmTurnAttrs é o gesto de andar a vez, desligado quando a fila não anda.
func gmTurnAttrs(v View, route string) templ.Attributes {
	attrs := templ.Attributes{"type": "button"}
	if v.GM.CanAdvance {
		attrs["data-on:click"] = tableCommand(v, "POST", route)
	} else {
		attrs["disabled"] = true
	}
	return attrs
}

// combatantDialog é o id da folha de uma linha.
func combatantDialog(row tableRow) string { return "combatant-" + row.ID }

// combatantMeta é a linha de cima da folha: o tipo e a iniciativa.
func combatantMeta(row tableRow) string {
	kind := "NPC"
	if row.IsSheet {
		kind = "Ficha"
	}
	return fmt.Sprintf("%s · iniciativa %d", kind, row.Initiative)
}

// conditionWritten é o nome da condição como a mesa o lê.
func conditionWritten(id string) string { return book.ConditionName(id) }

// openEditInline abre a edição DENTRO da folha (o lápis do topo), semeando os
// sinais que o `saveEditInline` manda.
func openEditInline(row tableRow) string {
	pv, pvMax := int64(0), int64(0)
	if row.PV != nil {
		pv, pvMax = row.PV.Current, row.PV.Max
	}
	return fmt.Sprintf("$edit_row = $edit_row === '%s' ? '' : '%s'; $edit_name = %s; $edit_initiative = %d; $edit_hp = %d; $edit_hp_max = %d",
		row.ID, row.ID, jsTextHow(row.Label), row.Initiative, pv, pvMax)
}

// saveEditInline grava e fecha a edição.
func saveEditInline(v View) string {
	return fmt.Sprintf("@post('%s/iniciativa/' + $edit_row + '/editar'); $edit_row = ''", v.SessionBase())
}

// openConditionsInline aponta a lista das condições para esta linha.
func openConditionsInline(row tableRow) string {
	return fmt.Sprintf("$condition_row = %q; $row_conditions = %q; $row_label = %s",
		row.ID, strings.Join(row.Conditions, ","), jsTextHow(row.Label))
}

// rowVitalBy é o gesto de mexer num pool por um passo escrito — "1", ou a
// expressão do valor digitado.
func rowVitalBy(v View, row tableRow, pool, verb, step string) string {
	return fmt.Sprintf("@post('%s/iniciativa/%s/vitais/%s/%s/%s')", v.SessionBase(), row.ID, pool, verb, step)
}

// poolLoss e poolGain são os verbos da folha: o PV leva dano e recebe cura; o
// PM se gasta e se recupera.
func poolLoss(pool string) string {
	if pool == "mp" {
		return "Gastar"
	}
	return "Dano"
}

func poolGain(pool string) string {
	if pool == "mp" {
		return "Recuperar"
	}
	return "Cura"
}

// restOptions são as qualidades do descanso do dia (p106), na ordem da tela.
var restOptions = []struct{ Value, Label string }{
	{"ruim", "Ruim"}, {"normal", "Normal"}, {"confortavel", "Confortável"}, {"luxuosa", "Luxuosa"},
}

// conditionCatalog são as condições do livro, para a lista de aplicar.
func conditionCatalog() []book.Condition { return book.Catalogs().Conditions }

// gmCurtainLabel e gmLensLabel dizem o que o toque FAZ, nos dois estados.
func gmCurtainLabel(b BoardView) string {
	if b.CurtainNotice {
		return "Abrir a cortina para a mesa"
	}
	return "Fechar a cortina"
}

func gmLensLabel(b BoardView) string {
	if b.Lens {
		return "Voltar à vista do mestre"
	}
	return "Ver a cena como a mesa vê"
}

// isBrush diz se a ferramenta é uma espécie de terreno (p238).
func isBrush(id string) bool {
	for _, k := range board.TerrainKinds {
		if string(k.ID) == id {
			return true
		}
	}
	return false
}

// terrainPressed é a expressão que acende o Terreno com qualquer espécie na mão.
func terrainPressed() string {
	parts := make([]string, 0, len(board.TerrainKinds))
	for _, k := range board.TerrainKinds {
		parts = append(parts, onIsTool(string(k.ID)))
	}
	return "(" + strings.Join(parts, " || ") + ")"
}

// boardWindowOpens abre o tabuleiro numa janela à parte, com o nome fixo:
// clicar de novo traz a mesma janela em vez de abrir outra.
func boardWindowOpens(v View) string {
	return fmt.Sprintf("const w = window.open('%s?janela=tabuleiro', '%s', 'popup,width=1280,height=800'); if (w) { w.focus() }", v.SessionBase(), boardWindowName)
}

// O TABULEIRO NUMA JANELA sai da aba original para dar a largura à fila e às
// Consultas (decisão do dono). É o mesmo acordo das notas (`notesWindowKey`):
// a janela anuncia por `localStorage` que tomou o tabuleiro e o devolve no
// `pagehide`, e a aba ouve o `storage`. O valor é o ID da sessão, porque a
// janela da sessão 4 não tem nada a dizer sobre a 7.
const (
	boardWindowKey  = "t20:tabuleiro-janela"
	boardWindowName = "t20-tabuleiro"
)

func takesTheBoard(v View) string {
	return fmt.Sprintf("localStorage.setItem('%s', '%d')", boardWindowKey, v.SessionID)
}

func givesTheBoardBack(v View) string {
	return fmt.Sprintf("if (localStorage.getItem('%s') === '%d') { localStorage.removeItem('%s') }",
		boardWindowKey, v.SessionID, boardWindowKey)
}

func seesTheBoardAway(v View) string {
	return fmt.Sprintf("$gm_board_away = localStorage.getItem('%s') === '%d'", boardWindowKey, v.SessionID)
}

func watchesTheBoardWindow(v View) string {
	return fmt.Sprintf("if (evt.key === '%s') { $gm_board_away = evt.newValue === '%d' }", boardWindowKey, v.SessionID)
}
