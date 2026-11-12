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
	Level              int          `json:"level"`
	Classes            []ClassEntry `json:"classes"`
	RaceID             string       `json:"raceId"`
	RaceAbilityChoices []string     `json:"raceAbilityChoices"`
	// RaceAncestry é a ascendência escolhida (o suraggel, p31): a habilidade
	// presa a uma ascendência só vale nela (ALE-505).
	RaceAncestry  string                           `json:"raceAncestry,omitempty"`
	PowerIDs      []string                         `json:"powerIds"`
	ClassChoices  map[string]ClassChoiceSelections `json:"classChoices"`
	GodPowers     []string                         `json:"godPowers"`
	Origin        string                           `json:"origin"`
	OriginChoices []string                         `json:"originChoices"`
	AttrTotals    map[string]int                   `json:"attrTotals"`
	// ItemGrants são os itens vestidos que dão PV ou PM (os anéis, p342), já com o não
	// empilhamento. Moram à parte porque não pertencem a nível nenhum.
	ItemGrants []vitalGrant `json:"-"`
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

// VitalLevelPart é uma parcela da conta de um nível. Source vazio é a classe, e
// Attribute diz que a parcela é aquele atributo, que é como a mesa a chama.
type VitalLevelPart struct {
	Source    string
	Attribute string
	Amount    int
}

// VitalLevelLine é a conta de um nível. Floored é o "pelo menos 1 PV" (p35): as
// parcelas somam menos que o Total.
//
// A linha de um item não é de nível nenhum: Level é zero, Source é o item e Note é o
// que o livro pede dele.
type VitalLevelLine struct {
	Level     int
	ClassName string
	Source    string
	Note      string
	Parts     []VitalLevelPart
	Floored   bool
	Total     int
}

// VitalLevels são as contas nível a nível dos dois poços.
type VitalLevels struct {
	PV []VitalLevelLine
	PM []VitalLevelLine
}

// VitalLevelsFor abre o PV e o PM máximos nível a nível, para a tela mostrar a conta.
//
// classByLevel é a classe de cada nível, na ordem em que foram pegos. Quando o registro
// não bate com as classes, vale a ordem delas: é ela que o `multiclassPvPool` usa, e uma
// conta em outra ordem não fecharia com o total.
//
// O QUE OS PODERES DÃO cai no nível em que a regra soma, pela DIFERENÇA entre um nível e
// o anterior. O livro dá o total e não o histórico: o poder que cresce com o nível é
// retroativo, então a linha mostra o efeito, e não quando o poder foi escolhido. É a
// diferença que faz a soma fechar por construção, seja qual for a escala.
//
//	world.VitalLevelsFor(ctx, []string{"Guerreiro", "Guerreiro"}).PV // [{Total: 24}, {Total: 9}]
func (r *Ruleset) VitalLevelsFor(ctx VitalContext, classByLevel []string) VitalLevels {
	order := levelOrderOf(ctx.Classes, classByLevel)
	con := ctx.AttrTotals["constitution"]
	grants := r.vitalGrants(ctx)
	out := VitalLevels{}
	for i, className := range order {
		level := i + 1
		pv := VitalLevelLine{Level: level, ClassName: className}
		pm := VitalLevelLine{Level: level, ClassName: className}
		if entry, known := classVitalsTable[className]; known {
			base := entry.pvPerLevel
			if i == 0 {
				base = entry.startHP
			}
			pv.Parts = append(pv.Parts, VitalLevelPart{Amount: base})
			if con != 0 {
				pv.Parts = append(pv.Parts, VitalLevelPart{Attribute: "constitution", Amount: con})
			}
			pv.Total = base + con
			// "Você sempre ganha pelo menos 1 PV ao subir de nível" (p35): o 1º nível
			// não é subida, e é por isso que o piso não vale nele.
			if i > 0 && pv.Total < 1 {
				pv.Total, pv.Floored = 1, true
			}
			pm.Parts = append(pm.Parts, VitalLevelPart{Amount: entry.mpPerLevel})
			pm.Total = entry.mpPerLevel
		}
		for _, grant := range grants {
			amount := evalModifierScale(grant.Amount, grant.Scale, level, ctx.AttrTotals)
			if i > 0 {
				amount -= evalModifierScale(grant.Amount, grant.Scale, level-1, ctx.AttrTotals)
			}
			if amount == 0 {
				continue
			}
			part := VitalLevelPart{Source: grant.Source, Amount: amount}
			if grant.Scale != nil && grant.Scale.Per == "attribute" {
				part.Attribute = grant.Scale.Attribute
			}
			line := &pm
			if grant.Target.K == "maxPv" {
				line = &pv
			}
			line.Parts = append(line.Parts, part)
			line.Total += amount
		}
		out.PV = append(out.PV, pv)
		out.PM = append(out.PM, pm)
	}
	for _, item := range ctx.ItemGrants {
		amount := evalModifierScale(item.Amount, item.Scale, ctx.Level, ctx.AttrTotals)
		line := VitalLevelLine{Source: item.Source, Note: item.Note, Total: amount}
		if item.Target.K == "maxPv" {
			out.PV = append(out.PV, line)
		} else {
			out.PM = append(out.PM, line)
		}
	}
	return out
}

