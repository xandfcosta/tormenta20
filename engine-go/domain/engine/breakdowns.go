package engine

// A camada de DECOMPOSIÇÃO: ela transforma os `ItemEffects` resolvidos num
// `ComputedSheet` em que nenhum número viaja sozinho — cada um leva as
// contribuições que o formaram.
//
// Isso é decisão de produto e não de engenharia: a ficha tem de responder "por
// que 12?" com a soma inteira na tela.
//
// Deslocamento, defesa, atributo e perícia moram aqui; magia, RD e PV
// temporários estão no `breakdowns_magic.go`.
//
// Oráculo de paridade: `engine-go/parity/<slug>.json`, chave `sheet`.

// BreakdownContribution é uma linha {source, amount, note?} — a contribuição na
// forma que a tela desenha. Ela NÃO carrega `bonusType`, ao contrário da
// `Contribution` da resolução: o empilhamento já foi decidido antes. Nota vazia
// é omitida para nunca viajar no fio.
type BreakdownContribution struct {
	Source string `json:"source"`
	Amount int    `json:"amount"`
	Note   string `json:"note,omitempty"`
}

// SourceAmount é uma linha {source, amount} — contribuição de atributo, de RD e
// de PV temporário, que não carrega nota.
type SourceAmount struct {
	Source string `json:"source"`
	Amount int    `json:"amount"`
}

type DefenseBreakdown struct {
	Base       int  `json:"base"`
	ItemBonus  int  `json:"itemBonus"`
	Total      int  `json:"total"`
	DexApplied bool `json:"dexApplied"`
	// Defesa contra ataques corpo a corpo e à distância. Iguais ao Total na
	// maioria das fichas; separam-se quando algo é DIRECIONAL, hoje só o Caído
	// (p394: −5 contra corpo a corpo, +5 contra à distância).
	VsMelee       int                     `json:"vsMelee"`
	VsRanged      int                     `json:"vsRanged"`
	Contributions []BreakdownContribution `json:"contributions"`
}

// ValueBreakdown é a forma {base, itemBonus, total, contributions} compartilhada
// pelo deslocamento e pelo limite de PM.
type ValueBreakdown struct {
	Base          int                     `json:"base"`
	ItemBonus     int                     `json:"itemBonus"`
	Total         int                     `json:"total"`
	Contributions []BreakdownContribution `json:"contributions"`
}

// TotalContribs é a forma {total, contributions} (spellDCBonus, pmCostMod).
type TotalContribs struct {
	Total         int                     `json:"total"`
	Contributions []BreakdownContribution `json:"contributions"`
}

type AttributeBreakdown struct {
	Total         int            `json:"total"`
	Contributions []SourceAmount `json:"contributions"`
}

type ExpertiseBreakdown struct {
	Name                string                  `json:"name"`
	Attribute           string                  `json:"attribute"`
	Base                int                     `json:"base"`
	ItemBonus           int                     `json:"itemBonus"`
	Total               int                     `json:"total"`
	HalfLevel           int                     `json:"halfLevel"`
	AttrValue           int                     `json:"attrValue"`
	Training            int                     `json:"training"`
	ItemContributions   []BreakdownContribution `json:"itemContributions"`
	ArmorPenaltyApplied int                     `json:"armorPenaltyApplied"`
}

// ComputedSheet junta todas as decomposições — a ficha rica que as cenas
// desenham, em que cada número chega com as contribuições que o formaram.
type ComputedSheet struct {
	Defense      DefenseBreakdown `json:"defense"`
	Displacement ValueBreakdown   `json:"displacement"`
	FlySpeed     int              `json:"flySpeed"`
	// Load é a p141 inteira — os espaços ocupados, o limite e a sobrecarga.
	Load            LoadBreakdown                 `json:"carga"`
	Attributes      map[string]AttributeBreakdown `json:"attributes"`
	PmLimit         ValueBreakdown                `json:"pmLimit"`
	BestBaseSpellCd *int                          `json:"bestBaseSpellCd"`
	// SpellCdByAttribute is the spell save CD keyed by casting attribute (p173),
	// so a spell row can pick the CD for any of its applicable classes without
	// re-deriving.
	SpellCdByAttribute map[string]int `json:"spellCdByAttribute"`
	SpellDCBonus       TotalContribs  `json:"spellDCBonus"`
	PmCostMod          TotalContribs  `json:"pmCostMod"`
	// AttackAll/DamageAll are the {k:attack|damage, scope:all} globals (Fúria,
	// Instinto Selvagem…) — the combat HUD adds them onto every weapon/attack.
	AttackAll       TotalContribs `json:"attackAll"`
	DamageAll       TotalContribs `json:"damageAll"`
	DamageReduction RdBreakdown   `json:"damageReduction"`
	// TempHpFury is tempHpFromPowers with furia active — the interesting branch
	// (Alma de Bronze). The base sheet (furia off) is always {0, []}.
	TempHpFury TempHpBreakdown      `json:"tempHpFuria"`
	Expertises []ExpertiseBreakdown `json:"expertises"`
	// Perícias em que o personagem FALHA AUTOMATICAMENTE — hoje só Reflexos, do
	// Indefeso (p394). É o motor quem responde isso, e não a UI reinterpretando
	// uma flag: a regra de quais condições implicam indefeso mora aqui.
	AutoFailExpertises []string `json:"autoFailExpertises"`
}

