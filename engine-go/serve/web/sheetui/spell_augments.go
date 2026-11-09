package sheetui

import (
	"slices"
	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/catalog"
	"t20engine/domain/sheet"
)

// OS APRIMORAMENTOS NA FOLHA DA MAGIA: os da magia e os dos poderes (p131), as
// linhas, os gestos que se apagam entre si e a prévia do custo.

// augmentRow é um aprimoramento, com o que ele custa e se está trancado.
type augmentRow struct {
	// Signal é o sinal que guarda a pilha, já com o `$`.
	Signal string
	PM     int
	// Stacks diz se ele EMPILHA. `muda` não empilha — trocar o tipo de dano duas
	// vezes não é trocar duas vezes mais.
	Stacks      bool
	Description string
	// Locked é o aprimoramento fora de alcance: um círculo acima do alcançável, ou
	// o poder que não se aplica a esta magia. Ele aparece com cadeado em vez de
	// sumir, e LockNote diz por quê: a pessoa merece saber que existe e o que
	// falta.
	Locked   bool
	LockNote string
	// Exclusive não aceita companhia na mesma conjuração (p171). Truque diz, além
	// disso, que a conjuração inteira custa zero. Ver GLOSSARY.
	Exclusive bool
	Cantrip   bool
	// Toggle, More e Less são os gestos MONTADOS, e não expressões escritas no
	// `.templ`, porque cada um precisa ver a VIZINHANÇA: ligar o exclusivo apaga
	// os outros e ligar qualquer outro apaga o exclusivo. É a mesma regra do
	// `augmentsReset` — quem troca limpa —, e é ela que impede a tela de
	// oferecer uma combinação que o servidor vai recusar.
	Toggle string
	More   string
	Less   string
}

// augmentRowsOf traduz os aprimoramentos da magia e os poderes da ficha que são
// aprimoramento (p131), trancando os fora de alcance e montando os gestos que se
// apagam entre si.
func augmentRowsOf(dto sheet.CharacterDTO, spellID string, spell catalog.Spell, castable int) []augmentRow {
	fromBook := spellOfBook(spellID)
	rows := make([]augmentRow, 0, len(spell.Augments))
	for i, a := range spell.Augments {
		row := augmentRow{
			Signal: augmentSignal(i), PM: a.PmCost, Stacks: a.Kind != "muda",
			Description: augmentDescription(spellID, i),
			Exclusive:   a.Exclusive, Cantrip: a.Cantrip,
		}
		if a.RequiresCircle != nil && *a.RequiresCircle > castable {
			row.Locked = true
			row.LockNote = "exige o " + strconv.Itoa(*a.RequiresCircle) + "º círculo"
		}
		rows = append(rows, row)
	}
	owned := sheet.OwnedPowers(dto)
	for _, power := range book.PowerAugments() {
		if !slices.ContainsFunc(owned, func(p sheet.OwnedPower) bool { return p.ID == power.PowerID }) {
			continue
		}
		bar := book.PowerAugmentBar(power.PowerID, fromBook)
		rows = append(rows, augmentRow{
			Signal: powerAugmentSignals[power.PowerID], PM: power.Augment.PmCost, Stacks: power.Augment.Kind != "muda",
			Description: power.Name + ": " + power.Augment.Description,
			Locked:      bar != "", LockNote: bar,
		})
	}
	return withAugmentGestures(rows)
}

// withAugmentGestures monta o que cada gesto APAGA: o exclusivo apaga todos os
// outros, os outros apagam os exclusivos, e a Magia Acelerada e o aprimoramento
// que muda a execução se apagam entre si. Sem isto a tela deixa montar uma
// combinação proibida, mostra um custo e o servidor recusa na hora do clique: a
// recusa é a fronteira, mas oferecer o que não vale é a tela mentindo.
func withAugmentGestures(rows []augmentRow) []augmentRow {
	var all, exclusive, execution []string
	quickened := powerAugmentSignals[book.QuickenedSpellPower]
	for _, row := range rows {
		all = append(all, row.Signal)
		if row.Exclusive {
			exclusive = append(exclusive, row.Signal)
		}
		if book.AugmentChangesExecution(row.Description) {
			execution = append(execution, row.Signal)
		}
	}
	for i := range rows {
		row := &rows[i]
		deletes := exclusive
		switch {
		case row.Exclusive:
			deletes = all
		case row.Signal == quickened:
			deletes = slices.Concat(exclusive, execution)
		case slices.Contains(execution, row.Signal):
			deletes = slices.Concat(exclusive, []string{quickened})
		}
		row.Toggle = clearSignals(deletes, row.Signal) + row.Signal + " = " + row.Signal + " ? 0 : 1"
		row.More = clearSignals(deletes, row.Signal) + row.Signal + "++"
		// Diminuir não apaga nada: tirar uma pilha não monta combinação nova.
		row.Less = row.Signal + " = Math.max(0, " + row.Signal + " - 1)"
	}
	return rows
}