// levelOrderOf é a classe de cada nível: o registro, se ele bater com as classes em
// quantidade e na primeira, ou as classes em sequência.
func levelOrderOf(classes []ClassEntry, classByLevel []string) []string {
	fromClasses := []string{}
	left := map[string]int{}
	for _, entry := range classes {
		left[entry.ClassName] += entry.Level
		for range entry.Level {
			fromClasses = append(fromClasses, entry.ClassName)
		}
	}
	if len(classByLevel) != len(fromClasses) || len(classByLevel) == 0 || classByLevel[0] != fromClasses[0] {
		return fromClasses
	}
	for _, className := range classByLevel {
		left[className]--
		if left[className] < 0 {
			return fromClasses
		}
	}
	return classByLevel
}

// itemVitalGrants são os modificadores de PV e PM dos itens que estão valendo.
//
// "Dois itens [...] que forneçam +1 em Fortitude não terá +2 — como os efeitos são da
// mesma fonte, não acumulam" (p226), então dois Anéis da Vitalidade valem um. O corte é aqui porque estes modificadores não passam pela pilha do
// `resolveStack`, que é de quem tem `ItemEffects`.
func (r *Ruleset) itemVitalGrants(ch Character) []vitalGrant {
	items := ch.Items
	best := map[string]int{}
	out := []vitalGrant{}
	for _, it := range items {
		if it.CatalogID == nil {
			continue
		}
		catalog := r.getCatalogItem(*it.CatalogID)
		if catalog == nil {
			continue
		}
		wear := r.wornWear(ch, it)
		for _, m := range r.ownItemMods(it, catalog, devotionOf(ch)) {
			if (m.Target.K != "maxPv" && m.Target.K != "maxPm") || !conditionMet(m, wear) {
				continue
			}
			key := m.Target.K + ":" + m.BonusType
			at, seen := best[key]
			if !seen {
				best[key] = len(out)
				out = append(out, vitalGrant{Modifier: m, Source: catalog.Name})
			} else if m.Amount > out[at].Amount {
				out[at] = vitalGrant{Modifier: m, Source: catalog.Name}
			}
		}
	}
	return out
}

// vitalGrant é um modificador de PV ou PM, com o nome de quem o concede.
type vitalGrant struct {
	Modifier
	Source string
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
	for _, grant := range append(r.vitalGrants(ctx), ctx.ItemGrants...) {
		amount := evalModifierScale(grant.Amount, grant.Scale, ctx.Level, ctx.AttrTotals)
		if grant.Target.K == "maxPv" {
			pv += amount
		} else {
			pm += amount
		}
	}
	return pv, pm
}

// vitalGrants são os modificadores de PV e PM que valem, já com o corte da p226.
func (r *Ruleset) vitalGrants(ctx VitalContext) []vitalGrant {
	seen := map[string]bool{}
	out := []vitalGrant{}
	for _, grant := range r.vitalGrantMods(ctx) {
		if grant.Target.K != "maxPv" && grant.Target.K != "maxPm" {
			continue
		}
		if grant.Scale != nil && grant.Scale.Per == "attribute" {
			key := grant.Target.K + ":" + grant.Scale.Attribute
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		out = append(out, grant)
	}
	return out
}

// vitalGrantMods collects every modifier from the abilities
// the character owns — race, class powers (per class at ITS level), general
// powers, the god power, and origin benefits. NOT items/activeEffects (those
// aren't vital-grant sources). Filtered to vital targets by the caller.
func (r *Ruleset) vitalGrantMods(ctx VitalContext) []vitalGrant {
	out := []vitalGrant{}
	add := func(source string, mods []Modifier) {
		for _, m := range mods {
			out = append(out, vitalGrant{Modifier: m, Source: source})
		}
	}
	powers := toSet(ctx.PowerIDs)

	if ctx.RaceID != "" {
		if race := r.getRace(ctx.RaceID); race != nil {
			add(race.Name, raceModifiers(race, toSet(ctx.RaceAbilityChoices), ctx.RaceAncestry))
		}
	}
	for _, ce := range ctx.Classes {
		for _, power := range r.ownedClassPowers(ce.ClassName, ce.Level, powers, ctx.ClassChoices[ce.ClassName]) {
			add(power.Name, power.Modifiers)
		}
	}
	for _, id := range ctx.PowerIDs {
		if gp := r.getGeneralPower(id); gp != nil {
			add(gp.Name, gp.Modifiers)
		}
	}
	for _, name := range ctx.GodPowers {
		if gp := r.grantedPowerByName(name); gp != nil {
			add(gp.Name, gp.Modifiers)
		}
	}
	if ctx.Origin != "" {
		if origin := r.getOrigin(ctx.Origin); origin != nil {
			add(origin.Name, originModifiers(origin, toSet(ctx.OriginChoices), r.generalPowerByUid))
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