// ComputeSheet monta a ficha decomposta de um `Character` cru sob os
// condicionais ligados — o caminho coleta → resolução → decomposição.
func (c *Catalogs) ComputeSheet(ch Character, activeConditionals map[string]bool) ComputedSheet {
	effects := ApplyActiveConditionals(ComputeItemEffects(c.ActiveItemsFor(ch)), activeConditionals)
	load := loadBreakdownOf(ch, inventorySlotsTotal(ch, effects))

	attrs := make(map[string]AttributeBreakdown, len(AttributeKeys))
	for _, a := range AttributeKeys {
		attrs[a] = attributeBreakdown(ch, a, effects)
	}
	expertises := []ExpertiseBreakdown{}
	for _, ex := range ch.Expertises {
		expertises = append(expertises, expertiseBreakdown(ch, ex, effects, load))
	}

	return ComputedSheet{
		Defense:            defenseBreakdown(ch, effects),
		Displacement:       displacementBreakdown(c.raceDisplacement(ch), effects, load),
		FlySpeed:           flySpeedTotal(effects),
		Load:               load,
		Attributes:         attrs,
		PmLimit:            pmLimitBreakdown(ch, effects),
		BestBaseSpellCd:    bestBaseSpellCd(ch, effects),
		SpellCdByAttribute: spellCdByAttribute(ch, effects),
		SpellDCBonus:       spellDCBonus(effects),
		PmCostMod:          pmCostMod(effects),
		AttackAll:          totalContribsFor(effects, ModifierTarget{K: "attack", Scope: "all"}),
		DamageAll:          totalContribsFor(effects, ModifierTarget{K: "damage", Scope: "all"}),
		DamageReduction:    characterDamageReduction(ch, effects),
		TempHpFury:         tempHpFromPowers(ch, effects, true),
		Expertises:         expertises,
		AutoFailExpertises: autoFailExpertises(effects),
	}
}

// effectiveAttribute: o atributo cru mais os modificadores de `attribute`.
func effectiveAttribute(ch Character, attr string, e ItemEffects) int {
	return ch.attributeValue(attr) + StatFor(e, ModifierTarget{K: "attribute", Name: attr}).Total
}

// defenseBreakdown: 10 + Destreza (quando ela se aplica) + modificadores.
func defenseBreakdown(ch Character, e ItemEffects) DefenseBreakdown {
	stat := StatFor(e, ModifierTarget{K: "defense"})
	dexApplied := !e.Flags["cannot-apply-dex-to-defense"]
	base := 10
	if dexApplied {
		base += effectiveAttribute(ch, "dexterity", e)
	}
	insolence := insolenciaDefense(ch, e)
	melee := StatFor(e, ModifierTarget{K: "defense", Scope: "melee"})
	ranged := StatFor(e, ModifierTarget{K: "defense", Scope: "ranged"})
	total := base + stat.Total + insolence
	contribs := stat.Contributions
	if insolence > 0 {
		contribs = concatContribs(contribs, []Contribution{
			{Source: "Insolência (p47)", BonusType: "untyped", Amount: insolence},
		})
	}
	return DefenseBreakdown{
		Base:          base,
		ItemBonus:     stat.Total + insolence,
		Total:         total,
		DexApplied:    dexApplied,
		VsMelee:       total + melee.Total,
		VsRanged:      total + ranged.Total,
		Contributions: withNoteContribs(concatContribs(contribs, melee.Contributions, ranged.Contributions)),
	}
}

// insolenciaDefense — Bucaneiro p47: "Você soma seu Carisma na Defesa, limitado
// pelo seu nível. Esta habilidade exige liberdade de movimentos; você não pode
// usá-la se estiver de armadura pesada ou na condição imóvel."
//
// O teto é o nível NA CLASSE (p226, "Limites de Nível"), com exemplo trabalhado
// na mesma página: "um bucaneiro de 2º nível com Car 3 soma +2 na Defesa".
//
// Vive aqui, e não como modificador de catálogo, porque o motor de itens não
// avalia `scale` fora de PV/PM e não tem noção de TETO — a Insolência precisa
// das duas coisas.
func insolenciaDefense(ch Character, e ItemEffects) int {
	if e.Flags["armadura-pesada"] || hasActiveCondition(ch, "imovel") {
		return 0
	}
	level := 0
	for _, entry := range ch.Classes {
		if entry.ClassName == "Bucaneiro" && entry.Level > level {
			level = entry.Level
		}
	}
	if level == 0 {
		return 0
	}
	return max(0, min(effectiveAttribute(ch, "charisma", e), level))
}

