package sheetui

import (
	"strings"

	"github.com/a-h/templ"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/serve/web/ds"
)

// A ABA PERÍCIAS DA FICHA PARALELA (ALE-473), pelo mockup aprovado: cada
// perícia é uma linha que ABRE A CONTA no lugar, e embaixo da conta ficam o
// atributo, o treino, os gastos de PM e o rolar. O dado é o mesmo da ficha de
// hoje (`expertisePanel`); daqui sai só o que a peça nova lê diferente.

// expertiseSumTerms é a conta de uma perícia, uma linha por parcela: o
// atributo, a metade do nível, o treino quando há, e cada item por si.
//
// A linha "Outros" é o RESTO até o total, e só aparece quando ele não é zero.
// Ela fecha a conta por construção: quem soma o que vê chega ao número
// escrito, mesmo que o motor some algo que não veio com nome.
//
//	expertiseSumTerms(ex) // [{Destreza +3} {Metade do nível +2} {Treinada +4} …]
func expertiseSumTerms(ex engine.ExpertiseBreakdown) []ds.SumTerm {
	terms := []ds.SumTerm{
		{Label: book.AttributeLabel(ex.Attribute), Value: book.WithSign(ex.AttrValue)},
		{Label: "Metade do nível", Value: book.WithSign(ex.HalfLevel)},
	}
	shown := ex.AttrValue + ex.HalfLevel
	if ex.Training != 0 {
		terms = append(terms, ds.SumTerm{Label: "Treinada", Note: strings.Join(ex.TrainedBy, ", "), Value: book.WithSign(ex.Training)})
		shown += ex.Training
	}
	for _, c := range ex.ItemContributions {
		terms = append(terms, ds.SumTerm{Label: c.Source, Note: c.Note, Value: book.WithSign(c.Amount), Penalty: c.Amount < 0})
		shown += c.Amount
	}
	if rest := ex.Total - shown; rest != 0 {
		terms = append(terms, ds.SumTerm{Label: "Outros", Value: book.WithSign(rest), Penalty: rest < 0})
	}
	return terms
}

// expertiseHint é a nota debaixo do nome: o atributo e o estado do treino.
// A perícia trancada não entra aqui: o estado dela é o `expertiseAlert`.
//
//	expertiseHint(row) // "FOR · treinada"
func expertiseHint(row expertiseRow) string {
	parts := []string{attributeAbbr[row.Attribute]}
	switch {
	case row.AutoFail:
		parts = append(parts, "falha automática")
	case row.Trained:
		parts = append(parts, "treinada")
	}
	return strings.Join(parts, " · ")
}

// expertiseAlert diz por que o número da perícia trancada não vale (p115): o
// número riscado sozinho não explicava nada (decisão do dono).
func expertiseAlert(row expertiseRow) string {
	if row.Locked {
		return "só treinada — você não tem treino"
	}
	return ""
}

// expertiseSumHead é a linha fechada. A falha automática não tem número (p394),
// e a perícia trancada mostra o dela riscado: ele existe e não vale.
func expertiseSumHead(row expertiseRow) ds.SumHead {
	total := row.Total
	if row.AutoFail {
		total = "—"
	}
	return ds.SumHead{Title: row.Name, Hint: expertiseHint(row), Alert: expertiseAlert(row), Total: total, Muted: row.Locked}
}

// expertiseSumAttrs são o que a linha faz além de abrir. O `name` comum deixa
// UMA conta aberta por vez (o navegador fecha a outra), porque os gastos de PM e o d20
// da mesa são sinais da ABA, e não da linha: duas contas abertas mostrariam o
// mesmo gasto marcado nas duas. Abrir zera os gastos, como o diálogo da ficha
// de hoje fazia.
func expertiseSumAttrs(row expertiseRow) templ.Attributes {
	attrs := templ.Attributes{"name": "expertise"}
	if reset := spendsResetCommand(row); reset != "" {
		attrs["data-on:toggle"] = "if (el.open) { " + reset + "}"
	}
	if !row.Trained {
		attrs["data-show"] = "!$expertise_trained_only"
	}
	return attrs
}

// savesOf e othersOf separam as três resistências das outras perícias, que a
// tela mostra em duas listas. A ordem dentro de cada uma é a do
// `sortedExpertises`.
func savesOf(rows []expertiseRow) []expertiseRow {
	var saves []expertiseRow
	for _, row := range rows {
		if isSave(row) {
			saves = append(saves, row)
		}
	}
	return saves
}

func othersOf(rows []expertiseRow) []expertiseRow {
	var others []expertiseRow
	for _, row := range rows {
		if !isSave(row) {
			others = append(others, row)
		}
	}
	return others
}

func isSave(row expertiseRow) bool {
	return theSaveNames[row.Name] && !row.Custom
}

// expertiseRollAttrs são os do botão de rolar. Na perícia trancada que um
// gasto destrava, o botão nasce apagado e acende com o gasto marcado.
func expertiseRollAttrs(row expertiseRow, click string) templ.Attributes {
	attrs := templ.Attributes{"type": "button", "data-on:click": click}
	if row.Locked {
		attrs["data-attr:disabled"] = "!" + spendsUnlockedExpr(row)
	}
	return attrs
}
