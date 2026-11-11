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

// "Seu equipamento desaparece (mas você mantém os benefícios de quaisquer itens vestidos)" (p63): o escudo
// empunhado deixa de dar Defesa na forma, e o vestido continua.
func TestTheShapeKeepsOnlyWhatIsWorn(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ptr := func(s string) *string { return &s }
	gear := func(shape *WildShape) Character {
		ch := druidIn(shape)
		ch.Items = []CharacterItem{
			{ID: 1, CatalogID: ptr("escudo-leve"), Name: "Escudo leve", Equipped: ptr("wielded"), Improvements: "[]"},
			{ID: 2, CatalogID: ptr("armadura-couro"), Name: "Armadura de couro", Equipped: ptr("vested"), Improvements: "[]"},
		}
		return ch
	}
	bare := druidIn(nil)
	bare.Items = nil
	human, geared := world.ComputeSheet(bare, nil), world.ComputeSheet(gear(nil), nil)
	shaped := world.ComputeSheet(gear(&WildShape{Form: WildResilient, Tier: WildTierNormal, DamageType: "impacto"}), nil)
	shield, armor := 1, geared.Defense.Total-human.Defense.Total-1
	if got := shaped.Defense.Total - human.Defense.Total; got != armor+5 {
		t.Errorf("na forma: Defesa %+d sobre o druida nu, want %+d (a armadura vestida e os +5 da forma, sem os +%d do escudo)",
			got, armor+5, shield)
	}
}

// A TORMENTA AFETA O SER, e por isso a arma natural de um poder da Tormenta continua na forma animal (decisão
// do dono); a da raça, não.
func TestTheTormentaWeaponStaysInTheShape(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ch := druidIn(&WildShape{Form: WildFerocious, Tier: WildTierNormal, DamageType: "corte"})
	ch.ClassPowers = `["class.druida.forma-selvagem","dentes-afiados"]`
	cards := world.ComputeWeaponCards(ch, nil)
	if cardNamed(cards, "Mordida") == nil {
		t.Error("a mordida dos Dentes Afiados continua na forma")
	}
	if cardNamed(cards, "Forma Feroz") == nil || cardNamed(cards, "Chifres") != nil {
		t.Error("na forma, a arma da forma entra e os chifres da raça saem")
	}
}

// A DEVOÇÃO TAMBÉM VEM DE FORA: o que um deus dá afeta o ser, e fica na forma, como o da Tormenta. O que é do
// próprio corpo — a raça — sai com ele (decisão do dono).
func TestTheDevotionWeaponStaysInTheShape(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ch := druidIn(&WildShape{Form: WildFerocious, Tier: WildTierNormal, DamageType: "corte"})
	ch.GodPowers = `["Presas Primordiais"]`
	ch.PowerUses = []PowerUse{{PowerID: "deus.kallyadranoch.presas-primordiais", Scope: "scene", Used: 1}}
	cards := world.ComputeWeaponCards(ch, nil)
	if cardNamed(cards, "Mordida") == nil {
		t.Error("as Presas Primordiais ativadas continuam na forma")
	}
	if cardNamed(cards, "Chifres") != nil {
		t.Error("os chifres são do corpo da raça, e saem na forma")
	}
}
