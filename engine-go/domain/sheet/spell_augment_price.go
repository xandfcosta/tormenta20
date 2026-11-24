package sheet

import (
	"fmt"
	"strconv"

	"t20engine/domain/book"
)

// O APRIMORAMENTO VISTO PELO CONJURADOR: o preço dos escolhidos, com a recusa do livro (p170-171), e o cadeado
// de cada um na folha da magia. As duas pontas fazem a mesma pergunta — o servidor recusa e a tela tranca —, e
// a resposta mora aqui para as duas não divergirem.
//
// O círculo do aprimoramento fechou uma FRONTEIRA que estava aberta (ALE-272, fatia 6): o limite existia só
// na tela, e um pedido montado à mão conjurava o que a regra não permite. Travar na tela é UX; a fronteira é
// o `AugmentPrice`, que o `character.Plays.Cast` chama.

// augmentBar é o que tranca um aprimoramento nesta ficha: a marca "Apenas …" que ela não cumpre, ou o círculo
// que ela não alcança. A marca vem antes: o círculo promete que o aprimoramento chega num nível, e o "Apenas
// Divinos" na mão de um arcanista não chega nunca. Zero é aberto.
type augmentBar struct {
	classOnly string
	circle    int
}

func (s SpellOnTheSheet) barOf(a book.SpellAugment) augmentBar {
	if !s.CastsAs(a.ClassOnly) {
		return augmentBar{classOnly: a.ClassOnly}
	}
	if a.RequiresCircle != nil && *a.RequiresCircle > s.CircleReach {
		return augmentBar{circle: *a.RequiresCircle}
	}
	return augmentBar{}
}

// AugmentLock é a nota do cadeado do aprimoramento `index` na folha da magia — "apenas divinos", "exige o 3º
// círculo" —, vazia quando ele está aberto. O aprimoramento trancado aparece em vez de sumir: quem não
// alcança merece saber que existe.
//
//	sheet.CasterOf(arcanista5).Spell("luz").AugmentLock(5) // "apenas divinos"
func (s SpellOnTheSheet) AugmentLock(index int) string {
	if index < 0 || index >= len(s.Spell.Augments) {
		return ""
	}
	bar := s.barOf(s.Spell.Augments[index])
	switch {
	case bar.classOnly != "":
		return "apenas " + bar.classOnly
	case bar.circle > 0:
		return "exige o " + strconv.Itoa(bar.circle) + "º círculo"
	}
	// O que MUDA A EXECUÇÃO tranca com o ritual ligado, que também a muda (p131), e mudança na mesma
	// característica não se acumula (p171).
	if s.ritual && s.Spell.Augments[index].Execution != "" {
		return "não combina com o ritual"
	}
	// O que TORNA a magia sustentada tranca com outra sustentada de pé (p227). Quem recusa no servidor é a
	// `CastRefusal`, que sabe a conjuração inteira; aqui é o cadeado do aprimoramento que a provocaria.
	if sustains := s.Spell.Augments[index].Sustains; sustains != nil && *sustains && s.SustainedClash != "" {
		return "só uma magia sustentada por vez"
	}
	return ""
}

// AugmentPrice é o PM de TODOS os aprimoramentos escolhidos — os da magia e os poderes que são aprimoramento
// (p131) —, ou a frase da recusa. Os da magia são conferidos primeiro: o poder pergunta por eles (a mudança de
// execução, o exclusivo), e um índice fora de faixa já foi recusado quando ele pergunta.
//
//	pm, refusal := sheet.CasterOf(dto).Spell("bola-de-fogo").AugmentPrice(picks, powerPicks)
func (s SpellOnTheSheet) AugmentPrice(picks []AugmentPick, powerPicks []PowerAugmentPick) (int, string) {
	spellPm, refusal := s.spellAugmentPrice(picks)
	if refusal != "" {
		return 0, refusal
	}
	powerPm, refusal := s.powerAugmentPrice(picks, powerPicks)
	if refusal != "" {
		return 0, refusal
	}
	return spellPm + powerPm, ""
}

