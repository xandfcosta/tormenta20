package book

import "testing"

// TROCAR A PERÍCIA E ROLAR DOIS d20 NO TESTE (ALE-500).
//
// A troca é uma oferta cujo bônus é a DIFERENÇA entre o total da outra perícia
// e o da original — o teste passa a valer o total da outra, e a prévia da tela
// continua sendo uma soma.

func swapTarget(skill, attribute string) SkillSpendTarget {
	return SkillSpendTarget{
		Skill: skill, Attribute: attribute,
		Totals: map[string]int{"Diplomacia": 7, "Enganação": 9, "Investigação": 5, "Intimidação": 3,
			"Conhecimento": 10, "Percepção": 4, "Misticismo": 2},
		TrainedIn: map[string]bool{"Conhecimento": true},
	}
}

// p88 Alpinista Social: "Você pode substituir testes de Diplomacia por testes
// de Enganação." Grátis, e só em Diplomacia.
func TestTheSocialClimberSwapsDiplomacyForDeception(t *testing.T) {
	climber := spendsOf(t, "origin.charlatao.alpinista-social")
	o, ok := offerOf(SkillSpendOffers(climber, swapTarget("Diplomacia", "charisma")), climber[0].ID)
	if !ok || o.Bonus != 9-7 || o.PM != 0 || o.SwapTo != "Enganação" {
		t.Errorf("Alpinista Social em Diplomacia: %+v, want troca grátis por Enganação (+2: 9 − 7)", o)
	}
	if _, ok := offerOf(SkillSpendOffers(climber, swapTarget("Intimidação", "charisma")), climber[0].ID); ok {
		t.Error("Alpinista Social oferecido em Intimidação — o livro diz Diplomacia")
	}
}

// p90 Palpite Fundamentado: "teste de perícia originalmente baseada em
// Inteligência ou Sabedoria, você pode gastar 2 PM para substituí-la por
// Conhecimento." O ATRIBUTO é o do livro, e não o que a ficha trocou.
func TestTheEducatedGuessSwapsAnIntOrWisSkillForKnowledge(t *testing.T) {
	guess := spendsOf(t, "origin.estudioso.palpite-fundamentado")
	o, ok := offerOf(SkillSpendOffers(guess, swapTarget("Misticismo", "intelligence")), guess[0].ID)
	if !ok || o.Bonus != 10-2 || o.PM != 2 {
		t.Errorf("Palpite em Misticismo (Int): %+v, want Conhecimento por 2 PM (+8)", o)
	}
	if _, ok := offerOf(SkillSpendOffers(guess, swapTarget("Diplomacia", "charisma")), guess[0].ID); ok {
		t.Error("Palpite Fundamentado oferecido em Diplomacia, que é de Carisma")
	}
	// A perícia de destino só para treinados, sem treino, não serve de troca.
	untrained := swapTarget("Misticismo", "intelligence")
	untrained.TrainedIn = map[string]bool{}
	if _, ok := offerOf(SkillSpendOffers(guess, untrained), guess[0].ID); ok {
		t.Error("Palpite oferecido sem treino em Conhecimento, que é só para treinados (p115)")
	}
}

// p91 Detetive: 1 PM "até o fim da cena". O PM é do GESTO que levanta a flag;
// enquanto ela está no ar, a troca é grátis em Percepção e Intuição.
func TestTheDetectiveSwapNeedsTheSceneFlag(t *testing.T) {
	detective := spendsOf(t, "origin.guarda.detetive")
	if _, ok := offerOf(SkillSpendOffers(detective, swapTarget("Percepção", "wisdom")), detective[0].ID); ok {
		t.Error("Detetive oferecido sem o gesto de cena")
	}
	withFlag := swapTarget("Percepção", "wisdom")
	withFlag.Flags = map[string]bool{detective[0].SkillSpend.RequiresFlag: true}
	o, ok := offerOf(SkillSpendOffers(detective, withFlag), detective[0].ID)
	if !ok || o.PM != 0 || o.Bonus != 5-4 || o.SwapTo != "Investigação" {
		t.Errorf("Detetive com a cena paga: %+v, want troca grátis por Investigação (+1)", o)
	}
}

