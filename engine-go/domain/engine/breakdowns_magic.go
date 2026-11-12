package engine

import (
	"sort"
	"strings"
)

// The magic / carrying / damage-reduction / temp-HP breakdowns. Split from
// breakdowns.go by responsibility: spell + PM economy, carga, RD, PV temporário.

type RdBreakdown struct {
	Total   int            `json:"total"`
	Sources []SourceAmount `json:"sources"`
	// Typed é a RD CONTRA UM TIPO — *"a redução pode ser contra um ou mais
	// tipos de dano específicos"* (p229). Ela não entra no `Total`, e é de
	// propósito: as duas alcançam coisas diferentes, e somá-las daria um número
	// que não vale contra ataque nenhum.
	//
	// LISTA e não mapa, ordenada pelo tipo: o oráculo compara JSON, e mapa em Go
	// não tem ordem.
	Typed []RdByType `json:"typed,omitempty"`
}

// RdByType é a redução contra UM tipo de dano, com de onde ela veio.
type RdByType struct {
	Type    string         `json:"type"`
	Total   int            `json:"total"`
	Sources []SourceAmount `json:"sources"`
}

type TempHpBreakdown struct {
	Total   int            `json:"total"`
	Sources []SourceAmount `json:"sources"`
}

// flySpeedTotal: fly speed from effects (0 = can't fly).
func flySpeedTotal(e ItemEffects) int {
	return max(0, StatFor(e, ModifierTarget{K: "flySpeed"}).Total)
}

// inventorySlotsTotal (p141): 10 +2/Força (or −1/Força negativa)
// + item slot mods.
func inventorySlotsTotal(ch Character, e ItemEffects) int {
	effStr := effectiveAttribute(ch, "strength", e)
	base := 10 + 2*effStr
	if effStr < 0 {
		base = 10 + effStr
	}
	return base + StatFor(e, ModifierTarget{K: "inventorySlots"}).Total
}

// casterLevelForPmLimit (p224): best spellcasting-class level,
// or character level for non-casters.
func casterLevelForPmLimit(ch Character) int {
	best, found := 0, false
	for _, entry := range ch.Classes {
		if !isSpellcastingClass(entry.ClassName) {
			continue
		}
		found = true
		best = max(best, entry.Level)
	}
	if !found {
		return ch.Level
	}
	return best
}

// pmLimitBreakdown is the per-CHARACTER summary behind the HUD's "Limite PM"
// tile: the best level across the character's spellcasting classes. It is a
// SUMMARY, not the cap for any given spell — for that, use SpellPmLimit below,
// which asks which class grants THAT spell.
//
// O BÔNUS é o que alcança a magia `at` (`ForSpell`): o orbe cristalino sobe o limite das arcanas (p160).
// A ficha o calcula com a magia vazia, que é o que vale para toda magia.
func pmLimitBreakdown(ch Character, e ItemEffects, at SpellAt) ValueBreakdown {
	base := max(1, casterLevelForPmLimit(ch))
	reach := e.ForSpell("pmLimit", at)
	bonus := reach.Total
	contributions := reach.Contributions
	// Magia Ilimitada: "soma seu atributo-chave no limite de PM" (p131).
	if attribute, _ := bestSpellcasting(ch, e); attribute != "" && e.Flags["magia-ilimitada"] {
		key := effectiveAttribute(ch, attribute, e)
		bonus += key
		contributions = append(contributions, BreakdownContribution{Source: "Magia Ilimitada", Amount: key})
	}
	total := base + bonus
	// O ritual "dobra seu limite de PM" (p131), e o que dobra é o limite inteiro. A diferença vira linha,
	// para a decomposição continuar fechando na soma.
	if doubled := pmLimitFactor(e).Applied(total); doubled != total {
		contributions = append(contributions, BreakdownContribution{Source: "Celebrar Ritual", Amount: doubled - total})
		total = doubled
	}
	return ValueBreakdown{Base: base, ItemBonus: bonus, Total: total, Contributions: contributions}
}

// pmLimitFactor é o multiplicador do limite de PM, vazio quando nada o multiplica.
func pmLimitFactor(e ItemEffects) Ratio {
	return e.Factors[targetKey(ModifierTarget{K: "pmLimit"})]
}

// bestBaseSpellCd is the best CD across the character's caster classes — p173:
// 10 + metade do nível DO PERSONAGEM + atributo-chave. Uses the FINAL attribute,
// so race and item bonuses count, and resolves the Arcanista's key attribute
// through its Caminho. Nil for non-casters.
func bestBaseSpellCd(ch Character, e ItemEffects) *int {
	_, best := bestSpellcasting(ch, e)
	return best
}

