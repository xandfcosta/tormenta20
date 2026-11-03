package engine

import "testing"

// A CONTA DE PV E PM NÍVEL A NÍVEL (p34-35). Os números são os da tabela de classe e
// do catálogo, escritos à mão.

func levelTotals(lines []VitalLevelLine) []int {
	totals := make([]int, len(lines))
	for i, line := range lines {
		totals[i] = line.Total
	}
	return totals
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// O guerreiro começa com 20 + Con, e cada nível soma 5 + Con (p34, p65).
func TestEachWarriorLevelAddsItsClassAndConstitution(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ctx := VitalContext{
		Level: 3, Classes: []ClassEntry{{ClassName: "Guerreiro", Level: 3}},
		AttrTotals: map[string]int{"constitution": 4},
	}
	got := world.VitalLevelsFor(ctx, []string{"Guerreiro", "Guerreiro", "Guerreiro"})

	if want := []int{24, 9, 9}; !sameInts(levelTotals(got.PV), want) {
		t.Errorf("PV por nível = %v, want %v", levelTotals(got.PV), want)
	}
	if want := []int{3, 3, 3}; !sameInts(levelTotals(got.PM), want) {
		t.Errorf("PM por nível = %v, want %v", levelTotals(got.PM), want)
	}
	first := got.PV[0].Parts
	if len(first) != 2 || first[0].Source != "" || first[0].Amount != 20 ||
		first[1].Attribute != "constitution" || first[1].Amount != 4 {
		t.Errorf("as parcelas do 1º nível vieram %+v, want 20 da classe e 4 de Constituição", first)
	}
}

// "Ganha os PV de um nível subsequente, não do primeiro" (p35), na ordem em que os
// níveis foram pegos.
func TestMulticlassLevelsFollowTheOrderTheyWereTaken(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ctx := VitalContext{
		Level: 4, Classes: []ClassEntry{{ClassName: "Guerreiro", Level: 3}, {ClassName: "Arcanista", Level: 1}},
		AttrTotals: map[string]int{"constitution": 1},
	}
	got := world.VitalLevelsFor(ctx, []string{"Guerreiro", "Arcanista", "Guerreiro", "Guerreiro"})

	// Guerreiro 20+1; Arcanista 2+1; Guerreiro 5+1 duas vezes.
	if want := []int{21, 3, 6, 6}; !sameInts(levelTotals(got.PV), want) {
		t.Errorf("PV por nível = %v, want %v", levelTotals(got.PV), want)
	}
	if got.PV[1].ClassName != "Arcanista" || got.PV[1].Level != 2 {
		t.Errorf("o 2º nível veio %+v, want Arcanista", got.PV[1])
	}
	if want := []int{3, 6, 3, 3}; !sameInts(levelTotals(got.PM), want) {
		t.Errorf("PM por nível = %v, want %v", levelTotals(got.PM), want)
	}
}

// Registro que não bate com as classes (ficha antiga) não pode desenhar uma conta
// falsa: vale a ordem das classes, que é a que o total usa.
func TestALevelRecordThatDoesNotMatchTheClassesIsIgnored(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ctx := VitalContext{
		Level: 2, Classes: []ClassEntry{{ClassName: "Guerreiro", Level: 1}, {ClassName: "Arcanista", Level: 1}},
		AttrTotals: map[string]int{},
	}
	for name, record := range map[string][]string{
		"vazio":         nil,
		"curto":         {"Guerreiro"},
		"outra classe":  {"Guerreiro", "Bardo"},
		"o seed depois": {"Arcanista", "Guerreiro"},
	} {
		got := world.VitalLevelsFor(ctx, record)
		if want := []int{20, 2}; !sameInts(levelTotals(got.PV), want) {
			t.Errorf("registro %s: PV por nível = %v, want %v", name, levelTotals(got.PV), want)
		}
	}
}

// "Você sempre ganha pelo menos 1 PV ao subir de nível" (p35).
func TestALevelNeverGivesLessThanOneHitPoint(t *testing.T) {
	world := BookRuleset(primeBook(t))
	ctx := VitalContext{
		Level: 2, Classes: []ClassEntry{{ClassName: "Arcanista", Level: 2}},
		AttrTotals: map[string]int{"constitution": -3},
	}
	got := world.VitalLevelsFor(ctx, []string{"Arcanista", "Arcanista"})

	// O 1º nível não tem piso: 8 − 3.
	if want := []int{5, 1}; !sameInts(levelTotals(got.PV), want) {
		t.Fatalf("PV por nível = %v, want %v", levelTotals(got.PV), want)
	}
	if got.PV[0].Floored || !got.PV[1].Floored {
		t.Errorf("o piso veio %v e %v, want só no 2º nível", got.PV[0].Floored, got.PV[1].Floored)
	}
}

// Bênção do Mana é +1 PM a cada dois níveis, arredondado para cima, e o caminho do
// arcanista soma o atributo-chave uma vez: cai no 1º nível.
func TestPowersLandOnTheLevelsTheirRuleAdds(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := Character{
		Level: 3, Intelligence: 5,
		Classes: []CharacterClass{{ClassName: "Arcanista", Level: 3}}, ClassPowers: "[]",
		GodPowers: `["Bênção do Mana"]`, ClassChoices: `{"Arcanista":{"caminho":"mago"}}`,
	}
	got := world.VitalLevelsFor(world.VitalContextFor(hero), []string{"Arcanista", "Arcanista", "Arcanista"})

	// 6 + 5 + 1; 6; 6 + 1.
	if want := []int{12, 6, 7}; !sameInts(levelTotals(got.PM), want) {
		t.Fatalf("PM por nível = %v, want %v", levelTotals(got.PM), want)
	}
	if !hasPart(got.PM[0], "", "intelligence", 5) {
		t.Errorf("o 1º nível não diz 5 de Inteligência: %+v", got.PM[0].Parts)
	}
	if !hasPart(got.PM[2], "Bênção do Mana", "", 1) {
		t.Errorf("o 3º nível não diz 1 de Bênção do Mana: %+v", got.PM[2].Parts)
	}
}

func hasPart(line VitalLevelLine, source, attribute string, amount int) bool {
	for _, part := range line.Parts {
		if part.Amount == amount && part.Attribute == attribute && (attribute != "" || part.Source == source) {
			return true
		}
	}
	return false
}

// AS LINHAS SOMAM O MÁXIMO DA FICHA, em toda classe, todo nível e todo poder do catálogo
// que mexe em PV ou PM. A soma vem antes das parcelas: uma tabela bonita que não fecha é
// a tela mentindo.
func TestTheLinesAlwaysAddUpToTheMaximum(t *testing.T) {
	cat := primeBook(t)
	world := BookRuleset(cat)
	powers := [][]string{{}}
	for _, power := range cat.generalByID {
		for _, m := range power.Modifiers {
			if m.Target.K == "maxPv" || m.Target.K == "maxPm" {
				powers = append(powers, []string{power.ID})
				break
			}
		}
	}
	if len(powers) < 3 {
		t.Fatalf("o controle falhou: só %d poderes de PV ou PM no catálogo", len(powers)-1)
	}
	measured := 0
	for className := range classVitalsTable {
		for _, level := range []int{1, 2, 5, 11, 20} {
			for _, con := range []int{-3, 0, 4} {
				for _, owned := range powers {
					ctx := VitalContext{
						Level: level, Classes: []ClassEntry{{ClassName: className, Level: level}}, PowerIDs: owned,
						AttrTotals: map[string]int{"constitution": con, "intelligence": 3, "wisdom": 2, "charisma": 1},
					}
					record := make([]string, level)
					for i := range record {
						record[i] = className
					}
					lines := world.VitalLevelsFor(ctx, record)
					pools := world.ComputeVitals(ctx)
					pv, pm := 0, 0
					for _, line := range lines.PV {
						pv += line.Total
					}
					for _, line := range lines.PM {
						pm += line.Total
					}
					measured++
					if pv != pools.PvMax || pm != pools.PmMax {
						t.Fatalf("%s nível %d, Con %d, poderes %v: as linhas somam %d PV e %d PM, e o máximo é %d e %d",
							className, level, con, owned, pv, pm, pools.PvMax, pools.PmMax)
					}
				}
			}
		}
	}
	// O denominador: as 14 classes, 5 níveis e 3 Constituições, por poder.
	if want := 14 * 5 * 3 * len(powers); measured != want {
		t.Errorf("a varredura mediu %d casos, want %d", measured, want)
	}
}

// OS ANÉIS (p342): "Você recebe +10 PV" e "+5 PM", somente vestidos. O dia de uso fica
// com o mestre, e a tela o avisa (decisão do dono).
func ringHero(rings ...string) Character {
	vested := "vested"
	hero := Character{Level: 1, Classes: []CharacterClass{{ClassName: "Guerreiro", Level: 1}}, ClassPowers: "[]"}
	for _, ring := range rings {
		id := ring
		hero.Items = append(hero.Items, CharacterItem{CatalogID: &id, Name: id, Equipped: &vested, Improvements: "[]"})
	}
	return hero
}

func TestAVestedRingAddsToTheMaximum(t *testing.T) {
	world := BookRuleset(primeBook(t))
	stowed := ringHero("anel-da-vitalidade")
	stowed.Items[0].Equipped = nil
	for _, c := range []struct {
		name   string
		hero   Character
		pv, pm int
	}{
		{"sem anel", ringHero(), 20, 3},
		{"guardado não soma", stowed, 20, 3},
		{"Anel da Vitalidade", ringHero("anel-da-vitalidade"), 30, 3},
		{"Anel da Energia", ringHero("anel-da-energia"), 20, 8},
		{"os dois", ringHero("anel-da-vitalidade", "anel-da-energia"), 30, 8},
		// "Efeitos de itens [...] acumulam com os de outras fontes, mas não entre si" (p226).
		{"dois iguais valem um", ringHero("anel-da-vitalidade", "anel-da-vitalidade"), 30, 3},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := world.VitalsForCharacter(c.hero)
			if got.PvMax != c.pv || got.PmMax != c.pm {
				t.Errorf("máximo = %d PV e %d PM, want %d e %d", got.PvMax, got.PmMax, c.pv, c.pm)
			}
		})
	}
}

// O anel não pertence a nível nenhum: ele é uma linha à parte, e a soma fecha com ela.
func TestARingIsItsOwnLineInTheAccount(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := ringHero("anel-da-vitalidade")
	got := world.VitalLevelsFor(world.VitalContextFor(hero), []string{"Guerreiro"})

	if len(got.PV) != 2 {
		t.Fatalf("a conta de PV tem %d linhas, want o nível e o anel: %+v", len(got.PV), got.PV)
	}
	ring := got.PV[1]
	if ring.Level != 0 || ring.Source != "Anel da vitalidade" || ring.Total != 10 {
		t.Errorf("a linha do anel veio %+v", ring)
	}
	if ring.Note == "" {
		t.Error("a linha do anel não traz o aviso do dia de uso")
	}
	if got.PV[0].Total != 20 {
		t.Errorf("o anel entrou no 1º nível: %+v", got.PV[0])
	}
}
