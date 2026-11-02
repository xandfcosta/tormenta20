package sheetui

import (
	"cmp"
	"slices"
	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/serve/web/ui"
)

// OS VITAIS da ficha: a barra de PV e PM, a reserva de temporários e a palavra
// de quem caiu. Saíram do `view.go` quando a palavra (ALE-366) o levou além do
// teto de 500 linhas — e eles mudam por outra razão que a casca da ficha.

type sheetVital struct {
	Current int64
	Max     int64
	// Fraction é "12/20", que é como a mesa fala.
	Fraction string
	// Percent é a largura da barra, entre 0 e 100.
	Percent int
	// Down é a palavra de quem caiu — morrendo, estável, morto (p236) —, vazia
	// de pé e sempre vazia no PM.
	Down string
	// Temp é o PV TEMPORÁRIO, e ele é uma parcela À PARTE e não um somando.
	//
	// A barra continua sendo o PV de verdade: somar o temporário ao atual faria
	// um herói a 50/137 com 70 de reserva desenhar 88% de vida com 36% de
	// carne. O livro autoriza os dois desenhos — *"são somados a seus pontos
	// atuais, mesmo que ultrapassem o máximo"* (p106) —, e o que decide é a
	// pergunta que a barra responde, que é "quanto apanhei".
	//
	// Vazio quer dizer que não há reserva, e aí a fileira fica IGUAL à de antes.
	Temp string
}

// vital monta a barra de PV ou PM.
//
// A FRAÇÃO é o que a mesa fala em voz alta ("doze de vinte"), e a porcentagem é
// só a largura da barra. Máximo ZERO não vira divisão por zero nem barra cheia:
// quem não tem mana tem a barra vazia e apagada.
func vital(current, max int64) sheetVital {
	return sheetVital{
		Current: current, Max: max,
		Fraction: ui.HitPoints(current) + "/" + strconv.FormatInt(max, 10),
		Percent:  ui.VitalPercent(current, max),
	}
}

// downed escreve no PV a palavra de quem caiu. O Sangrando que separa morrendo
// de estável está nas condições da ficha.
func downed(v sheetVital, activeConditions string) sheetVital {
	v.Down = ui.DownedWord(v.Current, v.Max,
		slices.Contains(sheet.UnmarshalStrings(activeConditions), engine.ConditionBleeding))
	return v
}

// withTempPool acrescenta ao vital a reserva que os efeitos ativos carregam.
//
// A conta é do `sheet`, e a leitura NÃO custa consulta: o agregado já traz os
// efeitos, porque a aba Efeitos os desenha.
//
// OS DOIS VITAIS, e aqui morava o motivo de só um: *"o livro tem pontos de mana
// temporários e este app ainda não os modela — nada os gasta"*. Hoje gasta: o
// `sheet.SpendMana` drena a poça antes do poço em toda cobrança de PM, e o
// número desenhado é consumido por conjurar, ativar poder e sustentar.
func withTempPool(v sheetVital, target string, effects []sheet.EffectDTO) sheetVital {
	if total := sheet.TempTotal(target, sheet.ModifierBlobsOf(effects)); total > 0 {
		v.Temp = "+" + strconv.Itoa(total)
	}
	return v
}

// vitalLevelRow é a linha da conta de um nível: de onde vem, o cálculo e o valor.
type vitalLevelRow struct {
	Title   string
	Account string
	Total   string
}

// vitalLevelsOf é a conta nível a nível dos dois poços, em linhas de tela.
//
// A ordem dos níveis é o registro de escolhas, e o motor a confere contra as classes.
// Sem motor ou com a ficha ilegível não há conta a mostrar, e a linha que a abre some.
func vitalLevelsOf(dto sheet.CharacterDTO) (pv, pm []vitalLevelRow) {
	if dto.Ruleset == nil {
		return nil, nil
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return nil, nil
	}
	levels := dto.Ruleset.VitalLevelsFor(dto.Ruleset.VitalContextFor(ec), classByLevelOf(dto))
	return vitalLevelRowsOf(levels.PV), vitalLevelRowsOf(levels.PM)
}

// classByLevelOf é a classe de cada nível, na ordem do registro.
func classByLevelOf(dto sheet.CharacterDTO) []string {
	rows := slices.Clone(dto.LevelChoices)
	rows = slices.DeleteFunc(rows, func(row sheet.LevelChoiceDTO) bool { return row.Kind != "class" })
	slices.SortStableFunc(rows, func(a, b sheet.LevelChoiceDTO) int { return cmp.Compare(a.Level, b.Level) })
	classes := make([]string, len(rows))
	for i, row := range rows {
		classes[i] = row.Ref
	}
	return classes
}

func vitalLevelRowsOf(lines []engine.VitalLevelLine) []vitalLevelRow {
	rows := make([]vitalLevelRow, len(lines))
	for i, line := range lines {
		rows[i] = vitalLevelRow{
			Title:   "Nível " + strconv.Itoa(line.Level) + " · " + line.ClassName,
			Account: vitalLevelAccount(line),
			Total:   strconv.Itoa(line.Total),
		}
	}
	return rows
}

// vitalLevelAccount escreve o cálculo de um nível.
//
//	vitalLevelAccount(line) // "6 da classe + 5 de Inteligência + 1 de Bênção do Mana"
func vitalLevelAccount(line engine.VitalLevelLine) string {
	account := ""
	for i, part := range line.Parts {
		amount := part.Amount
		if i > 0 {
			if amount < 0 {
				account += " − "
				amount = -amount
			} else {
				account += " + "
			}
		}
		account += strconv.Itoa(amount) + vitalPartName(part)
	}
	// "Você sempre ganha pelo menos 1 PV ao subir de nível" (p35).
	if line.Floored {
		account += ": mínimo 1"
	}
	return account
}

func vitalPartName(part engine.VitalLevelPart) string {
	switch {
	case part.Attribute != "":
		return " de " + book.AttributeLabel(part.Attribute)
	case part.Source != "":
		return " de " + part.Source
	}
	return " da classe"
}

func vitalLevelsID(which string) string { return "vital-" + which + "-levels" }

// vitalLevelsTitle é "Como chega em 87": o máximo, que é o que a conta explica.
func vitalLevelsTitle(vital sheetVital) string {
	return "Como chega em " + strconv.FormatInt(vital.Max, 10)
}