// bestSpellcasting é a classe conjuradora que dá a maior CD: o atributo-chave
// dela e a CD base. Vazio e nulo para quem não conjura por classe.
//
//	bestSpellcasting(samiraFeiticeira, effectsNone()) // "charisma", 19
func bestSpellcasting(ch Character, e ItemEffects) (string, *int) {
	attribute := ""
	var best *int
	for _, entry := range ch.Classes {
		attr := spellcastingAttributeFor(ch, entry.ClassName)
		if attr == "" {
			continue
		}
		dc := spellSaveDc(ch.Level, effectiveAttribute(ch, attr, e))
		if best == nil || dc > *best {
			d := dc
			best, attribute = &d, attr
		}
	}
	return attribute, best
}

// spellCdByAttribute is the per-class CD: the spell
// save CD keyed by casting attribute (10 + ½ nível + FINAL attr mod), so a spell
// row can pick the CD for any of its applicable classes without re-deriving.
func spellCdByAttribute(ch Character, e ItemEffects) map[string]int {
	out := make(map[string]int, len(AttributeKeys))
	for _, a := range AttributeKeys {
		out[a] = spellSaveDc(ch.Level, effectiveAttribute(ch, a, e))
	}
	return out
}

// spellDCBonus: item spellDC bonus + contributions.
func spellDCBonus(e ItemEffects) TotalContribs {
	return totalContribsFor(e, ModifierTarget{K: "spellDC"})
}

// ForSpell é o que chega à magia `at` numa das três contas, com os efeitos com que a ficha foi calculada.
// O mosaico da ficha pergunta só pela tradição da classe que conjura; a linha de cada magia, por ela
// inteira.
//
//	sheet.ForSpell("spellDC", SpellAt{Tradition: "arcana", School: "evocacao"}) // a varinha e o tomo
func (s ComputedSheet) ForSpell(kind string, at SpellAt) SpellReach {
	return s.effects.ForSpell(kind, at)
}

// PmLimitFor é o limite de PM por magia na magia `at`, com a decomposição: o nível de conjurador, o bônus
// que a alcança, a Magia Ilimitada e o ritual. O `PmLimit` da ficha é o mesmo cálculo com a magia vazia.
//
//	sheet.PmLimitFor(SpellAt{Tradition: "arcana"}) // com o orbe cristalino
func (s ComputedSheet) PmLimitFor(at SpellAt) ValueBreakdown {
	return pmLimitBreakdown(s.char, s.effects, at)
}

// pmCostMod é o corte de custo que vale para toda magia, pela regra da p226 (`ForSpell`).
func pmCostMod(e ItemEffects) TotalContribs {
	reach := e.ForSpell("pmCost", SpellAt{})
	return TotalContribs{Total: reach.Total, Contributions: reach.Contributions}
}

// characterDamageReduction é a RD geral do personagem, com a fonte de cada
// parcela.
//
// Toda ela vem de MODIFICADOR (ALE-486): as linhas do Bárbaro (p42), o Bastião
// (p55), a Especialização em Armadura (p54, p65) e a condição Petrificado
// (p394). O empilhamento é o de qualquer número da ficha (p226): habilidades
// diferentes acumulam, e os degraus da MESMA habilidade dividem um `bonusType`
// e disputam — é assim que a RD 10 do Bárbaro no 17º nível não soma com a 8 do
// 14º. Ela morava aqui em Go, lida pelo NOME da classe, e fazia a RD do
// Bárbaro e a do Bastião disputarem — o contrário do que a p226 diz (decisão do
// dono na ALE-486).
func characterDamageReduction(e ItemEffects) RdBreakdown {
	granted := StatFor(e, ModifierTarget{K: "damageReduction"})
	sources := []SourceAmount{}
	for _, c := range granted.Contributions {
		sources = append(sources, SourceAmount{c.Source, c.Amount})
	}
	return RdBreakdown{Total: granted.Total, Sources: sources, Typed: typedDamageReduction(e)}
}

// typedDamageReduction colhe as reduções CONTRA UM TIPO (p229).
//
// Duas fontes do mesmo tipo NÃO somam — vale a maior, como vale para a RD geral
// logo acima: "redução de fogo 10" e "redução de fogo 5" no mesmo personagem
// são a mesma espécie de proteção, e o livro não empilha espécies iguais. Tipos
// diferentes convivem, porque cada um alcança outra parcela.
func typedDamageReduction(e ItemEffects) []RdByType {
	const prefixo = "damageReduction:"
	out := []RdByType{}
	for key, stat := range e.ByTarget {
		tipo, found := strings.CutPrefix(key, prefixo)
		if !found || tipo == "" {
			continue
		}
		entry := RdByType{Type: tipo, Sources: []SourceAmount{}}
		for _, c := range stat.Contributions {
			entry.Total = max(entry.Total, c.Amount)
			entry.Sources = append(entry.Sources, SourceAmount{c.Source, c.Amount})
		}
		if entry.Total > 0 {
			out = append(out, entry)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Type < out[b].Type })
	return out
}

