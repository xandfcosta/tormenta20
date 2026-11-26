package book

import (
	"encoding/json"
	"sync"

	"t20engine/domain/catalog"
	"t20engine/domain/creature"
)

// ThreatRow é uma linha da Tabela 7-2 (p323), lida do recurso `threat-stats`.
type ThreatRow struct {
	ND            float64 `json:"nd"`
	Attack        int     `json:"attack"`
	Attacks       int     `json:"attacks"`
	Damage        string  `json:"damage"`
	Defense       int     `json:"defense"`
	HP            int     `json:"hp"`
	FocusedSkills int     `json:"focusedSkills"`
	OtherSkills   int     `json:"otherSkills"`
	DC            int     `json:"dc"`
}

// ThreatTier é um patamar (p35) com o pacote de atributos que o mestre distribui entre os seis (p322).
type ThreatTier struct {
	Name       string  `json:"name"`
	FirstND    float64 `json:"firstNd"`
	LastND     float64 `json:"lastNd"`
	Attributes []int   `json:"attributes"`
}

// ThreatStats é a linha da Tabela 7-2 de um ND, com o patamar dele: a "ficha de ameaça" do livro, com os valores
// finais e sem a origem de cada bônus (ver GLOSSARY, **estatísticas pelo ND**).
type ThreatStats struct {
	ThreatRow
	Tier ThreatTier
}

var (
	threatOnce  sync.Once
	threatTable struct {
		Rows  []ThreatRow  `json:"rows"`
		Tiers []ThreatTier `json:"tiers"`
	}
)

// threatRows lê o recurso uma vez.
func threatRows() ([]ThreatRow, []ThreatTier) {
	threatOnce.Do(func() {
		if raw, ok := catalog.Resource("threat-stats"); ok {
			_ = json.Unmarshal(raw, &threatTable)
		}
	})
	return threatTable.Rows, threatTable.Tiers
}

// ThreatNDs são os NDs que têm linha na Tabela 7-2, na ordem dela: 1/2 e 1 a 20.
func ThreatNDs() []float64 {
	rows, _ := threatRows()
	nds := make([]float64, len(rows))
	for i, row := range rows {
		nds[i] = row.ND
	}
	return nds
}

// ThreatStatsFor é a linha do ND na Tabela 7-2, ou false fora dela: a tabela vai do ND 1/2 ao 20, e um ND entre
// linhas (o 1/4 do Bandido) ou acima de 20 não tem número no livro — inventar um seria regra que ele não tem.
//
//	book.ThreatStatsFor(5) // {Attack: 18, Attacks: 2, Defense: 28, HP: 150, …, Tier: Veterano}
func ThreatStatsFor(nd float64) (ThreatStats, bool) {
	rows, tiers := threatRows()
	for _, row := range rows {
		if row.ND != nd {
			continue
		}
		for _, tier := range tiers {
			if nd >= tier.FirstND && nd <= tier.LastND {
				return ThreatStats{ThreatRow: row, Tier: tier}, true
			}
		}
	}
	return ThreatStats{}, false
}

// FillBlock escreve no bloco o que a tabela dá e o bloco tem onde guardar: a Defesa, os PV e os ataques — tantos
// quantos a linha diz, com o bônus e o dado dela, chamados "Ataque" para o mestre renomear —, SOBRESCREVENDO o que
// havia (decisão do dono). Quais perícias são focadas, a CD das habilidades e como distribuir o pacote de atributos
// são do mestre (p322), e o bloco volta com eles intactos.
//
//	stats.FillBlock(creature.Block{ND: 5}) // Defesa 28, PV 150, dois "Ataque +18 (1d12+11)"
func (s ThreatStats) FillBlock(b creature.Block) creature.Block {
	b.Defense, b.HP = s.Defense, s.HP
	b.Attacks = make([]creature.Attack, s.Attacks)
	for i := range b.Attacks {
		b.Attacks[i] = creature.Attack{Name: "Ataque", AttackBonus: s.Attack, Damage: s.Damage}
	}
	return b
}
