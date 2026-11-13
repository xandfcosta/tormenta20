package engine

// AS LINHAS DA CONTA: como uma parcela resolvida vira linha da decomposição que a ficha mostra ao tocar um
// número. As que entram fecham no total; as que não entraram vêm marcadas, com o motivo (`rowsOf`).

// totalContribsFor é a forma {total, linhas} de um alvo só: o ataque e o dano gerais, a CD de magia.
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
	counted, dropped := []Contribution{}, []Contribution{}
	for _, stat := range stats {
		counted = append(counted, stat.Contributions...)
		dropped = append(dropped, stat.Dropped...)
	}
	rows := withNoteContribs(counted)
	for _, c := range dropped {
		rows = append(rows, BreakdownContribution{Source: c.Source, Amount: c.Amount, Note: c.Note, Dropped: true})
	}
	return rows
}

// withNoteContribs são as parcelas da resolução como linhas da conta, com a nota.
func withNoteContribs(cs []Contribution) []BreakdownContribution {
	out := []BreakdownContribution{}
	for _, c := range cs {
		out = append(out, BreakdownContribution{Source: c.Source, Amount: c.Amount, Note: c.Note})
	}
	return out
}

// sourceAmountContribs são as parcelas como {fonte, valor}, sem a nota: a conta dos atributos não a mostra.
func sourceAmountContribs(cs []Contribution) []SourceAmount {
	out := []SourceAmount{}
	for _, c := range cs {
		out = append(out, SourceAmount{Source: c.Source, Amount: c.Amount})
	}
	return out
}
