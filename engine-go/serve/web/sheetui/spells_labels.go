package sheetui

import (
	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// O QUE A ABA MAGIAS ESCREVE sobre o PREÇO.
//
// Ele mora fora do `spells.go` porque aquele arquivo já está na dívida do teto
// de 500 linhas, e a linha de base só pode encolher — e porque é a divisão que
// o `effects_labels.go` e o `powers_labels.go` já fazem: rótulo muda quando a
// mesa lê mal, painel muda quando chega regra.

// costCutNote é a frase que avisa que este custo vai sair CORTADO.
//
// # Por que a tela informa em vez de calcular
//
// A prévia do custo é uma soma feita no cliente e não aplica modificador
// nenhum — é aproximação declarada, com o servidor como autoridade (ver
// `costPreview`). Com a Alta Arcana (p39) a diferença deixa de ser de 2 PM e
// passa a ser de 2×, e um número grande que ninguém explica é pior que um
// número grande. Levar a conta para o cliente seria escrever a regra uma
// segunda vez, para divergir no dia em que uma das duas mudasse (decisão do
// dono, 2026-10-16).
//
// # Por que ela nomeia o PODER
//
// "Cai pela metade" sem dono manda procurar. O fator do motor é um número por
// ALVO e não carrega procedência — quem sabe de onde ele veio é o acervo do
// personagem, e é ele que esta função percorre.
//
// FATOR CONDICIONADO fica de fora: a promessa dele depende de um interruptor, e
// anunciar o corte sem o "se" seria mentir metade do tempo. Nenhum dos de hoje
// é assim.
func costCutNote(dto sheet.CharacterDTO, tradition string) string {
	byPower := map[string][]engine.Modifier{}
	for _, p := range book.ClassPowersWithModifiers() {
		byPower[p.ID] = p.Modifiers
	}
	for _, owned := range ownedPowersOf(dto) {
		for _, m := range byPower[owned.ID] {
			if !cutsThisCost(m, tradition) {
				continue
			}
			return owned.Name + " corta este custo " + writtenRatio(*m.Factor) + writtenPage(owned.Page)
		}
	}
	return ""
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