// hasActiveCondition reporta se a condição está ligada na ficha.
func hasActiveCondition(ch Character, id string) bool {
	for _, active := range parseStringArray(ch.ActiveConditions) {
		if active == id {
			return true
		}
	}
	return false
}

// displacementBreakdown é o deslocamento total (com piso em 0) mais a
// sobrecarga: −3m enquanto a mochila passa do limite (p141). Ela entra como
// contribuição NOMEADA porque um deslocamento que cai sem dizer por quê é lido
// como defeito.
// displacementIgnoresArmorAndLoad é o sinal que o "Devagar e Sempre" do anão
// (p20) pendura. Ele vive no catálogo da raça, como todo modificador de raça —
// não há um `if raça == "Anão"` no motor, e não deve haver: a próxima raça com
// a mesma isenção entra pelo catálogo.
const displacementIgnoresArmorAndLoad = "displacement-ignores-armor-and-load"

// bookDefaultDisplacement é o deslocamento de quem o catálogo não conhece.
//
// Nove metros é o padrão do livro, do qual as raças que fogem dizem fugir com
// todas as letras ("é 6m EM VEZ DE 9m", p20). Cair aqui é o catálogo não
// conhecer a raça — e o remédio é transcrevê-la, não consultar a coluna.
const bookDefaultDisplacement = 9

// raceDisplacement é o deslocamento da RAÇA PRIMÁRIA.
//
// Primária e não a soma das raças: o número do verbete é um valor, não um
// bônus, e duas raças não andam somando metros. A secundária contribui
// habilidade, não deslocamento.
func (c *Catalogs) raceDisplacement(ch Character) int {
	if len(ch.Races) == 0 {
		return bookDefaultDisplacement
	}
	entry := c.raceEntryByName(ch.Races[0].Race)
	if entry == nil || entry.Speed == 0 {
		return bookDefaultDisplacement
	}
	return entry.Speed
}

// displacementBreakdown: o deslocamento da raça, mais o que modifica.
//
// A BASE VEM DA RAÇA E NÃO DA COLUNA (ALE-383). `characters.displacement` era
// um espelho escrito no nascimento, e ele divergiu: a seed tinha DOIS anões com
// 9m e um com 6m, com o livro dizendo 6m na p20 para os três.
func displacementBreakdown(base int, e ItemEffects, load LoadBreakdown) ValueBreakdown {
	// DEVAGAR E SEMPRE (p20): "seu deslocamento não é reduzido por uso de
	// armadura ou excesso de carga". São DUAS fontes e elas chegam por caminhos
	// diferentes — a armadura por modificador de catálogo, a carga pela conta de
	// espaços —, então a isenção aparece duas vezes aqui. O que ela NÃO isenta é
	// qualquer outra redução: uma magia de lentidão continua valendo.
	exempt := e.Flags[displacementIgnoresArmorAndLoad]

	stat := StatFor(e, ModifierTarget{K: "displacement"})
	contribs := withNoteContribs(stat.Contributions)
	bonus := stat.Total
	if armor := StatFor(e, ModifierTarget{K: "displacement", Scope: "armor"}); !exempt {
		bonus += armor.Total
		contribs = append(contribs, withNoteContribs(armor.Contributions)...)
	}
	if load.DisplacementPenalty != 0 && !exempt {
		bonus += load.DisplacementPenalty
		contribs = append(contribs, overloadContrib(load.DisplacementPenalty))
	}
	return ValueBreakdown{
		Base:          base,
		ItemBonus:     bonus,
		Total:         max(0, base+bonus),
		Contributions: contribs,
	}
}

// overloadContrib nomeia a fonte uma vez só — ela aparece no deslocamento e
// nas três perícias de penalidade de armadura, e duas grafias diferentes da
// mesma penalidade leriam como duas penalidades.
func overloadContrib(amount int) BreakdownContribution {
	return BreakdownContribution{Source: "Sobrecarga (p141)", Amount: amount}
}

// attributeBreakdown é o total do atributo com as contribuições dele, na forma
// {source, amount} — a nota não vai junto.
func attributeBreakdown(ch Character, attr string, e ItemEffects) AttributeBreakdown {
	stat := StatFor(e, ModifierTarget{K: "attribute", Name: attr})
	return AttributeBreakdown{
		Total:         ch.attributeValue(attr) + stat.Total,
		Contributions: sourceAmountContribs(stat.Contributions),
	}
}

