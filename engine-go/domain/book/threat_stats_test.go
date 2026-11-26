package book

import (
	"testing"

	"t20engine/domain/creature"
)

// A TABELA 7-2 PELO ND (p323), com o patamar da p35: o ND 1/2 é Iniciante, o 5 é Veterano, o 20 é Lenda. Fora da
// tabela — o ND 1/4 do Bandido, ou acima de 20 — não há linha, e o app não inventa uma.
func TestTheThreatStatsComeFromTheNDRow(t *testing.T) {
	for _, c := range []struct {
		nd           float64
		tier         string
		defense, hp  int
		attack, many int
	}{
		{0.5, "Iniciante", 15, 10, 7, 1},
		{5, "Veterano", 28, 150, 18, 2},
		{16, "Campeão", 56, 700, 45, 3},
		{20, "Lenda", 65, 900, 55, 4},
	} {
		got, ok := ThreatStatsFor(c.nd)
		if !ok {
			t.Fatalf("ND %v não achou linha na Tabela 7-2", c.nd)
		}
		if got.Tier.Name != c.tier || got.Defense != c.defense || got.HP != c.hp || got.Attack != c.attack || got.Attacks != c.many {
			t.Errorf("ND %v deu %+v; want %s, Defesa %d, PV %d, +%d x%d", c.nd, got, c.tier, c.defense, c.hp, c.attack, c.many)
		}
	}
	for _, nd := range []float64{0.25, 21, 0} {
		if _, ok := ThreatStatsFor(nd); ok {
			t.Errorf("ND %v achou linha, e a Tabela 7-2 vai do 1/2 ao 20", nd)
		}
	}
}

// PREENCHER PELO ND escreve Defesa, PV e os ataques — tantos quantos a tabela diz, com o bônus e o dado dela —,
// SOBRESCREVENDO o que o rascunho tinha (decisão do dono). O resto do bloco fica como estava: quais perícias são
// focadas, a CD e os atributos são do mestre (p322).
func TestFillingByTheNDWritesDefenseHPAndTheAttacks(t *testing.T) {
	stats, _ := ThreatStatsFor(5)
	before := creature.Block{
		ND: 5, Kind: "humanoide", Defense: 99, HP: 1, Strength: 3,
		Attacks: []creature.Attack{{Name: "Clava", AttackBonus: 2, Damage: "1d6"}},
		Skills:  []creature.Skill{{Name: "Furtividade", Bonus: 9}},
	}
	after := stats.FillBlock(before)
	if after.Defense != 28 || after.HP != 150 {
		t.Errorf("o ND 5 deu Defesa %d e PV %d; want 28 e 150", after.Defense, after.HP)
	}
	if len(after.Attacks) != 2 {
		t.Fatalf("o ND 5 faz 2 ataques por rodada, e o bloco ficou com %+v", after.Attacks)
	}
	for _, a := range after.Attacks {
		if a.Name != "Ataque" || a.AttackBonus != 18 || a.Damage != "1d12+11" {
			t.Errorf("o ataque do ND 5 saiu %+v; want Ataque +18 (1d12+11)", a)
		}
	}
	if after.Kind != "humanoide" || after.Strength != 3 || len(after.Skills) != 1 || after.Skills[0].Bonus != 9 {
		t.Errorf("preencher mexeu no que é do mestre: %+v", after)
	}
	if before.Defense != 99 || len(before.Attacks) != 1 {
		t.Error("preencher mudou o bloco de entrada, e não só o devolvido")
	}
}