// tempHpFromPowers: Alma de Bronze (Bárbaro p41) grants
// nível + Força temp PV while furia is active.
func tempHpFromPowers(ch Character, e ItemEffects, furyActive bool) TempHpBreakdown {
	empty := TempHpBreakdown{Total: 0, Sources: []SourceAmount{}}
	if !furyActive {
		return empty
	}
	chosen := parseChoiceSet(ch.ClassPowers)
	owns := false
	for _, id := range chosen.list {
		if id == "alma-de-bronze" || strings.HasSuffix(id, ".alma-de-bronze") {
			owns = true
			break
		}
	}
	if !owns {
		return empty
	}
	amount := ch.Level + effectiveAttribute(ch, "strength", e)
	return TempHpBreakdown{Total: amount, Sources: []SourceAmount{{"Alma de Bronze (Fúria, p41)", amount}}}
}

// SpellPmLimit is the p224 ceiling for ONE spell: the character's level in the
// CLASS that grants it, or the character level when the source is not a class
// (a race, an origin, a general power). `itemBonus` must be the RESOLVED
// `pmLimit` total (`ComputedSheet.PmLimit.ItemBonus`) — never a raw sum over
// equipped modifiers, which double-counts two `item`-typed bonuses and honours a
// `wielded` condition on a merely vested item.
//
// This is the authority the cast gate hangs on, and it is NOT what
// `pmLimitBreakdown` above returns: that one is the per-CHARACTER HUD summary
// ("best caster level"), a defensible summary for a tile but the wrong number
// for a specific spell. Two implementations of this cap disagree, and then the
// sheet offers what the server refuses.
//
// @example SpellPmLimit(bardo7Arcanista1, 0, []string{"Arcanista"}) // 1
func SpellPmLimit(ch Character, itemBonus int, spellClasses []string) int {
	grants := map[string]bool{}
	for _, name := range spellClasses {
		grants[name] = true
	}
	best := 0
	for _, entry := range ch.Classes {
		if grants[entry.ClassName] && entry.Level > best {
			best = entry.Level
		}
	}
	if best == 0 {
		best = ch.Level // race/origin/power grant: character level
	}
	return max(1, best) + itemBonus
}

// SpellPmLimitFor resolves the item bonus off the character's own effects and
// applies the p224 rule — the one call a caller should make, so nobody
// re-derives the bonus its own way.
//
// # Os condicionais vêm por PARÂMETRO, e antes não vinham
//
// Ela recomputava a ficha INTEIRA daqui de dentro, com `map[string]bool{}` no
// lugar dos condicionais — uma re-entrada no pipeline com entrada diferente da
// de quem chamou. O efeito medido: o `kharvos-o-guardiao-rubro` carrega um
// Medalhão de prata vestido cujo `pmLimit +1` está atrás de um condicional que
// ele tem LIGADO, e a ficha dele oferecia limite 8 enquanto o servidor recusava
// acima de 7 (ALE-357).
//
// O aviso já estava escrito no bloco do `SpellPmLimit` logo acima — *"two
// implementations of this cap disagree, and then the sheet offers what the
// server refuses"* —, sobre OUTRA divergência. Esta era a terceira.
//
// Hoje ela tem a mesma forma do `SpellPmCostFor` abaixo: resolve os efeitos com
// os condicionais recebidos e pergunta ao MESMO `pmLimitBreakdown` que a ficha
// usa, em vez de reconstruir a conta.
//
// A MAGIA escolhe o bônus que vale só nela (`ForSpell`): o orbe cristalino sobe o limite das arcanas
// (p160), e a divina do mesmo personagem fica no dela.
func (r *Ruleset) SpellPmLimitFor(
	ch Character, conditionals map[string]bool, spellClasses []string, at SpellAt,
) int {
	effects := r.EffectsOf(ch, conditionals)
	bonus := pmLimitBreakdown(ch, effects, at).ItemBonus
	return pmLimitFactor(effects).Applied(SpellPmLimit(ch, bonus, spellClasses))
}

