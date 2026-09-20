package engine

import (
	"reflect"
	"testing"
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
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			teve := PaySustained(c.sustentados, c.pm)
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