// spellAugmentPrice é o PM dos aprimoramentos da própria magia, ou a frase da recusa.
func (s SpellOnTheSheet) spellAugmentPrice(picks []AugmentPick) (int, string) {
	if len(picks) == 0 {
		return 0, ""
	}
	if s.Spell.Circle == 0 {
		return 0, "truque não recebe aprimoramento"
	}
	seen := map[int]bool{}
	total, executionChanger := 0, -1
	for _, p := range picks {
		if p.AugmentIndex < 0 || p.AugmentIndex >= len(s.Spell.Augments) {
			return 0, fmt.Sprintf("o aprimoramento %d não existe nesta magia", p.AugmentIndex)
		}
		if seen[p.AugmentIndex] {
			return 0, fmt.Sprintf("o aprimoramento %d veio duas vezes: junte as repetições num pedido só", p.AugmentIndex)
		}
		seen[p.AugmentIndex] = true
		if p.Stacks < 1 {
			return 0, fmt.Sprintf("o aprimoramento %d pede um número de repetições de 1 para cima (veio %d)", p.AugmentIndex, p.Stacks)
		}
		a := s.Spell.Augments[p.AugmentIndex]
		// O "MUDA" não se repete: "Mudanças na mesma característica da magia nunca se acumulam" (p171).
		if a.Kind == "muda" && p.Stacks > 1 {
			return 0, fmt.Sprintf("o aprimoramento %d muda a magia, e mudança não se acumula", p.AugmentIndex)
		}
		// O EXCLUSIVO recusa companhia: "truques não podem ser usados em conjunto com outros aprimoramentos"
		// (p171), e o livro repete a frase à mão na esfera da Invisibilidade (p195) e na Luz permanente (p197).
		//
		// A pergunta é feita DENTRO do laço, sobre cada pedido, e não sobre o primeiro: a ordem da lista é do
		// cliente, e um ramo que olhasse só `picks[0]` deixaria passar a mesma combinação escrita ao contrário.
		if a.Exclusive && len(picks) > 1 {
			return 0, fmt.Sprintf(
				"o aprimoramento %d não pode ser usado em conjunto com outros: escolha só ele",
				p.AugmentIndex)
		}
		// "Mudanças na mesma característica da magia nunca se acumulam" (p171): o ritual, que muda a execução
		// (p131), e dois aprimoramentos que a mudam.
		if a.Execution != "" && s.ritual {
			return 0, fmt.Sprintf("o ritual e o aprimoramento %d mudam a execução, e mudança não se acumula", p.AugmentIndex)
		}
		if a.Execution != "" && executionChanger >= 0 {
			return 0, fmt.Sprintf("os aprimoramentos %d e %d mudam a execução, e mudança não se acumula",
				executionChanger, p.AugmentIndex)
		}
		if a.Execution != "" {
			executionChanger = p.AugmentIndex
		}
		bar := s.barOf(a)
		if bar.classOnly != "" {
			return 0, fmt.Sprintf("o aprimoramento %d é \"apenas %s\", e esta ficha não lança %s assim",
				p.AugmentIndex, bar.classOnly, s.Spell.Name)
		}
		if bar.circle > 0 {
			return 0, fmt.Sprintf(
				"aprimoramento %d exige o %dº círculo e este personagem alcança o %dº",
				p.AugmentIndex, bar.circle, s.CircleReach)
		}
		total += a.PmCost * p.Stacks
	}
	return total, ""
}

// ChoosesTheCantrip diz se um dos aprimoramentos escolhidos é o truque, que zera a conjuração INTEIRA (p171).
//
// Índice fora de faixa é ignorado de propósito: quem recusa o pedido malformado é o `AugmentPrice`, que roda
// antes, e repetir a conferência aqui daria duas respostas para a mesma pergunta.
func (s SpellOnTheSheet) ChoosesTheCantrip(picks []AugmentPick) bool {
	for _, p := range picks {
		if p.AugmentIndex >= 0 && p.AugmentIndex < len(s.Spell.Augments) && s.Spell.Augments[p.AugmentIndex].Cantrip {
			return true
		}
	}
	return false
}
