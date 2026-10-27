package sheetui

import (
	"slices"

	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// O QUE A TELA ESCREVE SOBRE FATOR.
//
// O fator do motor é um número por ALVO ou por FONTE, e ele não carrega
// procedência nenhuma: quem sabe de onde ele veio é o acervo do personagem. As
// duas telas que precisam disso fazem a MESMA varredura — a aba Magias, para o
// custo cortado (p39), e o contador de degraus, para o bônus dobrado (p42) —, e
// é por isso que elas moram juntas aqui em vez de uma cópia em cada painel.
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
func costCutNote(dto sheet.CharacterDTO, tradition string) string {
	owned, factor, found := powerThatFactors(dto, func(m engine.Modifier) bool {
		return cutsThisCost(m, tradition)
	})
	if !found {
		return ""
	}
	return owned.Name + " corta este custo " + writtenRatio(factor) + writtenPage(owned.Page)
}

// spellFocusNote é a frase do Foco em Magia: "Foco em Magia tira 1 PM deste
// custo (p131)". Ela segue a regra do `costCutNote`: o número grande da prévia
// é o custo cheio, e quem desconta é o servidor ao conjurar (ALE-428, ALE-510).
func spellFocusNote(dto sheet.CharacterDTO, spellID string) string {
	if !slices.Contains(sheet.PowerPicks(dto.PowerChoices)["foco-em-magia"], spellID) {
		return ""
	}
	return "Foco em Magia tira 1 PM deste custo (p131)"
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
	if stancePowerID == "" {
		return ""
	}
	owned, factor, found := powerThatFactors(dto, func(m engine.Modifier) bool {
		return scalesThisSource(m, stancePowerID)
	})
	if !found {
		return ""
	}
	return owned.Name + " " + writtenScale(factor) + " este bônus" + writtenPage(owned.Page)
}

// powerThatFactors acha, no acervo do personagem, o poder cujo modificador
// casa — e devolve o fator dele junto.
//
// O PRIMEIRO vence, e isso é verdade enquanto houver um só: quem decide entre
// dois fatores no mesmo alvo é o `severest` do motor, e no dia em que dois
// poderes cortarem o mesmo número esta função tem de perguntar a ele em vez de
// parar no primeiro.
func powerThatFactors(
	dto sheet.CharacterDTO, matches func(engine.Modifier) bool,
) (ownedPower, engine.Ratio, bool) {
	byPower := map[string][]engine.Modifier{}
	for _, p := range book.ClassPowersWithModifiers() {
		byPower[p.ID] = p.Modifiers
	}
	for _, owned := range ownedPowersOf(dto) {
		for _, m := range byPower[owned.ID] {
			if matches(m) {
				return owned, *m.Factor, true
			}
		}
	}
	return ownedPower{}, engine.Ratio{}, false
}

// scalesThisSource diz se este modificador multiplica a parcela daquela FONTE —
// o alvo que a Fúria Titânica usa para dobrar a Fúria e não o ataque inteiro.
func scalesThisSource(m engine.Modifier, sourceID string) bool {
	if m.Factor == nil || m.Condition != nil || m.Target.K != "source" {
		return false
	}
	return m.Target.Name == sourceID
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

// cutsThisCost diz se este modificador corta o custo DESTA magia: ele é fator
// de `pmCost`, não pende de interruptor, e o escopo dele — quando existe — é a
// TRADIÇÃO da magia. A Alta Arcana corta as arcanas e deixa a divina do
// multiclasse em paz (p39).
func cutsThisCost(m engine.Modifier, tradition string) bool {
	if m.Factor == nil || m.Condition != nil || m.Target.K != "pmCost" {
		return false
	}
	return m.Target.Scope == "" || m.Target.Scope == tradition
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
