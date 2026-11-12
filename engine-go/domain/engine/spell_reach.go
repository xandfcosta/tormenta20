package engine

import "sort"

// O QUE VALE PARA UMA MAGIA, numa pergunta só.
//
// Três contas da magia dependem de modificador: a CD (`spellDC`), o limite de PM (`pmLimit`) e o custo
// (`pmCost`). Cada modificador delas diz o que alcança — a tradição (os esotéricos arcanos, p160), a
// escola (o tomo hermético), o alcance (o medalhão de prata) —, e vazio é "todas". A ficha, a lista de
// magias, a conjuração e as notas da tela perguntam isto, e nenhuma delas junta o geral com o específico
// por conta própria: havia quatro jeitos de juntar, e o −1 PM do cajado do poder não cabia em nenhum.

// SpellAt é a magia vista pelos modificadores: a tradição de quem a lança, a escola, o alcance e qual
// magia. Campo vazio é "não sei" ou "não importa", e só o modificador que também não pede aquele campo a
// alcança — o mosaico da ficha pergunta só pela tradição, e o bônus de uma escola não entra nele.
//
//	SpellAt{Tradition: "arcana", School: "evocacao", Range: "medio", Spell: "bola-de-fogo"}
type SpellAt struct {
	Tradition, School, Range, Spell string
}

// reachedBy diz se o modificador com este alvo alcança a magia: cada campo que ele pede, a magia tem.
func (at SpellAt) reachedBy(t ModifierTarget) bool {
	return (t.Tradition == "" || t.Tradition == at.Tradition) &&
		(t.School == "" || t.School == at.School) &&
		(t.Range == "" || t.Range == at.Range) &&
		(t.Spell == "" || t.Spell == at.Spell)
}

// SpellReach é o que chega a uma magia numa das três contas.
type SpellReach struct {
	// Total já sai pela regra da conta: no CUSTO, a p226 — reduções de fontes diferentes não se acumulam
	// (`resolvePmCostMod`); na CD e no limite, a soma de cada alvo, cada um já empilhado pela resolução.
	Total         int
	Contributions []BreakdownContribution
	// OnlyHere é a parte que vale SÓ por causa desta magia — a escola, o alcance, a magia —, que o mosaico
	// da ficha, perguntando só pela tradição, não mostra. É a nota da linha da magia na aba Magias.
	OnlyHere []BreakdownContribution
	// Factor é o fator que alcança a magia, o mais severo (ver `factor.go`): a Alta Arcana corta o custo
	// das arcanas pela metade (p39).
	Factor Ratio
}

// ForSpell é o que chega à magia `at` na conta `kind` (`spellDC`, `pmLimit` ou `pmCost`).
//
// A ORDEM das contribuições é do geral para o específico — o que vale para toda magia, depois o da
// tradição, depois o da escola e o do alcance —, e, dentro do mesmo nível, a de primeira aparição.
//
//	effects.ForSpell("pmCost", SpellAt{Tradition: "arcana", Range: "pessoal"}) // o medalhão e o cajado
func (e ItemEffects) ForSpell(kind string, at SpellAt) SpellReach {
	reached := []keyedTarget{}
	for _, kt := range e.targets {
		if kt.target.K == kind && at.reachedBy(kt.target) {
			reached = append(reached, kt)
		}
	}
	sort.SliceStable(reached, func(a, b int) bool {
		return specificity(reached[a].target) < specificity(reached[b].target)
	})

	out := SpellReach{Contributions: []BreakdownContribution{}}
	raw := []Contribution{}
	for _, kt := range reached {
		if factor, has := e.Factors[kt.key]; has {
			out.Factor = severest(out.Factor, factor)
		}
		stat, has := e.ByTarget[kt.key]
		if !has {
			continue
		}
		rows := withNoteContribs(stat.Contributions)
		raw = append(raw, stat.Contributions...)
		out.Total += stat.Total
		out.Contributions = append(out.Contributions, rows...)
		if beyondTheTradition(kt.target) {
			out.OnlyHere = append(out.OnlyHere, rows...)
		}
	}
	if kind == "pmCost" {
		out.Total = resolvePmCostMod(raw)
	}
	return out
}

// specificity é quantos campos da magia o alvo pede: o geral é 0, o da tradição 1, o da tradição e da
// escola 2.
func specificity(t ModifierTarget) int {
	n := 0
	for _, part := range []string{t.Tradition, t.School, t.Range, t.Spell} {
		if part != "" {
			n++
		}
	}
	return n
}

// beyondTheTradition diz se o alvo pede mais do que a tradição: é o que vale só nesta magia, e não em
// toda magia da lista.
func beyondTheTradition(t ModifierTarget) bool {
	return t.School != "" || t.Range != "" || t.Spell != ""
}
