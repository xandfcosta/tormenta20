package api

import (
	"context"
	"regexp"
	"testing"
)

// A CD DA MAGIA DA FEITICEIRA É A DE CARISMA (p37): "seu atributo-chave para
// lançar magias é definido pelo seu Caminho". O topo da aba Magias já dizia a CD
// do motor, que lê o caminho; a linha de cada magia lia o atributo do catálogo de
// classes, que é um só para o Arcanista, e dava a CD de Inteligência.
//
// Carisma 4 e Inteligência 1 no 8º nível: 10 + 4 + 4 = 18 (p173); pela
// Inteligência seriam 15.
func TestTheSorcerersSpellRowSaysTheCharismaDC(t *testing.T) {
	f := newSceneFixture(t)
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET classChoices = '{"Arcanista":{"caminho":"feiticeiro"}}', intelligence = 1, charisma = 4 WHERE id = ?`, f.charID); err != nil {
		t.Fatalf("fazer da arcanista da bancada uma feiticeira: %v", err)
	}
	learnSpell(t, f, "bola-de-fogo")
	if !regexp.MustCompile(`CD <b>18</b> · Carisma`).MatchString(parallelSheet(t, f, f.charID, "spells")) {
		t.Fatal("o topo da aba Magias não diz a CD 18 de Carisma — a bancada não é a feiticeira que o caso pede")
	}
	if got := spellCDAtTheTable(t, f, "Bola de Fogo"); got != 18 {
		t.Errorf("o Conjurar da Bola de Fogo diz CD %d, e a feiticeira conjura com Carisma: CD 18", got)
	}
}