// p45 Mestre dos Sussurros: "rola dois dados e usa o melhor", em Investigação
// (interrogar) e Enganação (intriga).
func TestTheMasterOfWhispersRollsTwoDice(t *testing.T) {
	whispers := spendsOf(t, "class.bardo.mestre-dos-sussurros")
	offers := SkillSpendOffers(whispers, swapTarget("Enganação", "charisma"))
	paid, err := PaySkillSpends(offers, []SkillSpendChoice{{Key: "mestre_dos_sussurros", PM: 1}})
	if err != nil || !paid.TwoDice || paid.PM != 0 || paid.Bonus != 0 {
		t.Errorf("Mestre dos Sussurros em Enganação: %+v (%v), want dois d20, grátis", paid, err)
	}
	if _, ok := offerOf(SkillSpendOffers(whispers, swapTarget("Diplomacia", "charisma")), whispers[0].ID); ok {
		t.Error("Mestre dos Sussurros oferecido em Diplomacia")
	}
}

// UMA troca por teste, e ela não se mistura com treino pago: o Eclético diz
// "treinado NESSA perícia", e a troca muda qual perícia é.
func TestASwapIsAloneAndDoesNotMixWithPaidTraining(t *testing.T) {
	both := spendsOf(t, "origin.charlatao.alpinista-social", "origin.estudioso.palpite-fundamentado", "class.bardo.ecletico")
	offers := SkillSpendOffers(both, swapTarget("Diplomacia", "charisma"))
	if _, err := PaySkillSpends(offers, []SkillSpendChoice{{Key: "alpinista_social", PM: 1}, {Key: "ecletico", PM: 1}}); err == nil {
		t.Error("troca com Eclético passou — o treino pago é da perícia original")
	}
	// Confissão e Rede de Contatos valem as duas em Investigação, e são duas
	// trocas: uma só por teste.
	interrogation := SkillSpendOffers(spendsOf(t, "origin.capanga.confissao", "origin.membro-de-guilda.rede-de-contatos"),
		swapTarget("Investigação", "intelligence"))
	if _, err := PaySkillSpends(interrogation, []SkillSpendChoice{{Key: "confissao", PM: 1}, {Key: "rede_de_contatos", PM: 1}}); err == nil {
		t.Error("duas trocas no mesmo teste passaram")
	}
	misticism := SkillSpendOffers(spendsOf(t, "origin.estudioso.palpite-fundamentado", "origin.charlatao.alpinista-social"),
		swapTarget("Misticismo", "intelligence"))
	if len(misticism) != 1 {
		t.Fatalf("Misticismo: %+v, want só o Palpite", misticism)
	}
	paid, err := PaySkillSpends(offers, []SkillSpendChoice{{Key: "alpinista_social", PM: 1}})
	if err != nil || paid.SwapTo != "Enganação" || paid.Bonus != 2 {
		t.Errorf("Alpinista sozinho: %+v (%v), want troca por Enganação (+2)", paid, err)
	}
}

// O ESPECIALISTA também é treino da perícia original (p73): dobrá-lo e trocar
// de perícia no mesmo teste é recusado, como o Eclético (ALE-501).
func TestASwapDoesNotMixWithTheSpecialist(t *testing.T) {
	owned := spendsOf(t, "origin.charlatao.alpinista-social", "class.ladino.especialista")
	target := swapTarget("Diplomacia", "charisma")
	target.Trained, target.TrainingBonus = true, 4
	target.PowerPicks = map[string][]string{"class.ladino.especialista": {"Diplomacia"}}
	offers := SkillSpendOffers(owned, target)
	if len(offers) != 2 {
		t.Fatalf("Diplomacia escolhida e treinada: esperava a troca e o Especialista, veio %+v", offers)
	}
	if _, err := PaySkillSpends(offers, []SkillSpendChoice{{Key: "alpinista_social", PM: 1}, {Key: "especialista", PM: 1}}); err == nil {
		t.Error("troca com Especialista passou — o treino dobrado é da perícia original")
	}
}
