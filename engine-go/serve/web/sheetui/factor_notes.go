package sheetui

import (
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// O QUE A TELA ESCREVE SOBRE FATOR.
//
// O fator do motor é um número por ALVO ou por FONTE, e a resolução guarda quem o
// pôs (`ItemEffects.FactorSources`). As duas telas que precisam disso — a aba
// Magias, para o custo cortado (p39), e o contador de degraus, para o bônus
// dobrado (p42) — perguntam ao motor e só escrevem a frase. Elas varriam o acervo
// do personagem lendo o alvo de cada poder por conta própria, e a varredura
// passou a prometer o corte da Alta Arcana em toda tradição quando o alvo ganhou
// campo de tradição.
//
// # Por que a tela INFORMA em vez de calcular
//
// As duas prévias somam no cliente e não aplicam fator nenhum — são aproximação
// declarada, com o servidor como autoridade. Levar a conta para lá seria
// escrever a regra uma segunda vez, para divergir no dia em que uma das duas
// mudasse (decisão do dono, 2026-10-16). O número grande continua sendo o cru, e
// a tela diz o que vai acontecer com ele.
//
// # FATOR CONDICIONADO fica de fora
//
// A promessa dele depende de um interruptor, e anunciar o corte sem o "se"
// seria mentir metade do tempo. Nenhum dos de hoje é assim.

// costCutNote é a frase que avisa que este custo vai sair CORTADO: "Alta Arcana
// corta este custo pela metade (p39)".
//
// Ela NOMEIA o poder porque "cai pela metade" sem dono manda procurar.
func costCutNote(computed engine.ComputedSheet, ok bool, tradition string) string {
	if !ok {
		return ""
	}
	reach := computed.ForSpell("pmCost", engine.SpellAt{Tradition: tradition})
	power, known := book.ClassPowers()[reach.FactorSource]
	if reach.Factor == (engine.Ratio{}) || !known {
		return ""
	}
	return power.Name + " corta este custo " + writtenRatio(reach.Factor) + writtenPage(power.BookPage)
}

// onlyHereCutNote é a frase de quem corta o custo SÓ DESTA magia, além do que a caixa "Custo PM" já
// mostra: "Medalhão de prata tira 1 PM deste custo", nas de alcance pessoal (p160), e "Foco em Magia tira
// 1 PM deste custo", na magia escolhida (p131). Ela segue a regra do `costCutNote`: o número da prévia é o
// cheio, e quem desconta é o servidor ao conjurar. A redução que perdeu a disputa da p226 não corta, e
// não entra na frase.
func onlyHereCutNote(computed engine.ComputedSheet, ok bool, at engine.SpellAt) string {
	if !ok {
		return ""
	}
	notes := []string{}
	for _, cut := range computed.ForSpell("pmCost", at).OnlyHere {
		if cut.Amount < 0 && !cut.Dropped {
			notes = append(notes, cut.Source+" tira "+strconv.Itoa(-cut.Amount)+" PM deste custo")
		}
	}
	return strings.Join(notes, " · ")
}

// stanceBonusNote é a frase do contador de degraus: "Fúria Titânica dobra este
// bônus (p42)".
//
// Ela existe pela mesma razão que a de cima, e o número que ela qualifica é o
// que o contador mostra — a base mais os degraus escolhidos. A Fúria Titânica
// dobra a PARCELA da Fúria (p42), então quem escolhe 5 PM lê "+5" no contador e
// sai com +10 na ficha; sem a frase, a diferença seria um número aparecendo do
// nada na aba Combate.
func stanceBonusNote(dto sheet.CharacterDTO, stancePowerID string) string {
	if stancePowerID == "" || dto.Ruleset == nil {
		return ""
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return ""
	}
	effects := dto.Ruleset.EffectsOf(ec, sheet.ToStringSet(dto.Conditionals))
	factor, source, set := effects.FactorOn(engine.ModifierTarget{K: "source", Name: stancePowerID})
	power, known := book.ClassPowers()[source]
	if !set || !known {
		return ""
	}
	return power.Name + " " + writtenScale(factor) + " este bônus" + writtenPage(power.BookPage)
}

// writtenScale é o fator que CRESCE, em palavras. O que encolhe tem frase
// própria na `writtenRatio`: "dobra este custo" e "corta este custo em dobro"
// seriam a mesma frase dizendo coisas opostas.
func writtenScale(r engine.Ratio) string {
	switch {
	case r.Num == 2 && r.Den == 1:
		return "dobra"
	case r.Num == 3 && r.Den == 1:
		return "triplica"
	default:
		return "multiplica por " + strconv.Itoa(r.Num) + "/" + strconv.Itoa(r.Den)
	}
}

// writtenRatio é o fator em palavras. A metade e o zero têm nome em português;
// o resto sai como fração, que é feio e verdadeiro — e não existe hoje.
func writtenRatio(r engine.Ratio) string {
	switch {
	case r.Num == 0:
		return "a zero"
	case r.Num == 1 && r.Den == 2:
		return "pela metade"
	case r.Num == 1 && r.Den == 3:
		return "a um terço"
	default:
		return "para " + strconv.Itoa(r.Num) + "/" + strconv.Itoa(r.Den)
	}
}

// writtenPage é " (p39)", e vazio quando o verbete não cita página — citar
// "(p0)" seria mandar a mesa procurar o que não existe.
func writtenPage(page int) string {
	if page <= 0 {
		return ""
	}
	return " (p" + strconv.Itoa(page) + ")"
}
