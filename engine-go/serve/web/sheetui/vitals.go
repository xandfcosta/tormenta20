package sheetui

import (
	"slices"
	"strconv"

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

// withTempHp acrescenta ao PV a reserva que os efeitos ativos carregam.
//
// A conta é do `sheet` (`TempHpTotal`), e a leitura NÃO custa consulta: o
// agregado já traz os efeitos, porque a aba Efeitos os desenha.
//
// PM não ganha o mesmo: o livro tem pontos de mana temporários (p106) e este
// app ainda não os modela — o motor só conhece o alvo `tempMp` como
// modificador, e nada os gasta. Desenhar um número que nada consome seria pior
// que não desenhá-lo.
func withTempHp(v sheetVital, effects []sheet.EffectDTO) sheetVital {
	blobs := make([]string, 0, len(effects))
	for _, e := range effects {
		blobs = append(blobs, e.Modifiers)
	}
	if total := sheet.TempHpTotal(blobs); total > 0 {
		v.Temp = "+" + strconv.Itoa(total)
	}
	return v
}