var armorPenaltyExpertises = map[string]bool{"Acrobacia": true, "Furtividade": true, "Ladinagem": true}

// expertiseBreakdown: ½ nível + atributo + treino + modificadores de item
// (expertise/expertiseAll/expertiseByAttribute) + penalidade de armadura, que
// tem duas fontes: a armadura vestida e a sobrecarga.
func expertiseBreakdown(ch Character, state CharacterExpertise, e ItemEffects, load LoadBreakdown) ExpertiseBreakdown {
	halfLevel := ch.Level / 2
	attrValue := effectiveAttribute(ch, state.Attribute, e)
	training := 0
	if state.Trained {
		training = trainingBonusForLevel(ch.Level)
	}
	base := halfLevel + attrValue + training

	stat := StatFor(e, ModifierTarget{K: "expertise", Name: state.Name})
	allStat := StatFor(e, ModifierTarget{K: "expertiseAll"})
	byAttrStat := StatFor(e, ModifierTarget{K: "expertiseByAttribute", Attribute: state.Attribute})
	merged := resolveStack(concatContribs(stat.Contributions, allStat.Contributions, byAttrStat.Contributions))
	itemContribs := withNoteContribs(merged.Contributions)

	armorPenaltyApplied := 0
	if armorPenaltyExpertises[state.Name] {
		for _, row := range armorPenaltyContribs(e, load) {
			armorPenaltyApplied += row.Amount
			itemContribs = append(itemContribs, row)
		}
	}

	itemBonus := merged.Total
	return ExpertiseBreakdown{
		Name:                state.Name,
		Attribute:           state.Attribute,
		Base:                base,
		ItemBonus:           itemBonus + armorPenaltyApplied,
		Total:               base + itemBonus + armorPenaltyApplied,
		HalfLevel:           halfLevel,
		AttrValue:           attrValue,
		Training:            training,
		ItemContributions:   itemContribs,
		ArmorPenaltyApplied: armorPenaltyApplied,
	}
}

// armorPenaltyContribs são as DUAS fontes de penalidade de armadura que a ficha
// conhece: a das peças vestidas (p153, "Aplique a penalidade de armadura em
// testes de Acrobacia, Furtividade e Ladinagem") e a sobrecarga, que a p141
// descreve com essas mesmas palavras — "sofre penalidade de armadura –5". São
// linhas separadas de propósito: somadas numa só, o jogador não descobre que
// metade dela some ao largar peso.
func armorPenaltyContribs(e ItemEffects, load LoadBreakdown) []BreakdownContribution {
	out := []BreakdownContribution{}
	if item := StatFor(e, ModifierTarget{K: "armorPenalty"}).Total; item != 0 {
		out = append(out, BreakdownContribution{Source: "Penalidade de armadura", Amount: item})
	}
	if load.ArmorPenalty != 0 {
		out = append(out, overloadContrib(load.ArmorPenalty))
	}
	return out
}

// totalContribsFor is the {total, contributions} shape for a single target
// (spellDC, pmCost, attack/damage globals).
func totalContribsFor(e ItemEffects, target ModifierTarget) TotalContribs {
	stat := StatFor(e, target)
	return TotalContribs{Total: stat.Total, Contributions: withNoteContribs(stat.Contributions)}
}

// withNoteContribs maps resolution Contributions to display rows, keeping note.
func withNoteContribs(cs []Contribution) []BreakdownContribution {
	out := []BreakdownContribution{}
	for _, c := range cs {
		bc := BreakdownContribution{Source: c.Source, Amount: c.Amount}
		if c.Note != "" {
			bc.Note = c.Note
		}
		out = append(out, bc)
	}
	return out
}

// sourceAmountContribs maps to {source, amount}, dropping note (attributes).
func sourceAmountContribs(cs []Contribution) []SourceAmount {
	out := []SourceAmount{}
	for _, c := range cs {
		out = append(out, SourceAmount{Source: c.Source, Amount: c.Amount})
	}
	return out
}

func concatContribs(lists ...[]Contribution) []Contribution {
	out := []Contribution{}
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

// autoFailExpertises lista as perícias em que o personagem falha
// AUTOMATICAMENTE. Hoje só Reflexos, pelo Indefeso (p394) e por tudo que o livro
// define COMO indefeso — paralisado, inconsciente, petrificado.
//
// Devolve sempre uma lista (nunca nil) para o JSON trazer `[]` em vez de `null`.
func autoFailExpertises(e ItemEffects) []string {
	out := []string{}
	if e.Flags[autoFailReflexosFlag] {
		out = append(out, "Reflexos")
	}
	return out
}
