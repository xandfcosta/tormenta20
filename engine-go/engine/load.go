package engine

import "math"

// A carga do livro (p141), inteira. O motor já sabia dizer QUANTO o personagem
// pode carregar (`inventorySlotsTotal`, o mesmo 10 +2/Força de sempre); o que
// faltava era o outro lado da conta — quanto ele CARREGA — e o que acontece
// quando um passa do outro. Enquanto essa metade não existia aqui, a tela
// somava sozinha, que é uma segunda implementação de regra do livro rodando no
// navegador (ALE-215).

// CoinsPerSlot — "Cada mil moedas, independentemente do tipo, ocupam 1 espaço"
// (p141). O dinheiro é carga como qualquer outra coisa: é por isso que os
// tibares entram na MESMA soma dos itens, e não num contador ao lado.
const CoinsPerSlot = 1000

// Sobrecarga (p141), ao pé da letra: "Se ultrapassar esse limite, fica
// sobrecarregado — sofre penalidade de armadura –5 e seu deslocamento é
// reduzido em –3m."
const (
	overloadArmorPenalty        = -5
	overloadDisplacementPenalty = -3
)

// LoadBreakdown é a carga resolvida: o que ocupa espaço, o teto, e a
// consequência de estourá-lo. `Max` é o segundo teto da p141 — "Você não pode
// carregar mais do que o dobro do seu limite" —, exposto para a mochila poder
// DIZER que o personagem passou dele; o motor não recusa a linha, porque o
// próprio livro deixa a regra de carga a critério do mestre.
type LoadBreakdown struct {
	// Items é quantidade × espaços de cada linha do inventário.
	Items float64 `json:"items"`
	// Coins são os espaços ocupados pelo dinheiro (CoinsPerSlot).
	Coins float64 `json:"coins"`
	Used  float64 `json:"used"`
	// Limit é o mesmo número que a ficha chama de espaços de inventário.
	Limit int `json:"limit"`
	Max   int `json:"max"`
	// Enforced diz se a mesa está USANDO a regra de carga (ALE-221). Com ela
	// desligada os espaços continuam contados — o livro pede que os jogadores
	// "não abusem", e para isso é preciso ver o número —, mas o personagem não
	// fica sobrecarregado e nenhuma penalidade sai daqui.
	Enforced bool `json:"enforced"`
	// Overloaded e OverMax são FALSE quando a regra está desligada, de propósito:
	// "sobrecarregado" é a condição do livro, e ela não existe numa mesa que não
	// usa a regra. Assim quem lê só este campo acerta sem saber do `Enforced`.
	Overloaded          bool `json:"overloaded"`
	OverMax             bool `json:"overMax"`
	ArmorPenalty        int  `json:"armorPenalty"`
	DisplacementPenalty int  `json:"displacementPenalty"`
}

// loadBreakdownOf resolve a carga contra um limite já calculado — quem calcula o
// limite é `inventorySlotsTotal`, e ele entra por parâmetro para os dois lados
// da conta não divergirem.
//
// A conta dos espaços roda SEMPRE, mesmo com a regra desligada: o livro autoriza
// o mestre a ignorá-la "desde que os jogadores não abusem" (p141), e não dá para
// vigiar abuso sem ver o número. O que o interruptor governa é a consequência.
//
// @example loadBreakdownOf(ch, 18).Overloaded // true com 19 espaços na mochila
func loadBreakdownOf(ch Character, limit int) LoadBreakdown {
	items := itemSlotsUsed(ch.Items)
	coins := coinSlots(ch.Tibar)
	used := items + coins
	enforced := !ch.IgnoredRules.Carga
	out := LoadBreakdown{
		Items:      items,
		Coins:      coins,
		Used:       used,
		Limit:      limit,
		Max:        limit * 2,
		Enforced:   enforced,
		Overloaded: enforced && used > float64(limit),
		OverMax:    enforced && used > float64(limit*2),
	}
	if out.Overloaded {
		out.ArmorPenalty = overloadArmorPenalty
		out.DisplacementPenalty = overloadDisplacementPenalty
	}
	return out
}

// itemSlotsUsed soma quantidade × espaços de cada linha. Os espaços de meio em
// meio (p141: alquímicos, poções e pergaminhos ocupam meio espaço) são exatos
// em binário, então a soma não precisa de tolerância.
func itemSlotsUsed(items []CharacterItem) float64 {
	total := 0.0
	for _, it := range items {
		total += float64(it.Quantity) * it.Slots
	}
	return total
}

// coinSlots conta MILHEIROS COMPLETOS: a p141 dá um espaço a cada mil moedas, e
// as 999 que não fecham o milheiro seguinte não ganham espaço nenhum.
func coinSlots(tibar float64) float64 {
	if tibar <= 0 {
		return 0
	}
	return math.Floor(tibar / CoinsPerSlot)
}
