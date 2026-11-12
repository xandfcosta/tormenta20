package engine

// AS LINHAS DA CONTA: como uma parcela resolvida vira linha da decomposição que a ficha mostra ao tocar um
// número. As que entram fecham no total; as que não entraram vêm marcadas, com o motivo (`rowsOf`).

// totalContribsFor is the {total, contributions} shape for a single target
// (spellDC, pmCost, attack/damage globals).
func totalContribsFor(e ItemEffects, target ModifierTarget) TotalContribs {
	stat := StatFor(e, target)
	return TotalContribs{Total: stat.Total, Contributions: rowsOf(stat)}
}

// rowsOf são as linhas da conta de uma ou mais pilhas: primeiro as parcelas que entram, na ordem, e depois
// as que NÃO entraram (`AggregatedStat.Dropped`), marcadas e com o motivo na nota. As que entram fecham no
// total; a tela mostra as outras apagadas e riscadas.
//
//	rowsOf(StatFor(e, ModifierTarget{K: "defense"})) // as armaduras, e a que perdeu o empilhamento
func rowsOf(stats ...AggregatedStat) []BreakdownContribution {
	alive, out := []Contribution{}, []Contribution{}
	for _, stat := range stats {
		alive = append(alive, stat.Contributions...)
		out = append(out, stat.Dropped...)
	}
	rows := withNoteContribs(alive)
	for _, c := range out {
		rows = append(rows, BreakdownContribution{Source: c.Source, Amount: c.Amount, Note: c.Note, Dropped: true})
	}
	return rows
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
