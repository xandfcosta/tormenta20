package api

import (
	"fmt"
	"strings"
	"testing"

	"t20engine/domain/sheet"
)

// ENTRAR NUMA CLASSE (p35) soma os PV dela, e quem estava caído pode levantar (p236), pela mesma regra da
// queda de nível (decisão do dono).
func TestEnteringAClassWhileDownGetsTheCharacterUp(t *testing.T) {
	f, id := arcanistaDeNivel(t, 5)
	// A ficha cai pelo caminho de verdade: com 3 PV, a Constituição baixa de 0 a −1 e tira 1 PV por nível
	// do arcanista 5 — ela fica com −2, e o 1º nível de Guerreiro (5 + Con, p37) a põe de pé com 2.
	constitution := func(value int) string { return fmt.Sprintf("/personagens/%d/atributos/constitution/%d", id, value) }
	sheetGesture(t, f, constitution(0))
	arrangePools(t, f.s, id, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.HpCurrent = 3
		return pools, nil
	})
	sheetGesture(t, f, constitution(-1))
	if got := activeConditionsOf(t, f, id); !isDown(got) {
		t.Fatalf("o controle falhou: a ficha devia estar caída, e as condições são %s", got)
	}
	down := poolsOf(t, f.s, id).HpCurrent

	sheetGesture(t, f, fmt.Sprintf("/personagens/%d/classes/Guerreiro/entra", id))

	if got := poolsOf(t, f.s, id).HpCurrent; got <= down || got < 1 {
		t.Fatalf("o Guerreiro devia somar PV à ficha caída com %d, e ela ficou com %d", down, got)
	}
	if got := activeConditionsOf(t, f, id); strings.Contains(got, "inconsciente") || strings.Contains(got, "sangrando") {
		t.Errorf("de pé, a ficha continua com as condições de quem caiu: %s", got)
	}
}
