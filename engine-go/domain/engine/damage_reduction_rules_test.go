package engine

import "testing"

// A REDUÇÃO DE DANO DE CLASSE, pela ficha inteira (ALE-486).
//
// Ela vem do catálogo: as linhas do Bárbaro (p42), o Bastião (p55) e a
// Especialização em Armadura (Cavaleiro p54, Guerreiro p65). E ACUMULA entre
// habilidades diferentes, pela p226 — "efeitos de habilidades e perícias
// acumulam entre si, exceto quando vierem da mesma habilidade" (decisão do dono
// na ALE-486; antes a RD do Bárbaro e a do Bastião disputavam).

func rdOf(t *testing.T, world *Ruleset, ch Character) int {
	t.Helper()
	return world.ComputeSheet(ch, nil).DamageReduction.Total
}

// "A partir do 5º nível […] você recebe redução de dano 2. A cada três níveis,
// sua RD aumenta em 2, até um máximo de RD 10 no 17º nível." (p42)
//
// Um caso por FRONTEIRA: o que quebra numa tabela de patamares é o limite.
func TestTheBarbarianDamageReductionGrowsEveryThreeLevels(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, tt := range []struct{ level, want int }{
		{4, 0}, {5, 2}, {7, 2}, {8, 4}, {10, 4}, {11, 6}, {14, 8}, {17, 10}, {20, 10},
	} {
		if got := rdOf(t, world, attributeSumHero("Bárbaro", tt.level, `[]`)); got != tt.want {
			t.Errorf("Bárbaro %d: RD = %d, want %d", tt.level, got, tt.want)
		}
	}
}

// "Especialização em Armadura. Se estiver usando armadura pesada, você recebe
// redução de dano 5 […] Pré-requisito: 12º nível" (p54, p65).
func TestTheArmorSpecializationNeedsHeavyArmorAndTheTwelfthLevel(t *testing.T) {
	world := BookRuleset(primeBook(t))
	heavy := func(ch Character) Character { return wearing(ch, "armadura-completa") }
	for _, class := range []struct{ name, id string }{
		{"Guerreiro", "guerreiro"}, {"Cavaleiro", "cavaleiro"},
	} {
		power := `["class.` + class.id + `.especializacao-em-armadura"]`
		if got := rdOf(t, world, heavy(attributeSumHero(class.name, 12, `[]`))); got != 0 {
			t.Errorf("%s 12 sem o poder: RD = %d, want 0", class.name, got)
		}
		if got := rdOf(t, world, heavy(attributeSumHero(class.name, 12, power))); got != 5 {
			t.Errorf("%s 12 com o poder, de armadura completa: RD = %d, want 5", class.name, got)
		}
		if got := rdOf(t, world, heavy(attributeSumHero(class.name, 20, power))); got != 5 {
			t.Errorf("%s 20: RD = %d, want 5 — é fixa, não escala", class.name, got)
		}
		if got := rdOf(t, world, attributeSumHero(class.name, 12, power)); got != 0 {
			t.Errorf("%s 12 sem armadura pesada: RD = %d, want 0", class.name, got)
		}
		// O pré-requisito vale depois da escolha: quem desceu para o 11º nível
		// perdeu o poder, mesmo com ele marcado.
		if got := rdOf(t, world, heavy(attributeSumHero(class.name, 11, power))); got != 0 {
			t.Errorf("%s 11 com o poder marcado: RD = %d, want 0 — o pré-requisito é o 12º", class.name, got)
		}
	}
	// A escolha de uma classe não vale para a outra num multiclasse.
	knight := heavy(attributeSumHero("Cavaleiro", 12, `["class.guerreiro.especializacao-em-armadura"]`))
	if got := rdOf(t, world, knight); got != 0 {
		t.Errorf("Cavaleiro com o poder do Guerreiro: RD = %d, want 0", got)
	}
}

func TestClassDamageReductionsFromDifferentAbilitiesAdd(t *testing.T) {
	world := BookRuleset(primeBook(t))

	// "cumulativa com a RD fornecida por Bastião" — as duas descrições se citam
	// (p54, p55).
	both := wearing(attributeSumHero("Cavaleiro", 12,
		`["class.cavaleiro.caminho-bastiao","class.cavaleiro.especializacao-em-armadura"]`), "armadura-completa")
	if got := rdOf(t, world, both); got != 10 {
		t.Errorf("Bastião + Especialização: RD = %d, want 10", got)
	}

	// p226: habilidades diferentes acumulam — a RD do Bárbaro e a do Bastião.
	multi := attributeSumHero("Bárbaro", 13, `["class.cavaleiro.caminho-bastiao"]`)
	multi.Classes = []CharacterClass{{ClassName: "Bárbaro", Level: 8}, {ClassName: "Cavaleiro", Level: 5}}
	multi = wearing(multi, "armadura-completa")
	if got := rdOf(t, world, multi); got != 4+5 {
		t.Errorf("Bárbaro 8 / Cavaleiro 5 com Bastião, de armadura completa: RD = %d, want 9 (4 + 5, p226)", got)
	}
}

// "PETRIFICADO. O personagem fica inconsciente e recebe redução de dano 8."
// (p394). Ela soma com a de classe, que é de outra origem (p226).
func TestPetrifiedAddsDamageReductionOnTopOfTheClass(t *testing.T) {
	world := BookRuleset(primeBook(t))
	stone := attributeSumHero("Guerreiro", 1, `[]`)
	stone.ActiveConditions = `["petrificado"]`
	if got := rdOf(t, world, stone); got != 8 {
		t.Errorf("petrificado: RD = %d, want 8", got)
	}
	barbarian := attributeSumHero("Bárbaro", 11, `[]`)
	barbarian.ActiveConditions = `["petrificado"]`
	if got := rdOf(t, world, barbarian); got != 6+8 {
		t.Errorf("Bárbaro 11 petrificado: RD = %d, want 14 (6 + 8)", got)
	}
}
