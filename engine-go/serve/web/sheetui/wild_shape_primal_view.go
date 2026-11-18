package sheetui

import (
	"encoding/json"
	"fmt"
	"strings"

	"t20engine/domain/engine"
)

// wildPrimalView é o que a Forma Primal (p62) acrescenta ao diálogo da Forma Selvagem: o resumo da fera
// primal, número a número com a forma de onde veio, e os rótulos das escolhas de arma e de tamanho.
// Tudo por chave de sinal, porque o diálogo escolhe as formas sem ir ao servidor.
type wildPrimalView struct {
	// Summaries é o resumo das duas formas, por "primeira:segunda:nível".
	Summaries map[string]string
	// Weapons e Sizes são o rótulo de cada forma nas escolhas, por "forma:nível".
	Weapons, Sizes map[string]string
}

func wildPrimalViewOf(tiers []wildTierOption) *wildPrimalView {
	view := &wildPrimalView{Summaries: map[string]string{}, Weapons: map[string]string{}, Sizes: map[string]string{}}
	for _, option := range tiers {
		for _, first := range engine.WildForms {
			stats, _ := engine.WildShapeStatsOf(first, option.Tier)
			key := string(first) + ":" + string(option.Tier)
			view.Weapons[key] = wildFormShortName(first) + " · " + wildWeaponLabel(stats)
			size := wildSizeLabel(stats.Size)
			if size == "" {
				size = "o da raça"
			}
			view.Sizes[key] = wildFormShortName(first) + " · " + size
			for _, second := range engine.WildForms {
				if second == first {
					continue
				}
				view.Summaries[string(first)+":"+string(second)+":"+string(option.Tier)] = wildPrimalSummary(first, second, option.Tier)
			}
		}
	}
	return view
}

// wildWeaponLabel é a arma natural da forma: "2× 1d10, crítico 19", ou "4d6".
func wildWeaponLabel(stats engine.WildShapeStats) string {
	label := stats.Damage
	if stats.Weapons > 1 {
		label = fmt.Sprintf("%d× %s", stats.Weapons, stats.Damage)
	}
	if stats.CritRange != 0 {
		label += fmt.Sprintf(", crítico %d", stats.CritRange)
	}
	return label
}

// wildPrimalSummary é cada número da fera primal com a forma de onde ele veio: "Força +10 (Feroz)".
// Vale o maior das duas (decisão do dono), e no empate a primeira.
func wildPrimalSummary(first, second engine.WildForm, tier engine.WildTier) string {
	a, _ := engine.WildShapeStatsOf(first, tier)
	b, _ := engine.WildShapeStatsOf(second, tier)
	parts := []string{}
	pick := func(label, format string, x, y int) {
		value, from := x, first
		if y > x {
			value, from = y, second
		}
		if value != 0 {
			parts = append(parts, fmt.Sprintf(format, label, value)+" ("+wildFormShortName(from)+")")
		}
	}
	pick("Força", "%s %+d", a.Strength, b.Strength)
	pick("Destreza", "%s %+d", a.Dexterity, b.Dexterity)
	pick("Defesa", "%s %+d", a.Defense, b.Defense)
	pick("RD", "%s %d", a.Reduction, b.Reduction)
	pick("deslocamento", "%s +%dm", a.PlusMetres, b.PlusMetres)
	pick("voo", "%s %dm", a.FlyMetres, b.FlyMetres)
	return strings.Join(parts, " · ") + ". Bônus iguais não somam: vale o maior."
}

// summaryExpr escreve o resumo das duas formas escolhidas; sem a segunda, quem escreve é o de hoje.
func (p *wildPrimalView) summaryExpr(single string) string {
	return "$wild_second ? (" + jsonOf(p.Summaries) + ")[$wild_form + ':' + $wild_second + ':' + $wild_tier] || '' : " + single
}

// labelExpr é o rótulo da forma escolhida no sinal `form` (a primeira ou a segunda), numa tabela.
func labelExpr(table map[string]string, form string) string {
	return "(" + jsonOf(table) + ")[$" + form + " + ':' + $wild_tier] || ''"
}

func jsonOf(table map[string]string) string {
	raw, err := json.Marshal(table)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
