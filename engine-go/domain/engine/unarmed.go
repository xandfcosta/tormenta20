package engine

import (
	"strconv"
	"strings"
)

// O ATAQUE DESARMADO, que é de todo personagem (p142-143, ALE-486).
//
// "Um ataque desarmado é considerado uma arma leve corpo a corpo que causa dano
// de impacto não letal (1d3 pontos de dano para criaturas Pequenas e Médias)",
// e "todas as criaturas são proficientes em ataques desarmados". Toda ficha
// ganha a carta dele, depois das armas empunhadas (decisão do dono): é
// informação que o livro dá a todos, e é por ela que a Briga, o Trincado e o
// Golpe Cruel do lutador chegam a algum número.
//
// O TAMANHO não é tratado aqui: o passo de dano da raça (`damageStep:all`, p106)
// já anda o dado de toda carta, e a Briga diz a mesma coisa — "Minúsculas
// diminuem esse dano em um passo, Grandes e Enormes aumentam" (p76).

// UnarmedName é o nome da carta desarmada na ficha.
const UnarmedName = "Ataque desarmado"

// UnarmedScope é o escopo do que vale SÓ no ataque desarmado: o dano do
// Trincado e a margem e o multiplicador do Golpe Cruel e do Violento (p77).
const UnarmedScope = "unarmed"

// unarmedStats é a "arma" desarmada do livro, antes da Briga.
func unarmedStats() *WeaponStats {
	return &WeaponStats{Damage: "1d3", CritRange: 20, CritMult: 2, Type: "impacto", Hand: "light", Purpose: "melee"}
}

// unarmedCard é a carta do ataque desarmado.
//
// A BRIGA (p75-77) troca o dado — 1d6 no 1º nível até 2d10 no 20º — e deixa o
// lutador escolher dano letal ou não letal, então a carta deixa de ser não
// letal. Quem mais a faz letal é quem levanta a flag `lethal-unarmed` (a
// Manopla).
func (r *Ruleset) unarmedCard(ch Character, b cardBasis) WeaponCard {
	w := unarmedStats()
	die, brawl := brawlDie(r.ActiveItemsFor(ch))
	if brawl {
		w.Damage = die
	}
	steps := StatFor(b.effects, ModifierTarget{K: "damageStep", Scope: UnarmedScope}).Total
	nonLethal := !brawl && !b.effects.Flags["lethal-unarmed"]
	scopes := append([]string{UnarmedScope}, weaponKindScopes(w)...)
	none := TotalContribs{Contributions: []BreakdownContribution{}}
	return b.card(ch, UnarmedName, w, scopes, none, steps, nonLethal)
}

// brawlDie é o maior dado desarmado que os poderes da ficha declaram
// (`unarmedDamage`), e se algum declara.
//
// O MAIOR, e não o último: os degraus da Briga são linhas separadas do
// catálogo, todas possuídas no nível alto, e a ordem delas na lista não é
// promessa de nada. O maior é o que o livro dá no nível.
func brawlDie(items []ActiveItem) (string, bool) {
	best, bestAverage := "", 0.0
	for _, it := range items {
		for _, m := range it.Modifiers {
			if m.Target.K != "unarmedDamage" || m.Dice == "" {
				continue
			}
			if avg := diceAverage(m.Dice); avg > bestAverage {
				best, bestAverage = m.Dice, avg
			}
		}
	}
	return best, best != ""
}

// diceAverage é a média de "NdF": N × (F+1) / 2. Notação que não é essa vale 0.
func diceAverage(dice string) float64 {
	count, faces, ok := strings.Cut(dice, "d")
	n, errN := strconv.Atoi(count)
	f, errF := strconv.Atoi(faces)
	if !ok || errN != nil || errF != nil {
		return 0
	}
	return float64(n) * float64(f+1) / 2
}
