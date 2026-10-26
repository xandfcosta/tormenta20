package sheetui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"t20engine/domain/book"
)

// OS GASTOS DE PM NO DIÁLOGO DA PERÍCIA (ALE-489).
//
// O diálogo mostra o que a ficha pode pagar NESTE teste — a oferta é do
// `book.SkillSpendOffers`, com a posse resolvida pelo `character` —, e o
// jogador marca. O sinal é um objeto, `expertise_spends`, com uma chave por
// gasto (`book.SpendKey`) valendo os PM pedidos; zero é "não marcado". A conta
// da prévia é só SOMA de números que o servidor mandou, e quem cobra e confere
// é o `RollExpertise`.

// expertiseSpend é um gasto oferecido numa perícia.
type expertiseSpend struct {
	Key   string
	Name  string
	PM    int
	Bonus int
	// PerPM é o Orgulho: o jogador escolhe quantos PM, até MaxPM.
	PerPM bool
	MaxPM int
	// Unlocks destrava a perícia só para treinados.
	Unlocks bool
}

// withSpends pendura na linha os gastos que valem nela. O teto do Orgulho é o
// menor entre o Carisma e o PM que a ficha tem.
func (row *expertiseRow) withSpends(offers []book.SkillSpendOffer, mana int) {
	row.Mana = mana
	for _, o := range offers {
		spend := expertiseSpend{
			Key: o.Key, Name: o.Name, PM: o.PM, Bonus: o.Bonus,
			PerPM: o.Kind == book.SpendPerPM, Unlocks: o.Unlocks,
		}
		if spend.PerPM {
			spend.MaxPM = min(o.MaxPM, mana)
			if spend.MaxPM < 1 {
				continue
			}
		}
		row.Spends = append(row.Spends, spend)
		row.SpendsUnlock = row.SpendsUnlock || spend.Unlocks
	}
}

// spendSignal é a referência ao sinal do gasto.
func spendSignal(key string) string {
	return "$expertise_spends." + key
}

// spendBonusExpr é o que o gasto soma agora: zero desmarcado, o bônus fixo
// marcado, e o bônus vezes os PM no Orgulho.
func spendBonusExpr(s expertiseSpend) string {
	if s.PerPM {
		return fmt.Sprintf("(%d * %s)", s.Bonus, spendSignal(s.Key))
	}
	return fmt.Sprintf("(%s > 0 ? %d : 0)", spendSignal(s.Key), s.Bonus)
}

// spendCostExpr é o PM que o gasto custa agora.
func spendCostExpr(s expertiseSpend) string {
	if s.PerPM {
		return fmt.Sprintf("(%d * %s)", s.PM, spendSignal(s.Key))
	}
	return fmt.Sprintf("(%s > 0 ? %d : 0)", spendSignal(s.Key), s.PM)
}

// spendsTotalExpr é o total da perícia com os gastos marcados, com sinal.
func spendsTotalExpr(row expertiseRow) string {
	parts := []string{strconv.Itoa(row.TotalValue)}
	for _, s := range row.Spends {
		parts = append(parts, spendBonusExpr(s))
	}
	sum := "(" + strings.Join(parts, " + ") + ")"
	return fmt.Sprintf("(%s >= 0 ? '+' : '') + %s", sum, sum)
}

// spendsCostExpr é o PM de todos os gastos marcados.
func spendsCostExpr(row expertiseRow) string {
	parts := []string{"0"}
	for _, s := range row.Spends {
		parts = append(parts, spendCostExpr(s))
	}
	return "(" + strings.Join(parts, " + ") + ")"
}

// spendsLeftExpr é o PM que sobra depois de pagar o que está marcado.
func spendsLeftExpr(row expertiseRow) string {
	return strconv.Itoa(row.Mana) + " - " + spendsCostExpr(row)
}

// spendsUnlockedExpr diz se algum gasto marcado destrava a perícia.
func spendsUnlockedExpr(row expertiseRow) string {
	parts := []string{"false"}
	for _, s := range row.Spends {
		if s.Unlocks {
			parts = append(parts, spendSignal(s.Key)+" > 0")
		}
	}
	return "(" + strings.Join(parts, " || ") + ")"
}

// spendToggleCommand marca ou desmarca o gasto. O Orgulho nasce com 1 PM.
func spendToggleCommand(s expertiseSpend) string {
	return fmt.Sprintf("%s = %s > 0 ? 0 : 1", spendSignal(s.Key), spendSignal(s.Key))
}

// spendStepCommand anda o Orgulho em `delta`, entre 1 e o teto.
func spendStepCommand(s expertiseSpend, delta int) string {
	return fmt.Sprintf("%s = Math.max(1, Math.min(%d, %s + %d))",
		spendSignal(s.Key), s.MaxPM, spendSignal(s.Key), delta)
}

// spendStepLabelExpr é o que o contador do Orgulho mostra: "+4 · 2 PM".
func spendStepLabelExpr(s expertiseSpend) string {
	return fmt.Sprintf("'+' + %s + ' · ' + %s + ' PM'", spendBonusExpr(s), spendCostExpr(s))
}

// spendChipCost é o que o chip diz antes de tocar: "+5 · 2 PM", ou "+2 por PM"
// no Orgulho, ou "destrava · 1 PM" na Cultura Exótica.
func spendChipCost(s expertiseSpend) string {
	switch {
	case s.PerPM:
		return fmt.Sprintf("+%d por PM", s.Bonus)
	case s.Bonus == 0:
		return fmt.Sprintf("destrava · %d PM", s.PM)
	}
	return fmt.Sprintf("+%d · %d PM", s.Bonus, s.PM)
}

// spendsResetCommand zera os gastos desta perícia ao abrir o diálogo: nada fica
// marcado de um teste para o outro.
func spendsResetCommand(row expertiseRow) string {
	out := ""
	for _, s := range row.Spends {
		out += spendSignal(s.Key) + " = 0; "
	}
	return out
}

// ExpertiseSpendsDeclaration é a declaração do sinal com TODAS as chaves que o
// catálogo conhece, valendo zero. As duas páginas que desenham o diálogo — a
// ficha e a Mesa — a usam: sem ela a chave nasceria texto ao ser lida (o defeito
// da ALE-499).
//
//	ExpertiseSpendsDeclaration() // "expertise_spends: {audacia: 0, …}"
func ExpertiseSpendsDeclaration() string {
	keys := map[string]bool{}
	for _, a := range book.Activations() {
		if a.SkillSpend != nil {
			keys[book.SpendKey(a.ID)] = true
		}
	}
	sorted := make([]string, 0, len(keys))
	for key := range keys {
		sorted = append(sorted, key+": 0")
	}
	sort.Strings(sorted)
	return "expertise_spends: {" + strings.Join(sorted, ", ") + "}"
}
