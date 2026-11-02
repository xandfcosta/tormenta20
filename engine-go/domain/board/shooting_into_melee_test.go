package board

import (
	"slices"
	"testing"

	"t20engine/domain/engine"
)

// "Uma criatura está em combate corpo a corpo se estiver dentro do alcance natural
// de qualquer inimigo (incluindo você)" (p233). O alcance natural é a Tabela 1-21
// (p107): 1,5m até Médio, 3m Grande, 4,5m Enorme e 9m Colossal.
func creatureAt(entry, kind string, x, y, footprint int) BoardToken {
	return BoardToken{ID: "t-" + entry, Kind: kind, X: x, Y: y, Footprint: footprint, EntryID: &entry}
}

func TestWhoCountsAsInMeleeCombat(t *testing.T) {
	archer := creatureAt("arqueira", "character", 0, 0, 1)
	for _, c := range []struct {
		name   string
		tokens []BoardToken
		target string
		want   bool
	}{
		{name: "sozinho, longe de todos", tokens: []BoardToken{creatureAt("ogro", "npc", 10, 0, 1)}, target: "ogro"},
		{name: "ao lado de um aliado de quem atira", tokens: []BoardToken{
			creatureAt("ogro", "npc", 10, 0, 1), creatureAt("guerreiro", "character", 11, 0, 1),
		}, target: "ogro", want: true},
		{name: "a diagonal também é adjacente", tokens: []BoardToken{
			creatureAt("ogro", "npc", 10, 0, 1), creatureAt("guerreiro", "character", 11, 1, 1),
		}, target: "ogro", want: true},
		{name: "duas casas de um Médio estão fora", tokens: []BoardToken{
			creatureAt("ogro", "npc", 10, 0, 1), creatureAt("guerreiro", "character", 12, 0, 1),
		}, target: "ogro"},
		{name: "ao lado de quem atira: incluindo você", tokens: []BoardToken{creatureAt("ogro", "npc", 1, 0, 1)}, target: "ogro", want: true},
		{name: "ao lado só de quem é do lado dele", tokens: []BoardToken{
			creatureAt("ogro", "npc", 10, 0, 1), creatureAt("goblin", "npc", 11, 0, 1),
		}, target: "ogro"},
		{name: "dentro dos 3m de um Grande", tokens: []BoardToken{
			creatureAt("heroi", "npc", 10, 0, 1), creatureAt("cavalo", "character", 12, 0, 2),
		}, target: "heroi", want: true},
		{name: "fora dos 3m de um Grande", tokens: []BoardToken{
			creatureAt("heroi", "npc", 10, 0, 1), creatureAt("cavalo", "character", 13, 0, 2),
		}, target: "heroi"},
		{name: "objeto não é criatura", tokens: []BoardToken{
			creatureAt("ogro", "npc", 10, 0, 1), {ID: "barril", Kind: "object", X: 11, Y: 0, Footprint: 1},
		}, target: "ogro"},
	} {
		t.Run(c.name, func(t *testing.T) {
			b := &BoardState{Tokens: append([]BoardToken{archer}, c.tokens...)}
			got := slices.Contains(SituationsBetween(b, "arqueira",
				AttackTargetOnTheBoard{EntryID: c.target}), engine.TargetInMeleeCombat)
			if got != c.want {
				t.Errorf("em combate corpo a corpo = %v, want %v", got, c.want)
			}
		})
	}
}

// A peça escondida não entrega a emboscada: o −5 apareceria na rolagem do jogador
// antes de o mestre revelar quem está lá.
func TestAHiddenTokenDoesNotPutTheTargetInMelee(t *testing.T) {
	hidden := creatureAt("espiao", "character", 11, 0, 1)
	hidden.Hidden = true
	b := &BoardState{Tokens: []BoardToken{
		creatureAt("arqueira", "character", 0, 0, 1), creatureAt("ogro", "npc", 10, 0, 1), hidden,
	}}
	if slices.Contains(SituationsBetween(b, "arqueira", AttackTargetOnTheBoard{EntryID: "ogro"}), engine.TargetInMeleeCombat) {
		t.Error("a peça escondida contou")
	}
}

// Atacar um objeto não é atirar em combate corpo a corpo: "contra uma criatura".
func TestAnObjectTargetIsNeverInMelee(t *testing.T) {
	b := &BoardState{Tokens: []BoardToken{
		creatureAt("arqueira", "character", 0, 0, 1),
		{ID: "porta", Kind: "object", X: 1, Y: 0, Footprint: 1},
	}}
	if slices.Contains(SituationsBetween(b, "arqueira", AttackTargetOnTheBoard{TokenID: "porta"}), engine.TargetInMeleeCombat) {
		t.Error("o objeto contou como criatura em combate")
	}
}
