package sheetui

import (
	"strconv"

	"t20engine/domain/book"
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
	// key é o aprimoramento para o conjurador, que diz o que ligá-lo apaga.
	key sheet.AugmentKey
}

// augmentRowsOf traduz os aprimoramentos da magia e os poderes da ficha que são
// aprimoramento (p131), trancando os fora de alcance e montando os gestos que se
// apagam entre si.
func augmentRowsOf(spell book.Spell, onSheet sheet.SpellOnTheSheet) []augmentRow {
	rows := make([]augmentRow, 0, len(spell.Augments))
	for i, a := range spell.Augments {
		row := augmentRow{
			Signal: augmentSignal(i), PM: a.PmCost, Stacks: a.Kind != "muda",
			Description: a.Description,
			Exclusive:   a.Exclusive, Cantrip: a.Cantrip,
			key: sheet.SpellAugmentKey(i),
		}
		if lock := onSheet.AugmentLock(i); lock != "" {
			row.Locked, row.LockNote = true, lock
		}
		rows = append(rows, row)
	}
	for _, power := range onSheet.PowerAugments {
		rows = append(rows, augmentRow{
			Signal: powerAugmentSignalOf(power.Power.PowerID), PM: power.Power.Augment.PmCost,
			Stacks:      power.Power.Augment.Kind != "muda",
			Description: power.Power.Name + ": " + power.Power.Augment.Description,
			Locked:      power.Lock != "", LockNote: power.Lock,
			key: sheet.PowerAugmentKey(power.Power.PowerID),
		})
	}
	return withAugmentGestures(rows, onSheet)
}

// withAugmentGestures monta os gestos de cada linha com o que ligá-la APAGA, que é do conjurador
// (`SpellOnTheSheet.Clears`): aqui só a chave vira sinal. Sem isto a tela deixa montar uma combinação que o
// servidor recusa, e mostra um custo que não vale: a recusa é a fronteira, mas oferecer o que não vale é a
// tela mentindo.
func withAugmentGestures(rows []augmentRow, onSheet sheet.SpellOnTheSheet) []augmentRow {
	for i := range rows {
		row := &rows[i]
		var clears []string
		for _, key := range onSheet.Clears(row.key) {
			clears = append(clears, signalOf(key))
		}
		row.Toggle = clearSignals(clears) + row.Signal + " = " + row.Signal + " ? 0 : 1"
		row.More = clearSignals(clears) + row.Signal + "++"
		// Diminuir não apaga nada: tirar uma pilha não monta combinação nova.
		row.Less = row.Signal + " = Math.max(0, " + row.Signal + " - 1)"
	}
	return rows
}

// signalOf é o sinal que guarda a pilha do aprimoramento da chave.
func signalOf(key sheet.AugmentKey) string {
	if key.PowerID != "" {
		return powerAugmentSignalOf(key.PowerID)
	}
	return augmentSignal(key.Index)
}

// clearSignals monta "$sinal = 0; " para cada sinal.
func clearSignals(signals []string) string {
	expr := ""
	for _, signal := range signals {
		expr += signal + " = 0; "
	}
	return expr
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

// powerAugmentSignals são os sinais dos poderes que são aprimoramento (p131), um por poder, pelo nome — eles
// valem para qualquer magia, e por isso não têm o índice da lista da magia.
//
// É a ÚNICA lista: a declaração no `<body>`, o zerar da folha, o gesto de cada linha e a leitura do corpo
// saem dela. Poder novo é uma linha, e o `TestEveryPowerAugmentHasASignal` reprova o que ficar sem ela — um
// sinal que falta não é erro no navegador, é o gesto não fazer nada.
var powerAugmentSignals = []struct{ power, name string }{
	{book.QuickenedSpellPower, "augment_quickened"},
	{book.AmplifiedSpellPower, "augment_amplified"},
	{book.DiscreetSpellPower, "augment_discreet"},
	{book.PiercingSpellPower, "augment_piercing"},
	{book.CombatPrayerPower, "augment_combat_prayer"},
	{"class.clerigo.magia-sagrada-profana", "augment_holy"},
	{"class.clerigo.comunhao-vital", "augment_vital_communion"},
	{"class.druida.espirito-dos-solsticios", "augment_solstice"},
	{book.PrairieFreedomPower, "augment_prairie"},
}

// powerAugmentSignalOf é o sinal do poder, com o `$`, ou vazio no poder sem sinal.
func powerAugmentSignalOf(power string) string {
	for _, s := range powerAugmentSignals {
		if s.power == power {
			return "$" + s.name
		}
	}
	return ""
}

// powerAugmentsDeclared é a declaração dos sinais dos poderes no `<body>`: "augment_quickened: 0, …".
func powerAugmentsDeclared() string {
	declared := ""
	for i, s := range powerAugmentSignals {
		if i > 0 {
			declared += ", "
		}
		declared += s.name + ": 0"
	}
	return declared
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