// clearSignals monta "$sinal = 0; " para cada sinal, pulando o próprio.
func clearSignals(signals []string, own string) string {
	expr := ""
	for _, signal := range signals {
		if signal != own {
			expr += signal + " = 0; "
		}
	}
	return expr
}

// augmentDescription lê o texto do aprimoramento do catálogo cru.
//
// O `catalog.Augment` do Go é um subconjunto deliberado e não carrega a
// descrição — mas a tela precisa dela, senão o jogador escolhe entre "1 PM" e
// "1 PM" sem saber o que cada um faz.
func augmentDescription(spellID string, index int) string {
	fromBook := spellOfBook(spellID)
	if index < len(fromBook.Augments) {
		return fromBook.Augments[index].Description
	}
	return ""
}

// augmentSignal é o sinal que guarda a pilha de um aprimoramento.
//
// São SEIS, reaproveitados por todas as magias, e não um por magia: só um
// diálogo abre por vez, e seis é o máximo do catálogo (Conjurar Monstro). Um
// sinal por magia daria 198 × 6 declarações no `<body>` para guardar seis
// números.
func augmentSignal(index int) string {
	return "$augment" + strconv.Itoa(index)
}

// powerAugmentSignals são os sinais dos poderes que são aprimoramento, um por poder.
var powerAugmentSignals = map[string]string{
	book.QuickenedSpellPower: "$augment_quickened",
	book.AmplifiedSpellPower: "$augment_amplified",
	book.DiscreetSpellPower:  "$augment_discreet",
}

// costPreview é a expressão que soma o custo na tela.
//
// Ela é PRÉVIA e não decisão: quem recusa é o servidor, com a regra inteira — o
// teto da p224, a redução de custo por item e o PM disponível. Escrever a regra
// aqui daria uma segunda conta do mesmo número.
//
// Aprimoramento TRANCADO fica de fora da soma: ele não tem contador, então o
// sinal dele nunca sobe — mas somá-lo mostraria um custo que o servidor não
// cobraria.
func costPreview(spell learnedSpellRow) string { return costNumber(spell) + " + ' PM'" }

func costNumber(spell learnedSpellRow) string {
	expr := strconv.Itoa(spell.BasePm)
	for _, a := range spell.Augments {
		if a.Locked {
			continue
		}
		expr += " + " + strconv.Itoa(a.PM) + " * " + a.Signal
	}
	// O TRUQUE zera a conjuração INTEIRA, e não a parcela dele: "reduz seu custo
	// em PM para zero" (p171). Somado como um `+0 PM`, o custo base ficava de pé
	// e a tela anunciava 1 PM sobre uma conjuração que o servidor não cobra
	// (ALE-339). A prévia continua sem decidir nada — ela só deixou de discordar.
	for _, a := range spell.Augments {
		if a.Cantrip {
			return "(" + a.Signal + " ? 0 : (" + expr + "))"
		}
	}
	return "(" + expr + ")"
}

// ritualBillPreview é a conta do ritual: "T$ 10 por PM gasto" (p131). Ela só aparece, e o app não
// cobra (decisão do dono).
func ritualBillPreview(spell learnedSpellRow) string {
	return "'Como ritual, T$ ' + " + costNumber(spell) + " * 10 + ' em oferendas e 1 hora de execução, ou o dobro (p131)'"
}

// augmentPriceWritten é a linha de baixo do aprimoramento: o que ele custa e o
// que ele impede.
//
// O TRUQUE não escreve "+0 PM", que é a mesma coisa que um aprimoramento de
// custo zero — e a Luz tem um desses (p197), então as duas linhas sairiam
// idênticas dizendo coisas diferentes. Ele escreve que ZERA a conjuração, que é
// o que a p171 diz.
//
// E a linha diz que ele não combina porque o gesto APAGA os outros: sem a frase,
// ligar o truque faz as escolhas anteriores sumirem sem explicação.
func augmentPriceWritten(a augmentRow) string {
	if a.Locked {
		return "+" + strconv.Itoa(a.PM) + " PM · " + a.LockNote
	}
	if a.Cantrip {
		return "a conjuração custa 0 PM · não combina com os outros"
	}
	if a.Exclusive {
		return "+" + strconv.Itoa(a.PM) + " PM · não combina com os outros"
	}
	return "+" + strconv.Itoa(a.PM) + " PM"
}
