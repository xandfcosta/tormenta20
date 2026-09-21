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
	casos := []struct {
		nome        string
		sustentados []string
		pm          int
		caido       bool
		quantos     int
		mantidos    []string
		caidos      []string
	}{
		{nome: "sem sustentada não custa nada", pm: 9},
		{nome: "uma sustentada custa 1 PM",
			sustentados: []string{"velocidade"}, pm: 9, quantos: 1, mantidos: []string{"velocidade"}},
		{nome: "três sustentadas custam 3 PM, uma por uma",
			sustentados: []string{"velocidade", "oracao", "forma-eterea"}, pm: 9, quantos: 3,
			mantidos: []string{"velocidade", "oracao", "forma-eterea"}},
		// "Se não o fizer, a habilidade termina" — sem mana, o efeito CAI, e
		// não fica de pé devendo.
		{nome: "sem mana nenhuma a sustentada cai",
			sustentados: []string{"velocidade"}, pm: 0, caidos: []string{"velocidade"}},
		// O MANA QUE HÁ paga o que der, e a ORDEM é a de quem foi conjurado
		// primeiro: o livro deixa a escolha com o jogador, e uma ordem estável e
		// explicável é o que permite a ele desfazer a diferença encerrando a
		// que quiser — encerrar também é ação livre.
		{nome: "com 2 PM e três sustentadas, as duas mais antigas ficam",
			sustentados: []string{"velocidade", "oracao", "forma-eterea"}, pm: 2, quantos: 2,
			mantidos: []string{"velocidade", "oracao"}, caidos: []string{"forma-eterea"}},
		{nome: "mana negativa é tratada como nenhuma",
			sustentados: []string{"velocidade"}, pm: -3, caidos: []string{"velocidade"}},
		// INCONSCIENTE NÃO SUSTENTA, e o mana cheio não muda nada: a 0 PV "você
		// cai inconsciente" (p236), e manter a habilidade exige uma AÇÃO LIVRE
		// no início do turno — que quem não age não faz. É por aqui que a
		// cláusula da morte da p227 chega ao app, cujo PV tem piso em zero.
		{nome: "quem caiu a 0 PV não paga, mesmo com mana de sobra",
			sustentados: []string{"velocidade", "oracao"}, pm: 99, caido: true,
			caidos: []string{"velocidade", "oracao"}},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			teve := PaySustained(c.sustentados, c.pm, !c.caido)
			if teve.Cost != c.quantos {
				t.Errorf("custou %d PM, quero %d", teve.Cost, c.quantos)
			}
			if !reflect.DeepEqual(teve.Paid, c.mantidos) {
				t.Errorf("ficaram de pé %v, quero %v", teve.Paid, c.mantidos)
			}
			if !reflect.DeepEqual(teve.Dropped, c.caidos) {
				t.Errorf("caíram %v, quero %v", teve.Dropped, c.caidos)
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
	fora := map[string]int{}
	medidos := 0
	for _, nome := range catalog.Resources() {
		bruto, ok := catalog.Resource(nome)
		if !ok {
			continue
		}
		medidos++
		var cru any
		if err := json.Unmarshal(bruto, &cru); err != nil {
			continue
		}
		if n := countSustained(cru); n > 0 && nome != "spells" {
			fora[nome] = n
		}
	}
	if medidos < 10 {
		t.Fatalf("só %d catálogos varridos — a varredura está olhando o lugar errado", medidos)
	}
	if len(fora) > 0 {
		t.Errorf("habilidade sustentada FORA das magias: %v — o limite de uma sustentada por vez "+
			"passa a precisar do filtro por fonte em `assertOnlyOneSustainedSpell` (p227)", fora)
	}
	t.Logf("%d catálogos varridos; a duração sustentada vive só nas magias", medidos)
}

// countSustained conta `"duration": "sustentada"` em qualquer profundidade.
func countSustained(no any) int {
	switch v := no.(type) {
	case map[string]any:
		n := 0
		for k, filho := range v {
			if k == "duration" && filho == "sustentada" {
				n++
			}
			n += countSustained(filho)
		}
		return n
	case []any:
		n := 0
		for _, filho := range v {
			n += countSustained(filho)
		}
		return n
	}
	return 0
}

// A RAZÃO DA QUEDA é diferente, e a mesa lê a diferença: sem mana é uma escolha
// que acabou, inconsciente é um personagem no chão.
func TestTheUpkeepSaysWhyTheAbilityEnded(t *testing.T) {
	semMana := PaySustained([]string{"velocidade"}, 0, true)
	if semMana.Unconscious {
		t.Error("cair por falta de mana não é cair por estar inconsciente")
	}
	noChao := PaySustained([]string{"velocidade"}, 99, false)
	if !noChao.Unconscious {
		t.Error("quem está a 0 PV cai por não poder agir, e não por falta de mana")
	}
	if noChao.Cost != 0 {
		t.Errorf("inconsciente não gasta PM, e gastou %d", noChao.Cost)
	}
}
