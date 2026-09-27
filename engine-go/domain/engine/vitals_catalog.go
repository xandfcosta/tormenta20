package engine

// OS POÇOS ORIENTADOS A CATÁLOGO (PV/PM máximos).
//
// O arquivo carregava um "v2" no nome que nunca distinguiu versão nenhuma: o
// que ele separa do `vitals.go` ao lado é a FONTE — aquele é a
// tabela fixa de classe, este junta as concessões dos catálogos primados e
// recebe os totais FINAIS de atributo. O nome novo diz isso (ALE-354).
//
// Distinct from vitals.go, which is catalog-FREE (a hardcoded class table): this
// one reuses that pool math but collects the maxPv/maxPm grants from the PRIMED
// catalogs, and takes the FINAL attribute totals — the ones computed over the
// resolved effects, not the raw column.

// VitalContext is the normalized input the grant collection reads. `RaceID` is
// the race NAME — `getRace` resolves by it. AttrTotals are the FINAL totals,
// after item and race mods.
type VitalContext struct {
	Level              int                              `json:"level"`
	Classes            []ClassEntry                     `json:"classes"`
	RaceID             string                           `json:"raceId"`
	RaceAbilityChoices []string                         `json:"raceAbilityChoices"`
	PowerIDs           []string                         `json:"powerIds"`
	ClassChoices       map[string]ClassChoiceSelections `json:"classChoices"`
	GodPower           string                           `json:"godPower"`
	Origin             string                           `json:"origin"`
	OriginChoices      []string                         `json:"originChoices"`
	AttrTotals         map[string]int                   `json:"attrTotals"`
}

type VitalPools struct {
	PvMax int `json:"pvMax"`
	PmMax int `json:"pmMax"`
}

// ComputeVitals: multiclass pools (p34-35) + the summed maxPv/maxPm grants,
// floored at 0. The pool helpers are shared with vitals.go
// (classVitalsTable/multiclass*).
func (r *Ruleset) ComputeVitals(ctx VitalContext) VitalPools {
	con := ctx.AttrTotals["constitution"]
	groupHP, groupMP := r.sumVitalGrants(ctx)
	return VitalPools{
		PvMax: max(0, multiclassPvPool(ctx.Classes, con)+groupHP),
		PmMax: max(0, multiclassMpPool(ctx.Classes)+groupMP),
	}
}

// sumVitalGrants sums maxPv/maxPm over owned abilities,
// evaluating each scale, with the p226 dedupe — "o valor de um mesmo atributo
// não se acumula em características do personagem. Ou seja, um clérigo/druida
// não soma duas vezes sua Sabedoria nos pontos de mana". Por ALVO e ATRIBUTO:
// dois atributos DIFERENTES no mesmo alvo continuam somando.
//
// A metade da DEFESA da mesma regra (o bucaneiro/nobre e o Carisma) não está
// implementada — o resolveStack agrupa por bonusType, não por atributo.
func (r *Ruleset) sumVitalGrants(ctx VitalContext) (pv, pm int) {
	seen := map[string]bool{}
	for _, m := range r.vitalGrantMods(ctx) {
		if m.Target.K != "maxPv" && m.Target.K != "maxPm" {
			continue
		}
		if m.Scale != nil && m.Scale.Per == "attribute" {
			key := m.Target.K + ":" + m.Scale.Attribute
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		amount := evalModifierScale(m.Amount, m.Scale, ctx.Level, ctx.AttrTotals)
		if m.Target.K == "maxPv" {
			pv += amount
		} else {
			pm += amount
		}
	}
	return pv, pm
}

// vitalGrantMods collects every modifier from the abilities
// the character owns — race, class powers (per class at ITS level), general
// powers, the god power, and origin benefits. NOT items/activeEffects (those
// aren't vital-grant sources). Filtered to vital targets by the caller.
func (r *Ruleset) vitalGrantMods(ctx VitalContext) []Modifier {
	out := []Modifier{}
	powers := toSet(ctx.PowerIDs)

	if ctx.RaceID != "" {
		if race := r.getRace(ctx.RaceID); race != nil {
			out = append(out, raceModifiers(race, toSet(ctx.RaceAbilityChoices))...)
		}
	}
	for _, ce := range ctx.Classes {
		for _, power := range r.ownedClassPowers(ce.ClassName, ce.Level, powers, ctx.ClassChoices[ce.ClassName]) {
			out = append(out, power.Modifiers...)
		}
	}
	for _, id := range ctx.PowerIDs {
		if gp := r.getGeneralPower(id); gp != nil {
			out = append(out, gp.Modifiers...)
		}
	}
	if ctx.GodPower != "" {
		if gp := r.grantedPowerByName(ctx.GodPower); gp != nil {
			out = append(out, gp.Modifiers...)
		}
	}
	if ctx.Origin != "" {
		if origin := r.getOrigin(ctx.Origin); origin != nil {
			out = append(out, originModifiers(origin, toSet(ctx.OriginChoices), r.generalPowerByUid)...)
		}
	}
	return out
}

// evalModifierScale evaluates a Modifier's VitalScale — the escada that makes a
// grant grow with level or with an attribute, instead of being flat.
func evalModifierScale(amount int, scale *VitalScale, level int, attrTotals map[string]int) int {
	if scale == nil || scale.Per == "" || scale.Per == "flat" {
		return amount
	}
	switch scale.Per {
	case "level":
		return amount * level
	case "levelStep":
		steps := level / scale.Step
		if scale.Round == "up" && level%scale.Step != 0 {
			steps++
		}
		return amount * steps
	case "patamar":
		return amount * patamarOf(level)
	default: // attribute
		return amount * attrTotals[scale.Attribute]
	}
}

// OS LIMIARES DE PATAMAR: iniciante, veterano, campeão, lenda.
//
// O livro conta patamar o tempo todo e o motor não o conhecia. Quem o trouxe foi
// o Coração Heroico (p91), que dá "+3 pontos de mana" e mais "+3 PM" a cada
// patamar novo — e o próprio poder IMPRIME os limiares, então eles não vêm de
// outra página.
var patamarThresholds = [...]int{5, 11, 17}

// patamarOf é o ORDINAL do patamar do nível: 1 no iniciante, 4 no lenda.
//
// É ORDINAL e não contagem de patamares ULTRAPASSADOS porque quem escala por ele
// já recebe o valor no 1º nível: o Coração Heroico dá +3 de saída e +3 a cada
// patamar, o que é 3×1, 3×2, 3×3 e 3×4 — um modificador só, em vez de um plano e
// outro escalado.
//
// A variável NÃO se chama como o número que ela representa, e o motivo não é de
// Go: o Tailwind detecta fontes sozinho além dos `@source` da folha — é assim que
// as classes das cenas entram, já que o glob explícito é de um nível só — e a
// varredura alcança os arquivos .go. Um nome de variável que coincida com uma
// utilidade dele (aqui seria a de `font-variant-numeric`) vira classe no bundle, e
// o passo "Fail if the stylesheet was stale" da CI reprova a folha por obsoleta.
//
// Este comentário evita escrever a palavra em minúsculas pela mesma razão — a
// primeira versão dele explicava a armadilha e a disparava de novo.
func patamarOf(level int) int {
	rank := 1
	for _, threshold := range patamarThresholds {
		if level >= threshold {
			rank++
		}
	}
	return rank
}