// pmCostDispute aplica a regra da p226 às contribuições de custo de PM, e diz quais fontes perderam:
//
//	"Reduções de Custo. Reduções no custo de PM não são cumulativas."
//
// REDUÇÕES competem entre FONTES — a melhor vence — mas somam dentro de uma
// fonte só, porque "os bônus dobram" da Força da Natureza (p63) é UMA habilidade
// se duplicando, não duas se acumulando. O `resolveStack` comum não serve aqui:
// ele compete por bonusType, e estas vêm todas como `untyped`.
//
// AS EXCEÇÕES são do verbete, e o livro escreve duas formas:
//
//   - "cumulativo com Mestre em Escola" (cajado do poder, p337): a parcela diz o id da outra fonte
//     (`StacksWith`), e as fontes LIGADAS somam e contam como uma redução só, que disputa com as outras;
//   - "cumulativo com outras reduções de custo" (Foco em Magia, p131): a parcela acumula com tudo
//     (`StacksWithAll`), e entra por cima da que venceu.
//
// O livro não tem exemplo de três reduções juntas; a leitura da dupla que disputa inteira é do dono.
//
// AUMENTOS (o Alquebrado, p394: "o custo em PM das habilidades aumenta em +1")
// somam normalmente — o livro só proíbe o acúmulo das reduções. As fontes de
// redução que não venceram voltam à parte: a conta da ficha as mostra riscadas.
func pmCostDispute(contribs []Contribution) (int, map[string]bool) {
	type cut struct {
		amount     int
		id         string
		stacksWith []string
		withAll    bool
	}
	bySource := map[string]*cut{}
	order := []string{}
	for _, c := range contribs {
		source, seen := bySource[c.Source]
		if !seen {
			source = &cut{id: c.SourceID}
			bySource[c.Source] = source
			order = append(order, c.Source)
		}
		source.amount += c.Amount
		source.stacksWith = append(source.stacksWith, c.StacksWith...)
		source.withAll = source.withAll || c.StacksWithAll
	}

	// A LIGAÇÃO vale nos dois sentidos — o cajado a declara, e o Mestre em Escola não precisa repeti-la —,
	// e por isso as fontes ligadas se juntam num grupo, e não numa lista de pares.
	group := map[string]string{}
	var root func(string) string
	root = func(source string) string {
		if group[source] == source {
			return source
		}
		group[source] = root(group[source])
		return group[source]
	}
	byID := map[string]string{}
	onTop, increases := 0, 0
	for _, source := range order {
		c := bySource[source]
		switch {
		case c.amount > 0:
			increases += c.amount
		case c.amount < 0 && c.withAll:
			onTop += c.amount
		case c.amount < 0:
			group[source] = source
			if c.id != "" {
				byID[c.id] = source
			}
		}
	}
	for _, source := range order {
		if _, disputes := group[source]; !disputes {
			continue
		}
		for _, id := range bySource[source].stacksWith {
			if partner, has := byID[id]; has {
				group[root(partner)] = root(source)
			}
		}
	}

	sums, best := map[string]int{}, ""
	for _, source := range order {
		if _, disputes := group[source]; disputes {
			sums[root(source)] += bySource[source].amount
		}
	}
	for _, source := range order {
		if _, disputes := group[source]; disputes && (best == "" || sums[root(source)] < sums[best]) {
			best = root(source)
		}
	}
	losers := map[string]bool{}
	for _, source := range order {
		if _, disputes := group[source]; disputes && root(source) != best {
			losers[source] = true
		}
	}
	return sums[best] + onTop + increases, losers
}

// SpellPmCostFor é o que a magia CUSTA de fato: o valor da Tabela 4-1 mais os
// aprimoramentos, com os modificadores de custo da p226 aplicados e o piso de
// 1 PM ("uma habilidade nunca pode ter seu custo reduzido para menos de 1 PM").
//
// O truque continua de graça: ele custa zero por natureza, não por redução, e o
// piso não alcança o que já era zero.
//
// É ela que o PORTÃO de conjurar tem de chamar, e não o valor bruto da tabela:
// calcular o modificador só para a ficha deixa o mosaico "Custo PM" dizendo −2
// enquanto a conjuração cobra preço cheio.
//
// O que alcança a magia é o `ForSpell`, com ela inteira: o cajado do poder corta as arcanas (p337), o
// medalhão de prata as de alcance pessoal (p160), o Foco em Magia a magia escolhida (p131), e todos
// entram na MESMA disputa da p226, com as exceções que o verbete escreve (`pmCostDispute`).
//
// @example SpellPmCostFor(druida20, 3, 0, nil, SpellAt{Tradition: "divina", Spell: "bola-de-fogo"}) // 1
func (r *Ruleset) SpellPmCostFor(
	ch Character, basePm, augmentPm int, conditionals map[string]bool, at SpellAt,
) int {
	raw := basePm + augmentPm
	if raw <= 0 {
		return 0
	}
	reach := r.EffectsOf(ch, conditionals).ForSpell("pmCost", at)
	mod := reach.Total
	// O FATOR É O ÚLTIMO, e o livro diz a ordem por extenso: a Alta Arcana
	// reduz à metade "após aplicar aprimoramentos e quaisquer outros efeitos
	// que reduzam custo" (p39). Antes da soma, o +1 do Alquebrado entraria
	// depois do corte e a magia de 6 PM sairia por 4 em vez de 3.
	//
	// E o PISO continua por último de todos: metade de 1 PM é zero, e a p226
	// não deixa descer abaixo de 1.
	return max(1, reach.Factor.Applied(raw+mod))
}
