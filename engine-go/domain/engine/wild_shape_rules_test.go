package engine

import "testing"

// p63 "Forma Selvagem. […] quando usa esta habilidade você adquire as características de uma das formas
// abaixo. Se ganhar uma arma natural, escolha seu tipo de dano entre corte, impacto e perfuração.
// Características não mencionadas não mudam. Na forma selvagem você não pode falar, empunhar itens […]."

func druidIn(shape *WildShape) Character {
	ch := wielding(attributeSumHero("Druida", 12, `["class.druida.forma-selvagem"]`), "adaga")
	ch.Races = []CharacterRace{{Race: "minotauro"}}
	ch.Expertises = []CharacterExpertise{{Name: "Furtividade", Attribute: "dexterity"}}
	ch.WildShape = shape
	return ch
}

func stealthOf(sheet ComputedSheet) int {
	for _, e := range sheet.Expertises {
		if e.Name == "Furtividade" {
			return e.Total
		}
	}
	return -999
}

func TestTheFerociousFormTradesTheBodyForItsWeapon(t *testing.T) {
	world := BookRuleset(primeBook(t))
	human := world.ComputeSheet(druidIn(nil), nil)
	ch := druidIn(&WildShape{Form: WildFerocious, Tier: WildTierNormal, DamageType: "perfuracao"})
	shaped := world.ComputeSheet(ch, nil)

	// "Você recebe Força +3, +2 na Defesa e uma arma natural que causa 1d8 pontos de dano."
	if got := shaped.Attributes["strength"].Total - human.Attributes["strength"].Total; got != 3 {
		t.Errorf("Forma Feroz: Força %+d, want +3", got)
	}
	if got := shaped.Defense.Total - human.Defense.Total; got != 2 {
		t.Errorf("Forma Feroz: Defesa %+d, want +2", got)
	}
	cards := world.ComputeWeaponCards(ch, nil)
	if cardNamed(cards, "adaga") != nil {
		t.Error("na forma selvagem não se empunha: a adaga não ataca")
	}
	if cardNamed(cards, "Chifres") != nil {
		t.Error("a forma animal não tem os chifres da raça (decisão do dono)")
	}
	claw := cardNamed(cards, "Forma Feroz")
	if claw == nil {
		t.Fatalf("a arma natural da forma devia ter carta: %d cartas", len(cards))
	}
	if claw.Damage != "1d8" || claw.DamageType != "perfuracao" {
		t.Errorf("a arma da Forma Feroz: %s %s, want 1d8 perfuracao", claw.Damage, claw.DamageType)
	}
	if claw.Natural != nil {
		t.Error("a arma da forma não tem ataque extra pago: ela é a arma do golpe")
	}
}

func TestTheFormSizeReplacesTheRaceAndKeepsTheWrittenDamage(t *testing.T) {
	world := BookRuleset(primeBook(t))
	human := world.ComputeSheet(druidIn(nil), nil)

	// "Aprimorada. Você recebe Força +5, +4 na Defesa e uma arma natural (dano 2d6). Seu tamanho muda para
	// Grande (–2 em Furtividade, +2 em testes de manobra)."
	big := druidIn(&WildShape{Form: WildFerocious, Tier: WildTierImproved, DamageType: "corte"})
	sheet := world.ComputeSheet(big, nil)
	if got := stealthOf(sheet) - stealthOf(human); got != -2 {
		t.Errorf("Feroz aprimorada é Grande: Furtividade %+d, want –2", got)
	}
	if claw := cardNamed(world.ComputeWeaponCards(big, nil), "Forma Feroz"); claw == nil || claw.Damage != "2d6" {
		t.Errorf("o dano da forma é o escrito, sem o passo do tamanho: %+v", claw)
	}

	// "Forma Sorrateira. Você recebe Destreza +2 e uma arma natural (dano 1d4). Seu tamanho muda para
	// Pequeno (+2 em Furtividade, –2 em testes de manobra)."
	small := world.ComputeSheet(druidIn(&WildShape{Form: WildSneaky, Tier: WildTierNormal, DamageType: "corte"}), nil)
	if got := stealthOf(small) - stealthOf(human); got != 4 {
		t.Errorf("Sorrateira é Pequena: Furtividade %+d, want +4 (Destreza +2 e tamanho +2)", got)
	}
}

func TestTheResilientFormReducesDamage(t *testing.T) {
	world := BookRuleset(primeBook(t))
	// "Você recebe +5 na Defesa, redução de dano 5 e uma arma natural que causa 1d6 pontos de dano."
	sheet := world.ComputeSheet(druidIn(&WildShape{Form: WildResilient, Tier: WildTierNormal, DamageType: "impacto"}), nil)
	if sheet.DamageReduction.Total != 5 {
		t.Errorf("Forma Resistente: RD %d, want 5", sheet.DamageReduction.Total)
	}
}

func TestTheAgileFormThreatensOnNineteen(t *testing.T) {
	world := BookRuleset(primeBook(t))
	// "Você recebe Destreza +2 e duas armas naturais (dano 1d6 cada, crítico 19)."
	cards := world.ComputeWeaponCards(druidIn(&WildShape{Form: WildAgile, Tier: WildTierNormal, DamageType: "corte"}), nil)
	if claw := cardNamed(cards, "Forma Ágil"); claw == nil || claw.Damage != "1d6" || claw.CritRange != 19 {
		t.Errorf("a arma da Forma Ágil: %+v, want 1d6 e crítico 19", claw)
	}
}

func TestTheSwiftFormMovesAsChosen(t *testing.T) {
	world := BookRuleset(primeBook(t))
	// "Você recebe Destreza +2, uma arma natural que causa 1d6 pontos de dano e um dos benefícios a seguir:
	// deslocamento 15m, deslocamento de escalada 9m ou deslocamento de natação 9m."
	fast := world.ComputeSheet(druidIn(&WildShape{Form: WildSwift, Tier: WildTierNormal, DamageType: "corte", Movement: WildMoveLand}), nil)
	if fast.Displacement.Total != SquaresForDisplacement(15) {
		t.Errorf("Veloz com deslocamento: %d quadrados, want %d (15m)", fast.Displacement.Total, SquaresForDisplacement(15))
	}
	// "Superior. Você recebe Destreza +6 e deslocamento de voo 18m." (Sorrateira)
	flying := world.ComputeSheet(druidIn(&WildShape{Form: WildSneaky, Tier: WildTierSuperior, DamageType: "corte"}), nil)
	if flying.FlySpeed != 18 {
		t.Errorf("Sorrateira superior voa 18m: %d", flying.FlySpeed)
	}
}
