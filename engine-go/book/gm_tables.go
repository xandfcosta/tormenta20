package book

import (
	"encoding/json"
	"sync"

	"t20engine/catalog"
)

// AS TABELAS DO CAPÍTULO 6 e o esqueleto de masmorra, tipados (ALE-278).
//
// Elas moravam na view do Improviso, que lia `catalog.Resource` DIRETO —
// exatamente o defeito que o guarda de fronteira da forja pegou no primeiro dia
// (`items.go` contornando a camada tipada) e que continuou sentado ali por três
// fatias. A regra que decidiu o destino é a mesma daquele achado: **o destino de
// uma função é a DEPENDÊNCIA dela**. Estas lêem catálogo, então são do livro —
// mesmo que só a Mesa do Mestre as use.
//
// Não confundir com `master`, que é a TELA (ver o GLOSSARIO): `gm` aqui é o nome
// do recurso no catálogo (`gm-tables`), e a fronteira não se renomeia por
// estética.

// RuinRow é uma linha da Tabela 6-4 (d6, p272), que casa por FAIXA.
type RuinRow struct {
	RollMin int    `json:"rollMin"`
	RollMax int    `json:"rollMax"`
	Outcome string `json:"outcome"`
	Label   string `json:"label"`
}

func (l RuinRow) Covers(r int) bool { return r >= l.RollMin && r <= l.RollMax }

// ChaseRow é uma linha da Tabela 6-5 (d20, p274), que também casa por faixa.
type ChaseRow struct {
	RollMin int     `json:"rollMin"`
	RollMax int     `json:"rollMax"`
	Kind    string  `json:"kind"`
	Test    *string `json:"test"`
	CD      *int    `json:"cd"`
	Example string  `json:"example"`
}

func (l ChaseRow) Covers(r int) bool { return r >= l.RollMin && r <= l.RollMax }

// RewardRow é a única das três que casa por valor EXATO e não por faixa: a
// tabela de recompensa/castigo tem uma linha por face do d6.
type RewardRow struct {
	Roll    int    `json:"roll"`
	Reward  string `json:"reward"`
	Castigo string `json:"castigo"`
}

func (l RewardRow) Covers(r int) bool { return l.Roll == r }

// DungeonIdea é uma linha da tabela de ideias de masmorra, casando por face.
type DungeonIdea struct {
	Roll  int    `json:"roll"`
	Label string `json:"label"`
}

func (i DungeonIdea) Covers(r int) bool { return i.Roll == r }

// DungeonSize é uma faixa de salas, com o ritmo que o livro recomenda para ela.
type DungeonSize struct {
	Size                   string `json:"size"`
	Label                  string `json:"label"`
	MinRooms               int    `json:"minRooms"`
	MaxRooms               int    `json:"maxRooms"`
	Pacing                 string `json:"pacing"`
	MaxSecondaryObjectives int    `json:"maxSecondaryObjectives"`
}

// GMTables é o recurso `gm-tables` inteiro: as três tabelas que se rola.
type GMTables struct {
	Ruina         []RuinRow         `json:"ruina"`
	ChaseEvents   []ChaseRow        `json:"chaseEvents"`
	RewardCastigo []RewardRow       `json:"rewardCastigo"`
	RewardLabels  map[string]string `json:"rewardLabels"`
	CastigoLabels map[string]string `json:"castigoLabels"`
}

// DungeonDesign é o recurso `dungeon-design`: o esqueleto de masmorra.
type DungeonDesign struct {
	Sizes          []string      `json:"sizes"`
	SizeTable      []DungeonSize `json:"sizeTable"`
	RoomsPerThreat int           `json:"roomsPerThreat"`
	Ideas          []DungeonIdea `json:"ideas"`
}

var (
	improvOnce   sync.Once
	gmTables     GMTables
	dungeonTable DungeonDesign
)

// ImprovTables lê os dois recursos uma vez e os devolve juntos, porque a cena
// do Improviso usa os dois na mesma carga e separá-los daria dois `sync.Once`
// para uma leitura só.
func ImprovTables() (GMTables, DungeonDesign) {
	improvOnce.Do(func() {
		if bruto, ok := catalog.Resource("gm-tables"); ok {
			_ = json.Unmarshal(bruto, &gmTables)
		}
		if bruto, ok := catalog.Resource("dungeon-design"); ok {
			_ = json.Unmarshal(bruto, &dungeonTable)
		}
	})
	return gmTables, dungeonTable
}
