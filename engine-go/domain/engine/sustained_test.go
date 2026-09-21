package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"t20engine/domain/catalog"
)

// A MANUTENÇÃO DO INÍCIO DO TURNO (T20 p227): "o personagem deve gastar 1 PM
// como uma ação livre no início de cada turno seu para manter o efeito ativo.
// Se não o fizer, a habilidade termina. Você pode manter diversas habilidades
// sustentadas, pagando o custo de cada uma".
func TestSustainedAbilitiesCostAPointOfManaEachTurn(t *testing.T) {
	cases := []struct {
		name        string
		sustained   []string
		pm          int
		down        bool
		wantCost    int
		wantPaid    []string
		wantDropped []string
	}{
		{name: "sem sustentada não custa nada", pm: 9},
		{name: "uma sustentada custa 1 PM",
			sustained: []string{"velocidade"}, pm: 9, wantCost: 1, wantPaid: []string{"velocidade"}},
		{name: "três sustentadas custam 3 PM, uma por uma",
			sustained: []string{"velocidade", "oracao", "forma-eterea"}, pm: 9, wantCost: 3,
			wantPaid: []string{"velocidade", "oracao", "forma-eterea"}},
		// "Se não o fizer, a habilidade termina" — sem mana, o efeito CAI, e
		// não fica de pé devendo.
		{name: "sem mana nenhuma a sustentada cai",
			sustained: []string{"velocidade"}, pm: 0, wantDropped: []string{"velocidade"}},
		// O MANA QUE HÁ paga o que der, e a ORDEM é a de quem foi conjurado
		// primeiro: o livro deixa a escolha com o jogador, e uma ordem estável e
		// explicável é o que permite a ele desfazer a diferença encerrando a
		// que quiser — encerrar também é ação livre.
		{name: "com 2 PM e três sustentadas, as duas mais antigas ficam",
			sustained: []string{"velocidade", "oracao", "forma-eterea"}, pm: 2, wantCost: 2,
			wantPaid: []string{"velocidade", "oracao"}, wantDropped: []string{"forma-eterea"}},
		{name: "mana negativa é tratada como nenhuma",
			sustained: []string{"velocidade"}, pm: -3, wantDropped: []string{"velocidade"}},
		// INCONSCIENTE NÃO SUSTENTA, e o mana cheio não muda nada: a 0 PV "você
		// cai inconsciente" (p236), e manter a habilidade exige uma AÇÃO LIVRE
		// no início do turno — que quem não age não faz. É por aqui que a
		// cláusula da morte da p227 chega ao app, cujo PV tem piso em zero.
		{name: "quem caiu a 0 PV não paga, mesmo com mana de sobra",
			sustained: []string{"velocidade", "oracao"}, pm: 99, down: true,
			wantDropped: []string{"velocidade", "oracao"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PaySustained(c.sustained, c.pm, ActionMoment{OnTurn: true, CanAct: !c.down})
			if got.Cost != c.wantCost {
				t.Errorf("custou %d PM, quero %d", got.Cost, c.wantCost)
			}
			if !reflect.DeepEqual(got.Paid, c.wantPaid) {
				t.Errorf("ficaram de pé %v, quero %v", got.Paid, c.wantPaid)
			}
			if !reflect.DeepEqual(got.Dropped, c.wantDropped) {
				t.Errorf("caíram %v, quero %v", got.Dropped, c.wantDropped)
			}
		})
	}
}

// SÓ MAGIA É SUSTENTADA, hoje — e é isso que deixa o limite de "uma sustentada
// por vez" contar todas sem olhar a fonte do efeito.
//
// O livro SEPARA as duas coisas: "você pode manter diversas habilidades
// sustentadas… mas apenas uma magia sustentada por vez" (p227). A separação só
// tem objeto quando existir uma habilidade sustentada que não seja magia, e
// nenhuma existe: as 32 estão todas no catálogo de magias. Este guarda reprova
// no dia em que uma aparecer, que é quando o
// `assertOnlyOneSustainedSpell` precisa passar a filtrar pela fonte.
func TestOnlySpellsAreSustainedInTheBook(t *testing.T) {
	outside := map[string]int{}
	measured := 0
	for _, name := range catalog.Resources() {
		payload, ok := catalog.Resource(name)
		if !ok {
			continue
		}
		measured++
		var raw any
		if err := json.Unmarshal(payload, &raw); err != nil {
			continue
		}
		if n := countSustained(raw); n > 0 && name != "spells" {
			outside[name] = n
		}
	}
	if measured < 10 {
		t.Fatalf("só %d catálogos varridos — a varredura está olhando o lugar errado", measured)
	}
	if len(outside) > 0 {
		t.Errorf("habilidade sustentada FORA das magias: %v — o limite de uma sustentada por vez "+
			"passa a precisar do filtro por fonte em `assertOnlyOneSustainedSpell` (p227)", outside)
	}
	t.Logf("%d catálogos varridos; a duração sustentada vive só nas magias", measured)
}

// countSustained conta `"duration": "sustentada"` em qualquer profundidade.
func countSustained(no any) int {
	switch v := no.(type) {
	case map[string]any:
		n := 0
		for k, child := range v {
			if k == "duration" && child == "sustentada" {
				n++
			}
			n += countSustained(child)
		}
		return n
	case []any:
		n := 0
		for _, child := range v {
			n += countSustained(child)
		}
		return n
	}
	return 0
}

// A RAZÃO DA QUEDA é diferente, e a mesa lê a diferença: sem mana é uma escolha
// que acabou, inconsciente é um personagem no chão.
func TestTheUpkeepSaysWhyTheAbilityEnded(t *testing.T) {
	noMana := PaySustained([]string{"velocidade"}, 0, ActionMoment{OnTurn: true, CanAct: true})
	if noMana.Unconscious {
		t.Error("cair por falta de mana não é cair por estar inconsciente")
	}
	unconscious := PaySustained([]string{"velocidade"}, 99, ActionMoment{OnTurn: true, CanAct: false})
	if !unconscious.Unconscious {
		t.Error("quem está a 0 PV cai por não poder agir, e não por falta de mana")
	}
	if unconscious.Cost != 0 {
		t.Errorf("inconsciente não gasta PM, e gastou %d", unconscious.Cost)
	}
}
